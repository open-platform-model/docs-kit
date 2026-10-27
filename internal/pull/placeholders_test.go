package pull

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
)

const phExtra = "\nInstall v{{opm:cli.version}} with `opm install --version v{{opm:opm-controller.version}}`.\n\n```bash\nopm get {{opm:core.version}}\n```\n\n[notes](https://example.com/v{{opm:cli.version}}) `<digest>`\n"

const phFilled = "\nInstall v1.0.0-beta.6 with `opm install --version v1.0.0-beta.4`.\n\n```bash\nopm get 2.0.0-beta.1\n```\n\n[notes](https://example.com/v1.0.0-beta.6) `<digest>`\n"

// publishPlaceholders publishes the v1.0 set with a page of the cli bundle
// that holds placeholders and the controller bundle recording a unit.
func (e *env) publishPlaceholders(units map[string]string) {
	e.t.Helper()
	e.publishDocs(coreSpec("2.0.0-beta.1"))
	ctl := controllerSpec()
	ctl.units = units
	e.publishDocs(ctl)
	cli := cliSpec("1.0.0-beta.6", map[string]string{"core": "2.0.0-beta.1", "opm-controller": "1.0.0-beta.4"})
	cli.extra = phExtra
	e.publishDocs(cli)
}

func TestFillPerSiteVersion(t *testing.T) {
	e := newEnv(t)
	e.publishPlaceholders(nil)
	o := e.options(e.docsConfig(v10))
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	got := readPage(t, o.Out, "_versions/v1.0/cli/content/reference/cli/opm-module.md")
	if !strings.HasSuffix(got, "Body of cli 1.0.0-beta.6.\n"+phFilled) {
		t.Fatalf("filled page:\n%s", got)
	}
	// The front matter and the other pages are as published.
	if !strings.HasPrefix(got, "---\ntitle: \"cli reference/cli/opm-module.md\"\n") {
		t.Fatalf("front matter changed:\n%s", got)
	}
	if other := readPage(t, o.Out, "_versions/v1.0/core/content/concepts/module.md"); strings.Contains(other, "{{") {
		t.Fatalf("another bundle changed:\n%s", other)
	}
}

// Two site versions pulled together each get the values of their own bundles.
func TestFillKeepsEachSiteVersionsNumbers(t *testing.T) {
	e := newEnv(t)
	e.publishDocs(coreSpec("2.0.0-beta.1"))
	e.publishDocs(controllerSpec())
	for _, v := range []string{"1.0.2", "1.1.0"} {
		cli := cliSpec(v, nil)
		cli.extra = "\nv{{opm:cli.version}}\n"
		e.publishDocs(cli)
		e.retag("cli", v, strings.Join(strings.Split(v, ".")[:2], "."))
	}
	o := e.options(e.docsConfig(`versions: {
	"v1.0": {anchor: {project: "cli", tag: "1.0"}}
	"v1.1": {anchor: {project: "cli", tag: "1.1"}}
}
`))
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	for sv, want := range map[string]string{"v1.0": "\nv1.0.2\n", "v1.1": "\nv1.1.0\n"} {
		if got := readPage(t, o.Out, "_versions/"+sv+"/cli/content/reference/cli/opm-module.md"); !strings.HasSuffix(got, want) {
			t.Errorf("%s page:\n%s", sv, got)
		}
	}
}

func TestFillUnitFromAManifest(t *testing.T) {
	e := newEnv(t)
	e.publishDocs(coreSpec("2.0.0-beta.1"))
	ctl := controllerSpec()
	ctl.units = map[string]string{"opm-controller-module": "0.1.1"}
	e.publishDocs(ctl)
	cli := cliSpec("1.0.0-beta.6", map[string]string{"core": "2.0.0-beta.1", "opm-controller": "1.0.0-beta.4"})
	cli.extra = "\nmodule {{opm:opm-controller-module.version}}\n"
	e.publishDocs(cli)
	o := e.options(e.docsConfig(v10))
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if got := readPage(t, o.Out, "_versions/v1.0/cli/content/reference/cli/opm-module.md"); !strings.HasSuffix(got, "\nmodule 0.1.1\n") {
		t.Fatalf("page:\n%s", got)
	}
}

