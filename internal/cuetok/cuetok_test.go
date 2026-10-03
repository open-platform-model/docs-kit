package cuetok

import (
	"strings"
	"testing"
)

func TestScanInterpolation(t *testing.T) {
	src := []byte("a: \"x\\(f(\"y\\(b)\"))z\" // c\nb: 1\n")
	toks, err := Scan("x.cue", src)
	if err != nil {
		t.Fatal(err)
	}
	parts := make([]string, 0, len(toks))
	for _, tk := range toks {
		parts = append(parts, tk.Kind+"="+tk.Lit)
	}
	got := strings.Join(parts, " ")
	want := `IDENT=a := INTERPOLATION="x\( (= IDENT=f (= INTERPOLATION="y\( (= IDENT=b )= INTERPOLATION=)" )= )= INTERPOLATION=)z" ,= IDENT=b := INT=1 ,=`
	if got != want {
		t.Fatalf("tokens\n got %s\nwant %s", got, want)
	}
}

func TestEqualIgnoresCommentsAndLayout(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"comment added", "a: 1\nb: 2\n", "// lead\na: 1 // trailing\n\nb: 2\n", true},
		{"written comma", "a: 1\nb: 2\n", "a: 1, b: 2\n", true},
		{"value changed", "a: 1\n", "a: 2\n", false},
		{"matchN changed", "matchN(1, [{a!: _}, {b!: _}])\n", "matchN(2, [{a!: _}, {b!: _}])\n", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, err := Scan("a.cue", []byte(c.a))
			if err != nil {
				t.Fatal(err)
			}
			b, err := Scan("b.cue", []byte(c.b))
			if err != nil {
				t.Fatal(err)
			}
			if got := Equal(a, b); got != c.want {
				t.Fatalf("Equal = %t, want %t", got, c.want)
			}
		})
	}
}

func TestScanRefusesBrokenSource(t *testing.T) {
	if _, err := Scan("x.cue", []byte("a: \"unterminated\n")); err == nil {
		t.Fatal("scanned an unterminated string")
	}
}
