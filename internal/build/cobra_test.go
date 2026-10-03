package build

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/command"
	"github.com/open-platform-model/docs-kit/internal/gittest"
)

// cobraConfig is the cli's planned file with the dump read from a file the
// stub command prints.
const cobraConfig = `bundles: cli: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/cli/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{
		kind:        "cobra"
		command:     ["cat", "dump.json"]
		section:     "reference/cli/"
		title:       "CLI Reference"
		description: "Every demo command and flag."
		weight:      2
	}, {
		kind: "markdown", dir: "docs/site", exclude: ["reference/cli/"]
	}]
}
`

func cobraBuild(t *testing.T, dump string) (string, error) {
	t.Helper()
	t.Setenv("GITHUB_REPOSITORY", "")
	r := gittest.New(t, "https://github.com/example/cli.git")
	r.Write(map[string]string{
		"docs-kit.cue":                     cobraConfig,
		"dump.json":                        dump,
		"docs/site/guide.md":               authoredOps,
		"docs/site/reference/cli/stale.md": authoredOps,
	})
	r.Commit("cli")
	out := filepath.Join(t.TempDir(), "out")
	_, err := Run(context.Background(), Options{Source: r.Dir, Out: out, Tool: "0.1.0", Check: true})
	return filepath.Join(out, "cli"), err
}

func goldenDump(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../../cobradump/testdata/dump.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCobraSource(t *testing.T) {
	dir, err := cobraBuild(t, goldenDump(t))
	if err != nil {
		t.Fatal(err)
	}
	m, err := bundle.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, len(m.Pages))
	for _, p := range m.Pages {
		paths = append(paths, p.Path)
		if strings.HasPrefix(p.Path, "reference/cli/") && (!p.Generated || p.Source != "") {
			t.Errorf("%s: generated %v, source %q", p.Path, p.Generated, p.Source)
		}
	}
	if got := strings.Join(paths, ","); got != "guide.md,reference/cli/_index.md,reference/cli/demo-completion.md,reference/cli/demo-version.md,reference/cli/demo-widget.md" {
		t.Fatalf("pages = %s", got)
	}
	if len(m.Data) != 1 || m.Data[0].Path != "cobra.json" || m.Data[0].Schema != "docs.opmodel.dev/data/cobra/v1" {
		t.Fatalf("data = %+v", m.Data)
	}
	want, _ := os.ReadFile("../render/testdata/golden/cobra/reference/cli/demo-widget.md")
	got, _ := os.ReadFile(filepath.Join(dir, "content", "reference", "cli", "demo-widget.md"))
	if !bytes.Equal(got, want) {
		t.Fatal("the built page differs from the renderer's golden page")
	}
}

func TestCobraUnknownDumpSchema(t *testing.T) {
	_, err := cobraBuild(t, strings.Replace(goldenDump(t), "docs.opmodel.dev/cobradump/v1", "docs.opmodel.dev/cobradump/v2", 1))
	var ce *command.Error
	var ue *UsageError
	if !errors.As(err, &ce) || errors.As(err, &ue) ||
		!strings.Contains(err.Error(), "cli: command `cat dump.json`") ||
		!strings.Contains(err.Error(), `"docs.opmodel.dev/cobradump/v2"`) ||
		!strings.Contains(err.Error(), `"docs.opmodel.dev/cobradump/v1"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestCobraBadDump(t *testing.T) {
	_, err := cobraBuild(t, strings.Replace(goldenDump(t), `"path": "demo widget create"`, `"path": "demo create"`, 1))
	if err == nil || !strings.Contains(err.Error(), "cli: command `cat dump.json`: command \"demo create\"") {
		t.Fatalf("err = %v", err)
	}
}

func TestCobraSectionOutsideOwns(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "")
	r := gittest.New(t, "https://github.com/example/cli.git")
	r.Write(map[string]string{
		"docs-kit.cue":                     strings.Replace(cobraConfig, `section:     "reference/cli/"`, `section:     "reference/commands/"`, 1),
		"dump.json":                        goldenDump(t),
		"docs/site/reference/cli/stale.md": authoredOps,
	})
	r.Commit("cli")
	_, err := Run(context.Background(), Options{Source: r.Dir, Out: filepath.Join(t.TempDir(), "out"), Tool: "0.1.0"})
	if err == nil || !strings.Contains(err.Error(), "content/reference/commands/_index.md is generated, but cli owns only reference/cli/") {
		t.Fatalf("err = %v", err)
	}
}
