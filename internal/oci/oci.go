// Package oci is opm-docs' registry client, on oras-go: push a bundle,
// tag it, list tags, resolve, and fetch manifests and blobs by digest.
package oci

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/errcode"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// Options configures a client.
type Options struct {
	// Anonymous sends no credential: the pull path.
	Anonymous bool
	// PlainHTTP talks HTTP instead of HTTPS (tests).
	PlainHTTP bool
	// TagPageSize, when set, is the tag list page size (tests).
	TagPageSize int
	// SettleTimeout bounds how long a fetch of a manifest just pushed waits
	// for the registry to serve it. Default 30 seconds.
	SettleTimeout time.Duration
}

// Client opens repositories.
type Client struct {
	opts   Options
	client *auth.Client
}

// New makes a client. Unless anonymous, credentials come from the docker
// config (`docker login`), else GITHUB_TOKEN for ghcr.io.
func New(o Options) *Client {
	if o.SettleTimeout == 0 {
		o.SettleTimeout = 30 * time.Second
	}
	c := &auth.Client{Client: retry.DefaultClient, Cache: auth.NewCache()}
	c.SetUserAgent("opm-docs")
	if !o.Anonymous {
		c.Credential = credential()
	}
	return &Client{opts: o, client: c}
}

func credential() auth.CredentialFunc {
	var docker auth.CredentialFunc
	if store, err := credentials.NewStoreFromDocker(credentials.StoreOptions{}); err == nil {
		docker = credentials.Credential(store)
	}
	token := os.Getenv("GITHUB_TOKEN")
	return func(ctx context.Context, host string) (auth.Credential, error) {
		if docker != nil {
			if cred, err := docker(ctx, host); err == nil && cred != auth.EmptyCredential {
				return cred, nil
			}
		}
		if token != "" && host == "ghcr.io" {
			return auth.Credential{Username: "opm-docs", Password: token}, nil
		}
		return auth.EmptyCredential, nil
	}
}

// Repo is one repository, "ghcr.io/open-platform-model/docs/catalog-opm".
type Repo struct {
	Name   string
	r      *remote.Repository
	settle time.Duration
}

// Repository opens a repository by name.
func (c *Client) Repository(name string) (*Repo, error) {
	r, err := remote.NewRepository(name)
	if err != nil {
		return nil, fmt.Errorf("repository %s: %w", name, err)
	}
	r.Client = c.client
	r.PlainHTTP = c.opts.PlainHTTP
	if c.opts.TagPageSize > 0 {
		r.TagListPageSize = c.opts.TagPageSize
	}
	return &Repo{Name: name, r: r, settle: c.opts.SettleTimeout}, nil
}

// Graph is the repository as oras-go content storage, for referrers.
func (r *Repo) Graph() *remote.Repository { return r.r }

// Pack builds a bundle manifest in memory: OCI 1.1, the bundle artifact
// type, the empty config, one layer and the annotations. The same layer
// and annotations give the same bytes.
func Pack(ctx context.Context, artifactType, layerType string, layer []byte, annotations map[string]string) (md ocispec.Descriptor, manifest []byte, ld ocispec.Descriptor, err error) {
	mem := memory.New()
	ld = content.NewDescriptorFromBytes(layerType, layer)
	if pushErr := mem.Push(ctx, ld, bytes.NewReader(layer)); pushErr != nil {
		return md, nil, ld, pushErr
	}
	md, err = oras.PackManifest(ctx, mem, oras.PackManifestVersion1_1, artifactType, oras.PackManifestOptions{
		Layers:              []ocispec.Descriptor{ld},
		ManifestAnnotations: annotations,
	})
	if err != nil {
		return md, nil, ld, err
	}
	manifest, err = content.FetchAll(ctx, mem, md)
	return md, manifest, ld, err
}

// PushBundle uploads the empty config and the layer, then the manifest:
// by tag when tag is set, by digest otherwise. It waits until the registry
// serves the manifest by digest.
func (r *Repo) PushBundle(ctx context.Context, md ocispec.Descriptor, manifest []byte, ld ocispec.Descriptor, layer []byte, tag string) error {
	empty := ocispec.DescriptorEmptyJSON
	if err := pushBlob(ctx, r.r, empty, empty.Data); err != nil {
		return fmt.Errorf("pushing the empty config to %s: %w", r.Name, err)
	}
	if err := pushBlob(ctx, r.r, ld, layer); err != nil {
		return fmt.Errorf("pushing the layer to %s: %w", r.Name, err)
	}
	var err error
	if tag != "" {
		err = r.r.PushReference(ctx, md, bytes.NewReader(manifest), tag)
	} else {
		err = r.r.Push(ctx, md, bytes.NewReader(manifest))
	}
	if err != nil && !errors.Is(err, errdef.ErrAlreadyExists) {
		return fmt.Errorf("pushing the manifest to %s: %w", r.Name, err)
	}
	_, err = r.Settled(ctx, md.Digest.String())
	return err
}

