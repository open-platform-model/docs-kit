package build

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/gittest"
	"github.com/open-platform-model/docs-kit/internal/render"
)

const fakeSchema = "docs.opmodel.dev/data/fake/v1"

// fakeExtractor writes its config's pages list as its data file; its
// renderer turns each entry into a page, the first one completable.
type fakeExtractor struct {
	pages []string
	file  string // the data file; "" is fake.json
}

func (fakeExtractor) Kind() string { return "fake" }

func (f fakeExtractor) Extract(context.Context, Input) (Data, error) {
	b, err := json.Marshal(f.pages)
	file := f.file
	if file == "" {
		file = "fake.json"
	}
	return Data{File: file, Schema: fakeSchema, Bytes: b}, err
}

type fakeRenderer struct{}

func (fakeRenderer) Schema() string { return fakeSchema }

const fakeTail = "## Resources\n\nGenerated.\n"

func (fakeRenderer) Render(data []byte, _ render.Target) ([]render.Page, error) {
	var paths []string
	if err := json.Unmarshal(data, &paths); err != nil {
		return nil, err
	}
	var out []render.Page
	for i, p := range paths {
		out = append(out, render.Page{
			Path:        p,
			Body:        "---\ntitle: \"Generated\"\ndescription: \"x\"\ntype: reference\n---\n\n" + fakeTail,
			Completable: i == 0,
			Heading:     "## Resources",
			Tail:        fakeTail,
		})
	}
	return out, nil
}

const authoredOps = "---\ntitle: \"Operator resources\"\ndescription: \"x\"\ntype: reference\n---\n\nIntro, with the [catalog](/catalogs/opm/4/).\n"

// docsBuild builds a docs-placed bundle owning owns from a fake extractor
// writing pages and a markdown source over docs/site holding files.
func docsBuild(t *testing.T, owns, pages []string, files map[string]string) (*bundle.Manifest, string, error) {
	t.Helper()
	return docsBuildWith(t, owns, fakeExtractor{pages: pages}, files)
}

func docsBuildWith(t *testing.T, owns []string, fake fakeExtractor, files map[string]string) (*bundle.Manifest, string, error) {
	t.Helper()
	extractors = append(extractors, fake)
	rendererFor = func(s string) (render.Renderer, error) {
		if s == fakeSchema {
			return fakeRenderer{}, nil
		}
		return render.For(s)
	}
	t.Cleanup(func() { extractors = extractors[:len(extractors)-1]; rendererFor = render.For })
	t.Setenv("GITHUB_REPOSITORY", "")
	r := gittest.New(t, "https://github.com/example/cli.git")
	files["README.md"] = "x\n"
	r.Write(files)
	r.Commit("docs")
	cfg := &config.Config{Path: "docs-kit.cue", Bundles: map[string]config.Bundle{"cli": {
		Placement: config.Placement{Kind: "docs", Root: "/docs/", Owns: owns},
		Version:   config.VersionRule{From: "tag", Prefix: "v"},
		Sources:   []config.Source{{Kind: "fake"}, {Kind: config.KindMarkdown, Markdown: &config.Markdown{Dir: "docs/site"}}},
	}}}
	out := filepath.Join(t.TempDir(), "out")
	_, err := buildProject(context.Background(), Options{Source: r.Dir, Out: out, Tool: "0.1.0"}, cfg, "cli", false)
	if err != nil {
		return nil, "", err
	}
	dir := filepath.Join(out, "cli")
	m, rerr := bundle.Read(dir)
	if rerr != nil {
		t.Fatal(rerr)
	}
	return m, dir, nil
}

func pageEntry(m *bundle.Manifest, path string) bundle.Page {
	for _, p := range m.Pages {
		if p.Path == path {
			return p
		}
	}
	return bundle.Page{}
}

