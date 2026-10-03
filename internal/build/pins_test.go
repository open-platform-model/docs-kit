package build

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/gittest"
)

const pinsConfig = `bundles: cli: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/cli/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{kind: "markdown", dir: "docs/site"}]
	pins: {
		command: ["cat", "pins.json"]
		projects: ["library", "core", "opm-operator"]
	}
}
`

func pinsBuild(t *testing.T, pins string) (*bundle.Manifest, error) {
	t.Helper()
	t.Setenv("GITHUB_REPOSITORY", "")
	r := gittest.New(t, "https://github.com/example/cli.git")
	r.Write(map[string]string{
		"docs-kit.cue":       pinsConfig,
		"pins.json":          pins,
		"docs/site/guide.md": authoredOps,
	})
	r.Commit("cli")
	out := filepath.Join(t.TempDir(), "out")
	if _, err := Run(context.Background(), Options{Source: r.Dir, Out: out, Tool: "0.1.0", Check: true}); err != nil {
		return nil, err
	}
	return bundle.Read(filepath.Join(out, "cli"))
}

func TestPins(t *testing.T) {
	m, err := pinsBuild(t, `{"schema": "docs.opmodel.dev/pins/v1", "pins": {"library": "1.0.0-beta.1", "core": "2.0.0-beta.1", "opm-operator": "1.0.0-beta.4"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Pins) != 3 || m.Pins["core"] != "2.0.0-beta.1" || m.Pins["opm-operator"] != "1.0.0-beta.4" {
		t.Fatalf("pins %v", m.Pins)
	}
	for _, c := range []struct{ name, doc, want string }{
		{"missing", `{"schema": "docs.opmodel.dev/pins/v1", "pins": {"library": "1.0.0", "opm-operator": "1.0.0"}}`, "no pin for core"},
		{"extra", `{"schema": "docs.opmodel.dev/pins/v1", "pins": {"library": "1.0.0", "core": "1.0.0", "opm-operator": "1.0.0", "cli": "1.0.0"}}`, "pins cli, which pins.projects"},
		{"v prefix", `{"schema": "docs.opmodel.dev/pins/v1", "pins": {"library": "v1.0.0", "core": "1.0.0", "opm-operator": "1.0.0"}}`, `library pins "v1.0.0"`},
		{"wrong schema", `{"schema": "docs.opmodel.dev/pins/v2", "pins": {}}`, `"docs.opmodel.dev/pins/v2"`},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := pinsBuild(t, c.doc)
			var ue *UsageError
			if err == nil || errors.As(err, &ue) || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), "cat pins.json") {
				t.Fatalf("err = %v, want an execution error naming %q and the command", err, c.want)
			}
		})
	}
}

func TestNoPinsWritesNone(t *testing.T) {
	m, err := bundle.Parse("manifest.json", releaseBuild(t, fixtureRepo(t))["manifest.json"])
	if err != nil {
		t.Fatal(err)
	}
	if m.Pins != nil {
		t.Fatalf("pins %v", m.Pins)
	}
}
