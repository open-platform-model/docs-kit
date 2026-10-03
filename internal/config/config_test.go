package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
)

const validConfig = `bundles: {
	"catalog-opm": {
		placement: {kind: "tab", root: "/catalogs/opm/"}
		version: {from: "tag", prefix: "opm-v"}
		sources: [
			{kind: "cue-catalog", module: "./opm"},
			{kind: "markdown", dir: "docs/catalogs/opm"},
		]
	}
}
`

func write(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad(t *testing.T) {
	cases := []struct {
		name, body string
		wantErr    []string
	}{
		{name: "valid", body: validConfig},
		{name: "package clause ignored", body: "package docs\n\n" + validConfig},
		{name: "misspelled key", body: strings.Replace(validConfig, "placement:", "placment:", 1),
			wantErr: []string{"docs-kit.cue:", "placment", "not allowed"}},
		{name: "bad project name", body: strings.Replace(validConfig, `"catalog-opm"`, `"Catalog_OPM"`, 1),
			wantErr: []string{"docs-kit.cue:", "Catalog_OPM"}},
		{name: "tab root outside catalogs", body: strings.Replace(validConfig, `"/catalogs/opm/"`, `"/docs/opm/"`, 1),
			wantErr: []string{"docs-kit.cue:", "root"}},
		{name: "no sources", body: strings.Replace(validConfig, "sources: [\n\t\t\t{kind: \"cue-catalog\", module: \"./opm\"},\n\t\t\t{kind: \"markdown\", dir: \"docs/catalogs/opm\"},\n\t\t]", "sources: []", 1),
			wantErr: []string{"sources"}},
		{name: "unknown source kind", body: strings.Replace(validConfig, `kind: "markdown"`, `kind: "javadoc"`, 1),
			wantErr: []string{"docs-kit.cue:", `"javadoc"`, "cue-catalog, markdown"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Load(write(t, "docs-kit.cue", c.body))
			if len(c.wantErr) > 0 {
				if err == nil {
					t.Fatalf("loaded %+v, want an error", got)
				}
				for _, w := range c.wantErr {
					if !strings.Contains(err.Error(), w) {
						t.Errorf("error %q does not name %q", err, w)
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			b := got.Bundles["catalog-opm"]
			if b.Placement.Root != "/catalogs/opm/" || b.Version.Prefix != "opm-v" || len(b.Sources) != 2 || b.Sources[1].Markdown.Dir != "docs/catalogs/opm" {
				t.Fatalf("decoded %+v", b)
			}
			if m, err := b.Sources[0].Value.LookupPath(cue.ParsePath("module")).String(); err != nil || m != "./opm" {
				t.Fatalf("the cue-catalog entry's own value: %q, %v", m, err)
			}
			if p := got.Projects(); len(p) != 1 || p[0] != "catalog-opm" {
				t.Fatalf("projects %v", p)
			}
		})
	}
}

func TestLoadPull(t *testing.T) {
	body := `tabs: {
	"catalog-opm": {repo: "open-platform-model/catalog_opm", root: "/catalogs/opm/", from: "4.4"}
}
`
	p, err := LoadPull(write(t, "bundles.cue", body))
	if err != nil {
		t.Fatal(err)
	}
	tab := p.Tabs["catalog-opm"]
	if p.Registry != "ghcr.io/open-platform-model/docs" || !tab.Edge || tab.From != "4.4" ||
		p.Signer.Issuer != "https://token.actions.githubusercontent.com" ||
		len(p.Signer.Refs) != 1 || p.Signer.Refs[0] != "refs/tags/v[0-9]*" || !strings.HasPrefix(p.Digest, "sha256:") {
		t.Fatalf("decoded %+v", p)
	}
	_, err = LoadPull(write(t, "bundles.cue", `tabs: "catalog-opm": {root: "/catalogs/opm/", from: "4.4"}`))
	if err == nil || !strings.Contains(err.Error(), "repo") || !strings.Contains(err.Error(), "catalog-opm") {
		t.Fatalf("missing repo: err = %v", err)
	}
}

const docsConfig = `bundles: {
	"cli": {
		placement: {kind: "docs", root: "/docs/", owns: [OWNS]}
		version: {from: "tag", prefix: "v"}
		sources: [
			{kind: "markdown", dir: "docs/site", exclude: [EXCLUDE]},
		]
	}
}
`

func docs(owns, exclude string) string {
	return strings.NewReplacer("OWNS", owns, "EXCLUDE", exclude).Replace(docsConfig)
}

func TestLoadDocsPlacement(t *testing.T) {
	c, err := Load(write(t, "docs-kit.cue", docs(`"reference/cli/", "reference/operator-resources.md"`, `"reference/cli/"`)))
	if err != nil {
		t.Fatal(err)
	}
	b := c.Bundles["cli"]
	if len(b.Placement.Owns) != 2 || b.Placement.Owns[1] != "reference/operator-resources.md" || len(b.Sources[0].Markdown.Exclude) != 1 {
		t.Fatalf("decoded %+v", b)
	}
	for _, x := range []struct{ name, body string }{
		{"nested owns", docs(`"reference/", "reference/cli/"`, `"x/"`)},
		{"an owned page under an owned directory", docs(`"reference/", "reference/cli.md"`, `"x/"`)},
		{"owned path form", docs(`"Reference/"`, `"x/"`)},
		{"owned page without .md", docs(`"reference/cli"`, `"x/"`)},
		{"bad glob", docs(`"reference/"`, `"reference/[cli"`)},
		{"tab root on a docs bundle", strings.Replace(docs(`"reference/"`, `"x/"`), `root: "/docs/"`, `root: "/catalogs/cli/"`, 1)},
		{"a catalog in a docs bundle", strings.Replace(docs(`"reference/"`, `"x/"`), `{kind: "markdown"`, `{kind: "cue-catalog", module: "./opm"},
			{kind: "markdown"`, 1)},
		{"citations on markdown", strings.Replace(docs(`"reference/"`, `"x/"`), `{kind: "markdown",`, `{kind: "markdown", citations: "strip",`, 1)},
		{"linked citations on a catalog", strings.Replace(validConfig, `module: "./opm"}`, `module: "./opm", citations: "link"}`, 1)},
		{"empty pins projects", strings.Replace(docs(`"reference/"`, `"x/"`), `version: {`, `pins: {command: ["cat", "pins.json"], projects: []}
		version: {`, 1)},
		{"empty pins command", strings.Replace(docs(`"reference/"`, `"x/"`), `version: {`, `pins: {command: [], projects: ["core"]}
		version: {`, 1)},
		{"a crd source in a tab bundle", strings.Replace(validConfig, `{kind: "cue-catalog", module: "./opm"}`, `{kind: "cue-catalog", module: "./opm"},
			{kind: "crd", dir: "./c", page: "r.md", title: "T", description: "D"}`, 1)},
		{"owns on a tab bundle", strings.Replace(validConfig, `root: "/catalogs/opm/"}`, `root: "/catalogs/opm/", owns: ["x/"]}`, 1)},
	} {
		t.Run(x.name, func(t *testing.T) {
			if _, err := Load(write(t, "docs-kit.cue", x.body)); err == nil {
				t.Fatal("loaded")
			}
		})
	}
	_, err = Load(write(t, "docs-kit.cue", docs(`"reference/", "reference/cli/"`, `"x/"`)))
	if err == nil || !strings.Contains(err.Error(), "reference/ and reference/cli/ nest") || !strings.Contains(err.Error(), "docs-kit.cue") {
		t.Fatalf("nested owns: %v", err)
	}
}

const defsConfig = `bundles: core: {
	placement: {kind: KIND, root: ROOT, owns: ["reference/definitions/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{
		kind: "cue-definitions", package: "./src", skip: [SKIP]
		section: "reference/definitions/", title: "Definitions", description: "d"
		pages: [{file: "components", title: "Components", description: "d", definitions: ["#Component"]}]
		exclude: {"#ComponentMap": "map shorthand"}
	}, {
		kind: "markdown", dir: "docs/site", exclude: ["reference/definitions/"]
	}]
}
`

func defs(kind, root, skip string) string {
	return strings.NewReplacer("KIND", kind, "ROOT", root, "SKIP", skip).Replace(defsConfig)
}

// A cue-definitions exclude (a map) and a markdown exclude (a list) load
// side by side: each kind decodes its own options.
func TestLoadPerKindOptions(t *testing.T) {
	c, err := Load(write(t, "docs-kit.cue", defs(`"docs"`, `"/docs/"`, `"*_pins.cue"`)))
	if err != nil {
		t.Fatal(err)
	}
	srcs := c.Bundles["core"].Sources
	if srcs[0].Markdown != nil || srcs[1].Markdown == nil || srcs[1].Markdown.Exclude[0] != "reference/definitions/" {
		t.Fatalf("sources %+v", srcs)
	}
	var opts struct {
		Exclude map[string]string `json:"exclude"`
	}
	if err := srcs[0].Value.Decode(&opts); err != nil || opts.Exclude["#ComponentMap"] != "map shorthand" {
		t.Fatalf("cue-definitions exclude %v: %v", opts.Exclude, err)
	}
	for _, x := range []struct{ name, body, want string }{
		{"in a tab bundle", strings.Replace(defs(`"tab"`, `"/catalogs/core/"`, `"*_pins.cue"`), `, owns: ["reference/definitions/"]`, "", 1), `placement kind "docs"`},
		{"bad skip glob", defs(`"docs"`, `"/docs/"`, `"[x"`), `sources[0].skip: "[x" is not a glob`},
	} {
		if _, err := Load(write(t, "docs-kit.cue", x.body)); err == nil || !strings.Contains(err.Error(), x.want) {
			t.Errorf("%s: err = %v, want %q", x.name, err, x.want)
		}
	}
}
