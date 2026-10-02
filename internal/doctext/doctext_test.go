package doctext

import (
	"strings"
	"testing"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/parser"
)

func TestSplitDoc(t *testing.T) {
	cases := []struct {
		name, doc, desc string
		want            []string
		wantErr         string
	}{
		{name: "description only", doc: "A thing.", desc: "A thing"},
		{name: "rest of the paragraph and later paragraphs", doc: "A thing. It renders\nan object.\n\nSecond paragraph.", desc: "A thing",
			want: []string{"It renders an object.", "Second paragraph."}},
		{name: "wrapped first sentence", doc: "A long\nthing. More.", desc: "A long thing", want: []string{"More."}},
		{name: "no doc comment", doc: "", desc: "A thing", wantErr: "no doc comment"},
		{name: "disagreeing doc comment", doc: "Another thing.", desc: "A thing", wantErr: "must open with the description"},
		{name: "description without its period", doc: "A thing, and more.", desc: "A thing", wantErr: "must open with the description"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := SplitDoc(c.doc, c.desc)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("err = %v, want one containing %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(got, "|") != strings.Join(c.want, "|") {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestClean(t *testing.T) {
	cases := map[string]string{
		// The catalog's rules.
		"failing the render loudly (0010 D28) beats it":           "failing the render loudly beats it",
		"refuses it structurally (0015 D10), so nothing is gated": "refuses it structurally, so nothing is gated",
		"with no fallback (0019:D22). Required.":                  "with no fallback. Required.",
		"per 0010:D4:R2 and 0010:D49 the key moves":               "the key moves",
		"no citation here (see docs/x.md)":                        "no citation here (see docs/x.md)",
		"Exactly one provider serves it (0010:D32).":              "Exactly one provider serves it.",
		"both apply (0010:D28/D9; 0015:D1:R2/R3)":                 "both apply",
		// Core's rules.
		"Reads the label. See SPEC.md § 3.2.":                "Reads the label.",
		"as SPEC.md § 2.2 states":                            "as states",
		"measured (0007 experiment 2) in practice":           "measured in practice",
		"see enhancements/0007/experiments/02-load for data": "see for data",
		"untouched text":                                     "untouched text",
	}
	for in, want := range cases {
		if got := Clean(in); got != want {
			t.Errorf("Clean(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCleanCode(t *testing.T) {
	code := "spec: x: #S\n\n#S: {\n\t// The name, with no fallback (0019\n\t// D22). Required.\n\tname!: string // Example: \"a\" (0010:D4)\n\t// Untouched\n\t//   indented example\n\tother?: int // (0010:D9)\n}"
	want := "spec: x: #S\n\n#S: {\n\t// The name, with no fallback. Required.\n\tname!: string // Example: \"a\"\n\t// Untouched\n\t//   indented example\n\tother?: int\n}"
	if got := CleanCode(code); got != want {
		t.Errorf("CleanCode:\n%s\nwant:\n%s", got, want)
	}
	long := "#S: {\n\t// A long paragraph that cites a decision (0010:D28) and keeps going for a while so that it\n\t// must wrap again after the citation is removed from it.\n\tx: int\n}"
	got := CleanCode(long)
	for _, l := range strings.Split(got, "\n") {
		if len(strings.ReplaceAll(l, "\t", "    ")) > 80 {
			t.Errorf("line longer than 80 columns: %q", l)
		}
	}
	if strings.Contains(got, "0010") {
		t.Errorf("citation left: %s", got)
	}
}

func TestMaintainerComments(t *testing.T) {
	src := `package p

// WHY: 0010 D28 explains this.
// It goes on.

// The schema. Doc.
// WHY the doc says this, which is dropped.
// More doc.
#S: {
	// WHY a block inside.
	a: int

	//// banner
	b: int
	// kept
	c: int
}
`
	f, err := parser.ParseFile("p.cue", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var def *ast.Field
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.Field); ok {
			def = fd
		}
	}
	StripMaintainerComments(def)
	if got := DocText(def); got != "The schema. Doc.\nMore doc." {
		t.Errorf("DocText = %q", got)
	}
	out, err := format.Node(def)
	if err != nil {
		t.Fatal(err)
	}
	s := CleanCode(string(out))
	for _, gone := range []string{"WHY", "banner", "0010"} {
		if strings.Contains(s, gone) {
			t.Errorf("%q left in:\n%s", gone, s)
		}
	}
	if !strings.Contains(s, "// kept") {
		t.Errorf("kept comment dropped:\n%s", s)
	}
}

func TestWrap(t *testing.T) {
	got := Wrap("aa bb cc dd", 5)
	if strings.Join(got, "|") != "aa bb|cc dd" {
		t.Fatalf("got %q", got)
	}
}
