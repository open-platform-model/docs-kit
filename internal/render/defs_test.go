package render

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/dialect"
	"github.com/open-platform-model/docs-kit/internal/extract/cuedefs"
)

var defsTarget = Target{Kind: KindDocs, Root: "/docs/", Segment: "1.2", Version: "1.2.3", Repo: "example/defs", Commit: commit}

// renderDefs renders the extractor's golden data file, as written.
func renderDefs(t *testing.T) []Page {
	t.Helper()
	data, err := os.ReadFile("../extract/cuedefs/testdata/cue-definitions.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := For(cuedefs.SchemaID)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := r.Render(data, defsTarget)
	if err != nil {
		t.Fatal(err)
	}
	return pages
}

func TestDefinitionsGolden(t *testing.T) {
	pages := renderDefs(t)
	dir := filepath.Join("testdata", "golden", "definitions")
	if *update {
		_ = os.RemoveAll(dir)
		for _, p := range pages {
			full := filepath.Join(dir, p.Path)
			_ = os.MkdirAll(filepath.Dir(full), 0o755)
			if err := os.WriteFile(full, []byte(p.Body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	names := make([]string, 0, len(pages))
	for _, p := range pages {
		names = append(names, p.Path)
		want, err := os.ReadFile(filepath.Join(dir, p.Path))
		if err != nil {
			t.Fatalf("%s: %v (run go test -run TestDefinitionsGolden -update)", p.Path, err)
		}
		if string(want) != p.Body {
			t.Errorf("%s differs from its golden page", p.Path)
		}
	}
	sort.Strings(names)
	if want := []string{"reference/definitions/_index.md", "reference/definitions/modules.md", "reference/definitions/types.md"}; strings.Join(names, " ") != strings.Join(want, " ") {
		t.Errorf("pages %v, want %v", names, want)
	}
	vs, err := dialect.Lint(dir, dialect.Options{Mode: dialect.Bundle, Bundle: dialect.BundleInfo{
		Kind: KindDocs, Root: "/docs/", Segment: "1.2", Owns: []string{"reference/definitions/"}, Pages: names,
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		t.Errorf("golden page breaks the dialect: %s", v)
	}
}

func TestDefinitionsPages(t *testing.T) {
	pages := renderDefs(t)
	byPath := map[string]Page{}
	for _, p := range pages {
		byPath[p.Path] = p
	}
	index := byPath["reference/definitions/_index.md"]
	if !index.Completable || index.Heading != "## Pages" || !strings.HasPrefix(index.Tail, "## Pages\n\n") {
		t.Errorf("index completable %v heading %q tail %q", index.Completable, index.Heading, index.Tail)
	}
	if !strings.Contains(index.Body, "weight: 2\n") || strings.Contains(index.Body, "type:") {
		t.Errorf("index front matter:\n%s", index.Body)
	}
	mod := byPath["reference/definitions/modules.md"].Body
	for _, want := range []string{
		"weight: 1\n",
		"type: reference\n",
		"## #Module\n",
		// A link to a definition on another page.
		"[`#NameType`](/docs/reference/definitions/types/#nametype)",
		"- Embeds: [`#Widget`](/docs/reference/definitions/modules/#widget)",
		"```text\nan indented line\n```",
		"**Example**\n\n```cue\n\"web\"\n\"db\"\n```",
	} {
		if !strings.Contains(mod, want) {
			t.Errorf("modules page lacks %q", want)
		}
	}
	for _, p := range pages {
		if strings.Contains(p.Body, "<!--") {
			t.Errorf("%s carries a marker comment", p.Path)
		}
	}
}

func TestDefinitionsNeedDocsPlacement(t *testing.T) {
	data, err := os.ReadFile("../extract/cuedefs/testdata/cue-definitions.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	_, err = defsRenderer{}.Render(data, Target{Root: "/catalogs/demo/", Segment: "1.2"})
	if err == nil || !strings.Contains(err.Error(), `placement kind "docs"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestDefinitionsProse(t *testing.T) {
	r := &defsRender{m: &cuedefs.Model{Section: "reference/definitions/"}, t: defsTarget, byName: map[string]*cuedefs.Definition{
		"#Module": {Name: "#Module", Anchor: "module", Page: "modules"},
	}}
	for in, want := range map[string]string{
		// A link's text is escaped; its URL and title stay as written.
		"[<img src=x onerror=alert(1)>](https://e.example)": "[\\<img src=x onerror=alert(1)>](https://e.example)",
		`[t #Module](/docs/x/#module "a title")`:            `[t #Module](/docs/x/#module "a title")`,
		// A Markdown link is left as written, its fragment included.
		"See [the guide](/docs/guide/#module) and #Module.": "See [the guide](/docs/guide/#module) and [`#Module`](/docs/reference/definitions/modules/#module).",
		"Code ``a ` #Module`` stays; #Other is code.":       "Code ``a ` #Module`` stays; `#Other` is code.",
		"An unmatched ` run and #Module.":                   "An unmatched ` run and [`#Module`](/docs/reference/definitions/modules/#module).",
		"Self #Self.x, $name and <b> {{x}}":                 "Self `#Self.x`, `$name` and \\<b> {\\{x}}",
	} {
		if got := r.markdown(in, "#Self"); got != want {
			t.Errorf("markdown(%q)\n got %q\nwant %q", in, got, want)
		}
	}
}

func TestDefinitionsFences(t *testing.T) {
	if f := fence("a ``` b"); f != "````" {
		t.Errorf("fence %q", f)
	}
	if f := fence("plain"); f != "```" {
		t.Errorf("fence %q", f)
	}
	m := &cuedefs.Model{Section: "reference/definitions/", ModulePath: "example.com/m@v1"}
	r := &defsRender{m: m, t: defsTarget, byName: map[string]*cuedefs.Definition{}}
	d := &cuedefs.Definition{
		Name: "#T", Anchor: "t", Page: "p", File: "src/t.cue", Summary: "T.", Shape: "value",
		CUE:   "#T: \"```\"",
		Notes: []string{"  pre ``` line", "  second"},
		Rules: []cuedefs.Rule{{Rule: "Match:", Code: "^`{3}$", By: "cue"}},
	}
	got := r.entry(d)
	for _, want := range []string{
		"````cue\n#T: \"```\"\n````\n",
		"````text\npre ``` line\nsecond\n````\n",
		"  ```text\n  ^`{3}$\n  ```\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("entry lacks %q:\n%s", want, got)
		}
	}
}

// The site renders links as written, so a doc comment link with any scheme
// but http or https is refused, naming the definition.
func TestDefinitionsRefuseUnsafeLinks(t *testing.T) {
	probe := "[<img src=x onerror=alert(1)>](https://e.example) and [a](javascript:alert(1))"
	for _, d := range []cuedefs.Definition{
		{Name: "#T", File: "src/t.cue", Summary: probe},
		{Name: "#T", File: "src/t.cue", Notes: []string{"- " + probe}},
		{Name: "#T", File: "src/t.cue", Rules: []cuedefs.Rule{{Rule: "[x](data:text/html,hi)", By: "cue"}}},
	} {
		m := &cuedefs.Model{Section: "reference/definitions/", Definitions: []cuedefs.Definition{d}}
		_, err := Definitions(m, defsTarget)
		if err == nil || !strings.Contains(err.Error(), "#T (src/t.cue)") || !strings.Contains(err.Error(), "http, https or relative") {
			t.Errorf("%+v: err = %v", d, err)
		}
	}
	ok := &cuedefs.Model{Section: "reference/definitions/", Definitions: []cuedefs.Definition{
		{Name: "#T", File: "src/t.cue", Page: "p", Anchor: "t", Summary: "[a](https://e.example), [b](HTTP://e.example), [c](/docs/x/#y) and [d](#t)."},
	}, Pages: []cuedefs.Page{{File: "p", Title: "P", Description: "d", Weight: 1, Definitions: []string{"#T"}}}}
	if _, err := Definitions(ok, defsTarget); err != nil {
		t.Errorf("safe links refused: %v", err)
	}
}
