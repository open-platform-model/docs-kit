// Package mdsafe checks a page the way the site's Markdown renderer reads
// it: it parses the body with goldmark configured as Hugo configures it
// for opmodel.dev and refuses what would put raw HTML or a script URL on
// the published page. Source text transforms are best effort; this check
// is the guarantee.
//
// goldmark is pinned to the version Hugo 0.167.0 (opmodel.dev's pinned
// Hugo) builds with, v1.8.6, so both read a page alike. The parser matches
// Hugo's defaults, which opmodel.dev's markup.goldmark config leaves in
// place (it sets only renderer.unsafe = true): GFM tables, strikethrough,
// linkify and task lists, definition lists, footnotes, the typographer,
// and attributes on headings.
package mdsafe

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// The versions the check is pinned to: goldmark as Hugo HugoVersion builds
// with it. A test compares GoldmarkVersion with the goldmark this binary is
// built with, so a dependency bump that drifts from the site's Hugo fails.
const (
	HugoVersion     = "0.167.0"
	GoldmarkVersion = "v1.8.6"
)

// Mode selects the pages a check is for. Today every mode applies the same
// rules; the mode names the producer, so a later rule can differ by it
// (docs-kit#27 runs the check on every generated page).
type Mode int

const (
	// Authored is text a repository wrote, transformed by a source (the
	// enhancements section's pages).
	Authored Mode = iota
	// Generated is text a renderer wrote from a doc model.
	Generated
)

// Page is one page under content/: its path, for messages, and its whole
// text, front matter included.
type Page struct {
	Path string
	Body []byte
}

// Violation is one refused construct.
type Violation struct {
	Path string
	Line int
	Msg  string
}

func (v Violation) String() string { return fmt.Sprintf("%s:%d: %s", v.Path, v.Line, v.Msg) }

// markdown is goldmark as Hugo configures it for opmodel.dev. Heading
// IDs are not generated, so any heading attribute is an author's block.
var markdown = newMarkdown()

func newMarkdown(opts ...goldmark.Option) goldmark.Markdown {
	return goldmark.New(append([]goldmark.Option{
		goldmark.WithExtensions(
			extension.Table, extension.Strikethrough, extension.Linkify, extension.TaskList,
			extension.DefinitionList, extension.Footnote, extension.Typographer,
		),
		goldmark.WithParserOptions(parser.WithAttribute()),
	}, opts...)...)
}

// Check parses a page and returns every construct it refuses, in source
// order: Hugo's internal context marker, raw HTML (inline or a block,
// comments included), a link, image, autolink or reference definition
// whose destination has a scheme other than http, https or mailto or is
// protocol-relative ("//host"), and a heading attribute block.
//
// Hugo rewrites a page before goldmark sees it: it un-indents every line
// that starts with its context marker, "{{__hugo_ctx" (hugocontext
// DedentMarkers, Hugo 0.167.0), so an indented code block can become live
// markup. The marker is refused wherever it appears, code included, and
// the body is parsed as Hugo would parse it after that step.
func Check(p Page, _ Mode) []Violation {
	body, offset := splitFrontMatter(p.Body)
	c := &checker{page: p, src: dedentMarkers(body), offset: offset}
	c.markers(body)
	doc := markdown.Parser().Parse(text.NewReader(c.src))
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			c.node(n)
		}
		return ast.WalkContinue, nil
	})
	sort.SliceStable(c.out, func(i, j int) bool { return c.out[i].Line < c.out[j].Line })
	return c.out
}

// hugoMarker opens Hugo's internal context marker, and reIndentedMarker
// is a marker line Hugo un-indents (hugocontext.hugoCtxIndentedRe).
var (
	hugoMarker       = []byte("{{__hugo_ctx")
	reIndentedMarker = regexp.MustCompile(`(?m)^[ \t]+({{__hugo_ctx[^\n]*}})`)
)

// dedentMarkers is Hugo's DedentMarkers: it removes the indentation of
// every line that starts with a context marker. It keeps every line.
func dedentMarkers(b []byte) []byte {
	if !bytes.Contains(b, hugoMarker) {
		return b
	}
	return reIndentedMarker.ReplaceAll(b, []byte("$1"))
}

type checker struct {
	page   Page
	src    []byte
	offset int // lines of front matter before src
	out    []Violation
}

