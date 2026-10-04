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
		Section             string            `json:"section"`
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
		Page:    crd.Page{Path: cfg.Page, Title: cfg.Title, Description: cfg.Description, Weight: cfg.Weight},
		Section: cfg.Section,
		Order:   cfg.Order, ReconciledBy: cfg.ReconciledBy, Doc: in.Doc, Outside: in.Outside,
	})
	if err != nil {
		return Data{}, fmt.Errorf("crd %s: %w", cfg.Dir, err)
	}
	data, err := m.Encode()
	if err != nil {
		return Data{}, err
	}
	return Data{File: crd.DataFile, Schema: crd.SchemaID, Bytes: data, Sources: crdSources(m), Inputs: crdInputs(m)}, nil
}

// crdSources: the one page, standing alone, documents the first CRD; in a
// section each kind's page documents its CRD and the index none.
func crdSources(m *crd.Model) map[string]string {
	if m.Layout != crd.LayoutSection {
		return map[string]string{m.Page.Path: m.Kinds[0].File}
	}
	out := map[string]string{}
	for i := range m.Kinds {
		out[*m.Kinds[i].Page] = m.Kinds[i].File
	}
	return out
}

// crdInputs: the one page and the section index take the newest date of
// every CRD and sample read, a kind's page that of its own CRD and sample.
func crdInputs(m *crd.Model) map[string][]string {
	out := map[string][]string{m.Page.Path: m.Read}
	if m.Layout == crd.LayoutSection {
		for i := range m.Kinds {
			out[*m.Kinds[i].Page] = m.Kinds[i].Read
		}
	}
	return out
}
