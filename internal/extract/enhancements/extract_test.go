package enhancements

import (
	"bytes"
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files")

const (
	fixtureRepo = "testdata/repo"
	testCommit  = "cccccccccccccccccccccccccccccccccccccccc"
)

// treePaths lists a tree the way git ls-tree -r -t does: every directory
// and file under root, relative and slash-separated.
func treePaths(t *testing.T, root string) map[string]string {
	t.Helper()
	paths := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == root {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		kind := "blob"
		if d.IsDir() {
			kind = "tree"
		}
		paths[filepath.ToSlash(rel)] = kind
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

// lastCommits are the dates the tests' history gives each entry folder;
// a folder not listed has no commit.
var lastCommits = map[string]string{
	"openspec/changes/0026-module-dictated-catalog-versions":     "2026-10-20T23:30:00-02:00", // 2026-10-21 in UTC
	"openspec/changes/archive/2026-10-21-0040-fixture-delivered": "2026-10-21T08:00:00Z",
	"openspec/changes/0042-fixture-no-specs":                     "2026-10-02T00:00:00Z",
}

func options(t *testing.T, root string) Options {
	t.Helper()
	return Options{
		Root: root, Dir: ".", Repo: "open-platform-model/enhancements", Commit: testCommit, Paths: treePaths(t, root),
		Title: "Enhancements", Description: "OPM's design record.",
		LastCommit: func(p string) (string, error) { return lastCommits[p], nil },
	}
}

// copyFixture copies the fixture repository into a temporary directory
// a test may change.
func copyFixture(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir(fixtureRepo, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(fixtureRepo, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func TestModelGolden(t *testing.T) {
	res, err := Extract(options(t, fixtureRepo))
	if err != nil {
		t.Fatal(err)
	}
	got, err := res.Model.Encode()
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join("testdata", "enhancements.golden.json")
	if *update {
		if err := os.WriteFile(file, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("%v (run go test -run TestModelGolden -update)", err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("%s differs:\n%s", file, got)
	}
	if _, err := Decode(got); err != nil {
		t.Fatal(err)
	}
}

func TestPages(t *testing.T) {
	res, err := Extract(options(t, fixtureRepo))
	if err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, len(res.Pages))
	byPath := map[string]Page{}
	for _, p := range res.Pages {
		paths = append(paths, p.Path)
		byPath[p.Path] = p
	}
	want := "0003/_index.md 0003/decisions.md 0003/design.md 0003/graduation.md 0003/operational.md 0003/problem.md 0003/questions.md 0003/risks.md " +
		"0025/_index.md 0025/decisions.md 0025/design.md 0025/graduation.md 0025/operational.md 0025/problem.md 0025/questions.md 0025/risks.md " +
		"0026/_index.md 0026/design.md 0026/proposal.md 0026/requirements.md 0026/tasks.md " +
		"0040/_index.md 0040/design.md 0040/proposal.md 0040/requirements.md 0040/tasks.md " +
		"0041/_index.md 0041/design.md 0041/proposal.md 0041/requirements.md " +
		"0042/_index.md 0042/design.md 0042/proposal.md " +
		"_index.md graph.md"
	if got := strings.Join(paths, " "); got != want {
		t.Fatalf("pages\n%s\nwant\n%s", got, want)
	}
	if p := byPath["0003/_index.md"]; p.Source != "archive/0003/README.md" || p.Weight != 4 || p.Title != "0003: Module Publishing Workflow" {
		t.Errorf("archived entry page %+v", p)
	}
	if p := byPath["0025/decisions.md"]; p.Source != "archive/0025/03-decisions.md" || p.Type != "explanation" || p.Weight != 3 || p.Description != "Each decision, numbered, with its rationale." {
		t.Errorf("document page %+v", p)
	}
	if p := byPath["0025/_index.md"]; p.Description != "Let a module state facts about itself as a whole, as named bundles of traits." {
		t.Errorf("summary not collapsed: %q", p.Description)
	}
}

// TestRefusals checks each fault of a repository fails the extraction
// naming what is wrong.
func TestRefusals(t *testing.T) {
	for _, c := range []struct {
		name   string
		change func(t *testing.T, root string)
		msg    string
	}{
		{"two decision files", func(t *testing.T, root string) {
			write(t, root, "archive/0025/03-decisions-old.md", "# Old\n")
		}, "entry 0025 holds archive/0025/03-decisions-old.md and archive/0025/03-decisions.md"},
		{"a missing document", func(t *testing.T, root string) {
			remove(t, root, "archive/0025/05-risks.md")
		}, "entry 0025 has no 05-*.md"},
		{"an eighth document", func(t *testing.T, root string) {
			write(t, root, "archive/0025/08-extra.md", "# Extra\n")
		}, "entry 0025 holds archive/0025/08-extra.md"},
		{"an id that is not the directory", func(t *testing.T, root string) {
			replace(t, root, "archive/0025/config.yaml", `id: "0025"`, `id: "0026"`)
		}, `archive/0025/config.yaml says id "0026"`},
		{"a live old-layout folder", func(t *testing.T, root string) {
			if err := os.MkdirAll(filepath.Join(root, "0030"), 0o755); err != nil {
				t.Fatal(err)
			}
		}, "entry 0030 is a live old-layout folder (0030/); convert it to a change under the changes directory or move it under archive/"},
		{"no README", func(t *testing.T, root string) {
			remove(t, root, "archive/0025/README.md")
		}, "archive/0025/README.md is missing"},
		{"no INDEX", func(t *testing.T, root string) {
			remove(t, root, "INDEX.md")
		}, "INDEX.md is missing"},
		{"a dangling link", func(t *testing.T, root string) {
			write(t, root, "archive/0025/06-operational.md", "# Operational\n\nSee [missing.md](missing.md).\n")
		}, `archive/0025/06-operational.md:3: link "missing.md" names nothing in the repository at cccccccccccc (archive/0025/missing.md)`},
		{"a link out of the repository", func(t *testing.T, root string) {
			write(t, root, "archive/0025/06-operational.md", "# Operational\n\nSee [up](../../../x.md).\n")
		}, `link "../../../x.md" climbs out of the repository`},
		{"a link to an entry that is not there", func(t *testing.T, root string) {
			write(t, root, "archive/0025/06-operational.md", "# Operational\n\nSee [0004](../0004/).\n")
		}, `link "../0004/" names nothing in the repository`},
		{"an unsafe scheme", func(t *testing.T, root string) {
			write(t, root, "archive/0025/06-operational.md", "# Operational\n\n[x](javascript:alert(1))\n")
		}, "the scheme javascript: is not allowed"},
		{"a shortcode", func(t *testing.T, root string) {
			write(t, root, "archive/0025/06-operational.md", "# Operational\n\n```text\n{{< figure >}}\n```\n")
		}, "archive/0025/06-operational.md:4: holds a Hugo shortcode delimiter"},
		{"an unclosed comment", func(t *testing.T, root string) {
			write(t, root, "archive/0025/06-operational.md", "# Operational\n\n<!-- never closed\n\nText.\n")
		}, "archive/0025/06-operational.md:3: an HTML comment opens with <!-- and never closes"},
		{"a malformed date", func(t *testing.T, root string) {
			replace(t, root, "archive/0025/config.yaml", `created: "2026-09-08"`, `created: "yesterday"`)
		}, "must be dates"},
		{"a malformed relation", func(t *testing.T, root string) {
			replace(t, root, "archive/0025/config.yaml", `  - "0003"`, `  - "3"`)
		}, `depends_on: "3" is not an entry id`},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := copyFixture(t)
			c.change(t, root)
			_, err := Extract(options(t, root))
			if err == nil || !strings.Contains(err.Error(), c.msg) {
				t.Fatalf("err = %v, want it to name %q", err, c.msg)
			}
		})
	}
}

// TestSymlinks checks a symlinked file or entry is refused, never read.
func TestSymlinks(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		root := copyFixture(t)
		remove(t, root, "archive/0025/05-risks.md")
		if err := os.Symlink(filepath.Join(root, "INDEX.md"), filepath.Join(root, "archive", "0025", "05-risks.md")); err != nil {
			t.Fatal(err)
		}
		if _, err := Extract(options(t, root)); err == nil || !strings.Contains(err.Error(), "archive/0025/05-risks.md is not a regular file") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("entry root component", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "real"), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "link")); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "real", "entries"), 0o750); err != nil {
			t.Fatal(err)
		}
		o := options(t, fixtureRepo)
		o.Root, o.Dir = root, "link/entries"
		if _, err := Extract(o); err == nil || !strings.Contains(err.Error(), "reached through the symlink link") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("entry", func(t *testing.T) {
		root := copyFixture(t)
		if err := os.Symlink(filepath.Join(root, "archive", "0025"), filepath.Join(root, "archive", "0026")); err != nil {
			t.Fatal(err)
		}
		if _, err := Extract(options(t, root)); err == nil || !strings.Contains(err.Error(), "entry archive/0026 is reached through the symlink archive/0026") {
			t.Fatalf("err = %v", err)
		}
	})
}

// TestClean checks the transforms on single texts.
func TestClean(t *testing.T) {
	x := &extraction{o: options(t, fixtureRepo), dir: ".", pages: map[string]target{
		"INDEX.md":                     {url: "/enhancements/", title: "the index"},
		"archive/0025/README.md":       {url: "/enhancements/0025/", title: "0025: Self-Describing Modules"},
		"archive/0025/03-decisions.md": {url: "/enhancements/0025/decisions/", title: "0025: Decisions"},
	}}
	for _, c := range []struct{ name, in, want string }{
		{"heading and blank lines", "\n# Title\n\n\nBody.\n", "Body.\n"},
		{"heading not first", "Intro.\n\n# Title\n", "Intro.\n\n# Title\n"},
		{"untagged fence", "```\nx\n```\n", "```text\nx\n```\n"},
		{"tilde fence", "~~~ \nx\n~~~\n", "~~~text\nx\n~~~\n"},
		{"tagged fence", "```cue\nx\n```\n", "```cue\nx\n```\n"},
		{"fence content untouched", "```\n<b> [a](x.md) <!-- c -->\n```\n", "```text\n<b> [a](x.md) <!-- c -->\n```\n"},
		{"comment inline", "a <!-- b --> c\n", "a  c\n"},
		{"comment in a code span", "a `<!-- b -->` c\n", "a `<!-- b -->` c\n"},
		{"link in a code span", "`[a](missing.md)` stays\n", "`[a](missing.md)` stays\n"},
		{"file-name text", "[03-decisions.md](03-decisions.md#d1)\n", "[0025: Decisions](/enhancements/0025/decisions/#d1)\n"},
		{"file-name text in code", "[`README.md`](README.md)\n", "[0025: Self-Describing Modules](/enhancements/0025/)\n"},
		{"other text", "[the decisions](03-decisions.md)\n", "[the decisions](/enhancements/0025/decisions/)\n"},
		{"angle destination", "[x](<03-decisions.md>)\n", "[x](/enhancements/0025/decisions/)\n"},
		{"angle external", "[x](<https://a.example/b c>)\n", "[x](<https://a.example/b c>)\n"},
		{"angle fragment with a blank", "[x](<03-decisions.md#a b>)\n", "[x](</enhancements/0025/decisions/#a b>)\n"},
		{"entry directory", "[0025](./)\n", "[0025](/enhancements/0025/)\n"},
		{"root", "[all](../../)\n", "[all](/enhancements/)\n"},
		{"github file", "[s](schemas/target.cue#L1)\n", "[s](https://github.com/open-platform-model/enhancements/blob/" + testCommit + "/archive/0025/schemas/target.cue#L1)\n"},
		{"github directory", "[s](schemas)\n", "[s](https://github.com/open-platform-model/enhancements/tree/" + testCommit + "/archive/0025/schemas)\n"},
		{"fragment and absolute", "[a](#x) [b](/docs/concepts/) [c](mailto:a@b.example)\n", "[a](#x) [b](/docs/concepts/) [c](mailto:a@b.example)\n"},
		{"reference definition", "[r]: 03-decisions.md \"t\"\n", "[r]: /enhancements/0025/decisions/ \"t\"\n"},
		{"html escaped", "a <b>c</b> <!x <?y </z\n", "a \\<b>c\\</b> \\<!x \\<?y \\</z\n"},
		{"autolink and code kept", "<https://a.example/> <a@b.example> `<x>` a < b\n", "<https://a.example/> <a@b.example> `<x>` a < b\n"},
		{"escaped bracket", "x \\](javascript:alert(1)) y\n", "x \\](javascript:alert(1)) y\n"},
		{"escaped backslash before bracket", "[a\\\\](03-decisions.md)\n", "[a\\\\](/enhancements/0025/decisions/)\n"},
		{"html after an escaped backslash", "a \\\\<img src=x> b\n", "a \\\\\\<img src=x> b\n"},
		{"title escaped", "[README.md](README.md) <i>\n", "[0025: Self-Describing Modules](/enhancements/0025/) \\<i>\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := x.clean("archive/0025/02-design.md", c.in)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("got  %q\nwant %q", got, c.want)
			}
		})
	}
}

