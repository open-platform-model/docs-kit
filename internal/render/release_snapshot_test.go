package render

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/doctext"
)

// A release snapshot is what hack/refresh-snapshots.sh copied from one
// released docs bundle into testdata/<kind>/: the release commit's
// docs-kit.cue and the inputs its bundle was built from, and under want/ the
// bundle's pages in its owned section (see each snapshot's SOURCE). A parity
// test renders the inputs with this checkout and compares every page with
// want/, so a change to what docs-kit writes for a released tree fails until
// the snapshot is refreshed from a release built with that change.

// snapshotSource loads testdata/<kind>/docs-kit.cue and returns the
// snapshot's directory and its one bundle's source of the given kind.
func snapshotSource(t *testing.T, kind, source string) (string, config.Source) {
	t.Helper()
	dir := filepath.Join("testdata", kind)
	cfg, err := config.Load(filepath.Join(dir, "docs-kit.cue"))
	if err != nil {
		t.Fatal(err)
	}
	projects := cfg.Projects()
	if len(projects) != 1 {
		t.Fatalf("%s/docs-kit.cue configures bundles %v, want one", dir, projects)
	}
	var found []config.Source
	for _, s := range cfg.Bundles[projects[0]].Sources {
		if s.Kind == source {
			found = append(found, s)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s/docs-kit.cue has %d %s sources, want one", dir, len(found), source)
	}
	return dir, found[0]
}

// citationPolicy is what a source's citations setting makes of a
// citation, as the build reads it.
func citationPolicy(citations string) doctext.Policy {
	if doctext.Policy(citations) == doctext.Link {
		return doctext.Link
	}
	return doctext.Strip
}

// compareWithRelease compares the rendered pages with the released pages
// under <dir>/want/<section>, page set and bodies. rewrite, when set, is
// applied to each released page first.
func compareWithRelease(t *testing.T, dir, section string, pages []Page, rewrite func(string) string) {
	t.Helper()
	wantDir := filepath.Join(dir, "want", filepath.FromSlash(section))
	entries, err := os.ReadDir(wantDir)
	if err != nil {
		t.Fatal(err)
	}
	want := make([]string, 0, len(entries))
	for _, e := range entries {
		want = append(want, section+e.Name())
	}
	got := make([]string, 0, len(pages))
	for _, p := range pages {
		got = append(got, p.Path)
		b, err := os.ReadFile(filepath.Join(dir, "want", filepath.FromSlash(p.Path)))
		if err != nil {
			t.Errorf("%s: the release has no such page", p.Path)
			continue
		}
		released := string(b)
		if rewrite != nil {
			released = rewrite(released)
		}
		if released != p.Body {
			t.Errorf("%s differs from the released page:\n%s", p.Path, firstDiff(released, p.Body))
		}
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("rendered pages %v, released pages %v", got, want)
	}
}

// firstDiff shows the first line where two pages differ.
func firstDiff(released, rendered string) string {
	al, bl := strings.Split(released, "\n"), strings.Split(rendered, "\n")
	for i := 0; i < len(al) || i < len(bl); i++ {
		var x, y string
		if i < len(al) {
			x = al[i]
		}
		if i < len(bl) {
			y = bl[i]
		}
		if x != y {
			return fmt.Sprintf("line %d:\n  released: %s\n  rendered: %s", i+1, x, y)
		}
	}
	return ""
}
