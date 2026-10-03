package render

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/dialect"
	"github.com/open-platform-model/docs-kit/internal/extract/enhancements"
)

const enhancementsRepo = "../extract/enhancements/testdata/repo"

var sectionTarget = Target{Kind: KindSection, Root: "/enhancements/", Segment: "edge", Edge: true, Version: "edge",
	Repo: "open-platform-model/enhancements", Commit: "cccccccccccccccccccccccccccccccccccccccc"}

// enhancementsFixture renders the enhancements source's fixture
// repository, its paths listed as git ls-tree lists them.
func enhancementsFixture(t *testing.T) []Page {
	t.Helper()
	paths := map[string]string{}
	err := filepath.WalkDir(enhancementsRepo, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == enhancementsRepo {
			return err
		}
		rel, _ := filepath.Rel(enhancementsRepo, p)
		paths[filepath.ToSlash(rel)] = map[bool]string{true: "tree", false: "blob"}[d.IsDir()]
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := enhancements.Extract(enhancements.Options{
		Root: enhancementsRepo, Dir: ".", Repo: sectionTarget.Repo, Commit: sectionTarget.Commit, Paths: paths,
		Title: "Enhancements", Description: "OPM's design record.",
	})
	if err != nil {
		t.Fatal(err)
	}
	pages, err := Enhancements(res.Pages, sectionTarget)
	if err != nil {
		t.Fatal(err)
	}
	return pages
}

func TestEnhancementsGolden(t *testing.T) {
	pages := enhancementsFixture(t)
	dir := filepath.Join("testdata", "golden", "enhancements")
	if *update {
		_ = os.RemoveAll(dir)
	}
	paths := make([]string, 0, len(pages))
	for _, p := range pages {
		paths = append(paths, p.Path)
		file := filepath.Join(dir, filepath.FromSlash(p.Path))
		if *update {
			_ = os.MkdirAll(filepath.Dir(file), 0o755)
			if err := os.WriteFile(file, []byte(p.Body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("%v (run go test -run TestEnhancementsGolden -update)", err)
		}
		if string(want) != p.Body {
			t.Errorf("%s differs from its golden page:\n%s", file, p.Body)
		}
	}
	vs, err := dialect.Lint(dir, dialect.Options{Mode: dialect.Bundle, Bundle: dialect.BundleInfo{
		Kind: KindSection, Root: "/enhancements/", Segment: "edge", Pages: paths,
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		t.Errorf("golden page breaks the dialect: %s", v)
	}
}

func TestEnhancementsNeedsASection(t *testing.T) {
	_, err := Enhancements(nil, Target{Kind: KindDocs, Root: "/docs/"})
	if err == nil || !strings.Contains(err.Error(), `placement kind "section"`) {
		t.Fatalf("err = %v", err)
	}
}
