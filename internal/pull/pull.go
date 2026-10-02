// Package pull is the site's side: it resolves each tab's versions from a
// registry's tags, verifies every bundle's signature before fetching its
// layer, unpacks and lints it, and writes the lock.
package pull

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/open-platform-model/docs-kit/internal/build"
	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/oci"
	"github.com/open-platform-model/docs-kit/internal/tags"
	"github.com/open-platform-model/docs-kit/internal/verify"
)

// Local takes one segment of a project from a local bundle tree.
type Local struct {
	Project, Segment, Dir string
}

// ParseLocal parses "<project>@<segment>=<dir>".
func ParseLocal(s string) (Local, error) {
	ps, dir, ok := strings.Cut(s, "=")
	p, seg, ok2 := strings.Cut(ps, "@")
	if !ok || !ok2 || p == "" || dir == "" || (seg != tags.Edge && !tags.IsMinorTag(seg)) {
		return Local{}, fmt.Errorf("--local %s: write <project>@<segment>=<dir>, the segment MAJOR.MINOR or edge", s)
	}
	return Local{Project: p, Segment: seg, Dir: dir}, nil
}

// Options configures a pull.
type Options struct {
	Config  string // bundles.cue
	Out     string // the unpack root
	Lock    string // the lock file to write
	Frozen  string // a lock to pull exactly
	Offline bool
	Locals  []Local
	Tool    string // the opm-docs version, recorded in the lock
	Cache   Cache
	Client  *oci.Client
	// Verifier makes the signature verifier; it is called only when a
	// bundle comes from a registry, so an all-local pull needs no network.
	Verifier func() (*verify.Verifier, error)
	Warn     func(string)
}

// UsageError is a mistake in the invocation or the config.
type UsageError struct{ Err error }

func (e *UsageError) Error() string { return e.Err.Error() }
func (e *UsageError) Unwrap() error { return e.Err }

func usagef(format string, args ...any) error { return &UsageError{fmt.Errorf(format, args...)} }

type puller struct {
	o        Options
	cfg      *config.Pull
	verifier *verify.Verifier
	written  map[string]bool // "<project>/<segment>"
	entries  []Entry
}

// Run pulls every tab and writes the lock.
func Run(ctx context.Context, o Options) (*Lock, error) {
	cfg, err := config.LoadPull(o.Config)
	if err != nil {
		return nil, &UsageError{err}
	}
	if o.Offline && o.Frozen == "" {
		return nil, usagef("--offline needs --frozen <lock>: an offline pull takes exactly the digests a lock names")
	}
	if o.Warn == nil {
		o.Warn = func(string) {}
	}
	p := &puller{o: o, cfg: cfg, written: map[string]bool{}}
	locals, frozen, err := p.prepare()
	if err != nil {
		return nil, err
	}
	for _, project := range cfg.Projects() {
		var err error
		switch {
		case len(locals[project]) > 0:
			err = p.locals(project, locals[project])
		case frozen != nil:
			err = p.frozen(ctx, project, frozen)
		default:
			err = p.resolve(ctx, project)
		}
		if err != nil {
			return nil, err
		}
	}
	if err := p.sweep(); err != nil {
		return nil, err
	}
	lock := &Lock{Schema: LockSchema, Tool: o.Tool, Config: cfg.Digest, Bundles: p.entries}
	data, err := lock.Encode()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(o.Lock), 0o750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(o.Lock, data, 0o644); err != nil { //nolint:gosec // the lock is a build output the site reads
		return nil, err
	}
	return lock, nil
}

// prepare groups the --local trees by project and reads a frozen lock.
func (p *puller) prepare() (map[string][]Local, *Lock, error) {
	locals := map[string][]Local{}
	for _, l := range p.o.Locals {
		if _, ok := p.cfg.Tabs[l.Project]; !ok {
			return nil, nil, usagef("--local %s@%s: %s is not a tab in %s", l.Project, l.Segment, l.Project, p.o.Config)
		}
		locals[l.Project] = append(locals[l.Project], l)
	}
	var frozen *Lock
	if p.o.Frozen != "" {
		var err error
		if frozen, err = p.readFrozen(); err != nil {
			return nil, nil, err
		}
	}
	return locals, frozen, os.MkdirAll(p.o.Out, 0o750)
}

