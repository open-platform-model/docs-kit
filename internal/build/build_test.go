package build

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/gittest"
)

const fixtureConfig = `bundles: {
	"catalog-demo": {
		placement: {kind: "tab", root: "/catalogs/demo/"}
		version: {from: "tag", prefix: "demo-v"}
		sources: [
			{kind: "cue-catalog", module: "./demo"},
			{kind: "markdown", dir: "docs/catalogs/demo"},
		]
	}
}
`

const authoredLanding = `---
title: "The demo catalog contract"
description: "What a demo catalog promises."
---

Every member is listed in [the traits](/catalogs/demo/1/traits/).
`

// fixtureRepo is a repository holding the fixture catalog, a docs-kit.cue
// and an authored landing, tagged demo-v1.2.3.
func fixtureRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	t.Setenv("GITHUB_REPOSITORY", "")
	t.Setenv("GITHUB_REF_TYPE", "")
	r := gittest.New(t, "https://github.com/example/demo.git")
	r.CopyTree("../extract/cuecatalog/testdata/catalog", ".")
	r.Write(map[string]string{
		"docs-kit.cue":                 fixtureConfig,
		"docs/catalogs/demo/_index.md": authoredLanding,
		".gitignore":                   "/out/\n",
	})
	r.Commit("catalog")
	r.Git("tag", "demo-v1.2.3")
	return r
}

func treeBytes(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		out[rel], _ = os.ReadFile(p)
		return nil
	})
	return out
}

func releaseBuild(t *testing.T, r *gittest.Repo) map[string][]byte {
	t.Helper()
	out := filepath.Join(t.TempDir(), "out")
	if _, err := Run(context.Background(), Options{Source: r.Dir, Out: out, Release: "demo-v1.2.3", Tool: "0.1.0"}); err != nil {
		t.Fatal(err)
	}
	return treeBytes(t, filepath.Join(out, "catalog-demo"))
}

func TestReleaseBuildIsDeterministic(t *testing.T) {
	r := fixtureRepo(t)
	a, b := releaseBuild(t, r), releaseBuild(t, r)
	if len(a) != len(b) || len(a) == 0 {
		t.Fatalf("%d and %d files", len(a), len(b))
	}
	for p, body := range a {
		if !bytes.Equal(body, b[p]) {
			t.Errorf("%s differs between two builds", p)
		}
	}
}

func TestReleaseManifest(t *testing.T) {
	r := fixtureRepo(t)
	tree := releaseBuild(t, r)
	m, err := bundle.Parse("manifest.json", tree["manifest.json"])
	if err != nil {
		t.Fatal(err)
	}
	head := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	if m.Version != "1.2.3" || m.Revision != 0 || m.Source.Ref != "demo-v1.2.3" || m.Source.Commit != head ||
		m.Source.Repo != "example/demo" || m.Created != gittest.Date || m.Source.Dirty {
		t.Fatalf("manifest %+v", m)
	}
}

func TestReleasePageEntries(t *testing.T) {
	m, err := bundle.Parse("manifest.json", releaseBuild(t, fixtureRepo(t))["manifest.json"])
	if err != nil {
		t.Fatal(err)
	}
	pages := map[string]bundle.Page{}
	for _, p := range m.Pages {
		pages[p.Path] = p
	}
	if p := pages["_index.md"]; p.Generated || p.Source != "docs/catalogs/demo/_index.md" || p.Lastmod != gittest.Date {
		t.Errorf("landing entry %+v", p)
	}
	if p := pages["traits/backup.md"]; !p.Generated || p.Source != "demo/traits/v1beta1/backup.cue" || p.Lastmod != gittest.Date {
		t.Errorf("member entry %+v", p)
	}
}

