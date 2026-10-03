// Package build turns a repository's docs-kit.cue into bundle trees: it
// resolves the version and commit, runs each source, renders, writes
// manifest.json and lints the result in bundle mode.
package build

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/command"
	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/dialect"
	"github.com/open-platform-model/docs-kit/internal/extract/markdown"
	"github.com/open-platform-model/docs-kit/internal/gitsrc"
	"github.com/open-platform-model/docs-kit/internal/render"
	"github.com/open-platform-model/docs-kit/internal/tags"
)

// ConfigFile is the default config name.
const ConfigFile = "docs-kit.cue"

// Options configures one build.
type Options struct {
	Config   string   // explicit --config; "" picks <source>/docs-kit.cue, else ./docs-kit.cue
	Projects []string // empty: every project
	Out      string   // output root; each project goes to <out>/<project>
	Source   string   // the source tree
	// Main is a work tree whose HEAD is the repository's main branch, for
	// a docs bundle's pages[].edit. "" is the current directory when it is
	// a work tree of the same repository (publish.yml's checkout of main
	// beside a release tree), else, for an edge build only, the source
	// tree; a release build without one writes no edit.
	Main string
	// Edits fixes each page's edit (page path -> edit; a page absent has
	// none) instead of reading the main tree: revise sets it when it
	// builds a pushed revision again, so the rebuild gives the pushed bytes.
	Edits   map[string]string
	Release string // the release tag; "" builds edge
	Tool    string // the opm-docs version, without "v"
	// Revision and Patches build a docs revision of Release: Source is the
	// release tree with Patches (the fix commits, oldest first) applied
	// and staged, as revise leaves it. PatchDates, from revise, is each
	// patched file's lastmod; without it a patched file keeps its date at
	// the release commit.
	Revision   int
	Patches    []string
	PatchDates map[string]string
	// Check runs every repository command twice and refuses differing
	// output (opm-docs check). Stderr takes the commands' stderr; nil is
	// os.Stderr.
	Check  bool
	Stderr io.Writer
}

// UsageError is a mistake in the invocation or the config.
type UsageError struct{ Err error }

func (e *UsageError) Error() string { return e.Err.Error() }
func (e *UsageError) Unwrap() error { return e.Err }

// LintError holds the dialect violations of a built bundle.
type LintError struct {
	Project    string
	Violations []string
}

func (e *LintError) Error() string {
	return fmt.Sprintf("%s: the built pages break the page dialect (%d violation(s))", e.Project, len(e.Violations))
}

// Result is one built bundle.
type Result struct {
	Project string
	Dir     string
	Pages   int
}

// Run builds every selected project.
func Run(ctx context.Context, o Options) ([]Result, error) {
	if err := checkRevision(o); err != nil {
		return nil, err
	}
	cfgPath, outside, err := configPath(o)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, &UsageError{err}
	}
	projects, err := selectProjects(cfg, o.Projects)
	if err != nil {
		return nil, err
	}
	var out []Result
	for _, p := range projects {
		r, err := buildProject(ctx, o, cfg, p, outside)
		if err != nil {
			return out, err
		}
		out = append(out, r)
	}
	return out, nil
}

// checkRevision refuses a revision without its release or its patches,
// and patches without a revision.
func checkRevision(o Options) error {
	switch {
	case o.Revision < 0:
		return &UsageError{fmt.Errorf("--revision %d: a revision is 0 or more", o.Revision)}
	case o.Revision == 0 && len(o.Patches) > 0:
		return &UsageError{fmt.Errorf("--patches needs --revision: revision 0 is the release itself")}
	case o.Revision > 0 && o.Release == "":
		return &UsageError{fmt.Errorf("--revision %d needs --release: a docs revision is of a release", o.Revision)}
	case o.Revision > 0 && len(o.Patches) == 0:
		return &UsageError{fmt.Errorf("--revision %d needs --patches, the fix commits applied to the release tree", o.Revision)}
	}
	seen := map[string]bool{}
	for _, p := range o.Patches {
		if !gitsrc.IsSHA(p) {
			return &UsageError{fmt.Errorf("--patches: %q is not a full 40-hex commit hash", p)}
		}
		if seen[p] {
			return &UsageError{fmt.Errorf("--patches: %s is listed twice", p)}
		}
		seen[p] = true
	}
	return nil
}

