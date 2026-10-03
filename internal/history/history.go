// Package history computes a tab's version history: it compares the doc
// model (data/catalog.json) of each segment a pull unpacked with the
// segment before it and records, per member, where it first appears, the
// segments it is in and how its spec fields changed, the members each
// segment removed, and the apiVersions of every name. It reads and writes
// no files; pull feeds it the models and writes history.json.
package history

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"

	"github.com/open-platform-model/docs-kit/internal/cuetok"
	"github.com/open-platform-model/docs-kit/internal/extract/cuecatalog"
	"github.com/open-platform-model/docs-kit/internal/tags"
	"github.com/open-platform-model/docs-kit/schema"
)

// SchemaID identifies the history format.
const SchemaID = "docs.opmodel.dev/history/v1"

// FileName is the history file's name under <out>/<project>/.
const FileName = "history.json"

// Comparison modes of a pair of segments.
const (
	// ModeFull compares everything: both bundles were built by one
	// opm-docs minor, so their type, default and ref strings agree.
	ModeFull = "full"
	// ModePaths compares only field paths and presence.
	ModePaths = "paths"
)

// Change operations, in the order they are recorded for one path.
const (
	OpRemoved  = "removed"
	OpAdded    = "added"
	OpPresence = "presence"
	OpType     = "type"
	OpDefault  = "default"
	OpRef      = "ref"
	OpSpec     = "spec"
)

// Segment is one segment's input: its name ("4.5" or "edge"), the
// opm-docs version that built its bundle (manifest.json's tool) and its
// doc model. Source names the doc model in errors, such as
// "catalog-opm 4.6 data/catalog.json".
type Segment struct {
	Name   string
	Tool   string
	Source string
	Model  *cuecatalog.Model
}

// InputError is a doc model a bundle carries that history cannot use: the
// bundle's fault, not opm-docs'.
type InputError struct {
	Source string
	Err    error
}

func (e *InputError) Error() string { return e.Source + ": " + e.Err.Error() }
func (e *InputError) Unwrap() error { return e.Err }

var (
	kinds     = []string{"resource", "trait", "blueprint"}
	presences = []string{"regular", "optional", "required"}
	// rePage is a member page path (C8): "<kind>s/<name>" or
	// "<kind>s/<name>-<apiVersion>".
	rePage = regexp.MustCompile(`^(resources|traits|blueprints)/[a-z0-9]+(-[a-z0-9]+)*$`)
)

// checkModel refuses a doc model whose members the history file could not
// carry, naming the member and the value.
func checkModel(s Segment) error {
	src := s.Source
	if src == "" {
		src = "segment " + s.Name
	}
	for i := range s.Model.Members {
		m := &s.Model.Members[i]
		var err error
		switch {
		case m.FQN == "":
			err = fmt.Errorf("member %d (%s %s) has no fqn", i, m.Kind, m.Name)
		case !slices.Contains(kinds, m.Kind):
			err = fmt.Errorf("member %s: kind %q is not resource, trait or blueprint", m.FQN, m.Kind)
		case !rePage.MatchString(m.Page):
			err = fmt.Errorf("member %s: page %q is not <kind>s/<name> (C8)", m.FQN, m.Page)
		}
		for _, f := range m.Spec.Fields {
			if err == nil && !slices.Contains(presences, f.Presence) {
				err = fmt.Errorf("member %s: field %s has presence %q, not regular, optional or required", m.FQN, f.Path, f.Presence)
			}
		}
		if err != nil {
			return &InputError{Source: src, Err: err}
		}
	}
	return nil
}

// History is history.json. Field order is the file's key order.
type History struct {
	Schema   string                         `json:"schema"`
	Project  string                         `json:"project"`
	Tool     string                         `json:"tool"`
	Floor    string                         `json:"floor"`
	Segments []string                       `json:"segments"`
	Compared []Pair                         `json:"compared"`
	Members  map[string]*Member             `json:"members"`
	Removed  map[string][]Removed           `json:"removed"`
	Lineage  map[string]map[string][]string `json:"lineage"`
}

// Pair is one comparison: a segment and the one before it.
type Pair struct {
	From string `json:"from"`
	To   string `json:"to"`
	Mode string `json:"mode"`
}

// Member is one member's history, keyed by its FQN.
type Member struct {
	Kind         string              `json:"kind"`
	Name         string              `json:"name"`
	APIVersion   string              `json:"apiVersion"`
	First        string              `json:"first"`
	FirstIsFloor bool                `json:"firstIsFloor"`
	In           []string            `json:"in"`
	Changes      map[string][]Change `json:"changes"`
}

