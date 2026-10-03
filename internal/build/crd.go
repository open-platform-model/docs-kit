package build

import (
	"context"
	"fmt"

	"github.com/open-platform-model/docs-kit/internal/extract/crd"
)

// crdExtractor reads controller-gen CRDs and their samples into
// data/crd.json (docs-kit C18).
type crdExtractor struct{}

func (crdExtractor) Kind() string { return "crd" }

func (crdExtractor) Extract(_ context.Context, in Input) (Data, error) {
	var cfg struct {
		Dir                 string            `json:"dir"`
		Samples             string            `json:"samples"`
		HideSamplesMatching []string          `json:"hideSamplesMatching"`
		StripLabels         map[string]string `json:"stripLabels"`
		Page                string            `json:"page"`
		Title               string            `json:"title"`
		Description         string            `json:"description"`
		Weight              *int              `json:"weight"`
		Order               []string          `json:"order"`
		ReconciledBy        map[string]string `json:"reconciledBy"`
	}
	if err := in.Config.Decode(&cfg); err != nil {
		return Data{}, err
	}
	m, err := crd.Extract(crd.Options{
		Root: in.Source, Dir: cfg.Dir, Samples: cfg.Samples,
		HideSamplesMatching: cfg.HideSamplesMatching, StripLabels: cfg.StripLabels,
		Page:  crd.Page{Path: cfg.Page, Title: cfg.Title, Description: cfg.Description, Weight: cfg.Weight},
		Order: cfg.Order, ReconciledBy: cfg.ReconciledBy, Doc: in.Doc, Outside: in.Outside,
	})
	if err != nil {
		return Data{}, fmt.Errorf("crd %s: %w", cfg.Dir, err)
	}
	data, err := m.Encode()
	if err != nil {
		return Data{}, err
	}
	// A page without an authored one documents the first CRD, and its
	// lastmod is the newest of every CRD and sample read.
	return Data{
		File: crd.DataFile, Schema: crd.SchemaID, Bytes: data,
		Sources: map[string]string{cfg.Page: m.Kinds[0].File},
		Inputs:  map[string][]string{cfg.Page: m.Read},
	}, nil
}
