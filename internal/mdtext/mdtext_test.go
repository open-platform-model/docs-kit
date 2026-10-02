package mdtext

import (
	"strings"
	"testing"
)

func TestText(t *testing.T) {
	cases := map[string]string{
		"pods are <sts>-<n>.<svc>":      `pods are \<sts\>-\<n\>.\<svc\>`,
		"keep `a_b|<c>` as code":        "keep `a_b|<c>` as code",
		"no {{< opm/x >}} shortcode":    `no {\{\< opm/x \>}} shortcode`,
		"a [link](x) and *stars* and |": `a \[link\](x) and \*stars\* and \|`,
		`a \ backslash`:                 `a \\ backslash`,
	}
	for in, want := range cases {
		if got := Text(in); got != want {
			t.Errorf("Text(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCodeAndCell(t *testing.T) {
	if got := Code("a`b"); got != "``a`b``" {
		t.Errorf("Code = %q", got)
	}
	if got := Code("`a"); got != "`` `a ``" {
		t.Errorf("Code = %q", got)
	}
	if got := Cell(Code(`"a" | "b"`)); got != "`\"a\" \\| \"b\"`" {
		t.Errorf("Cell = %q", got)
	}
	if got := YAMLString(`say "hi" \o/`); got != `"say \"hi\" \\o/"` {
		t.Errorf("YAMLString = %q", got)
	}
}

func TestDocNotes(t *testing.T) {
	s := Text("see docs/keep-more.md and `docs/code.md`, and docs/missing.md")
	if got := DocNotes(s); strings.Join(got, ",") != "docs/keep-more.md,docs/missing.md" {
		t.Errorf("DocNotes = %v", got)
	}
	got := LinkDocNotes(s, "https://x/blob/c/", func(n string) bool { return n == "docs/keep-more.md" })
	want := "see [docs/keep-more.md](https://x/blob/c/docs/keep-more.md) and `docs/code.md`, and docs/missing.md"
	if got != want {
		t.Errorf("LinkDocNotes = %q", got)
	}
}

func TestCheckShortcodes(t *testing.T) {
	if err := CheckShortcodes("p.md", Text("no {{< x >}} here")); err != nil {
		t.Fatal(err)
	}
	if err := CheckShortcodes("p.md", "a\n{{< x >}}\n"); err == nil || !strings.Contains(err.Error(), "p.md:2") {
		t.Fatalf("err = %v", err)
	}
}
