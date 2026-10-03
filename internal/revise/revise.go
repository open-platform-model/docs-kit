// Package revise builds a docs revision of a published release: it checks
// the fix commit, reads the fixes the newest published revision applied,
// applies them and the new fix to the release tree in a temporary
// worktree, refuses anything but a documentation change, and builds the
// next revision. It pushes nothing: push, signing and promote follow, as
// for a release.
package revise

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/build"
	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/gitsrc"
	"github.com/open-platform-model/docs-kit/internal/oci"
	"github.com/open-platform-model/docs-kit/internal/publish"
	"github.com/open-platform-model/docs-kit/internal/tags"
	"github.com/open-platform-model/docs-kit/internal/verify"
)

// DefaultMain is the ref a fix must be on.
const DefaultMain = "origin/main"

// Options configures one revision.
type Options struct {
	Repo     string // the checkout of main, with full history and its tags
	Main     string // the ref the fixes must be on; default origin/main
	Project  string
	Tag      string // the release's git tag, opm-v4.4.5
	Fix      string // the new fix, a 40-hex commit on main
	Out      string // the output root; the bundle goes to <Out>/<Project>
	Registry string // the registry prefix; the repository is <Registry>/<Project>
	Config   string // "" reads docs-kit.cue in the release tree, else in Repo
	Tool     string // the opm-docs version, without "v"
	Client   *oci.Client
	// Verifier and Policy check the signature of the revision whose
	// fixes are read, as pull does, before its source.patches is trusted.
	Verifier func() (*verify.Verifier, error)
	Policy   verify.Policy
}

// Result is the revision built.
type Result struct {
	Dir      string
	Version  string
	Revision int
	Patches  []string
	Pages    int
	// Rebuilt is set when the newest revision already ended with the fix
	// and was not promoted: it is built again under its own number.
	Rebuilt bool
}

// RefusedError lists the changes the documentation-only check refused.
type RefusedError struct {
	Fix     string
	Refused []gitsrc.Refusal
}

func (e *RefusedError) Error() string {
	lines := make([]string, 0, len(e.Refused))
	for _, r := range e.Refused {
		lines = append(lines, "  "+r.String())
	}
	return fmt.Sprintf("the fixes change more than documentation, so no revision is built:\n%s", strings.Join(lines, "\n"))
}

// Run builds the next docs revision of the release Tag with Fix applied
// after every fix the newest published revision carries.
func Run(ctx context.Context, o Options) (*Result, error) {
	if o.Main == "" {
		o.Main = DefaultMain
	}
	repo := gitsrc.Repo{Dir: o.Repo}
	if err := repo.CheckFix(ctx, o.Fix, o.Main); err != nil {
		return nil, err
	}
	version, commit, err := release(ctx, o, repo)
	if err != nil {
		return nil, err
	}
	if repo.IsAncestor(ctx, o.Fix, commit) {
		return nil, fmt.Errorf("the fix %s is already in the release %s; nothing to revise", o.Fix, o.Tag)
	}
	reg, err := o.Client.Repository(o.Registry + "/" + o.Project)
	if err != nil {
		return nil, err
	}
	p := &plan{o: o, reg: reg, version: version, commit: commit}
	if err := p.read(ctx); err != nil {
		return nil, err
	}
	for _, f := range p.prior {
		if err := repo.CheckFix(ctx, f, o.Main); err != nil {
			return nil, fmt.Errorf("%s.%d lists the fix %s, which no longer checks out: %w", version, p.newest.revision, f, err)
		}
	}
	patches := append(slices.Clone(p.prior), o.Fix)
	w, err := repo.AddWorktree(ctx, "refs/tags/"+o.Tag)
	if err != nil {
		return nil, err
	}
	defer func() { _ = w.Remove(ctx) }()
	picked, err := apply(ctx, repo, w, o.Tag, patches)
	if err != nil {
		return nil, err
	}
	results, err := build.Run(ctx, build.Options{
		Config:     buildConfig(o, w.Dir),
		Projects:   []string{o.Project},
		Out:        o.Out,
		Source:     w.Dir,
		Main:       o.Repo, // the checkout of main: a docs page's edit path
		Edits:      p.edits(),
		Release:    o.Tag,
		Tool:       o.Tool,
		Revision:   p.next,
		Patches:    patches,
		PatchDates: picked.Dates,
	})
	if err != nil {
		return nil, err
	}
	return &Result{Dir: results[0].Dir, Version: version, Revision: p.next, Patches: patches, Pages: results[0].Pages, Rebuilt: p.rebuilt}, nil
}

