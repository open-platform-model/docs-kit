package build

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/gittest"
	"github.com/open-platform-model/docs-kit/internal/render"
)

// editRepo is a repository whose tag v1.0.0 holds docs/site/start/install.md,
// docs/site/guide.md and docs/site/reference/operator-resources.md (which
// completes the fake extractor's completable page), and whose main has
// since renamed install.md to setup.md and added new.md.
func editRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	t.Setenv("GITHUB_REPOSITORY", "")
	t.Setenv("GITHUB_REF_TYPE", "")
	extractors = append(extractors, fakeExtractor{pages: []string{"reference/operator-resources.md", "reference/generated.md"}})
	rendererFor = func(s string) (render.Renderer, error) {
		if s == fakeSchema {
			return fakeRenderer{}, nil
		}
		return render.For(s)
	}
	t.Cleanup(func() { extractors = extractors[:len(extractors)-1]; rendererFor = render.For })
	r := gittest.New(t, "https://github.com/example/cli.git")
	page := func(title string) string { return strings.Replace(authoredOps, "Operator resources", title, 1) }
	r.Write(map[string]string{
		"README.md":                                 "x\n",
		"docs/site/start/install.md":                page("Install"),
		"docs/site/guide.md":                        page("Guide"),
		"docs/site/reference/operator-resources.md": authoredOps,
		"docs/catalogs/demo/_index.md":              "---\ntitle: \"Demo\"\ndescription: \"x\"\n---\n\nThe demo catalog.\n",
	})
	r.Commit("docs")
	r.Git("tag", "v1.0.0")
	r.Git("mv", "docs/site/start/install.md", "docs/site/start/setup.md")
	r.Write(map[string]string{"docs/site/new.md": page("New")})
	r.Commit("rename install, add new")
	return r
}

func editConfig(kind string) *config.Config {
	b := config.Bundle{
		Placement: config.Placement{Kind: "docs", Root: "/docs/", Owns: []string{"reference/"}},
		Version:   config.VersionRule{From: "tag", Prefix: "v"},
		Sources:   []config.Source{{Kind: "fake"}, {Kind: "markdown", Dir: "docs/site"}},
	}
	if kind == "tab" {
		b.Placement = config.Placement{Kind: "tab", Root: "/catalogs/demo/"}
		b.Sources = []config.Source{{Kind: "markdown", Dir: "docs/catalogs/demo"}}
	}
	return &config.Config{Path: "docs-kit.cue", Bundles: map[string]config.Bundle{"cli": b}}
}

// tagTree checks the tag out in a worktree beside the repository, as
// release mode's src/ and revise's temporary worktree are.
func tagTree(t *testing.T, r *gittest.Repo) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "src")
	r.Git("worktree", "add", "-q", "--detach", dir, "v1.0.0")
	return dir
}

func editBuild(t *testing.T, o Options, kind string) map[string]bundle.Page {
	t.Helper()
	o.Out, o.Tool = filepath.Join(t.TempDir(), "out"), "0.1.0"
	if _, err := buildProject(context.Background(), o, editConfig(kind), "cli", false); err != nil {
		t.Fatal(err)
	}
	m, err := bundle.Read(filepath.Join(o.Out, "cli"))
	if err != nil {
		t.Fatal(err)
	}
	pages := map[string]bundle.Page{}
	for _, p := range m.Pages {
		pages[p.Path] = p
	}
	return pages
}

func wantEdits(t *testing.T, pages map[string]bundle.Page, want map[string]string) {
	t.Helper()
	for path, p := range pages {
		if p.Edit != want[path] {
			t.Errorf("%s: edit %q, want %q (entry %+v)", path, p.Edit, want[path], p)
		}
	}
	for path := range want {
		if _, ok := pages[path]; !ok {
			t.Errorf("no page %s", path)
		}
	}
}

