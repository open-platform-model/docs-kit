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
			if b.Placement.Root != "/catalogs/opm/" || b.Version.Prefix != "opm-v" || len(b.Sources) != 2 || b.Sources[1].Dir != "docs/catalogs/opm" {
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
	if len(b.Placement.Owns) != 2 || b.Placement.Owns[1] != "reference/operator-resources.md" || len(b.Sources[0].Exclude) != 1 {
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

// siteConfig is opmodel.dev's planned bundles.cue for phase 2, with a tab.
const siteConfig = `tabs: "catalog-opm": {repo: "open-platform-model/catalog_opm", root: "/catalogs/opm/", from: "4.4"}
docs: {
	cli:            {repo: "open-platform-model/cli"}
	core:           {repo: "open-platform-model/core"}
	library:        {repo: "open-platform-model/library"}
	"opm-operator": {repo: "open-platform-model/opm-operator"}
}
versions: "v1.0": {
	anchor: {project: "cli", tag: "1.0"}
	pinned: ["library", "core", "opm-operator"]
}
`

func TestLoadPullSiteVersions(t *testing.T) {
	p, err := LoadPull(write(t, "bundles.cue", siteConfig+`versions: "v0.10": {anchor: {project: "cli", tag: "edge"}, tags: core: "2.0.0-beta.1"}
versions: "v0.9": {anchor: {project: "cli", tag: "0"}}
`))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(p.SiteVersions(), ","); got != "v0.9,v0.10,v1.0" {
		t.Fatalf("site versions %s", got)
	}
	v := p.Versions["v1.0"]
	if v.Anchor != (Anchor{"cli", "1.0"}) || strings.Join(v.Pinned, ",") != "library,core,opm-operator" || len(v.Tags) != 0 || p.Docs["core"].Repo != "open-platform-model/core" {
		t.Fatalf("decoded %+v", v)
	}
	if p.Versions["v0.9"].Pinned == nil || v.Role("core") != "pinned" || v.Role("cli") != "anchor" || p.Versions["v0.10"].Role("core") != "tag" || v.Role("opm") != "" {
		t.Fatalf("roles or default pinned: %+v", p.Versions)
	}
	// docs without versions is the site's state before its switch.
	if _, err := LoadPull(write(t, "bundles.cue", `docs: cli: {repo: "open-platform-model/cli"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPullSiteVersionRefusals(t *testing.T) {
	for _, c := range []struct {
		name, body string
		want       []string
	}{
		{"undeclared project", strings.Replace(siteConfig, `core:           {repo: "open-platform-model/core"}`, "", 1), []string{`"v1.0"`, "core", "not in docs"}},
		{"a project twice", strings.Replace(siteConfig, `["library", "core", "opm-operator"]`, `["library", "core", "cli"]`, 1), []string{`"v1.0"`, "cli twice"}},
		{"pinned and tags", strings.Replace(siteConfig, `pinned: ["library", "core", "opm-operator"]`, `pinned: ["core"], tags: core: "2"`, 1), []string{`"v1.0"`, "core twice"}},
		{"tab and docs", siteConfig + `docs: "catalog-opm": {repo: "open-platform-model/catalog_opm"}` + "\n", []string{"catalog-opm", "both a tab and a docs project"}},
		{"bad site version", strings.Replace(siteConfig, `"v1.0"`, `"1.0"`, 1), []string{"versions"}},
		{"bad tag", strings.Replace(siteConfig, `tag: "1.0"`, `tag: "v1.0"`, 1), []string{"tag"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := LoadPull(write(t, "bundles.cue", c.body))
			if err == nil {
				t.Fatal("loaded")
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not name %q", err, w)
				}
			}
		})
	}
}
