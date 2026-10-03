package publish

import (
	"context"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
)

// TestPushRefusesAPreview refuses a bundle built where the repository had
// no GitHub origin: its source links name a made-up repository.
func TestPushRefusesAPreview(t *testing.T) {
	e := newEnv(t)
	dir := e.bundleDir("4.4.5", 0, "")
	m, _ := bundle.Read(dir)
	m.Source.Repo = "local/catalog_opm"
	_ = bundle.Write(dir, m)
	_, err := Push(context.Background(), PushOptions{Dir: dir, Registry: e.registry, Client: e.client})
	if err == nil || !strings.Contains(err.Error(), "no GitHub origin (source.repo local/catalog_opm), so it is a preview") {
		t.Fatalf("err = %v", err)
	}
	if tags, _ := e.repo.Tags(context.Background()); len(tags) != 0 {
		t.Fatalf("a refused push wrote %v", tags)
	}
}
