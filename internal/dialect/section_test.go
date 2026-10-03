package dialect

import (
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
