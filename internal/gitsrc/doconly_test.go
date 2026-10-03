package gitsrc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/gittest"
)

// The release tree every documentation-only case starts from.
var released = map[string]string{
	"README.md":        "# Demo\n",
	"docs/guide.md":    "Guide.\n",
	"go.mod":           "module example.com/demo\n\ngo 1.26\n",
	"assets/logo.json": "{}\n",
	"opm/traits/v1alpha1/backup.cue": `package v1alpha1

import "strings"

// #Backup schedules backups of a component's volumes.
#Backup: {
	metadata: {
		name:        "backup"
		description: "Back up a component's volumes on a schedule"
	}
	spec: {
		schedule: string | *"0 2 * * *" // cron, UTC
		retain:   int & >=1
		target:   string @go(Target)
	}
	label: "\(metadata.name)-\(strings.ToLower("X"))"
}
`,
	"tool/main.go": `// Package main is a demo tool.
package main

import (
	_ "embed"
	"fmt"
)

//go:embed banner.txt
var banner string

var other string

//go:noinline
func helper() {}

func second() {}

// Config is the tool's configuration.
type Config struct {
	Name string
	Port int
}

func main() {
	x := 1; fmt.Println(banner, x, "a // not a comment")
}
`,
	"tool/banner.txt": "hello\n",
	"cgo/c.go": `package cgo

// #include <stdio.h>
import "C"

// Hello says hello.
func Hello() {}
`,
}

