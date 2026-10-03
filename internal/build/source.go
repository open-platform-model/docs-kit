package build

import (
	"context"
	"fmt"
	"regexp"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/docs-kit/internal/command"
	"github.com/open-platform-model/docs-kit/internal/doctext"
	"github.com/open-platform-model/docs-kit/internal/extract/cuecatalog"
	"github.com/open-platform-model/docs-kit/internal/render"
)

// An Extractor turns one configured source into one data file, which the
// renderer registered for the file's schema turns into pages.
type Extractor interface {
	Kind() string // the docs-kit.cue source kind, "cue-catalog"
	Extract(ctx context.Context, in Input) (Data, error)
}

// Input is what an extractor may read: the source tree, its own config
// entry (decoded by the extractor), the build identity, the command runner
// and the citation policy.
type Input struct {
	Source   string          // the --source tree
	Config   cue.Value       // this source's entry in docs-kit.cue, validated
	Version  string          // "4.4.5" or "edge"
	Release  string          // the release tag built, "opm-v4.4.5"; "" for edge
	Outside  bool            // the config came from outside the source tree (a backfill)
	Commands *command.Runner // runs repository commands (docs-kit C14)
	Doc      doctext.Policy  // what the source's citations become
}

// policy is a source's citation policy, Strip unless it says link.
func policy(citations string) doctext.Policy {
	if doctext.Policy(citations) == doctext.Link {
		return doctext.Link
	}
	return doctext.Strip
}

// Edge reports an edge build.
func (in Input) Edge() bool { return in.Release == "" }

// Data is one file under data/.
type Data struct {
	File   string // "catalog.json"
	Schema string // "docs.opmodel.dev/data/cue-catalog/v1"
	Bytes  []byte
	// Sources maps a page path the renderer will write to the repository
	// file it documents, for manifest.json pages[].source and lastmod.
	Sources map[string]string
	// Inputs maps a page path to every repository file it was built from,
	// when more than its Sources file; a standalone page's lastmod is then
	// the newest of their dates.
	Inputs map[string][]string
}

// markdownKind is the source kind that copies authored pages. It is no
// extractor: it writes no data file, and its pages may complete a
// renderer's completable page.
const markdownKind = "markdown"

// extractors is every extractor this opm-docs carries, in the order
// #Source lists their kinds. An extractor change adds its kind here and to
// #Source in schema/config.cue.
var extractors = []Extractor{
	cueCatalogExtractor{},
	crdExtractor{},
}

// Paths an extractor or renderer may write, checked before anything is
// written: the manifest's #Page.path and #DataFile.path (C3), which admit
// no "..", no absolute path and no upper case.
var (
	rePagePath = regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*/)*(_index|[a-z0-9]+(-[a-z0-9]+)*)\.md$`)
	reDataPath = regexp.MustCompile(`^[a-z0-9-]+\.json$`)
)

// rendererFor returns the renderer of a data schema; tests replace it.
var rendererFor = render.For

// extractorFor returns the extractor of a source kind.
func extractorFor(kind string) (Extractor, bool) {
	for _, e := range extractors {
		if e.Kind() == kind {
			return e, true
		}
	}
	return nil, false
}

// cueCatalogExtractor reads a CUE catalog module into data/catalog.json.
type cueCatalogExtractor struct{}

func (cueCatalogExtractor) Kind() string { return "cue-catalog" }

func (cueCatalogExtractor) Extract(_ context.Context, in Input) (Data, error) {
	var cfg struct {
		Module string `json:"module"`
	}
	if err := in.Config.Decode(&cfg); err != nil {
		return Data{}, err
	}
	model, err := cuecatalog.Extract(cuecatalog.Options{Root: in.Source, Module: cfg.Module})
	if err != nil {
		return Data{}, fmt.Errorf("cue-catalog %s: %w", cfg.Module, err)
	}
	// A release bundle is tagged with the tag's version; the catalog must
	// declare the same one, or its pages would contradict their tag.
	if !in.Edge() && model.Version != in.Version {
		return Data{}, fmt.Errorf("--release %s names version %s, but the catalog %s declares metadata.version %q; build the tag of the catalog's version, or advance the catalog's version on the release commit", in.Release, in.Version, cfg.Module, model.Version)
	}
	data, err := model.Encode()
	if err != nil {
		return Data{}, err
	}
	sources := map[string]string{}
	for i := range model.Members {
		sources[model.Members[i].Page+".md"] = model.Members[i].File
	}
	return Data{File: cuecatalog.DataFile, Schema: cuecatalog.SchemaID, Bytes: data, Sources: sources}, nil
}
