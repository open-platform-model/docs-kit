package verify_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry/remote"

	"github.com/open-platform-model/docs-kit/internal/ocitest"
	"github.com/open-platform-model/docs-kit/internal/verify"
	"github.com/open-platform-model/docs-kit/internal/verify/sigtest"
)

// The spike's real signature: a reusable workflow on a spike branch
// signing for docs-kit.
const spikeDigest = digest.Digest("sha256:3891fc3d62ce77fd33f838f7b7da4a780b1dfff554a39c1998614a6f8bc1399d")

func spikePolicy() verify.Policy {
	return verify.Policy{
		Issuer:     "https://token.actions.githubusercontent.com",
		Workflow:   "https://github.com/open-platform-model/docs-kit/.github/workflows/spike-sign.yml",
		Refs:       []string{"refs/heads/spike/*"},
		Repository: "https://github.com/open-platform-model/docs-kit",
		Ref:        "refs/heads/spike/ghcr",
	}
}

func publicGood(t *testing.T) *verify.Verifier {
	t.Helper()
	tm, err := verify.TrustedRoot(verify.TrustOptions{File: "testdata/public-good-trusted-root.json"})
	if err != nil {
		t.Fatal(err)
	}
	v, err := verify.New(tm)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func spikeBundle(t *testing.T) *bundle.Bundle {
	t.Helper()
	raw, err := os.ReadFile("testdata/spike-bundle.sigstore.json")
	if err != nil {
		t.Fatal(err)
	}
	var b bundle.Bundle
	if err := b.UnmarshalJSON(raw); err != nil {
		t.Fatal(err)
	}
	return &b
}

// TestRealSignature verifies the spike's GitHub Actions signature against
// the public-good trusted root, with SCT, transparency log and observer
// timestamp required, and refuses every identity the policy does not name.
func TestRealSignature(t *testing.T) {
	v := publicGood(t)
	b := spikeBundle(t)
	id, err := v.Bundle(b, spikeDigest, spikePolicy())
	if err != nil {
		t.Fatal(err)
	}
	if id.Repository != "https://github.com/open-platform-model/docs-kit" || id.Ref != "refs/heads/spike/ghcr" ||
		!strings.HasSuffix(id.Workflow, "spike-sign.yml@refs/heads/spike/ghcr") {
		t.Fatalf("identity %+v", id)
	}
	cases := []struct {
		name string
		edit func(*verify.Policy)
		d    digest.Digest
		want string
	}{
		{"wrong issuer", func(p *verify.Policy) { p.Issuer = "https://accounts.google.com" }, spikeDigest, "issuer"},
		{"workflow ref outside the allowed globs", func(p *verify.Policy) { p.Refs = []string{"refs/tags/v*"} }, spikeDigest, "matches none of refs/tags/v*"},
		{"the caller as the workflow", func(p *verify.Policy) {
			p.Workflow = "https://github.com/open-platform-model/docs-kit/.github/workflows/spike.yml"
		}, spikeDigest, "want the workflow"},
		{"wrong source repository", func(p *verify.Policy) { p.Repository = "https://github.com/open-platform-model/cli" }, spikeDigest, "only repository allowed"},
		{"wrong source ref", func(p *verify.Policy) { p.Ref = "refs/heads/main" }, spikeDigest, `want "refs/heads/main"`},
		{"digest mismatch", func(*verify.Policy) {}, digest.FromString("another manifest"), "verifying the signature"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := spikePolicy()
			c.edit(&p)
			_, err := v.Bundle(b, c.d, p)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want one containing %q", err, c.want)
			}
		})
	}
}

// pushRaw stores exact bytes in a test registry.
func pushRaw(t *testing.T, repo *remote.Repository, mediaType string, b []byte) ocispec.Descriptor {
	t.Helper()
	d := content.NewDescriptorFromBytes(mediaType, b)
	if err := repo.Push(context.Background(), d, bytes.NewReader(b)); err != nil {
		t.Fatal(err)
	}
	return d
}

