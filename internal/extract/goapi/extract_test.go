package goapi

import (
	"bytes"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/doctext"
)

var update = flag.Bool("update", false, "rewrite the golden doc model")

// fixtureConfig is the fixture module's config.
func fixtureConfig() Config {
	w := 4
	return Config{
		Module: "./mod", Root: "./lib", Packages: []string{"./lib/..."},
		Section: "reference/go-api/", Title: "Go API", Description: "Every exported package.", Weight: &w,
	}
}

func extractFixture(t *testing.T, edit func(*Options)) (*Result, error) {
	t.Helper()
	o := Options{Source: "testdata", Config: fixtureConfig(), Version: "1.2.3", Policy: doctext.Strip}
	if edit != nil {
		edit(&o)
	}
	return Extract(o)
}

func TestGolden(t *testing.T) {
	r, err := extractFixture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Model.Encode()
	if err != nil {
		t.Fatal(err)
	}
	const golden = "testdata/go-api.golden.json"
	if *update {
		if err := os.WriteFile(golden, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run go test -run TestGolden -update)", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("the model differs from %s (run go test -run TestGolden -update and review the diff)", golden)
	}
}

func byImportPath(t *testing.T, m *Model, ip string) *Package {
	t.Helper()
	for i := range m.Packages {
		if m.Packages[i].ImportPath == ip {
			return &m.Packages[i]
		}
	}
	t.Fatalf("no package %s", ip)
	return nil
}

