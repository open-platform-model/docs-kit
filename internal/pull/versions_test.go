package pull

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/verify"
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

// offline makes a pull fail if it touches the registry or the trusted root.
func (e *env) offline(o *Options) {
	e.stop()
	o.Verifier = func() (*verify.Verifier, error) { return nil, errors.New("a local pull fetched the trusted root") }
}

func TestSiteVersionAllLocal(t *testing.T) {
	e := newEnv(t)
	o := e.options(e.docsConfig(v10))
	e.offline(&o)
	pins := map[string]string{"core": "2.0.0-beta.1", "opm-operator": "1.0.0-beta.4"}
	o.Locals = []Local{
		{"cli", "v1.0", docsTree(t, cliSpec("edge", pins))}, // an author's tree: edge is not checked against the anchor's tag
		{"core", "v1.0", docsTree(t, coreSpec("2.0.0-beta.1"))},
		{"opm-operator", "v1.0", docsTree(t, operatorSpec())},
	}
	l, err := Run(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range l.Docs {
		if !d.Local || d.Digest != "" || d.Signer != nil || d.Tag != "" {
			t.Fatalf("entry %+v", d)
		}
	}
	data, _ := os.ReadFile(o.Lock)
	if !strings.Contains(string(data), "\"role\": \"anchor\",\n      \"local\": true,") || !strings.Contains(string(data), "\"dir\": \"_versions/v1.0/opm-operator\"") {
		t.Fatalf("lock:\n%s", data)
	}
}

func TestSiteVersionLocalRefusals(t *testing.T) {
	e := newEnv(t)
	o := e.options(e.docsConfig(v10))
	e.offline(&o)
	pins := map[string]string{"core": "2.0.0-beta.1", "opm-operator": "1.0.0-beta.4"}
	o.Locals = []Local{
		{"cli", "v1.0", docsTree(t, cliSpec("edge", pins))},
		{"core", "v1.0", docsTree(t, coreSpec("2.0.0-beta.1"))},
		{"opm-operator", "v1.0", docsTree(t, operatorSpec())},
	}
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	// A pinned tree off its pin is a usage error, and so is a project the
	// version does not pull.
	o.Locals[1] = Local{"core", "v1.0", docsTree(t, coreSpec("2.0.0-beta.2"))}
	if _, err := Run(context.Background(), o); !IsUsage(err) || !strings.Contains(err.Error(), "the tree is core 2.0.0-beta.2, and cli edge pins core 2.0.0-beta.1") {
		t.Fatalf("off its pin: %v", err)
	}
	o.Locals[1] = Local{"library", "v1.0", docsTree(t, docsSpec{project: "library", version: "1.0.0", pages: []string{"embedding/kernel.md"}})}
	if _, err := Run(context.Background(), o); !IsUsage(err) || !strings.Contains(err.Error(), "v1.0 does not pull library") {
		t.Fatalf("not in the version: %v", err)
	}
	// An anchor without a pin for a pinned project fails the pull, not the
	// invocation.
	o.Locals = []Local{
		{"cli", "v1.0", docsTree(t, cliSpec("1.0.0", map[string]string{"core": "2.0.0-beta.1"}))},
		{"core", "v1.0", docsTree(t, coreSpec("2.0.0-beta.1"))},
		{"opm-operator", "v1.0", docsTree(t, operatorSpec())},
	}
	if _, err := Run(context.Background(), o); err == nil || IsUsage(err) || !strings.Contains(err.Error(), "cli 1.0.0 (local) pins no version of opm-operator") {
		t.Fatalf("pin missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(o.Out, VersionsDir, "v1.0", "cli", "manifest.json")); err != nil {
		t.Fatal("the refused pull removed the previous version")
	}
}

// An author previews the cli reference: the local anchor's pins choose the
// pulled projects.
func TestSiteVersionLocalAnchorPinsPulledProjects(t *testing.T) {
	e := newEnv(t)
	e.publishV10()
	o := e.options(e.docsConfig(v10))
	o.Locals = []Local{{"cli", "v1.0", docsTree(t, cliSpec("1.0.0-beta.7", map[string]string{"core": "2.0.0-beta.2", "opm-operator": "1.0.0-beta.4"}))}}
	l, err := Run(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if got := docsOrder(l); got != "v1.0/anchor/cli@=1.0.0-beta.7,v1.0/pinned/core@2.0.0-beta.2=2.0.0-beta.2,v1.0/pinned/opm-operator@1.0.0-beta.4=1.0.0-beta.4" {
		t.Fatalf("docs %s", got)
	}
	if !l.Docs[0].Local || l.Docs[1].Local || l.Docs[0].Pins["core"] != "2.0.0-beta.2" {
		t.Fatalf("entries %+v", l.Docs)
	}
}

func TestSiteVersionOverlaps(t *testing.T) {
	pins := map[string]string{"core": "2.0.0"}
	cfg := `versions: "v1.0": {anchor: {project: "cli", tag: "1.0"}, pinned: ["core"]}`
	core := func(owns []string, pages ...string) docsSpec {
		return docsSpec{project: "core", version: "2.0.0", owns: owns, pages: pages}
	}
	for _, c := range []struct {
		name string
		cli  func(*docsSpec) // edits cli 1.0.0, which owns reference/cli/
		core docsSpec
		want string
	}{
		{"a page under another bundle's owned directory", nil, core([]string{"reference/definitions/"}, "reference/definitions/_index.md", "reference/cli/_index.md"),
			"v1.0: core 2.0.0 has reference/cli/_index.md, under reference/cli/, which cli 1.0.0 owns"},
		{"a page under another bundle's owned page", func(s *docsSpec) { s.pages = append(s.pages, "reference/operator-resources/extra.md") },
			core([]string{"reference/operator-resources.md"}, "reference/operator-resources.md"),
			"v1.0: cli 1.0.0 has reference/operator-resources/extra.md, under reference/operator-resources.md, which core 2.0.0 owns"},
		{"a page in two bundles, owned by neither", func(s *docsSpec) { s.pages = append(s.pages, "start/_index.md") }, core(nil, "start/_index.md"),
			"v1.0: start/_index.md is in both cli 1.0.0 and core 2.0.0"},
		{"cli.md and cli/_index.md serve one URL", func(s *docsSpec) { s.pages = append(s.pages, "guides/cli.md") }, core(nil, "guides/cli/_index.md"),
			"v1.0: guides/cli.md in cli 1.0.0 and guides/cli/_index.md in core 2.0.0 serve one URL, /docs/guides/cli/"},
		{"module.md and module/_index.md serve one URL", func(s *docsSpec) { s.pages = append(s.pages, "guides/module.md") }, core(nil, "guides/module/_index.md"),
			"v1.0: guides/module.md in cli 1.0.0 and guides/module/_index.md in core 2.0.0 serve one URL"},
		{"owned paths that nest, the pinned project outer", nil, core([]string{"reference/"}, "reference/definitions.md"),
			"v1.0: cli 1.0.0 owns reference/cli/ and core 2.0.0 owns reference/"},
		{"owned paths that nest, the anchor outer", func(s *docsSpec) { s.owns = []string{"reference/"} }, core([]string{"reference/definitions/"}, "reference/definitions/_index.md"),
			"v1.0: cli 1.0.0 owns reference/ and core 2.0.0 owns reference/definitions/"},
		{"an owned page and an owned directory at one URL", nil, core([]string{"reference/cli.md"}, "reference/cli.md"),
			"v1.0: cli 1.0.0 owns reference/cli/ and core 2.0.0 owns reference/cli.md"},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t)
			o := e.options(e.docsConfig(cfg))
			e.offline(&o)
			cli := cliSpec("1.0.0", pins)
			if c.cli != nil {
				c.cli(&cli)
			}
			o.Locals = []Local{{"cli", "v1.0", docsTree(t, cli)}, {"core", "v1.0", docsTree(t, c.core)}}
			_, err := Run(context.Background(), o)
			if err == nil || IsUsage(err) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
			if left, _ := os.ReadDir(filepath.Join(o.Out, VersionsDir)); len(left) != 0 {
				t.Fatalf("a refused version left %v", left)
			}
		})
	}
}

