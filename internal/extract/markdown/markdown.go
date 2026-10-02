// Package markdown is the markdown source: it copies one directory of
// authored pages into a bundle's content/, pins links into the bundle's own
// catalog to the build's segment, and records where each page came from.
package markdown

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	Root    string // the source tree
	Dir     string // repository-relative directory, from docs-kit.cue
	Catalog string // the bundle's placement root, "/catalogs/opm/"
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
	var pages []Page
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s: not a regular file; a page must be a regular file", filepath.Join(opts.Dir, rel))
		}
		body, err := os.ReadFile(p) //nolint:gosec // under the configured directory
		if err != nil {
			return err
		}
		src := filepath.ToSlash(filepath.Join(opts.Dir, rel))
		pg := Page{Path: filepath.ToSlash(rel), Body: Pin(string(body), opts.Catalog, opts.Major, opts.Segment), Source: src}
		if opts.Dates != nil {
			pg.Lastmod = opts.Dates(ctx, src)
		}
		pages = append(pages, pg)
		return nil
	})
	sort.Slice(pages, func(i, j int) bool { return pages[i].Path < pages[j].Path })
	return pages, err
}

// A link destination into a catalog through its major alias, inline
// (`](/catalogs/<name>/<MAJOR>/...`) or in a reference definition.
var reAlias = regexp.MustCompile(`(\]\(|\]:[ \t]*<?)(/catalogs/[a-z0-9]+(?:-[a-z0-9]+)*/)(0|[1-9][0-9]*)/`)

// Pin rewrites every link into the bundle's own catalog written through a
// major alias, when that major is the build's (or the build is edge), to
// the build's own segment. Links to another catalog, or to another major,
// are left alone.
func Pin(body, catalog, major, segment string) string {
	return reAlias.ReplaceAllStringFunc(body, func(m string) string {
		sub := reAlias.FindStringSubmatch(m)
		if sub[2] != catalog || (major != "" && sub[3] != major) {
			return m
		}
		return sub[1] + sub[2] + segment + "/"
	})
}

// IsLanding reports the root _index.md, which the renderer completes with
// the generated members block instead of replacing.
func (p Page) IsLanding() bool { return p.Path == "_index.md" }
