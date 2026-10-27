package placeholder

import (
	"reflect"
	"strings"
	"testing"
)

func TestNamesInListOrder(t *testing.T) {
	want := []string{"cli", "core", "library", "opm-controller", "opm-controller-module", "catalog-opm"}
	if got := Names(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
}

func TestLookup(t *testing.T) {
	for _, c := range []struct {
		name    string
		kind    Kind
		project string
	}{
		{"cli", Bundle, "cli"},
		{"core", Bundle, "core"},
		{"library", Bundle, "library"},
		{"opm-controller", Bundle, "opm-controller"},
		{"opm-controller-module", Unit, ""},
		{"catalog-opm", Bundle, "catalog-opm-docs"},
	} {
		n, ok := Lookup(c.name)
		if !ok || n.Kind != c.kind || n.Project != c.project {
			t.Errorf("Lookup(%q) = %+v, %v; want kind %v project %q", c.name, n, ok, c.kind, c.project)
		}
	}
	if _, ok := Lookup("opm-controler"); ok {
		t.Error("a misspelled name is known")
	}
}

func TestScan(t *testing.T) {
	for _, c := range []struct {
		name, in string
		want     []Verdict
		msg      []string
	}{
		{"every known name", "{{opm:cli.version}} {{opm:core.version}} {{opm:library.version}} {{opm:opm-controller.version}} {{opm:opm-controller-module.version}} {{opm:catalog-opm.version}}",
			[]Verdict{Known, Known, Known, Known, Known, Known}, make([]string, 6)},
		{"misspelled name", "x {{opm:opm-controler.version}}", []Verdict{UnknownName},
			[]string{`placeholder "{{opm:opm-controler.version}}": unknown name "opm-controler"; known names: cli, core, library, opm-controller, opm-controller-module, catalog-opm`}},
		{"unknown field", "{{opm:cli.tag}}", []Verdict{UnknownField},
			[]string{`placeholder "{{opm:cli.tag}}": unknown field "tag"; write {{opm:cli.version}}`}},
		{"unclosed", "{{opm:cli.version", []Verdict{Malformed},
			[]string{`placeholder "{{opm:cli.version": not closed or not of the form {{opm:<name>.version}}`}},
		{"no field", "{{opm:cli}}", []Verdict{Malformed},
			[]string{`placeholder "{{opm:cli}}": not closed or not of the form {{opm:<name>.version}}`}},
		{"space after the braces is not a placeholder", "{{ opm:cli.version}}", nil, nil},
		{"two in a line", "v{{opm:cli.version}} and v{{opm:core.version}}.", []Verdict{Known, Known}, make([]string, 2)},
		{"known then bad", "{{opm:cli.version}} {{opm:x.version}}", []Verdict{Known, UnknownName}, []string{"", `placeholder "{{opm:x.version}}": unknown name "x"; known names: cli, core, library, opm-controller, opm-controller-module, catalog-opm`}},
		{"in a link", "[n](https://x.example/v{{opm:cli.version}})", []Verdict{Known}, []string{""}},
		{"shown shortcode", "{{</* opm/figure */>}}", nil, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := Scan(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("Scan(%q) = %+v, want %d occurrences", c.in, got, len(c.want))
			}
			for i, o := range got {
				if o.Verdict != c.want[i] {
					t.Errorf("occurrence %d verdict %v, want %v", i, o.Verdict, c.want[i])
				}
				if o.Message() != c.msg[i] {
					t.Errorf("occurrence %d message %q, want %q", i, o.Message(), c.msg[i])
				}
				if c.in[o.Start:o.End] != o.Text {
					t.Errorf("occurrence %d offsets %d:%d do not give %q", i, o.Start, o.End, o.Text)
				}
			}
		})
	}
}

func TestFill(t *testing.T) {
	values := map[string]string{"cli": "1.0.2", "opm-controller": "1.0.0-beta.10"}
	in := "run v{{opm:cli.version}}\n```bash\nopm install --version v{{opm:opm-controller.version}}\n```\n[n](https://x.example/v{{opm:cli.version}}) <digest> {{ not }} text\n"
	got, left := Fill(in, values)
	want := "run v1.0.2\n```bash\nopm install --version v1.0.0-beta.10\n```\n[n](https://x.example/v1.0.2) <digest> {{ not }} text\n"
	if got != want || len(left) != 0 {
		t.Fatalf("Fill = %q, %+v", got, left)
	}
}

func TestFillLeavesWhatItCannotSupply(t *testing.T) {
	in := "a {{opm:core.version}} b {{opm:cli.version}} c {{opm:cli.tag}} d {{opm:cli.version"
	got, left := Fill(in, map[string]string{"cli": "2.0.0"})
	if want := "a {{opm:core.version}} b 2.0.0 c {{opm:cli.tag}} d {{opm:cli.version"; got != want {
		t.Errorf("Fill = %q, want %q", got, want)
	}
	texts := make([]string, 0, len(left))
	for _, o := range left {
		texts = append(texts, o.Text)
	}
	if want := "{{opm:core.version}} {{opm:cli.tag}} {{opm:cli.version"; strings.Join(texts, " ") != want {
		t.Errorf("left %q, want %q", texts, want)
	}
}

func TestFillWithoutPlaceholderIsIdentity(t *testing.T) {
	in := "plain {{< opm/figure >}} text\n"
	if got, left := Fill(in, map[string]string{"cli": "1"}); got != in || left != nil {
		t.Errorf("Fill changed plain text: %q %v", got, left)
	}
}
