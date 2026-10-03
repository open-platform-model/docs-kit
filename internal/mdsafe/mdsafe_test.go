package mdsafe

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
	htmltok "golang.org/x/net/html"

	"github.com/open-platform-model/docs-kit/internal/extract/enhancements"
	"github.com/open-platform-model/docs-kit/internal/render"
)

var modes = map[Mode]string{Generated: "generated", Authored: "authored"}

func check(body string, mode Mode) []Violation {
	return Check(Page{Path: "p.md", Body: []byte(body)}, mode)
}

// TestCheckRefuses holds what both modes refuse.
func TestCheckRefuses(t *testing.T) {
	for _, c := range []struct{ name, body, msg string }{
		{"inline html", "a <b>bold</b>\n", "p.md:1: raw HTML"},
		{"html block", "para\n\n<div>\nx\n</div>\n", "p.md:3: an HTML block"},
		{"script link", "[x](javascript:alert(1))\n", `p.md:1: link "javascript:alert(1)" has the scheme javascript:`},
		{"upper case", "[x](JAVASCRIPT:alert(1))\n", "has the scheme JAVASCRIPT:"},
		{"entity", "[x](javascript&colon;alert(1))\n", "has the scheme javascript:"},
		{"numeric reference", "[x](javascript&#58;alert(1))\n", "has the scheme javascript:"},
		{"image", "![x](data:image/svg+xml,abc)\n", `image "data:image/svg+xml,abc" has the scheme data:`},
		{"autolink", "<vbscript:x>\n", "autolink"},
		{"reference definition", "[r]: javascript:x\n", "reference definition"},
		{"protocol-relative", "[x](//evil.example/)\n", "is protocol-relative"},
		{"heading attributes", "## Title {onclick=\"x\"}\n", "p.md:1: a heading attribute block"},
		{"context marker in a code block", "    {{__hugo_ctx/}} x\n", "p.md:1: Hugo's internal context marker"},
		{"backslash colon", "[x](javascript\\:alert(1))\n", "has the scheme javascript:"},
		{"double reference", "[x](javascript&#38;colon;alert(1))\n", "has the scheme javascript:"},
		{"backslash then reference", "[x](javascript\\&#58;alert(1))\n", "has the scheme javascript:"},
		{"front matter offset", "---\ntitle: \"t\"\n---\n\n<b>x</b>\n", "p.md:5: raw HTML"},
		{"comment closed early by a browser", "<!-- a --!> <svg onload=alert(1)> -->\n", "p.md:1: an HTML block"},
		{"abrupt comment", "<!--> <svg onload=alert(1)> -->\n", "p.md:1: an HTML block"},
		{"abrupt comment with a dash", "<!---> <svg onload=alert(1)> -->\n", "p.md:1: an HTML block"},
		{"markup after a comment", "<!-- a --> <b>x</b>\n", "p.md:1: an HTML block"},
		{"unclosed comment block", "para\n\n<!-- a\n<b>x</b>\n", "p.md:3: an HTML block"},
		{"inline comment closed early", "x <!-- a --!> <b>y</b> --> z\n", "p.md:1: raw HTML"},
		{"comment then a tag inline", "x <!-- a --><b>y</b>\n", "p.md:1: raw HTML"},
	} {
		for mode, m := range modes {
			t.Run(m+"/"+c.name, func(t *testing.T) {
				vs := check(c.body, mode)
				if len(vs) == 0 || !strings.Contains(vs[0].String(), c.msg) {
					t.Fatalf("violations %v, want one naming %q", vs, c.msg)
				}
			})
		}
	}
}

