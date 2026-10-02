// Package verify checks a docs bundle's signature: it finds the cosign
// Sigstore bundle that refers to a manifest digest, verifies it with
// sigstore-go against the Sigstore trusted root, and checks the signer's
// identity against the policy: the GitHub OIDC issuer, docs-kit's publish
// workflow at an allowed ref, the repository that owns the project, and
// main.
package verify

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	sgverify "github.com/sigstore/sigstore-go/pkg/verify"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
)

// ArtifactType is the artifact type of a cosign v3 Sigstore bundle referrer.
const ArtifactType = "application/vnd.dev.sigstore.bundle.v0.3+json"

// maxBundle bounds a Sigstore bundle fetch.
const maxBundle = 1 << 20

// Policy is who may have signed a bundle.
type Policy struct {
	Issuer     string   // "https://token.actions.githubusercontent.com"
	Workflow   string   // the reusable workflow URL, without "@<ref>"
	Refs       []string // globs the workflow ref must match, "refs/tags/v[0-9]*"
	Repository string   // the Source Repository URI, "https://github.com/<owner>/<repo>"
	Ref        string   // the Source Repository Ref, "refs/heads/main"
}

// Identity is who signed a bundle, read from its verified certificate.
type Identity struct {
	Workflow   string // the certificate SAN
	Repository string // Source Repository URI
	Ref        string // Source Repository Ref
}

// Verifier verifies Sigstore bundles against trusted material.
type Verifier struct {
	v *sgverify.Verifier
}

// New makes a verifier that requires one signed certificate timestamp, one
// transparency-log entry and one observer timestamp.
func New(tm root.TrustedMaterial) (*Verifier, error) {
	return newVerifier(tm,
		sgverify.WithSignedCertificateTimestamps(1),
		sgverify.WithTransparencyLog(1),
		sgverify.WithObserverTimestamps(1))
}

// NewWithOptions makes a verifier with the given sigstore-go options; tests
// use it with a private certificate authority and no transparency log.
func NewWithOptions(tm root.TrustedMaterial, opts ...sgverify.VerifierOption) (*Verifier, error) {
	return newVerifier(tm, opts...)
}

func newVerifier(tm root.TrustedMaterial, opts ...sgverify.VerifierOption) (*Verifier, error) {
	v, err := sgverify.NewVerifier(tm, opts...)
	if err != nil {
		return nil, fmt.Errorf("making the signature verifier: %w", err)
	}
	return &Verifier{v: v}, nil
}

// UnsignedError means no Sigstore bundle refers to the digest.
type UnsignedError struct{ Digest digest.Digest }

func (e *UnsignedError) Error() string {
	return fmt.Sprintf("%s has no signature: nothing signed it with cosign (no Sigstore bundle refers to it)", e.Digest)
}

// Bundle verifies one Sigstore bundle for the manifest digest d under p.
func (v *Verifier) Bundle(b *bundle.Bundle, d digest.Digest, p Policy) (*Identity, error) {
	if d.Algorithm() != digest.SHA256 {
		return nil, fmt.Errorf("%s: only sha256 digests are verified", d)
	}
	raw, err := hex.DecodeString(d.Encoded())
	if err != nil {
		return nil, err
	}
	res, err := v.v.Verify(b, sgverify.NewPolicy(sgverify.WithArtifactDigest("sha256", raw), sgverify.WithoutIdentitiesUnsafe()))
	if err != nil {
		return nil, fmt.Errorf("verifying the signature of %s: %w", d, err)
	}
	if res.Signature == nil || res.Signature.Certificate == nil {
		return nil, fmt.Errorf("the signature of %s carries no certificate; a docs bundle is signed keyless", d)
	}
	c := res.Signature.Certificate
	id := &Identity{Workflow: c.SubjectAlternativeName, Repository: c.SourceRepositoryURI, Ref: c.SourceRepositoryRef}
	if err := p.check(c.Issuer, id); err != nil {
		return nil, fmt.Errorf("the signature of %s: %w", d, err)
	}
	return id, nil
}

