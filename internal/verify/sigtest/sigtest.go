// Package sigtest signs docs bundles for tests the way cosign does in
// GitHub Actions, with a private certificate authority: a keyless
// certificate carrying the GitHub OIDC extensions, a DSSE in-toto
// statement over the manifest digest, stored as a Sigstore bundle referrer.
package sigtest

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/sign"
	"github.com/sigstore/sigstore-go/pkg/testing/ca"
	sgverify "github.com/sigstore/sigstore-go/pkg/verify"
	"google.golang.org/protobuf/encoding/protojson"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry/remote"

	"github.com/open-platform-model/docs-kit/internal/verify"
)

// Identity is what a signing certificate says about the signer.
type Identity struct {
	Issuer     string
	Workflow   string // the SAN, "<workflow URL>@<ref>"
	Repository string // Source Repository URI
	Ref        string // Source Repository Ref
}

// Publisher is the identity docs-kit's publish workflow at v0.1.0 has when
// repo's main branch calls it.
func Publisher(repo string) Identity {
	return Identity{
		Issuer:     "https://token.actions.githubusercontent.com",
		Workflow:   "https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml@refs/tags/v0.1.0",
		Repository: "https://github.com/" + repo,
		Ref:        "refs/heads/main",
	}
}

// Policy is the pull policy that accepts Publisher(repo).
func Policy(repo string) verify.Policy {
	return verify.Policy{
		Issuer:     "https://token.actions.githubusercontent.com",
		Workflow:   "https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml",
		Refs:       []string{"refs/tags/v*"},
		Repository: "https://github.com/" + repo,
		Ref:        "refs/heads/main",
	}
}

// Authority is a private Fulcio stand-in.
type Authority struct {
	root, inter *x509.Certificate
	interKey    *ecdsa.PrivateKey
}

// New makes an authority.
func New(t testing.TB) *Authority {
	t.Helper()
	rootCert, rootKey, err := ca.GenerateRootCa()
	if err != nil {
		t.Fatal(err)
	}
	inter, interKey, err := ca.GenerateFulcioIntermediate(rootCert, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	return &Authority{root: rootCert, inter: inter, interKey: interKey}
}

type trusted struct {
	root.BaseTrustedMaterial
	cas []root.CertificateAuthority
}

func (t *trusted) FulcioCertificateAuthorities() []root.CertificateAuthority { return t.cas }

// Verifier verifies against the authority at the current time, with no
// transparency log or timestamp authority.
func (a *Authority) Verifier(t testing.TB) *verify.Verifier {
	t.Helper()
	tm := &trusted{cas: []root.CertificateAuthority{&root.FulcioCertificateAuthority{
		Root:                a.root,
		Intermediates:       []*x509.Certificate{a.inter},
		ValidityPeriodStart: time.Now().Add(-time.Hour),
		ValidityPeriodEnd:   time.Now().Add(time.Hour),
	}}}
	v, err := verify.NewWithOptions(tm, sgverify.WithCurrentTime())
	if err != nil {
		t.Fatal(err)
	}
	return v
}

type certProvider struct {
	a  *Authority
	id Identity
}

func ext(oid asn1.ObjectIdentifier, v string) (pkix.Extension, error) {
	b, err := asn1.MarshalWithParams(v, "utf8")
	return pkix.Extension{Id: oid, Value: b}, err
}

func (p certProvider) GetCertificate(_ context.Context, kp sign.Keypair, _ *sign.CertificateProviderOptions) ([]byte, error) {
	san, err := url.Parse(p.id.Workflow)
	if err != nil {
		return nil, err
	}
	var exts []pkix.Extension
	for _, e := range []struct {
		oid asn1.ObjectIdentifier
		v   string
	}{
		{certificate.OIDIssuerV2, p.id.Issuer},
		{certificate.OIDBuildSignerURI, p.id.Workflow},
		{certificate.OIDSourceRepositoryURI, p.id.Repository},
		{certificate.OIDSourceRepositoryRef, p.id.Ref},
	} {
		x, err := ext(e.oid, e.v)
		if err != nil {
			return nil, err
		}
		exts = append(exts, x)
	}
	tmpl := &x509.Certificate{
		SerialNumber:    big.NewInt(time.Now().UnixNano()),
		URIs:            []*url.URL{san},
		NotBefore:       time.Now().Add(-time.Minute),
		NotAfter:        time.Now().Add(10 * time.Minute),
		KeyUsage:        x509.KeyUsageDigitalSignature,
		ExtKeyUsage:     []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		ExtraExtensions: exts,
	}
	return x509.CreateCertificate(rand.Reader, tmpl, p.a.inter, kp.GetPublicKey(), p.a.interKey)
}

// Bundle signs digest d as id and returns the Sigstore bundle JSON.
func (a *Authority) Bundle(t testing.TB, d digest.Digest, id Identity) []byte {
	t.Helper()
	statement, err := json.Marshal(map[string]any{
		"_type":         "https://in-toto.io/Statement/v1",
		"subject":       []any{map[string]any{"digest": map[string]string{"sha256": d.Encoded()}}},
		"predicateType": "https://sigstore.dev/cosign/sign/v1",
		"predicate":     map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	kp, err := sign.NewEphemeralKeypair(&sign.EphemeralKeypairOptions{})
	if err != nil {
		t.Fatal(err)
	}
	pb, err := sign.Bundle(&sign.DSSEData{Data: statement, PayloadType: "application/vnd.in-toto+json"}, kp,
		sign.BundleOptions{CertificateProvider: certProvider{a: a, id: id}})
	if err != nil {
		t.Fatal(err)
	}
	out, err := protojson.Marshal(pb)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// Sign stores a Sigstore bundle for subject in repo as cosign v3 does: a
// referrer manifest of the bundle artifact type with the bundle as its one
// layer. A registry without the referrers API gets the fallback tag.
func (a *Authority) Sign(t testing.TB, repo *remote.Repository, subject ocispec.Descriptor, id Identity) {
	t.Helper()
	ctx := context.Background()
	b := a.Bundle(t, subject.Digest, id)
	ld := content.NewDescriptorFromBytes(verify.ArtifactType, b)
	if err := repo.Push(ctx, ld, bytes.NewReader(b)); err != nil {
		t.Fatal(err)
	}
	empty := ocispec.DescriptorEmptyJSON
	if ok, _ := repo.Exists(ctx, empty); !ok {
		if err := repo.Push(ctx, empty, bytes.NewReader(empty.Data)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := oras.PackManifest(ctx, repo, oras.PackManifestVersion1_1, verify.ArtifactType, oras.PackManifestOptions{
		Subject:             &subject,
		Layers:              []ocispec.Descriptor{ld},
		ManifestAnnotations: map[string]string{ocispec.AnnotationCreated: time.Now().UTC().Format(time.RFC3339)},
	}); err != nil {
		t.Fatal(fmt.Errorf("pushing the signature of %s: %w", subject.Digest, err))
	}
}
