// Package command runs a repository command for a build: a program the
// repository names in docs-kit.cue (an argv list, no shell) that prints one
// JSON document on stdout. The rules are docs-kit C14.
package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Limits of a command run.
const (
	DefaultTimeout   = 10 * time.Minute
	DefaultMaxOutput = 16 << 20 // bytes of stdout
)

// Runner runs the repository commands of one bundle's build.
type Runner struct {
	Dir     string // the source tree the command runs in
	Project string // OPM_DOCS_PROJECT
	Version string // OPM_DOCS_VERSION: "4.4.5" or "edge"
	// Twice runs every command twice and refuses differing outputs (check).
	Twice     bool
	Stderr    io.Writer     // the command's stderr; nil is os.Stderr
	Timeout   time.Duration // 0 is DefaultTimeout
	MaxOutput int           // 0 is DefaultMaxOutput
}

// Environ keeps, of env, the variables a repository command may see: PATH,
// HOME, TMPDIR, USER, LANG and LC_*, the proxy variables, Go's own
// variables (GO followed by capitals and digits only, such as GOPATH,
// GOFLAGS, GOPROXY, GOTOOLCHAIN; never GOOGLE_*), CGO_*, CUE_* and
// OPM_DOCS*. Nothing else passes, so a token in the
// build's environment (GITHUB_TOKEN, ACTIONS_*, a registry credential)
// never reaches the command.
func Environ(env []string) []string {
	var out []string
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if allowed(name) {
			out = append(out, kv)
		}
	}
	return out
}

var allowedNames = map[string]bool{
	"PATH": true, "HOME": true, "TMPDIR": true, "USER": true, "LANG": true,
	"HTTP_PROXY": true, "HTTPS_PROXY": true, "NO_PROXY": true,
	"http_proxy": true, "https_proxy": true, "no_proxy": true,
}

func allowed(name string) bool {
	if allowedNames[name] {
		return true
	}
	if reGoVar.MatchString(name) {
		return true
	}
	for _, p := range []string{"CGO_", "CUE_", "OPM_DOCS", "LC_"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

var reGoVar = regexp.MustCompile(`^GO[A-Z0-9]+$`)

// Error is a command that failed, ran too long or printed something other
// than one JSON document of the expected schema.
type Error struct {
	Argv    []string
	Project string
	Msg     string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: command `%s`: %s", e.Project, strings.Join(e.Argv, " "), e.Msg)
}

// Run runs argv and returns its stdout, one JSON document whose "schema"
// field is schema.
func (r *Runner) Run(ctx context.Context, argv []string, schema string) ([]byte, error) {
	if len(argv) == 0 {
		return nil, &Error{Argv: argv, Project: r.Project, Msg: "empty argv"}
	}
	out, err := r.once(ctx, argv, schema)
	if err != nil || !r.Twice {
		return out, err
	}
	again, err := r.once(ctx, argv, schema)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(out, again) {
		return nil, r.fail(argv, "two runs printed different output; a docs build must be deterministic (see docs-kit C14)")
	}
	return out, nil
}

func (r *Runner) fail(argv []string, format string, args ...any) *Error {
	return &Error{Argv: argv, Project: r.Project, Msg: fmt.Sprintf(format, args...)}
}

func (r *Runner) once(ctx context.Context, argv []string, schema string) ([]byte, error) {
	timeout := r.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	limit := r.MaxOutput
	if limit == 0 {
		limit = DefaultMaxOutput
	}
	parent := ctx
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...) //nolint:gosec // the repository's own command, by design (docs-kit C14)
	cmd.Dir = r.Dir
	cmd.Env = append(Environ(os.Environ()), "OPM_DOCS=1", "OPM_DOCS_PROJECT="+r.Project, "OPM_DOCS_VERSION="+r.Version)
	ownGroup(cmd)
	cmd.Stdin = nil
	cmd.Stderr = r.Stderr
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}
	stdout := &capped{limit: limit}
	cmd.Stdout = stdout
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	if perr := parent.Err(); perr != nil {
		return nil, r.fail(argv, "stopped: the build was canceled (%v)", context.Cause(parent))
	}
	if ctx.Err() == context.DeadlineExceeded {
		return nil, r.fail(argv, "ran over %s and was stopped", timeout)
	}
	if stdout.over {
		return nil, r.fail(argv, "printed more than %d bytes on stdout", limit)
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return nil, r.fail(argv, "exited with status %d", exit.ExitCode())
	}
	if err != nil {
		return nil, r.fail(argv, "%v", err)
	}
	return r.document(argv, stdout.Bytes(), schema)
}

// document checks that out is exactly one JSON object carrying schema.
func (r *Runner) document(argv []string, out []byte, schema string) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(out))
	var head struct {
		Schema string `json:"schema"`
	}
	if err := dec.Decode(&head); err != nil {
		return nil, r.fail(argv, "stdout must hold exactly one JSON document: %v", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, r.fail(argv, "stdout must hold exactly one JSON document, and more follows it; print logs on stderr")
	}
	if head.Schema != schema {
		return nil, r.fail(argv, "printed a document of schema %q; this build reads %q", head.Schema, schema)
	}
	return out, nil
}

// capped is a buffer that stops taking bytes past its limit. It holds the
// buffer in a field, not embedded, so io.Copy cannot bypass Write through
// the buffer's ReadFrom.
type capped struct {
	buf   bytes.Buffer
	limit int
	over  bool
}

func (c *capped) Write(p []byte) (int, error) {
	if c.buf.Len()+len(p) > c.limit {
		c.over = true
		return 0, errors.New("stdout over its limit")
	}
	return c.buf.Write(p)
}

func (c *capped) Bytes() []byte { return c.buf.Bytes() }
