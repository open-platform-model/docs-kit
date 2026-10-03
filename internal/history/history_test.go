package history

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/extract/cuecatalog"
)

const backupFQN = "opmodel.dev/catalogs/opm/traits/backup@v1alpha1"

func ptr(s string) *string { return &s }

// field is a spec field with a doc comment.
func field(path, typ, presence string) cuecatalog.Field {
	return cuecatalog.Field{Path: path, Type: typ, Presence: presence, Doc: "The " + path + "."}
}

// member is a trait whose page follows the C8 rule for a newest version.
func member(name, apiVersion, cue string, fields ...cuecatalog.Field) cuecatalog.Member {
	return cuecatalog.Member{
		Kind: "trait", Name: name, APIVersion: apiVersion,
		FQN:  "opmodel.dev/catalogs/opm/traits/" + name + "@" + apiVersion,
		Page: "traits/" + name,
		Spec: cuecatalog.Spec{Key: name, CUE: cue, Fields: fields},
	}
}

func seg(name, tool string, members ...cuecatalog.Member) Segment {
	return Segment{Name: name, Tool: tool, Model: &cuecatalog.Model{Schema: cuecatalog.SchemaID, Members: members}}
}

// backup is the backup trait with its two fields, schedule defaulting to
// "daily".
func backup(edit func(m *cuecatalog.Member)) cuecatalog.Member {
	sched := field("schedule", `*"daily" | string`, "required")
	sched.Default = ptr(`"daily"`)
	m := member("backup", "v1alpha1", "backup: {\n\tschedule!: *\"daily\" | string\n\tretention: {daily?: int}\n}",
		sched, field("retention", "struct", "regular"), field("retention.daily", "int", "optional"))
	if edit != nil {
		edit(&m)
	}
	return m
}

func compute(t *testing.T, segs ...Segment) *History {
	t.Helper()
	h, err := Compute("catalog-opm", "0.3.0", segs)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Encode(); err != nil {
		t.Fatal(err)
	}
	return h
}

func TestFieldChanges(t *testing.T) {
	cases := []struct {
		name string
		edit func(m *cuecatalog.Member)
		want []Change
	}{
		{"field removed", func(m *cuecatalog.Member) { m.Spec.Fields = m.Spec.Fields[:2] },
			[]Change{{Op: OpRemoved, Path: "retention.daily", From: ptr("int")}}},
		{"field added", func(m *cuecatalog.Member) {
			m.Spec.Fields = append(m.Spec.Fields, field("retention.weekly", "int", "optional"))
		}, []Change{{Op: OpAdded, Path: "retention.weekly", To: ptr("int")}}},
		{"presence", func(m *cuecatalog.Member) { m.Spec.Fields[2].Presence = "required" },
			[]Change{{Op: OpPresence, Path: "retention.daily", From: ptr("optional"), To: ptr("required")}}},
		{"type", func(m *cuecatalog.Member) { m.Spec.Fields[2].Type = "int & >=0" },
			[]Change{{Op: OpType, Path: "retention.daily", From: ptr("int"), To: ptr("int & >=0")}}},
		{"default", func(m *cuecatalog.Member) { m.Spec.Fields[0].Default = ptr(`"hourly"`) },
			[]Change{{Op: OpDefault, Path: "schedule", From: ptr(`"daily"`), To: ptr(`"hourly"`)}}},
		{"default dropped", func(m *cuecatalog.Member) { m.Spec.Fields[0].Default = nil },
			[]Change{{Op: OpDefault, Path: "schedule", From: ptr(`"daily"`)}}},
		{"ref", func(m *cuecatalog.Member) { m.Spec.Fields[0].Ref = ptr("opmodel.dev/catalogs/opm/schemas.#CronSchema") },
			[]Change{{Op: OpRef, Path: "schedule", To: ptr("opmodel.dev/catalogs/opm/schemas.#CronSchema")}}},
		{"presence and type, in table order", func(m *cuecatalog.Member) {
			m.Spec.Fields[2].Presence, m.Spec.Fields[2].Type = "required", "number"
		}, []Change{
			{Op: OpPresence, Path: "retention.daily", From: ptr("optional"), To: ptr("required")},
			{Op: OpType, Path: "retention.daily", From: ptr("int"), To: ptr("number")},
		}},
		{"current paths first, then removed ones", func(m *cuecatalog.Member) {
			m.Spec.Fields = []cuecatalog.Field{field("tiers", "list", "required"), m.Spec.Fields[0]}
		}, []Change{
			{Op: OpAdded, Path: "tiers", To: ptr("list")},
			{Op: OpRemoved, Path: "retention", From: ptr("struct")},
			{Op: OpRemoved, Path: "retention.daily", From: ptr("int")},
		}},
		{"a validator the walk cannot express", func(m *cuecatalog.Member) {
			m.Spec.CUE = "backup: {\n\tschedule!: *\"daily\" | string\n\tretention: {daily?: int}\n\tmatchN(1, [{retention!: _}])\n}"
		}, []Change{{Op: OpSpec}}},
		{"a doc comment fix", func(m *cuecatalog.Member) {
			m.Spec.Fields[0].Doc = "When the backup runs."
			m.Spec.CUE = "backup: {\n\t// When the backup runs.\n\tschedule!: *\"daily\" | string\n\tretention: {daily?: int}\n}"
		}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := compute(t, seg("4.5", "0.3.0", backup(nil)), seg("4.6", "0.3.1", backup(c.edit)))
			got := h.Members[backupFQN].Changes["4.6"]
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("changes\n got %s\nwant %s", show(got), show(c.want))
			}
			if want := []Pair{{From: "4.5", To: "4.6", Mode: ModeFull}}; !reflect.DeepEqual(h.Compared, want) {
				t.Fatalf("compared %+v", h.Compared)
			}
		})
	}
}

