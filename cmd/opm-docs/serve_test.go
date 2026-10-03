package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const serveConfig = `bundles: cli: {
	placement: {kind: "docs", root: "/docs/"}
	version: {from: "tag", prefix: "v"}
	sources: [{kind: "markdown", dir: "docs/site"}]
}
`

func TestServeUsage(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "docs-kit.cue")
	if err := os.WriteFile(cfg, []byte(serveConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir()) // no hugo, no task
	cases := []struct {
		name   string
		args   []string
		stderr string
	}{
		{"unknown flag", []string{"serve", "--nope"}, "--nope"},
		{"no hugo", []string{"serve", "--config", cfg}, "no hugo on PATH"},
		{"bad port", []string{"serve", "--config", cfg, "--port", "0"}, "--port 0"},
		{"version without site", []string{"serve", "--config", cfg, "--site-version", "v1.0"}, "--site-version v1.0 needs --site"},
		{"docs bundle without --site-version", []string{"serve", "--config", cfg, "--site", t.TempDir()}, "cli is a docs bundle: pass --site-version"},
		{"site without task", []string{"serve", "--config", cfg, "--site", t.TempDir(), "--site-version", "v1.0"}, "--site needs task"},
		{"port with site", []string{"serve", "--config", cfg, "--site", t.TempDir(), "--port", "8080"}, "--port with --site"},
		{"unknown project", []string{"serve", "--config", cfg, "--project", "core"}, "--project core"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			code := run(c.args, &out, &errb)
			if code != exitUsage || !strings.Contains(errb.String(), c.stderr) {
				t.Fatalf("exit %d (want 1)\nstderr: %s", code, errb.String())
			}
		})
	}
}