func TestDocumentationOnly(t *testing.T) {
	cases := []struct {
		name    string
		write   map[string]string
		remove  []string
		rename  [2]string
		symlink [2]string
		base    func(r *gittest.Repo) // extra release-tree setup
		refused []string              // "path: substring of the reason"
	}{
		{name: "markdown changed", write: map[string]string{"README.md": "# Demo, fixed\n"}},
		{name: "markdown added", write: map[string]string{"docs/new.md": "New.\n"}},
		{name: "markdown removed", remove: []string{"docs/guide.md"}},
		{name: "markdown renamed", rename: [2]string{"docs/guide.md", "docs/howto.md"}},
		{name: "markdown renamed to text", rename: [2]string{"docs/guide.md", "docs/guide.txt"},
			refused: []string{"docs/guide.txt: renamed from docs/guide.md"}},
		{name: "cue doc comment reworded", write: map[string]string{"opm/traits/v1alpha1/backup.cue": strings.Replace(released["opm/traits/v1alpha1/backup.cue"],
			"// #Backup schedules backups of a component's volumes.", "// #Backup schedules a backup of every volume of a component.", 1)}},
		{name: "cue comment added between fields", write: map[string]string{"opm/traits/v1alpha1/backup.cue": strings.Replace(released["opm/traits/v1alpha1/backup.cue"],
			"\t\tretain:   int & >=1\n", "\t\t// retain is how many backups to keep.\n\t\t// At least one.\n\t\tretain:   int & >=1\n", 1)}},
		{name: "cue trailing comment removed", write: map[string]string{"opm/traits/v1alpha1/backup.cue": strings.Replace(released["opm/traits/v1alpha1/backup.cue"],
			" // cron, UTC", "", 1)}},
		{name: "cue description value", write: map[string]string{"opm/traits/v1alpha1/backup.cue": strings.Replace(released["opm/traits/v1alpha1/backup.cue"],
			"on a schedule", "on a cron schedule", 1)},
			refused: []string{"opm/traits/v1alpha1/backup.cue: changes CUE values, not only comments; a change to code needs a patch release"}},
		{name: "cue default value", write: map[string]string{"opm/traits/v1alpha1/backup.cue": strings.Replace(released["opm/traits/v1alpha1/backup.cue"],
			`*"0 2 * * *"`, `*"0 3 * * *"`, 1)},
			refused: []string{"backup.cue: changes CUE values"}},
		{name: "cue constraint", write: map[string]string{"opm/traits/v1alpha1/backup.cue": strings.Replace(released["opm/traits/v1alpha1/backup.cue"],
			">=1", ">=2", 1)},
			refused: []string{"backup.cue: changes CUE values"}},
		{name: "cue attribute", write: map[string]string{"opm/traits/v1alpha1/backup.cue": strings.Replace(released["opm/traits/v1alpha1/backup.cue"],
			"@go(Target)", "@go(Dest)", 1)},
			refused: []string{"backup.cue: changes CUE values"}},
		{name: "cue interpolation text", write: map[string]string{"opm/traits/v1alpha1/backup.cue": strings.Replace(released["opm/traits/v1alpha1/backup.cue"],
			`)-\(strings`, `)_\(strings`, 1)},
			refused: []string{"backup.cue: changes CUE values"}},
		{name: "cue interpolation expression", write: map[string]string{"opm/traits/v1alpha1/backup.cue": strings.Replace(released["opm/traits/v1alpha1/backup.cue"],
			`ToLower("X")`, `ToUpper("X")`, 1)},
			refused: []string{"backup.cue: changes CUE values"}},
		{name: "cue comment turned into code", write: map[string]string{"opm/traits/v1alpha1/backup.cue": strings.Replace(released["opm/traits/v1alpha1/backup.cue"],
			" // cron, UTC", "\n\t\tcron: true", 1)},
			refused: []string{"backup.cue: changes CUE values"}},
		{name: "cue file added", write: map[string]string{"opm/traits/v1alpha1/extra.cue": "package v1alpha1\n"},
			refused: []string{"opm/traits/v1alpha1/extra.cue: added; only the comments of an existing .cue file"}},
		{name: "cue file removed", remove: []string{"opm/traits/v1alpha1/backup.cue"},
			refused: []string{"backup.cue: removed"}},
		{name: "cue file renamed", rename: [2]string{"opm/traits/v1alpha1/backup.cue", "opm/traits/v1alpha1/backups.cue"},
			refused: []string{"backups.cue: renamed from opm/traits/v1alpha1/backup.cue"}},
		{name: "cue file renamed to markdown", rename: [2]string{"opm/traits/v1alpha1/backup.cue", "opm/traits/v1alpha1/backup.md"},
			refused: []string{"backup.md: renamed from opm/traits/v1alpha1/backup.cue, which is not Markdown"}},
		{name: "go doc comment", write: map[string]string{"tool/main.go": strings.Replace(released["tool/main.go"],
			"// Config is the tool's configuration.", "// Config is how the tool is configured.\n// It has two fields.", 1)}},
		{name: "go comment added between fields", write: map[string]string{"tool/main.go": strings.Replace(released["tool/main.go"],
			"\tPort int\n", "\t// Port is the listen port.\n\tPort int\n", 1)}},
		{name: "go layout only", write: map[string]string{"tool/main.go": strings.Replace(released["tool/main.go"],
			"x := 1; fmt.Println", "x := 1\n\tfmt.Println", 1)}},
		{name: "go code", write: map[string]string{"tool/main.go": strings.Replace(released["tool/main.go"],
			"x := 1", "x := 2", 1)},
			refused: []string{"tool/main.go: changes Go code, not only comments; a change to code needs a patch release"}},
		{name: "go string that looks like a comment", write: map[string]string{"tool/main.go": strings.Replace(released["tool/main.go"],
			"a // not a comment", "a // still not a comment", 1)},
			refused: []string{"tool/main.go: changes Go code"}},
		{name: "go embed directive", write: map[string]string{"tool/main.go": strings.Replace(released["tool/main.go"],
			"//go:embed banner.txt", "//go:embed other.txt", 1)},
			refused: []string{"tool/main.go: changes a directive comment"}},
		{name: "go embed directive moved", write: map[string]string{"tool/main.go": strings.Replace(strings.Replace(released["tool/main.go"],
			"//go:embed banner.txt\nvar banner string", "var banner string", 1), "var other string", "//go:embed banner.txt\nvar other string", 1)},
			refused: []string{"tool/main.go: changes a directive comment"}},
		{name: "go noinline directive moved", write: map[string]string{"tool/main.go": strings.Replace(strings.Replace(released["tool/main.go"],
			"//go:noinline\nfunc helper", "func helper", 1), "func second", "//go:noinline\nfunc second", 1)},
			refused: []string{"tool/main.go: changes a directive comment"}},
		{name: "go comment added before a directive", write: map[string]string{"tool/main.go": strings.Replace(released["tool/main.go"],
			"//go:noinline\n", "// helper does nothing.\n//\n//go:noinline\n", 1)}},
		{name: "go build constraint added", write: map[string]string{"tool/main.go": "//go:build linux\n\n" + released["tool/main.go"]},
			refused: []string{"tool/main.go: changes a directive comment"}},
		{name: "go cgo preamble", write: map[string]string{"cgo/c.go": strings.Replace(released["cgo/c.go"],
			"// Hello says hello.", "// Hello greets.", 1)},
			refused: []string{`cgo/c.go: imports "C"`}},
		{name: "go file added", write: map[string]string{"tool/extra.go": "package main\n"},
			refused: []string{"tool/extra.go: added"}},
		{name: "embedded file", write: map[string]string{"tool/banner.txt": "bye\n"},
			refused: []string{"tool/banner.txt: changed, and it is not documentation"}},
		{name: "go.mod", write: map[string]string{"go.mod": "module example.com/demo\n\ngo 1.27\n"},
			refused: []string{"go.mod: changed, and it is not documentation"}},
		{name: "json", write: map[string]string{"assets/logo.json": "{\"a\": 1}\n"},
			refused: []string{"assets/logo.json: changed"}},
		{name: "markdown symlink", symlink: [2]string{"../README.md", "docs/link.md"},
			refused: []string{"docs/link.md: not a regular file"}},
		{name: "markdown symlink removed", base: func(r *gittest.Repo) {
			if err := os.Symlink("../README.md", filepath.Join(r.Dir, "docs", "old.md")); err != nil {
				t.Fatal(err)
			}
		}, remove: []string{"docs/old.md"},
			refused: []string{"docs/old.md: was not a regular file"}},
		{name: "markdown embedded into CUE", base: func(r *gittest.Repo) {
			r.Write(map[string]string{"opm/notes.cue": "@extern(embed)\n\npackage opm\n\nnotes: _ @embed(file=../docs/guide.md, type=text)\n"})
		}, write: map[string]string{"docs/guide.md": "Guide, changed.\n", "README.md": "# Demo!\n"},
			refused: []string{"README.md: opm/notes.cue embeds files into CUE values (@extern(embed))", "docs/guide.md: opm/notes.cue embeds"}},
		{name: "several refusals", write: map[string]string{"go.mod": "module x\n", "tool/main.go": strings.Replace(released["tool/main.go"], "x := 1", "x := 3", 1), "README.md": "ok\n"},
			refused: []string{"go.mod: changed", "tool/main.go: changes Go code"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gittest.New(t, "")
			r.Write(released)
			if tc.base != nil {
				tc.base(r)
			}
			base := r.CommitAt(gittest.Date, "release")
			r.Write(tc.write)
			for _, p := range tc.remove {
				r.Git("rm", "-q", p)
			}
			if tc.rename[0] != "" {
				r.Git("mv", tc.rename[0], tc.rename[1])
			}
			if tc.symlink[0] != "" {
				if err := os.Symlink(tc.symlink[0], filepath.Join(r.Dir, tc.symlink[1])); err != nil {
					t.Fatal(err)
				}
			}
			head := r.CommitAt(gittest.Date, "fix")
			got, err := Repo{Dir: r.Dir}.DocumentationOnly(t.Context(), base, head)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tc.refused) {
				t.Fatalf("refused %v, want %d refusal(s) %v", got, len(tc.refused), tc.refused)
			}
			for i, want := range tc.refused {
				if s := got[i].String(); !strings.Contains(s, want) {
					t.Errorf("refusal %d is %q, want it to contain %q", i, s, want)
				}
			}
		})
	}
}

func TestCueTokensInterpolation(t *testing.T) {
	src := []byte("a: \"x\\(f(\"y\\(b)\"))z\" // c\nb: 1\n")
	toks, err := cueTokens("x.cue", src)
	if err != nil {
		t.Fatal(err)
	}
	parts := make([]string, 0, len(toks))
	for _, tk := range toks {
		parts = append(parts, tk.kind+"="+tk.lit)
	}
	got := strings.Join(parts, " ")
	want := `IDENT=a := INTERPOLATION="x\( (= IDENT=f (= INTERPOLATION="y\( (= IDENT=b )= INTERPOLATION=)" )= )= INTERPOLATION=)z" ,= IDENT=b := INT=1 ,=`
	if got != want {
		t.Fatalf("tokens\n got %s\nwant %s", got, want)
	}
}
