package build

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/gittest"
)

const goAPIConfig = `bundles: "widgets": {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/go-api/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{
		kind:        "go-api"
		module:      "./mod"
		root:        "./lib"
		packages: ["./lib/..."]
		section:     "reference/go-api/"
		title:       "Go API"
		description: "Every exported package."
		citations:   "link"
	}, {kind: "markdown", dir: "docs/site"}]
}
`

// goAPIBuild builds the go-api extractor's fixture module as a docs
// bundle, with files added and then run before the build.
func goAPIBuild(t *testing.T, files map[string]string, then func(*gittest.Repo)) (*bundle.Manifest, string, error) {
	t.Helper()
	t.Setenv("GITHUB_REPOSITORY", "")
	if files["docs-kit.cue"] == "" {
		files["docs-kit.cue"] = goAPIConfig
	}
	files["docs/site/guide.md"] = "---\ntitle: \"Guide\"\ndescription: \"x\"\ntype: explanation\n---\n\nA guide.\n"
	r := gittest.New(t, "https://github.com/example/widgets.git")
	r.CopyTree(filepath.Join("..", "extract", "goapi", "testdata", "mod"), "mod")
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
		return nil, "", err
	}
	dir := filepath.Join(out, "widgets")
	m, err := bundle.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	return m, dir, nil
}

func TestGoAPIBundle(t *testing.T) {
	m, dir, err := goAPIBuild(t, map[string]string{}, func(r *gittest.Repo) {
		r.Write(map[string]string{"mod/lib/widget/widget.go": "package widget\n\n// Widget is newer.\ntype Widget struct{}\n"})
		r.CommitAt("2030-01-02T03:04:05Z", "widget")
	})
	if err != nil {
		t.Fatal(err)
	}
	p := pageEntry(m, "reference/go-api/widget.md")
	if !p.Generated || p.Source != "mod/lib/widget/doc.go" || p.Edit != "" {
		t.Fatalf("entry %+v: a generated page has its package's first file as source, and no edit", p)
	}
	if p.Lastmod != "2030-01-02T03:04:05Z" {
		t.Errorf("lastmod %q, want the newest file's commit date", p.Lastmod)
	}
	if p := pageEntry(m, "reference/go-api/_index.md"); !p.Generated || p.Edit != "" {
		t.Errorf("index %+v", p)
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "go-api.json")); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Join(dir, "content", "reference", "go-api", "widget.md"))
	if !strings.Contains(string(body), "makes widgets ([0010:D28](/enhancements/0010/decisions/)).") {
		t.Errorf("citations: link is not applied:\n%s", body)
	}
}

func TestGoAPIBundleCompleted(t *testing.T) {
	authored := "---\ntitle: \"Go API\"\ndescription: \"The library's Go API.\"\n---\n\nStart with the kernel.\n"
	m, dir, err := goAPIBuild(t, map[string]string{"docs/site/reference/go-api/_index.md": authored}, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Join(dir, "content", "reference", "go-api", "_index.md"))
	if !strings.HasPrefix(string(body), authored+"\n## Packages\n\n- [lib/gadget](/docs/reference/go-api/gadget/)") {
		t.Fatalf("completed index:\n%s", body)
	}
	if p := pageEntry(m, "reference/go-api/_index.md"); p.Generated || p.Source != "docs/site/reference/go-api/_index.md" {
		t.Errorf("index %+v", p)
	}
}

func TestGoAPIBundleDeterministic(t *testing.T) {
	outs := make([][]byte, 0, 2)
	for range 2 {
		_, dir, err := goAPIBuild(t, map[string]string{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		var b bytes.Buffer
		for _, f := range []string{"data/go-api.json", "content/reference/go-api/widget.md", "content/reference/go-api/_index.md"} {
			data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f)))
			if err != nil {
				t.Fatal(err)
			}
			b.Write(data)
		}
		outs = append(outs, b.Bytes())
	}
	if !bytes.Equal(outs[0], outs[1]) {
		t.Fatal("two builds differ")
	}
}

func TestGoAPIBundleOutsideOwns(t *testing.T) {
	_, _, err := goAPIBuild(t, map[string]string{"docs-kit.cue": strings.Replace(goAPIConfig, `owns: ["reference/go-api/"]`, `owns: ["reference/cli/"]`, 1)}, nil)
	if err == nil || !strings.Contains(err.Error(), "content/reference/go-api/_index.md is generated, but widgets owns only reference/cli/") {
		t.Fatalf("err = %v", err)
	}
}

func TestGoAPIBundleNoPackage(t *testing.T) {
	_, _, err := goAPIBuild(t, map[string]string{"docs-kit.cue": strings.Replace(goAPIConfig, `["./lib/..."]`, `["./lib/nope/..."]`, 1)}, nil)
	if err == nil || !strings.Contains(err.Error(), "packages ./lib/nope/... matches no package under ./mod") {
		t.Fatalf("err = %v", err)
	}
}