func (p *puller) readFrozen() (*Lock, error) {
	l, err := ReadLock(p.o.Frozen)
	if err != nil {
		return nil, &UsageError{err}
	}
	if l.Config != p.cfg.Digest {
		return nil, usagef("--frozen %s was written for the config %s, and %s is now %s; the trust policy comes from the config, so pull again without --frozen",
			p.o.Frozen, l.Config, p.o.Config, p.cfg.Digest)
	}
	for i := range l.Bundles {
		if e := &l.Bundles[i]; e.Local {
			return nil, usagef("--frozen %s holds the local entry %s@%s; pass it again with --local", p.o.Frozen, e.Project, e.Segment)
		}
	}
	return l, nil
}

// segmentDir is where a segment unpacks, and the lock's dir for it.
func (p *puller) segmentDir(project, segment string) (abs, rel string, err error) {
	abs = filepath.Join(p.o.Out, project, segment)
	lockDir, err := filepath.Abs(filepath.Dir(p.o.Lock))
	if err != nil {
		return "", "", err
	}
	full, err := filepath.Abs(abs)
	if err != nil {
		return "", "", err
	}
	r, err := filepath.Rel(lockDir, full)
	if err != nil {
		return "", "", err
	}
	return abs, filepath.ToSlash(r), nil
}

// locals takes every given segment of a project from local trees, with no
// registry and no signature.
func (p *puller) locals(project string, ls []Local) error {
	tab := p.cfg.Tabs[project]
	for _, l := range ls {
		m, err := bundle.Read(l.Dir)
		if err != nil {
			return err
		}
		if m.Segment() != l.Segment {
			return usagef("--local %s@%s=%s: the tree is version %s, whose segment is %s", l.Project, l.Segment, l.Dir, m.Version, m.Segment())
		}
		if err := checkBundle(m, project, tab); err != nil {
			return fmt.Errorf("--local %s@%s=%s: %w", l.Project, l.Segment, l.Dir, err)
		}
		// A round trip through the layer applies the same guards and limits
		// a pulled bundle gets.
		layer, err := bundle.Pack(l.Dir, parseCreated(m.Created))
		if err != nil {
			return err
		}
		_, rel, err := p.unpack(project, l.Segment, layer, fmt.Sprintf("--local %s@%s", project, l.Segment))
		if err != nil {
			return err
		}
		p.entries = append(p.entries, Entry{
			Project: project, Root: tab.Root, Segment: l.Segment, Local: true,
			Version: m.Version, Revision: m.Revision, Commit: m.Source.Commit, Dialect: m.Dialect, BuiltBy: m.Tool, Dir: rel,
		})
	}
	return nil
}

// checkBundle checks a manifest belongs in the tab.
func checkBundle(m *bundle.Manifest, project string, tab config.Tab) error {
	if m.Project != project {
		return fmt.Errorf("the bundle is project %s, not %s", m.Project, project)
	}
	if m.Placement.Kind != "tab" || m.Placement.Root != tab.Root {
		return fmt.Errorf("the bundle's placement is %s %s, and the tab's root is %s", m.Placement.Kind, m.Placement.Root, tab.Root)
	}
	return nil
}

