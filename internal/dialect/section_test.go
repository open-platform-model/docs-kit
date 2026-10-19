package dialect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSectionBundleMode lints a section bundle: a link into its root
// names one of its pages, and /catalogs/ links follow the docs-mode rules.
func TestSectionBundleMode(t *testing.T) {
	dir := filepath.Join("testdata", "section-bundle")
	pages := []string{"_index.md", "0025/_index.md", "0025/decisions.md"}
	vs, err := Lint(filepath.Join(dir, "content"), Options{Mode: Bundle, Bundle: BundleInfo{Kind: "section", Root: "/enhancements/", Segment: "edge", Pages: pages}})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(vs))
	for _, v := range vs {
		got = append(got, strings.TrimPrefix(v.String(), dir+"/"))
	}
	compare(t, "section-bundle", got, readLines(t, filepath.Join(dir, "expect")))
}

// TestSectionBundleNoIndex reports a link to the section page when the
// bundle has none.
func TestSectionBundleNoIndex(t *testing.T) {
	dir := filepath.Join("testdata", "section-bundle", "content")
	vs, err := Lint(dir, Options{Mode: Bundle, Bundle: BundleInfo{Kind: "section", Root: "/enhancements/", Segment: "edge", Pages: []string{"0025/_index.md", "0025/decisions.md"}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		if strings.Contains(v.Msg, `"/enhancements/": no page _index.md in this bundle`) {
			return
		}
	}
	t.Fatalf("no violation for the missing section page: %v", vs)
}

// TestSectionBundleGraph checks a link to the graph names the bundle's
// graph page.
func TestSectionBundleGraph(t *testing.T) {
	dir := filepath.Join("testdata", "section-bundle", "content")
	pages := []string{"_index.md", "0025/_index.md", "0025/decisions.md", "graph.md"}
	vs, err := Lint(dir, Options{Mode: Bundle, Bundle: BundleInfo{Kind: "section", Root: "/enhancements/", Segment: "edge", Pages: pages}})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		if strings.Contains(v.Msg, "graph") {
			t.Errorf("graph link refused: %s", v)
		}
	}
}

// TestSectionBundleCapabilities checks a link to a capability page names a
// page of the bundle.
func TestSectionBundleCapabilities(t *testing.T) {
	dir := t.TempDir()
	page := "---\ntitle: \"T\"\ndescription: \"D.\"\ntype: explanation\n---\n\n[all](/enhancements/capabilities/) [one](/enhancements/capabilities/single-build-render/) [gone](/enhancements/capabilities/gone/)\n"
	if err := os.WriteFile(filepath.Join(dir, "x.md"), []byte(page), 0o600); err != nil {
		t.Fatal(err)
	}
	pages := []string{"capabilities/_index.md", "capabilities/single-build-render.md", "x.md"}
	vs, err := Lint(dir, Options{Mode: Bundle, Bundle: BundleInfo{Kind: "section", Root: "/enhancements/", Segment: "edge", Pages: pages}})
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 || !strings.Contains(vs[0].Msg, `"/enhancements/capabilities/gone/": no page capabilities/gone/ in this bundle`) {
		t.Fatalf("violations %v, want the one for the missing capability page", vs)
	}
}
