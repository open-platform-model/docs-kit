package build

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/extract/cobra"
)

// cobraExtractor runs the repository's cobradump program (C14) and turns
// its command-tree dump into data/cobra.json (C19).
type cobraExtractor struct{}

func (cobraExtractor) Kind() string { return "cobra" }

func (cobraExtractor) Extract(ctx context.Context, in Input) (Data, error) {
	var cfg struct {
		Command     []string `json:"command"`
		Section     string   `json:"section"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Weight      int      `json:"weight"`
		Citations   string   `json:"citations"`
	}
	if err := in.Config.Decode(&cfg); err != nil {
		return Data{}, err
	}
	out, err := in.Commands.Run(ctx, cfg.Command, cobra.DumpSchema)
	if err != nil {
		return Data{}, err
	}
	m, err := cobra.FromDump(out, cobra.Options{
		Section: cfg.Section, Title: cfg.Title, Description: cfg.Description, Weight: cfg.Weight,
		Citations: string(policy(cfg.Citations)),
	})
	if err != nil {
		return Data{}, fmt.Errorf("%s: command `%s`: %w", in.Commands.Project, strings.Join(cfg.Command, " "), err)
	}
	data, err := m.Encode()
	if err != nil {
		return Data{}, err
	}
	return Data{File: cobra.DataFile, Schema: cobra.SchemaID, Bytes: data}, nil
}
