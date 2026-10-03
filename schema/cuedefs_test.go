package schema

import (
	"os"
	"strings"
	"testing"
)

// core's planned config validates; the cases below each break one rule.
func TestCueDefinitionsConfig(t *testing.T) {
	core, err := os.ReadFile("../internal/extract/cuedefs/testdata/core-docs-kit.cue")
	if err != nil {
		t.Fatal(err)
	}
	const page = `{file: "components", title: "Components", description: "d", definitions: ["#Component"]}`
	src := func(fields string) string {
		return `bundles: core: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/definitions/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{kind: "cue-definitions", ` + fields + `}]
}`
	}
	const base = `package: "./src", section: "reference/definitions/", title: "Definitions", description: "d", `
	for _, c := range []struct {
		name, cue string
		ok        bool
	}{
		{"core", string(core), true},
		{"minimal", src(base + `pages: [` + page + `]`), true},
		{"link citations", src(base + `citations: "link", pages: [` + page + `]`), true},
		{"page without definitions", src(base + `pages: [{file: "components", title: "Components", description: "d"}]`), false},
		{"page with empty definitions", src(base + `pages: [{file: "components", title: "Components", description: "d", definitions: []}]`), false},
		{"no pages", src(base + `pages: []`), false},
		{"name without #", src(base + `pages: [{file: "c", title: "C", description: "d", definitions: ["Component"]}]`), false},
		{"absolute package", src(strings.Replace(base, `"./src"`, `"/src"`, 1) + `pages: [` + page + `]`), false},
		{"section without slash", src(strings.Replace(base, `"reference/definitions/"`, `"reference/definitions"`, 1) + `pages: [` + page + `]`), false},
		{"exclude without reason", src(base + `pages: [` + page + `], exclude: {"#X": ""}`), false},
		{"exclude a non-definition", src(base + `pages: [` + page + `], exclude: {"X": "why"}`), false},
		{"package with ..", src(strings.Replace(base, `"./src"`, `"./src/../../etc"`, 1) + `pages: [` + page + `]`), false},
		{"package ending ..", src(strings.Replace(base, `"./src"`, `"./.."`, 1) + `pages: [` + page + `]`), false},
		{"intro", src(base + `intro: "Read me.", pages: [` + page + `]`), true},
		{"empty intro", src(base + `intro: "", pages: [` + page + `]`), false},
		{"weight zero", src(base + `weight: 0, pages: [` + page + `]`), false},
		{"unknown field", src(base + `module: "./src", pages: [` + page + `]`), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			ctx, _, err := Package()
			if err != nil {
				t.Fatal(err)
			}
			v := ctx.CompileString(c.cue)
			if v.Err() != nil {
				t.Fatal(v.Err())
			}
			_, err = Unify("#Config", v)
			if (err == nil) != c.ok {
				t.Fatalf("err = %v, want ok %v", err, c.ok)
			}
		})
	}
}