func write(t *testing.T, root, file, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(file)), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func remove(t *testing.T, root, file string) {
	t.Helper()
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(file))); err != nil {
		t.Fatal(err)
	}
}

func replace(t *testing.T, root, file, old, repl string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(file))
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), old) {
		t.Fatalf("%s holds no %q", file, old)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(b), old, repl, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
}

// entryOf finds an entry of the fixture by id.
func entryOf(t *testing.T, m *Model, id string) *Entry {
	t.Helper()
	for i := range m.Entries {
		if m.Entries[i].ID == id {
			return &m.Entries[i]
		}
	}
	t.Fatalf("no entry %s", id)
	return nil
}

// fixture extracts the mixed fixture and indexes its pages by path.
func fixture(t *testing.T) (res *Result, byPath map[string]Page) {
	t.Helper()
	res, err := Extract(options(t, fixtureRepo))
	if err != nil {
		t.Fatal(err)
	}
	byPath = map[string]Page{}
	for i := range res.Pages {
		byPath[res.Pages[i].Path] = res.Pages[i]
	}
	return res, byPath
}

func TestOpenSpecHeader(t *testing.T) {
	res, _ := fixture(t)
	e := entryOf(t, res.Model, "0026")
	if e.Layout != LayoutOpenSpec || e.Archived || e.Updated != "2026-10-21" || e.Breaking == nil || !*e.Breaking {
		t.Errorf("0026 %+v", e)
	}
	if e.Accepted == nil || e.Accepted.Date != "2026-10-16" || e.Accepted.Commit != "4be19d0" || e.Progress != (Progress{Done: 0, Total: 5}) {
		t.Errorf("0026 accepted %+v progress %+v", e.Accepted, e.Progress)
	}
	if got := strings.Join(e.Capabilities, " "); got != "authored-platform catalog-version-selection registration-acceptance single-build-render" {
		t.Errorf("capabilities %q", got)
	}
}

