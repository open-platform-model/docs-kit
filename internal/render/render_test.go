package render

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/dialect"
	"github.com/open-platform-model/docs-kit/internal/extract/cuecatalog"
)

var update = flag.Bool("update", false, "rewrite the golden pages")

const commit = "0123456789abcdef0123456789abcdef01234567"

func fixtureModel(t *testing.T) *cuecatalog.Model {
	t.Helper()
	m, err := cuecatalog.Extract(cuecatalog.Options{Root: "../extract/cuecatalog/testdata/catalog", Module: "./demo"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	m, err = cuecatalog.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func renderAll(t *testing.T, m *cuecatalog.Model, tg Target) map[string]string {
	t.Helper()
	pages, err := Catalog(m, tg)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, p := range pages {
		out[p.Path] = p.Body
	}
	landing, err := Landing(m, tg, "", "")
	if err != nil {
		t.Fatal(err)
	}
	out["_index.md"] = landing
	return out
}

func TestGolden(t *testing.T) {
	m := fixtureModel(t)
	for _, c := range []struct {
		name string
		tg   Target
	}{
		{"release", Target{Root: "/catalogs/demo/", Segment: "1.2", Repo: "example/demo", Commit: commit}},
		{"edge", Target{Root: "/catalogs/demo/", Segment: "edge", Edge: true, Repo: "example/demo", Commit: commit}},
	} {
		t.Run(c.name, func(t *testing.T) {
			pages := renderAll(t, m, c.tg)
			dir := filepath.Join("testdata", "golden", c.name)
			if *update {
				_ = os.RemoveAll(dir)
				for p, body := range pages {
					full := filepath.Join(dir, p)
					_ = os.MkdirAll(filepath.Dir(full), 0o755)
					if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			}
			var names []string
			for p, body := range pages {
				names = append(names, p)
				want, err := os.ReadFile(filepath.Join(dir, p))
				if err != nil {
					t.Fatalf("%s: %v (run go test -run TestGolden -update)", p, err)
				}
				if string(want) != body {
					t.Errorf("%s differs from its golden page", p)
				}
			}
			sort.Strings(names)
			vs, err := dialect.Lint(dir, dialect.Options{Mode: dialect.Bundle, Bundle: dialect.BundleInfo{Root: c.tg.Root, Segment: c.tg.Segment, Pages: names}})
			if err != nil {
				t.Fatal(err)
			}
			for _, v := range vs {
				t.Errorf("golden page breaks the dialect: %s", v)
			}
		})
	}
}

func TestPageFacts(t *testing.T) {
	m := fixtureModel(t)
	rel := renderAll(t, m, Target{Root: "/catalogs/demo/", Segment: "1.2", Repo: "example/demo", Commit: commit})
	edge := renderAll(t, m, Target{Root: "/catalogs/demo/", Segment: "edge", Edge: true, Repo: "example/demo", Commit: commit})
	for _, c := range []struct{ page, want string }{
		{"traits/backup.md", "[Container](/catalogs/demo/1.2/resources/container/)"},
		{"traits/backup.md", "([contract levels](/catalogs/demo/1.2/))"},
		{"traits/backup-v1alpha1.md", "Without one, rendering a component that attaches it fails; with two, the kernel refuses every render on that platform."},
		{"traits/scaling.md", "a component that attaches it still renders, and the render warns that the trait is not handled and ignores its values."},
		{"traits/scaling.md", `{\{\< x \>}}`},
		{"traits/scaling.md", "[docs/scaling-notes.md](https://github.com/example/demo/blob/" + commit + "/docs/scaling-notes.md)"},
		{"resources/queue.md", "> [!WARNING]\n> **Not implemented**"},
		{"resources/container.md", "`k8s.#EnvVar`: from `example.com/catalogs/demo/schemas/kubernetes/core/v1`, the vendored Kubernetes API types"},
		{"blueprints/web.md", "`res.#ContainerSchema`: the spec of [Container](/catalogs/demo/1.2/resources/container/)"},
		{"_index.md", "- [Traits](/catalogs/demo/1.2/traits/): 3"},
		{"resources/_index.md", "marked **Not implemented**: [Queue](/catalogs/demo/1.2/resources/queue/)."},
	} {
		if !strings.Contains(rel[c.page], c.want) {
			t.Errorf("%s does not hold %q", c.page, c.want)
		}
	}
	if !strings.Contains(edge["traits/backup.md"], "[Container](/catalogs/demo/edge/resources/container/)") ||
		!strings.Contains(edge["_index.md"], "`example.com/catalogs/demo@v1` at `main` (commit `0123456789ab`), unreleased.") {
		t.Errorf("edge pages:\n%s\n%s", edge["traits/backup.md"], edge["_index.md"])
	}
	for p, body := range rel {
		if strings.Contains(body, "<!--") {
			t.Errorf("%s carries a marker comment", p)
		}
	}
}

func TestAuthoredLanding(t *testing.T) {
	m := fixtureModel(t)
	tg := Target{Root: "/catalogs/demo/", Segment: "1.2", Repo: "example/demo", Commit: commit}
	authored := "---\ntitle: \"The demo catalog contract\"\ndescription: \"What a demo catalog promises.\"\n---\n\nThe contract.\n"
	got, err := Landing(m, tg, authored, "docs/catalogs/demo/_index.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, authored+"\n## Catalog members\n\n") {
		t.Fatalf("landing:\n%s", got)
	}
	_, err = Landing(m, tg, authored+"\n## Catalog members\n\nx\n", "docs/catalogs/demo/_index.md")
	if err == nil || !strings.Contains(err.Error(), "docs/catalogs/demo/_index.md") {
		t.Fatalf("err = %v", err)
	}
}
