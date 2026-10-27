package schema

import (
	"fmt"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/placeholder"
)

func unitsManifest(units string) string {
	return fmt.Sprintf(`{
  "schema": "docs.opmodel.dev/bundle/v1", "project": "opm-controller", "version": "1.0.0", "revision": 0,
  "source": {"repo": "open-platform-model/opm-controller", "commit": "%s", "ref": "v1.0.0"},
  "tool": "0.5.0", "dialect": 1,
  "placement": {"kind": "docs", "root": "/docs/"},
  "pages": [{"path": "_index.md", "generated": false}],
  "data": []%s
}`, strings.Repeat("a", 40), units)
}

func TestManifestUnits(t *testing.T) {
	for _, c := range []struct {
		name, units string
		ok          bool
	}{
		{"no units", ``, true},
		{"a module version", `, "units": {"opm-controller-module": "0.1.1"}`, true},
		{"a prerelease", `, "units": {"opm-controller-module": "0.2.0-beta.1"}`, true},
		{"a v prefix", `, "units": {"opm-controller-module": "v0.1.1"}`, false},
		{"an unknown unit", `, "units": {"x": "0.1.1"}`, false},
		{"a number", `, "units": {"opm-controller-module": 1}`, false},
	} {
		_, err := ValidateJSON("#Manifest", "manifest.json", []byte(unitsManifest(c.units)))
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok %v", c.name, err, c.ok)
		}
	}
}

func TestConfigUnits(t *testing.T) {
	base := `{"bundles": {"opm-controller": {"placement": {"kind": "docs", "root": "/docs/"}, "version": {"from": "tag", "prefix": "v"},
	"sources": [{"kind": "markdown", "dir": "docs/site"}]%s}}}`
	for _, c := range []struct {
		name, units string
		ok          bool
	}{
		{"none", ``, true},
		{"key only", `, "units": {"opm-controller-module": {"key": "modules/opm_controller"}}`, true},
		{"file and key", `, "units": {"opm-controller-module": {"file": "release/v.json", "key": "a"}}`, true},
		{"unit name x", `, "units": {"x": {"key": "a"}}`, false},
		{"empty key", `, "units": {"opm-controller-module": {"key": ""}}`, false},
		{"no key", `, "units": {"opm-controller-module": {}}`, false},
		{"parent path", `, "units": {"opm-controller-module": {"file": "../v.json", "key": "a"}}`, false},
		{"absolute path", `, "units": {"opm-controller-module": {"file": "/v.json", "key": "a"}}`, false},
	} {
		_, err := ValidateJSON("#Config", "docs-kit.json", []byte(fmt.Sprintf(base, c.units)))
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok %v", c.name, err, c.ok)
		}
	}
}

// TestUnitNamesMatchTheList keeps the schema's unit names equal to the unit
// names of the placeholder list.
func TestUnitNamesMatchTheList(t *testing.T) {
	for _, n := range placeholder.All() {
		units := fmt.Sprintf(`, "units": {%q: "0.1.1"}`, n.Name)
		_, err := ValidateJSON("#Manifest", "manifest.json", []byte(unitsManifest(units)))
		if (err == nil) != (n.Kind == placeholder.Unit) {
			t.Errorf("%s (unit %v): err = %v", n.Name, n.Kind == placeholder.Unit, err)
		}
	}
}
