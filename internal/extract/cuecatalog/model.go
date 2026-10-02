// Package cuecatalog is the cue-catalog extractor: it evaluates a CUE
// catalog module with the CUE Go API and writes the doc model,
// data/catalog.json, that the renderer reads. It never renders a page.
package cuecatalog

// SchemaID identifies the doc model this package writes.
const SchemaID = "docs.opmodel.dev/data/cue-catalog/v1"

// DataFile is the doc model's file name under data/.
const DataFile = "catalog.json"

// Model is data/catalog.json.
type Model struct {
	Schema       string        `json:"schema"`
	ModulePath   string        `json:"modulePath"`
	Version      string        `json:"version"`
	Members      []Member      `json:"members"`
	Transformers []Transformer `json:"transformers"`
	// DocNotes lists the repository notes (docs/<name>.md) that member notes
	// mention and that exist at the commit built, so the renderer can link
	// them without reading the source.
	DocNotes []string `json:"docNotes"`
}

// Member is one resource, trait or blueprint.
type Member struct {
	Kind              string        `json:"kind"` // resource, trait or blueprint
	Name              string        `json:"name"`
	APIVersion        string        `json:"apiVersion"`
	Level             string        `json:"level"` // alpha, beta or GA
	FQN               string        `json:"fqn"`
	ModulePath        string        `json:"modulePath"`
	Title             string        `json:"title"`
	Definition        string        `json:"definition"`
	Wrapper           *string       `json:"wrapper"`
	File              string        `json:"file"`
	Page              string        `json:"page"` // "traits/backup", relative to the bundle root
	Description       string        `json:"description"`
	Notes             []string      `json:"notes"`
	Category          *string       `json:"category"`
	Fulfilment        *string       `json:"fulfilment"` // catalog or provider; null for a blueprint
	Optional          *bool         `json:"optional"`   // a trait's default posture; null otherwise
	AppliesTo         []string      `json:"appliesTo"`
	ComposedResources []string      `json:"composedResources"`
	ComposedTraits    []string      `json:"composedTraits"`
	MatchLabels       []Label       `json:"matchLabels"`
	ServedBy          []Service     `json:"servedBy"`
	Mark              *string       `json:"mark"` // not-implemented, provided-by-platform or null
	Enforcement       []Enforcement `json:"enforcement"`
	Spec              Spec          `json:"spec"`
}

// Label is one match label or required label.
type Label struct {
	Key      string `json:"key"`
	Value    string `json:"value"` // CUE text: a quoted string, or a constraint
	Concrete bool   `json:"concrete"`
	Required bool   `json:"required"`
}

// Service is one transformer's demand on a member.
type Service struct {
	Transformer string `json:"transformer"`
	FQN         string `json:"fqn"`
	Demand      string `json:"demand"` // required or optional
}

// Enforcement is one rule and what enforces it. Rule is a Markdown sentence.
type Enforcement struct {
	Rule string `json:"rule"`
	By   string `json:"by"` // cue or kernel
}

// Spec is a member's spec, as text and as structured fields.
type Spec struct {
	Key      string     `json:"key"`
	CUE      string     `json:"cue"`
	Fields   []Field    `json:"fields"`
	Linked   []Linked   `json:"linked"`
	External []External `json:"external"`
}

// Field is one field of the structured spec.
type Field struct {
	Path     string  `json:"path"`
	Type     string  `json:"type"`
	Presence string  `json:"presence"` // regular, optional or required
	Default  *string `json:"default"`
	Doc      string  `json:"doc"`
	Ref      *string `json:"ref"`
}

// Linked is a reference to another member's own spec schema.
type Linked struct {
	Definition string `json:"definition"` // as the spec block writes it
	Page       string `json:"page"`
}

// External is a reference the spec block names without printing it.
type External struct {
	Definition string `json:"definition"` // as the spec block writes it
	Package    string `json:"package"`
	Vendored   bool   `json:"vendored"`
}

// Transformer is one entry of the catalog's #transformers map.
type Transformer struct {
	Name              string   `json:"name"`
	FQN               string   `json:"fqn"`
	Description       string   `json:"description"`
	RequiredLabels    []Label  `json:"requiredLabels"`
	RequiredResources []string `json:"requiredResources"`
	OptionalResources []string `json:"optionalResources"`
	RequiredTraits    []string `json:"requiredTraits"`
	OptionalTraits    []string `json:"optionalTraits"`
}
