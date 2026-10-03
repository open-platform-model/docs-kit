package revise

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/build"
	"github.com/open-platform-model/docs-kit/internal/gittest"
)

// TestSectionRefused checks a section bundle is refused before anything
// else: it publishes from main only, so it has no release to revise.
func TestSectionRefused(t *testing.T) {
	r := gittest.New(t, "https://github.com/open-platform-model/enhancements.git")
	r.Write(map[string]string{"docs-kit.cue": `bundles: enhancements: {
	placement: {kind: "section", root: "/enhancements/"}
	sources: [{kind: "enhancements", description: "The design record."}]
}
`})
	r.Commit("config")
	_, err := Run(context.Background(), Options{Repo: r.Dir, Project: "enhancements", Tag: "v1.0.0", Fix: strings.Repeat("a", 40)})
	var ue *build.UsageError
	if !errors.As(err, &ue) || !strings.Contains(err.Error(), "enhancements is a section bundle") {
		t.Fatalf("err = %v, want a usage error naming the section", err)
	}
}