// TestRealSignatureFromRegistry finds the spike's signature as GHCR stores
// it, through the referrers fallback tag, and verifies it.
func TestRealSignatureFromRegistry(t *testing.T) {
	host := ocitest.Registry(t)
	repo, err := remote.NewRepository(host + "/docs/spike")
	if err != nil {
		t.Fatal(err)
	}
	repo.PlainHTTP = true
	manifest, _ := os.ReadFile("testdata/spike-manifest.json")
	referrer, _ := os.ReadFile("testdata/spike-referrer.json")
	sig, _ := os.ReadFile("testdata/spike-bundle.sigstore.json")
	pushRaw(t, repo, ocispec.MediaTypeEmptyJSON, []byte("{}"))
	pushRaw(t, repo, verify.ArtifactType, sig)
	subject := pushRaw(t, repo, ocispec.MediaTypeImageManifest, manifest)
	if subject.Digest != spikeDigest {
		t.Fatalf("fixture digest %s", subject.Digest)
	}
	v := publicGood(t)
	if _, err := v.Digest(context.Background(), repo, subject, spikePolicy()); err == nil {
		t.Fatal("verified before the signature was stored")
	} else if ue := new(verify.UnsignedError); !errors.As(err, &ue) {
		t.Fatalf("err = %v, want an UnsignedError", err)
	}
	pushRaw(t, repo, ocispec.MediaTypeImageManifest, referrer)
	if _, err := v.Digest(context.Background(), repo, subject, spikePolicy()); err != nil {
		t.Fatal(err)
	}
}

// TestPrivateAuthority signs arbitrary digests with chosen identities, as
// the publish workflow would, and checks the policy on each.
func TestPrivateAuthority(t *testing.T) {
	a := sigtest.New(t)
	v := a.Verifier(t)
	d := digest.FromString("a bundle manifest")
	good := sigtest.Publisher("open-platform-model/catalog_opm")
	policy := sigtest.Policy("open-platform-model/catalog_opm")
	if _, err := v.Bundle(load(t, a.Bundle(t, d, good)), d, policy); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		edit func(*sigtest.Identity)
		want string
	}{
		{"wrong issuer", func(i *sigtest.Identity) { i.Issuer = "https://accounts.google.com" }, "issuer"},
		{"workflow at main of docs-kit", func(i *sigtest.Identity) {
			i.Workflow = "https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml@refs/heads/main"
		}, "matches none of"},
		{"a tag that is not a release", func(i *sigtest.Identity) {
			i.Workflow = "https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml@refs/tags/vendor-test"
		}, "matches none of"},
		{"wrong source repository", func(i *sigtest.Identity) { i.Repository = "https://github.com/open-platform-model/cli" }, "only repository allowed"},
		{"wrong source ref", func(i *sigtest.Identity) { i.Ref = "refs/heads/release-4.4" }, "refs/heads/release-4.4"},
	} {
		t.Run(c.name, func(t *testing.T) {
			id := good
			c.edit(&id)
			_, err := v.Bundle(load(t, a.Bundle(t, d, id)), d, policy)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want one containing %q", err, c.want)
			}
		})
	}
	if _, err := v.Bundle(load(t, a.Bundle(t, d, good)), digest.FromString("other"), policy); err == nil {
		t.Fatal("a signature over another digest verified")
	}
	if _, err := sigtest.New(t).Verifier(t).Bundle(load(t, a.Bundle(t, d, good)), d, policy); err == nil {
		t.Fatal("a certificate from an untrusted authority verified")
	}
}

func load(t *testing.T, raw []byte) *bundle.Bundle {
	t.Helper()
	var b bundle.Bundle
	if err := b.UnmarshalJSON(raw); err != nil {
		t.Fatal(err)
	}
	return &b
}