// unpack replaces <out>/<project>/<segment> with the layer and lints it in
// bundle mode. what names the source in errors.
func (p *puller) unpack(project, segment string, layer []byte, what string) (dir, rel string, err error) {
	if dir, rel, err = p.segmentDir(project, segment); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o750); err != nil {
		return "", "", err
	}
	// Unpack and lint beside the segment, and swap it in only when both
	// pass: a refused bundle leaves the previous segment in place.
	incoming := filepath.Join(filepath.Dir(dir), ".incoming-"+segment)
	if err := os.RemoveAll(incoming); err != nil {
		return "", "", err
	}
	defer os.RemoveAll(incoming)
	if _, err := bundle.Unpack(bytes.NewReader(layer), incoming, bundle.DefaultLimits); err != nil {
		return "", "", fmt.Errorf("%s: %w", what, err)
	}
	vs, err := build.Lint(incoming)
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", what, err)
	}
	if len(vs) > 0 {
		return "", "", &LintError{What: what, Violations: vs}
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", "", err
	}
	if err := os.Rename(incoming, dir); err != nil {
		return "", "", err
	}
	p.written[project+"/"+segment] = true
	return dir, rel, nil
}

// LintError is a pulled bundle that breaks the page dialect.
type LintError struct {
	What       string
	Violations []string
}

func (e *LintError) Error() string {
	return fmt.Sprintf("%s breaks the page dialect (%d violation(s))", e.What, len(e.Violations))
}

// resolve picks a tab's segments from the registry's tags: every minor at
// or above from, and edge when the tab shows it.
func (p *puller) resolve(ctx context.Context, project string) error {
	tab := p.cfg.Tabs[project]
	repo, err := p.o.Client.Repository(p.cfg.Registry + "/" + project)
	if err != nil {
		return err
	}
	all, err := repo.BuildTags(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", project, err)
	}
	segments, err := p.selectSegments(project, tab, all)
	if err != nil {
		return fmt.Errorf("%w in %s", err, repo.Name)
	}
	for _, s := range segments {
		desc, err := repo.Resolve(ctx, s)
		if err != nil {
			return fmt.Errorf("%s: %w", project, err)
		}
		if err := p.fetch(ctx, project, s, repo, desc); err != nil {
			return err
		}
	}
	return nil
}

// selectSegments picks a tab's segments from a repository's tags: every
// minor at or above from, then edge when the tab shows it and it exists.
func (p *puller) selectSegments(project string, tab config.Tab, all []string) ([]string, error) {
	var segments []string
	hasEdge := false
	for _, t := range all {
		switch {
		case t == tags.Edge:
			hasEdge = true
		case tags.IsMinorTag(t) && tags.CompareMinor(t, tab.From) >= 0:
			segments = append(segments, t)
		}
	}
	minors := len(segments)
	if tab.Edge && hasEdge {
		segments = append(segments, tags.Edge)
	} else if tab.Edge {
		p.o.Warn(fmt.Sprintf("%s has no edge tag yet; the tab shows no edge until main publishes", project))
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("%s: no build at or above %s and no edge build; the tab would be empty", project, tab.From)
	}
	if minors == 0 {
		p.o.Warn(fmt.Sprintf("%s has no minor at or above %s; the tab shows only edge", project, tab.From))
	}
	return segments, nil
}

// frozen pulls exactly the digests the lock names for a project.
func (p *puller) frozen(ctx context.Context, project string, l *Lock) error {
	tab := p.cfg.Tabs[project]
	want := p.cfg.Registry + "/" + project
	for i := range l.Bundles {
		e := &l.Bundles[i]
		if e.Project != project {
			continue
		}
		if e.Repository != want {
			return usagef("--frozen %s: %s@%s names the repository %s; the config pulls %s from %s", p.o.Frozen, project, e.Segment, e.Repository, project, want)
		}
		if e.Segment == tags.Edge && !tab.Edge || e.Segment != tags.Edge && tags.CompareMinor(e.Segment, tab.From) < 0 {
			return usagef("--frozen %s: %s@%s is not a segment the config shows (from %s, edge %t)", p.o.Frozen, project, e.Segment, tab.From, tab.Edge)
		}
		repo, err := p.o.Client.Repository(e.Repository)
		if err != nil {
			return err
		}
		d, err := digest.Parse(e.Digest)
		if err != nil {
			return err
		}
		if err := p.fetch(ctx, project, e.Segment, repo, ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: d, Size: -1}); err != nil {
			return err
		}
	}
	return nil
}

