package mdsafe

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/mod/modfile"
)

// TestGoldmarkPinned fails when the module requires another goldmark than
// the one opmodel.dev's pinned Hugo builds with: the check must parse a
// page as the site does, so a stray "go get -u" breaks the build. Bump
// GoldmarkVersion only with the site's Hugo.
func TestGoldmarkPinned(t *testing.T) {
	file := filepath.Join("..", "..", "go.mod")
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	f, err := modfile.Parse(file, b, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range f.Replace {
		if r.Old.Path == "github.com/yuin/goldmark" {
			t.Fatalf("go.mod replaces goldmark; the check must use goldmark %s as Hugo %s does", GoldmarkVersion, HugoVersion)
		}
	}
	for _, r := range f.Require {
		if r.Mod.Path != "github.com/yuin/goldmark" {
			continue
		}
		if r.Mod.Version != GoldmarkVersion {
			t.Fatalf("go.mod requires goldmark %s, but Hugo %s builds with %s; keep goldmark at the site's Hugo's version", r.Mod.Version, HugoVersion, GoldmarkVersion)
		}
		return
	}
	t.Fatal("go.mod does not require goldmark")
}
