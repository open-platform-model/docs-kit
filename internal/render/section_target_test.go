package render

import "testing"

// TestSectionTargetURL checks a section page publishes at its root with
// no segment.
func TestSectionTargetURL(t *testing.T) {
	tg := Target{Kind: KindSection, Root: "/enhancements/", Segment: "edge", Edge: true, Version: "edge"}
	for page, want := range map[string]string{"": "/enhancements/", "graph": "/enhancements/graph/", "0025/decisions": "/enhancements/0025/decisions/"} {
		if got := tg.URL(page); got != want {
			t.Errorf("URL(%q) = %q, want %q", page, got, want)
		}
	}
}
