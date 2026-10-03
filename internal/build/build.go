// Package build turns a repository's docs-kit.cue into bundle trees: it
// resolves the version and commit, runs each source, renders, writes
// manifest.json and lints the result in bundle mode.
package build

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/dialect"
	"github.com/open-platform-model/docs-kit/internal/extract/cuecatalog"
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
	Release  string   // the release tag; "" builds edge
	Tool     string   // the opm-docs version, without "v"
	// Revision and Patches build a docs revision of Release: Source is the
	// release tree with Patches (the fix commits, oldest first) applied
	// and staged, as revise leaves it.
	Revision int
	Patches  []string
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
		id.dirty, err = repo.PatchedDirty(ctx, o.Patches, o.Out)
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
		Placement: bundle.Placement{Kind: b.Placement.Kind, Root: b.Placement.Root},
	}
	s := &assembly{ctx: ctx, o: o, id: id, m: m, dir: dir, written: map[string]string{}, repo: gitsrc.Repo{Dir: o.Source}}
	if s.patched, err = s.repo.PatchDates(ctx, o.Patches); err != nil {
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
	ctx     context.Context
	o       Options
	id      *ident
	m       *bundle.Manifest
	dir     string
	repo    gitsrc.Repo
	written map[string]string // content path -> the source that wrote it
	patched map[string]string // a docs revision: patched file -> its newest patch's date
	model   *cuecatalog.Model
	landing *markdown.Page
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

func (s *assembly) target() render.Target {
	return render.Target{
		Root:    s.m.Placement.Root,
		Segment: s.id.build.Segment(),
		Edge:    s.id.build.Edge,
		Repo:    s.id.repo,
		Commit:  s.id.commit,
	}
}

func (s *assembly) sources(b config.Bundle, cfgPath string, outside bool) error {
	for _, src := range b.Sources {
		var err error
		switch src.Kind {
		case "cue-catalog":
			err = s.cueCatalog(src)
		case "markdown":
			err = s.markdown(src, cfgPath, outside)
		default:
			err = &UsageError{fmt.Errorf("%s: source kind %q is not built by this opm-docs", cfgPath, src.Kind)}
		}
		if err != nil {
			return err
		}
	}
	return s.renderCatalog()
}

func (s *assembly) cueCatalog(src config.Source) error {
	if s.model != nil {
		return fmt.Errorf("two cue-catalog sources in one bundle; a bundle documents one catalog")
	}
	model, err := cuecatalog.Extract(cuecatalog.Options{Root: s.o.Source, Module: src.Module})
	if err != nil {
		return fmt.Errorf("cue-catalog %s: %w", src.Module, err)
	}
	data, err := model.Encode()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(s.dir, bundle.DataDir, cuecatalog.DataFile), data, 0o644); err != nil { //nolint:gosec // published content
		return err
	}
	s.m.Data = append(s.m.Data, bundle.DataFile{Path: cuecatalog.DataFile, Schema: cuecatalog.SchemaID})
	// The renderer reads the doc model as written, never the extractor's
	// value.
	s.model, err = cuecatalog.Decode(data)
	return err
}

func (s *assembly) markdown(src config.Source, cfgPath string, outside bool) error {
	major := ""
	if !s.id.build.Edge {
		major = s.id.build.Version.MajorTag()
	}
	pages, err := markdown.Copy(s.ctx, markdown.Options{
		Root:     s.o.Source,
		Dir:      src.Dir,
		Catalog:  s.m.Placement.Root,
		Segment:  s.id.build.Segment(),
		Major:    major,
		Optional: outside,
		Dates:    func(_ context.Context, p string) string { return s.lastmod(p) },
	})
	var missing *markdown.MissingDirError
	if errors.As(err, &missing) {
		return fmt.Errorf("%s (named in %s): create it, or remove the markdown source", missing.Error(), cfgPath)
	}
	if err != nil {
		return err
	}
	for i := range pages {
		p := pages[i]
		if p.IsLanding() {
			s.landing = &pages[i]
			continue
		}
		if err := s.write(p.Path, p.Body, "markdown "+src.Dir, bundle.Page{Path: p.Path, Source: p.Source, Lastmod: p.Lastmod}); err != nil {
			return err
		}
	}
	return nil
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

// renderCatalog renders the catalog pages and the landing: the authored
// one completed with the members block, or a generated one.
func (s *assembly) renderCatalog() error {
	t := s.target()
	if s.model == nil {
		if s.landing != nil {
			return s.write(s.landing.Path, s.landing.Body, "markdown", bundle.Page{Path: s.landing.Path, Source: s.landing.Source, Lastmod: s.landing.Lastmod})
		}
		return nil
	}
	pages, err := render.Catalog(s.model, t)
	if err != nil {
		return err
	}
	files := map[string]string{}
	for i := range s.model.Members {
		files[s.model.Members[i].Page+".md"] = s.model.Members[i].File
	}
	for _, p := range pages {
		pg := bundle.Page{Path: p.Path, Generated: true}
		if f, ok := files[p.Path]; ok {
			pg.Source = f
			pg.Lastmod = s.lastmod(f)
		}
		if err := s.write(p.Path, p.Body, "cue-catalog", pg); err != nil {
			return err
		}
	}
	if s.landing == nil {
		body, err := render.Landing(s.model, t, "", "")
		if err != nil {
			return err
		}
		return s.write("_index.md", body, "cue-catalog", bundle.Page{Path: "_index.md", Generated: true})
	}
	body, err := render.Landing(s.model, t, s.landing.Body, s.landing.Source)
	if err != nil {
		return err
	}
	return s.write("_index.md", body, "markdown", bundle.Page{Path: "_index.md", Source: s.landing.Source, Lastmod: s.landing.Lastmod})
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
		Bundle: dialect.BundleInfo{Root: m.Placement.Root, Segment: m.Segment(), Pages: pages},
	})
	if err != nil {
		return nil, err
	}
	for _, v := range vs {
		out = append(out, v.String())
	}
	return out, nil
}
