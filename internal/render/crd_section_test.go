package render

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/dialect"
	"github.com/open-platform-model/docs-kit/internal/doctext"
	"github.com/open-platform-model/docs-kit/internal/extract/crd"
)

// crdSectionFixture renders the crd fixture tree in the section layout.
func crdSectionFixture(t *testing.T) []Page {
	t.Helper()
	weight := 7
	m, err := crd.Extract(crd.Options{
		Root: crdTree, Dir: "./config/crd/bases", Samples: "./config/samples",
		HideSamplesMatching: []string{"testing.opmodel.dev"},
		StripLabels:         map[string]string{"app.kubernetes.io/name": "opm-operator", "app.kubernetes.io/managed-by": "kustomize"},
		Section:             "reference/widgets/",
		Page:                crd.Page{Title: "Widget reference", Description: "One page per widget kind.", Weight: &weight},
		Order:               []string{"Widget"},
		ReconciledBy:        map[string]string{"Widget": "widget"},
		Doc:                 doctext.Link,
	})
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
	return pages
}

const authoredWidgetsIndex = `---
title: "Widget reference"
description: "What widgets are."
weight: 3
---

This section lists the widget kinds.
`

// TestCRDSectionGolden: the section layout's index, standing alone and
// completed, and its kind pages, each a golden page that passes docs
// bundle-mode lint.
func TestCRDSectionGolden(t *testing.T) {
	pages := crdSectionFixture(t)
	paths := make([]string, 0, len(pages))
	for _, p := range pages {
		paths = append(paths, p.Path)
	}
	if got := strings.Join(paths, ","); got != "reference/widgets/_index.md,reference/widgets/widget.md,reference/widgets/gadget.md,reference/widgets/sprocket.md" {
		t.Fatalf("pages %s", got)
	}
	index := pages[0]
	if !index.Completable || index.Heading != "## Kinds" || len(index.Headings) != 0 {
		t.Fatalf("index completable %v heading %q %q", index.Completable, index.Heading, index.Headings)
	}
	for _, p := range pages[1:] {
		if p.Completable {
			t.Errorf("%s is completable", p.Path)
		}
	}
	completed, err := Complete(authoredWidgetsIndex, "docs/site/reference/widgets/_index.md", index)
	if err != nil {
		t.Fatal(err)
	}
	checkCRDSectionGolden(t, "alone", pages, paths)
	checkCRDSectionGolden(t, "completed", []Page{{Path: index.Path, Body: completed}}, paths) // the kind pages are the same
}