// fetch verifies one bundle and only then fetches, unpacks and lints its
// layer.
func (p *puller) fetch(ctx context.Context, project, segment string, repo *oci.Repo, desc ocispec.Descriptor) error {
	tab := p.cfg.Tabs[project]
	what := fmt.Sprintf("%s %s (%s@%s)", project, segment, repo.Name, desc.Digest)
	man, err := p.manifest(ctx, repo, desc)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	b, err := checkManifest(man, project, segment)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	id, err := p.verify(ctx, repo, desc, verify.Policy{
		Issuer: p.cfg.Signer.Issuer, Workflow: p.cfg.Signer.Workflow, Refs: p.cfg.Signer.Refs,
		Repository: "https://github.com/" + tab.Repo, Ref: "refs/heads/main",
	})
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	layer, err := p.layer(ctx, repo, man.Layers[0])
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	dir, rel, err := p.unpack(project, segment, layer, what)
	if err != nil {
		return err
	}
	m, err := bundle.Read(dir)
	if err != nil {
		return err
	}
	if err := checkBundle(m, project, tab); err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	if m.Version != man.Annotations[bundle.AnnVersion] || strconv.Itoa(m.Revision) != man.Annotations[bundle.AnnDocsRev] {
		return fmt.Errorf("%s: manifest.json says %s revision %d, the annotations %s revision %s", what, m.Version, m.Revision, man.Annotations[bundle.AnnVersion], man.Annotations[bundle.AnnDocsRev])
	}
	p.entries = append(p.entries, Entry{
		Project: project, Root: tab.Root, Segment: segment, Tag: segment, Repository: repo.Name, Digest: desc.Digest.String(),
		Version: b.version, Revision: b.revision, Commit: m.Source.Commit, Dialect: m.Dialect, BuiltBy: m.Tool,
		Signer: &Signer{Workflow: id.Workflow, Repository: id.Repository, Ref: id.Ref}, Dir: rel,
	})
	return nil
}

type annotated struct {
	version  string
	revision int
}

// checkManifest checks the build a tag resolved to: a docs bundle of this
// project, in the tag's minor (or an edge build for edge), with one layer.
func checkManifest(man *ocispec.Manifest, project, segment string) (annotated, error) {
	var a annotated
	if man.ArtifactType != bundle.ArtifactType || len(man.Layers) != 1 || man.Layers[0].MediaType != bundle.LayerType {
		return a, fmt.Errorf("not a docs bundle (artifactType %q, %d layer(s))", man.ArtifactType, len(man.Layers))
	}
	if got := man.Annotations[bundle.AnnProject]; got != project {
		return a, fmt.Errorf("the bundle is project %q, not %s", got, project)
	}
	rev, err := strconv.Atoi(man.Annotations[bundle.AnnDocsRev])
	if err != nil {
		return a, fmt.Errorf("annotation %s: %w", bundle.AnnDocsRev, err)
	}
	b, err := tags.NewBuild(man.Annotations[bundle.AnnVersion], rev, "")
	if err != nil {
		return a, err
	}
	if b.Segment() != segment {
		return a, fmt.Errorf("tag %s names a build of %s, which is not in %s", segment, b, segment)
	}
	return annotated{version: man.Annotations[bundle.AnnVersion], revision: rev}, nil
}

