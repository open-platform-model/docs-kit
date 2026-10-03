package enhancements

import (
	"bytes"
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

func options(t *testing.T, root string) Options {
	t.Helper()
	return Options{
		Root: root, Dir: ".", Repo: "open-platform-model/enhancements", Commit: testCommit, Paths: treePaths(t, root),
		Title: "Enhancements", Description: "OPM's design record.",
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
		"_index.md graph.md"
	if got := strings.Join(paths, " "); got != want {
		t.Fatalf("pages\n%s\nwant\n%s", got, want)
	}
	if p := byPath["0003/_index.md"]; p.Source != "archive/0003/README.md" || p.Weight != 4 || p.Title != "0003: Module Publishing Workflow" {
		t.Errorf("archived entry page %+v", p)
	}
	if p := byPath["0025/decisions.md"]; p.Source != "0025/03-decisions.md" || p.Type != "explanation" || p.Weight != 3 || p.Description != "Each decision, numbered, with its rationale." {
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
			write(t, root, "0025/03-decisions-old.md", "# Old\n")
		}, "entry 0025 holds 0025/03-decisions-old.md and 0025/03-decisions.md"},
		{"a missing document", func(t *testing.T, root string) {
			remove(t, root, "0025/05-risks.md")
		}, "entry 0025 has no 05-*.md"},
		{"an eighth document", func(t *testing.T, root string) {
			write(t, root, "0025/08-extra.md", "# Extra\n")
		}, "entry 0025 holds 0025/08-extra.md"},
		{"an id that is not the directory", func(t *testing.T, root string) {
			replace(t, root, "0025/config.yaml", `id: "0025"`, `id: "0026"`)
		}, `0025/config.yaml says id "0026"`},
		{"an entry live and archived", func(t *testing.T, root string) {
			if err := os.MkdirAll(filepath.Join(root, "archive", "0025"), 0o755); err != nil {
				t.Fatal(err)
			}
		}, "entry 0025 is both 0025/ and archive/0025/"},
		{"no README", func(t *testing.T, root string) {
			remove(t, root, "0025/README.md")
		}, "0025/README.md is missing"},
		{"no INDEX", func(t *testing.T, root string) {
			remove(t, root, "INDEX.md")
		}, "INDEX.md is missing"},
		{"a dangling link", func(t *testing.T, root string) {
			write(t, root, "0025/06-operational.md", "# Operational\n\nSee [missing.md](missing.md).\n")
		}, `0025/06-operational.md:3: link "missing.md" names nothing in the repository at cccccccccccc (0025/missing.md)`},
		{"a link out of the repository", func(t *testing.T, root string) {
			write(t, root, "0025/06-operational.md", "# Operational\n\nSee [up](../../x.md).\n")
		}, `link "../../x.md" climbs out of the repository`},
		{"a link to an archived entry's old place", func(t *testing.T, root string) {
			write(t, root, "0025/06-operational.md", "# Operational\n\nSee [0003](../0003/).\n")
		}, `link "../0003/" names nothing in the repository`},
		{"an unsafe scheme", func(t *testing.T, root string) {
			write(t, root, "0025/06-operational.md", "# Operational\n\n[x](javascript:alert(1))\n")
		}, "the scheme javascript: is not allowed"},
		{"a shortcode", func(t *testing.T, root string) {
			write(t, root, "0025/06-operational.md", "# Operational\n\n```text\n{{< figure >}}\n```\n")
		}, "0025/06-operational.md:4: holds a Hugo shortcode delimiter"},
		{"an unclosed comment", func(t *testing.T, root string) {
			write(t, root, "0025/06-operational.md", "# Operational\n\n<!-- never closed\n\nText.\n")
		}, "0025/06-operational.md:3: an HTML comment opens with <!-- and never closes"},
		{"a malformed date", func(t *testing.T, root string) {
			replace(t, root, "0025/config.yaml", `created: "2026-09-08"`, `created: "yesterday"`)
		}, "must be dates"},
		{"a malformed relation", func(t *testing.T, root string) {
			replace(t, root, "0025/config.yaml", `  - "0003"`, `  - "3"`)
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
		remove(t, root, "0025/05-risks.md")
		if err := os.Symlink(filepath.Join(root, "INDEX.md"), filepath.Join(root, "0025", "05-risks.md")); err != nil {
			t.Fatal(err)
		}
		if _, err := Extract(options(t, root)); err == nil || !strings.Contains(err.Error(), "0025/05-risks.md is not a regular file") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("entry", func(t *testing.T) {
		root := copyFixture(t)
		if err := os.Symlink(filepath.Join(root, "0025"), filepath.Join(root, "0026")); err != nil {
			t.Fatal(err)
		}
		if _, err := Extract(options(t, root)); err == nil || !strings.Contains(err.Error(), "entry 0026 is a symlink") {
			t.Fatalf("err = %v", err)
		}
	})
}

// TestClean checks the transforms on single texts.
func TestClean(t *testing.T) {
	x := &extraction{o: options(t, fixtureRepo), dir: ".", pages: map[string]target{
		"INDEX.md":             {url: "/enhancements/", title: "the index"},
		"0025/README.md":       {url: "/enhancements/0025/", title: "0025: Self-Describing Modules"},
		"0025/03-decisions.md": {url: "/enhancements/0025/decisions/", title: "0025: Decisions"},
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
		{"entry directory", "[0025](./)\n", "[0025](/enhancements/0025/)\n"},
		{"root", "[all](../)\n", "[all](/enhancements/)\n"},
		{"github file", "[s](schemas/target.cue#L1)\n", "[s](https://github.com/open-platform-model/enhancements/blob/" + testCommit + "/0025/schemas/target.cue#L1)\n"},
		{"github directory", "[s](schemas)\n", "[s](https://github.com/open-platform-model/enhancements/tree/" + testCommit + "/0025/schemas)\n"},
		{"fragment and absolute", "[a](#x) [b](/docs/concepts/) [c](mailto:a@b.example)\n", "[a](#x) [b](/docs/concepts/) [c](mailto:a@b.example)\n"},
		{"reference definition", "[r]: 03-decisions.md \"t\"\n", "[r]: /enhancements/0025/decisions/ \"t\"\n"},
		{"html escaped", "a <b>c</b> <!x <?y </z\n", "a \\<b>c\\</b> \\<!x \\<?y \\</z\n"},
		{"autolink and code kept", "<https://a.example/> <a@b.example> `<x>` a < b\n", "<https://a.example/> <a@b.example> `<x>` a < b\n"},
		{"title escaped", "[README.md](README.md) <i>\n", "[0025: Self-Describing Modules](/enhancements/0025/) \\<i>\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := x.clean("0025/02-design.md", c.in)
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
