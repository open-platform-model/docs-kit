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
}

// Result is the revision built.
type Result struct {
	Dir      string
	Version  string
	Revision int
	Patches  []string
	Pages    int
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
	next, prior, err := published(ctx, o, version, commit)
	if err != nil {
		return nil, err
	}
	if slices.Contains(prior, o.Fix) {
		return nil, fmt.Errorf("the fix %s is already applied in %s.%d; nothing to revise", o.Fix, version, next-1)
	}
	for _, p := range prior {
		if err := repo.CheckFix(ctx, p, o.Main); err != nil {
			return nil, fmt.Errorf("%s.%d lists the fix %s, which no longer checks out: %w", version, next-1, p, err)
		}
	}
	patches := append(slices.Clone(prior), o.Fix)
	w, err := repo.AddWorktree(ctx, "refs/tags/"+o.Tag)
	if err != nil {
		return nil, err
	}
	defer func() { _ = w.Remove(ctx) }()
	if err := apply(ctx, repo, w, o.Tag, patches); err != nil {
		return nil, err
	}
	results, err := build.Run(ctx, build.Options{
		Config:   buildConfig(o, w.Dir),
		Projects: []string{o.Project},
		Out:      o.Out,
		Source:   w.Dir,
		Release:  o.Tag,
		Tool:     o.Tool,
		Revision: next,
		Patches:  patches,
	})
	if err != nil {
		return nil, err
	}
	return &Result{Dir: results[0].Dir, Version: version, Revision: next, Patches: patches, Pages: results[0].Pages}, nil
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

// published returns the next revision number of version and the fixes the
// newest published revision applied, oldest first.
func published(ctx context.Context, o Options, version, commit string) (next int, patches []string, err error) {
	r, err := o.Client.Repository(o.Registry + "/" + o.Project)
	if err != nil {
		return 0, nil, err
	}
	builds, err := publish.ReleaseBuilds(ctx, r)
	if err != nil {
		return 0, nil, err
	}
	v, _ := tags.ParseVersion(version)
	next, err = tags.NextRevision(v, builds)
	if err != nil {
		return 0, nil, fmt.Errorf("%s has no published release in %s (%s.0): publish the release first: dispatch mode: release", o.Tag, r.Name, version)
	}
	newest := fmt.Sprintf("%s.%d", version, next-1)
	m, err := fetchManifest(ctx, r, newest)
	if err != nil {
		return 0, nil, err
	}
	if m.Version != version || m.Revision != next-1 || m.Project != o.Project {
		return 0, nil, fmt.Errorf("%s:%s holds %s %s.%d, not %s %s", r.Name, newest, m.Project, m.Version, m.Revision, o.Project, newest)
	}
	if m.Source.Commit != commit {
		return 0, nil, fmt.Errorf("%s:%s was built from %s, but %s is %s; a revision applies fixes to the tree that release was built from", r.Name, newest, m.Source.Commit, o.Tag, commit)
	}
	return next, m.Source.Patches, nil
}

// fetchManifest reads manifest.json of a pushed build: the layer is
// unpacked with every check pull applies, then discarded.
func fetchManifest(ctx context.Context, r *oci.Repo, tag string) (*bundle.Manifest, error) {
	desc, err := r.Resolve(ctx, tag)
	if err != nil {
		return nil, err
	}
	m, _, err := r.Manifest(ctx, desc)
	if err != nil {
		return nil, err
	}
	if m.ArtifactType != bundle.ArtifactType || len(m.Layers) != 1 || m.Layers[0].MediaType != bundle.LayerType {
		return nil, fmt.Errorf("%s:%s is not a docs bundle", r.Name, tag)
	}
	layer, err := r.Blob(ctx, m.Layers[0], bundle.MaxLayerSize)
	if err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp("", "opm-docs-revise-bundle-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	bm, err := bundle.Unpack(bytes.NewReader(layer), filepath.Join(tmp, "bundle"), bundle.DefaultLimits)
	if err != nil {
		return nil, fmt.Errorf("%s:%s: %w", r.Name, tag, err)
	}
	return bm, nil
}

// apply picks the patches onto the release tree and refuses anything but
// documentation.
func apply(ctx context.Context, repo gitsrc.Repo, w *gitsrc.Worktree, tag string, patches []string) error {
	if err := w.CherryPick(ctx, patches); err != nil {
		return err
	}
	tree, err := w.IndexTree(ctx)
	if err != nil {
		return err
	}
	refused, err := repo.DocumentationOnly(ctx, "refs/tags/"+tag, tree)
	if err != nil {
		return err
	}
	if len(refused) > 0 {
		return &RefusedError{Fix: patches[len(patches)-1], Refused: refused}
	}
	return nil
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

// IsUsage reports an error that is the caller's: the config or the
// project.
func IsUsage(err error) bool {
	var ue *build.UsageError
	return errors.As(err, &ue)
}
