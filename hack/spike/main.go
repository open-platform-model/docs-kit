// Command spike checks, against GHCR, the assumptions the docs bundle format
// and its signing identity rest on: that an OCI 1.1 artifact keeps its
// artifactType, empty config and annotations; that an untagged manifest can
// be pushed and fetched by digest; that a cosign v3 Sigstore bundle can be
// found as a referrer and verified with sigstore-go under the signing policy;
// and that an anonymous client can resolve, verify and fetch a bundle.
//
// It is a throwaway probe, run from the spike workflow. Nothing imports it.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const (
	artifactType   = "application/vnd.opmodel.docs.bundle.v1"
	layerType      = "application/vnd.opmodel.docs.bundle.layer.v1.tar+gzip"
	sigstoreBundle = "application/vnd.dev.sigstore.bundle.v0.3+json"
	githubIssuer   = "https://token.actions.githubusercontent.com"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: spike push|verify|pull|tags [flags]")
		os.Exit(1)
	}
	ctx := context.Background()
	var err error
	switch os.Args[1] {
	case "push":
		err = cmdPush(ctx, os.Args[2:])
	case "verify":
		err = cmdVerify(ctx, os.Args[2:])
	case "pull":
		err = cmdPull(ctx, os.Args[2:])
	case "tags":
		err = cmdTags(ctx, os.Args[2:])
	default:
		err = fmt.Errorf("unknown subcommand %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "spike:", err)
		os.Exit(2)
	}
}

// repository opens a remote repository. With anonymous set, no credential is
// sent, so GHCR hands out an anonymous token or refuses.
func repository(ref string, anonymous bool) (*remote.Repository, error) {
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", ref, err)
	}
	client := &auth.Client{Client: retry.DefaultClient, Cache: auth.NewCache()}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" && !anonymous {
		client.Credential = auth.StaticCredential(repo.Reference.Registry, auth.Credential{
			Username: "opm-docs-spike",
			Password: token,
		})
	}
	repo.Client = client
	return repo, nil
}

