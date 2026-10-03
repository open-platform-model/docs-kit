package pull

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote/errcode"

	"github.com/open-platform-model/docs-kit/internal/build"
	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/oci"
)

// VersionsDir is the directory under out that holds every site version's
// docs bundles, <out>/_versions/<site-version>/<project>/. No project can
// take the name: a project name has no "_".
const VersionsDir = "_versions"

// docsBundle is one staged docs bundle of a site version.
type docsBundle struct {
	project, role string
	m             *bundle.Manifest
	what          string // how errors name it: "cli 1.0.0-beta.6"
	ref           string // its digest, or "local"
}

// stageVersions unpacks, lints and checks every site version into
// <out>/_versions/.incoming-<v>/, in version order. Nothing is swapped in
// here: swapVersions replaces each version whole once everything passed.
func (p *puller) stageVersions(ctx context.Context) error {
	for _, sv := range p.cfg.SiteVersions() {
		if err := p.stageVersion(ctx, sv); err != nil {
			return err
		}
	}
	return nil
}

// stageVersion resolves one site version: the anchor first, then each
// pinned project at the anchor's pin, then each project by its own tag.
func (p *puller) stageVersion(ctx context.Context, sv string) error {
	v := p.cfg.Versions[sv]
	incoming := filepath.Join(p.o.Out, VersionsDir, ".incoming-"+sv)
	if err := os.RemoveAll(incoming); err != nil {
		return err
	}
	p.versionOrder = append(p.versionOrder, sv)
	p.versionStaged[sv] = incoming // discard removes it if anything below fails
	if err := os.MkdirAll(incoming, 0o750); err != nil {
		return err
	}
	anchor, err := p.docsBundle(ctx, sv, v.Anchor.Project, RoleAnchor, v.Anchor.Tag, nil)
	if err != nil {
		return err
	}
	set := []*docsBundle{anchor}
	for _, project := range v.Pinned {
		pin, ok := anchor.m.Pins[project]
		if !ok {
			return fmt.Errorf("%s: %s (%s) pins no version of %s; a pinned project needs a pin in the anchor's manifest.json, so list it in the anchor's pins.projects, or pull it by its own tag under tags",
				sv, anchor.what, anchor.ref, project)
		}
		b, err := p.docsBundle(ctx, sv, project, RolePinned, pin, anchor)
		if err != nil {
			return pinError(err, sv, anchor, project, pin)
		}
		if b.m.Version != pin {
			return fmt.Errorf("%s: %s (%s) is version %s, and %s pins %s; pull it at its pin", sv, b.what, b.ref, b.m.Version, anchor.what, pin)
		}
		set = append(set, b)
	}
	for _, project := range v.TagProjects() {
		b, err := p.docsBundle(ctx, sv, project, RoleTag, v.Tags[project], nil)
		if err != nil {
			return err
		}
		set = append(set, b)
	}
	return checkOverlap(sv, set)
}

// checkOverlap refuses a site version whose docs bundles overlap: two
// owned paths that nest, a page under a path another bundle owns, or one
// content path in two bundles.
func checkOverlap(sv string, set []*docsBundle) error {
	for _, check := range []func([]*docsBundle) string{nestedOwns, pageUnderOwned, pageTwice} {
		if msg := check(set); msg != "" {
			return fmt.Errorf("%s: %s", sv, msg)
		}
	}
	return nil
}

func nestedOwns(set []*docsBundle) string {
	for i, a := range set {
		for _, b := range set[i+1:] {
			for _, oa := range a.m.Placement.Owns {
				for _, ob := range b.m.Placement.Owns {
					if config.Nests(oa, ob) || config.Nests(ob, oa) {
						return fmt.Sprintf("%s owns %s and %s owns %s; owned paths of one site version never nest, so narrow one of them", a.what, oa, b.what, ob)
					}
				}
			}
		}
	}
	return ""
}

func pageUnderOwned(set []*docsBundle) string {
	for _, a := range set {
		for _, b := range set {
			if a == b {
				continue
			}
			for _, page := range a.m.Pages {
				if o, ok := ownedBy(b, page.Path); ok {
					return fmt.Sprintf("%s has %s, under %s, which %s owns; a page under an owned path belongs to its owner, so move it or drop it", a.what, page.Path, o, b.what)
				}
			}
		}
	}
	return ""
}

// ownedBy returns the path of b's owns that holds page.
func ownedBy(b *docsBundle, page string) (string, bool) {
	for _, o := range b.m.Placement.Owns {
		if config.Nests(o, page) {
			return o, true
		}
	}
	return "", false
}

