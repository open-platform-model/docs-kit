package build

import (
	"context"
	"fmt"

	"github.com/open-platform-model/docs-kit/internal/extract/enhancements"
	"github.com/open-platform-model/docs-kit/internal/gitsrc"
	"github.com/open-platform-model/docs-kit/internal/render"
)

// enhancementsExtractor reads the enhancements repository into
// data/enhancements.json and the section's pages (docs-kit C21).
type enhancementsExtractor struct{}

func (enhancementsExtractor) Kind() string { return "enhancements" }

func (enhancementsExtractor) Extract(ctx context.Context, in Input) (Data, error) {
	var cfg struct {
		Dir         string `json:"dir"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := in.Config.Decode(&cfg); err != nil {
		return Data{}, err
	}
	// Relative links resolve against the commit built, not the work tree.
	paths, err := gitsrc.Repo{Dir: in.Source}.Tree(ctx, in.Commit)
	if err != nil {
		return Data{}, err
	}
	res, err := enhancements.Extract(enhancements.Options{
		Root: in.Source, Dir: cfg.Dir, Repo: in.Repo, Commit: in.Commit, Paths: paths,
		Title: cfg.Title, Description: cfg.Description,
	})
	if err != nil {
		return Data{}, fmt.Errorf("enhancements %s: %w", cfg.Dir, err)
	}
	data, err := res.Model.Encode()
	if err != nil {
		return Data{}, err
	}
	pages, err := render.Enhancements(res.Pages, in.Target)
	if err != nil {
		return Data{}, err
	}
	sources := make(map[string]string, len(res.Pages))
	for _, p := range res.Pages {
		sources[p.Path] = p.Source
	}
	return Data{File: enhancements.DataFile, Schema: enhancements.SchemaID, Bytes: data, Sources: sources, Pages: pages}, nil
}
