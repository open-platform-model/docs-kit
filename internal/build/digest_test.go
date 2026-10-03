package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-platform-model/docs-kit/internal/bundle"
)

// catalogDigest is the packed digest of the fixture catalog's release
// bundle, recorded before extractors and renderers were registered by kind
// and schema. A tab bundle must stay byte-identical across that refactor
// and every change after it that leaves catalog output alone; a change
// that alters catalog output on purpose updates it and says so.
const catalogDigest = "sha256:b975b0c14578f859a8fdb33b05a21da1e7d61f8048986f08ae84898c38f0efc9"

func TestCatalogBundleDigestUnchanged(t *testing.T) {
	r := fixtureRepo(t)
	out := filepath.Join(t.TempDir(), "out")
	if _, err := Run(context.Background(), Options{Source: r.Dir, Out: out, Release: "demo-v1.2.3", Tool: "0.1.0"}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(out, "catalog-demo")
	m, err := bundle.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	created, err := time.Parse(time.RFC3339, m.Created)
	if err != nil {
		t.Fatal(err)
	}
	layer, err := bundle.Pack(dir, created)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(layer)
	if got := "sha256:" + hex.EncodeToString(sum[:]); got != catalogDigest {
		t.Fatalf("the catalog bundle packs to %s, want %s", got, catalogDigest)
	}
}
