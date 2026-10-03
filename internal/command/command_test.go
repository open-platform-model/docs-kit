package command

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var echo string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "command-test-*")
	if err != nil {
		panic(err)
	}
	echo = filepath.Join(dir, "echo")
	if out, err := exec.Command("go", "build", "-o", echo, "./testdata/echo").CombinedOutput(); err != nil { //nolint:gosec,noctx // test setup
		panic(string(out))
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestRun(t *testing.T) {
	src := t.TempDir()
	for _, c := range []struct {
		name string
		r    Runner
		arg  string
		want string // in the output, or in the error when err
		err  bool
	}{
		{name: "ok", arg: "ok", want: `"value": 1`},
		{name: "environment and directory", arg: "env", want: `"env": "1,cli,edge"`},
		{name: "failure", arg: "fail", err: true, want: "exited with status 1"},
		{name: "trailing output", arg: "trailing", err: true, want: "exactly one JSON document"},
		{name: "unknown schema", arg: "schema", err: true, want: `"other/v1"`},
		{name: "timeout", arg: "sleep", r: Runner{Timeout: 200 * time.Millisecond}, err: true, want: "ran over 200ms"},
		{name: "output cap", arg: "big", r: Runner{MaxOutput: 1024}, err: true, want: "more than 1024 bytes"},
		{name: "nondeterminism", arg: "random", r: Runner{Twice: true}, err: true, want: "a docs build must be deterministic"},
		{name: "deterministic twice", arg: "ok", r: Runner{Twice: true}, want: `"value": 1`},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := c.r
			r.Dir, r.Project, r.Version = src, "cli", "edge"
			var stderr strings.Builder
			r.Stderr = &stderr
			out, err := r.Run(context.Background(), []string{echo, c.arg}, "test/v1")
			if c.err {
				var ce *Error
				if !errors.As(err, &ce) || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), "cli") || !strings.Contains(err.Error(), c.arg) {
					t.Fatalf("err = %v, want one naming %q, the project and the argv", err, c.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(out), c.want) {
				t.Fatalf("output %s, want %q", out, c.want)
			}
			if c.arg == "env" && !strings.Contains(string(out), src) {
				t.Fatalf("not run in the source tree: %s", out)
			}
		})
	}
}

func TestStderrPassesThrough(t *testing.T) {
	var stderr strings.Builder
	r := Runner{Dir: t.TempDir(), Project: "cli", Version: "edge", Stderr: &stderr}
	_, _ = r.Run(context.Background(), []string{echo, "fail"}, "test/v1")
	if !strings.Contains(stderr.String(), "failing") {
		t.Fatalf("stderr %q", stderr.String())
	}
}
