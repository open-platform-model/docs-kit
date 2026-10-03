package schema

import (
	"slices"
	"testing"
)

func TestDefinitionsLoad(t *testing.T) {
	for _, d := range []string{"#Manifest", "#Config", "#Pull", "#Lock", "#History"} {
		if _, v, err := Def(d); err != nil || !v.Exists() {
			t.Fatalf("%s: %v", d, err)
		}
	}
}

func TestSourceKinds(t *testing.T) {
	kinds, err := SourceKinds()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(kinds, "crd") || kinds[0] != "cue-catalog" || kinds[1] != "markdown" {
		t.Fatalf("kinds %v", kinds)
	}
}

func TestPlacement(t *testing.T) {
	for _, c := range []struct {
		json string
		ok   bool
	}{
		{`{"kind": "tab", "root": "/catalogs/opm/"}`, true},
		{`{"kind": "docs", "root": "/docs/"}`, true},
		{`{"kind": "docs", "root": "/docs/", "owns": ["reference/cli/", "reference/operator-resources.md"]}`, true},
		{`{"kind": "docs", "root": "/docs/", "owns": ["reference/cli"]}`, false},
		{`{"kind": "docs", "root": "/docs/", "owns": ["/reference/"]}`, false},
		{`{"kind": "docs", "root": "/docs/", "owns": ["reference/../x/"]}`, false},
		{`{"kind": "tab", "root": "/catalogs/opm/", "owns": ["x/"]}`, false},
	} {
		_, err := ValidateJSON("#Placement", "placement.json", []byte(c.json))
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok %v", c.json, err, c.ok)
		}
	}
}

func TestPageEdit(t *testing.T) {
	for _, c := range []struct {
		json string
		ok   bool
	}{
		{`{"path": "start/install.md", "source": "docs/site/start/install.md", "generated": false}`, true},
		{`{"path": "start/install.md", "source": "docs/site/start/install.md", "generated": false, "edit": "docs/site/start/install.md"}`, true},
		{`{"path": "start/install.md", "generated": false, "edit": ""}`, false},
		{`{"path": "start/install.md", "generated": false, "edit": 1}`, false},
		{`{"path": "start/install.md", "generated": true, "edit": "docs/site/start/install.md"}`, false},
		{`{"path": "start/install.md", "generated": false, "edit": "/docs/site/start/install.md"}`, false},
		{`{"path": "start/install.md", "generated": false, "edit": "./docs/site/start/install.md"}`, false},
		{`{"path": "start/install.md", "generated": false, "edit": "docs/../etc/passwd"}`, false},
		{`{"path": "start/install.md", "generated": false, "edit": "docs/site/.."}`, false},
		{`{"path": "start/install.md", "generated": false, "edit": "docs/site/..notes.md"}`, true},
	} {
		_, err := ValidateJSON("#Page", "page.json", []byte(c.json))
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok %v", c.json, err, c.ok)
		}
	}
}