func TestSelection(t *testing.T) {
	r, err := extractFixture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range r.Model.Packages {
		got = append(got, p.ImportPath+" "+p.Page)
	}
	// No internal package, no command, no test or windows-only symbol.
	want := []string{
		"example.com/widgets/lib/gadget gadget",
		"example.com/widgets/lib/sub/deep sub-deep",
		"example.com/widgets/lib/widget widget",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("packages\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	w := byImportPath(t, r.Model, "example.com/widgets/lib/widget")
	for _, f := range w.Funcs {
		if f.Name == "WindowsOnly" || f.Name == "TestOnly" || f.Name == "New" {
			t.Errorf("package funcs list %s", f.Name)
		}
	}
	if want := []string{"mod/lib/widget/doc.go", "mod/lib/widget/widget.go"}; !slices.Equal(w.Files, want) {
		t.Errorf("files %v, want %v", w.Files, want)
	}
	if got := r.Files["reference/go-api/widget.md"]; !slices.Equal(got, w.Files) {
		t.Errorf("page inputs %v", got)
	}
}

func TestConstructorAndMethods(t *testing.T) {
	r, err := extractFixture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	w := byImportPath(t, r.Model, "example.com/widgets/lib/widget")
	var widget *Type
	for i := range w.Types {
		if w.Types[i].Name == "Widget" {
			widget = &w.Types[i]
		}
	}
	if widget == nil || len(widget.Funcs) != 1 || widget.Funcs[0].Name != "New" || widget.Funcs[0].Recv != nil {
		t.Fatalf("Widget's constructors %+v", widget)
	}
	if len(widget.Methods) != 2 || *widget.Methods[0].Recv != "*Widget" || *widget.Methods[1].Recv != "Widget" {
		t.Fatalf("Widget's methods %+v", widget.Methods)
	}
	// Hugo makes the second "widgetspin" unique; links follow it.
	if widget.Methods[0].Anchor != "widgetspin-1" {
		t.Errorf("Widget.Spin anchor %q", widget.Methods[0].Anchor)
	}
}

func TestDeclarations(t *testing.T) {
	r, err := extractFixture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	w := byImportPath(t, r.Model, "example.com/widgets/lib/widget")
	widget := w.Types[1]
	if widget.Name != "Widget" {
		t.Fatalf("types %+v", w.Types)
	}
	if !strings.Contains(widget.Decl, "// contains filtered or unexported fields") || strings.Contains(widget.Decl, "hidden") || strings.Contains(widget.Decl, "WHY") {
		t.Errorf("Widget decl:\n%s", widget.Decl)
	}
	for _, f := range w.Funcs {
		if f.Name == "Undocumented" && (f.Doc != "" || f.Decl != "func Undocumented()") {
			t.Errorf("an undocumented symbol keeps its declaration and an empty doc: %+v", f)
		}
	}
}

func TestPackageDoc(t *testing.T) {
	r, err := extractFixture(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	w := byImportPath(t, r.Model, "example.com/widgets/lib/widget")
	for _, want := range []string{
		"[gadget.Gadget](/docs/reference/go-api/gadget/#gadget)",
		"[context.Context](https://pkg.go.dev/context#Context)",
		"### Spinning",
		"```text\nw := widget.New()",
		`\<b\>raw HTML\</b\>`,
		`{\{\< figure \>}}`,
		"`cue`",
	} {
		if !strings.Contains(w.Doc, want) {
			t.Errorf("package doc lacks %q:\n%s", want, w.Doc)
		}
	}
	for _, bad := range []string{"0010", "WHY", "maintainers"} {
		if strings.Contains(w.Doc, bad) || strings.Contains(w.Synopsis, bad) {
			t.Errorf("package doc holds %q", bad)
		}
	}
	if w.Synopsis != "Package widget makes widgets." {
		t.Errorf("synopsis %q", w.Synopsis)
	}
}

func TestLinkPolicy(t *testing.T) {
	r, err := extractFixture(t, func(o *Options) { o.Policy = doctext.Link })
	if err != nil {
		t.Fatal(err)
	}
	w := byImportPath(t, r.Model, "example.com/widgets/lib/widget")
	if !strings.Contains(w.Doc, "Package widget makes widgets ([0010:D28](/enhancements/0010/decisions/)).") {
		t.Errorf("package doc:\n%s", w.Doc)
	}
	if w.Synopsis != "Package widget makes widgets." {
		t.Errorf("a synopsis is plain text under both policies: %q", w.Synopsis)
	}
}

func TestDeterministic(t *testing.T) {
	out := make([][]byte, 0, 2)
	for range 2 {
		r, err := extractFixture(t, nil)
		if err != nil {
			t.Fatal(err)
		}
		b, err := r.Model.Encode()
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, b)
	}
	if !bytes.Equal(out[0], out[1]) {
		t.Fatal("two extractions differ")
	}
}

func TestPatternMatchesNothing(t *testing.T) {
	nope := func(o *Options) { o.Config.Packages = []string{"./lib/...", "./lib/nope/..."} }
	_, err := extractFixture(t, nope)
	if err == nil || err.Error() != "packages ./lib/nope/... matches no package under ./mod" {
		t.Fatalf("err = %v", err)
	}
	var warned []string
	r, err := extractFixture(t, func(o *Options) {
		nope(o)
		o.Lenient = true
		o.Warn = func(s string) { warned = append(warned, s) }
	})
	if err != nil || len(r.Model.Packages) != 3 || len(warned) != 1 {
		t.Fatalf("a backfill warns: err %v, warnings %v", err, warned)
	}
	// An internal package matches nothing: it is never documented.
	if _, err := extractFixture(t, func(o *Options) { o.Config.Packages = []string{"./lib/internal/secret"} }); err == nil {
		t.Fatal("an internal package was selected")
	}
}

func TestRootRefusals(t *testing.T) {
	for _, c := range []struct {
		name string
		edit func(*Options)
		want string
	}{
		{"package at root", func(o *Options) { o.Config.Root = "./lib/gadget"; o.Config.Packages = []string{"./lib/gadget"} }, "is root itself"},
		{"package outside root", func(o *Options) { o.Config.Root = "./lib/sub" }, "lies outside root"},
		{"missing module", func(o *Options) { o.Config.Module = "./nope" }, "module ./nope does not exist"},
		{"missing root", func(o *Options) { o.Config.Root = "./nope" }, "root ./nope does not exist"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := extractFixture(t, c.edit)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
		})
	}
}

// copyFixture copies the fixture module to a temporary source tree.
func copyFixture(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir("testdata/mod", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel("testdata", p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func TestRefusesSymlinks(t *testing.T) {
	for _, c := range []struct {
		name string
		link func(src string) error
		want string
	}{
		{"a linked go file", func(src string) error {
			return os.Symlink("widget.go", filepath.Join(src, "mod", "lib", "widget", "linked.go"))
		}, "linked.go is not a regular file"},
		{"a linked package directory", func(src string) error {
			return os.Symlink("widget", filepath.Join(src, "mod", "lib", "linked"))
		}, "./lib/linked is a symbolic link"},
		{"a linked go.mod", func(src string) error {
			if err := os.Rename(filepath.Join(src, "mod", "go.mod"), filepath.Join(src, "go.mod.real")); err != nil {
				return err
			}
			return os.Symlink("../go.mod.real", filepath.Join(src, "mod", "go.mod"))
		}, "go.mod is not a regular file"},
		{"a linked module", func(src string) error {
			return os.Symlink("mod", filepath.Join(src, "linked"))
		}, "module ./linked is a symbolic link"},
	} {
		t.Run(c.name, func(t *testing.T) {
			src := copyFixture(t)
			if err := c.link(src); err != nil {
				t.Fatal(err)
			}
			_, err := extractFixture(t, func(o *Options) {
				o.Source = src
				if c.name == "a linked module" {
					o.Config.Module = "./linked"
				}
			})
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
		})
	}
}

func TestRefusesLinkScheme(t *testing.T) {
	gadgetDoc := func(t *testing.T, u string) (*Result, error) {
		src := copyFixture(t)
		doc := "// Package gadget holds widgets; see [the trap].\n//\n// [the trap]: " + u + "\npackage gadget\n"
		if err := os.WriteFile(filepath.Join(src, "mod", "lib", "gadget", "doc.go"), []byte(doc), 0o600); err != nil {
			t.Fatal(err)
		}
		return extractFixture(t, func(o *Options) { o.Source = src })
	}
	// The doc comment parser makes a link of a few schemes only; the ones
	// other than http and https are refused.
	for _, u := range []string{"file:///etc/passwd", "ftp://example.com/x", "gopher://example.com", "mailto://x@example.com"} {
		_, err := gadgetDoc(t, u)
		if err == nil || !strings.Contains(err.Error(), "a link is http, https or relative") || !strings.Contains(err.Error(), "./lib/gadget") {
			t.Errorf("%s: err = %v", u, err)
		}
	}
	// Any other scheme is no link to the parser: the text stays escaped.
	r, err := gadgetDoc(t, "javascript:alert(1)")
	if err != nil {
		t.Fatal(err)
	}
	g := byImportPath(t, r.Model, "example.com/widgets/lib/gadget")
	if strings.Contains(g.Doc, "](javascript") || !strings.Contains(g.Doc, `\[the trap\]`) {
		t.Errorf("doc:\n%s", g.Doc)
	}
}

func TestPatternRE(t *testing.T) {
	for _, c := range []struct {
		pattern, path string
		ok            bool
	}{
		{"./opm/...", "opm", true},
		{"./opm/...", "opm/kernel", true},
		{"./opm/...", "opm/helper/objectset", true},
		{"./opm/...", "opmx", false},
		{"./opm/kernel", "opm/kernel", true},
		{"./opm/kernel", "opm/kernel/x", false},
		{"./...", ".", true},
		{"./...", "a/b", true},
		{"./", ".", true},
		{"./opm/.../set", "opm/helper/set", true},
		{"./opm/k...", "opm/kernel", true},
	} {
		if got := patternRE(c.pattern).MatchString(c.path); got != c.ok {
			t.Errorf("%s matches %s: %v, want %v", c.pattern, c.path, got, c.ok)
		}
	}
}
