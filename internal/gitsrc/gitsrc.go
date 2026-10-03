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
	return r.changed(ctx, func(statusEntry) bool { return true }, skip)
}

// Unstaged reports a work tree that differs from its index: a file
// changed and not staged, or an untracked file, ignoring paths under any
// of skip. After CherryPick the index holds the fixes, so this is the
// check that nothing else is in the tree.
func (r Repo) Unstaged(ctx context.Context, skip ...string) (bool, error) {
	return r.changed(ctx, func(e statusEntry) bool { return e.xy[0] == '?' || e.xy[1] != ' ' }, skip)
}

// statusEntry is one entry of `git status --porcelain -z`.
type statusEntry struct {
	xy   string // the two status letters
	path string // the path, the new one of a rename
}

func (r Repo) changed(ctx context.Context, counts func(statusEntry) bool, skip []string) (bool, error) {
	entries, err := r.status(ctx)
	if err != nil {
		return false, err
	}
	top, err := r.git(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if counts(e) && !under(filepath.Join(top, filepath.FromSlash(e.path)), skip) {
			return true, nil
		}
	}
	return false, nil
}

// status reads `git status --porcelain -z --untracked-files=all`: "XY
// path", NUL-terminated, with a rename's or copy's original path as one
// more field.
func (r Repo) status(ctx context.Context) ([]statusEntry, error) {
	cmd := r.cmd(ctx, "status", "--porcelain", "-z", "--untracked-files=all")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git status in %s: %w: %s", r.Dir, err, strings.TrimSpace(errb.String()))
	}
	fields := strings.Split(strings.TrimSuffix(out.String(), "\x00"), "\x00")
	var entries []statusEntry
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if len(f) < 4 {
			continue
		}
		e := statusEntry{xy: f[:2], path: f[3:]}
		if e.xy[0] == 'R' || e.xy[0] == 'C' {
			i++ // the original path
		}
		entries = append(entries, e)
	}
	return entries, nil
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

// HasFile reports whether path, relative to Dir, is a regular file
// (executable or not) in rev's tree; a directory, a symlink, a submodule
// or a path rev does not have is not.
func (r Repo) HasFile(ctx context.Context, rev, path string) bool {
	out, err := r.git(ctx, "--literal-pathspecs", "ls-tree", "-z", rev, "--", path)
	if err != nil || out == "" {
		return false
	}
	for _, e := range strings.Split(strings.TrimSuffix(out, "\x00"), "\x00") {
		meta, name, ok := strings.Cut(e, "\t")
		if !ok || name != path {
			continue
		}
		mode, _, _ := strings.Cut(meta, " ")
		return mode == "100644" || mode == "100755"
	}
	return false
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
