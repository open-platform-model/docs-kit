// Package enhancements is the enhancements source: it reads the
// enhancements repository's entries, writes the header data of each,
// data/enhancements.json, and turns INDEX.md, GRAPH.md and every entry's
// README and numbered documents into page bodies in the page dialect,
// cleaning them and resolving their relative links at build time.
package enhancements

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// SchemaID identifies the data file this package writes.
const SchemaID = "docs.opmodel.dev/data/enhancements/v1"

// DataFile is the data file's name under data/.
const DataFile = "enhancements.json"

// Model is data/enhancements.json: each entry's header data, by id.
type Model struct {
	Schema  string  `json:"schema"`
	Repo    string  `json:"repo"` // "open-platform-model/enhancements"
	Entries []Entry `json:"entries"`
}

// Entry is one enhancement's header data, from its config.yaml. Its
// authors and history are not copied.
type Entry struct {
	ID           string     `json:"id"` // "0025"
	Slug         string     `json:"slug"`
	Title        string     `json:"title"`
	Summary      string     `json:"summary"` // whitespace collapsed
	Status       string     `json:"status"`
	Category     string     `json:"category"`
	Affects      []string   `json:"affects"`
	Created      string     `json:"created"` // "2026-09-08"
	Updated      string     `json:"updated"`
	Archived     bool       `json:"archived"` // the entry lives under archive/
	DependsOn    []string   `json:"dependsOn"`
	Amends       []string   `json:"amends"`
	Supersedes   []string   `json:"supersedes"`
	Revives      []string   `json:"revives"`
	SupersededBy *string    `json:"supersededBy"`
	Page         string     `json:"page"` // the entry page's path under the section, "0025"
	Documents    []Document `json:"documents"`
}

// Document is one of an entry's seven numbered documents.
type Document struct {
	Slug  string `json:"slug"`  // "problem"
	Title string `json:"title"` // "Problem statement"
	Page  string `json:"page"`  // "0025/problem"
	File  string `json:"file"`  // repository-relative, "0025/01-problem.md"
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

// Decode reads data/enhancements.json.
func Decode(data []byte) (*Model, error) {
	var m Model
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Schema != SchemaID {
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
