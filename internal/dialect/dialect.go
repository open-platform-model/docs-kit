// Package dialect lints pages against the OPM page dialect: the Markdown
// subset, front matter and link forms the opmodel.dev site accepts.
//
// Dialect 1 is the rule list of docs/contracts.md C11, and this package is
// its only implementation. The rules and messages were ported from the site's
// retired shell lint, plus the /catalogs/ link forms. Every violation prints
// as "<file>:<line>: <message>".
package dialect

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Version is the page-dialect version this package enforces.
const Version = 1

// Mode selects the rules that depend on where a page is published.
type Mode int

const (
	// Docs lints a docs/site tree: a /catalogs/ link uses the bare tab root
	// or a major segment.
	Docs Mode = iota
	// Bundle lints a bundle's content/ tree: a link into the bundle's own
	// root uses its own segment and names a page of the bundle.
	Bundle
)

// BundleInfo describes the bundle a content tree belongs to.
type BundleInfo struct {
	// Kind is the placement kind: "tab" (also when empty), "docs" or
	// "section". A docs bundle's pages follow the docs-mode /catalogs/
	// rules, and a /docs/ link into a path the bundle owns names one of its
	// pages. A section bundle's pages follow the docs-mode /catalogs/ rules
	// too, and a link into its root names one of its pages.
	Kind    string
	Root    string   // placement root, "/catalogs/opm/", "/docs/" or "/enhancements/"
	Segment string   // a tab's "4.4" or "edge"
	Owns    []string // a docs bundle's owned paths, "reference/cli/", "reference/operator-resources.md"
	Pages   []string // content-relative page paths, "traits/backup.md", "_index.md"
}

// Options configures one lint run.
type Options struct {
	Mode   Mode
	Bundle BundleInfo // used only in Bundle mode
}

// docsBundle reports a bundle-mode lint of a docs-placed bundle.
func (o Options) docsBundle() bool { return o.Mode == Bundle && o.Bundle.Kind == "docs" }

// sectionBundle reports a bundle-mode lint of a section bundle.
func (o Options) sectionBundle() bool { return o.Mode == Bundle && o.Bundle.Kind == "section" }

// docsLinkRules reports whether /catalogs/ links follow the docs-mode
// rules: in docs mode, in a docs bundle and in a section bundle.
func (o Options) docsLinkRules() bool { return o.Mode == Docs || o.docsBundle() || o.sectionBundle() }

// Violation is one rule a page breaks.
type Violation struct {
	File string // the tree directory joined with the page's relative path
	Line int    // 0 for a file-level violation
	Msg  string
}

// String formats the violation as the shell lint prints it.
func (v Violation) String() string {
	return fmt.Sprintf("%s:%d: %s", v.File, v.Line, v.Msg)
}

// Figures are the shortcode figures a page may embed.
var figures = map[string]bool{
	"module-to-cluster": true, "roles-and-artifacts": true, "component-to-objects": true,
	"where-things-live": true, "three-ways-to-deploy": true, "helm-and-opm": true,
	"one-trait-any-provider": true, "what-opm-models": true, "two-models-one-boundary": true,
}

var (
	reName = regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*/)*(_index|index|[a-z0-9]+(-[a-z0-9]+)*)\.md$`)
)

// Lint lints one page tree: a docs/site directory in Docs mode, a bundle's
// content/ directory in Bundle mode. Files are visited in byte order of
// their paths, as the shell lint's `find | sort` does under LC_ALL=C.
func Lint(dir string, opts Options) ([]Violation, error) {
	dir = strings.TrimSuffix(dir, "/")
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return nil, fmt.Errorf("%s: not a directory", dir)
	}
	files, err := listFiles(dir)
	if err != nil {
		return nil, err
	}
	pages := map[string]bool{}
	for _, p := range opts.Bundle.Pages {
		pages[p] = true
	}
	var out []Violation
	for _, f := range files {
		vs, err := lintFile(dir, f, opts, pages)
		if err != nil {
			return nil, err
		}
		out = append(out, vs...)
	}
	return out, nil
}

// listFiles returns every regular file and symlink under dir, as full
// paths in byte order.
func listFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() || d.Type()&fs.ModeSymlink != 0 {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", dir, err)
	}
	sort.Strings(files)
	return files, nil
}

func lintFile(dir, f string, opts Options, pages map[string]bool) ([]Violation, error) {
	rel := strings.TrimPrefix(f, dir+"/")
	var out []Violation
	add := func(m string) { out = append(out, Violation{File: f, Line: 0, Msg: m}) }
	fi, err := os.Lstat(f)
	if err != nil {
		return nil, err
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		add("symlink; a page must be a regular file")
		return out, nil
	}
	switch {
	case strings.HasSuffix(rel, ".mdx"):
		add("MDX file; rename to .md and replace components with {{< opm/... >}} shortcodes")
		return out, nil
	case rel == "index.md" || strings.HasSuffix(rel, "/index.md"):
		add("index.md is a leaf bundle in Hugo; a section page is _index.md")
	case strings.HasSuffix(rel, ".md"):
	default:
		add("not a page; docs/site holds only .md files (no images or data)")
		return out, nil
	}
	if !reName.MatchString(rel) {
		add("file and directory names are lower-case kebab-case (a-z, 0-9, -)")
	}
	base := rel[strings.LastIndex(rel, "/")+1:]
	leaf := base != "_index.md" && base != "index.md"
	body, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", f, err)
	}
	p := &pageLint{file: f, leaf: leaf, opts: opts, pages: pages, line: map[string]int{}, val: map[string]string{}}
	p.run(splitLines(string(body)))
	return append(out, p.out...), nil
}

// splitLines splits text into records as awk does: a final newline ends the
// last record rather than starting an empty one.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}