// manifest fetches a manifest by digest, through the cache.
func (p *puller) manifest(ctx context.Context, repo *oci.Repo, desc ocispec.Descriptor) (*ocispec.Manifest, error) {
	raw, ok, err := p.o.Cache.Blob(desc.Digest)
	if err != nil {
		return nil, err
	}
	if !ok {
		if p.o.Offline {
			return nil, fmt.Errorf("manifest %s is not in the cache %s, and --offline fetches nothing", desc.Digest, p.o.Cache.Dir)
		}
		if desc.Size < 0 {
			if desc, err = repo.Resolve(ctx, desc.Digest.String()); err != nil {
				return nil, err
			}
		}
		if _, raw, err = repo.Manifest(ctx, desc); err != nil {
			return nil, err
		}
		if err := p.o.Cache.PutBlob(desc.Digest, raw); err != nil {
			return nil, err
		}
	}
	var m ocispec.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// verify checks the signature of desc: from the cache offline, else from
// the registry, caching the bundle that verified.
func (p *puller) verify(ctx context.Context, repo *oci.Repo, desc ocispec.Descriptor, pol verify.Policy) (*verify.Identity, error) {
	if p.verifier == nil {
		v, err := p.o.Verifier()
		if err != nil {
			return nil, err
		}
		p.verifier = v
	}
	if cached, ok, err := p.o.Cache.Signature(desc.Digest); err == nil && ok {
		if id, err := p.verifier.Raw(cached, desc.Digest, pol); err == nil {
			return id, nil
		} else if p.o.Offline {
			return nil, err
		}
	}
	if p.o.Offline {
		return nil, fmt.Errorf("the signature of %s is not in the cache %s, and --offline fetches nothing", desc.Digest, p.o.Cache.Dir)
	}
	if desc.Size < 0 {
		var err error
		if desc, err = repo.Resolve(ctx, desc.Digest.String()); err != nil {
			return nil, err
		}
	}
	raws, err := verify.Find(ctx, repo.Graph(), desc)
	if err != nil {
		return nil, err
	}
	id, raw, err := p.verifier.Any(raws, desc.Digest, pol)
	if err != nil {
		return nil, err
	}
	if err := p.o.Cache.PutSignature(desc.Digest, raw); err != nil {
		return nil, err
	}
	return id, nil
}

// layer fetches the layer by digest, through the cache, refusing one whose
// descriptor is larger than the bundle format allows before a byte moves.
func (p *puller) layer(ctx context.Context, repo *oci.Repo, d ocispec.Descriptor) ([]byte, error) {
	if d.Size > bundle.MaxLayerSize {
		return nil, fmt.Errorf("layer %s is %d bytes, more than the %d a bundle may hold", d.Digest, d.Size, bundle.MaxLayerSize)
	}
	b, ok, err := p.o.Cache.Blob(d.Digest)
	if err != nil || ok {
		return b, err
	}
	if p.o.Offline {
		return nil, fmt.Errorf("layer %s is not in the cache %s, and --offline fetches nothing", d.Digest, p.o.Cache.Dir)
	}
	if b, err = repo.Blob(ctx, d, bundle.MaxLayerSize); err != nil {
		return nil, err
	}
	return b, p.o.Cache.PutBlob(d.Digest, b)
}

// sweep removes every project or segment directory under out that this run
// did not write. A project's history.json is left alone.
func (p *puller) sweep() error {
	projects, err := os.ReadDir(p.o.Out)
	if err != nil {
		return err
	}
	for _, pd := range projects {
		if !pd.IsDir() {
			continue
		}
		pdir := filepath.Join(p.o.Out, pd.Name())
		if _, ok := p.cfg.Tabs[pd.Name()]; !ok {
			if err := os.RemoveAll(pdir); err != nil {
				return err
			}
			continue
		}
		segs, err := os.ReadDir(pdir)
		if err != nil {
			return err
		}
		for _, s := range segs {
			if s.IsDir() && !p.written[pd.Name()+"/"+s.Name()] {
				if err := os.RemoveAll(filepath.Join(pdir, s.Name())); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// IsUsage reports a usage error.
func IsUsage(err error) bool {
	var ue *UsageError
	return errors.As(err, &ue)
}

// parseCreated reads manifest.json's created time; only a local tree built
// before it was recorded lacks it, and packs at the epoch.
func parseCreated(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Unix(0, 0).UTC()
	}
	return t
}
