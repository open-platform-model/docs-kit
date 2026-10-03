package dialect

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// repoOrder is the order the site's test runner passes docs/site roots in.
var repoOrder = []string{"opm", "core", "catalog_opm", "cli", "library", "opm-operator"}

// TestConformance lints every case of the conformance set and compares the
// output, line for line, with the shell lint's recorded output.
func TestConformance(t *testing.T) {
	cases, err := filepath.Glob("testdata/conformance/*/shell.out")
	if err != nil || len(cases) == 0 {
		t.Fatalf("no conformance cases: %v", err)
	}
	for _, out := range cases {
		dir := filepath.Dir(out)
		name := filepath.Base(dir)
		t.Run(name, func(t *testing.T) {
			ws := copyCase(t, dir)
			var got []string
			for _, root := range roots(t, ws) {
				vs, err := Lint(filepath.Join(ws, root), Options{})
				if err != nil {
					t.Fatal(err)
				}
				for _, v := range vs {
					got = append(got, strings.TrimPrefix(v.String(), ws+"/"))
				}
			}
			compare(t, name, got, readLines(t, out))
		})
	}
}

// TestBundleMode lints a 4.4 bundle's content tree in bundle mode.
func TestBundleMode(t *testing.T) {
	content := filepath.Join("testdata", "bundle", "content")
	var pages []string
	err := filepath.Walk(content, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(content, p)
			pages = append(pages, filepath.ToSlash(rel))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	vs, err := Lint(content, Options{Mode: Bundle, Bundle: BundleInfo{Root: "/catalogs/opm/", Segment: "4.4", Pages: pages}})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(vs))
	for _, v := range vs {
		got = append(got, strings.TrimPrefix(v.String(), filepath.Join("testdata", "bundle")+"/"))
	}
	compare(t, "bundle", got, readLines(t, filepath.Join("testdata", "bundle", "expect")))
}

// TestDocsBundleMode lints a docs-placed bundle: docs-mode catalog links,
// and /docs/ links into what it owns name one of its pages.
func TestDocsBundleMode(t *testing.T) {
	content := filepath.Join("testdata", "docs-bundle", "content")
	pages := []string{"reference/cli/_index.md", "reference/cli/opm-module.md", "reference/extra.md"}
	vs, err := Lint(content, Options{Mode: Bundle, Bundle: BundleInfo{Kind: "docs", Root: "/docs/", Segment: "1.0",
		Owns: []string{"reference/cli/", "reference/extra.md", "reference/gone.md"}, Pages: pages}})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(vs))
	for _, v := range vs {
		got = append(got, strings.TrimPrefix(v.String(), filepath.Join("testdata", "docs-bundle")+"/"))
	}
	compare(t, "docs-bundle", got, readLines(t, filepath.Join("testdata", "docs-bundle", "expect")))
}

// TestDocsModeRejectsBundleLinks checks that docs mode, unlike bundle
// mode, refuses a minor segment for the same tree.
func TestDocsModeRejectsBundleLinks(t *testing.T) {
	vs, err := Lint(filepath.Join("testdata", "bundle", "content"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range vs {
		if strings.Contains(v.Msg, `"/catalogs/opm/4.4/traits/backup/": docs pages link catalogs through /catalogs/opm/4/`) {
			found = true
		}
	}
	if !found {
		t.Fatalf("docs mode accepted a minor catalog link: %v", vs)
	}
}

func TestMissingDir(t *testing.T) {
	if _, err := Lint(filepath.Join(t.TempDir(), "nope"), Options{}); err == nil {
		t.Fatal("Lint of a missing directory succeeded")
	}
}

func compare(t *testing.T, name string, got, want []string) {
	t.Helper()
	n := max(len(got), len(want))
	for i := range n {
		var g, w string
		if i < len(got) {
			g = got[i]
		}
		if i < len(want) {
			w = want[i]
		}
		if g != w {
			t.Errorf("%s: line %d:\n got: %s\nwant: %s", name, i+1, g, w)
		}
	}
}

// copyCase copies a case directory into a temporary workspace and runs its
// setup.sh there, as the site's test runner does.
func copyCase(t *testing.T, dir string) string {
	t.Helper()
	ws := t.TempDir()
	if err := os.CopyFS(ws, os.DirFS(dir)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ws, "setup.sh")); err == nil {
		if err := os.MkdirAll(filepath.Join(ws, "opm", "docs", "site", "start"), 0o755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(t.Context(), "sh", filepath.Join(ws, "setup.sh"))
		cmd.Env = append(os.Environ(), "WS="+ws)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup.sh: %v\n%s", err, out)
		}
	}
	return ws
}

// roots lists the case's docs/site trees in the runner's repository order,
// then any other repository in name order.
func roots(t *testing.T, ws string) []string {
	t.Helper()
	found, err := filepath.Glob(filepath.Join(ws, "*", "docs", "site"))
	if err != nil {
		t.Fatal(err)
	}
	rank := map[string]int{}
	for i, r := range repoOrder {
		rank[r] = i
	}
	out := make([]string, 0, len(found))
	for _, f := range found {
		out = append(out, strings.TrimPrefix(f, ws+"/"))
	}
	sort.Slice(out, func(i, j int) bool {
		ri, rj := repoRank(rank, out[i]), repoRank(rank, out[j])
		if ri != rj {
			return ri < rj
		}
		return out[i] < out[j]
	})
	return out
}

func repoRank(rank map[string]int, root string) int {
	repo := strings.SplitN(root, "/", 2)[0]
	if r, ok := rank[repo]; ok {
		return r
	}
	return len(rank)
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if sc.Text() != "" {
			out = append(out, sc.Text())
		}
	}
	return out
}

// A fence closes only on a run at least as long as the one that opened
// it: a three-backtick line inside a four-backtick fence is content, so
// the markup after it is not linted as prose (nor rendered as HTML).
func TestFenceClosesOnItsOwnLength(t *testing.T) {
	dir := t.TempDir()
	page := "---\ntitle: \"T\"\ndescription: \"D\"\ntype: reference\n---\n\n````yaml\nnote: |\n  ```\n  <img src=x>\n````\n\nAfter.\n\n```text\n<img src=y>\n```\n"
	if err := os.WriteFile(filepath.Join(dir, "p.md"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	vs, err := Lint(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 0 {
		t.Errorf("violations %v", vs)
	}
	page = strings.Replace(page, "  ```\n  <img src=x>\n````", "````\n<img src=x>\n````", 1)
	if err := os.WriteFile(filepath.Join(dir, "p.md"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	if vs, _ = Lint(dir, Options{}); len(vs) == 0 {
		t.Errorf("an image after a closed fence was not flagged")
	}
}