// release checks the tag against the project's prefix and returns its
// version and commit.
func release(ctx context.Context, o Options, repo gitsrc.Repo) (version, commit string, err error) {
	path := o.Config
	if path == "" {
		path = filepath.Join(o.Repo, build.ConfigFile)
	}
	cfg, err := config.Load(path)
	if err != nil {
		return "", "", &build.UsageError{Err: err}
	}
	b, ok := cfg.Bundles[o.Project]
	if !ok {
		return "", "", &build.UsageError{Err: fmt.Errorf("--project %s: %s configures %s", o.Project, cfg.Path, strings.Join(cfg.Projects(), ", "))}
	}
	if !strings.HasPrefix(o.Tag, b.Version.Prefix) {
		return "", "", fmt.Errorf("--tag %s: project %s takes tags with the prefix %q, such as %s4.4.5", o.Tag, o.Project, b.Version.Prefix, b.Version.Prefix)
	}
	version = strings.TrimPrefix(o.Tag, b.Version.Prefix)
	if _, err := tags.ParseVersion(version); err != nil {
		return "", "", fmt.Errorf("--tag %s: %w", o.Tag, err)
	}
	if commit, err = repo.Commit(ctx, "refs/tags/"+o.Tag); err != nil {
		return "", "", fmt.Errorf("the tag %s is not in %s; check out the repository with its tags and full history", o.Tag, o.Repo)
	}
	return version, commit, nil
}

// pushedRevision is one published revision of the release.
type pushedRevision struct {
	revision int
	digest   string
	manifest *bundle.Manifest
}

// plan decides which revision to build and on which fixes.
type plan struct {
	o       Options
	reg     *oci.Repo
	version string
	commit  string
	builds  []tags.Build
	newest  pushedRevision
	next    int      // the revision to build
	prior   []string // the fixes to apply before the new one
	rebuilt bool
}

// read finds the newest published revision and decides. A new fix goes on
// top of a signed newest revision. A re-run of a revision that failed
// after its push (the fix is the newest revision's last, and that revision
// is not promoted) builds the same revision again; when it was never
// signed, its fixes are trusted only as the signed revision before it plus
// this fix.
func (p *plan) read(ctx context.Context) error {
	var err error
	if p.builds, err = publish.ReleaseBuilds(ctx, p.reg); err != nil {
		return err
	}
	v, _ := tags.ParseVersion(p.version)
	next, err := tags.NextRevision(v, p.builds)
	if err != nil {
		return fmt.Errorf("%s has no published release in %s (%s.0): publish the release first: dispatch mode: release", p.o.Tag, p.reg.Name, p.version)
	}
	if p.newest, err = p.fetch(ctx, next-1); err != nil {
		return err
	}
	patches := p.newest.manifest.Source.Patches
	i := slices.Index(patches, p.o.Fix)
	if i < 0 {
		if err := p.signed(ctx, p.newest); err != nil {
			return fmt.Errorf("%w; finish the run that pushed it first: re-run it, or dispatch mode: revision with its last fix", err)
		}
		p.next, p.prior = next, patches
		return nil
	}
	promoted, err := p.promoted(ctx, p.newest)
	if err != nil {
		return err
	}
	if i != len(patches)-1 || promoted {
		return fmt.Errorf("the fix %s is already applied in %s.%d; nothing to revise", p.o.Fix, p.version, p.newest.revision)
	}
	if t := p.newest.manifest.Tool; t != p.o.Tool {
		return fmt.Errorf("%s.%d ends with this fix but is not promoted, and it was built by opm-docs %s, not %s, so building it again would not give the pushed bytes: re-run it with .opm-docs-version at v%s", p.version, p.newest.revision, t, p.o.Tool, t)
	}
	if err := p.signed(ctx, p.newest); err != nil {
		if err := p.trustedBefore(ctx, patches[:i]); err != nil {
			return err
		}
	}
	p.next, p.prior, p.rebuilt = p.newest.revision, patches[:i], true
	return nil
}

// edits is what a rebuilt revision's pages recorded as edit, so building
// it again gives the pushed bytes even after main moved; nil (read main)
// for a new revision.
func (p *plan) edits() map[string]string {
	if !p.rebuilt {
		return nil
	}
	out := map[string]string{}
	for _, pg := range p.newest.manifest.Pages {
		if pg.Edit != "" {
			out[pg.Path] = pg.Edit
		}
	}
	return out
}

// trustedBefore accepts the fixes of an unsigned revision n when revision
// n-1 is signed and applied exactly those.
func (p *plan) trustedBefore(ctx context.Context, prior []string) error {
	n := p.newest.revision
	if n == 0 {
		return fmt.Errorf("%s.0 is not signed; publish the release first: re-run its release", p.version)
	}
	prev, err := p.fetch(ctx, n-1)
	if err != nil {
		return err
	}
	if err := p.signed(ctx, prev); err != nil {
		return fmt.Errorf("%s.%d is not signed and neither is %w", p.version, n, err)
	}
	if !slices.Equal(prev.manifest.Source.Patches, prior) {
		return fmt.Errorf("%s.%d is not signed, and its fixes %v are not those of %s.%d plus this one", p.version, n, p.newest.manifest.Source.Patches, p.version, n-1)
	}
	return nil
}

