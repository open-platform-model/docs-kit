// Package crd is the crd extractor: it reads controller-gen
// CustomResourceDefinitions and their kubebuilder samples and writes the
// doc model, data/crd.json, that the renderer reads. It never renders a
// page.
package crd

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// SchemaID identifies the doc model this package writes.
const SchemaID = "docs.opmodel.dev/data/crd/v1"

// DataFile is the doc model's file name under data/.
const DataFile = "crd.json"

// The layouts a model records: one page holding every kind, or a section
// index and one page per kind.
const (
	LayoutPage    = "page"
	LayoutSection = "section"
)

// The citation policies a model records.
const (
	CitationsStrip = "strip"
	CitationsLink  = "link"
)

// EnforcedBy is the only enforcer a CRD proves for its rules.
const EnforcedBy = "api-server"

// Model is data/crd.json.
//
// Prose (summary, notes, field descriptions, rule messages) is plain
// source text with its whitespace folded and the citation policy applied:
// under "strip" no citation is left; under "link" decision citations
// ("0015:D3") stay as written and every other form is removed. Code spans
// stay in backticks. The renderer escapes the text and, under "link",
// links each decision citation.
type Model struct {
	Schema string `json:"schema"`
	// Citations is the source's citation policy: under "link" the renderer
	// links each decision citation left in the prose.
	Citations string `json:"citations"`
	// Layout is LayoutPage or LayoutSection; a data file without it is
	// LayoutPage.
	Layout string `json:"layout"`
	// Page is the completable page: the one page, or the section index.
	Page  Page   `json:"page"`
	Kinds []Kind `json:"kinds"`
	// Read lists every file the extraction read, repository-relative, for
	// the standalone page's lastmod; it is not written to the data file.
	Read []string `json:"-"`
}

// Page is the completable page: the one page the kinds render to, or the
// section index.
type Page struct {
	Path        string `json:"path"` // under content/, "reference/operator-resources.md" or "reference/operator/_index.md"
	Title       string `json:"title"`
	Description string `json:"description"`
	Weight      *int   `json:"weight"` // null when not configured
}

// Kind is one CustomResourceDefinition.
type Kind struct {
	Kind         string    `json:"kind"`
	Group        string    `json:"group"`
	Plural       string    `json:"plural"`
	Scope        string    `json:"scope"` // Namespaced or Cluster
	ShortNames   []string  `json:"shortNames"`
	Categories   []string  `json:"categories"`
	Subresources []string  `json:"subresources"` // status, scale
	Versions     []Version `json:"versions"`
	File         string    `json:"file"` // the CRD's file, repository-relative
	Summary      string    `json:"summary"`
	Notes        []string  `json:"notes"`
	Columns      []Column  `json:"columns"`
	Spec         []Field   `json:"spec"`
	Status       []Field   `json:"status"`
	Rules        []Rule    `json:"rules"`
	Sample       *Sample   `json:"sample"`
	ReconciledBy *string   `json:"reconciledBy"`
	// Page is the kind's page under content/ in the section layout,
	// "reference/operator/moduleinstance.md"; nil in the page layout.
	Page *string `json:"page"`
	// Read lists the files read for this kind, its CRD file first, for its
	// page's lastmod; it is not written to the data file.
	Read []string `json:"-"`
}

// Version is one version the CRD declares.
type Version struct {
	Name    string `json:"name"`
	Served  bool   `json:"served"`
	Storage bool   `json:"storage"`
}

// Column is one additional printer column.
type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	JSONPath string `json:"jsonPath"`
	Priority int    `json:"priority"`
}

// Field is one row of the spec or status table. Path starts at spec or
// status, dot-separated, "[]" for an array item and ".<key>" for a map
// value: "spec.module.path", "status.inventory.entries[].name".
type Field struct {
	Path        string   `json:"path"`
	Type        string   `json:"type"` // "string", "[]Condition", "map[string]string", "free-form object"
	Required    bool     `json:"required"`
	Description string   `json:"description"`
	Default     *string  `json:"default"` // the default as JSON text; null when none
	Enum        []string `json:"enum"`
}

// Rule is one rule the API server enforces from the schema. Its sentence
// is Rule, then Values each as code, then "; refused with: " and Message
// when one is set.
type Rule struct {
	Field   string   `json:"field"` // a field path; "" for the object itself
	Rule    string   `json:"rule"`  // "Required", "One of", "CEL rule"
	Values  []string `json:"values"`
	Message string   `json:"message"`
	By      string   `json:"by"` // always "api-server"
}

// Sample is the kind's example, re-encoded without comments.
type Sample struct {
	File string `json:"file"` // repository-relative
	YAML string `json:"yaml"`
}

// Encode writes the model as data/crd.json.
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

// Decode reads data/crd.json.
func Decode(data []byte) (*Model, error) {
	var m Model
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Schema != SchemaID {
		return nil, fmt.Errorf("doc model schema %q, want %q", m.Schema, SchemaID)
	}
	switch m.Layout {
	case "":
		m.Layout = LayoutPage
	case LayoutPage, LayoutSection:
	default:
		return nil, fmt.Errorf("doc model layout %q, want %q or %q", m.Layout, LayoutPage, LayoutSection)
	}
	return &m, nil
}