// configPath picks the config: --config when given, else the source
// tree's docs-kit.cue, else the current directory's. outside reports a
// config that does not come from the source tree.
func configPath(o Options) (path string, outside bool, err error) {
	src, err := filepath.Abs(o.Source)
	if err != nil {
		return "", false, err
	}
	p := o.Config
	if p == "" {
		p = filepath.Join(o.Source, ConfigFile)
		if _, err := os.Stat(p); err != nil {
			p = ConfigFile
		}
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", false, err
	}
	rel, relErr := filepath.Rel(src, abs)
	outside = relErr != nil || rel == ".." || strings.HasPrefix(rel, "../")
	if _, err := os.Stat(p); err != nil {
		return "", false, &UsageError{fmt.Errorf("no %s: write one at the repository root or pass --config", ConfigFile)}
	}
	return p, outside, nil
}

func selectProjects(cfg *config.Config, want []string) ([]string, error) {
	if len(want) == 0 {
		return cfg.Projects(), nil
	}
	for _, p := range want {
		if _, ok := cfg.Bundles[p]; !ok {
			return nil, &UsageError{fmt.Errorf("--project %s: %s configures %s", p, cfg.Path, strings.Join(cfg.Projects(), ", "))}
		}
	}
	sort.Strings(want)
	return want, nil
}

// ident is what a build is of: its version, revision, commit and time.
type ident struct {
	version string // "4.4.5" or "edge"
	build   tags.Build
	commit  string
	created time.Time
	ref     string
	repo    string
	dirty   bool
}

func resolve(ctx context.Context, o Options, b config.Bundle, project string) (*ident, error) {
	repo := gitsrc.Repo{Dir: o.Source}
	if !repo.IsRepo(ctx) {
		return nil, fmt.Errorf("%s is not a git work tree; a bundle records the commit it was built from", o.Source)
	}
	id := &ident{version: tags.Edge, repo: repo.Name(ctx)}
	rev := "HEAD"
	if o.Release != "" {
		if !strings.HasPrefix(o.Release, b.Version.Prefix) {
			return nil, &UsageError{fmt.Errorf("--release %s: project %s takes tags with the prefix %q, such as %s4.4.5", o.Release, project, b.Version.Prefix, b.Version.Prefix)}
		}
		id.version = strings.TrimPrefix(o.Release, b.Version.Prefix)
		if _, err := tags.ParseVersion(id.version); err != nil {
			return nil, &UsageError{fmt.Errorf("--release %s: %w", o.Release, err)}
		}
		rev = o.Release
		id.ref = o.Release
	}
	var err error
	if id.build, err = tags.NewBuild(id.version, o.Revision, ""); err != nil {
		return nil, err
	}
	if id.commit, err = repo.Commit(ctx, rev); err != nil {
		return nil, fmt.Errorf("resolving %s in %s: %w", rev, o.Source, err)
	}
	if o.Release != "" {
		head, err := repo.Commit(ctx, "HEAD")
		if err != nil {
			return nil, err
		}
		if head != id.commit {
			return nil, fmt.Errorf("%s is at %s, not at %s (%s); check out the tag there", o.Source, head[:12], o.Release, id.commit[:12])
		}
	} else {
		id.ref = repo.Branch(ctx)
	}
	if id.created, err = repo.CommitTime(ctx, id.commit); err != nil {
		return nil, err
	}
	if o.Revision > 0 {
		id.dirty, err = repo.Unstaged(ctx, o.Out)
	} else {
		id.dirty, err = repo.Dirty(ctx, o.Out)
	}
	if err != nil {
		return nil, err
	}
	return id, nil
}

