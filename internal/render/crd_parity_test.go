package render

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/extract/crd"
)

// controllerOptions is the crd source of the controller's release snapshot
// under testdata/controller, as its docs-kit.cue configures it, with the
// snapshot as the source tree.
func controllerOptions(t *testing.T) crd.Options {
	t.Helper()
	root, src := snapshotSource(t, "controller", "crd")
	var cfg struct {
		Dir                 string            `json:"dir"`
		Samples             string            `json:"samples"`
		HideSamplesMatching []string          `json:"hideSamplesMatching"`
		StripLabels         map[string]string `json:"stripLabels"`
		Page                string            `json:"page"`
		Section             string            `json:"section"`
		Title               string            `json:"title"`
		Description         string            `json:"description"`
		Weight              *int              `json:"weight"`
		Order               []string          `json:"order"`
		ReconciledBy        map[string]string `json:"reconciledBy"`
	}
	if err := src.Value.Decode(&cfg); err != nil {
		t.Fatal(err)
	}
	return crd.Options{
		Root: root, Dir: cfg.Dir, Samples: cfg.Samples,
		HideSamplesMatching: cfg.HideSamplesMatching, StripLabels: cfg.StripLabels,
		Page:    crd.Page{Path: cfg.Page, Title: cfg.Title, Description: cfg.Description, Weight: cfg.Weight},
		Section: cfg.Section,
		Order:   cfg.Order, ReconciledBy: cfg.ReconciledBy, Doc: citationPolicy(src.Citations),
	}
}

// releasedServedBy matches the served-by sentence of a bundle built before
// the renderer stopped naming the product: "The operator's `x` controller
// watches every X."
var releasedServedBy = regexp.MustCompile("(?m)^The operator's (`[^`]+`) controller watches every ([A-Z][A-Za-z0-9]*)\\.$")

// TestControllerCRDParity renders the controller's release snapshot with
// its own config, completes the section index with the authored one, and
// compares every page with the released bundle's. The snapshot's bundle
// predates the reconciler wording of the served-by sentence, so its
// sentences are rewritten before comparing; once a refresh brings a bundle
// built with that wording the rewrite matches nothing and the test asks
// for its removal.
func TestControllerCRDParity(t *testing.T) {
	o := controllerOptions(t)
	if o.Section == "" {
		t.Fatal("the snapshot's crd source has no section; the parity test compares a section")
	}
	m, err := crd.Extract(o)
	if err != nil {
		t.Fatal(err)
	}
	data, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	pages, err := crdRenderer{}.Render(data, Target{Kind: KindDocs, Root: "/docs/"})
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join("docs", "site", filepath.FromSlash(pages[0].Path))
	if authored, err := os.ReadFile(filepath.Join(o.Root, name)); err == nil {
		body, err := Complete(string(authored), name, pages[0])
		if err != nil {
			t.Fatal(err)
		}
		pages[0] = Page{Path: pages[0].Path, Body: body}
	}
	rewrote := false
	compareWithRelease(t, o.Root, o.Section, pages, func(s string) string {
		out := releasedServedBy.ReplaceAllString(s, "The $1 reconciler watches every $2.")
		rewrote = rewrote || out != s
		return out
	})
	if !rewrote {
		t.Error("no released page holds the product-named served-by sentence: delete releasedServedBy and the rewrite")
	}
}