func TestOpenSpecEntryPages(t *testing.T) {
	_, byPath := fixture(t)
	if p := byPath["0026/proposal.md"]; p.Source != "openspec/changes/0026-module-dictated-catalog-versions/proposal.md" || p.Title != "0026: Proposal" || p.Type != "explanation" || p.Weight != 1 {
		t.Errorf("proposal page %+v", p)
	}
	if p := byPath["0026/_index.md"]; p.Weight != 27 || strings.Contains(p.Body, "GENERATED") || strings.HasPrefix(p.Body, "#") {
		t.Errorf("entry page %+v", p)
	}
}

func TestOpenSpecRequirementsPage(t *testing.T) {
	_, byPath := fixture(t)
	req := byPath["0026/requirements.md"]
	if req.Source != "" || len(req.Inputs) != 4 || req.Weight != 3 {
		t.Errorf("requirements page %+v", req)
	}
	var heads []string
	for _, l := range strings.Split(req.Body, "\n") {
		if strings.HasPrefix(l, "## ") || strings.HasPrefix(l, "### ") && !strings.HasPrefix(l, "### Requirement") {
			heads = append(heads, l)
		}
	}
	if !strings.Contains(strings.Join(heads, "|"), "## authored-platform|### Purpose|### ADDED Requirements") || strings.Contains(req.Body, "\n## ADDED Requirements") {
		t.Errorf("requirements headings %q", heads)
	}
	if !strings.Contains(req.Body, "\n#### Requirement:") || !strings.Contains(req.Body, "\n##### Scenario:") {
		t.Errorf("requirements page does not demote requirement and scenario headings:\n%s", req.Body)
	}
}

