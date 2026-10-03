// Package config loads and validates docs-kit.cue, the file that tells
// opm-docs which bundles a repository builds, and bundles.cue, the site's
// pull config.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

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

// Source is one input of a bundle. Kind names the extractor or the
// markdown source; Value is the source's whole entry, validated, which an
// extractor decodes for its own options. Dir is the markdown source's.
type Source struct {
	Kind  string    `json:"kind"`
	Dir   string    `json:"dir,omitempty"`
	Value cue.Value `json:"-"`
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
	raw, _, err := parseFile(path)
	if err != nil {
		return nil, err
	}
	if err := checkKinds(path, raw); err != nil {
		return nil, err
	}
	v, err := schema.Unify("#Config", raw)
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
	for p, b := range c.Bundles {
		list := v.LookupPath(cue.MakePath(cue.Str("bundles"), cue.Str(p), cue.Str("sources")))
		for i := range b.Sources {
			b.Sources[i].Value = list.LookupPath(cue.MakePath(cue.Index(i)))
		}
	}
	return c, nil
}

// checkKinds refuses a source kind the schema does not admit before the
// schema is applied, so the message names the kind and the kinds this
// opm-docs builds instead of every branch of the #Source disjunction.
func checkKinds(path string, v cue.Value) error {
	known, err := schema.SourceKinds()
	if err != nil {
		return err
	}
	bv := v.LookupPath(cue.ParsePath("bundles"))
	if bv.IncompleteKind() != cue.StructKind {
		return nil // the schema reports a malformed bundles
	}
	bundles, err := bv.Fields()
	if err != nil {
		return schema.Format(err)
	}
	for bundles.Next() {
		srcs, err := bundles.Value().LookupPath(cue.ParsePath("sources")).List()
		if err != nil {
			continue
		}
		for i := 0; srcs.Next(); i++ {
			k, err := srcs.Value().LookupPath(cue.ParsePath("kind")).String()
			if err != nil || slices.Contains(known, k) {
				continue
			}
			return fmt.Errorf("%s: bundles.%q.sources[%d]: source kind %q is not one this opm-docs builds; it builds %s", path, bundles.Selector().Unquoted(), i, k, strings.Join(known, ", "))
		}
	}
	return nil
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
	v, raw, err := parseFile(path)
	if err != nil {
		return cue.Value{}, nil, err
	}
	u, err := schema.Unify(def, v)
	if err != nil {
		return cue.Value{}, nil, err
	}
	return u, raw, nil
}

// parseFile reads one CUE file, drops its package clause and builds it in
// the schema's context, without applying a schema.
func parseFile(path string) (cue.Value, []byte, error) {
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
	return v, raw, nil
}
