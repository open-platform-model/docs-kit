package render

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/extract/cobra"
	"github.com/open-platform-model/docs-kit/internal/extract/crd"
	"github.com/open-platform-model/docs-kit/internal/extract/cuecatalog"
	"github.com/open-platform-model/docs-kit/internal/extract/cuedefs"
	"github.com/open-platform-model/docs-kit/internal/extract/goapi"
)

// A Renderer turns one data file into pages. It reads the data file as
// written, never the extractor's value.
type Renderer interface {
	Schema() string
	Render(data []byte, t Target) ([]Page, error)
}

// renderers is every renderer this opm-docs carries, keyed by the data
// schema it reads.
var renderers = map[string]Renderer{
	cuecatalog.SchemaID: catalogRenderer{},
	cuedefs.SchemaID:    defsRenderer{},
	cobra.SchemaID:      cobraRenderer{},
	crd.SchemaID:        crdRenderer{},
	goapi.SchemaID:      goAPIRenderer{},
}

// For returns the renderer of a data schema.
func For(schema string) (Renderer, error) {
	r, ok := renderers[schema]
	if !ok {
		return nil, fmt.Errorf("no renderer reads data schema %q", schema)
	}
	return r, nil
}

// Complete is an authored page completed by a generated one: the authored
// front matter and body, one blank line, then p's Tail. An authored body
// that already holds p's Heading, or one of its Headings, is refused; name
// is the authored file.
func Complete(authored, name string, p Page) (string, error) {
	for _, h := range append([]string{p.Heading}, p.Headings...) {
		if headingRE(h).MatchString(authored) {
			return "", fmt.Errorf("%s already holds a %q heading; the build appends that section, so remove it from the authored page", name, h)
		}
	}
	return strings.TrimRight(authored, "\n") + "\n\n" + p.Tail, nil
}

// headingRE matches a heading line of the same level and text, whatever
// the spacing after the hashes or at the end of the line.
func headingRE(h string) *regexp.Regexp {
	level := strings.TrimRight(h[:len(h)-len(strings.TrimLeft(h, "#"))], " ")
	text := strings.TrimSpace(strings.TrimLeft(h, "#"))
	return regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(level) + `[ \t]+` + regexp.QuoteMeta(text) + `[ \t]*$`)
}

// catalogRenderer renders a cue-catalog data file: the kind indexes, the
// member pages and the landing, which an authored landing may complete.
type catalogRenderer struct{}

func (catalogRenderer) Schema() string { return cuecatalog.SchemaID }

func (catalogRenderer) Render(data []byte, t Target) ([]Page, error) {
	m, err := cuecatalog.Decode(data)
	if err != nil {
		return nil, err
	}
	pages, err := Catalog(m, t)
	if err != nil {
		return nil, err
	}
	alone, err := Landing(m, t, "", "")
	if err != nil {
		return nil, err
	}
	block, err := Block(m, t)
	if err != nil {
		return nil, err
	}
	return append(pages, Page{Path: "_index.md", Body: alone, Completable: true, Heading: membersHeading, Tail: block}), nil
}
