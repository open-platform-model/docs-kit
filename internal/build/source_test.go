package build

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/gittest"
	"github.com/open-platform-model/docs-kit/schema"
)

// The schema's #Source union and the registry name the same kinds.
func TestRegistryMatchesSchema(t *testing.T) {
	kinds, err := schema.SourceKinds()
	if err != nil {
		t.Fatal(err)
	}
	registered := make([]string, 0, len(extractors)+1)
	for _, e := range extractors {
		registered = append(registered, e.Kind())
	}
	registered = append(registered, markdownKind)
	slices.Sort(kinds)
	slices.Sort(registered)
	if !slices.Equal(kinds, registered) {
		t.Fatalf("#Source admits %v, the registry builds %v", kinds, registered)
	}
}

func TestOneSourcePerExtractorKind(t *testing.T) {
	r := fixtureRepo(t)
	r.Write(map[string]string{"docs-kit.cue": strings.Replace(fixtureConfig,
		`{kind: "markdown", dir: "docs/catalogs/demo"},`,
		`{kind: "markdown", dir: "docs/catalogs/demo"},
			{kind: "cue-catalog", module: "./demo"},`, 1)})
	_, err := Run(context.Background(), Options{Source: r.Dir, Out: filepath.Join(t.TempDir(), "out"), Tool: "0.1.0"})
	var ue *UsageError
	if !errors.As(err, &ue) || !strings.Contains(err.Error(), "sources[0] and sources[2] are both cue-catalog") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnknownKindNamesTheRegistered(t *testing.T) {
	r := gittest.New(t, "https://github.com/example/demo.git")
	r.Write(map[string]string{"docs-kit.cue": strings.Replace(fixtureConfig, `kind: "markdown"`, `kind: "javadoc"`, 1)})
	r.Commit("config")
	_, err := Run(context.Background(), Options{Source: r.Dir, Out: filepath.Join(t.TempDir(), "out"), Tool: "0.1.0"})
	var ue *UsageError
	if !errors.As(err, &ue) || !strings.Contains(err.Error(), `"javadoc"`) || !strings.Contains(err.Error(), "cue-catalog, markdown") {
		t.Fatalf("err = %v", err)
	}
}
