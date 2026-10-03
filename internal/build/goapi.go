package build

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/open-platform-model/docs-kit/internal/extract/goapi"
)

// goAPIExtractor reads a Go module's exported packages into
// data/go-api.json (docs-kit C20).
type goAPIExtractor struct{}

func (goAPIExtractor) Kind() string { return "go-api" }

func (goAPIExtractor) Extract(_ context.Context, in Input) (Data, error) {
	var cfg goapi.Config
	if err := in.Config.Decode(&cfg); err != nil {
		return Data{}, err
	}
	// A backfill warns where a normal build refuses; the warnings go where
	// the build's other diagnostics go.
	var stderr io.Writer = os.Stderr
	if in.Commands != nil && in.Commands.Stderr != nil {
		stderr = in.Commands.Stderr
	}
	r, err := goapi.Extract(goapi.Options{
		Source: in.Source, Config: cfg, Version: in.Version, Policy: in.Doc,
		Lenient: in.Outside,
		Warn:    func(s string) { fmt.Fprintln(stderr, "warning: go-api:", s) },
	})
	if err != nil {
		return Data{}, fmt.Errorf("go-api %s: %w", cfg.Module, err)
	}
	data, err := r.Model.Encode()
	if err != nil {
		return Data{}, err
	}
	// A package page documents its package's first file by name, and its
	// lastmod is the newest of the package's files.
	sources := map[string]string{}
	for page, files := range r.Files {
		sources[page] = files[0]
	}
	return Data{File: goapi.DataFile, Schema: goapi.SchemaID, Bytes: data, Sources: sources, Inputs: r.Files}, nil
}