func buildProject(ctx context.Context, o Options, cfg *config.Config, project string, outside bool) (Result, error) {
	b := cfg.Bundles[project]
	id, err := resolve(ctx, o, b, project)
	if err != nil {
		return Result{}, err
	}
	dir := filepath.Join(o.Out, project)
	if err := os.RemoveAll(dir); err != nil {
		return Result{}, err
	}
	for _, d := range []string{bundle.ContentDir, bundle.DataDir} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil { //nolint:gosec // published content
			return Result{}, err
		}
	}
	m := &bundle.Manifest{
		Schema:    bundle.SchemaID,
		Project:   project,
		Version:   id.version,
		Revision:  o.Revision,
		Source:    bundle.Source{Repo: id.repo, Commit: id.commit, Ref: id.ref, Dirty: id.dirty, Patches: o.Patches},
		Created:   id.created.Format(time.RFC3339),
		Tool:      o.Tool,
		Dialect:   dialect.Version,
		Placement: bundle.Placement{Kind: b.Placement.Kind, Root: b.Placement.Root, Owns: b.Placement.Owns},
	}
	s := &assembly{ctx: ctx, o: o, id: id, m: m, dir: dir, cfgPath: cfg.Path, written: map[string]string{}, repo: gitsrc.Repo{Dir: o.Source}, patched: o.PatchDates}
	if s.docs = b.Placement.Kind == render.KindDocs; s.docs && o.Edits == nil {
		s.main = mainTree(ctx, o, id.repo)
	}
	s.commands = &command.Runner{Dir: o.Source, Project: project, Version: id.version, Twice: o.Check, Stderr: o.Stderr}
	if err := s.pins(b.Pins); err != nil {
		return Result{}, err
	}
	if err := s.sources(b, cfg.Path, outside); err != nil {
		return Result{}, err
	}
	if err := s.finish(project); err != nil {
		return Result{}, err
	}
	return Result{Project: project, Dir: dir, Pages: len(m.Pages)}, nil
}

// assembly collects one bundle's pages and data.
type assembly struct {
	ctx      context.Context
	o        Options
	id       *ident
	m        *bundle.Manifest
	dir      string
	cfgPath  string
	commands *command.Runner
	repo     gitsrc.Repo
	docs     bool              // a docs bundle: its authored pages get pages[].edit
	main     *gitsrc.Repo      // the main tree, for pages[].edit; nil when there is none
	written  map[string]string // content path -> the source that wrote it
	patched  map[string]string // a docs revision: patched file -> its newest patch's date
}

// lastmod is a source file's last commit date at the commit built; in a
// docs revision, a patched file's is the date of the newest patch that
// touched it.
func (s *assembly) lastmod(path string) string {
	if d, ok := s.patched[path]; ok {
		return d
	}
	return s.repo.LastMod(s.ctx, s.id.commit, path)
}

// newest is the latest lastmod of paths; RFC 3339 UTC dates order as
// strings.
func (s *assembly) newest(paths []string) string {
	latest := ""
	for _, p := range paths {
		if d := s.lastmod(p); d > latest {
			latest = d
		}
	}
	return latest
}

// mainTree is the work tree whose HEAD is main: Options.Main when set,
// else the current directory when it is a work tree of the repository
// built (publish.yml runs every mode in the caller's checkout of main;
// in CI the name is GITHUB_REPOSITORY), else, for an edge build, the
// source tree. A release build has no other: its source tree is the tag,
// which would make edit name files main may no longer have, so it gets
// nil and a note on stderr.
func mainTree(ctx context.Context, o Options, repo string) *gitsrc.Repo {
	if o.Main != "" {
		return &gitsrc.Repo{Dir: o.Main}
	}
	if cwd := (gitsrc.Repo{Dir: "."}); cwd.IsRepo(ctx) && cwd.Name(ctx) == repo {
		return &cwd
	}
	if o.Release == "" {
		return &gitsrc.Repo{Dir: o.Source}
	}
	w := o.Stderr
	if w == nil {
		w = os.Stderr
	}
	fmt.Fprintf(w, "opm-docs build: the current directory is not a checkout of %s, so the pages of %s get no edit path; run build from the repository's checkout of main\n", repo, o.Release)
	return nil
}

