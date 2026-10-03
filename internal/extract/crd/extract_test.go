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
	if w.Summary != "Widget asks the controller to build one widget (0021:D4)." {
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

// writeTree writes files under a new root, with an empty samples
// directory s/ unless a file creates it.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "s"), 0o755); err != nil {
		t.Fatal(err)
	}
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

// crdYAML is a minimal CRD of kind; extra is appended to its schema, at
// the openAPIV3Schema's indent.
func crdYAML(kind, extra string) string {
	return "apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\nspec:\n  group: example.dev\n  names:\n    kind: " + kind +
		"\n    plural: " + strings.ToLower(kind) + "s\n  scope: Namespaced\n  versions:\n  - name: v1\n    served: true\n    storage: true\n    schema:\n      openAPIV3Schema:\n        type: object\n" + extra
}

// specField is a CRD whose spec has one property port, with body at the
// property's indent.
func specField(body string) string {
	return crdYAML("Widget", "        properties:\n          spec:\n            type: object\n            properties:\n              port:\n"+body)
}

func TestRefusals(t *testing.T) {
	crd := crdYAML
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
		{"int-or-string", map[string]string{"c/x.yaml": specField("                anyOf:\n                - type: integer\n                - type: string\n                x-kubernetes-int-or-string: true\n")}, nil, "spec.port: an int-or-string field (x-kubernetes-int-or-string), which is not supported yet"},
		{"stale reconciledBy", map[string]string{"c/x.yaml": crd("Widget", "")}, func(o *Options) { o.ReconciledBy = map[string]string{"Gizmo": "gizmo"} }, "reconciledBy names the kind Gizmo"},
		{"stale order", map[string]string{"c/x.yaml": crd("Widget", "")}, func(o *Options) { o.Order = []string{"Gizmo"} }, "order names the kind Gizmo"},
		{"leaves tree", map[string]string{"c/x.yaml": crd("Widget", "")}, func(o *Options) { o.Dir = "./../c" }, "leaves the source tree"},
		{"samples missing", map[string]string{"c/x.yaml": crd("Widget", "")}, func(o *Options) { o.Samples = "./nosuch" }, "samples ./nosuch does not exist; create it or remove samples"},
		{"kind name", map[string]string{"c/x.yaml": crd("Widget<script>", "")}, nil, `c/x.yaml: kind "Widget<script>" is not a Kubernetes kind name`},
		{"scope", map[string]string{"c/x.yaml": strings.Replace(crd("Widget", ""), "scope: Namespaced", "scope: <b>x</b>", 1)}, nil, `c/x.yaml: Widget: scope "<b>x</b>" is neither Namespaced nor Cluster`},
		{"column type", map[string]string{"c/x.yaml": strings.Replace(crd("Widget", ""), "  - name: v1\n", "  - name: v1\n    additionalPrinterColumns:\n    - name: Ready\n      type: <img>\n      jsonPath: .x\n", 1)}, nil, `c/x.yaml: Widget: printer column "Ready" has type "<img>"; want one of integer, number, string, boolean, date`},
		{"unknown CRD field", map[string]string{"c/x.yaml": strings.Replace(crd("Widget", ""), "  scope: Namespaced\n", "  scope: Namespaced\n  conversion:\n    strategy: None\n", 1)}, nil, `c/x.yaml: Widget: spec.conversion: unknown field "conversion"; the crd extractor reads only the fields its page shows`},
		{"unknown schema field", map[string]string{"c/x.yaml": specField("                type: string\n                x-kubernetes-foo: Port\n")}, nil, `c/x.yaml: Widget: spec.port: unknown field "x-kubernetes-foo"`},
		{"unknown item field", map[string]string{"c/x.yaml": specField("                type: array\n                items:\n                  type: object\n                  properties:\n                    name:\n                      type: string\n                      dependencies: {}\n")}, nil, `c/x.yaml: Widget: spec.port[].name: unknown field "dependencies"`},
		{"unknown root field", map[string]string{"c/x.yaml": crd("Widget", "        $schema: x\n")}, nil, `c/x.yaml: Widget: the object: unknown field "$schema"`},
		{"unknown CEL field", map[string]string{"c/x.yaml": specField("                type: string\n                x-kubernetes-validations:\n                - rule: self != ''\n                  severity: warn\n")}, nil, `c/x.yaml: Widget: spec.port: unknown field "severity"`},
		{"preserveUnknownFields true", map[string]string{"c/x.yaml": strings.Replace(crd("Widget", ""), "  scope: Namespaced\n", "  scope: Namespaced\n  preserveUnknownFields: true\n", 1)}, nil, "c/x.yaml: spec.preserveUnknownFields: true is not supported"},
		{"multipleOf", map[string]string{"c/x.yaml": specField("                type: integer\n                multipleOf: 2\n")}, nil, "spec.port: multipleOf"},
		{"format", map[string]string{"c/x.yaml": specField("                type: string\n                format: byte\n")}, nil, `spec.port: format "byte"`},
		{"messageExpression", map[string]string{"c/x.yaml": specField("                type: string\n                x-kubernetes-validations:\n                - rule: self != ''\n                  messageExpression: self\n")}, nil, "spec.port: messageExpression"},
		{"embedded resource", map[string]string{"c/x.yaml": specField("                type: object\n                x-kubernetes-embedded-resource: true\n")}, nil, "spec.port: x-kubernetes-embedded-resource"},
		{"tuple items", map[string]string{"c/x.yaml": specField("                type: array\n                items:\n                - type: string\n")}, nil, "spec.port: tuple items"},
		{"metadata constraints", map[string]string{"c/x.yaml": crd("Widget", "        properties:\n          metadata:\n            type: object\n            properties:\n              name:\n                maxLength: 20\n                type: string\n")}, nil, "Widget: metadata carries constraints the page does not show"},
		{"top-level property", map[string]string{"c/x.yaml": crd("Widget", "        properties:\n          data:\n            type: object\n")}, nil, `Widget: top-level property "data" is not shown`},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := Options{Root: writeTree(t, c.files), Dir: "./c", Samples: "./s", Doc: doctext.Strip}
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

func TestSamplesMissingInBackfill(t *testing.T) {
	o := Options{Root: writeTree(t, map[string]string{"c/x.yaml": crdYAML("Widget", "")}), Dir: "./c", Samples: "./nosuch", Outside: true, Doc: doctext.Strip}
	if _, err := Extract(o); err != nil {
		t.Fatalf("a backfill without the samples directory: %v", err)
	}
}

func TestRefusesSymlinks(t *testing.T) {
	root := writeTree(t, map[string]string{"real/x.yaml": crdYAML("Widget", ""), "real/sample.yaml": "apiVersion: example.dev/v1\nkind: Widget\n"})
	if err := os.MkdirAll(filepath.Join(root, "c"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "real", "x.yaml"), filepath.Join(root, "c", "x.yaml")); err != nil {
		t.Fatal(err)
	}
	_, err := Extract(Options{Root: root, Dir: "./c", Doc: doctext.Strip})
	if err == nil || !strings.Contains(err.Error(), "c/x.yaml is not a regular file") {
		t.Errorf("symlinked CRD: %v", err)
	}
	if err := os.Rename(filepath.Join(root, "real", "x.yaml"), filepath.Join(root, "c", "x.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "real", "sample.yaml"), filepath.Join(root, "s", "example.dev_v1_widget.yaml")); err != nil {
		t.Fatal(err)
	}
	_, err = Extract(Options{Root: root, Dir: "./c", Samples: "./s", Doc: doctext.Strip})
	if err == nil || !strings.Contains(err.Error(), "s/example.dev_v1_widget.yaml is not a regular file") {
		t.Errorf("symlinked sample: %v", err)
	}
}

func TestRefusesSymlinkedDirs(t *testing.T) {
	root := writeTree(t, map[string]string{"real/x.yaml": crdYAML("Widget", "")})
	if err := os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "c")); err != nil {
		t.Fatal(err)
	}
	_, err := Extract(Options{Root: root, Dir: "./c", Doc: doctext.Strip})
	if err == nil || !strings.Contains(err.Error(), "dir ./c is a symbolic link; name the directory itself") {
		t.Errorf("symlinked dir: %v", err)
	}
	if err := os.Symlink(filepath.Join(root, "s"), filepath.Join(root, "samples")); err != nil {
		t.Fatal(err)
	}
	_, err = Extract(Options{Root: root, Dir: "./real", Samples: "./samples", Doc: doctext.Strip})
	if err == nil || !strings.Contains(err.Error(), "samples ./samples is a symbolic link; name the directory itself") {
		t.Errorf("symlinked samples: %v", err)
	}
}

