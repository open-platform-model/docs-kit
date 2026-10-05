package render

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/dialect"
	"github.com/open-platform-model/docs-kit/internal/doctext"
	"github.com/open-platform-model/docs-kit/internal/extract/crd"
)

const crdTree = "../extract/crd/testdata/tree"

// crdFixture renders the crd extractor's fixture tree, through the data
// file as the build does.
func crdFixture(t *testing.T, doc doctext.Policy) Page {
	t.Helper()
	weight := 7
	m, err := crd.Extract(crd.Options{
		Root: crdTree, Dir: "./config/crd/bases", Samples: "./config/samples",
		HideSamplesMatching: []string{"testing.opmodel.dev"},
		StripLabels:         map[string]string{"app.kubernetes.io/name": "opm-operator", "app.kubernetes.io/managed-by": "kustomize"},
		Page:                crd.Page{Path: "reference/widgets.md", Title: "Widget resources", Description: "One entry per widget kind.", Weight: &weight},
		Order:               []string{"Widget"},
		ReconciledBy:        map[string]string{"Widget": "widget"},
		Doc:                 doc,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	r, err := For(crd.SchemaID)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := r.Render(data, Target{Kind: KindDocs, Root: "/docs/"})
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 {
		t.Fatalf("%d pages, want one", len(pages))
	}
	return pages[0]
}

const authoredWidgets = `---
title: "Widgets"
description: "What widgets are."
type: reference
weight: 3
---

This page lists the widget kinds.
`

func TestCRDGolden(t *testing.T) {
	p := crdFixture(t, doctext.Link)
	if p.Path != "reference/widgets.md" || !p.Completable || p.Heading != "## Widget" {
		t.Fatalf("page %s completable %v heading %q", p.Path, p.Completable, p.Heading)
	}
	completed, err := Complete(authoredWidgets, "docs/site/reference/widgets.md", p)
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"alone": p.Body, "completed": completed} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join("testdata", "golden", "crd-"+name)
			file := filepath.Join(dir, filepath.FromSlash(p.Path))
			if *update {
				_ = os.MkdirAll(filepath.Dir(file), 0o755)
				if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("%v (run go test -run TestCRDGolden -update)", err)
			}
			if string(want) != body {
				t.Errorf("%s differs from its golden page", file)
			}
			vs, err := dialect.Lint(dir, dialect.Options{Mode: dialect.Bundle, Bundle: dialect.BundleInfo{
				Kind: KindDocs, Root: "/docs/", Owns: []string{p.Path}, Pages: []string{p.Path},
			}})
			if err != nil {
				t.Fatal(err)
			}
			for _, v := range vs {
				t.Errorf("golden page breaks the dialect: %s", v)
			}
		})
	}
}

func TestCRDPageFacts(t *testing.T) {
	link := crdFixture(t, doctext.Link)
	strip := crdFixture(t, doctext.Strip)
	for _, c := range []struct {
		body, want string
		absent     bool
	}{
		{link.Body, "title: \"Widget resources\"\ndescription: \"One entry per widget kind.\"\ntype: reference\nweight: 7\n---\n\n## Widget\n", false},
		{link.Body, "Widget asks the controller to build one widget ([0021:D4](/enhancements/0021/decisions/)).", false},
		{link.Body, "see \\#Widget and [0021:D4/D9](/enhancements/0021/decisions/).", false},
		{link.Body, "| `spec.color` | `string` | No | `\"blue\"` | Color picks the paint, a \\<name\\> or `a\\|b`. |", false},
		{link.Body, "| Size\\|Count | integer | `.spec.size` | 1 |", false},
		{link.Body, "| the object | CEL rule `size(self.metadata.name) < 20`; refused with: name must be short | API server |", false},
		{link.Body, "| `spec` | CEL rule `self.size >= oldSelf.size`; refused with: size cannot shrink ([0021:D4](/enhancements/0021/decisions/)) | API server |", false},
		{link.Body, "| `spec.parts` | At most one item per `name` | API server |", false},
		{link.Body, "| `spec.size` | Greater than 0 | API server |", false},
		{link.Body, "### Served by\n\nThe `widget` reconciler watches every Widget.\n", false},
		{link.Body, "### Example\n\n```yaml\napiVersion: example.dev/v1\nkind: Widget\nmetadata:\n  labels:\n    app.kubernetes.io/name: demo\n    tier: front\n  name: widget-sample\nspec:\n  color: red\n  size: 3\n```\n", false},
		{link.Body, "## Gadget\n\nGadget is a cluster-wide gadget.\n\n### At a glance", false},
		{link.Body, "watches every Gadget", true},
		{link.Body, "watches every Sprocket", true},
		{strip.Body, "build one widget.", false},
		{strip.Body, "/enhancements/", true},
	} {
		if got := strings.Contains(c.body, c.want); got == c.absent {
			t.Errorf("page holds %q: %v, want %v", c.want, got, !c.absent)
		}
	}
	gadget := link.Body[strings.Index(link.Body, "## Gadget"):strings.Index(link.Body, "## Sprocket")]
	if strings.Contains(gadget, "### Example") {
		t.Errorf("the hidden Gadget sample is shown:\n%s", gadget)
	}
	if strings.Contains(link.Body, "\n\n\n") || !strings.HasSuffix(link.Body, "|\n") {
		t.Errorf("the page has a blank run or does not end in one newline")
	}
}

