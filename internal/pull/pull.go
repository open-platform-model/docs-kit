// Package pull is the site's side: it resolves each tab's versions from a
// registry's tags and each site version's docs bundles from its anchor's
// pins, verifies every bundle's signature before fetching its layer,
// unpacks and lints it, writes each tab's version history and the lock.
package pull

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/open-platform-model/docs-kit/internal/build"
	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/extract/cuecatalog"
	"github.com/open-platform-model/docs-kit/internal/history"
	"github.com/open-platform-model/docs-kit/internal/oci"
	"github.com/open-platform-model/docs-kit/internal/tags"
	"github.com/open-platform-model/docs-kit/internal/verify"
)

// Local takes one segment of a tab, or one docs project of a site
// version, from a local bundle tree. Segment is the tab's segment
// ("MAJOR.MINOR" or "edge") or a site version ("v1.0").
type Local struct {
	Project, Segment, Dir string
}

// Site reports whether the tree is a docs project of a site version.
func (l Local) Site() bool { return reSiteVersion.MatchString(l.Segment) }

var reSiteVersion = regexp.MustCompile(`^v(0|[1-9]\d*)\.(0|[1-9]\d*)$`)

// ParseLocal parses "<project>@<segment>=<dir>", where a segment
// "v<MAJOR>.<MINOR>" names a site version and "<MAJOR>.<MINOR>" or "edge"
// a tab's segment.
func ParseLocal(s string) (Local, error) {
	ps, dir, ok := strings.Cut(s, "=")
	p, seg, ok2 := strings.Cut(ps, "@")
	if ok && ok2 && p != "" && dir != "" && reSiteVersion.MatchString(seg) {
		return Local{Project: p, Segment: seg, Dir: dir}, nil
	}
	if !ok || !ok2 || p == "" || dir == "" || (seg != tags.Edge && !tags.IsMinorTag(seg)) {
		return Local{}, fmt.Errorf("--local %s: write <project>@<segment>=<dir>, the segment MAJOR.MINOR or edge for a tab, v<MAJOR>.<MINOR> for a site version", s)
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
	written  map[string]bool   // "<project>/<segment>", once swapped in
	staged   map[string]string // "<project>/<segment>": its unpacked and linted tree, not yet swapped in
	order    []string          // the staged keys, in the order they were staged
	entries  []Entry
	// Site versions, staged and swapped whole: "<v>" to its incoming tree.
	versionStaged  map[string]string
	versionOrder   []string
	versionWritten map[string]bool
	docs           []DocsEntry
	docsLocals     map[string]Local // "<v>/<project>"
	frozenLock     *Lock
}

// Run pulls every tab and every site version, writes each tab's
// history.json and the lock.
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
	p := &puller{o: o, cfg: cfg, written: map[string]bool{}, staged: map[string]string{},
		versionStaged: map[string]string{}, versionWritten: map[string]bool{}, docsLocals: map[string]Local{}}
	// A staged tree left behind by a failure is removed; a swapped one is
	// already gone from its staging path.
	defer p.discard()
	locals, frozen, err := p.prepare()
	if err != nil {
		return nil, err
	}
	p.frozenLock = frozen
	if err := p.stageAll(ctx, locals, frozen); err != nil {
		return nil, err
	}
	if err := p.stageVersions(ctx); err != nil {
		return nil, err
	}
	// Everything that can refuse runs before the first segment is swapped
	// in: a refused bundle, history or lock leaves the previous trees, the
	// previous history.json files and the previous lock as they were.
	histories, err := p.histories()
	if err != nil {
		return nil, err
	}
	lock := &Lock{Schema: LockSchema, Tool: o.Tool, Config: cfg.Digest, Bundles: p.entries, Docs: p.docs}
	for _, h := range histories {
		if h.data != nil {
			lock.History = append(lock.History, h.entry)
		}
	}
	data, err := lock.Encode()
	if err != nil {
		return nil, err
	}
	if err := p.commit(histories, data); err != nil {
		return nil, err
	}
	return lock, nil
}

// stageAll unpacks and lints every segment of every tab, from local trees,
// a frozen lock or the registry.
func (p *puller) stageAll(ctx context.Context, locals map[string][]Local, frozen *Lock) error {
	for _, project := range p.cfg.Projects() {
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
			return err
		}
	}
	return nil
}

// commit swaps the staged segments and site versions in, sweeps, and
// writes the histories and the lock.
func (p *puller) commit(histories []pendingHistory, lock []byte) error {
	if err := p.swap(); err != nil {
		return err
	}
	if err := p.swapVersions(); err != nil {
		return err
	}
	if err := p.sweep(); err != nil {
		return err
	}
	if err := p.sweepVersions(); err != nil {
		return err
	}
	if err := writeHistories(histories); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p.o.Lock), 0o750); err != nil {
		return err
	}
	return os.WriteFile(p.o.Lock, lock, 0o644) //nolint:gosec // the lock is a build output the site reads
}