// Change is one difference of a member against its previous segment.
type Change struct {
	Op   string  `json:"op"`
	Path string  `json:"path"`
	From *string `json:"from"`
	To   *string `json:"to"`
}

// Removed is a member the previous segment had and a segment no longer
// has.
type Removed struct {
	FQN        string `json:"fqn"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	APIVersion string `json:"apiVersion"`
	LastIn     string `json:"lastIn"`
	Page       string `json:"page"`
}

// Compute compares a tab's segments in order: minors ascending, edge
// last, each with the one before it. The oldest minor is the floor. tool
// is the opm-docs computing the history. It needs two segments or more.
func Compute(project, tool string, segs []Segment) (*History, error) {
	ordered, err := order(segs)
	if err != nil {
		return nil, err
	}
	h := &History{
		Schema:   SchemaID,
		Project:  project,
		Tool:     tool,
		Floor:    ordered[0].Name,
		Compared: []Pair{},
		Members:  map[string]*Member{},
		Removed:  map[string][]Removed{},
		Lineage:  map[string]map[string][]string{},
	}
	for i, s := range ordered {
		h.Segments = append(h.Segments, s.Name)
		h.addSegment(s)
		if i == 0 {
			continue
		}
		prev := ordered[i-1]
		mode, err := comparisonMode(prev.Tool, s.Tool)
		if err != nil {
			return nil, err
		}
		h.Compared = append(h.Compared, Pair{From: prev.Name, To: s.Name, Mode: mode})
		h.compare(prev, s, mode)
	}
	return h, nil
}

// order sorts the segments and checks there are two or more, each a minor
// or edge, none twice.
func order(segs []Segment) ([]Segment, error) {
	if len(segs) < 2 {
		return nil, fmt.Errorf("a history needs two segments or more, got %d", len(segs))
	}
	ordered := append([]Segment{}, segs...)
	for _, s := range ordered {
		if s.Name != tags.Edge && !tags.IsMinorTag(s.Name) {
			return nil, fmt.Errorf("segment %q is neither MAJOR.MINOR nor edge", s.Name)
		}
		if s.Model == nil {
			return nil, fmt.Errorf("segment %s has no doc model", s.Name)
		}
		if err := checkModel(s); err != nil {
			return nil, err
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool { return tags.CompareMinor(ordered[i].Name, ordered[j].Name) < 0 })
	for i := 1; i < len(ordered); i++ {
		if ordered[i].Name == ordered[i-1].Name {
			return nil, fmt.Errorf("segment %s is given twice", ordered[i].Name)
		}
	}
	return ordered, nil
}

// comparisonMode is full when both tools share MAJOR.MINOR, else paths.
func comparisonMode(a, b string) (string, error) {
	va, err := tags.ParseVersion(a)
	if err != nil {
		return "", fmt.Errorf("bundle tool %q: %w", a, err)
	}
	vb, err := tags.ParseVersion(b)
	if err != nil {
		return "", fmt.Errorf("bundle tool %q: %w", b, err)
	}
	if va.MinorTag() == vb.MinorTag() {
		return ModeFull, nil
	}
	return ModePaths, nil
}

// addSegment records the members of a segment and its lineage.
func (h *History) addSegment(s Segment) {
	for i := range s.Model.Members {
		m := &s.Model.Members[i]
		mh, ok := h.Members[m.FQN]
		if !ok {
			mh = &Member{
				Kind: m.Kind, Name: m.Name, APIVersion: m.APIVersion,
				First: s.Name, FirstIsFloor: s.Name == h.Floor,
				Changes: map[string][]Change{},
			}
			h.Members[m.FQN] = mh
		}
		if n := len(mh.In); n == 0 || mh.In[n-1] != s.Name {
			mh.In = append(mh.In, s.Name)
		}
		key := m.Kind + "/" + m.Name
		if h.Lineage[key] == nil {
			h.Lineage[key] = map[string][]string{}
		}
		versions := h.Lineage[key][s.Name]
		if !slices.Contains(versions, m.APIVersion) {
			versions = append(versions, m.APIVersion)
		}
		sort.SliceStable(versions, func(i, j int) bool { return cuecatalog.NewerAPIVersion(versions[i], versions[j]) })
		h.Lineage[key][s.Name] = versions
	}
}

// compare records the changes of every member cur shares with prev, and
// the members prev has and cur does not.
func (h *History) compare(prev, cur Segment, mode string) {
	before := map[string]*cuecatalog.Member{}
	for i := range prev.Model.Members {
		m := &prev.Model.Members[i]
		if _, ok := before[m.FQN]; !ok {
			before[m.FQN] = m
		}
	}
	now := map[string]bool{}
	for i := range cur.Model.Members {
		m := &cur.Model.Members[i]
		if now[m.FQN] {
			continue
		}
		now[m.FQN] = true
		old, ok := before[m.FQN]
		if !ok {
			continue
		}
		if cs := memberChanges(old, m, mode); len(cs) > 0 {
			h.Members[m.FQN].Changes[cur.Name] = cs
		}
	}
	for i := range prev.Model.Members {
		m := &prev.Model.Members[i]
		if now[m.FQN] || before[m.FQN] != m {
			continue
		}
		h.Removed[cur.Name] = append(h.Removed[cur.Name], Removed{
			FQN: m.FQN, Kind: m.Kind, Name: m.Name, APIVersion: m.APIVersion, LastIn: prev.Name, Page: m.Page,
		})
	}
}

// memberChanges compares one member's spec fields by path: the current
// segment's paths in its order, then the paths only the previous segment
// has, in its order. A field's doc is never compared, so a doc-comment fix
// is no change. When nothing else changed in full mode, a spec block whose
// CUE tokens differ (comments skipped) is one spec change, so a validator
// or a guard the field walk cannot express still shows.
func memberChanges(old, cur *cuecatalog.Member, mode string) []Change {
	before := fieldsByPath(old.Spec.Fields)
	seen := map[string]bool{}
	var cs []Change
	for _, f := range cur.Spec.Fields {
		if seen[f.Path] {
			continue
		}
		seen[f.Path] = true
		o, ok := before[f.Path]
		if !ok {
			cs = append(cs, Change{Op: OpAdded, Path: f.Path, From: nil, To: str(f.Type)})
			continue
		}
		if o.Presence != f.Presence {
			cs = append(cs, Change{Op: OpPresence, Path: f.Path, From: str(o.Presence), To: str(f.Presence)})
		}
		if mode != ModeFull {
			continue
		}
		if o.Type != f.Type {
			cs = append(cs, Change{Op: OpType, Path: f.Path, From: str(o.Type), To: str(f.Type)})
		}
		if !equalOptional(o.Default, f.Default) {
			cs = append(cs, Change{Op: OpDefault, Path: f.Path, From: o.Default, To: f.Default})
		}
		if !equalOptional(o.Ref, f.Ref) {
			cs = append(cs, Change{Op: OpRef, Path: f.Path, From: o.Ref, To: f.Ref})
		}
	}
	for _, o := range old.Spec.Fields {
		if seen[o.Path] {
			continue
		}
		seen[o.Path] = true
		cs = append(cs, Change{Op: OpRemoved, Path: o.Path, From: str(o.Type), To: nil})
	}
	if len(cs) == 0 && mode == ModeFull && !sameSpecText(old.Spec.CUE, cur.Spec.CUE) {
		cs = append(cs, Change{Op: OpSpec, Path: "", From: nil, To: nil})
	}
	return cs
}

func fieldsByPath(fs []cuecatalog.Field) map[string]*cuecatalog.Field {
	m := make(map[string]*cuecatalog.Field, len(fs))
	for i := range fs {
		if _, ok := m[fs[i].Path]; !ok {
			m[fs[i].Path] = &fs[i]
		}
	}
	return m
}

// sameSpecText compares two spec blocks by their CUE tokens with comments
// skipped. Two different blocks of which either does not scan, which the
// extractor never writes, count as changed.
func sameSpecText(a, b string) bool {
	if a == b {
		return true
	}
	ta, errA := cuetok.Scan("spec.cue", []byte(a))
	tb, errB := cuetok.Scan("spec.cue", []byte(b))
	if errA != nil || errB != nil {
		return false
	}
	return cuetok.Equal(ta, tb)
}

func equalOptional(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func str(s string) *string { return &s }

// Encode serializes the history: two-space indent, a trailing newline,
// struct keys in schema order, map keys sorted, no timestamps, so the same
// segments and tool give the same bytes. It validates the result against
// #History. Compute has already refused a bundle's bad data, so a failure
// here is a bug in opm-docs.
func (h *History) Encode() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(h); err != nil {
		return nil, err
	}
	if _, err := schema.ValidateJSON("#History", FileName, buf.Bytes()); err != nil {
		return nil, fmt.Errorf("history for %s does not validate: %w; report it against opm-docs", h.Project, err)
	}
	return buf.Bytes(), nil
}
