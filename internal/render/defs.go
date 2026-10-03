package render

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/extract/cuedefs"
	"github.com/open-platform-model/docs-kit/internal/mdtext"
)

// pagesHeading opens the definitions index's generated tail, which an
// authored index may complete.
const pagesHeading = "## Pages"

// defsRenderer renders a cue-definitions data file: the section index and
// one page per configured group (docs-kit C17).
type defsRenderer struct{}

func (defsRenderer) Schema() string { return cuedefs.SchemaID }

func (defsRenderer) Render(data []byte, t Target) ([]Page, error) {
	m, err := cuedefs.Decode(data)
	if err != nil {
		return nil, err
	}
	return Definitions(m, t)
}

// Definitions renders the section index and every page of a definitions
// model. Its pages link one another under /docs/, so it renders only into
// a docs bundle.
func Definitions(m *cuedefs.Model, t Target) ([]Page, error) {
	if t.Kind != KindDocs {
		return nil, fmt.Errorf("cue-definitions renders reference pages under /docs/; give the bundle placement kind \"docs\"")
	}
	r := &defsRender{m: m, t: t, byName: map[string]*cuedefs.Definition{}, pageTitle: map[string]string{}}
	for i := range m.Definitions {
		r.byName[m.Definitions[i].Name] = &m.Definitions[i]
	}
	for _, p := range m.Pages {
		r.pageTitle[p.File] = p.Title
	}
	index, err := r.index()
	if err != nil {
		return nil, err
	}
	pages := []Page{index}
	for _, p := range m.Pages {
		body, err := r.page(p)
		if err != nil {
			return nil, fmt.Errorf("%s%s: %w", m.Section, p.File, err)
		}
		pages = append(pages, Page{Path: m.Section + p.File + ".md", Body: body})
	}
	for _, p := range pages {
		if err := mdtext.CheckShortcodes(p.Path, p.Body); err != nil {
			return nil, err
		}
	}
	return pages, nil
}

type defsRender struct {
	m         *cuedefs.Model
	t         Target
	byName    map[string]*cuedefs.Definition
	pageTitle map[string]string
}

// indexURL is the section index's URL, "/docs/reference/definitions/".
func (r *defsRender) indexURL() string {
	return r.t.URL(strings.TrimSuffix(r.m.Section, "/"))
}

// link is the URL of a placed definition's entry, or "" when the pages
// place none of that name.
func (r *defsRender) link(name string) string {
	d, ok := r.byName[name]
	if !ok {
		return ""
	}
	return r.t.URL(r.m.Section+d.Page) + "#" + d.Anchor
}

func (r *defsRender) linkList(names []string) string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = "[`" + n + "`](" + r.link(n) + ")"
	}
	return strings.Join(out, ", ")
}

func (r *defsRender) index() (Page, error) {
	var b strings.Builder
	b.WriteString(pagesHeading + "\n\n")
	for _, p := range r.m.Pages {
		fmt.Fprintf(&b, "- [%s](%s): %s\n", p.Title, r.t.URL(r.m.Section+p.File), p.Description)
	}
	b.WriteString("\n## All definitions\n\n")
	b.WriteString("| Definition | Page | Summary |\n| --- | --- | --- |\n")
	for _, p := range r.m.Pages {
		for _, n := range p.Definitions {
			d := r.byName[n]
			sum := strings.ReplaceAll(r.markdown(d.Summary, n), "|", `\|`)
			fmt.Fprintf(&b, "| [`%s`](%s) | %s | %s |\n", n, r.link(n), p.Title, sum)
		}
	}
	tail := b.String()
	intro := "Every definition below belongs to `" + r.m.ModulePath + "`, the CUE module every OPM artifact is typed against. " +
		"An entry's summary, notes and field comments come from the definition's doc comment; " +
		"its spec, the definitions it uses and the rules CUE enforces are read off the CUE source."
	body, err := execute("defs-index.md.tmpl", map[string]any{
		"Title": r.m.Title, "Description": r.m.Description, "Weight": r.m.Weight,
		"Intro": intro, "Tail": strings.TrimRight(tail, "\n"),
	})
	if err != nil {
		return Page{}, err
	}
	return Page{Path: r.m.Section + "_index.md", Body: body, Completable: true, Heading: pagesHeading, Tail: tail}, nil
}

func (r *defsRender) page(p cuedefs.Page) (string, error) {
	entries := make([]string, 0, len(p.Definitions))
	for _, n := range p.Definitions {
		d, ok := r.byName[n]
		if !ok {
			return "", fmt.Errorf("page lists %s, which the model does not define", n)
		}
		entries = append(entries, r.entry(d))
	}
	return execute("defs-page.md.tmpl", map[string]any{
		"Title": p.Title, "Description": p.Description, "Weight": p.Weight,
		"OnPage": r.linkList(p.Definitions), "Index": r.indexURL(), "Entries": entries,
	})
}

