package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestReviseExitCodes(t *testing.T) {
	r := demoRepo(t)
	r.Git("tag", "demo-v1.2.3")
	r.Git("checkout", "-q", "-b", "side")
	r.Write(map[string]string{"notes.md": "notes\n"})
	r.Commit("side")
	side := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	r.Git("checkout", "-q", "main")
	r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
	t.Chdir(r.Dir)
	for _, c := range []struct {
		name string
		args []string
		code int
		want string
	}{
		{"missing fix", []string{"revise", "--project", "catalog-demo", "--tag", "demo-v1.2.3"}, 1, "--fix is required"},
		{"missing tag", []string{"revise", "--project", "catalog-demo", "--fix", side}, 1, "--tag is required"},
		{"fix not on main", []string{"revise", "--project", "catalog-demo", "--tag", "demo-v1.2.3", "--fix", side}, 2, "the fix must land on main first"},
		{"short fix", []string{"revise", "--project", "catalog-demo", "--tag", "demo-v1.2.3", "--fix", side[:7]}, 2, "not a full 40-hex commit hash"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			if code := run(c.args, &out, &errb); code != c.code || !strings.Contains(errb.String(), c.want) {
				t.Fatalf("exit %d (want %d): %s", code, c.code, errb.String())
			}
		})
	}
}