// fetch reads revision n of the release, checking it is that revision of
// that release built from the tag's commit.
func (p *plan) fetch(ctx context.Context, n int) (pushedRevision, error) {
	full := fmt.Sprintf("%s.%d", p.version, n)
	m, digest, err := fetchManifest(ctx, p.reg, full)
	if err != nil {
		return pushedRevision{}, err
	}
	if m.Version != p.version || m.Revision != n || m.Project != p.o.Project {
		return pushedRevision{}, fmt.Errorf("%s:%s holds %s %s.%d, not %s %s", p.reg.Name, full, m.Project, m.Version, m.Revision, p.o.Project, full)
	}
	if m.Source.Commit != p.commit {
		return pushedRevision{}, fmt.Errorf("%s:%s was built from %s, but %s is %s; a revision applies fixes to the tree that release was built from", p.reg.Name, full, m.Source.Commit, p.o.Tag, p.commit)
	}
	return pushedRevision{revision: n, digest: digest, manifest: m}, nil
}

// signed verifies a revision's signature under the policy pull applies.
func (p *plan) signed(ctx context.Context, r pushedRevision) error {
	if p.o.Verifier == nil {
		return fmt.Errorf("%s.%d: no signature verifier configured", p.version, r.revision)
	}
	v, err := p.o.Verifier()
	if err != nil {
		return err
	}
	desc, err := p.reg.Resolve(ctx, r.digest)
	if err != nil {
		return err
	}
	if _, err := v.Digest(ctx, p.reg.Graph(), desc, p.o.Policy); err != nil {
		return fmt.Errorf("%s.%d (%s) is not signed by docs-kit's publish workflow for this repository: %w", p.version, r.revision, r.digest, err)
	}
	return nil
}

// promoted reports a revision every moving tag promote would move to it
// already names. A partly promoted revision is not: building it again and
// promoting it moves the rest.
func (p *plan) promoted(ctx context.Context, r pushedRevision) (bool, error) {
	d, err := tags.NewBuild(p.version, r.revision, r.digest)
	if err != nil {
		return false, err
	}
	for _, t := range tags.Promotion(d, p.builds) {
		cur, err := p.reg.Resolve(ctx, t)
		if err != nil {
			if oci.IsNotFound(errors.Unwrap(err)) || oci.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		if cur.Digest.String() != r.digest {
			return false, nil
		}
	}
	return true, nil
}

// fetchManifest reads manifest.json of a pushed build: the layer is
// unpacked with every check pull applies, then discarded.
func fetchManifest(ctx context.Context, r *oci.Repo, tag string) (bm *bundle.Manifest, digest string, err error) {
	desc, err := r.Resolve(ctx, tag)
	if err != nil {
		return nil, "", err
	}
	m, _, err := r.Manifest(ctx, desc)
	if err != nil {
		return nil, "", err
	}
	if m.ArtifactType != bundle.ArtifactType || len(m.Layers) != 1 || m.Layers[0].MediaType != bundle.LayerType {
		return nil, "", fmt.Errorf("%s:%s is not a docs bundle", r.Name, tag)
	}
	layer, err := r.Blob(ctx, m.Layers[0], bundle.MaxLayerSize)
	if err != nil {
		return nil, "", err
	}
	tmp, err := os.MkdirTemp("", "opm-docs-revise-bundle-*")
	if err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(tmp)
	bm, err = bundle.Unpack(bytes.NewReader(layer), filepath.Join(tmp, "bundle"), bundle.DefaultLimits)
	if err != nil {
		return nil, "", fmt.Errorf("%s:%s: %w", r.Name, tag, err)
	}
	return bm, desc.Digest.String(), nil
}

// apply picks the patches onto the release tree and refuses a tree with
// anything else in it, a last fix that changes nothing, and anything but
// documentation.
func apply(ctx context.Context, repo gitsrc.Repo, w *gitsrc.Worktree, tag string, patches []string) (*gitsrc.Picked, error) {
	picked, err := w.CherryPick(ctx, patches)
	if err != nil {
		return nil, err
	}
	if dirty, err := w.Unstaged(ctx); err != nil || dirty {
		if err == nil {
			err = fmt.Errorf("applying the fixes left files in the work tree that are not in its index; nothing is built")
		}
		return nil, err
	}
	before := picked.Base
	if len(picked.Trees) > 1 {
		before = picked.Trees[len(picked.Trees)-2]
	}
	tree := picked.Trees[len(picked.Trees)-1]
	if tree == before {
		return nil, fmt.Errorf("the fix %s changes nothing in the release tree with the earlier fixes applied; nothing to revise", patches[len(patches)-1])
	}
	refused, err := repo.DocumentationOnly(ctx, "refs/tags/"+tag, tree)
	if err != nil {
		return nil, err
	}
	if len(refused) > 0 {
		return nil, &RefusedError{Fix: patches[len(patches)-1], Refused: refused}
	}
	return picked, nil
}

// buildConfig is the config build reads: --config when given, else the
// release tree's docs-kit.cue, else the checkout's, as a release build
// picks it.
func buildConfig(o Options, tree string) string {
	if o.Config != "" {
		return o.Config
	}
	if _, err := os.Stat(filepath.Join(tree, build.ConfigFile)); err == nil {
		return ""
	}
	return filepath.Join(o.Repo, build.ConfigFile)
}
