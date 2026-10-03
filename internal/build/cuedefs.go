package build

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/open-platform-model/docs-kit/internal/extract/cuedefs"
)

// cueDefinitionsExtractor reads a CUE package's exported definitions into
// data/cue-definitions.json (docs-kit C17).
type cueDefinitionsExtractor struct{}

func (cueDefinitionsExtractor) Kind() string { return "cue-definitions" }

func (cueDefinitionsExtractor) Extract(_ context.Context, in Input) (Data, error) {
	var cfg cuedefs.Config
	if err := in.Config.Decode(&cfg); err != nil {
		return Data{}, err
	}
	// A backfill warns where a normal build refuses; the warnings go where
	// the build's other diagnostics go.
	var stderr io.Writer = os.Stderr
	if in.Commands != nil && in.Commands.Stderr != nil {
		stderr = in.Commands.Stderr
	}
	m, err := cuedefs.Extract(cuedefs.Options{
		Root: in.Source, Config: cfg, Version: in.Version, Policy: in.Doc,
		Lenient: in.Outside,
		Warn:    func(s string) { fmt.Fprintln(stderr, "warning:", s) },
	})
	if err != nil {
		return Data{}, err
	}
	data, err := m.Encode()
	if err != nil {
		return Data{}, err
	}
	return Data{File: cuedefs.DataFile, Schema: cuedefs.SchemaID, Bytes: data}, nil
}
