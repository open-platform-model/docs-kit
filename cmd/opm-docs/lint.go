package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/docs-kit/internal/build"
	"github.com/open-platform-model/docs-kit/internal/dialect"
)

func newLintCmd() *cobra.Command {
	var bundleMode bool
	var dialectVersion int
	cmd := &cobra.Command{
		Use:   "lint DIR...",
		Short: "Lint page directories (or bundle directories) against the page dialect",
		Long: "Lint each DIR against the page dialect. A DIR is a docs/site tree, or with --bundle a bundle\n" +
			"directory (manifest.json, content/, data/). Every violation prints as <file>:<line>: <message>.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, dirs []string) error {
			if dialectVersion != dialect.Version {
				return usage("--dialect %d: this opm-docs lints dialect %d only", dialectVersion, dialect.Version)
			}
			var all []string
			for _, d := range dirs {
				var vs []string
				var err error
				if bundleMode {
					vs, err = lintBundle(d)
				} else {
					vs, err = lintDocs(d)
				}
				if err != nil {
					return err
				}
				all = append(all, vs...)
			}
			return report(cmd.OutOrStdout(), cmd.ErrOrStderr(), all, dirs)
		},
	}
	cmd.Flags().BoolVar(&bundleMode, "bundle", false, "each DIR is a bundle directory: check manifest.json and links into the bundle")
	cmd.Flags().IntVar(&dialectVersion, "dialect", dialect.Version, "page-dialect version to lint against")
	return cmd
}

func lintDocs(dir string) ([]string, error) {
	vs, err := dialect.Lint(dir, dialect.Options{Mode: dialect.Docs})
	if err != nil {
		return nil, usage("%v", err)
	}
	return strs(vs), nil
}

// lintBundle lints a bundle directory: its manifest, its listing and its
// content tree in bundle mode.
func lintBundle(dir string) ([]string, error) {
	return build.Lint(dir)
}

func strs(vs []dialect.Violation) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.String()
	}
	return out
}

func report(stdout, stderr io.Writer, violations, dirs []string) error {
	if len(violations) == 0 {
		fmt.Fprintf(stderr, "opm-docs lint: OK (%s)\n", strings.Join(dirs, " "))
		return nil
	}
	for _, v := range violations {
		fmt.Fprintln(stdout, v)
	}
	return &silentError{fmt.Sprintf("lint: %d violation(s); fix each page named above", len(violations))}
}