func TestReleaseLanding(t *testing.T) {
	tree := releaseBuild(t, fixtureRepo(t))
	landing := string(tree[filepath.Join("content", "_index.md")])
	pinned := strings.Replace(authoredLanding, "/catalogs/demo/1/traits/", "/catalogs/demo/1.2/traits/", 1)
	if !strings.HasPrefix(landing, pinned+"\n## Catalog members") {
		t.Fatalf("landing:\n%s", landing)
	}
}

func TestEdgeBuildAndDirty(t *testing.T) {
	r := fixtureRepo(t)
	out := filepath.Join(r.Dir, "out")
	if _, err := Run(context.Background(), Options{Source: r.Dir, Out: out, Tool: "0.1.0"}); err != nil {
		t.Fatal(err)
	}
	m, err := bundle.Read(filepath.Join(out, "catalog-demo"))
	if err != nil {
		t.Fatal(err)
	}
	if m.Version != "edge" || m.Source.Ref != "main" || m.Source.Dirty {
		t.Fatalf("edge manifest %+v", m)
	}
	r.Write(map[string]string{"docs/catalogs/demo/_index.md": authoredLanding + "\nMore.\n"})
	if _, err := Run(context.Background(), Options{Source: r.Dir, Out: out, Tool: "0.1.0"}); err != nil {
		t.Fatal(err)
	}
	if m, _ = bundle.Read(filepath.Join(out, "catalog-demo")); !m.Source.Dirty {
		t.Fatal("a modified work tree is not recorded as dirty")
	}
}

func TestConfigFromOutside(t *testing.T) {
	r := fixtureRepo(t)
	r.Git("rm", "-q", "-r", "docs-kit.cue", "docs")
	p := filepath.Join(r.Dir, "demo", "catalog.cue")
	b, _ := os.ReadFile(p)
	r.Write(map[string]string{"demo/catalog.cue": strings.Replace(string(b), `"1.2.3"`, `"1.2.4"`, 1)})
	r.Commit("an old release had neither")
	r.Git("tag", "demo-v1.2.4")
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, ConfigFile), []byte(fixtureConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	out := filepath.Join(cwd, "out")
	if _, err := Run(context.Background(), Options{Source: r.Dir, Out: out, Release: "demo-v1.2.4", Tool: "0.1.0"}); err != nil {
		t.Fatal(err)
	}
	m, err := bundle.Read(filepath.Join(out, "catalog-demo"))
	if err != nil {
		t.Fatal(err)
	}
	if m.Version != "1.2.4" || m.Pages[0].Path != "_index.md" || !m.Pages[0].Generated {
		t.Fatalf("manifest %+v", m)
	}
}

