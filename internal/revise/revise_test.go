package revise

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/build"
	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/gittest"
	"github.com/open-platform-model/docs-kit/internal/oci"
	"github.com/open-platform-model/docs-kit/internal/ocitest"
	"github.com/open-platform-model/docs-kit/internal/publish"
	"github.com/open-platform-model/docs-kit/internal/verify"
	"github.com/open-platform-model/docs-kit/internal/verify/sigtest"
)

const (
	project  = "catalog-demo"
	tag      = "demo-v1.2.3"
	backup   = "demo/traits/v1alpha1/backup.cue"
	landing  = "docs/catalogs/demo/_index.md"
	fixDateA = "2026-10-01T09:00:00Z"
	fixDateB = "2026-10-02T09:00:00Z"
)

const docsKit = `bundles: "catalog-demo": {
	placement: {kind: "tab", root: "/catalogs/demo/"}
	version: {from: "tag", prefix: "demo-v"}
	sources: [
		{kind: "cue-catalog", module: "./demo"},
		{kind: "markdown", dir: "docs/catalogs/demo"},
	]
}
`

const landingPage = `---
title: "The demo catalog contract"
description: "What a demo catalog promises."
---

Every member is listed below.
`

type env struct {
	t        *testing.T
	r        *gittest.Repo
	registry string
	client   *oci.Client
	repo     *oci.Repo
	auth     *sigtest.Authority
	release  string // the tag's commit
	tool     string // the opm-docs version revise runs as
}

const owner = "example/demo"

// newEnv makes a repository tagged demo-v1.2.3 with origin/main at HEAD,
// and an empty registry.
func newEnv(t *testing.T) *env {
	t.Helper()
	return newEnvWith(t, docsKit)
}

// newEnvWith is newEnv with cfg as the repository's docs-kit.cue.
func newEnvWith(t *testing.T, cfg string) *env {
	t.Helper()
	t.Setenv("GITHUB_REPOSITORY", "")
	t.Setenv("GITHUB_REF_TYPE", "")
	r := gittest.New(t, "https://github.com/example/demo.git")
	r.Write(map[string]string{"README.md": "# Demo\n"})
	r.Commit("start")
	r.CopyTree("../extract/cuecatalog/testdata/catalog", ".")
	r.Write(map[string]string{"docs-kit.cue": cfg, landing: landingPage, ".gitignore": "/out/\n"})
	r.Commit("catalog")
	r.Git("tag", tag)
	host := ocitest.Registry(t)
	c := oci.New(oci.Options{Anonymous: true, PlainHTTP: true})
	repo, err := c.Repository(host + "/docs/" + project)
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, r: r, registry: host + "/docs", client: c, repo: repo, auth: sigtest.New(t), release: r.Head(), tool: "0.2.0"}
	e.syncMain()
	return e
}

func (e *env) syncMain() { e.r.Git("update-ref", "refs/remotes/origin/main", "HEAD") }

// fix edits files on main and commits them at date.
func (e *env) fix(date string, edit map[string][2]string) string {
	e.t.Helper()
	for p, oldNew := range edit {
		b, err := os.ReadFile(filepath.Join(e.r.Dir, p))
		if err != nil {
			e.t.Fatal(err)
		}
		if !strings.Contains(string(b), oldNew[0]) {
			e.t.Fatalf("%s has no %q", p, oldNew[0])
		}
		e.r.Write(map[string]string{p: strings.Replace(string(b), oldNew[0], oldNew[1], 1)})
	}
	sha := e.r.CommitAt(date, "fix")
	e.syncMain()
	return sha
}

// push pushes a built bundle directory, as the workflow does after a build.
func (e *env) push(dir string) *publish.PushResult {
	e.t.Helper()
	res, err := publish.Push(context.Background(), publish.PushOptions{Dir: dir, Registry: e.registry, Client: e.client})
	if err != nil {
		e.t.Fatal(err)
	}
	return res
}

// sign signs a pushed digest as the publish workflow of owner's main.
func (e *env) sign(digest string) {
	e.t.Helper()
	d, err := e.repo.Resolve(context.Background(), digest)
	if err != nil {
		e.t.Fatal(err)
	}
	e.auth.Sign(e.t, e.repo.Graph(), d, sigtest.Publisher(owner))
}

