package schema

import "testing"

func TestDefinitionsLoad(t *testing.T) {
	for _, d := range []string{"#Manifest", "#Config", "#Pull", "#Lock"} {
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
	if len(kinds) != 2 || kinds[0] != "cue-catalog" || kinds[1] != "markdown" {
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