func show(cs []Change) string {
	b, _ := json.Marshal(cs)
	return string(b)
}

// Bundles of two opm-docs minors compare only paths and presence: no type,
// default, ref or spec change.
func TestPathsModeAcrossToolMinors(t *testing.T) {
	changed := backup(func(m *cuecatalog.Member) {
		m.Spec.Fields[2].Type = "int & >=0"
		m.Spec.Fields[0].Default = ptr(`"hourly"`)
		m.Spec.Fields[0].Ref = ptr("x.#Y")
		m.Spec.CUE += "\nmatchN(1, [])"
	})
	h := compute(t, seg("4.5", "0.3.0", backup(nil)), seg("4.6", "0.4.0", changed))
	if got := h.Members[backupFQN].Changes; len(got) != 0 {
		t.Fatalf("changes %v", got)
	}
	if want := []Pair{{From: "4.5", To: "4.6", Mode: ModePaths}}; !reflect.DeepEqual(h.Compared, want) {
		t.Fatalf("compared %+v", h.Compared)
	}
	changed.Spec.Fields[2].Presence = "required"
	h = compute(t, seg("4.5", "0.3.0", backup(nil)), seg("4.6", "0.4.0", changed))
	want := []Change{{Op: OpPresence, Path: "retention.daily", From: ptr("optional"), To: ptr("required")}}
	if got := h.Members[backupFQN].Changes["4.6"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("changes %s", show(got))
	}
}

func TestFirstFloorEdgeAndRemoval(t *testing.T) {
	beta := member("backup", "v1beta1", "backup: {}")
	old := member("old", "v1alpha1", "old: {}")
	old.Page = "traits/old-v1alpha1"
	scaling := member("scaling", "v1alpha1", "scaling: {}")
	h := compute(t,
		seg("edge", "0.3.0", backup(nil), beta, scaling),
		seg("4.6", "0.3.0", backup(nil), beta, old),
		seg("4.5", "0.3.0", backup(nil), old),
	)
	if want := []string{"4.5", "4.6", "edge"}; !reflect.DeepEqual(h.Segments, want) || h.Floor != "4.5" {
		t.Fatalf("segments %v floor %s", h.Segments, h.Floor)
	}
	wantPairs := []Pair{{From: "4.5", To: "4.6", Mode: ModeFull}, {From: "4.6", To: "edge", Mode: ModeFull}}
	if !reflect.DeepEqual(h.Compared, wantPairs) {
		t.Fatalf("compared %+v", h.Compared)
	}
	checks := []struct {
		fqn, first string
		floor      bool
		in         []string
	}{
		{backupFQN, "4.5", true, []string{"4.5", "4.6", "edge"}},
		{beta.FQN, "4.6", false, []string{"4.6", "edge"}},
		{scaling.FQN, "edge", false, []string{"edge"}},
		{old.FQN, "4.5", true, []string{"4.5", "4.6"}},
	}
	for _, c := range checks {
		m := h.Members[c.fqn]
		if m == nil || m.First != c.first || m.FirstIsFloor != c.floor || !reflect.DeepEqual(m.In, c.in) {
			t.Errorf("%s: %+v", c.fqn, m)
		}
	}
	want := map[string][]Removed{"edge": {{
		FQN: old.FQN, Kind: "trait", Name: "old", APIVersion: "v1alpha1", LastIn: "4.6", Page: "traits/old-v1alpha1",
	}}}
	if !reflect.DeepEqual(h.Removed, want) {
		t.Fatalf("removed %+v", h.Removed)
	}
	if got := h.Lineage["trait/backup"]; !reflect.DeepEqual(got, map[string][]string{
		"4.5": {"v1alpha1"}, "4.6": {"v1beta1", "v1alpha1"}, "edge": {"v1beta1", "v1alpha1"},
	}) {
		t.Fatalf("lineage %v", got)
	}
}

