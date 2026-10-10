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
		Changes     string `json:"changes"`
		Specs       string `json:"specs"`
		Legacy      string `json:"legacy"`
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
	repo := gitsrc.Repo{Dir: in.Source}
	shallow := -1 // asked at the first openspec entry: 1 shallow, 0 not
	lastCommit := func(path string) (string, error) {
		if shallow < 0 {
			sh, err := repo.Shallow(ctx)
			if err != nil {
				return "", err
			}
			shallow = 0
			if sh {
				shallow = 1
			}
		}
		if shallow == 1 {
			return "", fmt.Errorf("%s is a shallow clone, so the date of the last commit that touched this entry is unknown; check out the full history (actions/checkout with fetch-depth: 0)", in.Source)
		}
		return repo.LastCommitDate(ctx, in.Commit, path)
	}
	res, err := enhancements.Extract(enhancements.Options{
		Root: in.Source, Dir: cfg.Dir, Repo: in.Repo, Commit: in.Commit, Paths: paths,
		Title: cfg.Title, Description: cfg.Description, Changes: cfg.Changes, Specs: cfg.Specs, Legacy: cfg.Legacy, LastCommit: lastCommit,
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
	inputs := map[string][]string{}
	for i := range res.Pages {
		p := &res.Pages[i]
		if p.Source != "" {
			sources[p.Path] = p.Source
		}
		if len(p.Inputs) > 0 {
			inputs[p.Path] = p.Inputs
		}
	}
	return Data{File: enhancements.DataFile, Schema: enhancements.SchemaID, Bytes: data, Sources: sources, Inputs: inputs, Pages: pages}, nil
}
