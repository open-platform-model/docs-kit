package helptext

import (
	"reflect"
	"strings"
	"testing"
)

func eq(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %#v\nwant %#v", got, want)
	}
}

func TestParseTabIndentedSourceSplitsProsePreformattedAndExamples(t *testing.T) {
	long := "Publish a thing to its registry, at the coordinates it\ndeclares.\n\n" +
		"\tThe pipeline reads identity/identity.cue and\n\tpushes.\n\n" +
		"\tArguments:\n\t  path    Path to the directory\n\n" +
		"\tExamples:\n\t  # First\n\t  opm thing publish\n\n\t  # Second\n\t  opm thing publish ./src"
	p := Parse(long)
	eq(t, p.Desc, []Block{
		{Lines: []string{"Publish a thing to its registry, at the coordinates it declares."}},
		{Lines: []string{"The pipeline reads identity/identity.cue and pushes."}},
		{Lines: []string{"Arguments:"}},
		{Pre: true, Lines: []string{"path    Path to the directory"}},
	})
	eq(t, p.Examples, []string{"# First", "opm thing publish", "", "# Second", "opm thing publish ./src"})
}

func TestParseUnindentedSourceKeepsNestedIndentation(t *testing.T) {
	p := Parse("Create a package.\n\nThe package holds:\n\n  a.cue   first\n          continued\n  b.cue   second\n\nExit codes: 0 written.")
	if len(p.Desc) != 4 {
		t.Fatalf("desc = %#v", p.Desc)
	}
	eq(t, p.Desc[1].Lines, []string{"The package holds:"})
	eq(t, p.Desc[2], Block{Pre: true, Lines: []string{"a.cue   first", "        continued", "b.cue   second"}})
	eq(t, p.Desc[3].Lines, []string{"Exit codes: 0 written."})
	if len(p.Examples) != 0 {
		t.Fatalf("examples = %v", p.Examples)
	}
}

func TestParseProseAfterExamplesReturnsToTheDescription(t *testing.T) {
	p := Parse("Run it.\n\nExamples:\n  opm run\n\nSee also the guide.")
	eq(t, p.Examples, []string{"opm run"})
	if len(p.Desc) != 2 {
		t.Fatalf("desc = %#v", p.Desc)
	}
	eq(t, p.Desc[1].Lines, []string{"See also the guide."})
}

func TestParseEmpty(t *testing.T) {
	eq(t, Parse(""), Long{})
}

func TestProse(t *testing.T) {
	for _, tt := range []struct{ name, in, want string }{
		{"flags and env vars", "Use --registry, then OPM_REGISTRY.", "Use `--registry`, then `OPM_REGISTRY`."},
		{"paths and placeholders", "Pass --platform <dir> or ~/.opm/platform/.", "Pass `--platform <dir>` or `~/.opm/platform/`."},
		{"quoted command", "run 'opm module vet' first", "run `opm module vet` first"},
		{"apostrophes stay prose", "the cluster's Platform", "the cluster's Platform"},
		{"word pairs stay prose", "every beta/GA member", "every beta/GA member"},
		{"files", "next to values.cue and identity/identity.cue.", "next to `values.cue` and `identity/identity.cue`."},
		{"definitions", "a #ModuleInstance around it", "a `#ModuleInstance` around it"},
		{"quoted code word drops its quotes", `defaults to "<module>-debug".`, "defaults to `<module>-debug`."},
		{"lone operators are escaped", "[dir] > --platform", "`[dir]` &gt; `--platform`"},
		{"markdown characters are escaped", "#### Linux: 50% & more", `\#\#\#\# Linux: 50% &amp; more`},
		{"list marker at the start", "- not a list", `\- not a list`},
		{"ordinal at the start", "1. not a list", `1\. not a list`},
		{"whitespace collapses", "  a \t b  ", "a b"},
		{"neighboring code words share a span", "Pass --platform <dir> to render.", "Pass `--platform <dir>` to render."},
		{"home path joins its flag", "with --platform ~/.opm/platform to keep", "with `--platform ~/.opm/platform` to keep"},
		{"punctuation stays outside", "(--registry, then OPM_REGISTRY)", "(`--registry`, then `OPM_REGISTRY`)"},
		{"a value after a flag stays prose", "--version v1 takes", "`--version` v1 takes"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := Prose(tt.in); got != tt.want {
				t.Fatalf("Prose(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFence(t *testing.T) {
	eq(t, Fence("text", []string{"```go"}), "````text\n```go\n````\n")
	eq(t, Fence("sh", []string{"opm"}), "```sh\nopm\n```\n")
}

func TestFenceLang(t *testing.T) {
	eq(t, FenceLang([]string{"# comment", "opm module build", ""}, "opm"), "sh")
	eq(t, FenceLang([]string{"# comment", "demo build", ""}, "demo"), "sh")
	eq(t, FenceLang([]string{"demo build"}, "opm"), "text")
	eq(t, FenceLang([]string{"source <(opm completion bash)"}, "opm"), "text")
	eq(t, FenceLang([]string{"# only a comment"}, "opm"), "text")
}

func TestEscapeShortcodes(t *testing.T) {
	eq(t, EscapeShortcodes("{{< opm/x >}} {{% y %}}"), "{{</* opm/x */>}} {{%/* y */%}}")
}

func TestListItems(t *testing.T) {
	marker, items, ok := ListItems([]string{"- one", "- two"})
	eq(t, []any{marker, items, ok}, []any{"-", []string{"one", "two"}, true})
	marker, items, ok = ListItems([]string{"1. first", "2. second"})
	eq(t, []any{marker, items, ok}, []any{"1.", []string{"first", "second"}, true})
	for _, lines := range [][]string{{"- one", "  continued"}, {"- one", "1. two"}, {"0   the module was written"}} {
		if _, _, ok := ListItems(lines); ok {
			t.Fatalf("%q read as a list", lines)
		}
	}
}

func TestWriteBlocksListBecomesAMarkdownList(t *testing.T) {
	var b strings.Builder
	WriteBlocks(&b, Parse("Show version.\n\nDisplays:\n  - the CLI version\n  - the CUE SDK version").Desc, "opm", func(s string) string { return s })
	eq(t, b.String(), "Show version.\n\nDisplays:\n\n- the CLI version\n- the CUE SDK version\n")
}
