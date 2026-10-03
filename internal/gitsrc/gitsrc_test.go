package gitsrc

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// newRepo makes a repository with one commit at a fixed time.
func newRepo(t *testing.T) Repo {
	t.Helper()
	dir := t.TempDir()
	run := func(env []string, args ...string) {
		cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	date := []string{"GIT_COMMITTER_DATE=2026-09-30T12:00:00Z", "GIT_AUTHOR_DATE=2026-09-30T12:00:00Z"}
	run(nil, "init", "-q", "-b", "main")
	run(nil, "config", "user.email", "t@example.com")
	run(nil, "config", "user.name", "t")
	run(nil, "remote", "add", "origin", "https://github.com/open-platform-model/catalog_opm.git")
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(nil, "add", "a.md")
	run(date, "commit", "-q", "-m", "one")
	run(nil, "tag", "opm-v4.4.5")
	return Repo{Dir: dir}
}

func TestRepo(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "")
	t.Setenv("GITHUB_REF_TYPE", "")
	ctx := context.Background()
	r := newRepo(t)
	if !r.IsRepo(ctx) {
		t.Fatal("not a repo")
	}
	head, err := r.Commit(ctx, "HEAD")
	if err != nil || len(head) != 40 {
		t.Fatalf("head %q %v", head, err)
	}
	if tag, _ := r.Commit(ctx, "opm-v4.4.5"); tag != head {
		t.Fatalf("tag %s, head %s", tag, head)
	}
	if _, err := r.Commit(ctx, "opm-v9.9.9"); err == nil {
		t.Fatal("missing tag resolved")
	}
	ct, err := r.CommitTime(ctx, head)
	if err != nil || ct.Format("2006-01-02T15:04:05Z07:00") != "2026-09-30T12:00:00Z" {
		t.Fatalf("commit time %v %v", ct, err)
	}
	if got := r.LastMod(ctx, head, "a.md"); got != "2026-09-30T12:00:00Z" {
		t.Fatalf("lastmod %q", got)
	}
	if got := r.LastMod(ctx, head, "missing.md"); got != "" {
		t.Fatalf("lastmod of a missing file %q", got)
	}
	if got := r.Name(ctx); got != "open-platform-model/catalog_opm" {
		t.Fatalf("name %q", got)
	}
	if got := r.Branch(ctx); got != "main" {
		t.Fatalf("branch %q", got)
	}
}

func TestDirty(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)
	if d, err := r.Dirty(ctx); err != nil || d {
		t.Fatalf("clean tree dirty=%v %v", d, err)
	}
	out := filepath.Join(r.Dir, "out")
	_ = os.MkdirAll(out, 0o755)
	_ = os.WriteFile(filepath.Join(out, "x"), []byte("x"), 0o600)
	if d, _ := r.Dirty(ctx, out); d {
		t.Fatal("output directory made the tree dirty")
	}
	_ = os.WriteFile(filepath.Join(r.Dir, "a.md"), []byte("b"), 0o600)
	if d, _ := r.Dirty(ctx, out); !d {
		t.Fatal("a modified file did not make the tree dirty")
	}
}

func TestHasFile(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)
	write := func(name, body string) {
		p := filepath.Join(r.Dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("docs/site/start/install.md", "x")
	write("docs/site/run.sh", "x")
	if err := os.Chmod(filepath.Join(r.Dir, "docs", "site", "run.sh"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("start/install.md", filepath.Join(r.Dir, "docs", "site", "link.md")); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", "two"}} {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.Dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	for _, c := range []struct {
		rev, path string
		want      bool
	}{
		{"HEAD", "docs/site/start/install.md", true},
		{"HEAD", "docs/site/run.sh", true},
		{"HEAD", "a.md", true},
		{"opm-v4.4.5", "docs/site/start/install.md", false}, // not yet in that tree
		{"HEAD", "docs/site/start", false},                  // a directory
		{"HEAD", "docs/site/link.md", false},                // a symlink
		{"HEAD", "docs/site/start/*.md", false},             // a pathspec glob is literal
		{"HEAD", "missing.md", false},
		{"no-such-rev", "a.md", false},
	} {
		if got := r.HasFile(ctx, c.rev, c.path); got != c.want {
			t.Errorf("HasFile(%s, %s) = %v, want %v", c.rev, c.path, got, c.want)
		}
	}
	sub := Repo{Dir: filepath.Join(r.Dir, "docs")}
	if !sub.HasFile(ctx, "HEAD", "site/start/install.md") {
		t.Error("a path relative to a subdirectory Dir was not found")
	}
}
