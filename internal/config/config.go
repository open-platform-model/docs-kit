// Package config loads and validates docs-kit.cue, the file that tells
// opm-docs which bundles a repository builds, and bundles.cue, the site's
// pull config.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/parser"

	"github.com/open-platform-model/docs-kit/schema"
)

// Placement is where the site mounts a bundle's content.
type Placement struct {
	Kind string `json:"kind"`
	Root string `json:"root"`
}

// Source is one input of a bundle: a CUE catalog module or a directory of
// authored pages.
type Source struct {
	Kind   string `json:"kind"`
	Module string `json:"module,omitempty"`
	Dir    string `json:"dir,omitempty"`
}

// VersionRule says where a release version comes from.
type VersionRule struct {
	From   string `json:"from"`
	Prefix string `json:"prefix"`
}

// Bundle is one project's build configuration.
type Bundle struct {
	Placement Placement   `json:"placement"`
	Version   VersionRule `json:"version"`
	Sources   []Source    `json:"sources"`
}

// Config is a validated docs-kit.cue.
type Config struct {
	Path    string
	Bundles map[string]Bundle `json:"bundles"`
}

// Projects returns the configured projects in name order.
func (c *Config) Projects() []string {
	out := make([]string, 0, len(c.Bundles))
	for p := range c.Bundles {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// Load reads docs-kit.cue from path and validates it against #Config. A
// package clause is optional and ignored: the file is read as one CUE file.
func Load(path string) (*Config, error) {
	v, _, err := loadFile(path, "#Config")
	if err != nil {
		return nil, err
	}
	c := &Config{Path: path}
	if err := v.Decode(c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(c.Bundles) == 0 {
		return nil, fmt.Errorf("%s: no bundles: add bundles: {\"<project>\": {...}}", path)
	}
	return c, nil
}

// Tab is one tab the site shows, and who may sign its bundles.
type Tab struct {
	Repo string `json:"repo"`
	Root string `json:"root"`
	From string `json:"from"`
	Edge bool   `json:"edge"`
}

// Signer is the identity a bundle's signature must carry.
type Signer struct {
	Issuer   string   `json:"issuer"`
	Workflow string   `json:"workflow"`
	Refs     []string `json:"refs"`
}

// Pull is a validated bundles.cue, with defaults applied.
type Pull struct {
	Path     string
	Digest   string         // "sha256:<hex>" of the file's bytes
	Registry string         `json:"registry"`
	Signer   Signer         `json:"signer"`
	Tabs     map[string]Tab `json:"tabs"`
}

// Projects returns the configured tabs in name order.
func (p *Pull) Projects() []string {
	out := make([]string, 0, len(p.Tabs))
	for t := range p.Tabs {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// LoadPull reads the site's pull config and validates it against #Pull.
func LoadPull(path string) (*Pull, error) {
	v, raw, err := loadFile(path, "#Pull")
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(raw)
	p := &Pull{Path: path, Digest: "sha256:" + hex.EncodeToString(sum[:])}
	if err := v.Decode(p); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return p, nil
}

func loadFile(path, def string) (cue.Value, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("reading %s: %w", path, err)
	}
	f, err := parser.ParseFile(path, raw, parser.ParseComments)
	if err != nil {
		return cue.Value{}, nil, schema.Format(err)
	}
	kept := f.Decls[:0]
	for _, d := range f.Decls {
		if _, ok := d.(*ast.Package); ok {
			continue
		}
		kept = append(kept, d)
	}
	f.Decls = kept
	ctx, _, err := schema.Package()
	if err != nil {
		return cue.Value{}, nil, err
	}
	v := ctx.BuildFile(f)
	if err := v.Err(); err != nil {
		return cue.Value{}, nil, schema.Format(err)
	}
	u, err := schema.Unify(def, v)
	if err != nil {
		return cue.Value{}, nil, err
	}
	return u, raw, nil
}