// check applies the identity policy to a verified certificate.
func (p Policy) check(issuer string, id *Identity) error {
	if issuer != p.Issuer {
		return fmt.Errorf("issuer %q, want %q", issuer, p.Issuer)
	}
	wf, ref, ok := strings.Cut(id.Workflow, "@")
	if !ok || wf != p.Workflow {
		return fmt.Errorf("signed by %q, want the workflow %s at an allowed ref", id.Workflow, p.Workflow)
	}
	if !matchAny(p.Refs, ref) {
		return fmt.Errorf("signed by the workflow at %s, which matches none of %s", ref, strings.Join(p.Refs, ", "))
	}
	if id.Repository != p.Repository {
		return fmt.Errorf("signed for the repository %q, want %q (the only repository allowed to sign this project)", id.Repository, p.Repository)
	}
	if id.Ref != p.Ref {
		return fmt.Errorf("signed from %q, want %q", id.Ref, p.Ref)
	}
	return nil
}

func matchAny(globs []string, s string) bool {
	for _, g := range globs {
		if ok, err := path.Match(g, s); err == nil && ok {
			return true
		}
	}
	return false
}

// Find returns the raw Sigstore bundles that refer to subject in a
// repository, through the referrers API or its fallback tag.
func Find(ctx context.Context, repo content.ReadOnlyGraphStorage, subject ocispec.Descriptor) ([][]byte, error) {
	refs, err := registry.Referrers(ctx, repo, subject, ArtifactType)
	if err != nil {
		return nil, fmt.Errorf("finding the signature of %s: %w", subject.Digest, err)
	}
	if len(refs) == 0 {
		return nil, &UnsignedError{Digest: subject.Digest}
	}
	var out [][]byte
	var errs []error
	for _, r := range refs {
		raw, err := fetchBundle(ctx, repo, r)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out = append(out, raw)
	}
	if len(out) == 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}

// Raw verifies one Sigstore bundle given as JSON.
func (v *Verifier) Raw(raw []byte, d digest.Digest, p Policy) (*Identity, error) {
	var b bundle.Bundle
	if err := b.UnmarshalJSON(raw); err != nil {
		return nil, fmt.Errorf("decoding the Sigstore bundle of %s: %w", d, err)
	}
	return v.Bundle(&b, d, p)
}

// Any returns the identity of the first raw bundle that verifies under p,
// and that bundle.
func (v *Verifier) Any(raws [][]byte, d digest.Digest, p Policy) (*Identity, []byte, error) {
	var errs []error
	for _, raw := range raws {
		id, err := v.Raw(raw, d, p)
		if err == nil {
			return id, raw, nil
		}
		errs = append(errs, err)
	}
	return nil, nil, errors.Join(errs...)
}

// Digest finds the Sigstore bundles referring to subject and returns the
// identity of the first that verifies under p.
func (v *Verifier) Digest(ctx context.Context, repo content.ReadOnlyGraphStorage, subject ocispec.Descriptor, p Policy) (*Identity, error) {
	raws, err := Find(ctx, repo, subject)
	if err != nil {
		return nil, err
	}
	id, _, err := v.Any(raws, subject.Digest, p)
	return id, err
}

func fetchBundle(ctx context.Context, repo content.ReadOnlyGraphStorage, ref ocispec.Descriptor) ([]byte, error) {
	if ref.Size > maxBundle {
		return nil, fmt.Errorf("signature manifest %s is too large", ref.Digest)
	}
	m, err := content.FetchAll(ctx, repo, ref)
	if err != nil {
		return nil, fmt.Errorf("fetching signature manifest %s: %w", ref.Digest, err)
	}
	var man ocispec.Manifest
	if err := json.Unmarshal(m, &man); err != nil {
		return nil, fmt.Errorf("decoding signature manifest %s: %w", ref.Digest, err)
	}
	if len(man.Layers) != 1 || man.Layers[0].Size > maxBundle {
		return nil, fmt.Errorf("signature manifest %s does not hold one Sigstore bundle", ref.Digest)
	}
	raw, err := content.FetchAll(ctx, repo, man.Layers[0])
	if err != nil {
		return nil, fmt.Errorf("fetching the Sigstore bundle %s: %w", man.Layers[0].Digest, err)
	}
	return raw, nil
}
