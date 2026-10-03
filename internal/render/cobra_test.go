package render

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/dialect"
	"github.com/open-platform-model/docs-kit/internal/extract/cobra"
)

// cobraFixture is the doc model of cobradump's fixture tree, through the
// registered renderer's input: the encoded data file.
func cobraFixture(t *testing.T, citations string) []byte {
	t.Helper()
	dump, err := os.ReadFile("../../cobradump/testdata/dump.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	m, err := cobra.FromDump(dump, cobra.Options{Section: "reference/cli/", Title: "CLI Reference", Description: "Every demo command and flag.", Weight: 2, Citations: citations})
	if err != nil {
		t.Fatal(err)
	}
	data, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

var docsTarget = Target{Kind: KindDocs, Root: "/docs/", Segment: "1.0", Version: "1.0.0", Repo: "example/demo", Commit: commit}

func TestCobraGolden(t *testing.T) {
	r, err := For(cobra.SchemaID)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := r.Render(cobraFixture(t, "strip"), docsTarget)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join("testdata", "golden", "cobra")
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
		if p.Completable {
			t.Errorf("%s is completable", p.Path)
		}
		want, err := os.ReadFile(filepath.Join(dir, p.Path))
		if err != nil {
			t.Fatalf("%s: %v (run go test -run TestCobraGolden -update)", p.Path, err)
		}
		if string(want) != p.Body {
			t.Errorf("%s differs from its golden page", p.Path)
		}
	}
	sort.Strings(names)
	if got := strings.Join(names, ","); got != "reference/cli/_index.md,reference/cli/demo-completion.md,reference/cli/demo-version.md,reference/cli/demo-widget.md" {
		t.Fatalf("pages = %s", got)
	}
	vs, err := dialect.Lint(dir, dialect.Options{Mode: dialect.Bundle, Bundle: dialect.BundleInfo{Kind: KindDocs, Root: "/docs/", Owns: []string{"reference/cli/"}, Pages: names}})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		t.Errorf("golden page breaks the dialect: %s", v)
	}
}

func TestCobraPageFacts(t *testing.T) {
	r, _ := For(cobra.SchemaID)
	pages, err := r.Render(cobraFixture(t, "strip"), docsTarget)
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]string{}
	for _, p := range pages {
		body[p.Path] = p.Body
	}
	w := body["reference/cli/demo-widget.md"]
	for _, want := range []string{
		"---\ntitle: \"demo widget\"\ndescription: \"Work with widgets.\"\ntype: reference\n---\n\n",
		"Every command on this page also takes the [global flags](/docs/reference/cli/#global-flags).\n",
		"| [demo widget create](/docs/reference/cli/demo-widget/#demo-widget-create) | Create a widget. |",
		"Aliases: `w`.",
		"| `--cache` |  | string | `~/.cache/demo` | Cache directory. |",
		"| `--context` |  | string |  | Kubernetes context \\| cluster. |",
		"| `--namespace` | `-n` | string | `default` | Target namespace. |",
		"| `--rootfs` |  | string | `/home/uu/rootfs` | Not under the home directory. |",
		"**Examples**\n\n```sh\n# Create from the built-in template\ndemo widget create web\n\ndemo widget create web --template ./tpl\n\ndemo widget create api\ndemo w create {{</* x */>}}\n```",
		"- render the template\n- write the widget\n",
	} {
		if !strings.Contains(w, want) {
			t.Errorf("demo-widget.md lacks %q", want)
		}
	}
	for _, not := range []string{"--verbose", "--old", "--secret", "## demo widget secret", "## demo widget old", "/home/u/"} {
		if strings.Contains(w, not) {
			t.Errorf("demo-widget.md holds %q", not)
		}
	}
	idx := body["reference/cli/_index.md"]
	if !strings.HasPrefix(idx, "---\ntitle: \"CLI Reference\"\ndescription: \"Every demo command and flag.\"\nweight: 2\n---\n\n") ||
		!strings.Contains(idx, "| `--config` |  | string | `~/.demo/config.cue` | Path to config file (env: `DEMO_CONFIG`). |") {
		t.Errorf("index =\n%s", idx)
	}
}

func TestCobraTabTarget(t *testing.T) {
	r, _ := For(cobra.SchemaID)
	pages, err := r.Render(cobraFixture(t, "strip"), Target{Kind: KindTab, Root: "/catalogs/demo/", Segment: "1.2"})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pages {
		if p.Path == "reference/cli/demo-widget.md" && !strings.Contains(p.Body, "(/catalogs/demo/1.2/reference/cli/demo-widget/#demo-widget-list)") {
			t.Fatalf("tab links:\n%s", p.Body)
		}
	}
}