// layer builds a small deterministic bundle tree as a tar.gz.
func layer(version string, created time.Time) ([]byte, error) {
	files := []struct{ name, body string }{
		{"content/", ""},
		{"content/_index.md", "---\ntitle: Spike\ndescription: A spike bundle.\n---\n\nSpike " + version + ".\n"},
		{"data/", ""},
		{"manifest.json", `{"schema":"docs.opmodel.dev/bundle/v1","project":"spike","version":"` + version + `"}` + "\n"},
	}
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	tw := tar.NewWriter(gz)
	for _, f := range files {
		h := &tar.Header{Name: f.name, ModTime: created, Format: tar.FormatUSTAR}
		if strings.HasSuffix(f.name, "/") {
			h.Typeflag, h.Mode = tar.TypeDir, 0o755
		} else {
			h.Typeflag, h.Mode, h.Size = tar.TypeReg, 0o644, int64(len(f.body))
		}
		if err := tw.WriteHeader(h); err != nil {
			return nil, err
		}
		if _, err := io.WriteString(tw, f.body); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func pushBundle(ctx context.Context, repo *remote.Repository, version, revision, source string) (ocispec.Descriptor, map[string]string, error) {
	created := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	blob, err := layer(version, created)
	if err != nil {
		return ocispec.Descriptor{}, nil, err
	}
	desc := content.NewDescriptorFromBytes(layerType, blob)
	if err := repo.Push(ctx, desc, bytes.NewReader(blob)); err != nil && !isExists(err) {
		return ocispec.Descriptor{}, nil, fmt.Errorf("pushing layer: %w", err)
	}
	ann := map[string]string{
		"org.opencontainers.image.version":  version,
		"org.opencontainers.image.revision": strings.Repeat("0", 40),
		"org.opencontainers.image.source":   source,
		"org.opencontainers.image.created":  created.Format(time.RFC3339),
		"dev.opmodel.docs.project":          "spike",
		"dev.opmodel.docs.revision":         revision,
		"dev.opmodel.docs.dialect":          "1",
		"dev.opmodel.docs.tool":             "0.0.0-spike",
	}
	man, err := oras.PackManifest(ctx, repo, oras.PackManifestVersion1_1, artifactType, oras.PackManifestOptions{
		Layers:              []ocispec.Descriptor{desc},
		ManifestAnnotations: ann,
	})
	if err != nil {
		return ocispec.Descriptor{}, nil, fmt.Errorf("packing manifest: %w", err)
	}
	return man, ann, nil
}

func isExists(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already exists")
}

func cmdPush(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("push", flag.ExitOnError)
	ref := fs.String("repository", "", "repository, e.g. ghcr.io/open-platform-model/docs/spike")
	source := fs.String("source", "", "org.opencontainers.image.source")
	output := fs.String("github-output", os.Getenv("GITHUB_OUTPUT"), "file to append step outputs to")
	_ = fs.Parse(args)
	repo, err := repository(*ref, false)
	if err != nil {
		return err
	}

	tagged, ann, err := pushBundle(ctx, repo, "0.0.1", "0", *source)
	if err != nil {
		return err
	}
	fmt.Printf("pushed tagged manifest %s\n", tagged.Digest)
	for _, t := range []string{"0.0.1.0", "0.0.1", "0.0", "0"} {
		if err := repo.Tag(ctx, tagged, t); err != nil {
			return fmt.Errorf("tagging %s: %w", t, err)
		}
		fmt.Printf("tagged %s -> %s\n", t, tagged.Digest)
	}

	// The edge pattern: a manifest pushed by digest only, never tagged.
	untagged, _, err := pushBundle(ctx, repo, "edge", "0", *source)
	if err != nil {
		return err
	}
	fmt.Printf("pushed untagged manifest %s\n", untagged.Digest)
	if err := checkManifest(ctx, repo, untagged.Digest.String(), untagged, nil); err != nil {
		return fmt.Errorf("untagged manifest by digest: %w", err)
	}

	if err := listTags(ctx, repo); err != nil {
		return err
	}
	for _, r := range []string{"0.0.1.0", "0.0.1", "0.0", "0", tagged.Digest.String()} {
		if err := checkManifest(ctx, repo, r, tagged, ann); err != nil {
			return fmt.Errorf("fetching %s: %w", r, err)
		}
	}
	if *output != "" {
		f, err := os.OpenFile(*output, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		fmt.Fprintf(f, "tagged=%s\nuntagged=%s\n", tagged.Digest, untagged.Digest)
	}
	return nil
}

// checkManifest fetches a manifest by reference, prints it, and compares it
// with what was pushed: the same digest, artifactType, empty config and every
// annotation byte for byte.
func checkManifest(ctx context.Context, repo *remote.Repository, ref string, want ocispec.Descriptor, ann map[string]string) error {
	desc, rc, err := repo.FetchReference(ctx, ref)
	if err != nil {
		return err
	}
	defer rc.Close()
	raw, err := content.ReadAll(rc, desc)
	if err != nil {
		return err
	}
	fmt.Printf("--- manifest %s (%s, %s)\n%s\n", ref, desc.Digest, desc.MediaType, raw)
	if desc.Digest != want.Digest {
		return fmt.Errorf("digest %s, want %s", desc.Digest, want.Digest)
	}
	var m ocispec.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	var problems []string
	if m.ArtifactType != artifactType {
		problems = append(problems, fmt.Sprintf("artifactType %q", m.ArtifactType))
	}
	if m.Config.MediaType != ocispec.MediaTypeEmptyJSON || m.Config.Digest != ocispec.DescriptorEmptyJSON.Digest {
		problems = append(problems, fmt.Sprintf("config %s %s", m.Config.MediaType, m.Config.Digest))
	}
	if len(m.Layers) != 1 || m.Layers[0].MediaType != layerType {
		problems = append(problems, "layers")
	}
	for k, v := range ann {
		if m.Annotations[k] != v {
			problems = append(problems, fmt.Sprintf("annotation %s=%q, want %q", k, m.Annotations[k], v))
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("manifest changed: %s", strings.Join(problems, "; "))
	}
	fmt.Printf("ok: %s kept artifactType, empty config, one layer and %d annotations\n", ref, len(ann))
	return nil
}

func listTags(ctx context.Context, repo *remote.Repository) error {
	fmt.Println("--- tags")
	return repo.Tags(ctx, "", func(tags []string) error {
		for _, t := range tags {
			fmt.Println(t)
		}
		return nil
	})
}

func cmdTags(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("tags", flag.ExitOnError)
	ref := fs.String("repository", "", "repository")
	anonymous := fs.Bool("anonymous", false, "send no credential")
	_ = fs.Parse(args)
	repo, err := repository(*ref, *anonymous)
	if err != nil {
		return err
	}
	return listTags(ctx, repo)
}

type policy struct {
	san, sourceRepo, ref string
}

func (p *policy) flags(fs *flag.FlagSet, defSAN, defRepo, defRef string) {
	fs.StringVar(&p.san, "san", defSAN, "expected certificate SAN (Build Signer URI)")
	fs.StringVar(&p.sourceRepo, "repo", defRepo, "expected Source Repository URI")
	fs.StringVar(&p.ref, "ref", defRef, "expected Source Repository Ref")
}

// verifyDigest finds the Sigstore bundle referring to d and verifies it under
// the signing policy: the GitHub issuer, the SAN, the source repository and
// ref, an SCT, a transparency-log entry and an observer timestamp.
func verifyDigest(ctx context.Context, repo *remote.Repository, d digest.Digest, p policy) error {
	subject, err := repo.Resolve(ctx, d.String())
	if err != nil {
		return fmt.Errorf("resolving %s: %w", d, err)
	}
	refs, err := registry.Referrers(ctx, repo, subject, sigstoreBundle)
	if err != nil {
		return fmt.Errorf("listing referrers of %s: %w", d, err)
	}
	fmt.Printf("referrers of %s with artifactType %s: %d\n", d, sigstoreBundle, len(refs))
	if len(refs) == 0 {
		return fmt.Errorf("%s has no Sigstore bundle", d)
	}
	trusted, err := root.FetchTrustedRoot()
	if err != nil {
		return fmt.Errorf("fetching the Sigstore trusted root: %w", err)
	}
	verifier, err := verify.NewVerifier(trusted,
		verify.WithSignedCertificateTimestamps(1),
		verify.WithTransparencyLog(1),
		verify.WithObserverTimestamps(1))
	if err != nil {
		return err
	}
	san, err := verify.NewSANMatcher(p.san, "")
	if err != nil {
		return err
	}
	iss, err := verify.NewIssuerMatcher(githubIssuer, "")
	if err != nil {
		return err
	}
	id, err := verify.NewCertificateIdentity(san, iss, certificate.Extensions{
		SourceRepositoryURI: p.sourceRepo,
		SourceRepositoryRef: p.ref,
	})
	if err != nil {
		return err
	}
	digestBytes, err := hex.DecodeString(d.Encoded())
	if err != nil {
		return err
	}
	var lastErr error
	for _, r := range refs {
		b, err := fetchBundle(ctx, repo, r)
		if err != nil {
			lastErr = err
			continue
		}
		res, err := verifier.Verify(b, verify.NewPolicy(
			verify.WithArtifactDigest("sha256", digestBytes),
			verify.WithCertificateIdentity(id)))
		if err != nil {
			lastErr = fmt.Errorf("referrer %s: %w", r.Digest, err)
			fmt.Println("verify failed:", lastErr)
			continue
		}
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Printf("verified %s via referrer %s\n%s\n", d, r.Digest, out)
		return nil
	}
	return lastErr
}

func fetchBundle(ctx context.Context, repo *remote.Repository, ref ocispec.Descriptor) (*bundle.Bundle, error) {
	raw, err := content.FetchAll(ctx, repo, ref)
	if err != nil {
		return nil, fmt.Errorf("fetching referrer %s: %w", ref.Digest, err)
	}
	fmt.Printf("--- referrer %s\n%s\n", ref.Digest, raw)
	var m ocispec.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if len(m.Layers) != 1 {
		return nil, fmt.Errorf("referrer %s has %d layers", ref.Digest, len(m.Layers))
	}
	blob, err := content.FetchAll(ctx, repo, m.Layers[0])
	if err != nil {
		return nil, fmt.Errorf("fetching Sigstore bundle: %w", err)
	}
	var b bundle.Bundle
	if err := b.UnmarshalJSON(blob); err != nil {
		return nil, fmt.Errorf("decoding Sigstore bundle: %w", err)
	}
	return &b, nil
}

func cmdVerify(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	ref := fs.String("repository", "", "repository")
	dg := fs.String("digest", "", "manifest digest")
	anonymous := fs.Bool("anonymous", false, "send no credential")
	var p policy
	p.flags(fs, "", "", "")
	_ = fs.Parse(args)
	d, err := digest.Parse(*dg)
	if err != nil {
		return err
	}
	repo, err := repository(*ref, *anonymous)
	if err != nil {
		return err
	}
	return verifyDigest(ctx, repo, d, p)
}

func cmdPull(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("pull", flag.ExitOnError)
	ref := fs.String("repository", "ghcr.io/open-platform-model/docs/spike", "repository")
	tag := fs.String("tag", "0.0", "tag to resolve")
	var p policy
	p.flags(fs,
		"https://github.com/open-platform-model/docs-kit/.github/workflows/spike-sign.yml@refs/heads/spike/ghcr",
		"https://github.com/open-platform-model/docs-kit",
		"refs/heads/spike/ghcr")
	_ = fs.Parse(args)
	repo, err := repository(*ref, true)
	if err != nil {
		return err
	}
	desc, err := repo.Resolve(ctx, *tag)
	if err != nil {
		return fmt.Errorf("resolving %s anonymously: %w", *tag, err)
	}
	fmt.Printf("resolved %s to %s\n", *tag, desc.Digest)
	if err := verifyDigest(ctx, repo, desc.Digest, p); err != nil {
		return err
	}
	raw, err := content.FetchAll(ctx, repo, desc)
	if err != nil {
		return err
	}
	var m ocispec.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	if len(m.Layers) != 1 {
		return errors.New("bundle manifest has no single layer")
	}
	blob, err := content.FetchAll(ctx, repo, m.Layers[0])
	if err != nil {
		return fmt.Errorf("fetching layer by digest: %w", err)
	}
	fmt.Printf("fetched layer %s (%d bytes) anonymously\n", m.Layers[0].Digest, len(blob))
	return nil
}
