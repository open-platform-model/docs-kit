package cobra

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// golden is the dump cobradump's own test writes from its fixture tree, so
// the producer and the consumer of the format read one file.
const golden = "../../../cobradump/testdata/dump.golden.json"

func goldenDump(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

var opts = Options{Section: "reference/cli/", Title: "CLI Reference", Description: "Every command.", Weight: 2, Citations: "strip"}

func TestFromDump(t *testing.T) {
	m, err := FromDump(goldenDump(t), opts)
	if err != nil {
		t.Fatal(err)
	}
	pages := make([]string, 0, len(m.CLI.Commands))
	for _, c := range m.CLI.Commands {
		pages = append(pages, c.Page)
		for _, s := range c.Commands {
			if s.Page != "" {
				t.Fatalf("%s carries a page; only top-level commands do", s.Path)
			}
		}
	}
	if got := strings.Join(pages, ","); got != "reference/cli/demo-completion,reference/cli/demo-version,reference/cli/demo-widget" {
		t.Fatalf("pages = %s", got)
	}
	data, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	back, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := back.Encode()
	if !bytes.Equal(again, data) {
		t.Fatal("the doc model does not round-trip")
	}
}

func TestFromDumpRefuses(t *testing.T) {
	dump := string(goldenDump(t))
	for _, c := range []struct{ name, dump, want string }{
		{"another schema", strings.Replace(dump, DumpSchema, "docs.opmodel.dev/cobradump/v2", 1), `schema "docs.opmodel.dev/cobradump/v2"; this opm-docs reads "docs.opmodel.dev/cobradump/v1"`},
		{"an unknown field", strings.Replace(dump, `"runnable": false,`, `"runnable": false, "hidden": true,`, 1), "unknown field"},
		{"a wrong path", strings.Replace(dump, `"path": "demo widget create"`, `"path": "demo create"`, 1), `"demo create"`},
		{"no root", strings.Replace(dump, `"name": "demo",`, `"name": "",`, 1), "no root command"},
		{"a bad flag", strings.Replace(dump, `"name": "namespace"`, `"name": "--namespace"`, 1), `"--namespace"`},
		{"an upper-case command", strings.ReplaceAll(strings.ReplaceAll(dump, "demo version", "demo Version"), `"name": "version"`, `"name": "Version"`), "lower-case kebab-case"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.dump == dump {
				t.Fatal("the fixture edit did not apply")
			}
			_, err := FromDump([]byte(c.dump), opts)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
		})
	}
}
