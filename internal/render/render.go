// Package render turns the doc model into pages in the page dialect, with
// templates embedded in the tool. It reads the doc model and the bundle's
// identity, never source.
package render

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/open-platform-model/docs-kit/internal/extract/cuecatalog"
	"github.com/open-platform-model/docs-kit/internal/mdtext"
)

//go:embed templates/*.tmpl
var templateFiles embed.FS

var templates = template.Must(template.New("").Funcs(template.FuncMap{
	"yaml": mdtext.YAMLString,
}).ParseFS(templateFiles, "templates/*.tmpl"))

// Placement kinds a Target is rendered for.
const (
	KindTab  = "tab"
	KindDocs = "docs"
)

// Target is the bundle the pages are rendered for.
type Target struct {
	Kind    string // "tab" (also when empty) or "docs"
	Root    string // placement root, "/catalogs/opm/" or "/docs/"
	Segment string // a tab's "4.4" or "edge"; unused for docs
	Edge    bool
	Version string // "4.4.5" or "edge"
	Repo    string // "open-platform-model/catalog_opm"
	Commit  string // the 40-hex source commit
}

// URL is the published URL of a page path under the bundle ("" is the
// landing; "traits/backup" a member page): <root><segment>/<page>/ for a
// tab, /docs/<page>/ for a docs bundle, whose pages publish under the site
// version that pulls it.
func (t Target) URL(page string) string {
	u := t.Root
	if t.Kind != KindDocs {
		u += t.Segment + "/"
	}
	if page != "" {
		u += page + "/"
	}
	return u
}

// Page is one rendered file under content/.
type Page struct {
	Path string // relative to content/, "traits/backup.md"
	Body string // the page standing alone, front matter included
	// Completable: an authored page at Path from a markdown source of the
	// same bundle comes first, and Tail follows it (Complete).
	Completable bool
	// Heading is Tail's first heading line ("## Catalog members"); an
	// authored page that already holds it is refused.
	Heading string
	// Tail is the generated body without front matter, appended to an
	// authored page.
	Tail string
}

// Member kinds, as the doc model names them.
const (
	kindBlueprint = "blueprint"
	kindResource  = "resource"
	kindTrait     = "trait"
)

// kinds lists the member kinds in page order, blueprints first.
var kinds = []struct {
	kind, dir, title string
	weight           int
	what             string
}{
	{kindBlueprint, "blueprints", "Blueprints", 1, "A blueprint composes resources and traits into one workload shape that a component attaches in one step."},
	{kindResource, "resources", "Resources", 2, "A resource is something a component deploys. Every resource a component declares is a required demand."},
	{kindTrait, "traits", "Traits", 3, "A trait adds behaviour to a component. Its optional posture decides what happens when nothing on the platform handles it."},
}

// Catalog renders the kind indexes and every member page of a catalog.
// The landing is Landing's, so an authored one can take its place.
func Catalog(m *cuecatalog.Model, t Target) ([]Page, error) {
	r := &renderer{m: m, t: t, byFQN: map[string]*cuecatalog.Member{}}
	for i := range m.Members {
		r.byFQN[m.Members[i].FQN] = &m.Members[i]
	}
	var pages []Page
	for _, k := range kinds {
		body, err := r.kindIndex(k.kind)
		if err != nil {
			return nil, err
		}
		pages = append(pages, Page{Path: k.dir + "/_index.md", Body: body})
	}
	for i := range m.Members {
		mem := &m.Members[i]
		body, err := r.memberPage(mem)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", mem.FQN, err)
		}
		pages = append(pages, Page{Path: mem.Page + ".md", Body: body})
	}
	for _, p := range pages {
		if err := mdtext.CheckShortcodes(p.Path, p.Body); err != nil {
			return nil, err
		}
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Path < pages[j].Path })
	return pages, nil
}

type renderer struct {
	m     *cuecatalog.Model
	t     Target
	byFQN map[string]*cuecatalog.Member
}