// entry writes one definition in a fixed order: summary, at a glance,
// spec, example, notes, enforcement.
func (r *defsRender) entry(d *cuedefs.Definition) string {
	md := func(s string) string { return r.markdown(s, d.Name) }
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\n", d.Name)
	if d.Summary != "" {
		b.WriteString(md(d.Summary) + "\n\n")
	}
	b.WriteString("**At a glance**\n\n")
	fmt.Fprintf(&b, "- Source: `%s` in `%s`\n", d.File, r.m.ModulePath)
	fmt.Fprintf(&b, "- Shape: %s\n", d.Shape)
	if len(d.Embeds) > 0 {
		fmt.Fprintf(&b, "- Embeds: %s\n", r.linkList(d.Embeds))
	}
	if len(d.Uses) > 0 {
		fmt.Fprintf(&b, "- Uses: %s\n", r.linkList(d.Uses))
	}
	if len(d.UsedBy) > 0 {
		fmt.Fprintf(&b, "- Used by: %s\n", r.linkList(d.UsedBy))
	}
	fmt.Fprintf(&b, "\n**Spec**\n\n```cue\n%s\n```\n", d.CUE)
	if len(d.Example) > 0 {
		b.WriteString("\n**Example**\n\n```cue\n")
		for _, l := range d.Example {
			b.WriteString(l + "\n")
		}
		b.WriteString("```\n")
	}
	if len(d.Notes) > 0 {
		b.WriteString("\n**Notes**\n\n")
		b.WriteString(notesMarkdown(d.Notes, md))
	}
	if len(d.Rules) > 0 {
		fmt.Fprintf(&b, "\n**Enforcement**\n\nCUE enforces each of these rules on a value unified with `%s`:\n\n", d.Name)
		for _, rl := range d.Rules {
			fmt.Fprintf(&b, "- %s\n", md(rl.Rule))
			if rl.Code != "" {
				fmt.Fprintf(&b, "\n  ```text\n  %s\n  ```\n\n", rl.Code)
			}
		}
	}
	return b.String()
}

// notesMarkdown writes note blocks as Markdown: a paragraph as prose, a
// "- " entry as a list item, an indented entry as a line of a text block,
// and "" as a break that closes an open text block.
func notesMarkdown(notes []string, md func(string) string) string {
	var b strings.Builder
	inText, prevList := false, false
	closeText := func() {
		if inText {
			b.WriteString("```\n")
			inText = false
		}
	}
	for _, n := range notes {
		switch {
		case n == "":
			closeText()
			continue
		case strings.HasPrefix(n, "- "):
			closeText()
			if !prevList && b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString("- " + md(strings.TrimPrefix(n, "- ")) + "\n")
			prevList = true
			continue
		case strings.HasPrefix(n, "  "):
			if !inText {
				if b.Len() > 0 {
					b.WriteString("\n")
				}
				b.WriteString("```text\n")
				inText = true
			}
			b.WriteString(strings.TrimSpace(n) + "\n")
		default:
			closeText()
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(md(n) + "\n")
		}
		prevList = false
	}
	closeText()
	return b.String()
}

// reDollarRef matches a "$"-prefixed field name in prose: "$secretName".
var reDollarRef = regexp.MustCompile(`\$[A-Za-z_][A-Za-z0-9_]*`)

// reDefRef matches a definition reference in prose: "#Trait",
// "#Trait.optional", "#ctx.components".
var reDefRef = regexp.MustCompile(`#[A-Za-z_][A-Za-z0-9_]*(?:\.[#$A-Za-z_][A-Za-z0-9_]*)*`)

// markdown renders one line of definition prose: outside code spans it
// links a reference to a placed definition other than self, puts any
// other "#name" and "$name" in a code span, and escapes "<" and "{{" so no
// text reads as raw HTML or a shortcode. The prose is Markdown as written
// in the doc comment, so nothing else is escaped.
func (r *defsRender) markdown(s, self string) string {
	var b strings.Builder
	parts := strings.Split(s, "`")
	for i, p := range parts {
		if i > 0 {
			b.WriteString("`")
		}
		if i%2 == 1 && i < len(parts)-1 {
			b.WriteString(p) // inside a code span
			continue
		}
		p = strings.ReplaceAll(p, "<", `\<`)
		p = strings.ReplaceAll(p, "{{", `{\{`)
		p = reDefRef.ReplaceAllStringFunc(p, func(m string) string {
			base := m
			if j := strings.Index(m, "."); j > 0 {
				base = m[:j]
			}
			if base != self {
				if u := r.link(base); u != "" {
					return "[`" + m + "`](" + u + ")"
				}
			}
			return "`" + m + "`"
		})
		p = reDollarRef.ReplaceAllString(p, "`$0`")
		b.WriteString(p)
	}
	return b.String()
}