// A directory under a linked parent that leaves the source tree is
// refused, though no path element of its own is a link.
func TestRefusesDirsOutsideTheTree(t *testing.T) {
	outside := writeTree(t, map[string]string{"c/x.yaml": crdYAML("Widget", "")})
	root := writeTree(t, map[string]string{"real/x.yaml": crdYAML("Widget", "")})
	if err := os.Symlink(outside, filepath.Join(root, "elsewhere")); err != nil {
		t.Fatal(err)
	}
	_, err := Extract(Options{Root: root, Dir: "./elsewhere/c", Doc: doctext.Strip})
	if err == nil || !strings.Contains(err.Error(), "dir ./elsewhere/c resolves outside the source tree") {
		t.Errorf("dir: %v", err)
	}
	_, err = Extract(Options{Root: root, Dir: "./real", Samples: "./elsewhere/s", Doc: doctext.Strip})
	if err == nil || !strings.Contains(err.Error(), "samples ./elsewhere/s resolves outside the source tree") {
		t.Errorf("samples: %v", err)
	}
}

func TestSampleKeepsLargeIntegers(t *testing.T) {
	root := writeTree(t, map[string]string{
		"c/x.yaml":                     crdYAML("Widget", ""),
		"s/example.dev_v1_widget.yaml": "apiVersion: example.dev/v1\nkind: Widget\nspec:\n  big: 12345678901234567890\n  ratio: 0.5\n",
	})
	m, err := Extract(Options{Root: root, Dir: "./c", Samples: "./s", Doc: doctext.Strip})
	if err != nil {
		t.Fatal(err)
	}
	if y := m.Kinds[0].Sample.YAML; !strings.Contains(y, "big: 12345678901234567890\n") || !strings.Contains(y, "ratio: 0.5\n") {
		t.Errorf("sample:\n%s", y)
	}
	if strings.Join(m.Read, ",") != "c/x.yaml,s/example.dev_v1_widget.yaml" {
		t.Errorf("read %v", m.Read)
	}
}

