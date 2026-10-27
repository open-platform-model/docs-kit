package build

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/gittest"
)

const unitsConfig = `bundles: "opm-controller": {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/cli/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{kind: "markdown", dir: "docs/site"}]
	units: "opm-controller-module": key: "modules/opm_controller"
}
`

// unitsBuild builds a repository holding config and a release manifest with
// the given contents; a manifest of "" is left out.
func unitsBuild(t *testing.T, config, manifest string) (string, error) {
	t.Helper()
	t.Setenv("GITHUB_REPOSITORY", "")
	r := gittest.New(t, "https://github.com/example/opm-controller.git")
	files := map[string]string{"docs-kit.cue": config, "docs/site/guide.md": authoredOps}
	if manifest != "" {
		files[".release-please-manifest.json"] = manifest
	}
	r.Write(files)
	r.Commit("controller")
	out := filepath.Join(t.TempDir(), "out")
	_, err := Run(context.Background(), Options{Source: r.Dir, Out: out, Tool: "0.1.0", Check: true})
	return filepath.Join(out, "opm-controller"), err
}

func TestUnitsRecorded(t *testing.T) {
	dir, err := unitsBuild(t, unitsConfig, `{".": "1.0.0-beta.10", "modules/opm_controller": "0.1.1"}`)
	if err != nil {
		t.Fatal(err)
	}
	m, err := bundle.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Units) != 1 || m.Units["opm-controller-module"] != "0.1.1" {
		t.Fatalf("units %v", m.Units)
	}
	raw, err := os.ReadFile(filepath.Join(dir, bundle.ManifestFile))
	if err != nil || !strings.Contains(string(raw), `"units": {`) {
		t.Fatalf("manifest.json: %v\n%s", err, raw)
	}
}

func TestUnitsRefused(t *testing.T) {
	for _, c := range []struct{ name, manifest, want string }{
		{"missing file", "", "cannot read .release-please-manifest.json"},
		{"missing key", `{".": "1.0.0"}`, `has no key "modules/opm_controller"`},
		{"v prefix", `{"modules/opm_controller": "v0.1.1"}`, "not an exact version"},
		{"number", `{"modules/opm_controller": 1}`, "not an exact version"},
		{"not an object", `["0.1.1"]`, "not a JSON object"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := unitsBuild(t, unitsConfig, c.manifest)
			if err == nil || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), ".release-please-manifest.json") {
				t.Fatalf("error %v, want %q naming the file", err, c.want)
			}
		})
	}
}

func TestUnitsFileNamed(t *testing.T) {
	cfg := strings.Replace(unitsConfig, `key: "modules/opm_controller"`, `{file: "release/versions.json", key: "ctl"}`, 1)
	t.Setenv("GITHUB_REPOSITORY", "")
	r := gittest.New(t, "https://github.com/example/opm-controller.git")
	r.Write(map[string]string{"docs-kit.cue": cfg, "docs/site/guide.md": authoredOps, "release/versions.json": `{"ctl": "2.0.0"}`})
	r.Commit("controller")
	out := filepath.Join(t.TempDir(), "out")
	if _, err := Run(context.Background(), Options{Source: r.Dir, Out: out, Tool: "0.1.0", Check: true}); err != nil {
		t.Fatal(err)
	}
	m, err := bundle.Read(filepath.Join(out, "opm-controller"))
	if err != nil || m.Units["opm-controller-module"] != "2.0.0" {
		t.Fatalf("units %v, %v", m.Units, err)
	}
}

func TestNoUnitsNoKey(t *testing.T) {
	cfg := strings.Replace(unitsConfig, "\tunits: \"opm-controller-module\": key: \"modules/opm_controller\"\n", "", 1)
	dir, err := unitsBuild(t, cfg, `{"modules/opm_controller": "0.1.1"}`)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, bundle.ManifestFile))
	if err != nil || strings.Contains(string(raw), "units") {
		t.Fatalf("manifest.json holds units without a declaration: %v\n%s", err, raw)
	}
}