func pushBlob(ctx context.Context, r *remote.Repository, d ocispec.Descriptor, data []byte) error {
	exists, err := r.Exists(ctx, d)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if err := r.Push(ctx, d, bytes.NewReader(data)); err != nil && !errors.Is(err, errdef.ErrAlreadyExists) {
		return err
	}
	return nil
}

// Settled resolves a reference this run just wrote, retrying a not-found
// for the settle timeout: a registry may need a moment before it serves a
// new manifest by digest.
func (r *Repo) Settled(ctx context.Context, ref string) (ocispec.Descriptor, error) {
	deadline := time.Now().Add(r.settle)
	wait := 500 * time.Millisecond
	for {
		d, err := r.r.Resolve(ctx, ref)
		if err == nil || !IsNotFound(err) || time.Now().After(deadline) {
			if err != nil {
				return d, fmt.Errorf("resolving %s@%s: %w", r.Name, ref, err)
			}
			return d, nil
		}
		select {
		case <-ctx.Done():
			return d, ctx.Err()
		case <-time.After(wait):
		}
		wait *= 2
	}
}

// IsNotFound reports a registry's not-found answer.
func IsNotFound(err error) bool {
	if errors.Is(err, errdef.ErrNotFound) {
		return true
	}
	var re *errcode.ErrorResponse
	return errors.As(err, &re) && re.StatusCode == http.StatusNotFound
}

// Resolve resolves a tag or digest to a descriptor.
func (r *Repo) Resolve(ctx context.Context, ref string) (ocispec.Descriptor, error) {
	d, err := r.r.Resolve(ctx, ref)
	if err != nil {
		return d, fmt.Errorf("resolving %s:%s: %w", r.Name, ref, err)
	}
	return d, nil
}

// Tag points tag at the manifest d, whose bytes are given, with one
// manifest PUT: no fetch, so a manifest just pushed needs no settling.
func (r *Repo) Tag(ctx context.Context, d ocispec.Descriptor, manifest []byte, tag string) error {
	if err := r.r.PushReference(ctx, d, bytes.NewReader(manifest), tag); err != nil {
		return fmt.Errorf("tagging %s:%s: %w", r.Name, tag, err)
	}
	return nil
}

// Tags lists every tag, across pages, in the registry's order.
func (r *Repo) Tags(ctx context.Context) ([]string, error) {
	var out []string
	err := r.r.Tags(ctx, "", func(tags []string) error {
		out = append(out, tags...)
		return nil
	})
	if err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing the tags of %s: %w", r.Name, err)
	}
	return out, nil
}

// BuildTags lists the tags that may name builds: every tag except the
// cosign fallback tags "sha256-<hex>".
func (r *Repo) BuildTags(ctx context.Context) ([]string, error) {
	all, err := r.Tags(ctx)
	if err != nil {
		return nil, err
	}
	out := all[:0:0]
	for _, t := range all {
		if !strings.HasPrefix(t, "sha256-") {
			out = append(out, t)
		}
	}
	return out, nil
}

// maxManifest bounds a manifest fetch.
const maxManifest = 4 << 20

// Manifest fetches and decodes an image manifest by descriptor (by
// digest, never by tag).
func (r *Repo) Manifest(ctx context.Context, d ocispec.Descriptor) (*ocispec.Manifest, []byte, error) {
	if d.Size > maxManifest {
		return nil, nil, fmt.Errorf("%s@%s: manifest of %d bytes is larger than %d", r.Name, d.Digest, d.Size, maxManifest)
	}
	raw, err := content.FetchAll(ctx, r.r, d)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching %s@%s: %w", r.Name, d.Digest, err)
	}
	var m ocispec.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, nil, fmt.Errorf("decoding %s@%s: %w", r.Name, d.Digest, err)
	}
	return &m, raw, nil
}

// Blob fetches a blob by descriptor, refusing one larger than max before a
// byte is fetched, and checks its digest.
func (r *Repo) Blob(ctx context.Context, d ocispec.Descriptor, maxSize int64) ([]byte, error) {
	if d.Size > maxSize {
		return nil, fmt.Errorf("%s@%s: blob of %d bytes is larger than the %d allowed", r.Name, d.Digest, d.Size, maxSize)
	}
	b, err := content.FetchAll(ctx, r.r, d)
	if err != nil {
		return nil, fmt.Errorf("fetching %s@%s: %w", r.Name, d.Digest, err)
	}
	return b, nil
}
