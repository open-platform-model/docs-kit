package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExitCodes(t *testing.T) {
	const conf = "../../internal/dialect/testdata/conformance/"
	cases := []struct {
		name   string
		args   []string
		code   int
		stdout string
		stderr string
	}{
		{"version", []string{"version"}, 0, "opm-docs ", ""},
		{"unknown command", []string{"nope"}, 1, "", `unknown command "nope"`},
		{"unknown flag", []string{"lint", "--nope", "x"}, 1, "", "--nope"},
		{"lint without a dir", []string{"lint"}, 1, "", "requires at least 1 arg"},
		{"unsupported dialect", []string{"lint", "--dialect", "2", conf + "clean/core/docs/site"}, 1, "", "dialect 1 only"},
		{"clean tree", []string{"lint", conf + "clean/core/docs/site"}, 0, "", "OK"},
		{"violations", []string{"lint", conf + "link-md/core/docs/site"}, 2,
			`md-link.md:7: internal link "/docs/start/quickstart.md": write /docs/<section>/<page>/ with a trailing slash`, "2 violation(s)"},
		{"missing dir", []string{"lint", "does-not-exist"}, 1, "", "not a directory"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			code := run(c.args, &out, &errb)
			if code != c.code || !strings.Contains(out.String(), c.stdout) || !strings.Contains(errb.String(), c.stderr) {
				t.Fatalf("exit %d (want %d)\nstdout: %s\nstderr: %s", code, c.code, out.String(), errb.String())
			}
		})
	}
}