// edit is an authored page's pages[].edit: its source file's path when
// the main tree's HEAD has that file, else "". A file renamed on main is
// not followed: no Edit link beats a wrong one. A tab bundle has none; a
// rebuilt revision keeps what it recorded (Options.Edits).
func (s *assembly) edit(path, source string) string {
	switch {
	case !s.docs || source == "":
		return ""
	case s.o.Edits != nil:
		return s.o.Edits[path]
	case s.main == nil || !s.main.HasFile(s.ctx, "HEAD", source):
		return ""
	}
	return source
}

func (s *assembly) target() render.Target {
	return render.Target{
		Kind:    s.m.Placement.Kind,
		Root:    s.m.Placement.Root,
		Segment: s.id.build.Segment(),
		Edge:    s.id.build.Edge,
		Version: s.id.version,
		Repo:    s.id.repo,
		Commit:  s.id.commit,
	}
}

// authored is one markdown source's pages.
type authored struct {
	label string // "markdown docs/catalogs/opm"
	pages []markdown.Page
}

// extracted is one extractor source's data file and, once rendered, its
// pages.
type extracted struct {
	kind  string
	data  Data
	pages []render.Page
}

// sources runs every source in config order, then every renderer in the
// same order, then writes the authored pages and the generated ones; an
// authored page at the path of a completable generated page completes it.
func (s *assembly) sources(b config.Bundle, cfgPath string, outside bool) error {
	docs, gen, err := s.extract(b, cfgPath, outside)
	if err != nil {
		return err
	}
	completable, err := s.render(gen)
	if err != nil {
		return err
	}
	completing := map[string]markdown.Page{}
	for _, a := range docs {
		for _, p := range a.pages {
			if !completable[p.Path] {
				if err := s.write(p.Path, p.Body, a.label, bundle.Page{Path: p.Path, Source: p.Source, Lastmod: p.Lastmod, Edit: s.edit(p.Path, p.Source)}); err != nil {
					return err
				}
				continue
			}
			if prev, ok := completing[p.Path]; ok {
				return fmt.Errorf("content/%s is written by both %s and %s", p.Path, prev.Source, p.Source)
			}
			completing[p.Path] = p
		}
	}
	for _, e := range gen {
		if err := s.writeGenerated(e, completing); err != nil {
			return err
		}
	}
	return nil
}

// extract copies every markdown source and runs every extractor, writing
// its data file.
func (s *assembly) extract(b config.Bundle, cfgPath string, outside bool) ([]authored, []*extracted, error) {
	var docs []authored
	var gen []*extracted
	byKind := map[string]int{}
	dataBy := map[string]string{} // data file -> the kind that wrote it
	for i, src := range b.Sources {
		if src.Kind == markdownKind {
			pages, err := s.markdown(src, cfgPath, outside)
			if err != nil {
				return nil, nil, err
			}
			docs = append(docs, authored{label: "markdown " + src.Dir, pages: pages})
			continue
		}
		ex, ok := extractorFor(src.Kind)
		if !ok {
			return nil, nil, &UsageError{fmt.Errorf("%s: source kind %q is not built by this opm-docs", cfgPath, src.Kind)}
		}
		if j, dup := byKind[src.Kind]; dup {
			return nil, nil, &UsageError{fmt.Errorf("%s: sources[%d] and sources[%d] are both %s; a bundle holds one source of each extractor kind", cfgPath, j, i, src.Kind)}
		}
		byKind[src.Kind] = i
		d, err := ex.Extract(s.ctx, Input{
			Source: s.o.Source, Config: src.Value, Version: s.id.version, Release: s.o.Release, Outside: outside,
			Commands: s.commands, Doc: policy(src.Citations),
		})
		if err != nil {
			return nil, nil, err
		}
		if !reDataPath.MatchString(d.File) {
			return nil, nil, fmt.Errorf("%s wrote the data file %q; a data file is data/<lower-case-kebab>.json", src.Kind, d.File)
		}
		if prev, dup := dataBy[d.File]; dup {
			return nil, nil, fmt.Errorf("data/%s is written by both %s and %s", d.File, prev, src.Kind)
		}
		dataBy[d.File] = src.Kind
		if err := os.WriteFile(filepath.Join(s.dir, bundle.DataDir, d.File), d.Bytes, 0o644); err != nil { //nolint:gosec // published content
			return nil, nil, err
		}
		s.m.Data = append(s.m.Data, bundle.DataFile{Path: d.File, Schema: d.Schema})
		gen = append(gen, &extracted{kind: src.Kind, data: d})
	}
	return docs, gen, nil
}

