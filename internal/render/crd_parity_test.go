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

// enhancementParenRewrite drops, with its leading space, a parenthesis that
// holds only enhancement references, "(enhancement 0015)" or "(enhancements
// 0013 and 0019)". A bundle built before the strip policy removed such a
// parenthesis keeps it; the snapshot's bundle, built by opm-docs 0.7.1,
// predates that rule.
var enhancementParenRewrite = releaseRewrite{
	name: "enhancementParenRewrite",
	re:   regexp.MustCompile(`\s\((?:[Ee]nhancement\s+\d{4}|[Ee]nhancements\s+\d{4}(?:(?:,\s*|,?\s+and\s+)\d{4})*)\)`),
	repl: "",
}

// TestControllerCRDParity renders the controller's release snapshot with
// its own config, completes the section index with the authored one, and
// compares every page with the released bundle's. The snapshot's bundle
// predates the strip of a parenthesis that holds only enhancement
// references, so those parentheses are rewritten before comparing
// (enhancementParenRewrite); once a refresh brings a bundle built with the
// rule the rewrite matches nothing and the test asks for its removal.
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
	compareWithRelease(t, o.Root, o.Section, pages, []releaseRewrite{enhancementParenRewrite})
}