func execute(name string, data any) (string, error) {
	var b bytes.Buffer
	if err := templates.ExecuteTemplate(&b, name, data); err != nil {
		return "", err
	}
	// Every page and block ends with exactly one newline.
	return strings.TrimRight(b.String(), "\n") + "\n", nil
}

// catalogRef names the catalog build: "`<module>` version `<v>`", or for
// edge "`<module>` at `main` (commit `<12 hex>`), unreleased".
func (r *renderer) catalogRef() string {
	return CatalogRef(r.m, r.t)
}

// CatalogRef names a catalog build as the pages do.
func CatalogRef(m *cuecatalog.Model, t Target) string {
	c := mdtext.Code
	if t.Edge {
		commit := t.Commit
		if len(commit) > 12 {
			commit = commit[:12]
		}
		return fmt.Sprintf("%s at %s (commit %s), unreleased", c(m.ModulePath), c("main"), c(commit))
	}
	return fmt.Sprintf("%s version %s", c(m.ModulePath), c(m.Version))
}

func (r *renderer) memberLink(mem *cuecatalog.Member) string {
	return fmt.Sprintf("[%s](%s)", mem.Title, r.t.URL(mem.Page))
}

// links renders fqns as member links, or as code when no page exists.
func (r *renderer) links(fqns []string) string {
	var out []string
	for _, f := range fqns {
		if mem, ok := r.byFQN[f]; ok {
			out = append(out, r.memberLink(mem))
		} else {
			out = append(out, mdtext.Code(f))
		}
	}
	if len(out) == 0 {
		return "not declared"
	}
	return strings.Join(out, ", ")
}

type row struct{ Key, Value string }

// Alert texts, kept byte for byte.
const (
	providedAlert = "> [!IMPORTANT]\n> **Provided by your platform**\n>\n> This catalog defines the contract and ships no transformer for it. Your platform needs exactly one catalog that implements it. Without one, %s; with two, the kernel refuses every render on that platform.\n"
	notImplAlert  = "> [!WARNING]\n> **Not implemented**\n>\n> This catalog defines the %s and ships no transformer that handles it. On a platform where no other catalog handles it, %s.\n"
)

// alert is the At a glance alert of a marked member, or "".
func alert(mem *cuecatalog.Member) string {
	if mem.Mark == nil {
		return ""
	}
	uses := "attaches it"
	if mem.Kind == kindResource {
		uses = "declares it"
	}
	without := fmt.Sprintf("rendering a component that %s fails", uses)
	if mem.Kind == kindTrait && mem.Optional != nil && *mem.Optional {
		without = fmt.Sprintf("a component that %s still renders, and the render warns that the trait is not handled and ignores its values", uses)
	}
	if *mem.Mark == cuecatalog.MarkProvidedByPlatform {
		return fmt.Sprintf(providedAlert, without)
	}
	return fmt.Sprintf(notImplAlert, mem.Kind, without)
}

type memberView struct {
	Title, Description, Alert, Noun, SpecKey, SpecCUE, Notes, ServedBy string
	Rows, Enforcement                                                  []row
	Elsewhere                                                          []string
}

func (r *renderer) memberPage(mem *cuecatalog.Member) (string, error) {
	c := mdtext.Code
	v := memberView{
		Title:       mem.Title,
		Description: mem.Description,
		Alert:       alert(mem),
		Noun:        mem.Kind,
		SpecKey:     c("spec." + mem.Spec.Key),
		SpecCUE:     mem.Spec.CUE,
		Rows:        r.glance(mem),
		Notes:       r.notes(mem.Notes),
		ServedBy:    r.servedBy(mem),
	}
	for _, l := range mem.Spec.Linked {
		owner := strings.TrimSuffix(l.Page, "/")
		title := owner
		for i := range r.m.Members {
			if r.m.Members[i].Page == owner {
				title = r.m.Members[i].Title
			}
		}
		v.Elsewhere = append(v.Elsewhere, fmt.Sprintf("%s: the spec of [%s](%s)", c(l.Definition), title, r.t.URL(owner)))
	}
	for _, e := range mem.Spec.External {
		what := "from " + c(e.Package)
		if e.Vendored {
			what += ", the vendored Kubernetes API types"
		}
		v.Elsewhere = append(v.Elsewhere, fmt.Sprintf("%s: %s", c(e.Definition), what))
	}
	for _, e := range mem.Enforcement {
		v.Enforcement = append(v.Enforcement, row{mdtext.Cell(e.Rule), c(e.By)})
	}
	return execute("member.md.tmpl", v)
}

