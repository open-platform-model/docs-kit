// Package cobra reads the command-tree dump a CLI prints through the
// cobradump module (docs.opmodel.dev/cobradump/v1) and turns it into the
// cobra doc model, data/cobra.json (docs-kit C19). It never parses help
// text: Long and Example stay raw, for the renderer.
package cobra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Schema ids and the data file.
const (
	DumpSchema = "docs.opmodel.dev/cobradump/v1"
	SchemaID   = "docs.opmodel.dev/data/cobra/v1"
	DataFile   = "cobra.json"
)

// Model is the cobra doc model: the dump's tree with the source's section
// settings and, on each top-level command, the page that holds it.
type Model struct {
	Schema      string `json:"schema"`
	Section     string `json:"section"` // "reference/cli/"
	Title       string `json:"title"`
	Description string `json:"description"`
	Weight      int    `json:"weight,omitempty"`
	Citations   string `json:"citations"` // "strip" or "link"
	CLI         CLI    `json:"cli"`
}

// CLI is the root command. Flags are its local, non-persistent flags;
// GlobalFlags its persistent ones.
type CLI struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Short       string    `json:"short"`
	Long        string    `json:"long"`
	Example     string    `json:"example"`
	Aliases     []string  `json:"aliases"`
	UseLine     string    `json:"useLine"`
	Runnable    bool      `json:"runnable"`
	Flags       []Flag    `json:"flags"`
	GlobalFlags []Flag    `json:"globalFlags"`
	Commands    []Command `json:"commands"`
}

// Command is one available command. Page is set on a top-level command
// only: its page path under content/ without ".md".
type Command struct {
	Path           string    `json:"path"`
	Name           string    `json:"name"`
	Use            string    `json:"use"`
	UseLine        string    `json:"useLine"`
	Short          string    `json:"short"`
	Long           string    `json:"long"`
	Example        string    `json:"example"`
	Aliases        []string  `json:"aliases"`
	Runnable       bool      `json:"runnable"`
	Flags          []Flag    `json:"flags"`
	InheritedFlags []Flag    `json:"inheritedFlags"`
	Commands       []Command `json:"commands"`
	Page           string    `json:"page,omitempty"`
}

// Flag is one flag as `--help` shows it.
type Flag struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand"`
	Type      string `json:"type"`
	Default   string `json:"default"`
	Usage     string `json:"usage"`
}

// Options are the source's settings from docs-kit.cue.
type Options struct {
	Section     string
	Title       string
	Description string
	Weight      int
	Citations   string
}

// dump is the cobradump document.
type dump struct {
	Schema string `json:"schema"`
	CLI
}

var (
	reName      = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	reShorthand = regexp.MustCompile(`^[\x21-\x7E]?$`) // "" or one printable ASCII character
)

// FromDump validates a cobradump document and builds the doc model from it.
func FromDump(data []byte, o Options) (*Model, error) {
	var d dump
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return nil, fmt.Errorf("not a %s document: %w", DumpSchema, err)
	}
	if d.Schema != DumpSchema {
		return nil, fmt.Errorf("the dump has schema %q; this opm-docs reads %q", d.Schema, DumpSchema)
	}
	if d.Name == "" || d.Path == "" {
		return nil, fmt.Errorf("the dump names no root command")
	}
	if err := checkFlags(d.Path, append(append([]Flag{}, d.Flags...), d.GlobalFlags...)); err != nil {
		return nil, err
	}
	m := &Model{Schema: SchemaID, Section: o.Section, Title: o.Title, Description: o.Description, Weight: o.Weight, Citations: o.Citations, CLI: d.CLI}
	if err := checkCommands(d.Path, m.CLI.Commands); err != nil {
		return nil, err
	}
	for i := range m.CLI.Commands {
		c := &m.CLI.Commands[i]
		page := strings.ReplaceAll(c.Path, " ", "-")
		if !reName.MatchString(page) {
			return nil, fmt.Errorf("command %q: its page name %q is not lower-case kebab-case", c.Path, page)
		}
		c.Page = o.Section + page
	}
	return m, nil
}

// checkCommands checks each command's path against its parent's and the
// names of its flags, recursively.
func checkCommands(parent string, cmds []Command) error {
	seen := map[string]bool{}
	for i := range cmds {
		c := &cmds[i]
		if c.Path != parent+" "+c.Name {
			return fmt.Errorf("command %q under %q: its path is not %q", c.Path, parent, parent+" "+c.Name)
		}
		if !reName.MatchString(c.Name) {
			return fmt.Errorf("command %q: its name %q is not lower-case kebab-case, which its page and anchor need", c.Path, c.Name)
		}
		if seen[c.Name] {
			return fmt.Errorf("command %q is listed twice", c.Path)
		}
		seen[c.Name] = true
		if c.UseLine == "" {
			return fmt.Errorf("command %q has no use line", c.Path)
		}
		if err := checkFlags(c.Path, c.Flags); err != nil {
			return err
		}
		if err := checkFlags(c.Path, c.InheritedFlags); err != nil {
			return err
		}
		if err := checkCommands(c.Path, c.Commands); err != nil {
			return err
		}
	}
	return nil
}

func checkFlags(cmd string, flags []Flag) error {
	for _, f := range flags {
		if f.Name == "" || strings.HasPrefix(f.Name, "-") || !reShorthand.MatchString(f.Shorthand) {
			return fmt.Errorf("command %q: flag %q (shorthand %q) is not a flag name", cmd, f.Name, f.Shorthand)
		}
	}
	return nil
}

// Encode writes the model as indented JSON with one trailing newline.
func (m *Model) Encode() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Decode reads a data/cobra.json.
func Decode(data []byte) (*Model, error) {
	var m Model
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("data/%s: %w", DataFile, err)
	}
	if m.Schema != SchemaID {
		return nil, fmt.Errorf("data/%s has schema %q; this opm-docs reads %q", DataFile, m.Schema, SchemaID)
	}
	return &m, nil
}
