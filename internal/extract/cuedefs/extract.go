package cuedefs

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/parser"

	"github.com/open-platform-model/docs-kit/internal/doctext"
)

// Config is a cue-definitions source's entry in docs-kit.cue (docs-kit
// C17), as validated against #CueDefinitions.
type Config struct {
	Package     string            `json:"package"` // "./src"
	Skip        []string          `json:"skip"`    // globs on file base names
	Section     string            `json:"section"` // "reference/definitions/"
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Weight      int               `json:"weight"`
	Pages       []PageConfig      `json:"pages"`
	Exclude     map[string]string `json:"exclude"` // definition -> reason
}

// PageConfig is one configured page.
type PageConfig struct {
	File        string   `json:"file"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Definitions []string `json:"definitions"`
}

// Options configures one extraction.
type Options struct {
	Root    string // the source tree
	Config  Config
	Version string         // "2.0.0" or "edge"
	Policy  doctext.Policy // what prose citations become
	// Lenient: the config came from outside the source tree (a backfill),
	// so a placement problem is a warning passed to Warn, not an error.
	Lenient bool
	Warn    func(string)
}

// PlacementError lists every way the configured pages are out of step
// with the package.
type PlacementError struct {
	Package  string
	Problems []string
}

func (e *PlacementError) Error() string {
	return fmt.Sprintf("cue-definitions %s: the pages in docs-kit.cue are out of step with the package:\n  %s", e.Package, strings.Join(e.Problems, "\n  "))
}

// def is one exported top-level definition of the package.
type def struct {
	name  string
	file  string     // repo-relative slash path
	path  string     // the file on disk
	field *ast.Field // as parsed; never mutated
	doc   []string   // raw "//" lines of its doc comment
	refs  []string   // top-level definitions its value names, sorted
}

// Extract parses the package and builds the doc model.
func Extract(o Options) (*Model, error) {
	cfg := o.Config
	if o.Warn == nil {
		o.Warn = func(string) {}
	}
	dir := filepath.Join(o.Root, filepath.FromSlash(cfg.Package))
	defs, err := loadDefs(o.Root, dir, cfg.Skip)
	if err != nil {
		return nil, fmt.Errorf("cue-definitions %s: %w", cfg.Package, err)
	}
	modPath, err := modulePath(o.Root, dir)
	if err != nil {
		return nil, fmt.Errorf("cue-definitions %s: %w", cfg.Package, err)
	}
	pages, problems := place(cfg, defs)
	if len(problems) > 0 {
		if !o.Lenient {
			return nil, &PlacementError{Package: cfg.Package, Problems: problems}
		}
		for _, p := range problems {
			o.Warn(fmt.Sprintf("cue-definitions %s: %s", cfg.Package, p))
		}
	}

	m := &Model{
		Schema: SchemaID, ModulePath: modPath, Version: o.Version,
		Section: cfg.Section, Title: cfg.Title, Description: cfg.Description, Weight: cfg.Weight,
	}
	pageOf := map[string]string{}
	for i, p := range pages {
		m.Pages = append(m.Pages, Page{File: p.File, Title: p.Title, Description: p.Description, Weight: i + 1, Definitions: p.Definitions})
		for _, n := range p.Definitions {
			pageOf[n] = p.File
		}
	}
	usedBy := usedByOf(defs, pageOf)
	for _, p := range pages {
		for _, n := range p.Definitions {
			d := defs[n]
			entry, err := definition(d, defs, pageOf, usedBy[n], o.Policy)
			if err != nil {
				return nil, fmt.Errorf("cue-definitions %s: %s: %w", cfg.Package, n, err)
			}
			m.Definitions = append(m.Definitions, entry)
		}
	}
	for _, n := range sortedKeys(cfg.Exclude) {
		if _, ok := defs[n]; ok {
			m.Excluded = append(m.Excluded, Excluded{Name: n, Reason: cfg.Exclude[n]})
		}
	}
	return m, nil
}

// usedByOf inverts uses over the placed definitions, each list sorted.
func usedByOf(defs map[string]*def, pageOf map[string]string) map[string][]string {
	usedBy := map[string][]string{}
	for _, n := range sortedNames(defs) {
		if pageOf[n] == "" {
			continue
		}
		for _, u := range uses(defs, pageOf, n) {
			usedBy[u] = append(usedBy[u], n)
		}
	}
	return usedBy
}

// place checks the configured pages against the package: every exported
// definition is on exactly one page or excluded, and every configured name
// is declared. It returns the pages as built leniently (unknown names
// skipped, a second placement ignored, empty pages dropped) and every
// problem found.
func place(cfg Config, defs map[string]*def) (pages []PageConfig, problems []string) {
	placed := map[string]string{}
	for _, p := range cfg.Pages {
		kept := p
		kept.Definitions = nil
		for _, n := range p.Definitions {
			d, ok := defs[n]
			switch {
			case !ok:
				problems = append(problems, fmt.Sprintf("%s is placed on page %q, but the package declares no such definition; remove it from the page", n, p.File))
				continue
			case placed[n] != "":
				problems = append(problems, fmt.Sprintf("%s is placed on pages %q and %q; place it on one page", n, placed[n], p.File))
				continue
			case cfg.Exclude[n] != "":
				problems = append(problems, fmt.Sprintf("%s is placed on page %q and also excluded; remove it from one of them", n, p.File))
			case len(d.doc) == 0:
				problems = append(problems, fmt.Sprintf("%s (%s) has no doc comment; write one, its first sentence is the definition's summary", n, d.file))
			}
			placed[n] = p.File
			kept.Definitions = append(kept.Definitions, n)
		}
		if len(kept.Definitions) > 0 {
			pages = append(pages, kept)
		}
	}
	for _, n := range sortedKeys(cfg.Exclude) {
		if _, ok := defs[n]; !ok {
			problems = append(problems, fmt.Sprintf("%s is excluded, but the package declares no such definition; remove it from exclude", n))
		}
	}
	for _, n := range sortedNames(defs) {
		if placed[n] == "" && cfg.Exclude[n] == "" {
			problems = append(problems, fmt.Sprintf("%s (%s) is exported but neither placed in a page nor excluded in docs-kit.cue; add it to a page's definitions or to exclude with a reason", n, defs[n].file))
		}
	}
	return pages, problems
}

// definition builds one placed definition's entry.
func definition(d *def, defs map[string]*def, pageOf map[string]string, usedBy []string, policy doctext.Policy) (Definition, error) {
	doc := parseDoc(d.name, d.doc, policy)
	g := atAGlance(d)
	spec, err := specSource(d.path, d)
	if err != nil {
		return Definition{}, err
	}
	shape := g.shape
	if g.kind != "" {
		shape += ", `kind: " + g.kind + "`"
	}
	var embeds []string
	for _, e := range g.embeds {
		if pageOf[e] != "" {
			embeds = append(embeds, e)
		}
	}
	var example []string
	for _, ex := range doc.examples {
		example = append(example, exampleLines(ex)...)
	}
	var rules []Rule
	for _, r := range enforcement(d) {
		rules = append(rules, Rule{Rule: r.text, Code: r.code, By: "cue"})
	}
	return Definition{
		Name: d.name, Anchor: Anchor(d.name), Page: pageOf[d.name], File: d.file,
		Summary: doc.summary, Notes: noteStrings(doc.notes), Example: example,
		Shape: shape, Embeds: embeds, CUE: spec,
		Uses: uses(defs, pageOf, d.name), UsedBy: usedBy, Rules: rules,
	}, nil
}

// Anchor is the heading ID a site gives "## #Name": lower case, "#"
// dropped.
func Anchor(name string) string {
	return strings.ToLower(strings.TrimPrefix(name, "#"))
}

// loadDefs parses every .cue file of dir not matched by skip and returns
// the exported top-level definitions by name. root is the source tree the
// recorded file paths are relative to.
func loadDefs(root, dir string, skip []string) (map[string]*def, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.cue"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s holds no .cue file", dir)
	}
	sort.Strings(files)
	defs := map[string]*def{}
	for _, p := range files {
		if skipped(filepath.Base(p), skip) {
			continue
		}
		if err := fileDefs(root, p, defs); err != nil {
			return nil, err
		}
	}
	for _, d := range defs {
		seen := map[string]bool{}
		ast.Walk(d.field.Value, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				if _, top := defs[id.Name]; top && id.Name != d.name {
					seen[id.Name] = true
				}
			}
			return true
		}, nil)
		d.refs = sortedKeys(seen)
	}
	return defs, nil
}

// fileDefs adds the exported top-level definitions of one file.
func fileDefs(root, p string, defs map[string]*def) error {
	f, err := parseFile(p)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	for _, d := range f.Decls {
		fld, ok := d.(*ast.Field)
		if !ok {
			continue
		}
		name, _, err := ast.LabelName(fld.Label)
		if err != nil || !strings.HasPrefix(name, "#") {
			continue
		}
		if prev, dup := defs[name]; dup {
			return fmt.Errorf("%s is declared in both %s and %s", name, prev.file, rel)
		}
		defs[name] = &def{name: name, file: rel, path: p, field: fld, doc: docLines(fld)}
	}
	return nil
}

func skipped(base string, skip []string) bool {
	for _, s := range skip {
		if ok, _ := path.Match(s, base); ok {
			return true
		}
	}
	return false
}

func parseFile(p string) (*ast.File, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return parser.ParseFile(p, b, parser.ParseComments)
}

// docLines returns the raw lines of a node's doc comment: the comment
// group that ends on the line directly above it.
func docLines(n ast.Node) []string {
	var out []string
	for _, cg := range ast.Comments(n) {
		if !cg.Doc {
			continue
		}
		for _, c := range cg.List {
			out = append(out, c.Text)
		}
	}
	return out
}

// uses returns the placed definitions a definition names, following a
// reference to an unplaced one (an excluded map shorthand) through to
// what that one names.
func uses(defs map[string]*def, pageOf map[string]string, name string) []string {
	out := map[string]bool{}
	seen := map[string]bool{name: true}
	var walk func(string)
	walk = func(n string) {
		for _, r := range defs[n].refs {
			if seen[r] {
				continue
			}
			seen[r] = true
			if pageOf[r] != "" {
				out[r] = true
			} else {
				walk(r)
			}
		}
	}
	walk(name)
	return sortedKeys(out)
}

// modulePath reads the module path of the CUE module enclosing dir: the
// module field of the nearest cue.mod/module.cue at or above dir, within
// root.
func modulePath(root, dir string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	d, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		p := filepath.Join(d, "cue.mod", "module.cue")
		if _, err := os.Stat(p); err == nil {
			return moduleField(p)
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if d == rootAbs || d == filepath.Dir(d) {
			return "", fmt.Errorf("no cue.mod/module.cue at or above %s; the pages name the package's module", dir)
		}
		d = filepath.Dir(d)
	}
}

func moduleField(p string) (string, error) {
	f, err := parseFile(p)
	if err != nil {
		return "", err
	}
	for _, decl := range f.Decls {
		fld, ok := decl.(*ast.Field)
		if !ok {
			continue
		}
		if n, _, _ := ast.LabelName(fld.Label); n != "module" {
			continue
		}
		if bl, ok := fld.Value.(*ast.BasicLit); ok {
			if s, err := strconv.Unquote(bl.Value); err == nil && s != "" {
				return s, nil
			}
		}
	}
	return "", fmt.Errorf("%s has no module field holding a string", p)
}

func sortedNames(defs map[string]*def) []string {
	out := make([]string, 0, len(defs))
	for n := range defs {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
