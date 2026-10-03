package serve

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MinHugo is the oldest Hugo the skeleton runs on: 0.146.0 introduced the
// template layout it uses (layouts/_markup, layouts/_shortcodes).
var MinHugo = [3]int{0, 146, 0}

// UsageError is a mistake the author fixes before serve can run: a flag,
// a missing or too old program.
type UsageError struct{ Err error }

func (e *UsageError) Error() string { return e.Err.Error() }
func (e *UsageError) Unwrap() error { return e.Err }

func usage(format string, args ...any) error {
	return &UsageError{fmt.Errorf(format, args...)}
}

// lookPath finds a program on PATH; tests replace it.
var lookPath = defaultLookPath

var defaultLookPath = exec.LookPath

// findHugo returns the path of hugo on PATH, refusing a missing one or one
// older than MinHugo, naming the version found.
func findHugo(ctx context.Context) (string, error) {
	path, err := lookPath("hugo")
	if err != nil {
		return "", usage("no hugo on PATH: install Hugo %s or later (https://gohugo.io/installation/), or preview through the site with --site", minHugo())
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "version").Output()
	if err != nil {
		return "", usage("%s version: %v; install Hugo %s or later", path, err, minHugo())
	}
	v, ok := parseHugoVersion(string(out))
	if !ok {
		return "", usage("%s version printed %q, which names no version; install Hugo %s or later", path, strings.TrimSpace(string(out)), minHugo())
	}
	if older(v, MinHugo) {
		return "", usage("%s is Hugo %d.%d.%d; serve needs %s or later", path, v[0], v[1], v[2], minHugo())
	}
	return path, nil
}

var reHugoVersion = regexp.MustCompile(`\bv(\d+)\.(\d+)\.(\d+)`)

// parseHugoVersion reads the version from `hugo version` output, such as
// "hugo v0.167.0-3fff6fb5 linux/amd64 BuildDate=...".
func parseHugoVersion(out string) ([3]int, bool) {
	m := reHugoVersion.FindStringSubmatch(out)
	if m == nil {
		return [3]int{}, false
	}
	var v [3]int
	for i := range v {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return [3]int{}, false
		}
		v[i] = n
	}
	return v, true
}

func older(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func minHugo() string {
	return fmt.Sprintf("%d.%d.%d", MinHugo[0], MinHugo[1], MinHugo[2])
}

// runChild runs cmd until it exits or ctx ends. On ctx's end the child gets
// an interrupt (the terminal's Ctrl-C reaches it anyway, as it shares the
// foreground process group) and up to ten seconds to stop. It reports
// whether ctx ended, which callers treat as the author stopping serve.
func runChild(ctx context.Context, cmd *exec.Cmd) (stopped bool, err error) {
	dieWithParent(cmd)
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = 10 * time.Second
	err = cmd.Run()
	if ctx.Err() != nil {
		// The author stopped serve; the interrupted child's exit status
		// is no failure.
		return true, nil //nolint:nilerr // see above
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return false, fmt.Errorf("`%s` exited with status %d", strings.Join(cmd.Args, " "), exit.ExitCode())
	}
	if err != nil {
		return false, fmt.Errorf("`%s`: %w", strings.Join(cmd.Args, " "), err)
	}
	return false, nil
}
