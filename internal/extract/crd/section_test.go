package crd

import (
	"slices"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/doctext"
)

// TestSectionLayout: a section names the index and one page per kind, and
// records each kind's files for its page's lastmod.
func TestSectionLayout(t *testing.T) {
	o := fixtureOptions(doctext.Link)
	o.Page.Path, o.Section = "", "reference/widgets/"
	m, err := Extract(o)
	if err != nil {
		t.Fatal(err)
	}
	if m.Layout != LayoutSection || m.Page.Path != "reference/widgets/_index.md" {
		t.Fatalf("layout %q page %q", m.Layout, m.Page.Path)
	}
	pages := make([]string, 0, len(m.Kinds))
	for i := range m.Kinds {
		pages = append(pages, *m.Kinds[i].Page)
	}
	if got := strings.Join(pages, ","); got != "reference/widgets/widget.md,reference/widgets/gadget.md,reference/widgets/sprocket.md" {
		t.Errorf("pages %s", got)
	}
	w := kindOf(t, m, "Widget")
	if !slices.Equal(w.Read, []string{w.File, "config/samples/example.dev_v1_widget.yaml"}) {
		t.Errorf("Widget read %v", w.Read)
	}
	data, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"layout": "section"`) || !strings.Contains(string(data), `"page": "reference/widgets/widget.md"`) {
		t.Errorf("data file lacks the layout or a kind page:\n%s", data)
	}
	back, err := Decode(data)
	if err != nil || back.Layout != LayoutSection || *back.Kinds[0].Page != "reference/widgets/widget.md" {
		t.Fatalf("decode: %v", err)
	}
}

func TestSectionRefusesCaseCollision(t *testing.T) {
	root := writeTree(t, map[string]string{"c/a.yaml": crdYAML("Widget", ""), "c/b.yaml": crdYAML("WIDGET", "")})
	_, err := Extract(Options{Root: root, Dir: "./c", Section: "reference/w/", Doc: doctext.Strip})
	if err == nil || !strings.Contains(err.Error(), "the kinds WIDGET and Widget would both be the page reference/w/widget.md") {
		t.Errorf("err %v", err)
	}
}

func TestDecodeLayout(t *testing.T) {
	m, err := Decode([]byte(`{"schema": "` + SchemaID + `", "kinds": []}`))
	if err != nil || m.Layout != LayoutPage {
		t.Fatalf("a data file without layout: %v %+v", err, m)
	}
	if _, err := Decode([]byte(`{"schema": "` + SchemaID + `", "layout": "tabs"}`)); err == nil {
		t.Fatal("an unknown layout decoded")
	}
}
