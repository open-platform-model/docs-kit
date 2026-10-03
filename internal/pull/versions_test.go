package pull

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
)

// docsSpec describes a docs bundle tree for a test.
type docsSpec struct {
	project, version string
	revision         int
	owns, pages      []string // pages under content/
	pins             map[string]string
	edit             func(*bundle.Manifest)
}

// docsTree writes a docs bundle directory.
func docsTree(t *testing.T, s docsSpec) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), s.project)
	ref := "v" + s.version
	if s.version == "edge" {
		ref = "main"
	}
	m := &bundle.Manifest{
		Schema: bundle.SchemaID, Project: s.project, Version: s.version, Revision: s.revision,
		Source:  bundle.Source{Repo: "open-platform-model/" + s.project, Commit: cmt, Ref: ref},
		Created: "2026-09-30T12:00:00Z", Tool: "0.1.0", Dialect: 1,
		Placement: bundle.Placement{Kind: "docs", Root: "/docs/", Owns: s.owns},
		Data:      []bundle.DataFile{}, Pins: s.pins,
	}
	for _, p := range s.pages {
		f := filepath.Join(dir, bundle.ContentDir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(f), 0o750); err != nil {
			t.Fatal(err)
		}
		kind := "type: reference\n"
		if filepath.Base(p) == "_index.md" {
			kind = "" // a section overview declares no type
		}
		body := "---\ntitle: \"" + s.project + " " + p + "\"\ndescription: \"A page.\"\n" + kind + "---\n\nBody of " + s.project + " " + s.version + ".\n"
		if err := os.WriteFile(f, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		m.Pages = append(m.Pages, bundle.Page{Path: p, Generated: true})
	}
	if s.edit != nil {
		s.edit(m)
	}
	if err := bundle.Write(dir, m); err != nil {
		t.Fatal(err)
	}
	return dir
}

func cliSpec(version string, pins map[string]string) docsSpec {
	return docsSpec{project: "cli", version: version, owns: []string{"reference/cli/"}, pages: []string{"reference/cli/_index.md", "reference/cli/opm-module.md"}, pins: pins}
}

func coreSpec(version string) docsSpec {
	return docsSpec{project: "core", version: version, owns: []string{"reference/definitions/"}, pages: []string{"reference/definitions/_index.md", "concepts/module.md"}}
}

func operatorSpec() docsSpec {
	return docsSpec{project: "opm-operator", version: "1.0.0-beta.4", owns: []string{"reference/operator-resources.md"}, pages: []string{"reference/operator-resources.md"}}
}

// publishDocs pushes, signs as its own repository and promotes a docs
// bundle.
func (e *env) publishDocs(s docsSpec) {
	e.t.Helper()
	e.publishProject(s.project, docsTree(e.t, s), s.version, "open-platform-model/"+s.project)
}

// docsConfig is a pull config with the four phase-2 docs projects and the
// given versions block.
func (e *env) docsConfig(versions string) string {
	return e.config(`registry: "` + e.registry + `"
docs: {
	cli:            {repo: "open-platform-model/cli"}
	core:           {repo: "open-platform-model/core"}
	library:        {repo: "open-platform-model/library"}
	"opm-operator": {repo: "open-platform-model/opm-operator"}
}
` + versions)
}

const v10 = `versions: "v1.0": {
	anchor: {project: "cli", tag: "1.0"}
	pinned: ["core", "opm-operator"]
}
`

func docsOrder(l *Lock) string {
	s := make([]string, 0, len(l.Docs))
	for i := range l.Docs {
		d := &l.Docs[i]
		s = append(s, d.Site+"/"+d.Role+"/"+d.Project+"@"+d.Tag+"="+d.Version)
	}
	return strings.Join(s, ",")
}

