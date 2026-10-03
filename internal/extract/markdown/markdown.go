// Package markdown is the markdown source: it copies one directory of
// authored pages into a bundle's content/, pins links into the bundle's own
// catalog to the build's segment, and records where each page came from.
package markdown

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Page is one authored page.
type Page struct {
	Path    string // relative to content/
	Body    string
	Source  string // repository-relative file
	Lastmod string // RFC 3339, or "" without history
}

// Dates gives a file's last commit date; nil or "" when unknown.
type Dates func(ctx context.Context, repoRelPath string) string

// Options configures one copy.
type Options struct {
	Root string // the source tree
	Dir  string // repository-relative directory, from docs-kit.cue
	// Include and Exclude select the files copied: patterns relative to
	// Dir, each a path.Match glob or a directory ending "/". A file is
	// copied when it matches some include (or Include is empty) and no
	// exclude.
	Include []string
	Exclude []string
	// Catalog is a tab bundle's placement root, "/catalogs/opm/", whose
	// links Pin rewrites; "" (a docs bundle) copies pages as written.
	Catalog string
	Segment string // the build's segment, "4.4" or "edge"
	Major   string // the build version's major, "4"; "" for edge
	// Optional: a missing Dir yields no pages instead of an error (the
	// config came from outside the source tree).
	Optional bool
	Dates    Dates
}

// MissingDirError is returned for a missing directory that is not optional.
type MissingDirError struct{ Dir string }

func (e *MissingDirError) Error() string { return "markdown dir " + e.Dir + " does not exist" }

// Copy reads every file under the directory, in path order.
func Copy(ctx context.Context, opts Options) ([]Page, error) {
	dir := filepath.Join(opts.Root, filepath.FromSlash(opts.Dir))
	st, err := os.Stat(dir)
	if os.IsNotExist(err) {
		if opts.Optional {
			return nil, nil
		}
		return nil, &MissingDirError{Dir: opts.Dir}
	}
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("markdown dir %s is not a directory", opts.Dir)
	}
	sel := newSelector(opts.Include, opts.Exclude)
	var pages []Page
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		if !sel.selects(filepath.ToSlash(rel)) {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s: not a regular file; a page must be a regular file", filepath.Join(opts.Dir, rel))
		}
		pg, err := readPage(ctx, opts, p, filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		pages = append(pages, pg)
		return nil
	})
	if err != nil {
		return nil, err
	}
	// A pattern that matches nothing is a mistake, except in a config from
	// outside the source tree, which may predate the files it names.
	if unused := sel.unused(); len(unused) > 0 && !opts.Optional {
		return nil, fmt.Errorf("markdown dir %s: %s matches no file; fix or remove the pattern", opts.Dir, unused[0])
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Path < pages[j].Path })
	return pages, nil
}

// readPage reads the file at full, rel under the directory: pinned in a
// tab bundle, dated when Dates is set.
func readPage(ctx context.Context, opts Options, full, rel string) (Page, error) {
	body, err := os.ReadFile(full)
	if err != nil {
		return Page{}, err
	}
	src := path.Join(opts.Dir, rel)
	text := string(body)
	if opts.Catalog != "" {
		text = Pin(text, opts.Catalog, opts.Major, opts.Segment)
	}
	pg := Page{Path: rel, Body: text, Source: src}
	if opts.Dates != nil {
		pg.Lastmod = opts.Dates(ctx, src)
	}
	return pg, nil
}

// selector applies include and exclude patterns and remembers which of
// them matched a file.
type selector struct {
	include, exclude []string
	used             map[string]bool // "include <pattern>" -> matched a file
}

func newSelector(include, exclude []string) *selector {
	return &selector{include: include, exclude: exclude, used: map[string]bool{}}
}

// selects reports whether the file at rel (slash-separated, relative to
// the directory) is copied.
func (s *selector) selects(rel string) bool {
	in := len(s.include) == 0
	for _, p := range s.include {
		if match(p, rel) {
			s.used["include "+p] = true
			in = true
		}
	}
	out := false
	for _, p := range s.exclude {
		if match(p, rel) {
			s.used["exclude "+p] = true
			out = true
		}
	}
	return in && !out
}

// unused lists the patterns that matched no file, as `include "<p>"`.
func (s *selector) unused() []string {
	var out []string
	for _, l := range []struct {
		name string
		pats []string
	}{{"include", s.include}, {"exclude", s.exclude}} {
		for _, p := range l.pats {
			if !s.used[l.name+" "+p] {
				out = append(out, fmt.Sprintf("%s %q", l.name, p))
			}
		}
	}
	return out
}

// match matches one pattern: a directory ending "/" holds every file under
// it; anything else is a path.Match glob over the whole relative path.
func match(pattern, rel string) bool {
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(rel, pattern)
	}
	ok, err := path.Match(pattern, rel)
	return err == nil && ok
}

// A link destination into a catalog through its major alias, inline
// (`](/catalogs/<name>/<MAJOR>/...`) or in a reference definition.
var reAlias = regexp.MustCompile(`(\]\(|\]:[ \t]*<?)(/catalogs/[a-z0-9]+(?:-[a-z0-9]+)*/)(0|[1-9][0-9]*)/`)

// Pin rewrites every link into the bundle's own catalog written through a
// major alias, when that major is the build's (or the build is edge), to
// the build's own segment. Links to another catalog, or to another major,
// are left alone.
//
// Text inside a fenced code block is an example, never a link, and is
// left as written.
func Pin(body, catalog, major, segment string) string {
	lines := strings.Split(body, "\n")
	fence := ""
	for i, l := range lines {
		if f := fenceOf(l); f != "" {
			switch {
			case fence == "":
				fence = f
			case strings.HasPrefix(f, fence):
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		lines[i] = reAlias.ReplaceAllStringFunc(l, func(m string) string {
			sub := reAlias.FindStringSubmatch(m)
			if sub[2] != catalog || (major != "" && sub[3] != major) {
				return m
			}
			return sub[1] + sub[2] + segment + "/"
		})
	}
	return strings.Join(lines, "\n")
}

// fenceOf returns a line's code fence marker (three or more backticks or
// tildes after at most three spaces), or "".
func fenceOf(l string) string {
	t := strings.TrimLeft(l, " ")
	if len(l)-len(t) > 3 || len(t) < 3 || (t[0] != '`' && t[0] != '~') {
		return ""
	}
	n := 0
	for n < len(t) && t[n] == t[0] {
		n++
	}
	if n < 3 {
		return ""
	}
	return t[:n]
}

// IsLanding reports the root _index.md, which the renderer completes with
// the generated members block instead of replacing.
func (p Page) IsLanding() bool { return p.Path == "_index.md" }
