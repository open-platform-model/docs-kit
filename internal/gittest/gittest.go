// Package gittest makes throwaway git repositories for tests.
package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Date is the committer and author time of every commit Commit makes.
const Date = "2026-09-30T12:00:00Z"

// Repo is a temporary repository.
type Repo struct {
	t   testing.TB
	Dir string
}

// New makes an empty repository on branch main with a GitHub origin.
func New(t testing.TB, origin string) *Repo {
	t.Helper()
	r := &Repo{t: t, Dir: t.TempDir()}
	r.Git("init", "-q", "-b", "main")
	r.Git("config", "user.email", "test@example.com")
	r.Git("config", "user.name", "test")
	r.Git("config", "commit.gpgsign", "false")
	r.Git("config", "tag.gpgsign", "false")
	if origin != "" {
		r.Git("remote", "add", "origin", origin)
	}
	return r
}

// Git runs one git command in the repository and fails the test on error.
func (r *Repo) Git(args ...string) string {
	r.t.Helper()
	return r.GitAt(Date, args...)
}

// GitAt runs one git command with date as the committer and author time.
func (r *Repo) GitAt(date string, args ...string) string {
	r.t.Helper()
	cmd := exec.CommandContext(r.t.Context(), "git", append([]string{"-C", r.Dir}, args...)...) //nolint:gosec // test helper
	cmd.Env = append(os.Environ(), "GIT_COMMITTER_DATE="+date, "GIT_AUTHOR_DATE="+date)
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// Head is the full hash of HEAD.
func (r *Repo) Head() string {
	r.t.Helper()
	return strings.TrimSpace(r.Git("rev-parse", "HEAD"))
}

// Write writes files (path relative to the repository: content).
func (r *Repo) Write(files map[string]string) {
	r.t.Helper()
	for p, body := range files {
		full := filepath.Join(r.Dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			r.t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			r.t.Fatal(err)
		}
	}
}

// CopyTree copies a directory into the repository at dst.
func (r *Repo) CopyTree(src, dst string) {
	r.t.Helper()
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(r.Dir, filepath.FromSlash(dst), rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		b, err := os.ReadFile(p) //nolint:gosec // test helper
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o600) //nolint:gosec // test helper writing under its temporary repository
	})
	if err != nil {
		r.t.Fatal(err)
	}
}

// Commit stages everything and commits it.
func (r *Repo) Commit(msg string) {
	r.t.Helper()
	r.CommitAt(Date, msg)
}

// CommitAt stages everything and commits it at date (RFC 3339), returning
// the commit's hash.
func (r *Repo) CommitAt(date, msg string) string {
	r.t.Helper()
	r.Git("add", "-A")
	r.GitAt(date, "commit", "-q", "-m", msg)
	return r.Head()
}