func pageTwice(set []*docsBundle) string {
	seen := map[string]*docsBundle{}
	for _, b := range set {
		for _, page := range b.m.Pages {
			if first, ok := seen[page.Path]; ok {
				return fmt.Sprintf("%s is in both %s and %s; a page of a site version comes from one bundle, so drop it from one of them", page.Path, first.what, b.what)
			}
			seen[page.Path] = b
		}
	}
	return ""
}

// absent reports an answer that means the tag is not there for an
// anonymous pull: not found, or denied, which is how GHCR answers for a
// package that does not exist yet (or is not public).
func absent(err error) bool {
	if oci.IsNotFound(err) {
		return true
	}
	var re *errcode.ErrorResponse
	return errors.As(err, &re) && (re.StatusCode == http.StatusUnauthorized || re.StatusCode == http.StatusForbidden)
}

// missingTagError is a tag the registry does not have.
type missingTagError struct {
	sv, repo, tag string
	err           error
}

func (e *missingTagError) Error() string {
	return fmt.Sprintf("%s: %s has no tag %s, or is not public; publish a build in that line, or name a tag that exists in the config (%v)", e.sv, e.repo, e.tag, e.err)
}

func (e *missingTagError) Unwrap() error { return e.err }

// pinError turns a pinned release with no bundle into a message naming
// what to publish.
func pinError(err error, sv string, anchor *docsBundle, project, pin string) error {
	var mt *missingTagError
	if !errors.As(err, &mt) {
		return err
	}
	return fmt.Errorf("%s: %s pins %s %s, and %s has no bundle for it (or the package is not public); publish it: run %s's docs workflow in release mode for its %s release",
		sv, anchor.what, project, pin, mt.repo, project, pin)
}

// docsBundle stages one project of a site version and records its lock
// entry: from a --local tree, else from a frozen lock, else from the
// registry at tag. anchor is the staged anchor when the project is pinned.
func (p *puller) docsBundle(ctx context.Context, sv, project, role, tag string, anchor *docsBundle) (*docsBundle, error) {
	if l, ok := p.docsLocals[sv+"/"+project]; ok {
		return p.localDocs(sv, project, role, l, tag, anchor)
	}
	if p.frozenLock != nil {
		return p.frozenDocs(ctx, sv, project, role)
	}
	repo, err := p.o.Client.Repository(p.cfg.Registry + "/" + project)
	if err != nil {
		return nil, err
	}
	desc, err := repo.Resolve(ctx, tag)
	if err != nil {
		if absent(err) {
			return nil, &missingTagError{sv: sv, repo: repo.Name, tag: tag, err: err}
		}
		return nil, fmt.Errorf("%s %s %s: %w", sv, project, tag, err)
	}
	return p.fetchDocs(ctx, sv, project, role, tag, repo, desc)
}

// frozenDocs fetches the digest a frozen lock names for a project of a
// site version.
func (p *puller) frozenDocs(ctx context.Context, sv, project, role string) (*docsBundle, error) {
	i := slices.IndexFunc(p.frozenLock.Docs, func(e DocsEntry) bool { return e.Site == sv && e.Project == project })
	if i < 0 {
		return nil, usagef("--frozen %s has no entry for %s %s, which the config pulls as %s; pull again without --frozen", p.o.Frozen, sv, project, role)
	}
	e := p.frozenLock.Docs[i]
	repo, err := p.o.Client.Repository(e.Repository)
	if err != nil {
		return nil, err
	}
	d, err := digest.Parse(e.Digest)
	if err != nil {
		return nil, err
	}
	return p.fetchDocs(ctx, sv, project, role, e.Tag, repo, ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: d, Size: -1})
}

// fetchDocs verifies one docs bundle as its project's, and only then
// fetches, unpacks and lints it into the staged version.
func (p *puller) fetchDocs(ctx context.Context, sv, project, role, tag string, repo *oci.Repo, desc ocispec.Descriptor) (*docsBundle, error) {
	what := fmt.Sprintf("%s %s %s (%s@%s)", sv, project, tag, repo.Name, desc.Digest)
	f, err := p.verified(ctx, repo, desc, project, p.cfg.Docs[project].Repo, tag, what)
	if err != nil {
		return nil, err
	}
	m, rel, err := p.unpackDocs(sv, project, f.layer, what)
	if err != nil {
		return nil, err
	}
	if err := checkDocsBundle(m, project); err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	if err := f.matches(m); err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	p.docs = append(p.docs, DocsEntry{
		Site: sv, Project: project, Role: role, Tag: tag, Repository: repo.Name, Digest: desc.Digest.String(),
		Version: f.version, Revision: f.revision, Commit: m.Source.Commit, Dialect: m.Dialect, BuiltBy: m.Tool,
		Signer: &Signer{Workflow: f.id.Workflow, Repository: f.id.Repository, Ref: f.id.Ref}, Pins: anchorPins(role, m), Dir: rel,
	})
	return &docsBundle{project: project, role: role, m: m, what: project + " " + m.Version, ref: desc.Digest.String()}, nil
}

