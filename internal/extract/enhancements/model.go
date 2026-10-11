// Package enhancements is the enhancements source: it reads the
// enhancements repository's entries in both layouts (the old numbered
// documents, and OpenSpec changes), writes the header data of each,
// data/enhancements.json, and turns INDEX.md, GRAPH.md, every entry's
// documents and every capability spec into page bodies in the page
// dialect, cleaning them and resolving their relative links at build time.
package enhancements

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// SchemaID identifies the data file this package writes. SchemaIDv1 is the
// schema before the OpenSpec layout: Decode still reads it.
const (
	SchemaID   = "docs.opmodel.dev/data/enhancements/v2"
	SchemaIDv1 = "docs.opmodel.dev/data/enhancements/v1"
)

// DataFile is the data file's name under data/.
const DataFile = "enhancements.json"

// The two layouts an entry can have.
const (
	LayoutLegacy   = "legacy"
	LayoutOpenSpec = "openspec"
)

// Model is data/enhancements.json: each entry's header data, by id.
type Model struct {
	Schema  string  `json:"schema"`
	Repo    string  `json:"repo"` // "open-platform-model/enhancements"
	Entries []Entry `json:"entries"`
}

// Entry is one enhancement's header data. A legacy entry (the old layout)
// keeps its v1 fields; an openspec entry carries the fields marked
// "openspec". The fields of the other layout are left out of the JSON.
type Entry struct {
	ID       string
	Layout   string
	Slug     string
	Title    string
	Summary  string // whitespace collapsed
	Status   string
	Category string
	Affects  []string
	Created  string // "2026-09-08"
	Updated  string // legacy: stored; openspec: the last commit that touched the folder
	Archived bool   // the entry lives under archive/
	Page     string // the entry page's path under the section, "0025"

	Documents []Document

	// Legacy only.
	DependsOn    []string
	Amends       []string
	Supersedes   []string
	Revives      []string
	SupersededBy *string

	// Openspec only.
	Breaking      *bool // nil when unset (a draft)
	After         []string
	SupersededIDs []string
	Accepted      *Accepted
	Progress      Progress
	Capabilities  []string
	Aliases       []Alias
}

// Accepted is the "<date> <commit>" an accepted entry records.
type Accepted struct {
	Date   string `json:"date"`
	Commit string `json:"commit"`
}

// Progress counts the task lines of an entry's tasks.md.
type Progress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

// Alias is an old page path (From) that redirects to the page that replaces
// it (To), both under the section, "0026/problem" to "0026/proposal".
type Alias struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Document is one of an entry's documents.
type Document struct {
	Slug  string `json:"slug"`  // "problem"
	Title string `json:"title"` // "Problem statement"
	Page  string `json:"page"`  // "0025/problem"
	File  string `json:"file"`  // repository-relative, "0025/01-problem.md"
}

// legacyJSON is a legacy entry as written: the v1 fields and the layout.
type legacyJSON struct {
	ID           string     `json:"id"`
	Layout       string     `json:"layout"`
	Slug         string     `json:"slug"`
	Title        string     `json:"title"`
	Summary      string     `json:"summary"`
	Status       string     `json:"status"`
	Category     string     `json:"category"`
	Affects      []string   `json:"affects"`
	Created      string     `json:"created"`
	Updated      string     `json:"updated"`
	Archived     bool       `json:"archived"`
	DependsOn    []string   `json:"dependsOn"`
	Amends       []string   `json:"amends"`
	Supersedes   []string   `json:"supersedes"`
	Revives      []string   `json:"revives"`
	SupersededBy *string    `json:"supersededBy"`
	Page         string     `json:"page"`
	Documents    []Document `json:"documents"`
}

// openspecJSON is an openspec entry as written.
type openspecJSON struct {
	ID           string     `json:"id"`
	Layout       string     `json:"layout"`
	Slug         string     `json:"slug"`
	Title        string     `json:"title"`
	Summary      string     `json:"summary"`
	Status       string     `json:"status"`
	Category     string     `json:"category"`
	Breaking     *bool      `json:"breaking"`
	Affects      []string   `json:"affects"`
	Created      string     `json:"created"`
	Updated      string     `json:"updated"`
	Archived     bool       `json:"archived"`
	After        []string   `json:"after"`
	SupersededBy []string   `json:"supersededBy"`
	Accepted     *Accepted  `json:"accepted"`
	Progress     Progress   `json:"progress"`
	Capabilities []string   `json:"capabilities"`
	Page         string     `json:"page"`
	Documents    []Document `json:"documents"`
	Aliases      []Alias    `json:"aliases"`
}

