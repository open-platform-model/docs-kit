package goapi

import (
	"go/doc/comment"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/doctext"
)

// printDoc parses a doc comment and prints it as the extractor does, every
// doc link resolved to /docs/x/.
func printDoc(t *testing.T, text string) string {
	t.Helper()
	p := &comment.Parser{LookupSym: func(_, name string) bool { return name == "Widget" }}
	pr := &mdPrinter{policy: doctext.Strip, docURL: func(*comment.DocLink) (string, bool) { return "/docs/x/", false }, where: "probe"}
	out, err := pr.markdown(p.Parse(text), 3)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// The site renders raw HTML and Hugo shortcodes, so source text is
// escaped; code spans, links and fences keep working.
func TestEscapeProbes(t *testing.T) {
	for _, c := range []struct{ name, in, want string }{
		{"raw HTML", "a <script>alert(1)</script> b", `a \<script\>alert(1)\</script\> b`},
		{"shortcode", "a {{< x >}} and {{{% y %}}}", `a \{\{\< x \>\}\} and \{\{\{% y %\}\}\}`},
		{"attribute block in prose", "a {.class style=x} b", `a \{.class style=x\} b`},
		{"heading attributes", "# Title {.class style=color:red}\n\ntext", "### Title \\{.class style=color:red\\}\n\ntext"},
		{"heading id", "# Title {#kernel}\n\ntext", "### Title \\{\\#kernel\\}\n\ntext"},
		{"heading code span", "# Use `x {id=y}`\n\ntext", "### Use \\`x \\{id=y\\}\\`\n\ntext"},
		{"control characters", "a \x01" + "0\x01 b\x00c\x7f", "a 0 bc"},
		{"code span", "use `a<b` here", "use `a<b` here"},
		// The doc comment parser reads `` and '' as curly quotes, so a
		// span of two backticks never reaches the printer.
		{"double backticks", "use ``a'' here", "use \u201ca\u201d here"},
		{"unpaired backtick", "a ` b", "a \\` b"},
		{"fence at line start", "```go then text```", "\\`\\`\\`go then text\\`\\`\\`"},
		{"emphasis and tables", "a *b* _c_ | d", `a \*b\* \_c\_ \| d`},
		{"image", "see ![Widget] now", `see \![Widget](/docs/x/) now`},
		{"list marker", "- not a list", `\- not a list`},
		{"ordered marker", "12. not a list", `12\. not a list`},
		{"heading marker", "#hash", `\#hash`},
		{"link text", "see [a *b* <c>].\n\n[a *b* <c>]: https://example.com/a(b)", `see [a \*b\* \<c\>](https://example.com/a%28b%29).`},
		{"doc link text", "the [*Widget] value", `the [\*Widget](/docs/x/) value`},
		{"doc link inside a code span", "a `x [Widget] y` b", "a `x [Widget] y` b"},
		{"brackets in a code span", "a `[Widget]` b", "a `[Widget]` b"},
		{"heading", "# A # {{<b>}}\n\ntext", "### A \\# \\{\\{\\<b\\>\\}\\}\n\ntext"},
		{"code block", "x:\n\n\t```\n\t{{ y }}\n", "x:\n\n````text\n```\n{{ y }}\n````"},
		{"why line", "Keep.\nWHY dropped.", "Keep. WHY dropped."},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := printDoc(t, c.in); got != c.want {
				t.Errorf("got\n%s\nwant\n%s", got, c.want)
			}
		})
	}
}

func TestLists(t *testing.T) {
	got := printDoc(t, "Items:\n  - one *a*\n  - two\n\nNumbered:\n\n 1. first\n\n 2. second\n")
	want := "Items:\n\n- one \\*a\\*\n- two\n\nNumbered:\n\n1. first\n\n2. second"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestFence(t *testing.T) {
	for in, want := range map[string]string{"x": "```", "a ``` b": "````", "````": "`````"} {
		if got := Fence(in); got != want {
			t.Errorf("Fence(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAnchors(t *testing.T) {
	for in, want := range map[string]string{
		"Kernel":                         "kernel",
		"Kernel.Render":                  "kernelrender",
		"Every operation shares nothing": "every-operation-shares-nothing",
		"Foo_Bar":                        "foo_bar",
		`A \# {\{\<b\>}}`:                "a--b",
		"[x](/docs/a/#b) y":              "x-y",
		"`code` span":                    "code-span",
	} {
		if got := HeadingAnchor(in); got != want {
			t.Errorf("HeadingAnchor(%q) = %q, want %q", in, got, want)
		}
	}
	var a Anchors
	got := []string{a.Next("x"), a.Next("x"), a.Next("x-1"), a.Next("x"), a.Next("")}
	if strings.Join(got, " ") != "x x-1 x-1-1 x-2 heading" {
		t.Errorf("unique anchors %v", got)
	}
}

func TestCheckURL(t *testing.T) {
	for u, ok := range map[string]bool{
		"https://x": true, "HTTP://x": true, "/docs/a/": false, "a/b": false, "#frag": false,
		"javascript:alert(1)": false, "file:///etc/passwd": false, "mailto:a@b": false, "data:x": false,
	} {
		if got := checkURL(u) == nil; got != ok {
			t.Errorf("checkURL(%q) ok = %v, want %v", u, got, ok)
		}
	}
}

// A doc link's URL is percent-encoded, so an import path that ends the
// link destination cannot inject markup.
func TestDocLinkURLEncoded(t *testing.T) {
	p := &comment.Parser{LookupPackage: func(name string) (string, bool) { return "x", name == "foo" }}
	pr := &mdPrinter{policy: doctext.Strip, where: "probe", docURL: func(*comment.DocLink) (string, bool) {
		return `https://pkg.go.dev/evil)<details open ontoggle=alert(1)>#Bar`, false
	}}
	out, err := pr.markdown(p.Parse("see [foo.Bar] now"), 3)
	if err != nil {
		t.Fatal(err)
	}
	if want := "see [foo.Bar](https://pkg.go.dev/evil%29%3Cdetails%20open%20ontoggle=alert%281%29%3E#Bar) now"; out != want {
		t.Errorf("got\n%s\nwant\n%s", out, want)
	}
}
