package cuecatalog

import (
	"bytes"
	"flag"
	"os"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files")

func extractFixture(t *testing.T) *Model {
	t.Helper()
	m, err := Extract(Options{Root: "testdata/catalog", Module: "./demo"})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestGolden(t *testing.T) {
	got, err := extractFixture(t).Encode()
	if err != nil {
		t.Fatal(err)
	}
	const golden = "testdata/catalog.golden.json"
	if *update {
		if err := os.WriteFile(golden, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("data/catalog.json differs from %s; run go test -run TestGolden -update and review the diff", golden)
	}
	again, _ := extractFixture(t).Encode()
	if !bytes.Equal(got, again) {
		t.Fatal("two runs differ")
	}
}

func find(t *testing.T, m *Model, fqn string) *Member {
	t.Helper()
	for i := range m.Members {
		if m.Members[i].FQN == fqn {
			return &m.Members[i]
		}
	}
	t.Fatalf("no member %s", fqn)
	return nil
}

const base = "example.com/catalogs/demo/"

func TestCounts(t *testing.T) {
	m := extractFixture(t)
	if len(m.Members) != 6 || len(m.Transformers) != 2 {
		t.Fatalf("%d members, %d transformers", len(m.Members), len(m.Transformers))
	}
}

func TestTwoAPIVersions(t *testing.T) {
	m := extractFixture(t)
	a := find(t, m, base+"traits/backup@v1alpha1")
	if a.Page != "traits/backup-v1alpha1" || *a.Mark != MarkProvidedByPlatform || len(a.ServedBy) != 0 ||
		*a.Fulfilment != "provider" || *a.Optional || a.Wrapper == nil || *a.Wrapper != "#Backup" {
		t.Errorf("backup v1alpha1: %+v", a)
	}
	b := find(t, m, base+"traits/backup@v1beta1")
	if b.Page != "traits/backup" || b.Mark != nil || len(b.ServedBy) != 1 || b.ServedBy[0].Demand != "optional" {
		t.Errorf("backup v1beta1: %+v", b)
	}
}

func TestMarksAndServedBy(t *testing.T) {
	m := extractFixture(t)
	if q := find(t, m, base+"resources/queue@v1"); *q.Mark != MarkNotImplemented || len(q.ServedBy) != 0 {
		t.Errorf("queue: %+v", q)
	}
	if s := find(t, m, base+"traits/scaling@v1beta1"); *s.Mark != MarkNotImplemented || !*s.Optional {
		t.Errorf("scaling: %+v", s)
	}
	web := find(t, m, base+"blueprints/web@v1")
	if web.Mark != nil || len(web.ServedBy) != 1 || web.ServedBy[0].Transformer != "deployment" || len(web.Enforcement) != 2 {
		t.Errorf("web: %+v", web)
	}
	if len(web.Spec.Linked) != 1 || web.Spec.Linked[0].Page != "resources/container" || web.Spec.Linked[0].Definition != "res.#ContainerSchema" {
		t.Errorf("web linked %+v", web.Spec.Linked)
	}
}

func TestSpecFields(t *testing.T) {
	m := extractFixture(t)
	container := find(t, m, base+"resources/container@v1")
	fields := map[string]Field{}
	for _, f := range container.Spec.Fields {
		fields[f.Path] = f
	}
	if f := fields["name"]; f.Presence != "required" || f.Default != nil || f.Doc != "The container name, unique in the component." {
		t.Errorf("name field %+v", f)
	}
	if f := fields["replicas"]; f.Presence != "regular" || f.Default == nil || *f.Default != "1" || f.Doc != "How many replicas run." {
		t.Errorf("replicas field %+v", f)
	}
	if f := fields["env[]"]; f.Ref == nil || *f.Ref != base+"schemas/kubernetes/core/v1.#EnvVar" {
		t.Errorf("env[] field %+v", f)
	}
	if _, ok := fields["env[].name"]; ok {
		t.Error("the walk expanded a vendored type")
	}
}

func TestSpecReferences(t *testing.T) {
	m := extractFixture(t)
	ext := find(t, m, base+"resources/container@v1").Spec.External
	if len(ext) != 1 || !ext[0].Vendored || ext[0].Definition != "k8s.#EnvVar" {
		t.Errorf("external %+v", ext)
	}
	if f := find(t, m, base+"traits/scaling@v1beta1").Spec.Fields; len(f) != 4 || f[3].Path != "zones.[string]" {
		t.Errorf("scaling fields %+v", f)
	}
}

func TestCleanedText(t *testing.T) {
	m := extractFixture(t)
	a := find(t, m, base+"traits/backup@v1alpha1")
	if strings.Join(a.Notes, "|") != "A platform's backup adapter reads it.|Exactly one provider serves it." {
		t.Errorf("backup notes %q", a.Notes)
	}
	container := find(t, m, base+"resources/container@v1")
	for _, s := range []string{container.Spec.CUE, strings.Join(container.Notes, " ")} {
		if strings.Contains(s, "WHY") || strings.Contains(s, "0010") || strings.Contains(s, "0015") {
			t.Errorf("maintainer text or citation left: %s", s)
		}
	}
	if len(m.DocNotes) != 1 || m.DocNotes[0] != "docs/scaling-notes.md" {
		t.Errorf("doc notes %v", m.DocNotes)
	}
}

func TestRefusals(t *testing.T) {
	for _, c := range []struct{ name, file, from, to, want string }{
		{"doc comment without the description", "demo/resources/v1/queue.cue", "// A message queue a component declares.", "// Something else.",
			"must open with the description"},
		{"blank description", "demo/resources/v1/queue.cue", `description:    "A message queue a component declares"`, `description: ""`,
			"metadata.description is blank"},
		{"trait without a default posture", "demo/traits/v1beta1/scaling.cue", "optional:   bool | *true", "optional: bool",
			"optional states no default posture"},
		{"two spec fields", "demo/resources/v1/queue.cue", "spec: queue: {", "spec: other: 1\n\tspec: queue: {",
			"expected exactly one field"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := copyFixture(t)
			p := root + "/" + c.file
			b, _ := os.ReadFile(p)
			nb := strings.Replace(string(b), c.from, c.to, 1)
			if nb == string(b) {
				t.Fatal("replacement did not apply")
			}
			_ = os.WriteFile(p, []byte(nb), 0o600)
			_, err := Extract(Options{Root: root, Module: "./demo"})
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want one containing %q", err, c.want)
			}
		})
	}
}