// MarshalJSON writes the entry in the shape of its layout.
func (e Entry) MarshalJSON() ([]byte, error) {
	if e.Layout == LayoutOpenSpec {
		return json.Marshal(openspecJSON{
			ID: e.ID, Layout: e.Layout, Slug: e.Slug, Title: e.Title, Summary: e.Summary, Status: e.Status,
			Category: e.Category, Breaking: e.Breaking, Affects: e.Affects, Created: e.Created, Updated: e.Updated,
			Archived: e.Archived, After: e.After, SupersededBy: e.SupersededIDs, Accepted: e.Accepted,
			Progress: e.Progress, Capabilities: e.Capabilities, Page: e.Page, Documents: e.Documents, Aliases: e.Aliases,
		})
	}
	return json.Marshal(legacyJSON{
		ID: e.ID, Layout: LayoutLegacy, Slug: e.Slug, Title: e.Title, Summary: e.Summary, Status: e.Status,
		Category: e.Category, Affects: e.Affects, Created: e.Created, Updated: e.Updated, Archived: e.Archived,
		DependsOn: e.DependsOn, Amends: e.Amends, Supersedes: e.Supersedes, Revives: e.Revives,
		SupersededBy: e.SupersededBy, Page: e.Page, Documents: e.Documents,
	})
}

// UnmarshalJSON reads an entry of either layout; a v1 entry has none and
// is legacy.
func (e *Entry) UnmarshalJSON(b []byte) error {
	var probe struct {
		Layout string `json:"layout"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return err
	}
	if probe.Layout == LayoutOpenSpec {
		var o openspecJSON
		if err := json.Unmarshal(b, &o); err != nil {
			return err
		}
		*e = Entry{
			ID: o.ID, Layout: o.Layout, Slug: o.Slug, Title: o.Title, Summary: o.Summary, Status: o.Status,
			Category: o.Category, Breaking: o.Breaking, Affects: o.Affects, Created: o.Created, Updated: o.Updated,
			Archived: o.Archived, After: o.After, SupersededIDs: o.SupersededBy, Accepted: o.Accepted,
			Progress: o.Progress, Capabilities: o.Capabilities, Page: o.Page, Documents: o.Documents, Aliases: o.Aliases,
		}
		return nil
	}
	var l legacyJSON
	if err := json.Unmarshal(b, &l); err != nil {
		return err
	}
	*e = Entry{
		ID: l.ID, Layout: LayoutLegacy, Slug: l.Slug, Title: l.Title, Summary: l.Summary, Status: l.Status,
		Category: l.Category, Affects: l.Affects, Created: l.Created, Updated: l.Updated, Archived: l.Archived,
		DependsOn: l.DependsOn, Amends: l.Amends, Supersedes: l.Supersedes, Revives: l.Revives,
		SupersededBy: l.SupersededBy, Page: l.Page, Documents: l.Documents,
	}
	return nil
}

// Encode writes the model as data/enhancements.json.
func (m *Model) Encode() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode reads data/enhancements.json, schema v2 or v1.
func Decode(data []byte) (*Model, error) {
	var m Model
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Schema != SchemaID && m.Schema != SchemaIDv1 {
		return nil, fmt.Errorf("data schema %q, want %q", m.Schema, SchemaID)
	}
	return &m, nil
}

// docKinds are the seven numbered documents, in order: number n is
// docKinds[n-1]. Slugs, titles and descriptions are the site's.
var docKinds = []struct{ slug, title, description string }{
	{"problem", "Problem statement", "What is wrong today, and for whom."},
	{"design", "Design", "How the proposal works."},
	{"decisions", "Decisions", "Each decision, numbered, with its rationale."},
	{"graduation", "Graduation criteria", "What has to be true before the design moves on."},
	{"risks", "Risks and alternatives", "What could go wrong, and what was weighed instead."},
	{"operational", "Operational concerns", "What it means for running and upgrading OPM."},
	{"questions", "Open questions", "What is still undecided."},
}

// The graph page's front matter.
const (
	graphTitle       = "Relationship graph"
	graphDescription = "How the enhancements depend on and amend one another, by category."
)