func TestLineageNewestFirst(t *testing.T) {
	h := compute(t,
		seg("4.5", "0.3.0", member("backup", "v1alpha1", "")),
		seg("4.6", "0.3.0", member("backup", "v1alpha1", ""), member("backup", "v1", ""), member("backup", "v1beta2", ""), member("backup", "v1beta1", "")),
	)
	if got := h.Lineage["trait/backup"]["4.6"]; !reflect.DeepEqual(got, []string{"v1", "v1beta2", "v1beta1", "v1alpha1"}) {
		t.Fatalf("lineage %v", got)
	}
}

func TestMinorsOrderNumerically(t *testing.T) {
	h := compute(t, seg("4.10", "0.3.0"), seg("4.9", "0.3.0"))
	if h.Floor != "4.9" || !reflect.DeepEqual(h.Segments, []string{"4.9", "4.10"}) {
		t.Fatalf("floor %s segments %v", h.Floor, h.Segments)
	}
}

func TestRefusals(t *testing.T) {
	cases := map[string][]Segment{
		"one segment":   {seg("4.5", "0.3.0")},
		"twice":         {seg("4.5", "0.3.0"), seg("4.5", "0.3.0")},
		"not a segment": {seg("4.5", "0.3.0"), seg("main", "0.3.0")},
		"no model":      {seg("4.5", "0.3.0"), {Name: "4.6", Tool: "0.3.0"}},
		"bad tool":      {seg("4.5", "0.3.0"), seg("4.6", "dev")},
	}
	for name, segs := range cases {
		if _, err := Compute("catalog-opm", "0.3.0", segs); err == nil {
			t.Errorf("%s: computed", name)
		}
	}
}

func TestEncodeIsDeterministic(t *testing.T) {
	segs := func() []Segment {
		return []Segment{
			seg("edge", "0.3.0", backup(func(m *cuecatalog.Member) { m.Spec.Fields[0].Default = ptr(`"hourly"`) }), member("b", "v1", "")),
			seg("4.5", "0.3.0", backup(nil), member("a", "v1", ""), member("c", "v1", "")),
		}
	}
	a, err := compute(t, segs()...).Encode()
	if err != nil {
		t.Fatal(err)
	}
	reversed := segs()
	reversed[0], reversed[1] = reversed[1], reversed[0]
	b, err := compute(t, reversed...).Encode()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("two encodes differ:\n%s\n%s", a, b)
	}
	if !bytes.HasSuffix(a, []byte("}\n")) || !strings.HasPrefix(string(a), "{\n  \"schema\": \"docs.opmodel.dev/history/v1\",\n  \"project\"") {
		t.Fatalf("serialization:\n%s", a)
	}
}

// A model the schema refuses (an unknown kind) fails the encode, naming the
// project.
func TestEncodeValidates(t *testing.T) {
	odd := member("x", "v1", "")
	odd.Kind = "widget"
	h, err := Compute("catalog-opm", "0.3.0", []Segment{seg("4.5", "0.3.0", odd), seg("edge", "0.3.0")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Encode(); err == nil || !strings.Contains(err.Error(), "history for catalog-opm does not validate") {
		t.Fatalf("err = %v", err)
	}
}