// Keys that enforce no rule the page would miss are accepted and ignored:
// an embedded LocalObjectReference with x-kubernetes-map-type, title,
// example, externalDocs, a CEL reason and fieldPath, the CRD's status
// block, preserveUnknownFields: false, and a version's deprecation.
func TestIgnoredKeys(t *testing.T) {
	doc := strings.Replace(crdYAML("Widget", `        title: Widget
        properties:
          spec:
            type: object
            properties:
              secretRef:
                description: SecretRef names a Secret.
                externalDocs:
                  url: https://example.com
                properties:
                  name:
                    default: ""
                    description: Name of the referent.
                    example: my-secret
                    type: string
                type: object
                x-kubernetes-map-type: atomic
                x-kubernetes-validations:
                - fieldPath: .name
                  message: name is required
                  reason: FieldValueRequired
                  rule: self.name != ''
`), "  - name: v1\n", "  - name: v1\n    deprecated: true\n    deprecationWarning: use v2\n", 1)
	doc = strings.Replace(doc, "  scope: Namespaced\n", "  scope: Namespaced\n  preserveUnknownFields: false\n", 1)
	doc += "status:\n  acceptedNames:\n    kind: \"\"\n    plural: \"\"\n  conditions: null\n  storedVersions: null\n"
	m, err := Extract(Options{Root: writeTree(t, map[string]string{"c/x.yaml": doc}), Dir: "./c", Doc: doctext.Strip})
	if err != nil {
		t.Fatal(err)
	}
	k := m.Kinds[0]
	if len(k.Spec) != 2 || k.Spec[0].Path != "spec.secretRef" || k.Spec[1].Path != "spec.secretRef.name" {
		t.Fatalf("spec %+v", k.Spec)
	}
	if len(k.Rules) != 1 || k.Rules[0].Message != "name is required" {
		t.Errorf("rules %+v", k.Rules)
	}
}