func TestRefusals(t *testing.T) {
	cases := []struct {
		name  string
		edit  func(r *gittest.Repo)
		o     Options
		usage bool
		want  []string
	}{
		{name: "missing markdown dir", edit: func(r *gittest.Repo) { r.Git("rm", "-q", "-r", "docs"); r.Commit("rm") },
			want: []string{"docs/catalogs/demo", "docs-kit.cue"}},
		{name: "wrong prefix", o: Options{Release: "v1.2.3"}, usage: true, want: []string{"v1.2.3", `"demo-v"`}},
		{name: "not semver", o: Options{Release: "demo-v1.2"}, usage: true, want: []string{"demo-v1.2"}},
		{name: "tag and catalog version differ", edit: func(r *gittest.Repo) { r.Git("tag", "demo-v1.2.4") },
			o: Options{Release: "demo-v1.2.4"}, want: []string{"demo-v1.2.4", "1.2.4", `"1.2.3"`, "./demo"}},
		{name: "misspelled key", edit: func(r *gittest.Repo) {
			r.Write(map[string]string{"docs-kit.cue": strings.Replace(fixtureConfig, "placement:", "placment:", 1)})
		}, usage: true, want: []string{"placment", "docs-kit.cue"}},
		{name: "unknown project", o: Options{Projects: []string{"nope"}}, usage: true, want: []string{"nope", "catalog-demo"}},
		{name: "doc comment without description", edit: func(r *gittest.Repo) {
			p := filepath.Join(r.Dir, "demo", "resources", "v1", "queue.cue")
			b, _ := os.ReadFile(p)
			r.Write(map[string]string{"demo/resources/v1/queue.cue": strings.Replace(string(b), "// A message queue a component declares.", "// Something else.", 1)})
		}, want: []string{"example.com/catalogs/demo/resources/queue@v1", "demo/resources/v1/queue.cue", "A message queue a component declares.", "Something else."}},
		{name: "heading collision", edit: func(r *gittest.Repo) {
			r.Write(map[string]string{"docs/catalogs/demo/_index.md": authoredLanding + "\n## Catalog members\n"})
		}, want: []string{"docs/catalogs/demo/_index.md", "Catalog members"}},
		{name: "path written twice", edit: func(r *gittest.Repo) {
			r.Write(map[string]string{"docs/catalogs/demo/traits/backup.md": "---\ntitle: \"x\"\ndescription: \"x\"\ntype: reference\n---\n"})
		}, want: []string{"content/traits/backup.md", "markdown docs/catalogs/demo", "cue-catalog"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := fixtureRepo(t)
			if c.edit != nil {
				c.edit(r)
			}
			o := c.o
			o.Source, o.Out, o.Tool = r.Dir, filepath.Join(t.TempDir(), "out"), "0.1.0"
			o.Config = filepath.Join(r.Dir, ConfigFile)
			_, err := Run(context.Background(), o)
			if err == nil {
				t.Fatal("built")
			}
			if _, serr := os.Stat(filepath.Join(o.Out, "catalog-demo", bundle.ManifestFile)); serr == nil {
				t.Error("a refused build wrote manifest.json")
			}
			var ue *UsageError
			if errors.As(err, &ue) != c.usage {
				t.Errorf("usage error = %v, want %v: %v", !c.usage, c.usage, err)
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not name %q", err, w)
				}
			}
		})
	}
}

func TestLintViolationsFailTheBuild(t *testing.T) {
	r := fixtureRepo(t)
	r.Write(map[string]string{"docs/catalogs/demo/guide.md": "---\ntitle: \"x\"\ndescription: \"x\"\ntype: reference\n---\n\n[bad](/catalogs/demo/1.2/traits/nope/)\n"})
	_, err := Run(context.Background(), Options{Source: r.Dir, Out: filepath.Join(t.TempDir(), "out"), Release: "", Tool: "0.1.0"})
	var le *LintError
	if !errors.As(err, &le) || len(le.Violations) != 1 || !strings.Contains(le.Violations[0], "guide.md:7:") {
		t.Fatalf("err = %v %+v", err, le)
	}
}

func TestRevisionOptions(t *testing.T) {
	r := fixtureRepo(t)
	sha := strings.Repeat("a", 40)
	for _, c := range []struct {
		name string
		o    Options
		want string
	}{
		{"negative", Options{Release: "demo-v1.2.3", Revision: -1}, "a revision is 0 or more"},
		{"patches without revision", Options{Release: "demo-v1.2.3", Patches: []string{sha}}, "--patches needs --revision"},
		{"revision of edge", Options{Revision: 1, Patches: []string{sha}}, "needs --release"},
		{"revision without patches", Options{Release: "demo-v1.2.3", Revision: 1}, "needs --patches"},
		{"short patch", Options{Release: "demo-v1.2.3", Revision: 1, Patches: []string{"abc"}}, "not a full 40-hex"},
		{"patch twice", Options{Release: "demo-v1.2.3", Revision: 1, Patches: []string{sha, sha}}, "listed twice"},
	} {
		t.Run(c.name, func(t *testing.T) {
			c.o.Source, c.o.Out, c.o.Tool = r.Dir, t.TempDir(), "0.1.0"
			_, err := Run(context.Background(), c.o)
			var ue *UsageError
			if !errors.As(err, &ue) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want a usage error with %q", err, c.want)
			}
		})
	}
}