// markers refuses every line of body holding Hugo's context marker.
func (c *checker) markers(body []byte) {
	for i, l := range bytes.Split(body, []byte("\n")) {
		if bytes.Contains(l, hugoMarker) {
			c.out = append(c.out, Violation{Path: c.page.Path, Line: c.offset + i + 1,
				Msg: "Hugo's internal context marker {{__hugo_ctx; Hugo rewrites the page around it, so remove it"})
		}
	}
}

func (c *checker) add(n ast.Node, msg string) {
	c.out = append(c.out, Violation{Path: c.page.Path, Line: c.offset + c.line(n), Msg: msg})
}

func (c *checker) node(n ast.Node) {
	switch n := n.(type) {
	case *ast.RawHTML:
		c.add(n, "raw HTML; the site would render it, so write it as Markdown or in a code span")
	case *ast.HTMLBlock:
		c.add(n, "an HTML block; the site would render it, so write it as Markdown or in a code fence")
	case *ast.Link:
		c.dest(n, "link", n.Destination)
	case *ast.Image:
		c.dest(n, "image", n.Destination)
	case *ast.AutoLink:
		c.dest(n, "autolink", n.URL(c.src))
	case *ast.LinkReferenceDefinition:
		c.dest(n, "reference definition", n.Destination)
	case *ast.Heading:
		if len(n.Attributes()) > 0 {
			c.add(n, "a heading attribute block ({...}); the site would set those attributes, so remove it")
		}
	}
}

// reScheme is a URL scheme, after the characters a browser drops.
var reScheme = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9+.-]*):`)

// dest refuses a destination with a scheme other than http, https or
// mailto, or one starting "//". The renderer decodes a destination as
// util.URLEscape(dest, true) does (backslash escapes, then numeric, then
// named character references), here repeated until nothing changes, and
// a browser strips leading control characters and blanks and drops tabs
// and line breaks inside a URL, so the scheme is read as both would.
func (c *checker) dest(n ast.Node, what string, d []byte) {
	r := decode(d)
	u := strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, string(r))
	u = strings.TrimLeftFunc(u, func(r rune) bool { return r <= ' ' })
	if strings.HasPrefix(u, "//") || strings.HasPrefix(u, `\\`) || strings.HasPrefix(u, `/\`) {
		c.add(n, fmt.Sprintf("%s %q is protocol-relative; link with http, https or mailto", what, string(d)))
		return
	}
	m := reScheme.FindStringSubmatch(u)
	if m == nil {
		return
	}
	switch strings.ToLower(m[1]) {
	case "http", "https", "mailto":
		return
	}
	c.add(n, fmt.Sprintf("%s %q has the scheme %s:; link with http, https or mailto", what, string(d), m[1]))
}

// decode applies goldmark's destination decoding until it is stable.
func decode(d []byte) []byte {
	for i := 0; i < 16; i++ {
		next := util.ResolveEntityNames(util.ResolveNumericReferences(util.UnescapePunctuations(d)))
		if bytes.Equal(next, d) {
			break
		}
		d = next
	}
	return d
}

// line is the 1-based line of the body a node starts on: its own position,
// else its first text segment's, else its nearest block's first line.
func (c *checker) line(n ast.Node) int {
	return bytes.Count(c.src[:c.pos(n)], []byte("\n")) + 1
}

func (c *checker) pos(n ast.Node) int {
	if p := n.Pos(); p >= 0 && p <= len(c.src) {
		return p
	}
	switch n := n.(type) {
	case *ast.RawHTML:
		if n.Segments.Len() > 0 {
			return n.Segments.At(0).Start
		}
	case *ast.Text:
		return n.Segment.Start
	}
	if n.Type() == ast.TypeBlock && n.Lines().Len() > 0 {
		return n.Lines().At(0).Start
	}
	for ch := n.FirstChild(); ch != nil; ch = ch.NextSibling() {
		if t, ok := ch.(*ast.Text); ok {
			return t.Segment.Start
		}
	}
	if p := n.Parent(); p != nil {
		return c.pos(p)
	}
	return 0
}

// splitFrontMatter returns the body after a front matter block that opens
// on line 1 with "---" and closes with "---", and the number of lines
// before it; a page without one is all body.
func splitFrontMatter(b []byte) (body []byte, before int) {
	if !bytes.HasPrefix(b, []byte("---\n")) {
		return b, 0
	}
	lines := bytes.SplitAfter(b, []byte("\n"))
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(string(lines[i]), " \t\r\n") == "---" {
			n := 0
			for _, l := range lines[:i+1] {
				n += len(l)
			}
			return b[n:], i + 1
		}
	}
	return b, 0
}