func TestOpenSpecReadmeLinks(t *testing.T) {
	_, byPath := fixture(t)
	readme := byPath["0026/_index.md"].Body
	for _, want := range []string{
		"[0026: Proposal](/enhancements/0026/proposal/)",
		"[specs/](/enhancements/0026/requirements/)",
		"(https://github.com/open-platform-model/enhancements/tree/" + testCommit + "/openspec/changes/0026-module-dictated-catalog-versions/evidence)",
		"[0026: Tasks](/enhancements/0026/tasks/)",
	} {
		if !strings.Contains(readme, want) {
			t.Errorf("README page lacks %q:\n%s", want, readme)
		}
	}
}

func TestOpenSpecEntryVariants(t *testing.T) {
	res, _ := fixture(t)
	d := entryOf(t, res.Model, "0040")
	if !d.Archived || d.Updated != "2026-10-21" || d.Progress != (Progress{Done: 2, Total: 2}) || d.Breaking == nil || *d.Breaking {
		t.Errorf("0040 %+v", d)
	}
	if r := entryOf(t, res.Model, "0041"); r.Updated != "2026-09-20" {
		t.Errorf("0041 has no commit, so updated is created: %q", r.Updated)
	}
	n := entryOf(t, res.Model, "0042")
	if n.Breaking != nil || len(n.Documents) != 2 || n.Capabilities == nil || n.Progress != (Progress{}) {
		t.Errorf("0042 %+v", n)
	}
	if l := entryOf(t, res.Model, "0003"); l.Layout != LayoutLegacy || len(l.Aliases) != 0 {
		t.Errorf("legacy entry %+v", l)
	}
}

