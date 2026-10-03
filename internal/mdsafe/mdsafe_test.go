package mdsafe

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/open-platform-model/docs-kit/internal/extract/enhancements"
	"github.com/open-platform-model/docs-kit/internal/render"
)

func check(body string) []Violation {
	return Check(Page{Path: "p.md", Body: []byte(body)}, Authored)
}

func TestCheckRefuses(t *testing.T) {
	for _, c := range []struct{ name, body, msg string }{
		{"inline html", "a <b>bold</b>\n", "p.md:1: raw HTML"},
		{"html block", "para\n\n<div>\nx\n</div>\n", "p.md:3: an HTML block"},
		{"comment", "a <!-- c --> b\n", "p.md:1: raw HTML"},
		{"script link", "[x](javascript:alert(1))\n", `p.md:1: link "javascript:alert(1)" has the scheme javascript:`},
		{"upper case", "[x](JAVASCRIPT:alert(1))\n", "has the scheme JAVASCRIPT:"},
		{"entity", "[x](javascript&colon;alert(1))\n", "has the scheme javascript:"},
		{"numeric reference", "[x](javascript&#58;alert(1))\n", "has the scheme javascript:"},
		{"image", "![x](data:image/svg+xml,abc)\n", `image "data:image/svg+xml,abc" has the scheme data:`},
		{"autolink", "<vbscript:x>\n", "autolink"},
		{"reference definition", "[r]: javascript:x\n", "reference definition"},
		{"protocol-relative", "[x](//evil.example/)\n", "is protocol-relative"},
		{"heading attributes", "## Title {onclick=\"x\"}\n", "p.md:1: a heading attribute block"},
		{"front matter offset", "---\ntitle: \"t\"\n---\n\n<b>x</b>\n", "p.md:5: raw HTML"},
	} {
		t.Run(c.name, func(t *testing.T) {
			vs := check(c.body)
			if len(vs) == 0 || !strings.Contains(vs[0].String(), c.msg) {
				t.Fatalf("violations %v, want one naming %q", vs, c.msg)
			}
		})
	}
}

func TestCheckAllows(t *testing.T) {
	body := "---\ntitle: \"t\"\ndescription: \"d\"\n---\n\n## Heading\n\n" +
		"[a](https://x.example/) [b](http://x.example/) [c](mailto:a@b.example) [d](/docs/x/) [e](#f) [g](rel.md)\n" +
		"<https://x.example/> www.x.example `<b>code</b>` \\<b> a < b\n\n" +
		"```html\n<script>x</script>\n```\n\n[r]: https://x.example/r\n"
	if vs := check(body); len(vs) != 0 {
		t.Fatalf("refused %v", vs)
	}
}

// inert reports HTML that holds no element or URL a browser would act on
// beyond what Markdown itself writes.
var reActive = regexp.MustCompile(`(?i)<(svg|img|details|script|iframe|object|embed|style)\b|<!\[CDATA|<[a-z]+(\s+[a-z-]+(="[^"]*")?)*\s+on[a-z]+\s*=|(href|src)="\s*(javascript|vbscript|data):`)

func toHTML(t *testing.T, body []byte) []byte {
	t.Helper()
	md := newMarkdown(goldmark.WithRendererOptions(html.WithUnsafe()))
	var out bytes.Buffer
	b, _ := splitFrontMatter(body)
	if err := md.Convert(b, &out); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func probes(t *testing.T) map[string][]byte {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("testdata", "probes", "*.md"))
	if err != nil || len(files) != 24 {
		t.Fatalf("%d probes: %v", len(files), err)
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

// TestProbesRaw runs the review's probes of docs-kit#36 through the check
// alone, with no transform: each is refused, or renders as inert text.
func TestProbesRaw(t *testing.T) {
	for name, body := range probes(t) {
		t.Run(name, func(t *testing.T) {
			if vs := Check(Page{Path: name, Body: body}, Authored); len(vs) > 0 {
				return
			}
			if h := toHTML(t, body); reActive.Match(h) {
				t.Fatalf("accepted, and renders active HTML:\n%s", h)
			}
		})
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
				if vs := Check(Page{Path: p.Path, Body: []byte(p.Body)}, Authored); len(vs) > 0 {
					return
				}
				if h := toHTML(t, []byte(p.Body)); reActive.Match(h) {
					t.Fatalf("accepted, and renders active HTML:\n%s", h)
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
