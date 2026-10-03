package cuedefs

import (
	"bytes"
	"errors"
	"flag"
	"os"
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
	if len(m.Excluded) != 2 || m.Excluded[0].Name != "#ComponentMap" || m.Excluded[0].Reason != "map shorthand" {
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
	if _, err := fixture(t, func(c *Config) { c.Skip = []string{"["} }, Options{}); err == nil || !strings.Contains(err.Error(), "not a valid glob") {
		t.Errorf("bad glob: %v", err)
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