// An entry without a requirements or tasks page redirects those slugs to
// its own page.
func TestOpenSpecAliases(t *testing.T) {
	res, _ := fixture(t)
	got := map[string]string{}
	for _, a := range entryOf(t, res.Model, "0042").Aliases {
		got[a.From] = a.To
	}
	if got["0042/decisions"] != "0042" || got["0042/graduation"] != "0042" || got["0042/problem"] != "0042/proposal" || got["0042/risks"] != "0042/design" || got["0042/questions"] != "0042/proposal" || got["0042/operational"] != "0042/design" {
		t.Errorf("0042 aliases %v", got)
	}
	if _, ok := got["0042/design"]; ok {
		t.Error("design is a new slug as well: it needs no alias")
	}
	full := map[string]string{}
	for _, a := range entryOf(t, res.Model, "0026").Aliases {
		full[a.From] = a.To
	}
	if full["0026/decisions"] != "0026/requirements" || full["0026/graduation"] != "0026/tasks" {
		t.Errorf("0026 aliases %v", full)
	}
}

func TestOpenSpecRefusals(t *testing.T) {
	const c26 = "openspec/changes/0026-module-dictated-catalog-versions/"
	for _, c := range []struct {
		name   string
		change func(t *testing.T, root string)
		msg    string
	}{
		{"no design", func(t *testing.T, root string) { remove(t, root, c26+"design.md") }, "entry 0026: " + c26 + "design.md is missing"},
		{"no README", func(t *testing.T, root string) { remove(t, root, c26+"README.md") }, "entry 0026: " + c26 + "README.md is missing"},
		{"no config", func(t *testing.T, root string) { remove(t, root, c26+"config.yaml") }, "entry 0026: " + c26 + "config.yaml is missing"},
		{"a folder of another name", func(t *testing.T, root string) {
			mkdir(t, root, "openspec/changes/fix-things")
		}, "change folder openspec/changes/fix-things is not <id>-<slug> or a seed-<capability> change"},
		{"an archived folder of another name", func(t *testing.T, root string) {
			mkdir(t, root, "openspec/changes/archive/0043-x")
		}, "change folder openspec/changes/archive/0043-x is not <date>-<id>-<slug>"},
		{"an id that is not the folder's", func(t *testing.T, root string) {
			replace(t, root, c26+"config.yaml", `id: "0026"`, `id: "0027"`)
		}, `says id "0027"`},
		{"a slug that is not the folder's", func(t *testing.T, root string) {
			replace(t, root, c26+"config.yaml", "slug: module-dictated-catalog-versions", "slug: other")
		}, `says slug "other"`},
		{"a malformed accepted", func(t *testing.T, root string) {
			replace(t, root, c26+"config.yaml", `accepted: "2026-10-16 4be19d0"`, `accepted: "yesterday"`)
		}, `accepted "yesterday" must read`},
		{"a malformed after", func(t *testing.T, root string) {
			replace(t, root, c26+"config.yaml", `  - "0010"`, `  - "10"`)
		}, `"10" is not an entry id`},
		{"an id in both layouts", func(t *testing.T, root string) {
			mkdir(t, root, "archive/0026")
		}, "entry 0026 is both archive/0026/ and openspec/changes/0026-module-dictated-catalog-versions/"},
		{"an id twice in the openspec layout", func(t *testing.T, root string) {
			mkdir(t, root, "openspec/changes/archive/2026-10-30-0042-again")
		}, "entry 0042 is both"},
		{"a capability name", func(t *testing.T, root string) {
			mkdir(t, root, c26+"specs/Bad_Name")
		}, "is not a capability name"},
		{"a spec file missing", func(t *testing.T, root string) {
			mkdir(t, root, c26+"specs/empty")
		}, "specs/empty/spec.md is missing"},
		{"a level-6 heading in a delta", func(t *testing.T, root string) {
			write(t, root, c26+"specs/authored-platform/spec.md", "## ADDED Requirements\n\n###### Deep\n")
		}, "a level-6 heading cannot move down"},
		{"a dangling link in a delta", func(t *testing.T, root string) {
			write(t, root, c26+"specs/authored-platform/spec.md", "## ADDED Requirements\n\nSee [x](missing.md).\n")
		}, "specs/authored-platform/spec.md:3: link \"missing.md\" names nothing"},
		{"a live legacy folder", func(t *testing.T, root string) {
			mkdir(t, root, "0030")
		}, "entry 0030 is a live old-layout folder"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := copyFixture(t)
			c.change(t, root)
			_, err := Extract(options(t, root))
			if err == nil || !strings.Contains(err.Error(), c.msg) {
				t.Fatalf("err = %v, want it to name %q", err, c.msg)
			}
		})
	}
}

