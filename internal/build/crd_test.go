package build

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/gittest"
)

const crdConfig = `bundles: "widgets": {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/widgets.md"]}
	version: {from: "tag", prefix: "v"}
	sources: [
		{kind: "markdown", dir: "docs/site"},
		{
			kind:    "crd"
			dir:     "./config/crd/bases"
			samples: "./config/samples"
			hideSamplesMatching: ["testing.opmodel.dev"]
			stripLabels: {"app.kubernetes.io/managed-by": "kustomize"}
			page:        "reference/widgets.md"
			title:       "Widget resources"
			description: "One entry per widget kind."
			weight:      7
			order: ["Widget"]
			reconciledBy: {Widget: "widget"}
			citations: "link"
		},
	]
}
`

const authoredWidgets = "---\ntitle: \"Widgets\"\ndescription: \"What widgets are.\"\ntype: reference\n---\n\nThis page lists the widget kinds.\n"

// crdBuild builds the crd extractor's fixture tree, with files added, as
// a docs bundle through docs-kit.cue.
func crdBuild(t *testing.T, files map[string]string) (m *bundle.Manifest, dir string) {
	t.Helper()
	return crdBuildThen(t, files, nil)
}

// crdBuildThen is crdBuild with more commits made by then before the
// build.
func crdBuildThen(t *testing.T, files map[string]string, then func(*gittest.Repo)) (m *bundle.Manifest, dir string) {
	t.Helper()
	t.Setenv("GITHUB_REPOSITORY", "")
	tree := filepath.Join("..", "extract", "crd", "testdata", "tree")
	err := filepath.WalkDir(tree, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(tree, p)
		files[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	files["docs-kit.cue"] = crdConfig
	files["docs/site/guide.md"] = "---\ntitle: \"Guide\"\ndescription: \"x\"\ntype: explanation\n---\n\nA guide.\n"
	r := gittest.New(t, "https://github.com/example/widgets.git")
	r.Write(files)
	r.Commit("docs")
	if then != nil {
		then(r)
	}
	cfg, err := config.Load(filepath.Join(r.Dir, "docs-kit.cue"))
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out")
	if _, err := buildProject(context.Background(), Options{Source: r.Dir, Out: out, Tool: "0.1.0"}, cfg, "widgets", false); err != nil {
		var le *LintError
		if errors.As(err, &le) {
			t.Fatalf("%v: %s", err, strings.Join(le.Violations, "; "))
		}
		t.Fatal(err)
	}
	dir = filepath.Join(out, "widgets")
	if m, err = bundle.Read(dir); err != nil {
		t.Fatal(err)
	}
	return m, dir
}

func TestCRDBundleCompleted(t *testing.T) {
	m, dir := crdBuild(t, map[string]string{"docs/site/reference/widgets.md": authoredWidgets})
	body, _ := os.ReadFile(filepath.Join(dir, "content", "reference", "widgets.md"))
	if !strings.HasPrefix(string(body), authoredWidgets+"\n## Widget\n") {
		t.Fatalf("completed page:\n%s", body)
	}
	if p := pageEntry(m, "reference/widgets.md"); p.Generated || p.Source != "docs/site/reference/widgets.md" || p.Edit != "docs/site/reference/widgets.md" {
		t.Fatalf("entry %+v", p)
	}
	if len(m.Data) != 1 || m.Data[0].Path != "crd.json" || m.Data[0].Schema != "docs.opmodel.dev/data/crd/v1" {
		t.Fatalf("data %+v", m.Data)
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "crd.json")); err != nil {
		t.Fatal(err)
	}
}

func TestCRDBundleAlone(t *testing.T) {
	m, dir := crdBuild(t, map[string]string{})
	body, _ := os.ReadFile(filepath.Join(dir, "content", "reference", "widgets.md"))
	if !strings.HasPrefix(string(body), "---\ntitle: \"Widget resources\"\ndescription: \"One entry per widget kind.\"\ntype: reference\nweight: 7\n---\n\n## Widget\n") {
		t.Fatalf("standalone page:\n%s", body)
	}
	if !strings.Contains(string(body), "[0021:D4](/enhancements/0021/decisions/)") {
		t.Errorf("citations: \"link\" did not link the decision citation")
	}
	p := pageEntry(m, "reference/widgets.md")
	if !p.Generated || p.Source != "config/crd/bases/example.dev_widgets.yaml" || p.Lastmod == "" || p.Edit != "" {
		t.Fatalf("entry %+v: a generated page has a source and lastmod, and no edit", p)
	}
}

// The standalone page's lastmod is the newest date of every file read: a
// sample changed after the CRDs moves it.
func TestCRDBundleLastmodNewest(t *testing.T) {
	m, _ := crdBuildThen(t, map[string]string{}, func(r *gittest.Repo) {
		r.Write(map[string]string{"config/samples/example.dev_v1_gadget.yaml": "apiVersion: example.dev/v1\nkind: Gadget\nmetadata:\n  name: g\n"})
		r.CommitAt("2030-01-02T03:04:05Z", "sample")
	})
	if p := pageEntry(m, "reference/widgets.md"); p.Lastmod != "2030-01-02T03:04:05Z" {
		t.Fatalf("lastmod %q, want the sample's commit date", p.Lastmod)
	}
}
