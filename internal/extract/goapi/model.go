// Package goapi reads a Go module's exported packages, without building or
// type-checking them, into data/go-api.json (docs-kit C20): per package its
// doc, constants, variables, functions and types, each with its
// gofmt-printed declaration and its doc comment as Markdown.
package goapi

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// SchemaID is the data file's schema; DataFile its name under data/.
const (
	SchemaID = "docs.opmodel.dev/data/go-api/v1"
	DataFile = "go-api.json"
)

// Model is data/go-api.json.
type Model struct {
	Schema      string    `json:"schema"`
	ModulePath  string    `json:"modulePath"` // go.mod's module path
	Version     string    `json:"version"`    // "1.0.0-beta.1" or "edge"
	Section     string    `json:"section"`    // "reference/go-api/"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Weight      *int      `json:"weight"`
	Packages    []Package `json:"packages"` // by import path
}

// Package is one documented package: its page and its exported API.
type Package struct {
	ImportPath string `json:"importPath"`
	Name       string `json:"name"`
	// Page is the package directory relative to the source's root, "/" as
	// "-": "helper-objectset".
	Page string `json:"page"`
	// Synopsis is the package doc's first sentence, plain text with every
	// citation removed.
	Synopsis string   `json:"synopsis"`
	Doc      string   `json:"doc"` // Markdown
	Consts   []Value  `json:"consts"`
	Vars     []Value  `json:"vars"`
	Funcs    []Func   `json:"funcs"`
	Types    []Type   `json:"types"`
	Files    []string `json:"files"` // repo-relative, by name
}

// Value is one constant or variable declaration, a group or a single
// name, headed by its first name.
type Value struct {
	Names  []string `json:"names"`
	Anchor string   `json:"anchor"`
	Decl   string   `json:"decl"` // gofmt-printed, without its doc comment
	Doc    string   `json:"doc"`  // Markdown
}

// Func is a function or a method. Recv is the method's receiver type as
// declared ("*Kernel"), null for a function.
type Func struct {
	Name   string  `json:"name"`
	Recv   *string `json:"recv"`
	Anchor string  `json:"anchor"`
	Decl   string  `json:"decl"`
	Doc    string  `json:"doc"`
}

// Type is an exported type with the constants, variables, constructors
// and methods go/doc groups under it.
type Type struct {
	Name    string  `json:"name"`
	Anchor  string  `json:"anchor"`
	Decl    string  `json:"decl"`
	Doc     string  `json:"doc"`
	Consts  []Value `json:"consts"`
	Vars    []Value `json:"vars"`
	Funcs   []Func  `json:"funcs"`
	Methods []Func  `json:"methods"`
}

// Encode writes the model as indented JSON with a trailing newline, every
// list present, empty or not.
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
	if m.Packages == nil {
		m.Packages = []Package{}
	}
	for i := range m.Packages {
		p := &m.Packages[i]
		p.Consts, p.Vars = values(p.Consts), values(p.Vars)
		p.Funcs = funcs(p.Funcs)
		if p.Types == nil {
			p.Types = []Type{}
		}
		if p.Files == nil {
			p.Files = []string{}
		}
		for j := range p.Types {
			t := &p.Types[j]
			t.Consts, t.Vars = values(t.Consts), values(t.Vars)
			t.Funcs, t.Methods = funcs(t.Funcs), funcs(t.Methods)
		}
	}
}

func values(v []Value) []Value {
	if v == nil {
		return []Value{}
	}
	for i := range v {
		if v[i].Names == nil {
			v[i].Names = []string{}
		}
	}
	return v
}

func funcs(f []Func) []Func {
	if f == nil {
		return []Func{}
	}
	return f
}

// Decode reads data/go-api.json.
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