func TestCRDCompleteRefusesHeading(t *testing.T) {
	p := crdFixture(t, doctext.Strip)
	_, err := Complete(authoredWidgets+"\n## Widget\n\nOld text.\n", "docs/site/reference/widgets.md", p)
	if err == nil || !strings.Contains(err.Error(), `already holds a "## Widget" heading`) {
		t.Errorf("err %v", err)
	}
}

// probeTree is a one-kind tree whose CRD description and sample carry
// the given text.
func probeTree(t *testing.T, desc, sample string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"c/x.yaml":                     "apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\nspec:\n  group: example.dev\n  names:\n    kind: Widget\n    plural: widgets\n  scope: Namespaced\n  versions:\n  - name: v1\n    served: true\n    storage: true\n    schema:\n      openAPIV3Schema:\n        type: object\n        description: " + strconvQuote(desc) + "\n",
		"s/example.dev_v1_widget.yaml": sample,
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

func strconvQuote(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}

func renderProbe(t *testing.T, root string) string {
	t.Helper()
	m, err := crd.Extract(crd.Options{Root: root, Dir: "./c", Samples: "./s", Doc: doctext.Link,
		Page: crd.Page{Path: "reference/widgets.md", Title: "W", Description: "W."}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	pages, err := crdRenderer{}.Render(data, Target{Kind: KindDocs, Root: "/docs/"})
	if err != nil {
		t.Fatal(err)
	}
	return pages[0].Body
}

// A sample's block scalar holding a fence line cannot close the example's
// fence and let the HTML after it reach the page.
func TestCRDSampleFenceBreakout(t *testing.T) {
	sample := "apiVersion: example.dev/v1\nkind: Widget\nnote: |\n  ```\n  <script>alert(1)</script>\n  ~~~~\n"
	body := renderProbe(t, probeTree(t, "Widget is a widget.", sample))
	want := "### Example\n\n`````yaml\napiVersion: example.dev/v1\nkind: Widget\nnote: |\n  ```\n  <script>alert(1)</script>\n  ~~~~\n`````\n"
	if !strings.Contains(body, want) {
		t.Errorf("the sample's fence:\n%s", body)
	}
	if crdFence("a: b\n") != "```" {
		t.Errorf("a plain sample keeps the three-backtick fence")
	}
}

// Only a citation becomes a link, built by the renderer: Markdown links
// written in a description stay text.
func TestCRDForgedLink(t *testing.T) {
	desc := "Widget is a widget. See [0021:D4](/enhancements/0099/decisions/), [x](javascript:alert(1)) and {{{< y >}}} under 0021:D4."
	body := renderProbe(t, probeTree(t, desc, "a: b\n"))
	// A link needs an unescaped "](": every bracket from the description is
	// escaped, so only the renderer's own citation links remain.
	if regexp.MustCompile(`(^|[^\\])\]\((/enhancements/0099/|javascript:)`).MatchString(body) {
		t.Errorf("a description forged a link:\n%s", body)
	}
	if !strings.Contains(body, `\[x\](javascript:alert(1))`) || !strings.Contains(body, "under [0021:D4](/enhancements/0021/decisions/).") {
		t.Errorf("notes:\n%s", body)
	}
	if strings.Contains(body, "{{") {
		t.Errorf("a {{ is left:\n%s", body)
	}
}

func TestCRDCompleteRefusesAnyKindHeading(t *testing.T) {
	p := crdFixture(t, doctext.Strip)
	_, err := Complete(authoredWidgets+"\n## Sprocket\n\nOld text.\n", "docs/site/reference/widgets.md", p)
	if err == nil || !strings.Contains(err.Error(), `already holds a "## Sprocket" heading`) {
		t.Errorf("err %v", err)
	}
}
