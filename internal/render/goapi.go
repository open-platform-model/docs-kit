package render

import (
	"fmt"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/extract/goapi"
	"github.com/open-platform-model/docs-kit/internal/mdtext"
)

// packagesHeading opens the Go API index's generated tail, which an
// authored index may complete.
const packagesHeading = "## Packages"

// goAPIRenderer renders a go-api data file: the section index and one page
// per package (docs-kit C20).
type goAPIRenderer struct{}

func (goAPIRenderer) Schema() string { return goapi.SchemaID }

func (goAPIRenderer) Render(data []byte, t Target) ([]Page, error) {
	m, err := goapi.Decode(data)
	if err != nil {
		return nil, err
	}
	return GoAPI(m, t)
}

// GoAPI renders the section index and every package page of a Go API
// model. Its pages and doc links live under /docs/, so it renders only
// into a docs bundle.
func GoAPI(m *goapi.Model, t Target) ([]Page, error) {
	if t.Kind != KindDocs {
		return nil, fmt.Errorf("go-api renders reference pages under /docs/; give the bundle placement kind \"docs\"")
	}
	index, err := goAPIIndex(m, t)
	if err != nil {
		return nil, err
	}
	pages := []Page{index}
	for i := range m.Packages {
		p := &m.Packages[i]
		body, err := goAPIPage(m, p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.ImportPath, err)
		}
		pages = append(pages, Page{Path: m.Section + p.Page + ".md", Body: body})
	}
	for _, p := range pages {
		if err := mdtext.CheckShortcodes(p.Path, p.Body); err != nil {
			return nil, err
		}
	}
	return pages, nil
}

// relPath is a package's import path relative to its module: the page
// title, "opm/kernel".
func relPath(m *goapi.Model, p *goapi.Package) string {
	return strings.TrimPrefix(p.ImportPath, m.ModulePath+"/")
}

func goAPIIndex(m *goapi.Model, t Target) (Page, error) {
	var b strings.Builder
	b.WriteString(packagesHeading + "\n\n")
	for i := range m.Packages {
		p := &m.Packages[i]
		fmt.Fprintf(&b, "- [%s](%s)", goapi.Prose(relPath(m, p)), t.URL(m.Section+p.Page))
		if p.Synopsis != "" {
			b.WriteString(": " + goapi.Prose(p.Synopsis))
		}
		b.WriteString("\n")
	}
	tail := b.String()
	weight := 0
	if m.Weight != nil {
		weight = *m.Weight
	}
	body, err := execute("goapi-index.md.tmpl", map[string]any{
		"Title": m.Title, "Description": m.Description, "Weight": weight,
		"Intro": "Every package below belongs to the Go module " + mdtext.Code(m.ModulePath) + ".",
		"Tail":  strings.TrimRight(tail, "\n"),
	})
	if err != nil {
		return Page{}, err
	}
	return Page{Path: m.Section + "_index.md", Body: body, Completable: true, Heading: packagesHeading, Tail: tail}, nil
}

// goAPIPage writes a package page: the import line, the package doc, then
// the sections in goapi.PageSections' order, which is the order the
// extractor assigned the anchors in.
func goAPIPage(m *goapi.Model, p *goapi.Package) (string, error) {
	desc := p.Synopsis
	if desc == "" {
		desc = "Package " + p.Name + "."
	}
	imp := fmt.Sprintf("import %q", p.ImportPath)
	f := goapi.Fence(imp)
	secs := goapi.PageSections(p)
	sections := make([]string, 0, len(secs))
	for _, s := range secs {
		var b strings.Builder
		b.WriteString("## " + s.Heading)
		for _, e := range s.Entries {
			f := goapi.Fence(e.Decl)
			fmt.Fprintf(&b, "\n\n%s %s\n\n%sgo\n%s\n%s", strings.Repeat("#", e.Level), e.Heading, f, e.Decl, f)
			if e.Doc != "" {
				b.WriteString("\n\n" + e.Doc)
			}
		}
		sections = append(sections, b.String())
	}
	return execute("goapi-page.md.tmpl", map[string]any{
		"Title": relPath(m, p), "Description": desc,
		"Import": f + "go\n" + imp + "\n" + f, "Doc": p.Doc, "Sections": sections,
	})
}