// TestSeedsAreNoEntries checks a seed change is no entry and no page, and
// a symlinked change is refused.
func TestSeedsAndSymlinks(t *testing.T) {
	res, err := Extract(options(t, fixtureRepo))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range res.Model.Entries {
		if strings.Contains(e.Slug, "seed") {
			t.Errorf("seed %s is an entry", e.Slug)
		}
	}
	root := copyFixture(t)
	if err := os.Symlink(filepath.Join(root, "openspec", "changes", "0042-fixture-no-specs"), filepath.Join(root, "openspec", "changes", "0043-link")); err != nil {
		t.Fatal(err)
	}
	if _, err := Extract(options(t, root)); err == nil || !strings.Contains(err.Error(), "reached through the symlink openspec/changes/0043-link") {
		t.Fatalf("err = %v", err)
	}
	root = copyFixture(t)
	c := "openspec/changes/0042-fixture-no-specs/"
	remove(t, root, c+"design.md")
	if err := os.Symlink(filepath.Join(root, c, "proposal.md"), filepath.Join(root, c, "design.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := Extract(options(t, root)); err == nil || !strings.Contains(err.Error(), "design.md is not a regular file") {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdatedFromHistory(t *testing.T) {
	o := options(t, fixtureRepo)
	o.LastCommit = func(string) (string, error) { return "", errors.New("shallow") }
	if _, err := Extract(o); err == nil || !strings.Contains(err.Error(), "entry 0026: shallow") {
		t.Fatalf("err = %v, want the history error named with the entry", err)
	}
	o.LastCommit = nil
	res, err := Extract(o)
	if err != nil {
		t.Fatal(err)
	}
	if got := entryOf(t, res.Model, "0026").Updated; got != "2026-09-08" {
		t.Errorf("without history updated is created, got %q", got)
	}
	o.LastCommit = func(string) (string, error) { return "later", nil }
	if _, err := Extract(o); err == nil || !strings.Contains(err.Error(), `commit date "later"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestProgress(t *testing.T) {
	got := progress("## 1. A\n\n- [x] 1.1 a\n- [ ] 1.2 b\n* [X] 1.3 c\n\n```\n- [ ] 9.9 not a task\n```\n\n- [ ] 2.1 d\ntext - [ ] not a line start\n")
	if got != (Progress{Done: 2, Total: 4}) {
		t.Errorf("progress %+v", got)
	}
}

func TestDemoteHeadings(t *testing.T) {
	got, err := demoteHeadings("f.md", "## A\n### B\n```\n## in a fence\n```\n#hashtag\n    # indented code\n####### seven\n#### D\n")
	if err != nil {
		t.Fatal(err)
	}
	want := "### A\n#### B\n```\n## in a fence\n```\n#hashtag\n    # indented code\n####### seven\n##### D\n"
	if got != want {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}
}

func TestDataRoundTrip(t *testing.T) {
	res, err := Extract(options(t, fixtureRepo))
	if err != nil {
		t.Fatal(err)
	}
	b, err := res.Model.Encode()
	if err != nil {
		t.Fatal(err)
	}
	m, err := Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	again, err := m.Encode()
	if err != nil || !bytes.Equal(b, again) {
		t.Fatalf("a decoded model does not encode to the same bytes: %v", err)
	}
	v1 := []byte(`{"schema": "docs.opmodel.dev/data/enhancements/v1", "repo": "r", "entries": [{"id": "0001", "slug": "a", "supersededBy": "0002"}]}`)
	old, err := Decode(v1)
	if err != nil || old.Entries[0].Layout != LayoutLegacy || old.Entries[0].SupersededBy == nil || *old.Entries[0].SupersededBy != "0002" {
		t.Fatalf("v1 data: %+v %v", old, err)
	}
	if _, err := Decode([]byte(`{"schema": "docs.opmodel.dev/data/enhancements/v3"}`)); err == nil {
		t.Fatal("an unknown schema was accepted")
	}
}

func mkdir(t *testing.T, root, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
		t.Fatal(err)
	}
}