// render runs the renderer of each data file, in source order, and
// returns the paths of the completable pages.
func (s *assembly) render(gen []*extracted) (map[string]bool, error) {
	t := s.target()
	completable := map[string]bool{}
	rendered := map[string]string{} // page path -> the extractor kind that rendered it
	for _, e := range gen {
		r, err := rendererFor(e.data.Schema)
		if err != nil {
			return nil, err
		}
		// The renderer reads the data file as written, never the
		// extractor's value.
		if e.pages, err = r.Render(e.data.Bytes, t); err != nil {
			return nil, err
		}
		for _, p := range e.pages {
			if !rePagePath.MatchString(p.Path) {
				return nil, fmt.Errorf("%s rendered the page path %q; a page path is lower-case kebab-case segments ending in .md", e.kind, p.Path)
			}
			if prev, ok := rendered[p.Path]; ok {
				if p.Completable && completable[p.Path] {
					return nil, fmt.Errorf("content/%s is a completable page of both %s and %s; an authored page completes one generated page", p.Path, prev, e.kind)
				}
				return nil, fmt.Errorf("content/%s is rendered by both %s and %s", p.Path, prev, e.kind)
			}
			rendered[p.Path] = e.kind
			if p.Completable {
				completable[p.Path] = true
			}
		}
	}
	return completable, nil
}

// writeGenerated writes one extractor's pages: a completable page with an
// authored page at its path becomes the authored page completed.
func (s *assembly) writeGenerated(e *extracted, completing map[string]markdown.Page) error {
	for _, p := range e.pages {
		// A completed page is still a renderer's page: it lies under owns.
		if err := s.owned(p.Path); err != nil {
			return err
		}
		if a, ok := completing[p.Path]; ok && p.Completable {
			body, err := render.Complete(a.Body, a.Source, p)
			if err != nil {
				return err
			}
			if err := s.write(p.Path, body, markdownKind, bundle.Page{Path: p.Path, Source: a.Source, Lastmod: a.Lastmod, Edit: s.edit(p.Path, a.Source)}); err != nil {
				return err
			}
			continue
		}
		pg := bundle.Page{Path: p.Path, Generated: true}
		if f, ok := e.data.Sources[p.Path]; ok {
			pg.Source = f
			pg.Lastmod = s.lastmod(f)
		}
		if in, ok := e.data.Inputs[p.Path]; ok {
			pg.Lastmod = s.newest(in)
		}
		if err := s.write(p.Path, p.Body, e.kind, pg); err != nil {
			return err
		}
	}
	return nil
}