// promote verifies and promotes a signed digest, as the workflow does.
func (e *env) promote(digest string) {
	e.t.Helper()
	if _, err := publish.Promote(context.Background(), publish.PromoteOptions{Project: project, Digest: digest, Registry: e.registry,
		Client: e.client, Verifier: e.auth.Verifier(e.t), Policy: sigtest.Policy(owner)}); err != nil {
		e.t.Fatal(err)
	}
}

// publishBuilt pushes, signs and promotes a built directory.
func (e *env) publishBuilt(dir string) {
	e.t.Helper()
	d := e.push(dir).Digest
	e.sign(d)
	e.promote(d)
}

// tagAs points tag at a pushed digest by hand: a promote that stopped
// half-way.
func (e *env) tagAs(digest, tag string) {
	e.t.Helper()
	ctx := context.Background()
	d, err := e.repo.Resolve(ctx, digest)
	if err != nil {
		e.t.Fatal(err)
	}
	_, raw, err := e.repo.Manifest(ctx, d)
	if err != nil {
		e.t.Fatal(err)
	}
	if err := e.repo.Tag(ctx, d, raw, tag); err != nil {
		e.t.Fatal(err)
	}
}

// publishRelease builds revision 0 from the tag and pushes it.
func (e *env) publishRelease() {
	e.t.Helper()
	wt := e.r.Dir + "-release"
	e.r.Git("worktree", "add", "-q", "--detach", wt, tag)
	e.t.Cleanup(func() { _ = os.RemoveAll(wt) })
	out := filepath.Join(e.t.TempDir(), "out")
	if _, err := build.Run(context.Background(), build.Options{Source: wt, Out: out, Release: tag, Tool: "0.1.0"}); err != nil {
		e.t.Fatal(err)
	}
	e.publishBuilt(filepath.Join(out, project))
}

func (e *env) revise(fix string) (*Result, string, error) {
	e.t.Helper()
	out := filepath.Join(e.t.TempDir(), "out")
	res, err := Run(context.Background(), Options{
		Repo: e.r.Dir, Project: project, Tag: tag, Fix: fix, Out: out,
		Registry: e.registry, Tool: e.tool, Client: e.client,
		Verifier: func() (*verify.Verifier, error) { return e.auth.Verifier(e.t), nil }, Policy: sigtest.Policy(owner),
	})
	return res, filepath.Join(out, project), err
}

func (e *env) tags() []string {
	e.t.Helper()
	ts, err := e.repo.Tags(context.Background())
	if err != nil {
		e.t.Fatal(err)
	}
	slices.Sort(ts)
	return ts
}

func (e *env) page(dir, path string) string {
	e.t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "content", path))
	if err != nil {
		e.t.Fatal(err)
	}
	return string(b)
}

func lastmod(m *bundle.Manifest, path string) string {
	for _, p := range m.Pages {
		if p.Path == path {
			return p.Lastmod
		}
	}
	return "missing"
}

// built revises fix and checks the manifest: version 1.2.3 at revision
// rev, built from the release commit with exactly patches applied, clean,
// by this tool, and nothing pushed.
func (e *env) built(fix string, rev int, patches ...string) (m *bundle.Manifest, dir string) {
	e.t.Helper()
	before := e.tags()
	res, dir, err := e.revise(fix)
	if err != nil {
		e.t.Fatal(err)
	}
	m, err = bundle.Read(dir)
	if err != nil {
		e.t.Fatal(err)
	}
	if res.Revision != rev || !slices.Equal(res.Patches, patches) {
		e.t.Fatalf("result %+v, want revision %d with %v", res, rev, patches)
	}
	if m.Version != "1.2.3" || m.Revision != rev || m.Source.Ref != tag || m.Source.Commit != e.release || !slices.Equal(m.Source.Patches, patches) || m.Source.Dirty || m.Tool != "0.2.0" {
		e.t.Fatalf("manifest %+v", m)
	}
	if after := e.tags(); !slices.Equal(before, after) {
		e.t.Fatalf("revise wrote to the registry: %v -> %v", before, after)
	}
	if ws := e.r.Git("worktree", "list"); strings.Count(ws, "\n") != 2 {
		e.t.Fatalf("the temporary worktree is left:\n%s", ws)
	}
	return m, dir
}

// refused revises fix and wants an error containing want.
func (e *env) refused(fix, want string) {
	e.t.Helper()
	if _, _, err := e.revise(fix); err == nil || !strings.Contains(err.Error(), want) {
		e.t.Fatalf("revise %s: %v, want an error with %q", fix[:12], err, want)
	}
}

