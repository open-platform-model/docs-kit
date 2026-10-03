package gitsrc

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

var reSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

// IsSHA reports a full, lower-case, 40-hex commit hash.
func IsSHA(s string) bool { return reSHA.MatchString(s) }

// CheckFix refuses a fix commit a docs revision may not apply: anything
// but a 40-hex hash of a commit with exactly one parent that is an
// ancestor of main (a ref such as "origin/main").
func (r Repo) CheckFix(ctx context.Context, fix, main string) error {
	if !IsSHA(fix) {
		return fmt.Errorf("the fix %q is not a full 40-hex commit hash; pass the commit's full SHA-1 hash (a SHA-256 repository is not supported)", fix)
	}
	if _, err := r.git(ctx, "cat-file", "-e", fix+"^{commit}"); err != nil {
		return fmt.Errorf("the fix %s is not a commit in %s; land the fix on main first, and check out main with full history", fix, r.Dir)
	}
	parents, err := r.git(ctx, "rev-list", "--parents", "-n", "1", fix)
	if err != nil {
		return err
	}
	if n := len(strings.Fields(parents)) - 1; n != 1 {
		return fmt.Errorf("the fix %s has %d parents; a docs revision applies a single-parent commit, so pass the commit that made the change, not a merge", fix, n)
	}
	if _, err := r.git(ctx, "rev-parse", "--verify", "--quiet", main+"^{commit}"); err != nil {
		return fmt.Errorf("%s does not resolve in %s; check out the repository with full history", main, r.Dir)
	}
	if !r.IsAncestor(ctx, fix, main) {
		return fmt.Errorf("the fix %s is not on %s; the fix must land on main first", fix, main)
	}
	return nil
}

// IsAncestor reports whether commit a is an ancestor of (or is) b.
func (r Repo) IsAncestor(ctx context.Context, a, b string) bool {
	return r.cmd(ctx, "merge-base", "--is-ancestor", a, b).Run() == nil
}

// Worktree is a temporary, detached git worktree.
type Worktree struct {
	Repo
	parent Repo
}

// AddWorktree checks rev out, detached, in a new temporary directory.
// Remove it with Remove on every path.
func (r Repo) AddWorktree(ctx context.Context, rev string) (*Worktree, error) {
	dir, err := os.MkdirTemp("", "opm-docs-revise-*")
	if err != nil {
		return nil, err
	}
	if _, err := r.git(ctx, "worktree", "add", "--detach", "--quiet", dir, rev); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	return &Worktree{Repo: Repo{Dir: dir}, parent: r}, nil
}

// Remove deletes the worktree and its administrative files, even when ctx
// was canceled.
func (w *Worktree) Remove(ctx context.Context) error {
	ctx = context.WithoutCancel(ctx)
	_, err := w.parent.git(ctx, "worktree", "remove", "--force", w.Dir)
	if rmErr := os.RemoveAll(w.Dir); err == nil {
		err = rmErr
	}
	if _, pruneErr := w.parent.git(ctx, "worktree", "prune"); err == nil {
		err = pruneErr
	}
	return err
}

// ConflictError is a fix that does not apply to the tree it was picked
// onto.
type ConflictError struct {
	Commit string
	Files  []string
}

func (e *ConflictError) Error() string {
	files := "files git did not name"
	if len(e.Files) > 0 {
		files = strings.Join(e.Files, ", ")
	}
	return fmt.Sprintf("the fix %s conflicts in %s; land one fix on main that makes the whole change and revise with that", e.Commit, files)
}

// Picked is what CherryPick applied.
type Picked struct {
	Base  string   // the index tree before the first pick
	Trees []string // the index tree after each pick
	// Dates maps every path a pick added or changed in this tree to the
	// committer date, RFC 3339 UTC, of the newest pick that did: a patched
	// file's lastmod. The paths are read from the index after each pick,
	// so a file renamed on main after the release counts under its name in
	// the release tree.
	Dates map[string]string
}

// CherryPick applies each commit in order to the index and work tree with
// `git cherry-pick --no-commit`, committing nothing, and records what each
// changed. A conflict is a *ConflictError naming the files.
func (r Repo) CherryPick(ctx context.Context, commits []string) (*Picked, error) {
	base, err := r.IndexTree(ctx)
	if err != nil {
		return nil, err
	}
	p := &Picked{Base: base, Dates: map[string]string{}}
	newest := map[string]time.Time{}
	prev := base
	for _, c := range commits {
		if _, err := r.git(ctx, "cherry-pick", "--no-commit", c); err != nil {
			files, ferr := r.git(ctx, "diff", "--name-only", "--diff-filter=U")
			if ferr != nil || files == "" {
				return nil, fmt.Errorf("applying the fix %s: %w", c, err)
			}
			return nil, &ConflictError{Commit: c, Files: strings.Split(files, "\n")}
		}
		tree, err := r.IndexTree(ctx)
		if err != nil {
			return nil, err
		}
		when, err := r.CommitTime(ctx, c)
		if err != nil {
			return nil, err
		}
		names, err := r.git(ctx, "diff-tree", "-r", "--name-only", "--no-renames", "--diff-filter=d", prev, tree)
		if err != nil {
			return nil, err
		}
		for _, n := range strings.Split(names, "\n") {
			if cur, ok := newest[n]; n != "" && (!ok || when.After(cur)) {
				newest[n] = when
			}
		}
		p.Trees = append(p.Trees, tree)
		prev = tree
	}
	for n, t := range newest {
		p.Dates[n] = t.Format(time.RFC3339)
	}
	return p, nil
}

// IndexTree writes the index as a tree object and returns its hash: after
// CherryPick, the release tree with the fixes applied.
func (r Repo) IndexTree(ctx context.Context) (string, error) {
	return r.git(ctx, "write-tree")
}
