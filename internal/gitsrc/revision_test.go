package gitsrc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/gittest"
)

// history is a release tag and two fixes on main after it, with
// origin/main at the newest commit and a branch commit off main.
type history struct {
	r            *gittest.Repo
	release      string // commit of the tag opm-v4.4.5
	fixA, fixB   string // documentation fixes on main
	later        string // a code change on main after the tag
	conflict     string // a fix to a line the release tree does not have
	merge        string // a merge commit on main
	branch, root string // a fix never merged to main; the root commit
}

func newHistory(t *testing.T) *history {
	t.Helper()
	h := &history{r: gittest.New(t, "")}
	r := h.r
	r.Write(map[string]string{"docs/a.md": "A.\n", "docs/b.md": "B.\n", "x.cue": "package x\n\nv: 1\n"})
	h.root = r.CommitAt("2026-09-01T00:00:00Z", "root")
	r.Write(map[string]string{"notes.md": "one\n"})
	h.release = r.CommitAt("2026-09-02T00:00:00Z", "release")
	r.Git("tag", "opm-v4.4.5")
	r.Write(map[string]string{"docs/a.md": "A, fixed.\n"})
	h.fixA = r.CommitAt("2026-09-03T00:00:00Z", "fix a")
	r.Write(map[string]string{"x.cue": "package x\n\nv: 2\n", "notes.md": "two\n"})
	h.later = r.CommitAt("2026-09-04T00:00:00Z", "code and notes")
	r.Write(map[string]string{"docs/a.md": "A, fixed again.\n", "docs/b.md": "B, fixed.\n"})
	h.fixB = r.CommitAt("2026-09-05T00:00:00Z", "fix b")
	r.Write(map[string]string{"notes.md": "three\n"})
	h.conflict = r.CommitAt("2026-09-06T00:00:00Z", "notes again")
	r.Git("checkout", "-q", "-b", "side", h.release)
	r.Write(map[string]string{"side.md": "side\n"})
	r.CommitAt("2026-09-07T00:00:00Z", "side")
	r.Git("checkout", "-q", "main")
	r.GitAt("2026-09-08T00:00:00Z", "merge", "-q", "--no-ff", "-m", "merge side", "side")
	h.merge = r.Head()
	r.Git("checkout", "-q", "-b", "unmerged", h.release)
	r.Write(map[string]string{"docs/a.md": "A, fixed on a branch.\n"})
	h.branch = r.CommitAt("2026-09-09T00:00:00Z", "unmerged fix")
	r.Git("checkout", "-q", "main")
	r.Git("update-ref", "refs/remotes/origin/main", h.merge)
	return h
}

