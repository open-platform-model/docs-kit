// Package publish writes bundles to a registry: push uploads a built
// bundle under its immutable full tag (or by digest for edge), and promote
// verifies a pushed digest's signature and moves the moving tags of its
// line to it.
package publish

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/open-platform-model/docs-kit/internal/build"
	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/gitsrc"
	"github.com/open-platform-model/docs-kit/internal/oci"
	"github.com/open-platform-model/docs-kit/internal/tags"
	"github.com/open-platform-model/docs-kit/internal/verify"
)

// DefaultRegistry is where bundles live.
const DefaultRegistry = "ghcr.io/open-platform-model/docs"

// PushOptions configures a push.
type PushOptions struct {
	Dir      string // a bundle directory, out/<project>
	Registry string // the registry prefix; the repository is <Registry>/<project>
	Client   *oci.Client
	// Limits and MaxLayer default to the limits pull applies; tests lower
	// them.
	Limits   bundle.Limits
	MaxLayer int
}

// PushResult is what push prints.
type PushResult struct {
	Digest string `json:"digest"`
	Tag    string `json:"tag"` // the full tag, "" for edge
	// Existing is set when the full tag already named this digest and
	// nothing was written.
	Existing bool `json:"-"`
}

// Push validates a built bundle, packs it deterministically and pushes it:
// a release build under its full tag, which it never overwrites; an edge
// build by digest only.
func Push(ctx context.Context, o PushOptions) (*PushResult, error) {
	m, created, err := readForPush(o.Dir)
	if err != nil {
		return nil, err
	}
	b, err := tags.NewBuild(m.Version, m.Revision, "")
	if err != nil {
		return nil, err
	}
	layer, err := packWithin(o, created)
	if err != nil {
		return nil, err
	}
	md, raw, ld, err := oci.Pack(ctx, bundle.ArtifactType, bundle.LayerType, layer, m.Annotations())
	if err != nil {
		return nil, err
	}
	repo, err := o.Client.Repository(o.Registry + "/" + m.Project)
	if err != nil {
		return nil, err
	}
	res := &PushResult{Digest: md.Digest.String(), Tag: b.FullTag()}
	if res.Tag != "" {
		cur, err := repo.Resolve(ctx, res.Tag)
		switch {
		case err == nil && cur.Digest == md.Digest:
			res.Existing = true
			return res, nil
		case err == nil:
			return nil, fmt.Errorf("%s is already published as %s, and this build is %s; a full tag is never overwritten, and a documentation fix is a docs revision", res.Tag, cur.Digest, md.Digest)
		case !oci.IsNotFound(err):
			return nil, err
		}
	}
	if err := repo.PushBundle(ctx, md, raw, ld, layer, res.Tag); err != nil {
		return nil, err
	}
	return res, nil
}

// packWithin packs the bundle, refusing one over the limits pull applies.
func packWithin(o PushOptions, created time.Time) ([]byte, error) {
	if o.Limits == (bundle.Limits{}) {
		o.Limits = bundle.DefaultLimits
	}
	if o.MaxLayer == 0 {
		o.MaxLayer = bundle.MaxLayerSize
	}
	if err := bundle.CheckLimits(o.Dir, o.Limits); err != nil {
		return nil, fmt.Errorf("%w; a pull would refuse it", err)
	}
	layer, err := bundle.Pack(o.Dir, created)
	if err != nil {
		return nil, err
	}
	if len(layer) > o.MaxLayer {
		return nil, fmt.Errorf("%s packs to a %d-byte layer, more than the %d bytes a pull fetches", o.Dir, len(layer), o.MaxLayer)
	}
	return layer, nil
}

// readForPush reads and checks a bundle a push may publish: valid, clean,
// with its created time, and passing the bundle-mode lint.
func readForPush(dir string) (*bundle.Manifest, time.Time, error) {
	m, err := bundle.Read(dir)
	if err != nil {
		return nil, time.Time{}, err
	}
	if gitsrc.IsLocal(m.Source.Repo) {
		return nil, time.Time{}, fmt.Errorf("%s was built in a repository with no GitHub origin (source.repo %s), so it is a preview: its source links name no real repository; build in a clone of the repository", dir, m.Source.Repo)
	}
	if m.Source.Dirty {
		return nil, time.Time{}, fmt.Errorf("%s was built from a work tree with uncommitted changes (source.dirty); build from a clean checkout", dir)
	}
	if m.Created == "" {
		return nil, time.Time{}, fmt.Errorf("%s: manifest.json has no created time; rebuild it with this opm-docs", dir)
	}
	created, err := time.Parse(time.RFC3339, m.Created)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("%s: created %q: %w", dir, m.Created, err)
	}
	vs, err := build.Lint(dir)
	if err != nil {
		return nil, time.Time{}, err
	}
	if len(vs) > 0 {
		return nil, time.Time{}, &build.LintError{Project: m.Project, Violations: vs}
	}
	return m, created, nil
}

// PromoteOptions configures a promote.
type PromoteOptions struct {
	Project  string
	Digest   string
	Registry string
	Client   *oci.Client
	Verifier *verify.Verifier
	Policy   verify.Policy
	Log      func(string)
}

// PromoteResult lists what promote did.
type PromoteResult struct {
	Build   string
	Moved   []string
	Skipped []string
}

