package crd

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/doctext"
)

var update = flag.Bool("update", false, "rewrite the golden doc model")

// fixtureOptions is the fixture tree's crd source.
func fixtureOptions(doc doctext.Policy) Options {
	weight := 7
	return Options{
		Root: "testdata/tree", Dir: "./config/crd/bases", Samples: "./config/samples",
		HideSamplesMatching: []string{"testing.opmodel.dev"},
		StripLabels:         map[string]string{"app.kubernetes.io/name": "opm-operator", "app.kubernetes.io/managed-by": "kustomize"},
		Page:                Page{Path: "reference/widgets.md", Title: "Widget resources", Description: "One entry per widget kind.", Weight: &weight},
		Order:               []string{"Widget"},
		ReconciledBy:        map[string]string{"Widget": "widget"},
		Doc:                 doc,
	}
}

func TestGolden(t *testing.T) {
	m, err := Extract(fixtureOptions(doctext.Link))
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "crd.golden.json")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run go test -run TestGolden -update)", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("data/crd.json differs from %s (run go test -run TestGolden -update and review the diff)", golden)
	}
	back, err := Decode(got)
	if err != nil || len(back.Kinds) != 3 {
		t.Fatalf("decode: %v", err)
	}
}

func kindOf(t *testing.T, m *Model, name string) Kind {
	t.Helper()
	for i := range m.Kinds {
		if m.Kinds[i].Kind == name {
			return m.Kinds[i]
		}
	}
	t.Fatalf("no kind %s", name)
	return Kind{}
}

func TestModel(t *testing.T) {
	m, err := Extract(fixtureOptions(doctext.Link))
	if err != nil {
		t.Fatal(err)
	}
	order := make([]string, 0, len(m.Kinds))
	for i := range m.Kinds {
		order = append(order, m.Kinds[i].Kind)
	}
	if got := strings.Join(order, ","); got != "Widget,Gadget,Sprocket" {
		t.Errorf("order %s: want the configured kinds first, then by name", got)
	}
}

func TestWidget(t *testing.T) {
	m, err := Extract(fixtureOptions(doctext.Link))
	if err != nil {
		t.Fatal(err)
	}
	w := kindOf(t, m, "Widget")
	if w.Sample == nil || w.Sample.File != "config/samples/example.dev_v1_widget.yaml" {
		t.Fatalf("Widget sample %+v: want the kubebuilder-named file", w.Sample)
	}
	y := w.Sample.YAML
	for _, bad := range []string{"OCIRepository", "second-widget", "large-widget", "#", "managed-by"} {
		if strings.Contains(y, bad) {
			t.Errorf("sample holds %q:\n%s", bad, y)
		}
	}
	for _, good := range []string{"app.kubernetes.io/name: demo", "tier: front", "name: widget-sample"} {
		if !strings.Contains(y, good) {
			t.Errorf("sample lacks %q:\n%s", good, y)
		}
	}
	if w.ReconciledBy == nil || *w.ReconciledBy != "widget" {
		t.Errorf("reconciledBy %v", w.ReconciledBy)
	}
	if w.Summary != "Widget asks the controller to build one widget ([0021:D4](/enhancements/0021/decisions/))." {
		t.Errorf("summary %q", w.Summary)
	}
}

func TestGadgetAndSprocket(t *testing.T) {
	m, err := Extract(fixtureOptions(doctext.Link))
	if err != nil {
		t.Fatal(err)
	}
	g := kindOf(t, m, "Gadget")
	if g.Sample != nil {
		t.Errorf("Gadget sample %+v: a sample naming testing.opmodel.dev is hidden", g.Sample)
	}
	if len(g.Spec) != 1 || g.Spec[0].Path != "spec.module" || !g.Spec[0].Required {
		t.Errorf("Gadget spec %+v: want spec.module required", g.Spec)
	}
	if g.ReconciledBy != nil {
		t.Errorf("Gadget reconciledBy %v", *g.ReconciledBy)
	}

	s := kindOf(t, m, "Sprocket")
	if s.Sample != nil {
		t.Errorf("Sprocket has no sample file, got %+v", s.Sample)
	}
	if s.Summary != "Sprocket has no sample." {
		t.Errorf("Sprocket summary %q: SPEC.md pointers are removed", s.Summary)
	}
}