func TestDocsCompletablePage(t *testing.T) {
	m, dir, err := docsBuild(t, []string{"reference/"}, []string{"reference/operator-resources.md"},
		map[string]string{"docs/site/reference/operator-resources.md": authoredOps, "docs/site/guide.md": strings.Replace(authoredOps, "Intro", "Guide", 1)})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Join(dir, "content", "reference", "operator-resources.md"))
	if string(body) != authoredOps+"\n"+fakeTail {
		t.Fatalf("completed page:\n%s", body)
	}
	if p := pageEntry(m, "reference/operator-resources.md"); p.Generated || p.Source != "docs/site/reference/operator-resources.md" {
		t.Fatalf("entry %+v", p)
	}
	if m.Placement.Kind != "docs" || len(m.Placement.Owns) != 1 || m.Placement.Owns[0] != "reference/" {
		t.Fatalf("placement %+v", m.Placement)
	}
	guide, _ := os.ReadFile(filepath.Join(dir, "content", "guide.md"))
	if !strings.Contains(string(guide), "(/catalogs/opm/4/)") {
		t.Fatalf("a docs bundle's authored page was rewritten:\n%s", guide)
	}
}

func TestDocsCompletablePageAlone(t *testing.T) {
	m, dir, err := docsBuild(t, []string{"reference/"}, []string{"reference/operator-resources.md"},
		map[string]string{"docs/site/guide.md": authoredOps})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Join(dir, "content", "reference", "operator-resources.md"))
	if !strings.HasPrefix(string(body), "---\ntitle: \"Generated\"") {
		t.Fatalf("standalone page:\n%s", body)
	}
	if p := pageEntry(m, "reference/operator-resources.md"); !p.Generated {
		t.Fatalf("entry %+v", p)
	}
}

func TestDocsRefusals(t *testing.T) {
	for _, c := range []struct {
		name  string
		owns  []string
		pages []string
		files map[string]string
		want  []string
	}{
		{"heading collision", []string{"reference/"}, []string{"reference/operator-resources.md"},
			map[string]string{"docs/site/reference/operator-resources.md": authoredOps + "\n## Resources\n"},
			[]string{"docs/site/reference/operator-resources.md", "## Resources"}},
		{"generated page outside owns", []string{"reference/cli/"}, []string{"reference/cli/opm.md", "reference/commands/opm.md"},
			map[string]string{"docs/site/guide.md": authoredOps},
			[]string{"content/reference/commands/opm.md", "placement.owns", "reference/cli/"}},
		{"completed page outside owns", []string{"reference/cli/"}, []string{"reference/operator-resources.md"},
			map[string]string{"docs/site/reference/operator-resources.md": authoredOps},
			[]string{"content/reference/operator-resources.md", "placement.owns"}},
		{"page path with ..", []string{"reference/"}, []string{"reference/../escape.md"},
			map[string]string{"docs/site/guide.md": authoredOps},
			[]string{"fake rendered the page path", "reference/../escape.md"}},
		{"page rendered twice", []string{"reference/"}, []string{"reference/a.md", "reference/a.md"},
			map[string]string{"docs/site/guide.md": authoredOps},
			[]string{"content/reference/a.md", "both fake and fake"}},
		{"owning nothing", nil, []string{"reference/cli/opm.md"},
			map[string]string{"docs/site/guide.md": authoredOps},
			[]string{"content/reference/cli/opm.md", "owns nothing"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := docsBuild(t, c.owns, c.pages, c.files)
			var ue *UsageError
			if err == nil || errors.As(err, &ue) {
				t.Fatalf("err = %v, want an execution error", err)
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not name %q", err, w)
				}
			}
		})
	}
}

func TestBadDataFile(t *testing.T) {
	for _, f := range []string{"../escape.json", "Data.json", "sub/x.json"} {
		_, _, err := docsBuildWith(t, []string{"reference/"}, fakeExtractor{pages: []string{"reference/a.md"}, file: f},
			map[string]string{"docs/site/guide.md": authoredOps})
		if err == nil || !strings.Contains(err.Error(), "fake wrote the data file") {
			t.Errorf("%s: err = %v", f, err)
		}
	}
}

// A committed generated page left in the markdown dir collides with the
// extractor's page of the same path, naming both sources.
func TestDocsCommittedGeneratedPage(t *testing.T) {
	_, _, err := docsBuild(t, []string{"reference/"}, []string{"reference/a.md", "reference/components.md"},
		map[string]string{"docs/site/reference/components.md": authoredOps})
	if err == nil {
		t.Fatal("no collision")
	}
	for _, w := range []string{"content/reference/components.md", "markdown docs/site", "fake"} {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error %q does not name %q", err, w)
		}
	}
}
