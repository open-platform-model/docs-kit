// Package config loads and validates docs-kit.cue, the file that tells
// opm-docs which bundles a repository builds, and bundles.cue, the site's
// pull config.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	slashpath "path"
	"slices"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/parser"

	"github.com/open-platform-model/docs-kit/schema"
)

// Placement is where the site mounts a bundle's content. Owns, for a docs
// bundle, lists the content paths it owns exclusively.
type Placement struct {
	Kind string   `json:"kind"`
	Root string   `json:"root"`
	Owns []string `json:"owns,omitempty"`
}

// Source kinds the config itself reads options of.
const (
	// KindMarkdown copies authored pages; it is no extractor.
	KindMarkdown = "markdown"
	// KindCueDefinitions is checked here for what the schema cannot state.
	KindCueDefinitions = "cue-definitions"
)

// Source is one input of a bundle. Kind names the extractor or the
// markdown source; Value is the source's whole entry, validated, which an
// extractor decodes for its own options. Each kind decodes its own
// options, since two kinds may give one name different types (a markdown
// exclude is a list of globs, a cue-definitions exclude a map).
type Source struct {
	Kind string `json:"kind"`
	// Citations is an extractor source's citation policy, "strip" or
	// "link"; "" for markdown, which copies text as written.
	Citations string `json:"citations,omitempty"`
	// Markdown is set for a markdown source only.
	Markdown *Markdown `json:"-"`
	Value    cue.Value `json:"-"`
}