func TestRevisions(t *testing.T) {
	e := newEnv(t)
	fixA := e.fix(fixDateA, map[string][2]string{backup: {"A platform's backup\n// adapter reads it.", "A platform's backup\n// adapter reads it every night."}})
	e.refused(fixA, "publish the release first: dispatch mode: release")
	e.publishRelease()

	// The first revision: a comment-only fix in CUE.
	m, dir := e.built(fixA, 1, fixA)
	if !strings.Contains(e.page(dir, "traits/backup-v1alpha1.md"), "every night") {
		t.Fatal("the fix is not in the revision's page")
	}
	if got := lastmod(m, "traits/backup-v1alpha1.md"); got != fixDateA {
		t.Fatalf("the patched member's lastmod is %s, want the fix's date", got)
	}
	if got := lastmod(m, "traits/backup.md"); got != gittest.Date {
		t.Fatalf("an unpatched member's lastmod is %s, want the release's", got)
	}
	e.publishBuilt(dir)
	e.refused(fixA, "already applied in 1.2.3.1")

	// The second revision carries the first fix: a Markdown fix.
	fixB := e.fix(fixDateB, map[string][2]string{landing: {"Every member is listed below.", "Every member of the demo catalog is listed below."}})
	m, dir = e.built(fixB, 2, fixA, fixB)
	if !strings.Contains(e.page(dir, "traits/backup-v1alpha1.md"), "every night") || !strings.Contains(e.page(dir, "_index.md"), "of the demo catalog") {
		t.Fatal("the second revision lacks one of the two fixes")
	}
	if got := lastmod(m, "_index.md"); got != fixDateB {
		t.Fatalf("the patched landing's lastmod is %s", got)
	}
	e.publishBuilt(dir)
	if at, _ := e.repo.Resolve(context.Background(), "1.2"); at.Digest.String() == "" {
		t.Fatal("1.2 does not resolve")
	}
}

// A run that fails after its push is finished by running it again.
func TestRerun(t *testing.T) {
	e := newEnv(t)
	e.publishRelease()
	fixA := e.fix(fixDateA, map[string][2]string{landing: {"Every member is listed below.", "Every member is listed here."}})

	// Pushed, never signed: rebuilt to the same digest, trusted because
	// 1.2.3.0 is signed and 1.2.3.1 is 1.2.3.0's fixes plus this one.
	_, dir := e.built(fixA, 1, fixA)
	first := e.push(dir).Digest
	_, dir = e.built(fixA, 1, fixA)
	if again := e.push(dir); !again.Existing || again.Digest != first {
		t.Fatalf("the rebuilt revision is %s (existing %v), want %s unchanged", again.Digest, again.Existing, first)
	}

	// A new fix on top of an unsigned revision is refused.
	fixB := e.fix(fixDateB, map[string][2]string{landing: {"Every member is listed here.", "Each member is listed here."}})
	e.refused(fixB, "1.2.3.1 ("+first+") is not signed by docs-kit's publish workflow")

	// Built by another opm-docs: building it again would not give the
	// pushed bytes.
	e.tool = "0.2.1"
	e.refused(fixA, "built by opm-docs 0.2.0, not 0.2.1")
	e.tool = "0.2.0"

	// Signed and partly promoted: 1.2.3 moved, 1.2 and 1 did not. Rebuilt,
	// so the promote that follows moves the rest.
	e.sign(first)
	e.tagAs(first, "1.2.3")
	e.built(fixA, 1, fixA)
	e.promote(first)
	e.refused(fixA, "already applied in 1.2.3.1")

	// fixA is not the last fix of the unpromoted 1.2.3.2: refused.
	_, dir = e.built(fixB, 2, fixA, fixB)
	e.push(dir)
	e.refused(fixA, "already applied in 1.2.3.2")
}

// A docs bundle of the same name: authored pages only, so they get edit.
const docsKitDocs = `bundles: "catalog-demo": {
	placement: {kind: "docs", root: "/docs/"}
	version: {from: "tag", prefix: "demo-v"}
	sources: [{kind: "markdown", dir: "docs/catalogs/demo"}]
}
`

