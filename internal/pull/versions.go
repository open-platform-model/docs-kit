package pull

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-platform-model/docs-kit/internal/build"
	"github.com/open-platform-model/docs-kit/internal/bundle"
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
	anchor, err := p.docsBundle(ctx, sv, v.Anchor.Project, RoleAnchor, v.Anchor.Tag)
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
		b, err := p.docsBundle(ctx, sv, project, RolePinned, pin)
		if err != nil {
			return pinError(err, sv, anchor, project, pin)
		}
		set = append(set, b)
	}
	for _, project := range v.TagProjects() {
		b, err := p.docsBundle(ctx, sv, project, RoleTag, v.Tags[project])
		if err != nil {
			return err
		}
		set = append(set, b)
	}
	return nil
}

// missingTagError is a tag the registry does not have.
type missingTagError struct {
	sv, repo, tag string
	err           error
}

func (e *missingTagError) Error() string {
	return fmt.Sprintf("%s: %s has no tag %s; publish a build in that line, or name a tag that exists in the config (%v)", e.sv, e.repo, e.tag, e.err)
}

func (e *missingTagError) Unwrap() error { return e.err }

// pinError turns a pinned release with no bundle into a message naming
// what to publish.
func pinError(err error, sv string, anchor *docsBundle, project, pin string) error {
	var mt *missingTagError
	if !errors.As(err, &mt) {
		return err
	}
	return fmt.Errorf("%s: %s pins %s %s, and %s has no bundle for it; publish it: run %s's docs workflow in release mode for its %s release",
		sv, anchor.what, project, pin, mt.repo, project, pin)
}

// docsBundle stages one project of a site version at tag, from the
// registry, and records its lock entry.
func (p *puller) docsBundle(ctx context.Context, sv, project, role, tag string) (*docsBundle, error) {
	repo, err := p.o.Client.Repository(p.cfg.Registry + "/" + project)
	if err != nil {
		return nil, err
	}
	desc, err := repo.Resolve(ctx, tag)
	if err != nil {
		if oci.IsNotFound(err) {
			return nil, &missingTagError{sv: sv, repo: repo.Name, tag: tag, err: err}
		}
		return nil, fmt.Errorf("%s %s %s: %w", sv, project, tag, err)
	}
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
