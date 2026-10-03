package render

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/mdsafe"
)

// TestGoldenPagesPassTheMarkupCheck runs every golden page of every
// renderer through the markup check with every rule, as bundle-mode lint
// runs it on a generated page.
func TestGoldenPagesPassTheMarkupCheck(t *testing.T) {
	n := 0
	err := filepath.WalkDir(filepath.Join("testdata", "golden"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		n++
		for _, v := range mdsafe.Check(mdsafe.Page{Path: p, Body: b}, mdsafe.Generated) {
			t.Error(v)
		}
		return nil
	})
	if err != nil || n < 60 {
		t.Fatalf("%d golden pages: %v", n, err)
	}
}

// A cue-catalog note that reads as a heading with an attribute block, or
// holds raw HTML or a shortcode, renders as text.
func TestCatalogNoteMarkup(t *testing.T) {
	m := fixtureModel(t)
	m.Members[0].Notes = append(m.Members[0].Notes,
		`## Install {.hx:fixed style="position:fixed"}`,
		"Setext {#kernel}\n---",
		"<!-- c --> <details open ontoggle=alert(1)> {{< x >}}")
	pages := renderAll(t, m, Target{Root: "/catalogs/demo/", Segment: "1.2", Repo: "example/demo", Commit: commit})
	found := false
	for path, body := range pages {
		for _, v := range mdsafe.Check(mdsafe.Page{Path: path, Body: []byte(body)}, mdsafe.Generated) {
			t.Error(v)
		}
		found = found || strings.Contains(body, `## Install \{.hx:fixed style="position:fixed"\}`)
	}
	if !found {
		t.Error("no page holds the escaped heading")
	}
}
