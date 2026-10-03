package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/docs-kit/internal/serve"
	"github.com/open-platform-model/docs-kit/internal/version"
)

func newServeCmd() *cobra.Command {
	var o serve.Options
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Build this repository's bundles and preview them on a local Hugo",
		Long: "Build the configured bundles as edge bundles (a dirty work tree allowed) and serve them on\n" +
			"http://127.0.0.1:<port>/ with the host's hugo, on a minimal site that shows page content, not the\n" +
			"site's design. A tab bundle serves at <root>edge/, a docs bundle at /docs/. A change under a\n" +
			"bundle's sources rebuilds it; a rebuild that fails is printed and the last good build stays.\n\n" +
			"With --site <dir>, build once and preview through an opmodel.dev checkout instead: its\n" +
			"`task bundles:pull` and `task serve`, with OPM_BUNDLES_LOCAL naming each built tree.",
		Example: "  opm-docs serve\n  opm-docs serve --site ../opmodel.dev --site-version v1.0",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if o.Port < 1 || o.Port > 65535 {
				return usage("--port %d: a port is 1 to 65535", o.Port)
			}
			if o.SiteVersion != "" && o.Site == "" {
				return usage("--site-version %s needs --site: the skeleton serves a docs bundle at /docs/ whatever its site version", o.SiteVersion)
			}
			o.Build.Tool, o.Build.Stderr = version.Version, cmd.ErrOrStderr()
			o.Stdin, o.Stdout, o.Stderr = cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr()
			var err error
			if o.Site != "" {
				err = serve.RunSite(cmd.Context(), o)
			} else {
				err = serve.Run(cmd.Context(), o, func(w io.Writer, err error) { printBuildError(w, err) })
			}
			return serveError(cmd.OutOrStdout(), err)
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.Build.Config, "config", "", "the config file (default: docs-kit.cue in --source, else in the current directory)")
	f.StringSliceVar(&o.Build.Projects, "project", nil, "serve only this project (repeatable; default every project)")
	f.StringVar(&o.Build.Source, "source", ".", "the source tree")
	f.IntVar(&o.Port, "port", 1313, "the preview's port on 127.0.0.1 (ignored with --site)")
	f.StringVar(&o.Site, "site", "", "an opmodel.dev checkout: preview through its own task bundles:pull and task serve")
	f.StringVar(&o.SiteVersion, "site-version", "", "with --site: the site version a docs bundle previews in, such as v1.0")
	return cmd
}

// serveError maps serve's errors to exit codes: its usage errors and a
// first build's config errors exit 1, a failed first build 2 (with its
// lint violations printed), anything else 2.
func serveError(stdout io.Writer, err error) error {
	if err == nil {
		return nil
	}
	var ue *serve.UsageError
	if errors.As(err, &ue) {
		return asUsage(err)
	}
	var bf *serve.BuildError
	if errors.As(err, &bf) {
		return buildError(stdout, bf.Err)
	}
	return buildError(stdout, err)
}

// printBuildError prints a rebuild's failure while serve watches: lint
// violations one per line, as build prints them, else the error.
func printBuildError(w io.Writer, err error) {
	if e := buildError(w, err); e != nil {
		var se *silentError
		if errors.As(e, &se) {
			fmt.Fprintln(w, se.msg)
			return
		}
		fmt.Fprintln(w, e)
	}
}