// TestCheckComments: a generated page holds no comment; an authored page
// may hold comments a browser closes where goldmark does.
func TestCheckComments(t *testing.T) {
	for _, c := range []struct{ name, body string }{
		{"inline", "a <!-- c --> b\n"},
		{"block", "para\n\n<!-- Check against: core/src/x.cue\nand more -->\n\nnext\n"},
		{"two in a block", "<!-- a --> <!-- b -->\n"},
		{"in a table cell", "| a | b <!-- c --> |\n|---|---|\n| d | e |\n"},
		{"in a list item", "1. step\n\n   <!-- note -->\n"},
		{"dashes inside", "<!-- a -- b --->\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if vs := check(c.body, Authored); len(vs) != 0 {
				t.Errorf("authored: refused %v", vs)
			}
			if vs := check(c.body, Generated); len(vs) == 0 || !strings.Contains(vs[0].Msg, "HTML") || strings.Contains(vs[0].Msg, "comment") {
				t.Errorf("generated: violations %v, want raw HTML refused", vs)
			}
		})
	}
	if vs := check("<b>x</b>\n", Authored); len(vs) == 0 || !strings.Contains(vs[0].Msg, "other than an HTML comment") {
		t.Errorf("authored: violations %v, want the message to name what passes", vs)
	}
}

func TestCheckAllows(t *testing.T) {
	body := "---\ntitle: \"t\"\ndescription: \"d\"\n---\n\n## Heading\n\n" +
		"[a](https://x.example/) [b](http://x.example/) [c](mailto:a@b.example) [d](/docs/x/) [e](#f) [g](rel.md)\n" +
		"<https://x.example/> www.x.example `<b>code</b>` \\<b> a < b\n\n" +
		"## Install \\{.hx:fixed\\}\n\n" +
		"```html\n<script>x</script>\n```\n\n[r]: https://x.example/r\n"
	for mode, m := range modes {
		if vs := check(body, mode); len(vs) != 0 {
			t.Fatalf("%s: refused %v", m, vs)
		}
	}
}

// emitted is every element goldmark (with Hugo's extensions) writes for
// Markdown, so any other element in a rendered page came from raw HTML.
var emitted = func() map[string]bool {
	m := map[string]bool{}
	for _, e := range strings.Fields("p h1 h2 h3 h4 h5 h6 a em strong code pre ul ol li blockquote hr br img " +
		"table thead tbody tr th td del input dl dt dd sup div section") {
		m[e] = true
	}
	return m
}()

// urlAttrs are the attributes whose value a browser loads or follows.
var urlAttrs = map[string]bool{"href": true, "src": true, "action": true, "formaction": true, "xlink:href": true, "poster": true, "data": true}

// active reads rendered HTML as a browser tokenizes it and names the first
// thing in it a browser would act on beyond what Markdown writes: an
// element goldmark does not emit, an event handler attribute, or a URL
// that is not http, https, mailto, a fragment or root-absolute. allow adds
// elements a site's templates emit. Comments are tokens of their own, so a
// comment's text is inert, and one a browser closes early is not.
func active(h []byte, allow map[string]bool) string {
	z := htmltok.NewTokenizer(bytes.NewReader(h))
	for {
		switch z.Next() {
		case htmltok.ErrorToken:
			return ""
		case htmltok.TextToken, htmltok.EndTagToken, htmltok.CommentToken, htmltok.DoctypeToken:
		case htmltok.StartTagToken, htmltok.SelfClosingTagToken:
			tok := z.Token()
			if !emitted[tok.Data] && !allow[tok.Data] {
				return "element <" + tok.Data + ">"
			}
			for _, a := range tok.Attr {
				k := strings.ToLower(a.Key)
				if a.Namespace != "" {
					k = a.Namespace + ":" + k
				}
				if strings.HasPrefix(k, "on") {
					return "attribute " + k + " on <" + tok.Data + ">"
				}
				if urlAttrs[k] && !safeURL(a.Val) {
					return fmt.Sprintf("%s=%q on <%s>", k, a.Val, tok.Data)
				}
			}
		}
	}
}

func safeURL(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	if strings.HasPrefix(v, "//") {
		return false
	}
	for _, p := range []string{"http:", "https:", "mailto:", "#", "/"} {
		if strings.HasPrefix(v, p) {
			return true
		}
	}
	return false
}