// prepare groups the --local trees by project and reads a frozen lock.
func (p *puller) prepare() (map[string][]Local, *Lock, error) {
	locals := map[string][]Local{}
	for _, l := range p.o.Locals {
		if l.Site() {
			if err := p.docsLocal(l); err != nil {
				return nil, nil, err
			}
			continue
		}
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

// docsLocal records a --local tree for a docs project of a site version.
func (p *puller) docsLocal(l Local) error {
	sv := l.Segment
	v, ok := p.cfg.Versions[sv]
	if !ok {
		return usagef("--local %s@%s: %s is not a site version in %s", l.Project, sv, sv, p.o.Config)
	}
	if v.Role(l.Project) == "" {
		return usagef("--local %s@%s: %s does not pull %s; it pulls %s", l.Project, sv, sv, l.Project, strings.Join(versionProjects(v), ", "))
	}
	key := sv + "/" + l.Project
	if _, dup := p.docsLocals[key]; dup {
		return usagef("--local %s@%s is given twice", l.Project, sv)
	}
	p.docsLocals[key] = l
	return nil
}

func versionProjects(v config.SiteVersion) []string {
	return append(append([]string{v.Anchor.Project}, v.Pinned...), v.TagProjects()...)
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
	if err := p.checkFrozenDocs(l); err != nil {
		return nil, err
	}
	return l, nil
}

// checkFrozenDocs refuses a frozen lock's docs entry the config would not
// pull as it is written: a site version or project the config does not
// name, another role, another repository or tag, a local entry, or a
// pinned entry off the locked anchor's pin.
func (p *puller) checkFrozenDocs(l *Lock) error {
	anchors := map[string]*DocsEntry{}
	seen := map[string]bool{}
	for i := range l.Docs {
		e := &l.Docs[i]
		if seen[e.Site+"/"+e.Project] {
			return usagef("--frozen %s locks %s %s twice", p.o.Frozen, e.Site, e.Project)
		}
		seen[e.Site+"/"+e.Project] = true
		if err := p.checkFrozenEntry(e); err != nil {
			return err
		}
		if e.Role == RoleAnchor {
			anchors[e.Site] = e
		}
	}
	// Every configured project of every site version is locked, or comes
	// from --local.
	for _, sv := range p.cfg.SiteVersions() {
		v := p.cfg.Versions[sv]
		for _, project := range versionProjects(v) {
			if _, local := p.docsLocals[sv+"/"+project]; !local && !seen[sv+"/"+project] {
				return usagef("--frozen %s has no entry for %s %s, which the config pulls as %s; pull again without --frozen", p.o.Frozen, sv, project, v.Role(project))
			}
		}
	}
	for i := range l.Docs {
		e := &l.Docs[i]
		if e.Role != RolePinned {
			continue
		}
		pin := ""
		if a := anchors[e.Site]; a != nil {
			pin = a.Pins[e.Project]
		}
		if pin == "" || e.Version != pin || e.Tag != pin {
			if pin == "" {
				pin = "no version of it"
			}
			return usagef("--frozen %s: %s %s is locked at version %s, tag %s, and the locked anchor pins %s", p.o.Frozen, e.Site, e.Project, e.Version, e.Tag, pin)
		}
	}
	return nil
}

// checkFrozenEntry checks one docs entry against the config; a pinned
// entry's tag is checked against the anchor's pin by the caller.
func (p *puller) checkFrozenEntry(e *DocsEntry) error {
	name := e.Site + " " + e.Project
	if e.Local {
		return usagef("--frozen %s holds the local entry %s@%s; pass it again with --local", p.o.Frozen, e.Project, e.Site)
	}
	v, ok := p.cfg.Versions[e.Site]
	if !ok {
		return usagef("--frozen %s: %s is not a site version in %s", p.o.Frozen, e.Site, p.o.Config)
	}
	if role := v.Role(e.Project); role != e.Role {
		if role == "" {
			role = "nothing; the version does not pull it"
		}
		return usagef("--frozen %s: %s is locked as %s, and the config pulls it as %s", p.o.Frozen, name, e.Role, role)
	}
	if want := p.cfg.Registry + "/" + e.Project; e.Repository != want {
		return usagef("--frozen %s: %s names the repository %s; the config pulls %s from %s", p.o.Frozen, name, e.Repository, e.Project, want)
	}
	want := v.Tags[e.Project]
	switch e.Role {
	case RolePinned:
		return nil
	case RoleAnchor:
		want = v.Anchor.Tag
	}
	if e.Tag != want {
		return usagef("--frozen %s: %s is locked at tag %s, and the config resolves it at %s", p.o.Frozen, name, e.Tag, want)
	}
	return nil
}

// segmentDir is where a segment unpacks, and the lock's dir for it.
func (p *puller) segmentDir(project, segment string) (abs, rel string, err error) {
	abs = filepath.Join(p.o.Out, project, segment)
	rel, err = p.relToLock(abs)
	return abs, rel, err
}

// relToLock is a path relative to the lock's directory, slash-separated.
func (p *puller) relToLock(path string) (string, error) {
	lockDir, err := filepath.Abs(filepath.Dir(p.o.Lock))
	if err != nil {
		return "", err
	}
	full, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	r, err := filepath.Rel(lockDir, full)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(r), nil
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

// unpack unpacks the layer beside <out>/<project>/<segment> and lints it
// in bundle mode, and stages it: swap moves it into place only once every
// bundle, the history and the lock have passed, so a refusal leaves the
// previous segment in place. It returns the staged tree and the lock's dir
// for the segment. what names the source in errors.
func (p *puller) unpack(project, segment string, layer []byte, what string) (staged, rel string, err error) {
	dir, rel, err := p.segmentDir(project, segment)
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o750); err != nil {
		return "", "", err
	}
	incoming := filepath.Join(filepath.Dir(dir), ".incoming-"+segment)
	if err := os.RemoveAll(incoming); err != nil {
		return "", "", err
	}
	key := project + "/" + segment
	if _, ok := p.staged[key]; !ok {
		p.order = append(p.order, key)
	}
	p.staged[key] = incoming // discard removes it if anything below fails
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
	return incoming, rel, nil
}

// swap moves every staged tree into its segment directory.
func (p *puller) swap() error {
	for _, key := range p.order {
		project, segment, _ := strings.Cut(key, "/")
		dir := filepath.Join(p.o.Out, project, segment)
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		if err := os.Rename(p.staged[key], dir); err != nil {
			return err
		}
		p.written[key] = true
	}
	return nil
}

// discard removes every staged tree that was not swapped in.
func (p *puller) discard() {
	for _, key := range p.order {
		if !p.written[key] {
			_ = os.RemoveAll(p.staged[key])
		}
	}
	for _, sv := range p.versionOrder {
		if !p.versionWritten[sv] {
			_ = os.RemoveAll(p.versionStaged[sv])
		}
	}
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
	f, err := p.verified(ctx, repo, desc, project, tab.Repo, segment, what)
	if err != nil {
		return err
	}
	dir, rel, err := p.unpack(project, segment, f.layer, what)
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
	if err := f.matches(m); err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	p.entries = append(p.entries, Entry{
		Project: project, Root: tab.Root, Segment: segment, Tag: segment, Repository: repo.Name, Digest: desc.Digest.String(),
		Version: f.version, Revision: f.revision, Commit: m.Source.Commit, Dialect: m.Dialect, BuiltBy: m.Tool,
		Signer: &Signer{Workflow: f.id.Workflow, Repository: f.id.Repository, Ref: f.id.Ref}, Dir: rel,
	})
	return nil
}

// fetched is a bundle whose signature verified, with its layer.
type fetched struct {
	annotated
	man   *ocispec.Manifest
	id    *verify.Identity
	layer []byte
}

// matches checks the unpacked manifest.json agrees with the annotations
// that were verified.
func (f *fetched) matches(m *bundle.Manifest) error {
	if m.Version != f.man.Annotations[bundle.AnnVersion] || strconv.Itoa(m.Revision) != f.man.Annotations[bundle.AnnDocsRev] {
		return fmt.Errorf("manifest.json says %s revision %d, the annotations %s revision %s", m.Version, m.Revision, f.man.Annotations[bundle.AnnVersion], f.man.Annotations[bundle.AnnDocsRev])
	}
	return nil
}

// verified checks the build desc names is a bundle of project in tag's
// line, verifies its signature as owner's, and only then fetches its
// layer. what names the bundle in errors.
func (p *puller) verified(ctx context.Context, repo *oci.Repo, desc ocispec.Descriptor, project, owner, tag, what string) (*fetched, error) {
	man, err := p.manifest(ctx, repo, desc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	b, err := checkManifest(man, project, tag)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	id, err := p.verify(ctx, repo, desc, verify.Policy{
		Issuer: p.cfg.Signer.Issuer, Workflow: p.cfg.Signer.Workflow, Refs: p.cfg.Signer.Refs,
		Repository: "https://github.com/" + owner, Ref: "refs/heads/main",
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	layer, err := p.layer(ctx, repo, man.Layers[0])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	return &fetched{annotated: b, man: man, id: id, layer: layer}, nil
}

type annotated struct {
	version  string
	revision int
}

// checkManifest checks the build a tag resolved to: a docs bundle of this
// project, in the tag's line, with one layer.
func checkManifest(man *ocispec.Manifest, project, tag string) (annotated, error) {
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
	if !inLine(tag, b) {
		return a, fmt.Errorf("tag %s names a build of %s, which is not in %s", tag, b, tag)
	}
	return annotated{version: man.Annotations[bundle.AnnVersion], revision: rev}, nil
}

// inLine reports whether build b may be what tag names (C4): an edge build
// for edge, a build of that MAJOR.MINOR for a minor tag, of that MAJOR for
// a major tag, and of exactly that version for a release tag.
func inLine(tag string, b tags.Build) bool {
	switch {
	case tag == tags.Edge:
		return b.Edge
	case b.Edge:
		return false
	case tags.IsMinorTag(tag):
		return b.Version.MinorTag() == tag
	case reMajor.MatchString(tag):
		return b.Version.MajorTag() == tag
	}
	return b.Version.String() == tag
}

var reMajor = regexp.MustCompile(`^(0|[1-9]\d*)$`)

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
// did not write. A project's history.json is left alone, and _versions/ is
// sweepVersions'.
func (p *puller) sweep() error {
	projects, err := os.ReadDir(p.o.Out)
	if err != nil {
		return err
	}
	for _, pd := range projects {
		if !pd.IsDir() || pd.Name() == VersionsDir {
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

// pendingHistory is one tab's history.json, computed and not yet written;
// data is nil when the file is to be removed.
type pendingHistory struct {
	file  string
	data  []byte
	entry HistoryEntry
}

// histories computes <out>/<project>/history.json for every tab with two
// segments or more holding a cue-catalog doc model, from the trees this
// run staged, and marks it for removal from a tab with fewer. It is
// recomputed on every run, since a docs revision changes a segment's data
// after the fact. It writes nothing.
func (p *puller) histories() ([]pendingHistory, error) {
	var out []pendingHistory
	for _, project := range p.cfg.Projects() {
		segs, err := p.catalogSegments(project)
		if err != nil {
			return nil, err
		}
		file := filepath.Join(p.o.Out, project, history.FileName)
		if len(segs) < 2 {
			out = append(out, pendingHistory{file: file})
			continue
		}
		h, err := history.Compute(project, p.o.Tool, segs)
		if err != nil {
			return nil, fmt.Errorf("history for %s: %w", project, err)
		}
		data, err := h.Encode()
		if err != nil {
			return nil, err
		}
		rel, err := p.relToLock(file)
		if err != nil {
			return nil, err
		}
		if rel != project+"/"+history.FileName {
			return nil, usagef("--lock %s: the lock records %s as %s, and a lock with history must sit in --out %s (the default <out>/lock.json)", p.o.Lock, file, rel, p.o.Out)
		}
		out = append(out, pendingHistory{file: file, data: data, entry: HistoryEntry{Project: project, Digest: digest.FromBytes(data).String(), Path: rel}})
	}
	return out, nil
}

// writeHistories writes or removes each tab's history.json.
func writeHistories(hs []pendingHistory) error {
	for _, h := range hs {
		if h.data == nil {
			if err := os.Remove(h.file); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			continue
		}
		if err := os.WriteFile(h.file, h.data, 0o644); err != nil { //nolint:gosec // the history is a build output the site reads
			return err
		}
	}
	return nil
}

// catalogSegments reads the doc model of every segment of a project this
// run staged. A segment whose manifest lists no cue-catalog data file
// takes no part.
func (p *puller) catalogSegments(project string) ([]history.Segment, error) {
	var segs []history.Segment
	for i := range p.entries {
		e := &p.entries[i]
		if e.Project != project {
			continue
		}
		dir := p.staged[project+"/"+e.Segment]
		m, err := bundle.Read(dir)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(m.Data, bundle.DataFile{Path: cuecatalog.DataFile, Schema: cuecatalog.SchemaID}) {
			continue
		}
		source := fmt.Sprintf("%s %s %s/%s", project, e.Segment, bundle.DataDir, cuecatalog.DataFile)
		b, err := os.ReadFile(filepath.Join(dir, bundle.DataDir, cuecatalog.DataFile))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", source, err)
		}
		model, err := cuecatalog.Decode(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", source, err)
		}
		segs = append(segs, history.Segment{Name: e.Segment, Tool: m.Tool, Source: source, Model: model})
	}
	return segs, nil
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