func TestStrip(t *testing.T) {
	m, err := Extract(fixtureOptions(doctext.Strip))
	if err != nil {
		t.Fatal(err)
	}
	w := kindOf(t, m, "Widget")
	if w.Summary != "Widget asks the controller to build one widget." {
		t.Errorf("summary %q", w.Summary)
	}
}

func TestNoSamples(t *testing.T) {
	o := fixtureOptions(doctext.Strip)
	o.Samples = ""
	m, err := Extract(o)
	if err != nil {
		t.Fatal(err)
	}
	for i := range m.Kinds {
		if m.Kinds[i].Sample != nil {
			t.Errorf("%s has a sample without samples configured", m.Kinds[i].Kind)
		}
	}
}

func TestSplitDocuments(t *testing.T) {
	docs := splitDocuments([]byte("---\na: 1\n--- # two\nb: 2\n---x: 3\n---\n\n"))
	if len(docs) != 2 || string(docs[0]) != "a: 1\n" || string(docs[1]) != "b: 2\n---x: 3\n" {
		t.Errorf("docs %q", docs)
	}
}

func TestRefusals(t *testing.T) {
	write := func(t *testing.T, files map[string]string) string {
		t.Helper()
		root := t.TempDir()
		for name, body := range files {
			p := filepath.Join(root, name)
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return root
	}
	crd := func(kind, extra string) string {
		return "apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\nspec:\n  group: example.dev\n  names:\n    kind: " + kind +
			"\n    plural: " + strings.ToLower(kind) + "s\n  scope: Namespaced\n  versions:\n  - name: v1\n    served: true\n    storage: true\n    schema:\n      openAPIV3Schema:\n        type: object\n" + extra
	}
	for _, c := range []struct {
		name  string
		files map[string]string
		opts  func(*Options)
		want  string
	}{
		{"empty dir", map[string]string{"c/x.txt": ""}, nil, "holds no .yaml file"},
		{"other kind", map[string]string{"c/x.yaml": "apiVersion: v1\nkind: ConfigMap\n"}, nil, "c/x.yaml: holds a v1 ConfigMap"},
		{"bad yaml", map[string]string{"c/x.yaml": "a: [\n"}, nil, "c/x.yaml: does not parse as YAML"},
		{"bad sample", map[string]string{"c/x.yaml": crd("Widget", ""), "s/example.dev_v1_widget.yaml": "a: [\n"}, nil, "s/example.dev_v1_widget.yaml: does not parse as YAML"},
		{"duplicate kind", map[string]string{"c/a.yaml": crd("Widget", ""), "c/b.yaml": crd("Widget", "")}, nil, "c/a.yaml and c/b.yaml both define the kind Widget"},
		{"oneOf", map[string]string{"c/x.yaml": crd("Widget", "        oneOf:\n        - required: [a]\n")}, nil, "schema constructs the reference does not render: the object: oneOf"},
		{"stale reconciledBy", map[string]string{"c/x.yaml": crd("Widget", "")}, func(o *Options) { o.ReconciledBy = map[string]string{"Gizmo": "gizmo"} }, "reconciledBy names the kind Gizmo"},
		{"stale order", map[string]string{"c/x.yaml": crd("Widget", "")}, func(o *Options) { o.Order = []string{"Gizmo"} }, "order names the kind Gizmo"},
		{"leaves tree", map[string]string{"c/x.yaml": crd("Widget", "")}, func(o *Options) { o.Dir = "./../c" }, "leaves the source tree"},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := Options{Root: write(t, c.files), Dir: "./c", Samples: "./s", Doc: doctext.Strip}
			if c.opts != nil {
				c.opts(&o)
			}
			_, err := Extract(o)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err %v, want %q", err, c.want)
			}
		})
	}
}
