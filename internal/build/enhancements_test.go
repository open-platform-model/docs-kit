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
	"github.com/open-platform-model/docs-kit/internal/gittest"
)

const enhancementsSectionConfig = `bundles: enhancements: {
	placement: {kind: "section", root: "/enhancements/"}
	sources: [{kind: "enhancements", description: "OPM's design record."}]
}
`

// enhancementsRepo is the enhancements source's fixture repository,
// committed with its docs-kit.cue.
func enhancementsRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	t.Setenv("GITHUB_REPOSITORY", "")
	r := gittest.New(t, "https://github.com/open-platform-model/enhancements.git")
	r.CopyTree(filepath.Join("..", "extract", "enhancements", "testdata", "repo"), ".")
	r.Write(map[string]string{"docs-kit.cue": enhancementsSectionConfig})
	r.Commit("entries")
	return r
}

func buildEnhancements(t *testing.T, r *gittest.Repo, release string) (string, error) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "out")
	_, err := Run(context.Background(), Options{Source: r.Dir, Out: out, Tool: "0.5.0", Release: release})
	return filepath.Join(out, "enhancements"), err
}

// builtEnhancements builds r and reads its manifest, failing on any error.
func builtEnhancements(t *testing.T, r *gittest.Repo) (string, *bundle.Manifest) {
	t.Helper()
	dir, err := buildEnhancements(t, r, "")
	if err != nil {
		var le *LintError
		if errors.As(err, &le) {
			t.Fatalf("%v: %s", err, strings.Join(le.Violations, "; "))
		}
		t.Fatal(err)
	}
	m, err := bundle.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir, m
}

func TestEnhancementsSectionBuild(t *testing.T) {
	r := enhancementsRepo(t)
	dir, m := builtEnhancements(t, r)
	if m.Version != "edge" || m.Placement.Kind != "section" || m.Placement.Root != "/enhancements/" || len(m.Pages) != 18 {
		t.Fatalf("manifest %s %+v, %d pages", m.Version, m.Placement, len(m.Pages))
	}
	if len(m.Data) != 1 || m.Data[0].Path != "enhancements.json" || m.Data[0].Schema != "docs.opmodel.dev/data/enhancements/v1" {
		t.Fatalf("data %+v", m.Data)
	}
	for _, p := range m.Pages {
		if !p.Generated || p.Edit != "" || p.Source == "" || p.Lastmod == "" {
			t.Errorf("page %+v: a section page is generated, has a source and lastmod, and never an edit", p)
		}
	}
	if p := pageEntry(m, "0003/decisions.md"); p.Source != "archive/0003/03-decisions.md" {
		t.Errorf("archived document %+v", p)
	}
	body, _ := os.ReadFile(filepath.Join(dir, "content", "0025", "_index.md"))
	if want := "https://github.com/open-platform-model/enhancements/blob/" + r.Head() + "/0025/config.yaml"; !strings.Contains(string(body), want) {
		t.Errorf("no link to %s in\n%s", want, body)
	}
}

// The same commit builds the same bytes.
func TestEnhancementsSectionDeterministic(t *testing.T) {
	r := enhancementsRepo(t)
	dir, _ := builtEnhancements(t, r)
	again, err := buildEnhancements(t, r, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"manifest.json", "data/enhancements.json", "content/0025/design.md"} {
		a, _ := os.ReadFile(filepath.Join(dir, f))
		b, _ := os.ReadFile(filepath.Join(again, f))
		if !bytes.Equal(a, b) {
			t.Errorf("%s differs between two builds of one commit", f)
		}
	}
}

func TestEnhancementsSectionRefusesRelease(t *testing.T) {
	r := enhancementsRepo(t)
	r.Git("tag", "v1.0.0")
	_, err := buildEnhancements(t, r, "v1.0.0")
	var ue *UsageError
	if !errors.As(err, &ue) || !strings.Contains(err.Error(), "enhancements is a section bundle; it builds from main only (edge)") {
		t.Fatalf("err = %v, want a usage error naming the section", err)
	}
}

// A relative link resolves against the commit built: a file only in the
// work tree is not there.
func TestEnhancementsLinkAtTheCommit(t *testing.T) {
	r := enhancementsRepo(t)
	r.Write(map[string]string{
		"0025/notes.md":          "Notes.\n",
		"0025/06-operational.md": "# Operational\n\nSee [the notes](notes.md).\n",
	})
	_, err := buildEnhancements(t, r, "")
	if err == nil || !strings.Contains(err.Error(), `0025/06-operational.md:3: link "notes.md" names nothing in the repository at`) {
		t.Fatalf("err = %v", err)
	}
}