func TestActive(t *testing.T) {
	for h, want := range map[string]bool{
		`<p><a href="https://x.example/">x</a> <code>&lt;b&gt;</code></p>`: false,
		`<!-- <svg onload=alert(1)> -->`:                                   false,
		`<h2 id="x">t</h2><input disabled="" type="checkbox">`:             false,
		`<!-- a --!> <svg onload=alert(1)> -->`:                            true,
		`<!--> <svg> -->`:                                                  true,
		`<p onclick="x">t</p>`:                                             true,
		`<a href=" JavaScript:x">t</a>`:                                    true,
		`<a href="rel.md">t</a>`:                                           true,
		`<a href="//evil.example/">t</a>`:                                  true,
		`<details open>`:                                                   true,
	} {
		if got := active([]byte(h), nil) != ""; got != want {
			t.Errorf("active(%q) = %v, want %v", h, got, want)
		}
	}
}

func toHTML(t *testing.T, body []byte) []byte {
	t.Helper()
	md := newMarkdown(goldmark.WithRendererOptions(html.WithUnsafe()))
	var out bytes.Buffer
	b, _ := splitFrontMatter(body)
	if err := md.Convert(dedentMarkers(b), &out); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// probeCount is the number of probes under testdata/probes: the docs-kit#36
// reviews' 33, and the comment forms of the authored rules.
const probeCount = 40

func probes(t *testing.T) map[string][]byte {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("testdata", "probes", "*.md"))
	if err != nil || len(files) != probeCount {
		t.Fatalf("%d probes, want %d: %v", len(files), probeCount, err)
	}
	out := map[string][]byte{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		out[filepath.Base(f)] = b
	}
	return out
}

// TestProbesRaw runs every probe through the check alone, with no
// transform, in both modes: each is refused, or renders as inert text.
func TestProbesRaw(t *testing.T) {
	for name, body := range probes(t) {
		h := toHTML(t, body)
		for mode, m := range modes {
			t.Run(m+"/"+name, func(t *testing.T) {
				if vs := Check(Page{Path: name, Body: body}, mode); len(vs) > 0 {
					t.Logf("refused: %s", vs[0].Msg)
					return
				}
				if a := active(h, nil); a != "" {
					t.Fatalf("accepted, and renders %s:\n%s", a, h)
				}
			})
		}
	}
}

// TestProbesThroughTheSection puts each probe into an enhancement document
// and runs the whole section: the transforms refuse it, the check refuses
// the page, or the page renders as inert text.
func TestProbesThroughTheSection(t *testing.T) {
	for name, body := range probes(t) {
		t.Run(name, func(t *testing.T) {
			root := fixture(t)
			if err := os.WriteFile(filepath.Join(root, "0025", "06-operational.md"), append([]byte("# Operational\n\n"), body...), 0o600); err != nil {
				t.Fatal(err)
			}
			res, err := enhancements.Extract(enhancements.Options{Root: root, Dir: ".", Repo: "open-platform-model/enhancements",
				Commit: strings.Repeat("c", 40), Paths: paths(t, root), Title: "Enhancements", Description: "d"})
			if err != nil {
				return // refused by a transform
			}
			pages, err := render.Enhancements(res.Pages, render.Target{Kind: render.KindSection, Root: "/enhancements/"})
			if err != nil {
				return
			}
			for _, p := range pages {
				if p.Path != "0025/operational.md" {
					continue
				}
				if vs := Check(Page{Path: p.Path, Body: []byte(p.Body)}, Generated); len(vs) > 0 {
					return
				}
				if h := toHTML(t, []byte(p.Body)); active(h, nil) != "" {
					t.Fatalf("accepted, and renders %s:\n%s", active(h, nil), h)
				}
			}
		})
	}
}

const fixtureRepo = "../extract/enhancements/testdata/repo"

func fixture(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir(fixtureRepo, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(fixtureRepo, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o750)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func paths(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == root {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		out[filepath.ToSlash(rel)] = map[bool]string{true: "tree", false: "blob"}[d.IsDir()]
		return nil
	})
	return out
}