// owned refuses a generated page of a docs bundle outside every path the
// bundle owns.
func (s *assembly) owned(page string) error {
	pl := s.m.Placement
	if pl.Kind != render.KindDocs {
		return nil
	}
	for _, o := range pl.Owns {
		if config.Nests(o, page) {
			return nil
		}
	}
	owns := "nothing"
	if len(pl.Owns) > 0 {
		owns = "only " + strings.Join(pl.Owns, ", ")
	}
	return fmt.Errorf("%s: content/%s is generated, but %s owns %s; add it to placement.owns", s.cfgPath, page, s.m.Project, owns)
}

func (s *assembly) markdown(src config.Source, cfgPath string, outside bool) ([]markdown.Page, error) {
	opts := markdown.Options{
		Root:     s.o.Source,
		Dir:      src.Dir,
		Include:  src.Include,
		Exclude:  src.Exclude,
		Optional: outside,
		Dates:    func(_ context.Context, p string) string { return s.lastmod(p) },
	}
	// Links into the bundle's own catalog are pinned only in a tab bundle;
	// a docs bundle's pages are copied as written.
	if s.m.Placement.Kind != render.KindDocs {
		opts.Catalog = s.m.Placement.Root
		opts.Segment = s.id.build.Segment()
		if !s.id.build.Edge {
			opts.Major = s.id.build.Version.MajorTag()
		}
	}
	pages, err := markdown.Copy(s.ctx, opts)
	var missing *markdown.MissingDirError
	if errors.As(err, &missing) {
		return nil, fmt.Errorf("%s (named in %s): create it, or remove the markdown source", missing.Error(), cfgPath)
	}
	return pages, err
}

// write adds one page, refusing a path two sources write.
func (s *assembly) write(path, body, by string, page bundle.Page) error {
	if prev, ok := s.written[path]; ok {
		return fmt.Errorf("content/%s is written by both %s and %s", path, prev, by)
	}
	s.written[path] = by
	full := filepath.Join(s.dir, bundle.ContentDir, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil { //nolint:gosec // published content
		return err
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil { //nolint:gosec // published content
		return err
	}
	s.m.Pages = append(s.m.Pages, page)
	return nil
}

// finish writes manifest.json and lints the bundle.
func (s *assembly) finish(project string) error {
	sort.Slice(s.m.Pages, func(i, j int) bool { return s.m.Pages[i].Path < s.m.Pages[j].Path })
	if len(s.m.Pages) == 0 {
		return fmt.Errorf("%s: the bundle has no pages; configure a cue-catalog or markdown source that yields one", project)
	}
	if err := bundle.Write(s.dir, s.m); err != nil {
		return fmt.Errorf("%s: %w", project, err)
	}
	if err := bundle.CheckTree(s.dir, s.m); err != nil {
		return err
	}
	vs, err := Lint(s.dir)
	if err != nil {
		return err
	}
	if len(vs) > 0 {
		return &LintError{Project: project, Violations: vs}
	}
	return nil
}

// Lint lints a bundle directory in bundle mode: its listing, then its
// content tree. Each violation reads "<file>:<line>: <message>".
func Lint(dir string) ([]string, error) {
	m, err := bundle.Read(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	if err := bundle.CheckTree(dir, m); err != nil {
		var te *bundle.TreeError
		if !errors.As(err, &te) {
			return nil, err
		}
		for _, p := range te.Problems {
			out = append(out, fmt.Sprintf("%s:0: %s", filepath.Join(dir, bundle.ManifestFile), p))
		}
	}
	pages := make([]string, 0, len(m.Pages))
	for _, p := range m.Pages {
		pages = append(pages, p.Path)
	}
	vs, err := dialect.Lint(filepath.Join(dir, bundle.ContentDir), dialect.Options{
		Mode:   dialect.Bundle,
		Bundle: dialect.BundleInfo{Kind: m.Placement.Kind, Root: m.Placement.Root, Segment: m.Segment(), Owns: m.Placement.Owns, Pages: pages},
	})
	if err != nil {
		return nil, err
	}
	for _, v := range vs {
		out = append(out, v.String())
	}
	return out, nil
}