// The page dialect refuses index.md in a bundle, so a pull never reaches
// the overlap check with one; the check still reads it as its section's URL.
func TestCheckOverlapIndexPage(t *testing.T) {
	bundleOf := func(project string, pages ...string) *docsBundle {
		m := &bundle.Manifest{Project: project, Version: "1.0.0"}
		for _, p := range pages {
			m.Pages = append(m.Pages, bundle.Page{Path: p})
		}
		return &docsBundle{project: project, m: m, what: project + " 1.0.0"}
	}
	err := checkOverlap("v1.0", []*docsBundle{bundleOf("cli", "guides/module.md"), bundleOf("core", "guides/module/index.md")})
	if err == nil || !strings.Contains(err.Error(), "v1.0: guides/module.md in cli 1.0.0 and guides/module/index.md in core 1.0.0 serve one URL, /docs/guides/module/") {
		t.Fatalf("err = %v", err)
	}
}

func TestPageURL(t *testing.T) {
	for in, want := range map[string]string{
		"_index.md": "", "index.md": "", "start.md": "start", "start/_index.md": "start", "start/index.md": "start",
		"reference/cli/opm-module.md": "reference/cli/opm-module", "reference/indexes.md": "reference/indexes",
	} {
		if got := pageURL(in); got != want {
			t.Errorf("pageURL(%s) = %q, want %q", in, got, want)
		}
	}
}

