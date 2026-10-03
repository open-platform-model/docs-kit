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
