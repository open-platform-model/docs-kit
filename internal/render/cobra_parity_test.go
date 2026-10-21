package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/extract/cobra"
)

// TestCLICommandParity renders the cli's release snapshot under
// testdata/cli (see its SOURCE): the dump the release's hack/docskit-dump
// printed, with the cobra source of the release's docs-kit.cue, and compares
// every page with the released bundle's.
func TestCLICommandParity(t *testing.T) {
	dir, src := snapshotSource(t, "cli", "cobra")
	var cfg struct {
		Section     string `json:"section"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Weight      int    `json:"weight"`
	}
	if err := src.Value.Decode(&cfg); err != nil {
		t.Fatal(err)
	}
	dump, err := os.ReadFile(filepath.Join(dir, "dump.json"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := cobra.FromDump(dump, cobra.Options{
		Section: cfg.Section, Title: cfg.Title, Description: cfg.Description, Weight: cfg.Weight,
		Citations: string(citationPolicy(src.Citations)),
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	pages, err := cobraRenderer{}.Render(data, Target{Kind: KindDocs, Root: "/docs/"})
	if err != nil {
		t.Fatal(err)
	}
	compareWithRelease(t, dir, cfg.Section, pages, nil)
}
