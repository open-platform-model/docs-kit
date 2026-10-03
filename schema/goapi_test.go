package schema

import (
	"os"
	"strings"
	"testing"
)

// The library's planned config validates; the cases below each break one
// rule.
func TestGoAPIConfig(t *testing.T) {
	library, err := os.ReadFile("../internal/extract/goapi/testdata/library-docs-kit.cue")
	if err != nil {
		t.Fatal(err)
	}
	src := func(fields string) string {
		return `bundles: library: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/go-api/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{kind: "go-api", ` + fields + `}]
}`
	}
	const base = `module: "./", root: "./opm", packages: ["./opm/..."], section: "reference/go-api/", title: "Go API", description: "d"`
	with := func(from, to string) string { return src(strings.Replace(base, from, to, 1)) }
	for _, c := range []struct {
		name, cue string
		ok        bool
	}{
		{"library", string(library), true},
		{"minimal", src(base), true},
		{"weight and citations", src(base + `, weight: 3, citations: "link"`), true},
		{"several patterns", with(`["./opm/..."]`, `["./opm/kernel", "./opm/helper/..."]`), true},
		{"no patterns", with(`["./opm/..."]`, `[]`), false},
		{"absolute module", with(`module: "./"`, `module: "/"`), false},
		{"module with ..", with(`module: "./"`, `module: "./../x"`), false},
		{"root without ./", with(`root: "./opm"`, `root: "opm"`), false},
		{"root with ..", with(`root: "./opm"`, `root: "./opm/.."`), false},
		{"pattern without ./", with(`["./opm/..."]`, `["opm/..."]`), false},
		{"pattern with ..", with(`["./opm/..."]`, `["./../x/..."]`), false},
		{"section without slash", with(`"reference/go-api/"`, `"reference/go-api"`), false},
		{"empty title", with(`title: "Go API"`, `title: ""`), false},
		{"weight zero", src(base + `, weight: 0`), false},
		{"unknown field", src(base + `, intro: "x"`), false},
		{"bad citations", src(base + `, citations: "keep"`), false},
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
