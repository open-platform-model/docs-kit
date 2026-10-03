package cuedefs

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/doctext"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// LoadConfig reads the first cue-definitions source of a docs-kit.cue.
func loadConfig(t *testing.T, path string) Config {
	t.Helper()
	c, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range c.Projects() {
		for _, s := range c.Bundles[p].Sources {
			if s.Kind == "cue-definitions" {
				var cfg Config
				if err := s.Value.Decode(&cfg); err != nil {
					t.Fatal(err)
				}
				return cfg
			}
		}
	}
	t.Fatalf("%s has no cue-definitions source", path)
	return Config{}
}

func fixture(t *testing.T, edit func(*Config), o Options) (*Model, error) {
	t.Helper()
	cfg := loadConfig(t, "testdata/defs/config.cue")
	if edit != nil {
		edit(&cfg)
	}
	o.Root, o.Config = "testdata/defs", cfg
	if o.Version == "" {
		o.Version = "1.2.3"
	}
	return Extract(o)
}

func TestGolden(t *testing.T) {
	m, err := fixture(t, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	const golden = "testdata/cue-definitions.golden.json"
	if *update {
		if err := os.WriteFile(golden, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("data/cue-definitions.json differs from %s; run go test -run TestGolden -update and review the diff", golden)
	}
	back, err := Decode(got)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := back.Encode()
	if !bytes.Equal(got, again) {
		t.Fatal("decode and encode do not round-trip")
	}
}

func find(t *testing.T, m *Model, name string) *Definition {
	t.Helper()
	for i := range m.Definitions {
		if m.Definitions[i].Name == name {
			return &m.Definitions[i]
		}
	}
	t.Fatalf("no definition %s", name)
	return nil
}

func TestModel(t *testing.T) {
	m, err := fixture(t, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if m.ModulePath != "example.com/defs@v1" || m.Section != "reference/definitions/" || m.Weight != 2 {
		t.Errorf("model head %+v", m)
	}
	names := make([]string, 0, len(m.Definitions))
	for _, d := range m.Definitions {
		names = append(names, d.Name)
	}
	if want := []string{"#Module", "#Component", "#Widget", "#NameType", "#LabelsType", "#Mode"}; !slices.Equal(names, want) {
		t.Errorf("definitions %v, want %v", names, want)
	}
	if len(m.Pages) != 2 || m.Pages[1].Weight != 2 {
		t.Errorf("pages %+v", m.Pages)
	}
	if len(m.Excluded) != 3 || m.Excluded[0].Name != "#ComponentMap" || m.Excluded[0].Reason != "map shorthand" {
		t.Errorf("excluded %+v", m.Excluded)
	}
}

func TestUses(t *testing.T) {
	m, err := fixture(t, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	mod := find(t, m, "#Module")
	// Uses follow the excluded map shorthand through to #Component.
	if want := []string{"#Component", "#LabelsType", "#NameType"}; !slices.Equal(mod.Uses, want) {
		t.Errorf("#Module uses %v, want %v", mod.Uses, want)
	}
	if c := find(t, m, "#Component"); !slices.Equal(c.UsedBy, []string{"#Module"}) || !slices.Equal(c.Embeds, []string{"#Widget"}) {
		t.Errorf("#Component usedBy %v embeds %v", c.UsedBy, c.Embeds)
	}
}

func TestEntry(t *testing.T) {
	m, err := fixture(t, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	mod := find(t, m, "#Module")
	if mod.File != "src/module.cue" || mod.Anchor != "module" || mod.Page != "modules" {
		t.Errorf("#Module where %q %q %q", mod.File, mod.Anchor, mod.Page)
	}
	if mod.Summary != "The unit an author publishes." {
		t.Errorf("summary %q", mod.Summary)
	}
	for _, bad := range []string{"_count", "0010:", "WHY", "SPEC.md", "separated from the field"} {
		if strings.Contains(mod.CUE, bad) {
			t.Errorf("spec block keeps %q:\n%s", bad, mod.CUE)
		}
	}
	if strings.Contains(strings.Join(mod.Notes, "\n"), "WHY") {
		t.Errorf("notes keep a WHY line: %q", mod.Notes)
	}
}

func TestPinsSkipped(t *testing.T) {
	defs, err := loadDefs("testdata/defs", "testdata/defs/src", []string{"*_pins.cue"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := defs["#Pinned"]; ok {
		t.Fatal("#Pinned is in the model")
	}
	defs, _ = loadDefs("testdata/defs", "testdata/defs/src", nil)
	if _, ok := defs["#Pinned"]; !ok {
		t.Fatal("without skip, #Pinned is missing")
	}
}

func TestLinkPolicy(t *testing.T) {
	m, err := fixture(t, nil, Options{Policy: doctext.Link})
	if err != nil {
		t.Fatal(err)
	}
	mod := find(t, m, "#Module")
	if mod.Summary != "The unit an author publishes ([0010:D4](/enhancements/0010/decisions/))." {
		t.Errorf("summary %q", mod.Summary)
	}
	// A spec block's comment drops citations under both policies.
	if strings.Contains(mod.CUE, "0010:") {
		t.Errorf("spec block keeps a citation:\n%s", mod.CUE)
	}
}

func TestPlacement(t *testing.T) {
	for _, c := range []struct {
		name string
		edit func(*Config)
		want []string
	}{
		{"unplaced", func(c *Config) { delete(c.Exclude, "#Unused") },
			[]string{"#Unused (src/types.cue) is exported but neither placed in a page nor excluded in docs-kit.cue"}},
		{"twice", func(c *Config) { c.Pages[1].Definitions = append(c.Pages[1].Definitions, "#Widget") },
			[]string{`#Widget is placed on pages "modules" and "types"`}},
		{"unknown", func(c *Config) { c.Pages[0].Definitions = append(c.Pages[0].Definitions, "#Policy") },
			[]string{`#Policy is placed on page "modules", but the package declares no such definition`}},
		{"unknown excluded", func(c *Config) { c.Exclude["#Gone"] = "old" },
			[]string{"#Gone is excluded, but the package declares no such definition"}},
		{"anchors collide", func(c *Config) {
			delete(c.Exclude, "#MODE")
			c.Pages[1].Definitions = append(c.Pages[1].Definitions, "#MODE")
		}, []string{`#Mode and #MODE on page "types" share the anchor #mode`}},
		{"placed and excluded", func(c *Config) { c.Exclude["#Mode"] = "x" },
			[]string{`#Mode is placed on page "types" and also excluded`}},
		{"all at once", func(c *Config) {
			delete(c.Exclude, "#Unused")
			c.Pages[0].Definitions = append(c.Pages[0].Definitions, "#Policy")
		}, []string{"#Unused (src/types.cue)", "#Policy is placed"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := fixture(t, c.edit, Options{})
			var pe *PlacementError
			if !errors.As(err, &pe) {
				t.Fatalf("err = %v", err)
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error lacks %q:\n%v", w, err)
				}
			}
			if len(pe.Problems) != len(c.want) {
				t.Errorf("problems %q, want %d", pe.Problems, len(c.want))
			}
			// A backfill warns instead and builds.
			var warned []string
			m, err := fixture(t, c.edit, Options{Lenient: true, Warn: func(s string) { warned = append(warned, s) }})
			if err != nil {
				t.Fatalf("lenient: %v", err)
			}
			if len(warned) != len(pe.Problems) {
				t.Errorf("warnings %q, problems %q", warned, pe.Problems)
			}
			for _, d := range m.Definitions {
				if d.Name == "#Policy" || d.Name == "#Unused" {
					t.Errorf("%s is in the model", d.Name)
				}
			}
		})
	}
}

// A backfill drops a page left without a definition, and the weights
// follow the remaining pages.
func TestBackfillDropsEmptyPage(t *testing.T) {
	edit := func(c *Config) {
		c.Pages = append([]PageConfig{{File: "policies", Title: "Policies", Description: "d", Definitions: []string{"#Policy"}}}, c.Pages...)
	}
	m, err := fixture(t, edit, Options{Lenient: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Pages) != 2 || m.Pages[0].File != "modules" || m.Pages[0].Weight != 1 {
		t.Fatalf("pages %+v", m.Pages)
	}
}

func TestErrors(t *testing.T) {
	if _, err := fixture(t, func(c *Config) { c.Package = "./nothing" }, Options{}); err == nil || !strings.Contains(err.Error(), "holds no .cue file") {
		t.Errorf("missing package: %v", err)
	}
}

func TestText(t *testing.T) {
	for _, c := range []struct{ in, sum, rest string }{
		{"Semver 2.0", "Semver 2.0.", ""},
		{"A thing. data holds more.", "A thing.", "data holds more."},
		{"Uses e.g. a value. Then more.", "Uses e.g. a value.", "Then more."},
	} {
		sum, rest := firstSentence(c.in)
		if sum != c.sum || rest != c.rest {
			t.Errorf("firstSentence(%q) = %q, %q", c.in, sum, rest)
		}
	}
	for in, want := range map[[2]string]string{
		{"#Module", "#Module: The portable thing"}:           "The portable thing",
		{"#NameType", "NameType: RFC 1123 DNS label"}:        "RFC 1123 DNS label",
		{"#SnakeNameType", "SnakeNameType: snake_case name"}: "snake_case name",
		{"#IdentityPackage", "IdentityPackage is the shape"}: "#IdentityPackage is the shape",
	} {
		if got := stripNameLabel(in[0], in[1]); got != want {
			t.Errorf("stripNameLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

// A placed definition whose doc comment yields no summary (missing, or
// only rationale) fails a normal build; a backfill warns and leaves the
// summary empty.
func TestPlacementNoSummary(t *testing.T) {
	for name, doc := range map[string]string{
		"no doc comment": "",
		"only rationale": "// WHY: maintainers only.\n",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.CopyFS(root, os.DirFS("testdata/defs")); err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(root, "src", "module.cue")
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			b = bytes.Replace(b, []byte("// #Widget: shared fields every component embeds.\n"), []byte(doc), 1)
			if err := os.WriteFile(p, b, 0o600); err != nil {
				t.Fatal(err)
			}
			cfg := loadConfig(t, "testdata/defs/config.cue")
			_, err = Extract(Options{Root: root, Config: cfg, Version: "1.2.3"})
			if err == nil || !strings.Contains(err.Error(), "#Widget (src/module.cue) has no summary") {
				t.Fatalf("err = %v", err)
			}
			var warned []string
			m, err := Extract(Options{Root: root, Config: cfg, Version: "1.2.3", Lenient: true, Warn: func(s string) { warned = append(warned, s) }})
			if err != nil {
				t.Fatal(err)
			}
			if len(warned) != 1 || find(t, m, "#Widget").Summary != "" {
				t.Fatalf("warnings %q, summary %q", warned, find(t, m, "#Widget").Summary)
			}
		})
	}
}

// Uses count references to the top-level definition only: not a field
// label, a selector's field or a nested definition that shadows it; a
// quoted "#X" label declares a regular field.
func TestTopLevelReferences(t *testing.T) {
	root := t.TempDir()
	src := "package x\n\n" +
		"// #A is a struct.\n#A: {b: #B, c: {#C: int, d: #C}, e: y.#D, #D: int, f: #E}\n" +
		"// #B is an int.\n#B: int\n" +
		"// #C is an int.\n#C: int\n" +
		"// #D is an int.\n#D: int\n" +
		"\"#Q\": string\n" +
		"y: {#D: string}\n"
	if err := os.WriteFile(filepath.Join(root, "a.cue"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	// #E is declared in another file of the package.
	if err := os.WriteFile(filepath.Join(root, "b.cue"), []byte("package x\n\n// #E is a string.\n#E: string\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	defs, err := loadDefs(root, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := defs["#Q"]; ok {
		t.Error("a quoted label was taken as a definition")
	}
	if got := defs["#A"].refs; !slices.Equal(got, []string{"#B", "#E"}) {
		t.Errorf("#A refs %v, want [#B #E]", got)
	}
}

// A shape's kind span is long enough for a backtick in the kind.
func TestShapeCodeSpan(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.cue"), []byte("package x\n\n// #K pins an odd kind.\n#K: {kind: \"a`b\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	defs, err := loadDefs(root, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	d, err := definition(defs["#K"], defs, map[string]string{"#K": "p"}, nil, doctext.Strip)
	if err != nil {
		t.Fatal(err)
	}
	if want := "struct, closed, ``kind: \"a`b\"``"; d.Shape != want {
		t.Errorf("shape %q, want %q", d.Shape, want)
	}
}
