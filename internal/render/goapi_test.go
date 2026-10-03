package render

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/dialect"
	"github.com/open-platform-model/docs-kit/internal/extract/goapi"
)

var goAPITarget = Target{Kind: KindDocs, Root: "/docs/", Segment: "1.2", Version: "1.2.3", Repo: "example/widgets", Commit: commit}

// renderGoAPI renders a go-api data file, as written.
func renderGoAPI(t *testing.T, file string) []Page {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	r, err := For(goapi.SchemaID)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := r.Render(data, goAPITarget)
	if err != nil {
		t.Fatal(err)
	}
	return pages
}

// checkGolden compares pages with a golden directory (rewritten under
// -update) and lints them as a docs bundle owning owns.
func checkGolden(t *testing.T, pages []Page, dir string, owns []string) {
	t.Helper()
	if *update {
		_ = os.RemoveAll(dir)
		for _, p := range pages {
			full := filepath.Join(dir, p.Path)
			_ = os.MkdirAll(filepath.Dir(full), 0o755)
			if err := os.WriteFile(full, []byte(p.Body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	names := make([]string, 0, len(pages))
	for _, p := range pages {
		names = append(names, p.Path)
		want, err := os.ReadFile(filepath.Join(dir, p.Path))
		if err != nil {
			t.Fatalf("%s: %v (run go test ./internal/render -run %s -update)", p.Path, err, t.Name())
		}
		if string(want) != p.Body {
			t.Errorf("%s differs from its golden page", p.Path)
		}
	}
	var golden []string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(dir, p)
			golden = append(golden, filepath.ToSlash(rel))
		}
		return nil
	})
	slices.Sort(golden)
	slices.Sort(names)
	if !slices.Equal(golden, names) {
		t.Errorf("pages %v, golden %v", names, golden)
	}
	vs, err := dialect.Lint(dir, dialect.Options{Mode: dialect.Bundle, Bundle: dialect.BundleInfo{
		Kind: KindDocs, Root: "/docs/", Segment: "1.2", Owns: owns, Pages: names,
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		t.Errorf("golden page breaks the dialect: %s", v)
	}
}

func TestGoAPIGolden(t *testing.T) {
	pages := renderGoAPI(t, "../extract/goapi/testdata/go-api.golden.json")
	checkGolden(t, pages, filepath.Join("testdata", "golden", "go-api"), []string{"reference/go-api/"})
	index := pages[0]
	if index.Path != "reference/go-api/_index.md" || !index.Completable || index.Heading != "## Packages" || !strings.HasPrefix(index.Tail, "## Packages\n\n") {
		t.Fatalf("index %+v", index)
	}
}

// The anchors the extractor assigned, and its doc links use, are the ones
// Hugo gives the headings the renderer writes.
func TestGoAPIAnchors(t *testing.T) {
	data, err := os.ReadFile("../extract/goapi/testdata/go-api.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	m, err := goapi.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	checkAnchors(t, m)
}

func checkAnchors(t *testing.T, m *goapi.Model) {
	t.Helper()
	pages, err := GoAPI(m, goAPITarget)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]string{}
	for _, p := range pages {
		byPath[p.Path] = p.Body
	}
	reLink := regexp.MustCompile(`\]\((/docs/[^)#]*)(?:#([^)]*))?\)`)
	anchorsOf := map[string]map[string]bool{}
	for i := range m.Packages {
		p := &m.Packages[i]
		body := byPath[m.Section+p.Page+".md"]
		var a goapi.Anchors
		at := map[string]string{} // heading text -> anchor, first occurrence
		all := map[string]bool{}
		for _, h := range goapi.Headings(body) {
			id := a.Next(goapi.HeadingAnchor(h))
			all[id] = true
			if _, ok := at[h]; !ok {
				at[h] = id
			}
		}
		anchorsOf["/docs/"+m.Section+p.Page+"/"] = all
		for _, s := range goapi.PageSections(p) {
			for _, e := range s.Entries {
				if !all[e.Anchor] {
					t.Errorf("%s: %s has anchor %q, which no heading of the page gets", p.Page, e.Key, e.Anchor)
				}
			}
		}
	}
	for _, body := range byPath {
		for _, l := range reLink.FindAllStringSubmatch(body, -1) {
			if l[2] == "" {
				continue
			}
			if as, ok := anchorsOf[l[1]]; ok && !as[l[2]] {
				t.Errorf("link %s#%s names no heading of that page", l[1], l[2])
			}
		}
	}
}

func TestGoAPIRefusals(t *testing.T) {
	m := &goapi.Model{Schema: goapi.SchemaID, ModulePath: "example.com/m", Section: "reference/go-api/", Title: "T", Description: "D",
		Packages: []goapi.Package{{ImportPath: "example.com/m/a", Name: "a", Page: "a", Consts: []goapi.Value{{Names: []string{"X"}, Anchor: "x", Decl: `const X = "{{< x >}}"`}}}}}
	if _, err := GoAPI(m, goAPITarget); err == nil || !strings.Contains(err.Error(), "opens a Hugo shortcode") {
		t.Errorf("a shortcode in a declaration: %v", err)
	}
	m.Packages[0].Consts = nil
	if _, err := GoAPI(m, Target{Kind: KindTab, Root: "/catalogs/x/", Segment: "1.0"}); err == nil {
		t.Error("rendered into a tab bundle")
	}
	// A synopsis is escaped on the index and quoted in front matter.
	m.Packages[0].Synopsis = `Package a reads <b> and {{ x }}, "quoted".`
	pages, err := GoAPI(m, goAPITarget)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pages[0].Body, `- [a](/docs/reference/go-api/a/): Package a reads \<b\> and {\{ x }}, "quoted".`) {
		t.Errorf("index:\n%s", pages[0].Body)
	}
	if !strings.Contains(pages[1].Body, `description: "Package a reads <b> and {{ x }}, \"quoted\"."`) {
		t.Errorf("page:\n%s", pages[1].Body)
	}
	// Front matter is not escaped, so a shortcode there is refused.
	m.Packages[0].Synopsis = "Package a reads {{< x >}}."
	if _, err := GoAPI(m, goAPITarget); err == nil || !strings.Contains(err.Error(), "opens a Hugo shortcode") {
		t.Errorf("a shortcode in a synopsis: %v", err)
	}
}
