package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/docs-kit/internal/oci"
	"github.com/open-platform-model/docs-kit/internal/publish"
	"github.com/open-platform-model/docs-kit/internal/revise"
	"github.com/open-platform-model/docs-kit/internal/verify"
	"github.com/open-platform-model/docs-kit/internal/version"
)

func newReviseCmd() *cobra.Command {
	var o revise.Options
	var trustedRoot string
	cmd := &cobra.Command{
		Use:   "revise",
		Short: "Build the next docs revision of a published release with a documentation fix from main",
		Long: "Check that --fix is a single-parent commit on origin/main, apply the fixes of the newest published\n" +
			"revision of --tag and then --fix to the release tree in a temporary worktree, refuse anything but\n" +
			"a documentation change (Markdown, and comments in .cue and .go files), and build the next revision\n" +
			"into <out>/<project>/. It pushes nothing: push, sign and promote it as a release.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			for _, f := range []struct{ name, value string }{{"--project", o.Project}, {"--tag", o.Tag}, {"--fix", o.Fix}} {
				if f.value == "" {
					return usage("%s is required: revise --project <project> --tag <release tag> --fix <commit on main>", f.name)
				}
			}
			o.Repo, o.Tool = ".", version.Version
			o.Client = oci.New(oci.Options{})
			// The fixes the newest revision lists are trusted only after its
			// signature verifies, under the policy promote applies.
			repo := os.Getenv("GITHUB_REPOSITORY")
			verifier := trust(trustedRoot, false, nil)
			o.Verifier = func() (*verify.Verifier, error) {
				if repo == "" {
					return nil, errors.New("GITHUB_REPOSITORY is unset: revise checks the published revision was signed for the repository it runs for, so it runs in GitHub Actions")
				}
				return verifier()
			}
			o.Policy = verify.Policy{Issuer: githubIssuer, Workflow: publishWorkflow, Refs: []string{publishRefs},
				Repository: "https://github.com/" + repo, Ref: mainRef}
			res, err := revise.Run(cmd.Context(), o)
			if err != nil {
				return buildError(cmd.OutOrStdout(), err)
			}
			if res.Rebuilt {
				fmt.Fprintf(cmd.ErrOrStderr(), "opm-docs revise: %s.%d already ends with this fix and is not promoted; built it again\n", res.Version, res.Revision)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "opm-docs revise: %s %s.%d (%d pages, %d fix(es))\n", res.Dir, res.Version, res.Revision, res.Pages, len(res.Patches))
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.Project, "project", "", "the project (required)")
	f.StringVar(&o.Tag, "tag", "", "the release's git tag, such as opm-v4.4.5 (required)")
	f.StringVar(&o.Fix, "fix", "", "the 40-hex commit on main whose documentation change to apply (required)")
	f.StringVar(&o.Out, "out", "out", "the output directory")
	f.StringVar(&o.Registry, "registry", publish.DefaultRegistry, "the registry prefix; the repository is <registry>/<project>")
	f.StringVar(&o.Config, "config", "", "the config file (default: docs-kit.cue in the release tree, else in the current directory)")
	f.StringVar(&trustedRoot, "trusted-root", "", "a Sigstore trusted_root.json instead of the TUF-fetched one")
	_ = f.MarkHidden("trusted-root")
	return cmd
}