func TestSiteVersionFrozenOffline(t *testing.T) {
	e := newEnv(t)
	e.publishV10()
	o := e.options(e.docsConfig(v10))
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	saved := filepath.Join(e.work, "saved.lock")
	want, _ := os.ReadFile(o.Lock)
	_ = os.WriteFile(saved, want, 0o600)
	// A tag moved since: the frozen pull ignores it.
	e.publishDocs(cliSpec("1.0.0-beta.7", map[string]string{"core": "2.0.0-beta.2", "opm-operator": "1.0.0-beta.4"}))
	_ = os.RemoveAll(o.Out)
	o.Frozen, o.Offline = saved, true
	e.stop()
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(o.Lock); !bytes.Equal(got, want) {
		t.Fatalf("offline lock differs:\n%s\nwant:\n%s", got, want)
	}
}

// A frozen lock may only name what the config pulls, in the role it pulls
// it.
func TestSiteVersionFrozenOutsideTheConfig(t *testing.T) {
	e := newEnv(t)
	e.publishV10()
	o := e.options(e.docsConfig(v10))
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	good, _ := os.ReadFile(o.Lock)
	for _, c := range []struct{ name, from, to, want string }{
		{"a changed role", `"role": "pinned",
      "tag": "2.0.0-beta.1"`, `"role": "tag",
      "tag": "2.0.0-beta.1"`, "v1.0 core is locked as tag, and the config pulls it as pinned"},
		{"another site version", `"site": "v1.0",
      "project": "core"`, `"site": "v0.9",
      "project": "core"`, "v0.9 is not a site version"},
		{"another repository", `"repository": "` + e.registry + `/core"`, `"repository": "ghcr.io/evil/core"`, "names the repository ghcr.io/evil/core"},
		{"a pinned entry off the anchor's pin", `"version": "2.0.0-beta.1",
      "revision"`, `"version": "2.0.0-beta.2",
      "revision"`, "v1.0 core is locked at version 2.0.0-beta.2, tag 2.0.0-beta.1, and the locked anchor pins 2.0.0-beta.1"},
		{"another anchor tag", `"tag": "1.0"`, `"tag": "1"`, "v1.0 cli is locked at tag 1, and the config resolves it at 1.0"},
	} {
		t.Run(c.name, func(t *testing.T) {
			bad := strings.Replace(string(good), c.from, c.to, 1)
			if bad == string(good) {
				t.Fatal("replacement did not apply")
			}
			p := filepath.Join(t.TempDir(), "lock.json")
			_ = os.WriteFile(p, []byte(bad), 0o600)
			oo := o
			oo.Frozen = p
			if _, err := Run(context.Background(), oo); !IsUsage(err) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestParseLocalSiteVersion(t *testing.T) {
	if l, err := ParseLocal("cli@v1.0=out/cli"); err != nil || !l.Site() || l.Segment != "v1.0" || l.Dir != "out/cli" {
		t.Fatalf("%+v %v", l, err)
	}
	if l, _ := ParseLocal("catalog-opm@4.4=x"); l.Site() {
		t.Fatal("a tab segment parsed as a site version")
	}
	for _, bad := range []string{"cli@v1=dir", "cli@v1.0.0=dir", "cli@V1.0=dir", "cli@v01.0=dir"} {
		if _, err := ParseLocal(bad); err == nil {
			t.Errorf("%s parsed", bad)
		}
	}
}

// A registry that denies an anonymous pull, as GHCR does for a package
// that does not exist yet, reads as a pinned release without a bundle.
func TestSiteVersionDeniedPinReadsAsMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors":[{"code":"DENIED","message":"requested access to the resource is denied"}]}`))
	}))
	defer srv.Close()
	e := newEnv(t)
	e.registry = strings.TrimPrefix(srv.URL, "http://") + "/docs"
	o := e.options(e.docsConfig(`versions: "v1.0": {anchor: {project: "cli", tag: "1.0"}, pinned: ["core"]}`))
	o.Locals = []Local{{"cli", "v1.0", docsTree(t, cliSpec("1.0.0-beta.6", map[string]string{"core": "2.0.0-beta.1"}))}}
	_, err := Run(context.Background(), o)
	if err == nil || !strings.Contains(err.Error(), "v1.0: cli 1.0.0-beta.6 pins core 2.0.0-beta.1, and "+e.registry+"/core has no bundle for it") {
		t.Fatalf("err = %v", err)
	}
}