// glance is the At a glance table.
func (r *renderer) glance(mem *cuecatalog.Member) []row {
	c := mdtext.Code
	var rows []row
	add := func(k, v string) { rows = append(rows, row{k, mdtext.Cell(v)}) }
	add("FQN", c(mem.FQN))
	add("API version", fmt.Sprintf("%s, %s ([contract levels](%s))", c(mem.APIVersion), mem.Level, r.t.URL("")))
	add("Module path", c(mem.ModulePath))
	add("Definition", fmt.Sprintf("%s in %s", c(mem.Definition), c(mem.File)))
	if mem.Wrapper != nil {
		add("Component wrapper", c(*mem.Wrapper))
	}
	add("Catalog", r.catalogRef())
	if mem.Category != nil {
		add("Category", c(*mem.Category))
	}
	if mem.Fulfilment != nil {
		switch *mem.Fulfilment {
		case "catalog":
			add("Fulfilment", c("catalog")+": the declaring catalog implements it")
		case "provider":
			add("Fulfilment", c("provider")+": a catalog on the platform implements it, never this one")
		}
	}
	if mem.Kind == kindTrait && mem.Optional != nil {
		if *mem.Optional {
			add("Optional posture", "advisory: "+c("optional")+" defaults to "+c("true")+", so an unhandled trait warns and the render continues; a module may override it where it attaches the trait")
		} else {
			add("Optional posture", "load-bearing: "+c("optional")+" defaults to "+c("false")+", so an unhandled trait fails the render; a module may override it where it attaches the trait")
		}
		add("Applies to (declared)", r.links(mem.AppliesTo))
	}
	if mem.Kind == kindBlueprint {
		add("Composed resources", r.links(mem.ComposedResources))
		add("Composed traits", r.links(mem.ComposedTraits))
	}
	for _, l := range mem.MatchLabels {
		v := c(fmt.Sprintf("%q: %s", l.Key, l.Value))
		if l.Required {
			v = c(fmt.Sprintf("%q!: %s", l.Key, l.Value)) + " (required)"
		}
		add("Match label", v)
	}
	return rows
}

// notes renders the note paragraphs: prose escaped, and a repository note
// the doc model lists linked to the source at the bundle's commit.
func (r *renderer) notes(paras []string) string {
	known := map[string]bool{}
	for _, n := range r.m.DocNotes {
		known[n] = true
	}
	base := fmt.Sprintf("https://github.com/%s/blob/%s/", r.t.Repo, r.t.Commit)
	out := make([]string, 0, len(paras))
	for _, p := range paras {
		out = append(out, mdtext.LinkDocNotes(mdtext.Text(p), base, func(n string) bool { return known[n] }))
	}
	return strings.Join(out, "\n\n")
}

func (r *renderer) transformerDescription(fqn string) (name, desc string) {
	for i := range r.m.Transformers {
		if t := &r.m.Transformers[i]; t.FQN == fqn {
			return t.Name, t.Description
		}
	}
	return fqn, ""
}

