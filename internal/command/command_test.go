package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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
	if out, err := exec.Command("go", "build", "-o", echo, "./testdata/echo").CombinedOutput(); err != nil { //nolint:noctx // test setup
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

func TestNoTokenReachesACommand(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghs_secret")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/creds.json")
	t.Setenv("GOFLAGS", "-mod=mod")
	r := Runner{Dir: t.TempDir(), Project: "cli", Version: "edge"}
	out, err := r.Run(context.Background(), []string{echo, "token"}, "test/v1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"token": ""`) || !strings.Contains(string(out), `"gopath": "-mod=mod"`) {
		t.Fatalf("environment leaked or Go variable dropped: %s", out)
	}
}

func TestEnviron(t *testing.T) {
	got := strings.Join(Environ([]string{"PATH=/bin", "GITHUB_TOKEN=x", "GOPATH=/g", "GOOGLE_X=y", "CUE_REGISTRY=r", "OPM_DOCS_X=1", "ACTIONS_RUNTIME_TOKEN=t", "DOCKER_CONFIG=/d", "LC_ALL=C"}), " ")
	if got != "PATH=/bin GOPATH=/g CUE_REGISTRY=r OPM_DOCS_X=1 LC_ALL=C" {
		t.Fatalf("%s", got)
	}
}

// A timeout kills the command's whole process group, children included.
func TestTimeoutKillsTheGroup(t *testing.T) {
	var stderr strings.Builder
	r := Runner{Dir: t.TempDir(), Project: "cli", Version: "edge", Timeout: 500 * time.Millisecond, Stderr: &stderr}
	_, err := r.Run(context.Background(), []string{echo, "spawn"}, "test/v1")
	if err == nil || !strings.Contains(err.Error(), "ran over") {
		t.Fatalf("err = %v", err)
	}
	var pid int
	if _, err := fmt.Sscanf(stderr.String(), "child %d", &pid); err != nil {
		t.Fatalf("no child pid in %q", stderr.String())
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		p, err := os.FindProcess(pid)
		if err != nil || p.Signal(syscall.Signal(0)) != nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("child %d outlived the timeout", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	go func() { time.Sleep(200 * time.Millisecond); cancel(errors.New("interrupted")) }()
	r := Runner{Dir: t.TempDir(), Project: "cli", Version: "edge"}
	_, err := r.Run(ctx, []string{echo, "sleep"}, "test/v1")
	if err == nil || !strings.Contains(err.Error(), "canceled (interrupted)") {
		t.Fatalf("err = %v", err)
	}
}