func readPage(t *testing.T, out, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// publishV10 publishes cli 1.0.0-beta.6 and the releases it pins, plus a
// newer core that nothing pins.
func (e *env) publishV10() {
	e.t.Helper()
	e.publishDocs(coreSpec("2.0.0-beta.1"))
	e.publishDocs(coreSpec("2.0.0-beta.2"))
	e.publishDocs(operatorSpec())
	e.publishDocs(cliSpec("1.0.0-beta.6", map[string]string{"core": "2.0.0-beta.1", "opm-operator": "1.0.0-beta.4", "library": "1.0.0-beta.1"}))
}

func TestSiteVersionFromPins(t *testing.T) {
	e := newEnv(t)
	e.publishV10()
	o := e.options(e.docsConfig(v10))
	l, err := Run(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if got := docsOrder(l); got != "v1.0/anchor/cli@1.0=1.0.0-beta.6,v1.0/pinned/core@2.0.0-beta.1=2.0.0-beta.1,v1.0/pinned/opm-operator@1.0.0-beta.4=1.0.0-beta.4" {
		t.Fatalf("docs %s", got)
	}
	a := l.Docs[0]
	if a.Pins["core"] != "2.0.0-beta.1" || a.Dir != "_versions/v1.0/cli" || a.Signer == nil || a.Signer.Repository != "https://github.com/open-platform-model/cli" || l.Docs[1].Pins != nil {
		t.Fatalf("entries %+v", l.Docs)
	}
	if !strings.Contains(readPage(t, o.Out, "_versions/v1.0/core/content/concepts/module.md"), "core 2.0.0-beta.1.") {
		t.Fatal("core is not the pinned release")
	}
	first, _ := os.ReadFile(o.Lock)
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if again, _ := os.ReadFile(o.Lock); !bytes.Equal(first, again) {
		t.Fatalf("two pulls of one resolution differ:\n%s\n%s", first, again)
	}
}

// A docs revision of a pinned release follows through its release tag.
func TestSiteVersionFollowsADocsRevision(t *testing.T) {
	e := newEnv(t)
	e.publishV10()
	o := e.options(e.docsConfig(v10))
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	e.publishDocs(docsSpec{project: "core", version: "2.0.0-beta.1", revision: 1, owns: []string{"reference/definitions/"},
		pages: []string{"reference/definitions/_index.md", "concepts/module.md", "concepts/bundle.md"}})
	l, err := Run(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if c := l.Docs[1]; c.Project != "core" || c.Version != "2.0.0-beta.1" || c.Revision != 1 || c.Tag != "2.0.0-beta.1" {
		t.Fatalf("core %+v", c)
	}
	if _, err := os.Stat(filepath.Join(o.Out, VersionsDir, "v1.0", "core", "content", "concepts", "bundle.md")); err != nil {
		t.Fatal("the revision's new page is missing")
	}
}

func TestSiteVersionOwnTagsAndTabs(t *testing.T) {
	e := newEnv(t)
	e.publish("4.4.5", owner, nil)
	e.publishDocs(cliSpec("1.0.0-beta.6", nil))
	e.publishDocs(coreSpec("2.0.0-beta.1"))
	o := e.options(e.config(`registry: "` + e.registry + `"
tabs: "catalog-opm": {repo: "` + owner + `", root: "/catalogs/opm/", from: "4.4", edge: false}
docs: {
	cli:  {repo: "open-platform-model/cli"}
	core: {repo: "open-platform-model/core"}
}
versions: "v1.0": {anchor: {project: "cli", tag: "1"}, tags: core: "2.0.0-beta.1"}
`))
	// Left over from an older config: a site version and a stray directory.
	_ = os.MkdirAll(filepath.Join(o.Out, VersionsDir, "v0.9", "cli"), 0o750)
	_ = os.MkdirAll(filepath.Join(o.Out, VersionsDir, "v1.0", "library"), 0o750)
	l, err := Run(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if got := docsOrder(l); got != "v1.0/anchor/cli@1=1.0.0-beta.6,v1.0/tag/core@2.0.0-beta.1=2.0.0-beta.1" || segments(l) != "4.4" {
		t.Fatalf("docs %s, tabs %s", got, segments(l))
	}
	left, _ := os.ReadDir(filepath.Join(o.Out, VersionsDir))
	inV1, _ := os.ReadDir(filepath.Join(o.Out, VersionsDir, "v1.0"))
	if len(left) != 1 || len(inV1) != 2 {
		t.Fatalf("_versions holds %v, v1.0 holds %v", left, inV1)
	}
	if _, err := os.Stat(filepath.Join(o.Out, project, "4.4")); err != nil {
		t.Fatal("the tab is gone")
	}
	// A config without versions removes them all.
	o.Config = e.config("")
	l, err = Run(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(o.Lock)
	if _, err := os.Stat(filepath.Join(o.Out, VersionsDir)); !os.IsNotExist(err) || len(l.Docs) != 0 || strings.Contains(string(data), `"docs"`) {
		t.Fatalf("versions left behind: %v\n%s", err, data)
	}
}

func TestSiteVersionRefusals(t *testing.T) {
	cases := []struct {
		name  string
		setup func(e *env)
		cfg   string
		want  []string
	}{
		{"a pinned project without a pin", func(e *env) {
			e.publishDocs(cliSpec("1.0.0-beta.6", map[string]string{"opm-operator": "1.0.0-beta.4"}))
			e.publishDocs(operatorSpec())
		}, v10, []string{"v1.0", "cli 1.0.0-beta.6", "sha256:", "pins no version of core"}},
		{"a pinned release without a bundle", func(e *env) {
			e.publishDocs(cliSpec("1.0.0-beta.6", map[string]string{"core": "2.0.0-beta.1", "opm-operator": "1.0.0-beta.5"}))
			e.publishDocs(coreSpec("2.0.0-beta.1"))
			e.publishDocs(operatorSpec())
		}, v10, []string{"v1.0", "cli 1.0.0-beta.6", "pins opm-operator 1.0.0-beta.5", "has no bundle", "publish it"}},
		{"a pinned project never published", func(e *env) {
			e.publishDocs(cliSpec("1.0.0-beta.6", map[string]string{"core": "2.0.0-beta.1", "opm-operator": "1.0.0-beta.4"}))
			e.publishDocs(coreSpec("2.0.0-beta.1"))
		}, v10, []string{"pins opm-operator 1.0.0-beta.4", "has no bundle"}},
		{"an anchor without its tag", func(e *env) {
			e.publishDocs(cliSpec("0.9.0", nil))
		}, v10, []string{"v1.0", "/cli has no tag 1.0"}},
		{"an anchor outside its line", func(e *env) {
			e.publishDocs(cliSpec("1.0.0-beta.6", nil))
			e.publishDocs(cliSpec("1.1.0", nil))
			e.retag("cli", "1.1", "1.0")
		}, `versions: "v1.0": anchor: {project: "cli", tag: "1.0"}`, []string{"v1.0 cli 1.0", "tag 1.0 names a build of 1.1.0.0, which is not in 1.0"}},
		{"a tab placement", func(e *env) {
			e.publishDocs(cliSpec("1.0.0", nil))
			e.publishDocs(docsSpec{project: "core", version: "2.0.0", pages: []string{"_index.md"}, edit: func(m *bundle.Manifest) {
				m.Placement = bundle.Placement{Kind: "tab", Root: "/catalogs/core/"}
			}})
		}, `versions: "v1.0": {anchor: {project: "cli", tag: "1.0"}, tags: core: "2"}`, []string{"v1.0 core 2", "placement is tab /catalogs/core/", "only docs bundles"}},
		{"signed by another repository", func(e *env) {
			e.publishProject("cli", docsTree(e.t, cliSpec("1.0.0", nil)), "1.0.0", "open-platform-model/core")
		}, `versions: "v1.0": anchor: {project: "cli", tag: "1.0"}`, []string{"v1.0 cli 1.0", "only repository allowed"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t)
			c.setup(e)
			_, err := Run(context.Background(), e.options(e.docsConfig(c.cfg)))
			if err == nil || IsUsage(err) {
				t.Fatalf("err = %v, want an execution error", err)
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not name %q", err, w)
				}
			}
		})
	}
}

// A refused site version keeps the previous one, whole, and the lock.
func TestRefusedSiteVersionKeepsThePreviousOne(t *testing.T) {
	e := newEnv(t)
	e.publishV10()
	o := e.options(e.docsConfig(v10))
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	lockBefore, _ := os.ReadFile(o.Lock)
	cliBefore := readPage(t, o.Out, "_versions/v1.0/cli/content/reference/cli/opm-module.md")
	// The next cli pins a core release that has no bundle: the anchor
	// itself is fine, and is not swapped in.
	e.publishDocs(cliSpec("1.0.0-beta.7", map[string]string{"core": "2.0.0-beta.3", "opm-operator": "1.0.0-beta.4"}))
	if _, err := Run(context.Background(), o); err == nil {
		t.Fatal("pulled")
	}
	if got := readPage(t, o.Out, "_versions/v1.0/cli/content/reference/cli/opm-module.md"); got != cliBefore {
		t.Fatalf("the previous cli changed:\n%s", got)
	}
	if lockAfter, _ := os.ReadFile(o.Lock); !bytes.Equal(lockBefore, lockAfter) {
		t.Fatal("the lock changed")
	}
	if left, _ := filepath.Glob(filepath.Join(o.Out, VersionsDir, ".incoming-*")); len(left) != 0 {
		t.Fatalf("left %v", left)
	}
}

// retag points tag at the build another tag names, by hand.
func (e *env) retag(project, from, to string) {
	e.t.Helper()
	ctx := context.Background()
	repo, _ := e.client.Repository(e.registry + "/" + project)
	d, err := repo.Resolve(ctx, from)
	if err != nil {
		e.t.Fatal(err)
	}
	_, raw, err := repo.Manifest(ctx, d)
	if err != nil {
		e.t.Fatal(err)
	}
	if err := repo.Tag(ctx, d, raw, to); err != nil {
		e.t.Fatal(err)
	}
}
