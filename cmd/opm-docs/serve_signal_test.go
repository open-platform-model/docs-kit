//go:build unix

package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/open-platform-model/docs-kit/internal/gittest"
)

// TestMain runs opm-docs itself when a test starts this binary as a child
// with OPM_DOCS_TEST_MAIN set, so a test can signal a real process.
func TestMain(m *testing.M) {
	if os.Getenv("OPM_DOCS_TEST_MAIN") != "" {
		os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
	}
	os.Exit(m.Run())
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// serveChild is one opm-docs serve process.
type serveChild struct {
	cmd    *exec.Cmd
	stderr *lockedBuffer
	done   chan error
}

func startServe(t *testing.T, repo, tmp string, port int) *serveChild {
	t.Helper()
	c := &serveChild{stderr: &lockedBuffer{}, done: make(chan error, 1)}
	c.cmd = exec.CommandContext(context.Background(), os.Args[0], "serve", "--port", strconv.Itoa(port))
	c.cmd.Dir = repo
	c.cmd.Env = append(os.Environ(), "OPM_DOCS_TEST_MAIN=1", "TMPDIR="+tmp, "GITHUB_REPOSITORY=")
	c.cmd.Stdout, c.cmd.Stderr = c.stderr, c.stderr
	if err := c.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { c.done <- c.cmd.Wait() }()
	deadline := time.Now().Add(60 * time.Second)
	for !strings.Contains(c.stderr.String(), "opm-docs serve: cli at ") {
		select {
		case err := <-c.done:
			t.Fatalf("serve exited before serving: %v\n%s", err, c.stderr.String())
		default:
		}
		if time.Now().After(deadline) {
			_ = c.cmd.Process.Kill()
			t.Fatalf("serve never printed its URL:\n%s", c.stderr.String())
		}
		time.Sleep(100 * time.Millisecond)
	}
	return c
}

func (c *serveChild) wait(t *testing.T) error {
	t.Helper()
	select {
	case err := <-c.done:
		return err
	case <-time.After(30 * time.Second):
		_ = c.cmd.Process.Kill()
		t.Fatalf("serve did not stop:\n%s", c.stderr.String())
		return nil
	}
}

// portClosed waits until nothing listens on port: hugo is gone.
func portClosed(t *testing.T, port int) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		conn, err := (&net.Dialer{Timeout: time.Second}).DialContext(context.Background(), "tcp", "127.0.0.1:"+strconv.Itoa(port))
		if err != nil {
			return
		}
		_ = conn.Close()
		if time.Now().After(deadline) {
			t.Fatalf("something still listens on %d: hugo outlived serve", port)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// serveDirs lists the serve directories under tmp.
func serveDirs(t *testing.T, tmp string) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(tmp, "opm-docs-serve-*"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// TestServeSignals stops a real serve with SIGTERM and SIGHUP, and on
// Linux kills one: hugo must never outlive it, and its temporary tree
// must be gone (after a kill, swept by the next serve).
func TestServeSignals(t *testing.T) {
	if _, err := exec.LookPath("hugo"); err != nil {
		if os.Getenv("OPM_DOCS_REQUIRE_HUGO") != "" {
			t.Fatal("OPM_DOCS_REQUIRE_HUGO is set and there is no hugo on PATH")
		}
		t.Skip("no hugo on PATH")
	}
	r := gittest.New(t, "https://github.com/example/cli.git")
	r.Write(map[string]string{
		"docs-kit.cue":               serveConfig,
		"docs/site/start/install.md": "---\ntitle: \"Install\"\ndescription: \"Install it.\"\ntype: how-to\n---\n\nWords.\n",
	})
	r.Commit("docs")
	tmp := t.TempDir()

	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGHUP} {
		port := freeTCPPort(t)
		c := startServe(t, r.Dir, tmp, port)
		if err := c.cmd.Process.Signal(sig); err != nil {
			t.Fatal(err)
		}
		if err := c.wait(t); err != nil {
			t.Fatalf("%v: serve exited with %v\n%s", sig, err, c.stderr.String())
		}
		portClosed(t, port)
		if d := serveDirs(t, tmp); len(d) != 0 {
			t.Fatalf("%v: serve left %v", sig, d)
		}
	}

	if runtime.GOOS != "linux" {
		return // only Linux has the parent-death signal
	}
	port := freeTCPPort(t)
	c := startServe(t, r.Dir, tmp, port)
	if err := c.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	var exit *exec.ExitError
	if err := c.wait(t); !errors.As(err, &exit) {
		t.Fatalf("killed serve: %v", err)
	}
	portClosed(t, port)
	killed := serveDirs(t, tmp)
	if len(killed) != 1 {
		t.Fatalf("a killed serve's tree: %v", killed)
	}
	port = freeTCPPort(t)
	c = startServe(t, r.Dir, tmp, port)
	if _, err := os.Stat(killed[0]); err == nil {
		t.Fatalf("the next serve did not sweep the killed one's tree %s", killed[0])
	}
	if err := c.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := c.wait(t); err != nil {
		t.Fatal(err)
	}
	if d := serveDirs(t, tmp); len(d) != 0 {
		t.Fatalf("serve left %v", d)
	}
}