// A value the site version cannot supply refuses the pull and leaves the
// previous trees and lock as they were.
func TestFillRefusals(t *testing.T) {
	for _, c := range []struct {
		name  string
		extra string
		want  []string
	}{
		{"no bundle for the name", "\n{{opm:library.version}}\n",
			[]string{"v1.0: cli 1.0.0-beta.7: reference/cli/_index.md", `"{{opm:library.version}}"`, "pulls no library bundle"}},
		{"no bundle records the unit", "\n{{opm:opm-controller-module.version}}\n",
			[]string{"v1.0", "reference/cli/_index.md", `"{{opm:opm-controller-module.version}}"`, "records the unit opm-controller-module"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t)
			e.publishV10()
			o := e.options(e.docsConfig(v10))
			if _, err := Run(context.Background(), o); err != nil {
				t.Fatal(err)
			}
			before := readPage(t, o.Out, "_versions/v1.0/cli/content/reference/cli/opm-module.md")
			lockBefore, _ := os.ReadFile(o.Lock)
			cli := cliSpec("1.0.0-beta.7", map[string]string{"core": "2.0.0-beta.1", "opm-controller": "1.0.0-beta.4"})
			cli.extra = c.extra
			e.publishDocs(cli)
			e.retag("cli", "1.0.0-beta.7", "1.0")
			_, err := Run(context.Background(), o)
			if err == nil || IsUsage(err) {
				t.Fatalf("error %v, want a refusal", err)
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q lacks %q", err, w)
				}
			}
			if got := readPage(t, o.Out, "_versions/v1.0/cli/content/reference/cli/opm-module.md"); got != before {
				t.Fatalf("the previous cli changed:\n%s", got)
			}
			if lockAfter, _ := os.ReadFile(o.Lock); !bytes.Equal(lockBefore, lockAfter) {
				t.Fatal("the lock changed")
			}
			if left, _ := filepath.Glob(filepath.Join(o.Out, VersionsDir, ".incoming-*")); len(left) != 0 {
				t.Fatalf("left %v", left)
			}
		})
	}
}

// A docs bundle at tag edge has no version to put in a page.
func TestFillRefusesEdge(t *testing.T) {
	e := newEnv(t)
	e.publishDocs(coreSpec("2.0.0-beta.1"))
	e.publishDocs(controllerSpec())
	cli := cliSpec("edge", map[string]string{"core": "2.0.0-beta.1", "opm-controller": "1.0.0-beta.4"})
	cli.extra = "\nv{{opm:cli.version}}\n"
	e.publishDocs(cli)
	o := e.options(e.docsConfig(`versions: "v1.0": {
	anchor: {project: "cli", tag: "edge"}
	pinned: ["core", "opm-controller"]
}
`))
	_, err := Run(context.Background(), o)
	if err == nil || !strings.Contains(err.Error(), `"{{opm:cli.version}}"`) || !strings.Contains(err.Error(), "the value is edge, not a version") {
		t.Fatalf("error %v", err)
	}
}

func TestFillRefusesTwoBundlesRecordingAUnit(t *testing.T) {
	e := newEnv(t)
	units := map[string]string{"opm-controller-module": "0.1.1"}
	e.publishPlaceholders(units)
	cli := cliSpec("1.0.0-beta.7", map[string]string{"core": "2.0.0-beta.1", "opm-controller": "1.0.0-beta.4"})
	cli.units = units
	e.publishDocs(cli)
	o := e.options(e.docsConfig(v10))
	_, err := Run(context.Background(), o)
	if err == nil || !strings.Contains(err.Error(), "cli 1.0.0-beta.7") || !strings.Contains(err.Error(), "opm-controller 1.0.0-beta.4") || !strings.Contains(err.Error(), "both record the unit opm-controller-module") {
		t.Fatalf("error %v", err)
	}
}

