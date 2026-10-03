package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/docs-kit/internal/build"
	"github.com/open-platform-model/docs-kit/internal/version"
)

func newBuildCmd() *cobra.Command {
	var o build.Options
	var edge bool
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the configured bundles into out/<project>/",
		Long: "Read docs-kit.cue, run each source, render and lint, and write each bundle tree to\n" +
			"<out>/<project>/. Without --release it builds the edge bundle of the commit checked out.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if edge && o.Release != "" {
				return usage("--edge and --release %s: choose one", o.Release)
			}
			o.Tool = version.Version
			return runBuild(cmd, o)
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.Config, "config", "", "the config file (default: docs-kit.cue in --source, else in the current directory)")
	f.StringSliceVar(&o.Projects, "project", nil, "build only this project (repeatable; default every project)")
	f.StringVar(&o.Out, "out", "out", "the output directory")
	f.StringVar(&o.Source, "source", ".", "the source tree: every source and all git history resolve against it")
	f.BoolVar(&edge, "edge", false, "build the edge bundle of the commit checked out (the default)")
	f.StringVar(&o.Release, "release", "", "build the release of this git tag, such as opm-v4.4.5")
	// A docs revision's build, run by revise on the patched release tree.
	f.IntVar(&o.Revision, "revision", 0, "build this docs revision of --release (revise sets it)")
	f.StringSliceVar(&o.Patches, "patches", nil, "the fix commits applied to the release tree, oldest first (revise sets it)")
	_ = f.MarkHidden("revision")
	_ = f.MarkHidden("patches")
	return cmd
}

func runBuild(cmd *cobra.Command, o build.Options) error {
	results, err := build.Run(cmd.Context(), o)
	if err != nil {
		return buildError(cmd.OutOrStdout(), err)
	}
	for _, r := range results {
		fmt.Fprintf(cmd.ErrOrStderr(), "opm-docs build: %s (%d pages)\n", r.Dir, r.Pages)
	}
	return nil
}

// buildError prints lint violations and maps the error to its exit code.
func buildError(stdout io.Writer, err error) error {
	var le *build.LintError
	if errors.As(err, &le) {
		for _, v := range le.Violations {
			fmt.Fprintln(stdout, v)
		}
		return &silentError{le.Error() + "; fix each page named above"}
	}
	var ue *build.UsageError
	if errors.As(err, &ue) {
		return asUsage(err)
	}
	return err
}

func newCheckCmd() *cobra.Command {
	var o build.Options
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Build every configured bundle into a temporary directory and fail on any problem",
		Long:  "The pull-request gate: build and lint each bundle as build does, writing nothing in the work tree.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			tmp, err := os.MkdirTemp("", "opm-docs-check-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tmp)
			o.Out, o.Source, o.Tool = tmp, ".", version.Version
			results, err := build.Run(cmd.Context(), o)
			if err != nil {
				return buildError(cmd.OutOrStdout(), err)
			}
			for _, r := range results {
				fmt.Fprintf(cmd.ErrOrStderr(), "opm-docs check: %s OK (%d pages)\n", r.Project, r.Pages)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&o.Config, "config", "", "the config file (default docs-kit.cue)")
	cmd.Flags().StringSliceVar(&o.Projects, "project", nil, "check only this project (repeatable; default every project)")
	return cmd
}
