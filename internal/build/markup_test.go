package build

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
)

const authoredGuide = "---\ntitle: \"Guide\"\ndescription: \"A guide.\"\ntype: how-to\n---\n\n"

// An authored page keeps a writer's note; generated pages get every rule.
func TestMarkupCheckEveryPage(t *testing.T) {
	r := fixtureRepo(t)
	r.Write(map[string]string{"docs/catalogs/demo/guide.md": authoredGuide +
		"<!-- Check against: src/traits.cue -->\n\n| a | b <!-- c --> |\n|---|---|\n| d | e |\n"})
	r.Commit("guide")
	out := filepath.Join(t.TempDir(), "out")
	res, err := Run(context.Background(), Options{Source: r.Dir, Out: out, Tool: "0.1.0"})
	if err != nil || len(res) != 1 {
		t.Fatalf("an authored page with writer's notes: %v", err)
	}
	dir := res[0].Dir
	if vs, err := Lint(dir); err != nil || len(vs) != 0 {
		t.Fatalf("lint: %v %v", vs, err)
	}
	// The same note in a generated page is refused.
	gen := generatedPage(t, dir)
	b, err := os.ReadFile(gen)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gen, append(b, "\n<!-- note -->\n\n<details open ontoggle=alert(1)>\n"...), 0o600); err != nil {
		t.Fatal(err)
	}
	vs, err := Lint(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 2 || !strings.Contains(vs[0], "an HTML block") || !strings.Contains(vs[1], "an HTML block") {
		t.Fatalf("violations %v, want the comment and the element in %s", vs, gen)
	}
}

// generatedPage is the file of a bundle's first generated page.
func generatedPage(t *testing.T, dir string) string {
	t.Helper()
	m, err := bundle.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range m.Pages {
		if p.Generated {
			return filepath.Join(dir, bundle.ContentDir, filepath.FromSlash(p.Path))
		}
	}
	t.Fatal("no generated page")
	return ""
}

// Raw HTML other than a comment fails the build of an authored page.
func TestMarkupCheckAuthoredHTML(t *testing.T) {
	r := fixtureRepo(t)
	r.Write(map[string]string{"docs/catalogs/demo/guide.md": authoredGuide + "Some <b>bold</b> text.\n\n## Install {.hx:fixed}\n"})
	r.Commit("guide")
	_, err := Run(context.Background(), Options{Source: r.Dir, Out: filepath.Join(t.TempDir(), "out"), Tool: "0.1.0"})
	// <b> and </b> are two raw HTML nodes.
	var le *LintError
	if !errors.As(err, &le) || len(le.Violations) != 3 ||
		!strings.Contains(le.Violations[0], "guide.md:7: raw HTML other than an HTML comment") ||
		!strings.Contains(le.Violations[2], "guide.md:9: a heading attribute block") {
		t.Fatalf("err = %v %+v", err, le)
	}
}

// cue-definitions prose is Markdown as written, so a doc comment that
// writes a heading attribute block fails the build, naming the page and
// the line.
func TestMarkupCheckDefinitionsHeadingAttributes(t *testing.T) {
	r := defsRepo(t)
	b, err := os.ReadFile(filepath.Join(r.Dir, "src", "types.cue"))
	if err != nil {
		t.Fatal(err)
	}
	src := strings.Replace(string(b), "// #Mode is either fast or safe.\n",
		"// #Mode is either fast or safe.\n//\n// ## Choosing {.hx:fixed}\n//\n// Pick fast.\n", 1)
	r.Write(map[string]string{"src/types.cue": src})
	r.Commit("heading")
	_, err = Run(context.Background(), Options{Source: r.Dir, Out: filepath.Join(t.TempDir(), "out"), Tool: "0.1.0"})
	var le *LintError
	if !errors.As(err, &le) || len(le.Violations) != 1 || !strings.Contains(le.Violations[0], "types.md:") ||
		!strings.Contains(le.Violations[0], "a heading attribute block") {
		t.Fatalf("err = %v %+v", err, le)
	}
}
