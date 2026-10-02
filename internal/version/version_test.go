package version

import "testing"

func TestString_NamesTheTool(t *testing.T) {
	if got, want := String(), "opm-docs "+Version; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
