package schema

import "testing"

func TestDefinitionsLoad(t *testing.T) {
	for _, d := range []string{"#Manifest", "#Config", "#Pull", "#Lock", "#History"} {
		if _, v, err := Def(d); err != nil || !v.Exists() {
			t.Fatalf("%s: %v", d, err)
		}
	}
}