// localDocs takes a project of a site version from a local tree, with no
// registry and no signature. A pinned tree must be the pinned release.
func (p *puller) localDocs(sv, project, role string, l Local, pin string, anchor *docsBundle) (*docsBundle, error) {
	flag := fmt.Sprintf("--local %s@%s=%s", project, sv, l.Dir)
	m, err := bundle.Read(l.Dir)
	if err != nil {
		return nil, err
	}
	if err := checkDocsBundle(m, project); err != nil {
		return nil, fmt.Errorf("%s: %w", flag, err)
	}
	if role == RolePinned && m.Version != pin {
		return nil, usagef("%s: the tree is %s %s, and %s pins %s %s; build the pinned release, or drop --local to pull it", flag, project, m.Version, anchor.what, project, pin)
	}
	// A round trip through the layer applies the same guards and limits a
	// pulled bundle gets.
	layer, err := bundle.Pack(l.Dir, parseCreated(m.Created))
	if err != nil {
		return nil, err
	}
	m, rel, err := p.unpackDocs(sv, project, layer, fmt.Sprintf("--local %s@%s", project, sv))
	if err != nil {
		return nil, err
	}
	p.docs = append(p.docs, DocsEntry{
		Site: sv, Project: project, Role: role, Local: true,
		Version: m.Version, Revision: m.Revision, Commit: m.Source.Commit, Dialect: m.Dialect, BuiltBy: m.Tool, Pins: anchorPins(role, m), Dir: rel,
	})
	return &docsBundle{project: project, role: role, m: m, what: project + " " + m.Version, ref: "local"}, nil
}

// anchorPins is what the lock records of a bundle's pins: the anchor's,
// as read; nothing for any other role.
func anchorPins(role string, m *bundle.Manifest) map[string]string {
	if role != RoleAnchor || len(m.Pins) == 0 {
		return nil
	}
	return m.Pins
}

// checkDocsBundle checks a manifest belongs in a site version.
func checkDocsBundle(m *bundle.Manifest, project string) error {
	if m.Project != project {
		return fmt.Errorf("the bundle is project %s, not %s", m.Project, project)
	}
	if m.Placement.Kind != "docs" {
		return fmt.Errorf("the bundle's placement is %s %s; a site version takes only docs bundles (placement docs)", m.Placement.Kind, m.Placement.Root)
	}
	return nil
}

// unpackDocs unpacks a layer into the staged version and lints it in
// bundle mode. It returns the manifest and the lock's dir for it, the
// directory it will have once the version is swapped in.
func (p *puller) unpackDocs(sv, project string, layer []byte, what string) (*bundle.Manifest, string, error) {
	dir := filepath.Join(p.versionStaged[sv], project)
	if _, err := bundle.Unpack(bytes.NewReader(layer), dir, bundle.DefaultLimits); err != nil {
		return nil, "", fmt.Errorf("%s: %w", what, err)
	}
	vs, err := build.Lint(dir)
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", what, err)
	}
	if len(vs) > 0 {
		return nil, "", &LintError{What: what, Violations: vs}
	}
	m, err := bundle.Read(dir)
	if err != nil {
		return nil, "", err
	}
	rel, err := p.relToLock(filepath.Join(p.o.Out, VersionsDir, sv, project))
	if err != nil {
		return nil, "", err
	}
	return m, rel, nil
}

// swapVersions replaces each staged site version whole.
func (p *puller) swapVersions() error {
	for _, sv := range p.versionOrder {
		dir := filepath.Join(p.o.Out, VersionsDir, sv)
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		if err := os.Rename(p.versionStaged[sv], dir); err != nil {
			return err
		}
		p.versionWritten[sv] = true
	}
	return nil
}

// sweepVersions removes every entry under <out>/_versions/ this run did
// not write, and the directory itself when the config has no versions.
func (p *puller) sweepVersions() error {
	base := filepath.Join(p.o.Out, VersionsDir)
	if len(p.cfg.Versions) == 0 {
		return os.RemoveAll(base)
	}
	es, err := os.ReadDir(base)
	if err != nil {
		return err
	}
	for _, e := range es {
		if !p.versionWritten[e.Name()] {
			if err := os.RemoveAll(filepath.Join(base, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