// checkCRDSectionGolden compares pages with the golden set name and lints
// that set as a docs bundle holding paths.
func checkCRDSectionGolden(t *testing.T, name string, set []Page, paths []string) {
	t.Helper()
	dir := filepath.Join("testdata", "golden", "crd-section-"+name)
	for _, p := range set {
		file := filepath.Join(dir, filepath.FromSlash(p.Path))
		if *update {
			_ = os.MkdirAll(filepath.Dir(file), 0o755)
			if err := os.WriteFile(file, []byte(p.Body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("%v (run go test -run TestCRDSectionGolden -update)", err)
		}
		if string(want) != p.Body {
			t.Errorf("%s differs from its golden page", file)
		}
	}
	vs, err := dialect.Lint(dir, dialect.Options{Mode: dialect.Bundle, Bundle: dialect.BundleInfo{
		Kind: KindDocs, Root: "/docs/", Owns: []string{"reference/widgets/"}, Pages: paths,
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		t.Errorf("%s golden page breaks the dialect: %s", name, v)
	}
}

func TestCRDSectionPageFacts(t *testing.T) {
	pages := crdSectionFixture(t)
	byPath := map[string]string{}
	for _, p := range pages {
		byPath[p.Path] = p.Body
	}
	index, widget, gadget := byPath["reference/widgets/_index.md"], byPath["reference/widgets/widget.md"], byPath["reference/widgets/gadget.md"]
	for _, c := range []struct {
		name, body, want string
		absent           bool
	}{
		{"index front matter", index, "---\ntitle: \"Widget reference\"\ndescription: \"One page per widget kind.\"\nweight: 7\n---\n\n", false},
		{"index has no type", index, "type:", true},
		{"index row", index, "| [Widget](/docs/reference/widgets/widget/) | Namespaced | Widget asks the controller to build one widget ([0021:D4](/enhancements/0021/decisions/)). |\n", false},
		{"kind front matter", widget, "---\ntitle: \"Widget\"\ndescription: \"Widget asks the controller to build one widget.\"\ntype: reference\nweight: 1\n---\n\nWidget asks", false},
		{"kind parts one level up", widget, "\n## At a glance\n", false},
		{"kind spec at h2", widget, "\n## Spec\n", false},
		{"no kind heading", widget, "## Widget\n", true},
		{"no h3 parts", widget, "\n### ", true},
		{"served by", widget, "## Served by\n\nThe `widget` reconciler watches every Widget.\n", false},
		{"gadget weight", gadget, "weight: 2\n", false},
	} {
		if got := strings.Contains(c.body, c.want); got == c.absent {
			t.Errorf("%s: page holds %q: %v, want %v", c.name, c.want, got, !c.absent)
		}
	}
	for path, body := range byPath {
		if strings.Contains(body, "\n\n\n") || !strings.HasSuffix(body, "\n") || strings.HasSuffix(body, "\n\n") {
			t.Errorf("%s has a blank run or does not end in one newline", path)
		}
	}
}

func TestCRDSectionCompleteRefusesHeading(t *testing.T) {
	_, err := Complete(authoredWidgetsIndex+"\n## Kinds\n\nOld list.\n", "docs/site/reference/widgets/_index.md", crdSectionFixture(t)[0])
	if err == nil || !strings.Contains(err.Error(), `already holds a "## Kinds" heading`) {
		t.Errorf("err %v", err)
	}
}

// The index's links are checked: without its kind pages in the bundle,
// bundle-mode lint refuses them.
func TestCRDSectionIndexLinksChecked(t *testing.T) {
	dir := filepath.Join("testdata", "golden", "crd-section-completed")
	vs, err := dialect.Lint(dir, dialect.Options{Mode: dialect.Bundle, Bundle: dialect.BundleInfo{
		Kind: KindDocs, Root: "/docs/", Owns: []string{"reference/widgets/"}, Pages: []string{"reference/widgets/_index.md"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 3 {
		t.Errorf("%d violations, want one per kind link: %v", len(vs), vs)
	}
}

// TestOperatorCRDSection renders opm-operator's CRDs as the section its
// docs-kit.cue will declare: the index and one page per kind, weighted in
// the configured order, each page's body its crdref entry with the parts
// one level up.
func TestOperatorCRDSection(t *testing.T) {
	root := filepath.Join("testdata", "operator")
	o := operatorOptions(root)
	single, err := crd.Extract(o)
	if err != nil {
		t.Fatal(err)
	}
	o.Page = crd.Page{Title: "Operator Reference", Description: "One page per operator resource kind."}
	o.Section = "reference/operator/"
	m, err := crd.Extract(o)
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
	want := []string{"_index", "moduleinstance", "modulepackage", "platform", "transformerregistration"}
	if len(pages) != len(want) {
		t.Fatalf("%d pages, want %d", len(pages), len(want))
	}
	entries, err := CRDEntries(single)
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range pages {
		if p.Path != "reference/operator/"+want[i]+".md" {
			t.Errorf("page %d is %s, want %s.md", i, p.Path, want[i])
		}
		if i == 0 {
			continue
		}
		if !strings.Contains(p.Body, fmt.Sprintf("\nweight: %d\n", i)) {
			t.Errorf("%s: weight is not %d", p.Path, i)
		}
		// The kind's single-page entry, its parts lifted one level, is the
		// page's body.
		k := single.Kinds[i-1].Kind
		start := strings.Index(entries, "## "+k+"\n")
		end := len(entries)
		if i < len(single.Kinds) {
			end = strings.Index(entries, "## "+single.Kinds[i].Kind+"\n")
		}
		entry := strings.TrimPrefix(entries[start:end], "## "+k+"\n\n")
		entry = strings.TrimRight(strings.ReplaceAll(entry, "\n### ", "\n## "), "\n") + "\n"
		if body := p.Body[strings.Index(p.Body, "\n---\n\n")+6:]; body != entry {
			t.Errorf("%s: the body is not the kind's entry lifted one level", p.Path)
		}
	}
}