// A rerun after main renamed a page rebuilds the pushed bytes: the
// rebuilt revision keeps the edit it recorded instead of reading main.
func TestRerunKeepsEdit(t *testing.T) {
	e := newEnvWith(t, docsKitDocs)
	e.publishRelease()
	fixA := e.fix(fixDateA, map[string][2]string{landing: {"Every member is listed below.", "Every member is listed here."}})
	m, dir := e.built(fixA, 1, fixA)
	if p := m.Pages[0]; p.Edit != landing {
		t.Fatalf("page %+v, want edit %s", p, landing)
	}
	first := e.push(dir).Digest

	e.r.Git("mv", landing, "docs/catalogs/demo/contract.md")
	e.r.Commit("rename the landing on main")
	e.syncMain()

	m, dir = e.built(fixA, 1, fixA)
	if p := m.Pages[0]; p.Edit != landing {
		t.Fatalf("rebuilt page %+v, want the recorded edit %s", p, landing)
	}
	if again := e.push(dir); !again.Existing || again.Digest != first {
		t.Fatalf("the rebuilt revision is %s (existing %v), want %s unchanged", again.Digest, again.Existing, first)
	}
}

func TestNoOpFix(t *testing.T) {
	e := newEnv(t)
	e.publishRelease()
	// The second fix undoes the first; picked alone onto the release, it
	// changes nothing.
	e.fix(fixDateA, map[string][2]string{landing: {"Every member is listed below.", "Every member is listed here."}})
	undo := e.fix(fixDateB, map[string][2]string{landing: {"Every member is listed here.", "Every member is listed below."}})
	e.refused(undo, "changes nothing in the release tree")
}

func TestRefusals(t *testing.T) {
	e := newEnv(t)
	e.publishRelease()
	ctx := context.Background()

	// A value change, after a comment change in the same file.
	code := e.fix(fixDateA, map[string][2]string{backup: {
		`description:    "Scheduled backup policy for a component's state"`,
		`description:    "Scheduled backup policy for the state of a component"`,
	}})
	_, _, err := e.revise(code)
	var re *RefusedError
	if !errors.As(err, &re) || !strings.Contains(err.Error(), backup+": changes CUE values, not only comments; a change to code needs a patch release") {
		t.Fatalf("a description change: %v", err)
	}

	// A fix not on main.
	e.r.Git("checkout", "-q", "-b", "side")
	e.r.Write(map[string]string{landing: landingPage + "\nSide.\n"})
	branchFix := e.r.CommitAt(fixDateA, "side fix")
	e.r.Git("checkout", "-q", "main")
	if _, _, err := e.revise(branchFix); err == nil || !strings.Contains(err.Error(), "the fix must land on main first") {
		t.Fatalf("a fix off main: %v", err)
	}

	// A merge commit.
	e.r.GitAt(fixDateA, "merge", "-q", "--no-ff", "-m", "merge side", "side")
	e.syncMain()
	if _, _, err := e.revise(e.r.Head()); err == nil || !strings.Contains(err.Error(), "has 2 parents") {
		t.Fatalf("a merge commit: %v", err)
	}

	// A Markdown fix whose earlier fix conflicts: two edits of one line,
	// the second picked alone after a revision that carries neither.
	e.fix(fixDateA, map[string][2]string{landing: {"Every member is listed below.", "Each member is listed below."}})
	two := e.fix(fixDateB, map[string][2]string{landing: {"Each member is listed below.", "Each member is listed here."}})
	if _, _, err := e.revise(two); err == nil || !strings.Contains(err.Error(), "conflicts in "+landing) {
		t.Fatalf("a conflicting fix: %v", err)
	}

	// The wrong prefix.
	_, err = Run(ctx, Options{Repo: e.r.Dir, Project: project, Tag: "v1.2.3", Fix: two, Out: t.TempDir(), Registry: e.registry, Tool: e.tool, Client: e.client})
	if err == nil || !strings.Contains(err.Error(), `prefix "demo-v"`) {
		t.Fatalf("wrong prefix: %v", err)
	}
	if ws := e.r.Git("worktree", "list"); strings.Count(ws, "\n") != 2 {
		t.Fatalf("a refused revision left its worktree:\n%s", ws)
	}
}

func TestFixAlreadyInRelease(t *testing.T) {
	e := newEnv(t)
	e.publishRelease()
	if _, _, err := e.revise(e.release); err == nil || !strings.Contains(err.Error(), "already in the release "+tag) {
		t.Fatalf("a fix in the release: %v", err)
	}
}
