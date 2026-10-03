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
	release  string // the tag's commit
}

// newEnv makes a repository tagged demo-v1.2.3 with origin/main at HEAD,
// and an empty registry.
func newEnv(t *testing.T) *env {
	t.Helper()
	t.Setenv("GITHUB_REPOSITORY", "")
	t.Setenv("GITHUB_REF_TYPE", "")
	r := gittest.New(t, "https://github.com/example/demo.git")
	r.Write(map[string]string{"README.md": "# Demo\n"})
	r.Commit("start")
	r.CopyTree("../extract/cuecatalog/testdata/catalog", ".")
	r.Write(map[string]string{"docs-kit.cue": docsKit, landing: landingPage, ".gitignore": "/out/\n"})
	r.Commit("catalog")
	r.Git("tag", tag)
	host := ocitest.Registry(t)
	c := oci.New(oci.Options{Anonymous: true, PlainHTTP: true})
	repo, err := c.Repository(host + "/docs/" + project)
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, r: r, registry: host + "/docs", client: c, repo: repo, release: r.Head()}
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
func (e *env) push(dir string) {
	e.t.Helper()
	if _, err := publish.Push(context.Background(), publish.PushOptions{Dir: dir, Registry: e.registry, Client: e.client}); err != nil {
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
	e.push(filepath.Join(out, project))
}

func (e *env) revise(fix string) (*Result, string, error) {
	e.t.Helper()
	out := filepath.Join(e.t.TempDir(), "out")
	res, err := Run(context.Background(), Options{
		Repo: e.r.Dir, Project: project, Tag: tag, Fix: fix, Out: out,
		Registry: e.registry, Tool: "0.2.0", Client: e.client,
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

func TestRevisions(t *testing.T) {
	e := newEnv(t)
	fixA := e.fix(fixDateA, map[string][2]string{backup: {"A platform's backup\n// adapter reads it.", "A platform's backup\n// adapter reads it every night."}})
	if _, _, err := e.revise(fixA); err == nil || !strings.Contains(err.Error(), "publish the release first: dispatch mode: release") {
		t.Fatalf("no revision 0: %v", err)
	}
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
	e.push(dir)

	// The fix again: refused.
	if _, _, err := e.revise(fixA); err == nil || !strings.Contains(err.Error(), "already applied in 1.2.3.1") {
		t.Fatalf("a fix applied twice: %v", err)
	}

	// The second revision carries the first fix: a Markdown fix.
	fixB := e.fix(fixDateB, map[string][2]string{landing: {"Every member is listed below.", "Every member of the demo catalog is listed below."}})
	m, dir = e.built(fixB, 2, fixA, fixB)
	if !strings.Contains(e.page(dir, "traits/backup-v1alpha1.md"), "every night") || !strings.Contains(e.page(dir, "_index.md"), "of the demo catalog") {
		t.Fatal("the second revision lacks one of the two fixes")
	}
	if got := lastmod(m, "_index.md"); got != fixDateB {
		t.Fatalf("the patched landing's lastmod is %s", got)
	}
	e.push(dir)
	if ts := e.tags(); !slices.Contains(ts, "1.2.3.2") {
		t.Fatalf("tags %v", ts)
	}
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
	_, err = Run(ctx, Options{Repo: e.r.Dir, Project: project, Tag: "v1.2.3", Fix: two, Out: t.TempDir(), Registry: e.registry, Tool: "0.2.0", Client: e.client})
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
