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
	"github.com/open-platform-model/docs-kit/internal/gittest"
)

const authoredDefsIndex = "---\ntitle: \"Definitions\"\ndescription: \"Authored.\"\nweight: 1\n---\n\nAn authored introduction.\n"

// defsRepo is a repository holding the cue-definitions fixture package and
// its docs-kit.cue, plus an authored definitions index under docs/site.
func defsRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	t.Setenv("GITHUB_REPOSITORY", "")
	r := gittest.New(t, "https://github.com/example/defs.git")
	r.CopyTree("../extract/cuedefs/testdata/defs/src", "src")
	cfg, err := os.ReadFile("../extract/cuedefs/testdata/defs/config.cue")
	if err != nil {
		t.Fatal(err)
	}
	r.Write(map[string]string{
		"docs-kit.cue": strings.Replace(string(cfg), "\t}]\n}", "\t}, {kind: \"markdown\", dir: \"docs/site\"}]\n}", 1),
		"docs/site/reference/definitions/_index.md": authoredDefsIndex,
	})
	r.Commit("defs")
	return r
}

func TestBuildDefinitions(t *testing.T) {
	r := defsRepo(t)
	out := filepath.Join(t.TempDir(), "out")
	if _, err := Run(context.Background(), Options{Source: r.Dir, Out: out, Tool: "0.1.0"}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(out, "defs")
	m, err := bundle.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Data) != 1 || m.Data[0].Path != "cue-definitions.json" || m.Data[0].Schema != "docs.opmodel.dev/data/cue-definitions/v1" {
		t.Errorf("data %+v", m.Data)
	}
	if p := pageEntry(m, "reference/definitions/modules.md"); !p.Generated {
		t.Errorf("modules page %+v", p)
	}
	// The authored index is completed by the generated tail.
	index, _ := os.ReadFile(filepath.Join(dir, "content", "reference", "definitions", "_index.md"))
	if !strings.HasPrefix(string(index), authoredDefsIndex+"\n## Pages\n") {
		t.Errorf("index:\n%s", index)
	}
	if p := pageEntry(m, "reference/definitions/_index.md"); p.Generated || p.Source != "docs/site/reference/definitions/_index.md" {
		t.Errorf("index entry %+v", p)
	}
}

// A definition the config neither places nor excludes fails a normal
// build, naming the definition and its file.
func TestBuildDefinitionsUnplaced(t *testing.T) {
	r := defsRepo(t)
	r.Write(map[string]string{"src/policy.cue": "package defs\n\n// #Policy is new.\n#Policy: string\n"})
	r.Commit("policy")
	_, err := Run(context.Background(), Options{Source: r.Dir, Out: filepath.Join(t.TempDir(), "out"), Tool: "0.1.0", Check: true})
	var ue *UsageError
	if err == nil || errors.As(err, &ue) || !strings.Contains(err.Error(), "#Policy (src/policy.cue) is exported but neither placed") {
		t.Fatalf("err = %v, want an execution error naming #Policy", err)
	}
}

// A backfill (config from outside the tree) that places a definition the
// tree lacks warns and builds.
func TestBuildDefinitionsBackfill(t *testing.T) {
	r := defsRepo(t)
	cfg, err := os.ReadFile(filepath.Join(r.Dir, "docs-kit.cue"))
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "docs-kit.cue")
	if err := os.WriteFile(outside, bytes.Replace(cfg, []byte(`"#Mode"]`), []byte(`"#Mode", "#Policy"]`), 1), 0o600); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	_, err = Run(context.Background(), Options{Source: r.Dir, Config: outside, Out: filepath.Join(t.TempDir(), "out"), Tool: "0.1.0", Stderr: &stderr})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "warning: cue-definitions ./src: #Policy is placed on page \"types\"") {
		t.Fatalf("stderr %q", stderr.String())
	}
}