func TestCheckFix(t *testing.T) {
	h := newHistory(t)
	repo := Repo{Dir: h.r.Dir}
	ctx := t.Context()
	for _, tc := range []struct {
		name, fix, want string
	}{
		{"on main", h.fixA, ""},
		{"short hash", h.fixA[:12], "not a full 40-hex commit hash"},
		{"upper case", strings.ToUpper(h.fixA), "not a full 40-hex commit hash"},
		{"unknown commit", strings.Repeat("ab", 20), "is not a commit"},
		{"merge commit", h.merge, "has 2 parents"},
		{"root commit", h.root, "has 0 parents"},
		{"not on main", h.branch, "the fix must land on main first"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := repo.CheckFix(ctx, tc.fix, "origin/main")
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
	if err := repo.CheckFix(ctx, h.fixA, "origin/nope"); err == nil || !strings.Contains(err.Error(), "does not resolve") {
		t.Fatalf("missing main ref: %v", err)
	}
}

func worktrees(t *testing.T, r *gittest.Repo) int {
	t.Helper()
	return strings.Count(r.Git("worktree", "list", "--porcelain"), "worktree ")
}

func TestWorktreeCherryPick(t *testing.T) {
	h := newHistory(t)
	repo := Repo{Dir: h.r.Dir}
	ctx := t.Context()

	w, err := repo.AddWorktree(ctx, "opm-v4.4.5")
	if err != nil {
		t.Fatal(err)
	}
	if worktrees(t, h.r) != 2 {
		t.Fatal("no worktree added")
	}
	if err := w.CherryPick(ctx, []string{h.fixA, h.fixB}); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"docs/a.md": "A, fixed again.\n", "docs/b.md": "B, fixed.\n", "x.cue": "package x\n\nv: 1\n", "notes.md": "one\n"}
	for p, body := range want {
		if b, _ := os.ReadFile(filepath.Join(w.Dir, p)); string(b) != body {
			t.Fatalf("%s is %q, want %q: the fixes apply to the release tree alone", p, b, body)
		}
	}
	if head, _ := w.Commit(ctx, "HEAD"); head != h.release {
		t.Fatalf("HEAD moved to %s; --no-commit commits nothing", head)
	}
	tree, err := w.IndexTree(ctx)
	if err != nil {
		t.Fatal(err)
	}
	refused, err := repo.DocumentationOnly(ctx, "opm-v4.4.5", tree)
	if err != nil || len(refused) != 0 {
		t.Fatalf("documentation-only: %v %v", refused, err)
	}
	if err := w.Remove(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(w.Dir); !os.IsNotExist(err) {
		t.Fatal("the worktree directory is left")
	}
	if worktrees(t, h.r) != 1 {
		t.Fatal("the worktree is still registered")
	}
}

func TestCherryPickConflict(t *testing.T) {
	h := newHistory(t)
	repo := Repo{Dir: h.r.Dir}
	ctx, cancel := context.WithCancel(t.Context())
	w, err := repo.AddWorktree(ctx, "opm-v4.4.5")
	if err != nil {
		t.Fatal(err)
	}
	err = w.CherryPick(ctx, []string{h.fixA, h.conflict})
	var ce *ConflictError
	if !errors.As(err, &ce) || ce.Commit != h.conflict || strings.Join(ce.Files, ",") != "notes.md" {
		t.Fatalf("got %v, want a conflict of %s in notes.md", err, h.conflict)
	}
	if !strings.Contains(err.Error(), "conflicts in notes.md; land one fix on main") {
		t.Fatalf("message %q", err)
	}
	cancel() // removal still runs
	if err := w.Remove(ctx); err != nil {
		t.Fatal(err)
	}
	if worktrees(t, h.r) != 1 {
		t.Fatal("the worktree is still registered after a conflict")
	}
	if st := h.r.Git("status", "--porcelain"); st != "" {
		t.Fatalf("the main work tree changed:\n%s", st)
	}
}

func TestPatchDates(t *testing.T) {
	h := newHistory(t)
	got, err := Repo{Dir: h.r.Dir}.PatchDates(t.Context(), []string{h.fixA, h.fixB})
	if err != nil {
		t.Fatal(err)
	}
	if got["docs/a.md"] != "2026-09-05T00:00:00Z" || got["docs/b.md"] != "2026-09-05T00:00:00Z" || len(got) != 2 {
		t.Fatalf("dates %v", got)
	}
	got, _ = Repo{Dir: h.r.Dir}.PatchDates(t.Context(), []string{h.fixA})
	if got["docs/a.md"] != "2026-09-03T00:00:00Z" || len(got) != 1 {
		t.Fatalf("dates %v", got)
	}
}

func TestPatchedDirty(t *testing.T) {
	h := newHistory(t)
	ctx := t.Context()
	w, err := Repo{Dir: h.r.Dir}.AddWorktree(ctx, "opm-v4.4.5")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Remove(ctx) }()
	if err := w.CherryPick(ctx, []string{h.fixA, h.fixB}); err != nil {
		t.Fatal(err)
	}
	if d, err := w.PatchedDirty(ctx, []string{h.fixA, h.fixB}); err != nil || d {
		t.Fatalf("patched tree dirty=%v %v", d, err)
	}
	if d, _ := w.PatchedDirty(ctx, []string{h.fixA}); !d {
		t.Fatal("a staged file no patch touches did not make the tree dirty")
	}
	out := filepath.Join(w.Dir, "out")
	if err := os.MkdirAll(out, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "x"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if d, _ := w.PatchedDirty(ctx, []string{h.fixA, h.fixB}, out); d {
		t.Fatal("the output directory made the tree dirty")
	}
	if d, _ := w.PatchedDirty(ctx, []string{h.fixA, h.fixB}); !d {
		t.Fatal("an untracked file did not make the tree dirty")
	}
	if err := os.WriteFile(filepath.Join(w.Dir, "docs", "a.md"), []byte("edited\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if d, _ := w.PatchedDirty(ctx, []string{h.fixA, h.fixB}, out); !d {
		t.Fatal("an unstaged edit did not make the tree dirty")
	}
}
