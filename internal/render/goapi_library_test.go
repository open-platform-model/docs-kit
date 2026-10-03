package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/dialect"
	"github.com/open-platform-model/docs-kit/internal/doctext"
	"github.com/open-platform-model/docs-kit/internal/extract/goapi"
)

// libraryGoAPI extracts the library tree at root with the go-api source of
// the library's planned docs-kit.cue and renders it.
func libraryGoAPI(t *testing.T, root, version string) (*goapi.Model, []Page) {
	t.Helper()
	c, err := config.Load("../extract/goapi/testdata/library-docs-kit.cue")
	if err != nil {
		t.Fatal(err)
	}
	var cfg goapi.Config
	for _, s := range c.Bundles["library"].Sources {
		if s.Kind == "go-api" {
			if err := s.Value.Decode(&cfg); err != nil {
				t.Fatal(err)
			}
		}
	}
	r, err := goapi.Extract(goapi.Options{Source: root, Config: cfg, Version: version, Policy: doctext.Strip})
	if err != nil {
		t.Fatal(err)
	}
	data, err := r.Model.Encode()
	if err != nil {
		t.Fatal(err)
	}
	pages, err := goAPIRenderer{}.Render(data, goAPITarget)
	if err != nil {
		t.Fatal(err)
	}
	return r.Model, pages
}

// TestLibraryGoAPI renders the library's Go API from the frozen copy under
// testdata/library (see its SOURCE): its pages equal the golden pages,
// pass the dialect and every anchor and link resolves.
func TestLibraryGoAPI(t *testing.T) {
	m, pages := libraryGoAPI(t, filepath.Join("testdata", "library"), "1.0.0-beta.1")
	if len(m.Packages) != 9 || len(pages) != 10 {
		t.Errorf("%d packages, %d pages; want 9 and 10", len(m.Packages), len(pages))
	}
	checkGolden(t, pages, filepath.Join("testdata", "golden", "library"), []string{"reference/go-api/"})
	checkAnchors(t, m)
}

// TestLibraryGoAPILive renders the library checkout OPM_LIBRARY_CHECKOUT
// names: the pages pass the dialect, every anchor and link resolves, and
// the log lists the exported symbols without a doc comment.
func TestLibraryGoAPILive(t *testing.T) {
	root := os.Getenv("OPM_LIBRARY_CHECKOUT")
	if root == "" {
		t.Skip("OPM_LIBRARY_CHECKOUT names no library checkout")
	}
	m, pages := libraryGoAPI(t, root, "edge")
	dir := t.TempDir()
	names := make([]string, 0, len(pages))
	for _, p := range pages {
		full := filepath.Join(dir, filepath.FromSlash(p.Path))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(p.Body), 0o600); err != nil {
			t.Fatal(err)
		}
		names = append(names, p.Path)
	}
	vs, err := dialect.Lint(dir, dialect.Options{Mode: dialect.Bundle, Bundle: dialect.BundleInfo{
		Kind: KindDocs, Root: "/docs/", Segment: "1.0", Owns: []string{"reference/go-api/"}, Pages: names,
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		t.Errorf("a page breaks the dialect: %s", v)
	}
	checkAnchors(t, m)
	t.Logf("%d packages, %d pages; without a doc comment: %s", len(m.Packages), len(pages), strings.Join(undocumented(m), ", "))
}

// undocumented lists the exported symbols of a model without a doc
// comment, as "opm/kernel.RenderError.Error".
func undocumented(m *goapi.Model) []string {
	var out []string
	for i := range m.Packages {
		p := &m.Packages[i]
		rel := strings.TrimPrefix(p.ImportPath, m.ModulePath+"/")
		if p.Doc == "" {
			out = append(out, rel)
		}
		for _, s := range goapi.PageSections(p) {
			for _, e := range s.Entries {
				if e.Doc == "" {
					out = append(out, rel+"."+e.Key)
				}
			}
		}
	}
	return out
}
