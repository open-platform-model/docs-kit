package config

import (
	"strings"
	"testing"
)

func crdConfig(layout string) string {
	return `bundles: "opm-operator": {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{kind: "crd", dir: "./c", ` + layout + ` title: "T", description: "D"}]
}
`
}

// TestLoadCRDLayout: a crd source takes exactly one of page and section.
func TestLoadCRDLayout(t *testing.T) {
	for _, ok := range []string{`page: "reference/r.md",`, `section: "reference/operator/",`} {
		if _, err := Load(write(t, "docs-kit.cue", crdConfig(ok))); err != nil {
			t.Errorf("%s: %v", ok, err)
		}
	}
	for name, layout := range map[string]string{
		"both":    `page: "reference/r.md", section: "reference/operator/",`,
		"neither": ``,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Load(write(t, "docs-kit.cue", crdConfig(layout)))
			if err == nil || !strings.Contains(err.Error(), "a crd source takes exactly one of page") {
				t.Fatalf("%v", err)
			}
		})
	}
	if _, err := Load(write(t, "docs-kit.cue", crdConfig(`section: "reference/operator",`))); err == nil {
		t.Fatal("a section without its trailing slash loaded")
	}
}