// Promote verifies the signature of a pushed digest D, then moves each
// moving tag of D's line to D when D is the newest build of that line. Just
// before each move it resolves the tag again and leaves it when it already
// names a newer build, since releases of different versions may publish at
// the same time.
func Promote(ctx context.Context, o PromoteOptions) (*PromoteResult, error) {
	logf := o.Log
	if logf == nil {
		logf = func(string) {}
	}
	repo, err := o.Client.Repository(o.Registry + "/" + o.Project)
	if err != nil {
		return nil, err
	}
	desc, err := repo.Settled(ctx, o.Digest)
	if err != nil {
		return nil, err
	}
	pd, err := buildOf(ctx, repo, desc)
	if err != nil {
		return nil, err
	}
	d, raw := pd.build, pd.raw
	if _, err := o.Verifier.Digest(ctx, repo.Graph(), desc, o.Policy); err != nil {
		return nil, fmt.Errorf("%s@%s is not promoted: %w", repo.Name, desc.Digest, err)
	}
	builds := []tags.Build{d}
	if !d.Edge {
		if builds, err = ReleaseBuilds(ctx, repo); err != nil {
			return nil, err
		}
	}
	res := &PromoteResult{Build: d.String()}
	for _, t := range tags.Promotion(d, builds) {
		newer, err := newerAt(ctx, repo, t, pd)
		if err != nil {
			return nil, err
		}
		if newer != "" {
			logf(fmt.Sprintf("%s stays on %s, which is newer than %s", t, newer, d))
			res.Skipped = append(res.Skipped, t)
			continue
		}
		if err := repo.Tag(ctx, desc, raw, t); err != nil {
			return nil, err
		}
		logf(fmt.Sprintf("%s -> %s (%s)", t, desc.Digest, d))
		res.Moved = append(res.Moved, t)
	}
	for _, l := range tags.Lines(d) {
		if !contains(res.Moved, l.Tag) && !contains(res.Skipped, l.Tag) {
			res.Skipped = append(res.Skipped, l.Tag)
		}
	}
	return res, nil
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// pushedBuild is a pushed manifest's build identity, read from its
// annotations, and its bytes.
type pushedBuild struct {
	build   tags.Build
	created string // org.opencontainers.image.created
	raw     []byte
}

// buildOf reads a manifest's build identity from its annotations.
func buildOf(ctx context.Context, repo *oci.Repo, desc ocispec.Descriptor) (pushedBuild, error) {
	m, raw, err := repo.Manifest(ctx, desc)
	if err != nil {
		return pushedBuild{}, err
	}
	if m.ArtifactType != bundle.ArtifactType {
		return pushedBuild{}, fmt.Errorf("%s@%s is not a docs bundle (artifactType %q)", repo.Name, desc.Digest, m.ArtifactType)
	}
	rev, err := strconv.Atoi(m.Annotations[bundle.AnnDocsRev])
	if err != nil {
		return pushedBuild{}, fmt.Errorf("%s@%s: annotation %s: %w", repo.Name, desc.Digest, bundle.AnnDocsRev, err)
	}
	b, err := tags.NewBuild(m.Annotations[bundle.AnnVersion], rev, desc.Digest.String())
	if err != nil {
		return pushedBuild{}, fmt.Errorf("%s@%s: %w", repo.Name, desc.Digest, err)
	}
	return pushedBuild{build: b, created: m.Annotations[bundle.AnnCreated], raw: raw}, nil
}

// ReleaseBuilds lists the repository's release builds: every tag of the
// form <version>.<revision> whose manifest's annotations say exactly that.
func ReleaseBuilds(ctx context.Context, repo *oci.Repo) ([]tags.Build, error) {
	names, err := repo.BuildTags(ctx)
	if err != nil {
		return nil, err
	}
	var out []tags.Build
	for _, n := range names {
		if _, _, ok := tags.SplitFullTag(n); !ok {
			continue
		}
		desc, err := repo.Resolve(ctx, n)
		if err != nil {
			return nil, err
		}
		pb, err := buildOf(ctx, repo, desc)
		if err != nil {
			return nil, err
		}
		if b := pb.build; !b.Edge && b.FullTag() == n {
			out = append(out, b)
		}
	}
	return out, nil
}

// newerAt returns the build tag t names now when it is newer than d, else
// "". It refuses a tag that already is some build's full tag, which a
// prerelease's release tag can collide with (the release tag 1.0.0-beta.5
// is also the full tag of 1.0.0-beta revision 5): a full tag never moves.
// An edge tag never moves back to an older commit: the current edge build
// stays when its created time is later than d's.
func newerAt(ctx context.Context, repo *oci.Repo, t string, d pushedBuild) (string, error) {
	cur, err := repo.Resolve(ctx, t)
	if err != nil {
		if oci.IsNotFound(errors.Unwrap(err)) || oci.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}
	if cur.Digest.String() == d.build.Digest {
		return "", nil
	}
	pb, err := buildOf(ctx, repo, cur)
	if err != nil {
		return "", err
	}
	b := pb.build
	if !b.Edge && b.FullTag() == t {
		return "", fmt.Errorf("%s is the full tag of build %s@%s and never moves; %s cannot take it as a moving tag", t, b, cur.Digest, d.build)
	}
	if d.build.Edge {
		if b.Edge && laterThan(pb.created, d.created) {
			return fmt.Sprintf("edge built at %s", pb.created), nil
		}
		return "", nil
	}
	if !b.Edge && tags.CompareBuilds(b, d.build) > 0 {
		return b.String(), nil
	}
	return "", nil
}

// laterThan compares two RFC 3339 times; an unreadable time is never later.
func laterThan(a, b string) bool {
	ta, err1 := time.Parse(time.RFC3339, a)
	tb, err2 := time.Parse(time.RFC3339, b)
	return err1 == nil && err2 == nil && ta.After(tb)
}
