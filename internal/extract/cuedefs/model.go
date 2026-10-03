// Package cuedefs is the cue-definitions extractor: it parses a CUE
// package, never evaluating it, and writes the doc model of its exported
// definitions, data/cue-definitions.json, grouped into pages by the
// author's inclusion list. It never renders a page.
package cuedefs

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// SchemaID identifies the doc model this package writes.
const SchemaID = "docs.opmodel.dev/data/cue-definitions/v1"

// DataFile is the doc model's file name under data/.
const DataFile = "cue-definitions.json"

// Model is data/cue-definitions.json.
type Model struct {
	Schema      string `json:"schema"`
	ModulePath  string `json:"modulePath"`
	Version     string `json:"version"`
	Section     string `json:"section"` // "reference/definitions/"
	Title       string `json:"title"`
	Description string `json:"description"`
	Weight      int    `json:"weight,omitempty"` // the section index's; 0 when unset
	// Intro is the section index's opening paragraph, Markdown as
	// configured; "" when unset, and the renderer names the module path.
	Intro string `json:"intro,omitempty"`
	Pages []Page `json:"pages"`
	// Definitions in page order, then in each page's configured order.
	Definitions []Definition `json:"definitions"`
	Excluded    []Excluded   `json:"excluded"` // by name
}

// Page is one reference page.
type Page struct {
	File        string   `json:"file"` // "components": <section>components.md
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Weight      int      `json:"weight"` // its 1-based position
	Definitions []string `json:"definitions"`
}

// Definition is one exported top-level definition placed on a page. Prose
// fields (summary, notes, rule) are plain text with Markdown code spans,
// citations handled per the source's policy; the renderer links the
// definition names they mention.
type Definition struct {
	Name   string `json:"name"`   // "#Component"
	Anchor string `json:"anchor"` // "component"
	Page   string `json:"page"`   // the page's file
	File   string `json:"file"`   // repo-relative: "src/component.cue"
	// Summary is the doc comment's first sentence, a leading name label
	// removed; "" when the definition has no doc comment.
	Summary string `json:"summary"`
	// Notes are the rest of the doc comment, one block per entry: a
	// paragraph; "- item", a list item; a line starting with two spaces,
	// a preformatted line; "", a paragraph break.
	Notes []string `json:"notes"`
	// Example holds the lines of the doc comment's "Example:" and "Usage:"
	// values, as CUE.
	Example []string `json:"example"`
	// Shape is what the definition is: "struct, closed", "map",
	// "string constraint", ..., with "`kind: "X"`" appended when it pins
	// a kind.
	Shape  string   `json:"shape"`
	Embeds []string `json:"embeds"` // placed definitions embedded at its top level
	CUE    string   `json:"cue"`    // the formatted spec block
	Uses   []string `json:"uses"`   // placed definitions it names, through excluded ones
	UsedBy []string `json:"usedBy"`
	Rules  []Rule   `json:"rules"`
}

// Rule is one constraint the definition's syntax states.
type Rule struct {
	Rule string `json:"rule"`
	// Code, when set, is shown as a text block under the rule (a regular
	// expression would read as a Markdown link inside a code span).
	Code string `json:"code,omitempty"`
	By   string `json:"by"` // "cue"
}

// Excluded is a definition the pages leave out, with the author's reason.
type Excluded struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Encode writes the model as indented JSON with a trailing newline. Every
// list is written, empty or not.
func (m *Model) Encode() ([]byte, error) {
	m.normalize()
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (m *Model) normalize() {
	if m.Pages == nil {
		m.Pages = []Page{}
	}
	if m.Definitions == nil {
		m.Definitions = []Definition{}
	}
	if m.Excluded == nil {
		m.Excluded = []Excluded{}
	}
	for i := range m.Pages {
		if m.Pages[i].Definitions == nil {
			m.Pages[i].Definitions = []string{}
		}
	}
	for i := range m.Definitions {
		d := &m.Definitions[i]
		for _, l := range []*[]string{&d.Notes, &d.Example, &d.Embeds, &d.Uses, &d.UsedBy} {
			if *l == nil {
				*l = []string{}
			}
		}
		if d.Rules == nil {
			d.Rules = []Rule{}
		}
	}
}

// Decode reads data/cue-definitions.json.
func Decode(data []byte) (*Model, error) {
	var m Model
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", DataFile, err)
	}
	if m.Schema != SchemaID {
		return nil, fmt.Errorf("%s: schema %q, want %q", DataFile, m.Schema, SchemaID)
	}
	return &m, nil
}
