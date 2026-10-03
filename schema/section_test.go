package schema

import (
	"fmt"
	"strings"
	"testing"
)

func TestSectionPlacement(t *testing.T) {
	for _, c := range []struct {
		json string
		ok   bool
	}{
		{`{"kind": "section", "root": "/enhancements/"}`, true},
		{`{"kind": "section", "root": "/docs/"}`, false},
		{`{"kind": "section", "root": "/enhancements/rfcs/"}`, false},
		{`{"kind": "section", "root": "/enhancements/", "owns": ["x/"]}`, false},
	} {
		_, err := ValidateJSON("#Placement", "placement.json", []byte(c.json))
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok %v", c.json, err, c.ok)
		}
	}
}

// sectionManifest is a section bundle's manifest.json with the given
// version and first page.
func sectionManifest(version, page string) string {
	rev := "0"
	return fmt.Sprintf(`{
  "schema": "docs.opmodel.dev/bundle/v1", "project": "enhancements", "version": %q, "revision": %s,
  "source": {"repo": "open-platform-model/enhancements", "commit": "%s", "ref": "main"},
  "tool": "0.5.0", "dialect": 1,
  "placement": {"kind": "section", "root": "/enhancements/"},
  "pages": [%s],
  "data": [{"path": "enhancements.json", "schema": "docs.opmodel.dev/data/enhancements/v1"}]
}`, version, rev, strings.Repeat("a", 40), page)
}

func TestSectionManifest(t *testing.T) {
	page := `{"path": "_index.md", "source": "INDEX.md", "generated": true}`
	for _, c := range []struct {
		name, json string
		ok         bool
	}{
		{"edge", sectionManifest("edge", page), true},
		{"a release", sectionManifest("1.0.0", page), false},
		{"an edit", sectionManifest("edge", `{"path": "_index.md", "source": "INDEX.md", "generated": false, "edit": "INDEX.md"}`), false},
	} {
		_, err := ValidateJSON("#Manifest", "manifest.json", []byte(c.json))
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok %v", c.name, err, c.ok)
		}
	}
}

// TestSectionBundleVersionOptional checks a section bundle's config may
// leave out version, and every other bundle still needs one.
func TestSectionBundleVersionOptional(t *testing.T) {
	src := `"sources": [{"kind": "markdown", "dir": "docs"}]`
	for _, c := range []struct {
		json string
		ok   bool
	}{
		{`{"placement": {"kind": "section", "root": "/enhancements/"}, ` + src + `}`, true},
		{`{"placement": {"kind": "section", "root": "/enhancements/"}, "version": {"from": "tag", "prefix": "v"}, ` + src + `}`, true},
		{`{"placement": {"kind": "docs", "root": "/docs/"}, ` + src + `}`, false},
		{`{"placement": {"kind": "tab", "root": "/catalogs/opm/"}, ` + src + `}`, false},
	} {
		_, err := ValidateJSON("#Bundle", "bundle.json", []byte(c.json))
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok %v", c.json, err, c.ok)
		}
	}
}