// A placeholder in a tab bundle fails the pull, names the page and leaves
// the previous tree.
func TestLeftoverInATabBundle(t *testing.T) {
	e := newEnv(t)
	e.publish("4.4.5", owner, nil)
	o := e.options(e.config(""))
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	lockBefore, _ := os.ReadFile(o.Lock)
	dir := tree(t, "4.4.6", nil)
	m, err := bundle.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	page := m.Pages[0].Path
	f := filepath.Join(dir, bundle.ContentDir, filepath.FromSlash(page))
	raw, _ := os.ReadFile(f)
	if err := os.WriteFile(f, append(raw, []byte("\nv{{opm:cli.version}}\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	e.publishDir(dir, "4.4.6", owner)
	_, err = Run(context.Background(), o)
	if err == nil || !strings.Contains(err.Error(), page) || !strings.Contains(err.Error(), "placeholder left after the fill") {
		t.Fatalf("error %v", err)
	}
	if lockAfter, _ := os.ReadFile(o.Lock); !bytes.Equal(lockBefore, lockAfter) {
		t.Fatal("the lock changed")
	}
	if _, err := os.Stat(filepath.Join(o.Out, "catalog-opm", "4.4", "manifest.json")); err != nil {
		t.Fatalf("the previous tree is gone: %v", err)
	}
}

// A placeholder in a section bundle fails the pull.
func TestLeftoverInASectionBundle(t *testing.T) {
	e := newEnv(t)
	e.publish("4.4.5", owner, nil)
	dir := sectionTree(t)
	f := filepath.Join(dir, "content", "_index.md")
	raw, _ := os.ReadFile(f)
	if err := os.WriteFile(f, append(raw, []byte("\nv{{opm:cli.version}}\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	e.publishProject(sectionProject, dir, "edge", sectionOwner)
	_, err := Run(context.Background(), e.options(e.sectionConfig()))
	if err == nil || !strings.Contains(err.Error(), "_index.md") || !strings.Contains(err.Error(), "placeholder left after the fill") {
		t.Fatalf("error %v", err)
	}
}

// A data file keeps its text, and a frozen pull fills the same as the pull
// that wrote the lock.
func TestFillFrozenEqualsFirstPull(t *testing.T) {
	e := newEnv(t)
	e.publishPlaceholders(nil)
	o := e.options(e.docsConfig(v10))
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	first := readPage(t, o.Out, "_versions/v1.0/cli/content/reference/cli/opm-module.md")
	lock, _ := os.ReadFile(o.Lock)
	saved := filepath.Join(e.work, "saved.lock")
	_ = os.WriteFile(saved, lock, 0o600)
	_ = os.RemoveAll(o.Out)
	o.Frozen = saved
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if got := readPage(t, o.Out, "_versions/v1.0/cli/content/reference/cli/opm-module.md"); got != first {
		t.Fatalf("frozen page differs:\n%s\nwant:\n%s", got, first)
	}
	if again, _ := os.ReadFile(o.Lock); !bytes.Equal(again, lock) {
		t.Fatal("the lock changed")
	}
}

func TestFillLeavesDataFilesAlone(t *testing.T) {
	e := newEnv(t)
	e.publishDocs(coreSpec("2.0.0-beta.1"))
	e.publishDocs(controllerSpec())
	cli := cliSpec("1.0.0-beta.6", map[string]string{"core": "2.0.0-beta.1", "opm-controller": "1.0.0-beta.4"})
	cli.extra = "\nv{{opm:cli.version}}\n"
	dir := docsTree(t, cli)
	if err := os.MkdirAll(filepath.Join(dir, bundle.DataDir), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, bundle.DataDir, "note.json"), []byte(`{"v": "{{opm:cli.version}}"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	m, _ := bundle.Read(dir)
	m.Data = []bundle.DataFile{{Path: "note.json", Schema: "docs.opmodel.dev/data/note/v1"}}
	if err := bundle.Write(dir, m); err != nil {
		t.Fatal(err)
	}
	e.publishProject("cli", dir, "1.0.0-beta.6", "open-platform-model/cli")
	o := e.options(e.docsConfig(v10))
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if got := readPage(t, o.Out, "_versions/v1.0/cli/data/note.json"); got != `{"v": "{{opm:cli.version}}"}` {
		t.Fatalf("data file changed: %s", got)
	}
}

func TestFillLocalTree(t *testing.T) {
	e := newEnv(t)
	o := e.options(e.docsConfig(v10))
	e.offline(&o)
	pins := map[string]string{"core": "2.0.0-beta.1", "opm-controller": "1.0.0-beta.4"}
	cli := cliSpec("1.0.0-beta.6", pins)
	cli.extra = phExtra
	o.Locals = []Local{
		{"cli", "v1.0", docsTree(t, cli)},
		{"core", "v1.0", docsTree(t, coreSpec("2.0.0-beta.1"))},
		{"opm-controller", "v1.0", docsTree(t, controllerSpec())},
	}
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if got := readPage(t, o.Out, "_versions/v1.0/cli/content/reference/cli/opm-module.md"); !strings.HasSuffix(got, phFilled) {
		t.Fatalf("local page:\n%s", got)
	}
}