func TestEditRelease(t *testing.T) {
	r := editRepo(t)
	pages := editBuild(t, Options{Source: tagTree(t, r), Main: r.Dir, Release: "v1.0.0"}, "docs")
	wantEdits(t, pages, map[string]string{
		"guide.md":                        "docs/site/guide.md",
		"reference/operator-resources.md": "docs/site/reference/operator-resources.md", // completed: authored
		"start/install.md":                "",                                          // renamed on main
		"reference/generated.md":          "",
	})
	if p := pages["start/install.md"]; p.Source != "docs/site/start/install.md" {
		t.Errorf("renamed page keeps its source: %+v", p)
	}
}

// The default main tree is the current directory when it is a work tree
// of the repository built, as in publish.yml's release mode.
func TestEditReleaseFromMainCheckout(t *testing.T) {
	r := editRepo(t)
	src := tagTree(t, r)
	t.Chdir(r.Dir)
	pages := editBuild(t, Options{Source: src, Release: "v1.0.0"}, "docs")
	if pages["guide.md"].Edit != "docs/site/guide.md" || pages["start/install.md"].Edit != "" {
		t.Fatalf("pages %+v", pages)
	}
}

func TestEditEdge(t *testing.T) {
	r := editRepo(t)
	// The test's directory is another repository, so the source tree is
	// the main tree.
	pages := editBuild(t, Options{Source: r.Dir}, "docs")
	wantEdits(t, pages, map[string]string{
		"guide.md":                        "docs/site/guide.md",
		"new.md":                          "docs/site/new.md",
		"start/setup.md":                  "docs/site/start/setup.md",
		"reference/operator-resources.md": "docs/site/reference/operator-resources.md",
		"reference/generated.md":          "",
	})
}

func TestEditRevision(t *testing.T) {
	r := editRepo(t)
	src := tagTree(t, r)
	fix := r.Head()
	// The fix's new page, applied and staged as revise leaves the tree.
	if err := os.WriteFile(filepath.Join(src, "docs", "site", "new.md"), []byte(strings.Replace(authoredOps, "Operator resources", "New", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	r.Git("-C", src, "add", "docs/site/new.md")
	pages := editBuild(t, Options{Source: src, Main: r.Dir, Release: "v1.0.0", Revision: 1, Patches: []string{fix}}, "docs")
	wantEdits(t, pages, map[string]string{
		"guide.md":                        "docs/site/guide.md",
		"new.md":                          "docs/site/new.md",
		"start/install.md":                "",
		"reference/operator-resources.md": "docs/site/reference/operator-resources.md",
		"reference/generated.md":          "",
	})
}

// A tab bundle's pages never carry edit, so a site whose opm-docs
// predates it still reads every catalog bundle.
func TestEditNotInTabBundle(t *testing.T) {
	r := editRepo(t)
	pages := editBuild(t, Options{Source: r.Dir, Main: r.Dir}, "tab")
	if len(pages) == 0 {
		t.Fatal("no pages")
	}
	for path, p := range pages {
		if p.Edit != "" {
			t.Errorf("%s: edit %q in a tab bundle", path, p.Edit)
		}
	}
}

// A release build with no checkout of main writes no edit and says so; it
// never reads the tag tree as main.
func TestEditReleaseWithoutMain(t *testing.T) {
	r := editRepo(t)
	var stderr strings.Builder
	// The test's directory is another repository, and no Main is given.
	pages := editBuild(t, Options{Source: tagTree(t, r), Release: "v1.0.0", Stderr: &stderr}, "docs")
	for path, p := range pages {
		if p.Edit != "" {
			t.Errorf("%s: edit %q without a main tree", path, p.Edit)
		}
	}
	if !strings.Contains(stderr.String(), "get no edit path") || !strings.Contains(stderr.String(), "v1.0.0") {
		t.Errorf("stderr %q", stderr.String())
	}
}

// Options.Edits, set by revise for a rebuild, replaces the main tree.
func TestEditFixed(t *testing.T) {
	r := editRepo(t)
	pages := editBuild(t, Options{Source: tagTree(t, r), Main: r.Dir, Release: "v1.0.0",
		Edits: map[string]string{"start/install.md": "docs/site/start/install.md"}}, "docs")
	wantEdits(t, pages, map[string]string{
		"start/install.md":                "docs/site/start/install.md",
		"guide.md":                        "",
		"reference/operator-resources.md": "",
		"reference/generated.md":          "",
	})
}