// Markdown is a markdown source's options.
type Markdown struct {
	Dir     string   `json:"dir"`
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

// Pins names the command that prints a build's pins and the projects it
// must pin.
type Pins struct {
	Command  []string `json:"command"`
	Projects []string `json:"projects"`
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
	Pins      *Pins       `json:"pins,omitempty"`
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
	for _, p := range c.Projects() {
		b := c.Bundles[p]
		list := v.LookupPath(cue.MakePath(cue.Str("bundles"), cue.Str(p), cue.Str("sources")))
		for i := range b.Sources {
			src := &b.Sources[i]
			src.Value = list.LookupPath(cue.MakePath(cue.Index(i)))
			if src.Kind != KindMarkdown {
				continue
			}
			src.Markdown = &Markdown{}
			if err := src.Value.Decode(src.Markdown); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
		}
		if err := checkBundle(p, b); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	return c, nil
}

// checkBundle applies the rules the schema cannot state: owned paths that
// do not nest, a docs bundle without a catalog, the cue-definitions, crd
// and go-api sources in a docs bundle only, and well-formed patterns.
func checkBundle(project string, b Bundle) error {
	owns := b.Placement.Owns
	for i, a := range owns {
		for _, o := range owns[i+1:] {
			if Nests(a, o) || Nests(o, a) {
				return fmt.Errorf("bundles.%q.placement.owns: %s and %s nest; list the outer path alone", project, a, o)
			}
		}
	}
	docs := b.Placement.Kind == "docs"
	for i, src := range b.Sources {
		at := fmt.Sprintf("bundles.%q.sources[%d]", project, i)
		switch {
		case docs && src.Kind == "cue-catalog":
			return fmt.Errorf("%s: a docs bundle carries no cue-catalog source; a catalog is a tab bundle of its own", at)
		case !docs && src.Kind == KindCueDefinitions:
			return fmt.Errorf("%s: a cue-definitions source writes pages under /docs/; give the bundle placement kind \"docs\"", at)
		case !docs && src.Kind == "crd":
			return fmt.Errorf("%s: a crd source belongs in a docs bundle (placement kind \"docs\"); its page is a /docs/ reference page", at)
		case !docs && src.Kind == "go-api":
			return fmt.Errorf("%s: a go-api source writes pages under /docs/; give the bundle placement kind \"docs\"", at)
		}
		if err := checkGlobs(at, src); err != nil {
			return err
		}
	}
	return nil
}

// checkGlobs refuses a pattern that is not a glob: a markdown source's
// include and exclude, a cue-definitions source's skip.
func checkGlobs(at string, src Source) error {
	type list struct {
		name string
		pats []string
	}
	var lists []list
	switch {
	case src.Markdown != nil:
		lists = []list{{"include", src.Markdown.Include}, {"exclude", src.Markdown.Exclude}}
	case src.Kind == KindCueDefinitions:
		var opts struct {
			Skip []string `json:"skip"`
		}
		if err := src.Value.Decode(&opts); err != nil {
			return fmt.Errorf("%s: %w", at, err)
		}
		lists = []list{{"skip", opts.Skip}}
	}
	for _, l := range lists {
		for _, pat := range l.pats {
			if _, err := slashpath.Match(strings.TrimSuffix(pat, "/"), ""); err != nil {
				return fmt.Errorf("%s.%s: %q is not a glob: %w", at, l.name, pat, err)
			}
		}
	}
	return nil
}

// Nests reports whether the owned path inner lies under, or is, outer: a
// directory ("reference/") holds every path under it; a page
// ("reference/operator-resources.md") holds only itself.
func Nests(outer, inner string) bool {
	if strings.HasSuffix(outer, "/") {
		return strings.HasPrefix(inner, outer)
	}
	return outer == inner
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

// DocsProject is a project that may be placed in a site version's /docs/,
// and the only repository allowed to sign it.
type DocsProject struct {
	Repo string `json:"repo"`
}

// Anchor is the bundle that chooses a site version's pinned bundles.
type Anchor struct {
	Project string `json:"project"`
	Tag     string `json:"tag"`
}

// SiteVersion is one site version's docs bundles: the anchor, the projects
// pulled at the anchor's pins and the projects pulled by their own tag.
type SiteVersion struct {
	Anchor Anchor            `json:"anchor"`
	Pinned []string          `json:"pinned"`
	Tags   map[string]string `json:"tags"`
}

// TagProjects returns the projects pulled by their own tag, in name order.
func (v SiteVersion) TagProjects() []string {
	out := make([]string, 0, len(v.Tags))
	for p := range v.Tags {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// Role is the role a project plays in the version: "anchor", "pinned" or
// "tag"; "" when the version does not name it.
func (v SiteVersion) Role(project string) string {
	switch {
	case v.Anchor.Project == project:
		return "anchor"
	case slices.Contains(v.Pinned, project):
		return "pinned"
	}
	if _, ok := v.Tags[project]; ok {
		return "tag"
	}
	return ""
}

// Pull is a validated bundles.cue, with defaults applied.
type Pull struct {
	Path     string
	Digest   string                 // "sha256:<hex>" of the file's bytes
	Registry string                 `json:"registry"`
	Signer   Signer                 `json:"signer"`
	Tabs     map[string]Tab         `json:"tabs"`
	Docs     map[string]DocsProject `json:"docs"`
	Versions map[string]SiteVersion `json:"versions"`
}

// SiteVersions returns the configured site versions in numeric order.
func (p *Pull) SiteVersions() []string {
	out := make([]string, 0, len(p.Versions))
	for v := range p.Versions {
		out = append(out, v)
	}
	slices.SortFunc(out, CompareSiteVersions)
	return out
}

// CompareSiteVersions orders two site versions ("v1.0") by MAJOR, then
// MINOR, numerically.
func CompareSiteVersions(a, b string) int {
	am, an := splitSiteVersion(a)
	bm, bn := splitSiteVersion(b)
	if am != bm {
		return cmpInt(am, bm)
	}
	return cmpInt(an, bn)
}

func splitSiteVersion(v string) (major, minor int) {
	ma, mi, _ := strings.Cut(strings.TrimPrefix(v, "v"), ".")
	major, _ = strconv.Atoi(ma)
	minor, _ = strconv.Atoi(mi)
	return major, minor
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
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
	if err := p.checkVersions(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return p, nil
}

// checkVersions applies the rules the schema cannot state: a project is
// a tab or a docs project, not both; every project a site version names is
// a docs project; and a version names a project once.
func (p *Pull) checkVersions() error {
	docs := make([]string, 0, len(p.Docs))
	for d := range p.Docs {
		docs = append(docs, d)
	}
	sort.Strings(docs)
	for _, d := range docs {
		if _, ok := p.Tabs[d]; ok {
			return fmt.Errorf("%s is both a tab and a docs project; a project has one placement, so remove it from tabs or from docs", d)
		}
	}
	for _, sv := range p.SiteVersions() {
		v := p.Versions[sv]
		names := append(append([]string{v.Anchor.Project}, v.Pinned...), v.TagProjects()...)
		seen := map[string]bool{}
		for _, n := range names {
			if seen[n] {
				return fmt.Errorf("versions.%q names %s twice; a project plays one role in a site version (anchor, pinned or tags)", sv, n)
			}
			seen[n] = true
			if _, ok := p.Docs[n]; !ok {
				return fmt.Errorf("versions.%q names %s, which is not in docs; add docs: %q: {repo: \"<owner>/<repo>\"}", sv, n, n)
			}
		}
	}
	return nil
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
