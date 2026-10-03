// Package gitsrc reads what a build needs from the source repository's
// git history: the commit built, its time, whether the work tree is dirty,
// the repository name and each file's last commit date. For a docs
// revision it also checks a fix commit, applies fixes to a release tree in
// a temporary worktree and checks that they change documentation only.
package gitsrc

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Repo is a git work tree.
type Repo struct {
	Dir string
}

func (r Repo) cmd(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "git", append([]string{"-C", r.Dir}, args...)...) //nolint:gosec // git with arguments this package builds
}

func (r Repo) git(ctx context.Context, args ...string) (string, error) {
	cmd := r.cmd(ctx, args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s in %s: %w: %s", strings.Join(args, " "), r.Dir, err, strings.TrimSpace(errb.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// IsRepo reports whether Dir is inside a git work tree.
func (r Repo) IsRepo(ctx context.Context) bool {
	out, err := r.git(ctx, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// Commit resolves a revision ("HEAD", a tag) to its full commit hash.
func (r Repo) Commit(ctx context.Context, rev string) (string, error) {
	return r.git(ctx, "rev-parse", "--verify", "--quiet", rev+"^{commit}")
}

// CommitTime is a commit's committer time, in UTC.
func (r Repo) CommitTime(ctx context.Context, commit string) (time.Time, error) {
	out, err := r.git(ctx, "show", "-s", "--format=%cI", commit)
	if err != nil {
		return time.Time{}, err
	}
	t, err := time.Parse(time.RFC3339, out)
	if err != nil {
		return time.Time{}, fmt.Errorf("commit time %q of %s: %w", out, commit, err)
	}
	return t.UTC(), nil
}

// Dirty reports uncommitted changes in the work tree, untracked files
// included, ignoring paths under any of skip (an output directory inside
// the tree).
func (r Repo) Dirty(ctx context.Context, skip ...string) (bool, error) {
	out, err := r.git(ctx, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return false, err
	}
	top, err := r.git(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 4 {
			continue
		}
		p := filepath.Join(top, filepath.FromSlash(strings.Trim(line[3:], `"`)))
		if !under(p, skip) {
			return true, nil
		}
	}
	return false, nil
}

func under(p string, dirs []string) bool {
	for _, d := range dirs {
		abs, err := filepath.Abs(d)
		if err != nil {
			continue
		}
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			abs = resolved
		}
		if rel, err := filepath.Rel(abs, p); err == nil && rel != ".." && !strings.HasPrefix(rel, "../") {
			return true
		}
	}
	return false
}

// LastMod is the committer date, RFC 3339, of the last commit at or before
// commit that touched path (relative to Dir); "" when the history does not
// have it.
func (r Repo) LastMod(ctx context.Context, commit, path string) string {
	out, err := r.git(ctx, "log", "-1", "--format=%cI", commit, "--", path)
	if err != nil || out == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, out)
	if err != nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// Branch is the branch checked out: GITHUB_REF_NAME for a branch build in
// GitHub Actions (whose checkout is detached), else the symbolic HEAD.
func (r Repo) Branch(ctx context.Context) string {
	if os.Getenv("GITHUB_REF_TYPE") == "branch" && os.Getenv("GITHUB_REF_NAME") != "" {
		return os.Getenv("GITHUB_REF_NAME")
	}
	out, err := r.git(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || out == "" {
		return "HEAD"
	}
	return out
}

var reGitHub = regexp.MustCompile(`github\.com[:/]([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+?)(?:\.git)?/?$`)

// Name is the repository's "owner/name": GITHUB_REPOSITORY in GitHub
// Actions, else parsed from the origin remote, else "local/<directory>".
func (r Repo) Name(ctx context.Context) string {
	if n := os.Getenv("GITHUB_REPOSITORY"); n != "" {
		return n
	}
	if url, err := r.git(ctx, "remote", "get-url", "origin"); err == nil {
		if m := reGitHub.FindStringSubmatch(url); m != nil {
			return m[1] + "/" + m[2]
		}
	}
	top, err := r.git(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		top = r.Dir
	}
	name := regexp.MustCompile(`[^A-Za-z0-9_.-]`).ReplaceAllString(filepath.Base(top), "-")
	return "local/" + name
}
