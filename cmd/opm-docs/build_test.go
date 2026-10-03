package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/gittest"
)

func demoRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	t.Setenv("GITHUB_REPOSITORY", "")
	t.Setenv("GITHUB_REF_TYPE", "")
	r := gittest.New(t, "https://github.com/example/demo.git")
	r.CopyTree("../../internal/extract/cuecatalog/testdata/catalog", ".")
	r.Write(map[string]string{
		"docs-kit.cue": `bundles: "catalog-demo": {
	placement: {kind: "tab", root: "/catalogs/demo/"}
	version: {from: "tag", prefix: "demo-v"}
	sources: [{kind: "cue-catalog", module: "./demo"}]
}
`,
		".gitignore": "/out/\n",
	})
	r.Commit("catalog")
	return r
}

func TestBuildAndCheck(t *testing.T) {
	r := demoRepo(t)
	t.Chdir(r.Dir)
	var out, errb bytes.Buffer
	if code := run([]string{"build", "--nope"}, &out, &errb); code != 1 || !strings.Contains(errb.String(), "--nope") {
		t.Fatalf("build --nope: exit %d: %s", code, errb.String())
	}
	errb.Reset()
	if code := run([]string{"build"}, &out, &errb); code != 0 {
		t.Fatalf("build: exit %d: %s", code, errb.String())
	}
	if _, err := os.Stat(filepath.Join("out", "catalog-demo", "manifest.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll("out"); err != nil {
		t.Fatal(err)
	}
	errb.Reset()
	if code := run([]string{"check"}, &out, &errb); code != 0 {
		t.Fatalf("check: exit %d: %s", code, errb.String())
	}
	if _, err := os.Stat("out"); !os.IsNotExist(err) {
		t.Fatal("check wrote out/ in the work tree")
	}
	if st := r.Git("status", "--porcelain"); st != "" {
		t.Fatalf("check changed the work tree:\n%s", st)
	}
	p := filepath.Join("demo", "resources", "v1", "queue.cue")
	b, _ := os.ReadFile(p)
	_ = os.WriteFile(p, []byte(strings.Replace(string(b), "// A message queue a component declares.", "// Something else.", 1)), 0o600)
	errb.Reset()
	if code := run([]string{"check"}, &out, &errb); code != 2 || !strings.Contains(errb.String(), "resources/queue@v1") {
		t.Fatalf("check with a bad doc comment: exit %d: %s", code, errb.String())
	}
	errb.Reset()
	if code := run([]string{"build", "--release", "v1.2.3"}, &out, &errb); code != 1 || !strings.Contains(errb.String(), `"demo-v"`) {
		t.Fatalf("wrong prefix: exit %d: %s", code, errb.String())
	}
}

func TestBuildRefusesTagVersionMismatch(t *testing.T) {
	r := demoRepo(t)
	t.Chdir(r.Dir)
	r.Git("tag", "demo-v1.2.4")
	var out, errb bytes.Buffer
	if code := run([]string{"build", "--release", "demo-v1.2.4"}, &out, &errb); code != 2 || !strings.Contains(errb.String(), "version 1.2.4") || !strings.Contains(errb.String(), `"1.2.3"`) {
		t.Fatalf("tag and catalog version differ: exit %d: %s", code, errb.String())
	}
}
