package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"github.com/open-platform-model/docs-kit/internal/extract/goapi"
)

// hugoIDs are the heading ids Hugo gives a page: goldmark parses it with
// heading attributes on (Hugo's default), and each heading without an id
// attribute gets the github-style anchor of its plain text, made unique.
// The function returns them in page order, and fails on a heading that
// carries an attribute, which no generated page may.
func hugoIDs(t *testing.T, name, page string) []string {
	t.Helper()
	body := page
	if strings.HasPrefix(body, "---\n") {
		if i := strings.Index(body[4:], "\n---\n"); i >= 0 {
			body = body[4+i+5:]
		}
	}
	src := []byte(body)
	md := goldmark.New(goldmark.WithParserOptions(parser.WithAttribute()))
	doc := md.Parser().Parse(text.NewReader(src))
	var a goapi.Anchors
	var ids []string
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		h, ok := n.(*ast.Heading)
		if !ok || !entering {
			return ast.WalkContinue, nil
		}
		if len(h.Attributes()) > 0 {
			t.Errorf("%s: a heading carries attributes: %q", name, plainText(h, src))
		}
		ids = append(ids, a.Next(goapi.Anchor(plainText(h, src))))
		return ast.WalkSkipChildren, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

// plainText is a node's text as Hugo's TextPlain reads it: text and code
// span content, without markup.
func plainText(n ast.Node, src []byte) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c := c.(type) {
		case *ast.Text:
			b.Write(c.Segment.Value(src))
			if c.SoftLineBreak() {
				b.WriteByte('\n')
			}
		case *ast.String:
			b.Write(c.Value)
		default:
			b.WriteString(plainText(c, src))
		}
	}
	return b.String()
}

// The anchors the extractor computes (goapi.Headings, HeadingAnchor) are
// the ids goldmark and Hugo's id rule give the rendered pages, for the
// fixture and the frozen library.
func TestGoAPIHeadingIDsMatchGoldmark(t *testing.T) {
	for _, dir := range []string{"go-api", "library"} {
		root := filepath.Join("testdata", "golden", dir)
		err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			var a goapi.Anchors
			var ours []string
			for _, h := range goapi.Headings(string(b)) {
				ours = append(ours, a.Next(goapi.HeadingAnchor(h)))
			}
			got := hugoIDs(t, p, string(b))
			if strings.Join(got, " ") != strings.Join(ours, " ") {
				t.Errorf("%s: goldmark ids\n%v\nextractor anchors\n%v", p, got, ours)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// A heading the printer escaped carries no attribute block.
func TestGoAPIHeadingAttributesEscaped(t *testing.T) {
	for _, h := range []string{`### Title \{.class style=color:red\}`, `### Title \{\#kernel\}`, "### Use \\`x \\{id=y\\}\\`"} {
		ids := hugoIDs(t, h, h+"\n")
		if len(ids) != 1 || ids[0] == "kernel" {
			t.Errorf("%s: ids %v", h, ids)
		}
	}
	// The probe itself: unescaped, goldmark reads the attributes.
	md := goldmark.New(goldmark.WithParserOptions(parser.WithAttribute()))
	src := []byte("### Title {#kernel}\n")
	doc := md.Parser().Parse(text.NewReader(src))
	if id, ok := doc.FirstChild().AttributeString("id"); !ok || string(id.([]byte)) != "kernel" {
		t.Errorf("goldmark ignores heading attributes here (%v); the escape test proves nothing", id)
	}
}
