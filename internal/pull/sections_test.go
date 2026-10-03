package pull

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/verify"
)

const (
	sectionProject = "enhancements"
	sectionOwner   = "open-platform-model/enhancements"
)

// sectionTree writes an edge bundle of the enhancements section: the
// section page and one entry page linking it.
func sectionTree(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), sectionProject)
	files := map[string]string{
		"content/_index.md":      "---\ntitle: \"Enhancements\"\ndescription: \"The design record.\"\n---\n\n[0025](/enhancements/0025/)\n",
		"content/0025/_index.md": "---\ntitle: \"0025: Modules\"\ndescription: \"Modules.\"\nweight: 26\n---\n\n[All entries](/enhancements/).\n",
		"data/enhancements.json": "{\"schema\": \"docs.opmodel.dev/data/enhancements/v1\", \"repo\": \"" + sectionOwner + "\", \"entries\": []}\n",
	}
	for p, body := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	m := &bundle.Manifest{
		Schema: bundle.SchemaID, Project: sectionProject, Version: "edge",
		Source:  bundle.Source{Repo: sectionOwner, Commit: strings.Repeat("e", 40), Ref: "main"},
		Created: "2026-10-03T12:00:00Z", Tool: "0.5.0", Dialect: 1,
		Placement: bundle.Placement{Kind: "section", Root: "/enhancements/"},
		Pages: []bundle.Page{
			{Path: "0025/_index.md", Source: "0025/README.md", Generated: true},
			{Path: "_index.md", Source: "INDEX.md", Generated: true},
		},
		Data: []bundle.DataFile{{Path: "enhancements.json", Schema: "docs.opmodel.dev/data/enhancements/v1"}},
	}
	if err := bundle.Write(dir, m); err != nil {
		t.Fatal(err)
	}
	return dir
}

// sectionConfig is a pull config with the catalog tab and the section.
func (e *env) sectionConfig() string {
	e.t.Helper()
	return e.config(`registry: "` + e.registry + `"
tabs: "catalog-opm": {repo: "` + owner + `", root: "/catalogs/opm/", from: "4.4"}
sections: enhancements: {repo: "` + sectionOwner + `", root: "/enhancements/"}
`)
}

func sectionEntry(t *testing.T, l *Lock) Entry {
	t.Helper()
	for i := range l.Bundles {
		if l.Bundles[i].Project == sectionProject {
			return l.Bundles[i]
		}
	}
	t.Fatalf("no entry for %s in %+v", sectionProject, l.Bundles)
	return Entry{}
}

func TestPullSection(t *testing.T) {
	e := newEnv(t)
	e.publish("4.4.5", owner, nil)
	e.publishProject(sectionProject, sectionTree(t), "edge", sectionOwner)
	o := e.options(e.sectionConfig())
	l, err := Run(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	en := sectionEntry(t, l)
	if en.Root != "/enhancements/" || en.Segment != "edge" || en.Tag != "edge" || en.Dir != "enhancements/edge" || en.Signer == nil || en.Signer.Repository != "https://github.com/"+sectionOwner {
		t.Fatalf("entry %+v", en)
	}
	if _, err := os.Stat(filepath.Join(o.Out, sectionProject, "edge", "content", "0025", "_index.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(o.Out, sectionProject, "history.json")); !os.IsNotExist(err) {
		t.Fatal("a section got a history")
	}
	// The lock pulls the same section again, frozen.
	o.Frozen = o.Lock
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatalf("frozen: %v", err)
	}
}

func TestPullSectionRefusals(t *testing.T) {
	for _, c := range []struct {
		name  string
		setup func(e *env)
		want  []string
	}{
		{"no edge", func(e *env) {
			// The repository holds only a release, never an edge build.
			e.publishProject(sectionProject, tree(e.t, "1.0.0", func(m *bundle.Manifest) { m.Project = sectionProject }), "1.0.0", sectionOwner)
		}, []string{"enhancements has no edge build; the section would be empty"}},
		{"signed by another repository", func(e *env) {
			e.publishProject(sectionProject, sectionTree(e.t), "edge", owner)
		}, []string{"enhancements", "only repository allowed"}},
		{"a tab bundle under the section", func(e *env) {
			e.publishProject(sectionProject, tree(e.t, "edge", func(m *bundle.Manifest) { m.Project = sectionProject; m.Source.Ref = "main" }), "edge", sectionOwner)
		}, []string{"the bundle's placement is tab /catalogs/opm/, and the section's root is /enhancements/"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t)
			e.publish("4.4.5", owner, nil)
			c.setup(e)
			_, err := Run(context.Background(), e.options(e.sectionConfig()))
			if err == nil {
				t.Fatal("pulled")
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not name %q", err, w)
				}
			}
		})
	}
}

func TestPullSectionLocal(t *testing.T) {
	e := newEnv(t)
	o := e.options(e.config(`sections: enhancements: {repo: "` + sectionOwner + `", root: "/enhancements/"}
`))
	e.stop() // an all-local pull needs no registry
	o.Verifier = func() (*verify.Verifier, error) { return nil, errors.New("a local pull fetched the trusted root") }
	o.Locals = []Local{{sectionProject, "edge", sectionTree(t)}}
	l, err := Run(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if en := sectionEntry(t, l); !en.Local || en.Root != "/enhancements/" || en.Segment != "edge" {
		t.Fatalf("entry %+v", en)
	}
	o.Locals = []Local{{sectionProject, "1.0", sectionTree(t)}}
	if _, err := Run(context.Background(), o); !IsUsage(err) || !strings.Contains(err.Error(), "whose only segment is edge") {
		t.Fatalf("a minor of a section: %v", err)
	}
}

func TestPullSectionInTwoRoles(t *testing.T) {
	e := newEnv(t)
	cfg := e.config(`sections: enhancements: {repo: "` + sectionOwner + `", root: "/enhancements/"}
docs: enhancements: {repo: "` + sectionOwner + `"}
`)
	if _, err := Run(context.Background(), e.options(cfg)); !IsUsage(err) || !strings.Contains(err.Error(), "enhancements is both a section and a docs project") {
		t.Fatalf("err = %v", err)
	}
}

func TestPullSectionFrozenWithoutIt(t *testing.T) {
	e := newEnv(t)
	e.publish("4.4.5", owner, nil)
	e.publishProject(sectionProject, sectionTree(t), "edge", sectionOwner)
	o := e.options(e.sectionConfig())
	l, err := Run(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	kept := l.Bundles[:0]
	for _, en := range l.Bundles {
		if en.Project != sectionProject {
			kept = append(kept, en)
		}
	}
	l.Bundles = kept
	data, err := l.Encode()
	if err != nil {
		t.Fatal(err)
	}
	frozen := filepath.Join(e.work, "frozen.json")
	if err := os.WriteFile(frozen, data, 0o600); err != nil {
		t.Fatal(err)
	}
	o.Frozen = frozen
	if _, err := Run(context.Background(), o); !IsUsage(err) || !strings.Contains(err.Error(), "has no entry for the section enhancements") {
		t.Fatalf("err = %v", err)
	}
}
