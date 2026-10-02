package oci

import (
	"bytes"
	"context"
	"slices"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/ocitest"
)

func TestPushTagList(t *testing.T) {
	ctx := context.Background()
	host := ocitest.Registry(t)
	c := New(Options{Anonymous: true, PlainHTTP: true, TagPageSize: 2})
	r, err := c.Repository(host + "/docs/demo")
	if err != nil {
		t.Fatal(err)
	}
	if tags, err := r.Tags(ctx); err != nil || len(tags) != 0 {
		t.Fatalf("tags of an empty repository: %v %v", tags, err)
	}
	ann := map[string]string{"org.opencontainers.image.created": "2026-09-30T12:00:00Z", "dev.opmodel.docs.project": "demo"}
	md, raw, ld, err := Pack(ctx, "application/vnd.opmodel.docs.bundle.v1", "application/vnd.opmodel.docs.bundle.layer.v1.tar+gzip", []byte("layer"), ann)
	if err != nil {
		t.Fatal(err)
	}
	md2, raw2, _, _ := Pack(ctx, "application/vnd.opmodel.docs.bundle.v1", "application/vnd.opmodel.docs.bundle.layer.v1.tar+gzip", []byte("layer"), ann)
	if md2.Digest != md.Digest || !bytes.Equal(raw2, raw) {
		t.Fatal("packing is not deterministic")
	}
	if err := r.PushBundle(ctx, md, raw, ld, []byte("layer"), "1.2.3.0"); err != nil {
		t.Fatal(err)
	}
	// A re-push of the same bundle is a no-op.
	if err := r.PushBundle(ctx, md, raw, ld, []byte("layer"), "1.2.3.0"); err != nil {
		t.Fatal(err)
	}
	for _, tag := range []string{"1.2.3", "1.2", "1", "sha256-" + md.Digest.Encoded()} {
		if err := r.Tag(ctx, md, raw, tag); err != nil {
			t.Fatal(err)
		}
	}
	tags, err := r.Tags(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 5 {
		t.Fatalf("tags across pages: %v", tags)
	}
	builds, _ := r.BuildTags(ctx)
	slices.Sort(builds)
	if !slices.Equal(builds, []string{"1", "1.2", "1.2.3", "1.2.3.0"}) {
		t.Fatalf("build tags %v", builds)
	}
	fetchAndUntagged(t, r, md.Digest.String(), ann)
}

func fetchAndUntagged(t *testing.T, r *Repo, want string, ann map[string]string) {
	t.Helper()
	ctx := context.Background()
	d, err := r.Resolve(ctx, "1.2")
	if err != nil || d.Digest.String() != want {
		t.Fatalf("resolve: %v %v", d, err)
	}
	m, _, err := r.Manifest(ctx, d)
	if err != nil || m.ArtifactType != "application/vnd.opmodel.docs.bundle.v1" || m.Annotations["dev.opmodel.docs.project"] != "demo" {
		t.Fatalf("manifest %+v %v", m, err)
	}
	if b, err := r.Blob(ctx, m.Layers[0], 100); err != nil || string(b) != "layer" {
		t.Fatalf("blob %q %v", b, err)
	}
	if _, err := r.Blob(ctx, m.Layers[0], 2); err == nil {
		t.Fatal("an oversized blob was fetched")
	}
	// An untagged push by digest.
	md3, raw3, ld3, _ := Pack(ctx, "application/vnd.opmodel.docs.bundle.v1", "application/vnd.opmodel.docs.bundle.layer.v1.tar+gzip", []byte("edge"), ann)
	if err := r.PushBundle(ctx, md3, raw3, ld3, []byte("edge"), ""); err != nil {
		t.Fatal(err)
	}
	if tags, _ := r.Tags(ctx); len(tags) != 5 {
		t.Fatalf("an untagged push wrote a tag: %v", tags)
	}
	if _, err := r.Resolve(ctx, md3.Digest.String()); err != nil {
		t.Fatal(err)
	}
}