// servedBy renders the Served by section body, ending in a blank line.
func (r *renderer) servedBy(mem *cuecatalog.Member) string {
	c := mdtext.Code
	var b strings.Builder
	switch {
	case mem.Kind == kindBlueprint && len(mem.ServedBy) > 0:
		fmt.Fprintf(&b, "These transformers in %s require only what this blueprint supplies: its match labels answer their required labels, and it composes every resource and trait they require. A listed transformer can still emit nothing for a component that leaves out the optional fields it renders from.\n\n", c(r.m.ModulePath))
		b.WriteString("| Transformer | What it does |\n| --- | --- |\n")
		for _, s := range mem.ServedBy {
			name, desc := r.transformerDescription(s.FQN)
			fmt.Fprintf(&b, "| %s | %s |\n", c(name), mdtext.Cell(mdtext.Text(desc)))
		}
	case mem.Kind == kindBlueprint:
		fmt.Fprintf(&b, "No transformer in %s requires only what this blueprint supplies.\n", c(r.m.ModulePath))
	case len(mem.ServedBy) > 0:
		fmt.Fprintf(&b, "These transformers in %s require this %s or read it when present.\n\n", r.catalogRef(), mem.Kind)
		b.WriteString("| Transformer | Demand | What it does |\n| --- | --- | --- |\n")
		for _, s := range mem.ServedBy {
			name, desc := r.transformerDescription(s.FQN)
			fmt.Fprintf(&b, "| %s | %s | %s |\n", c(name), s.Demand, mdtext.Cell(mdtext.Text(desc)))
		}
	default:
		fmt.Fprintf(&b, "No transformer in %s requires this %s or reads it.\n", c(r.m.ModulePath), mem.Kind)
	}
	return b.String()
}

type kindView struct {
	Title, Description, What, Catalog, Plural, NotImplemented, Provided string
	Weight                                                              int
}

func (r *renderer) kindIndex(kind string) (string, error) {
	for _, k := range kinds {
		if k.kind != kind {
			continue
		}
		var ms []*cuecatalog.Member
		for i := range r.m.Members {
			if r.m.Members[i].Kind == kind {
				ms = append(ms, &r.m.Members[i])
			}
		}
		sort.Slice(ms, func(i, j int) bool { return ms[i].Page < ms[j].Page })
		var notImpl, provided []string
		for _, m := range ms {
			switch {
			case m.Mark == nil:
			case *m.Mark == cuecatalog.MarkNotImplemented:
				notImpl = append(notImpl, r.memberLink(m))
			default:
				provided = append(provided, r.memberLink(m))
			}
		}
		plural := strings.ToLower(k.title)
		return execute("kind.md.tmpl", kindView{
			Title:          k.title,
			Description:    fmt.Sprintf("The %d %s of the abstraction catalog, one generated page each.", len(ms), plural),
			Weight:         k.weight,
			What:           k.what,
			Catalog:        r.catalogRef(),
			Plural:         plural,
			NotImplemented: strings.Join(notImpl, ", "),
			Provided:       strings.Join(provided, ", "),
		})
	}
	return "", fmt.Errorf("unknown member kind %q", kind)
}

type kindLink struct {
	Title, URL string
	Count      int
}

// Block is the generated "Catalog members" section a landing ends with.
func Block(m *cuecatalog.Model, t Target) (string, error) {
	counts := map[string]int{}
	for i := range m.Members {
		counts[m.Members[i].Kind]++
	}
	ks := make([]kindLink, 0, len(kinds))
	for _, k := range kinds {
		ks = append(ks, kindLink{Title: k.title, URL: t.URL(k.dir), Count: counts[k.kind]})
	}
	return execute("block.md.tmpl", struct {
		Catalog string
		Kinds   []kindLink
	}{CatalogRef(m, t), ks})
}

// membersHeading is the landing's generated section heading.
const membersHeading = "## Catalog members"

// Landing renders the bundle's _index.md. With an authored landing, its
// front matter and body come first, then one blank line and the block; an
// authored body that already holds the block's heading is refused. Without
// one, the landing holds generated front matter and the block alone.
func Landing(m *cuecatalog.Model, t Target, authored, name string) (string, error) {
	block, err := Block(m, t)
	if err != nil {
		return "", err
	}
	if authored != "" {
		return Complete(authored, name, Page{Heading: membersHeading, Tail: block})
	}
	catalog := strings.TrimSuffix(strings.TrimPrefix(t.Root, "/catalogs/"), "/")
	return execute("landing.md.tmpl", struct{ Title, Description, Block string }{
		Title:       catalog + " catalog",
		Description: fmt.Sprintf("Every member of %s, by kind.", mdtext.Code(m.ModulePath)),
		Block:       block,
	})
}
