package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/docs-kit/internal/oci"
	"github.com/open-platform-model/docs-kit/internal/publish"
	"github.com/open-platform-model/docs-kit/internal/pull"
	"github.com/open-platform-model/docs-kit/internal/verify"
	"github.com/open-platform-model/docs-kit/internal/version"
)

// The signer every bundle must carry (docs/contracts.md C9).
const (
	githubIssuer    = "https://token.actions.githubusercontent.com"
	publishWorkflow = "https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml"
	publishRefs     = "refs/tags/v[0-9]*"
	mainRef         = "refs/heads/main"
)

func newPushCmd() *cobra.Command {
	var o publish.PushOptions
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Pack a built bundle and push it under its immutable full tag (by digest for edge)",
		Long: "Validate the bundle in --dir, pack it deterministically and push it: a release build under\n" +
			"<version>.<revision>, which is never overwritten; an edge build by digest only. Prints\n" +
			"{\"digest\": ..., \"tag\": ...} on stdout. The moving tags are promote's.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if o.Dir == "" {
				return usage("--dir is required: the bundle directory build wrote, such as out/catalog-opm")
			}
			o.Client = oci.New(oci.Options{})
			res, err := publish.Push(cmd.Context(), o)
			if err != nil {
				return buildError(cmd.OutOrStdout(), err)
			}
			if res.Existing {
				fmt.Fprintf(cmd.ErrOrStderr(), "opm-docs push: %s already names %s; nothing written\n", res.Tag, res.Digest)
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(res)
		},
	}
	cmd.Flags().StringVar(&o.Dir, "dir", "", "the bundle directory to push (required)")
	cmd.Flags().StringVar(&o.Registry, "registry", publish.DefaultRegistry, "the registry prefix; the repository is <registry>/<project>")
	return cmd
}

// trust makes the signature verifier from the Sigstore trusted root.
func trust(file string, offline bool, warn func(string)) func() (*verify.Verifier, error) {
	return func() (*verify.Verifier, error) {
		tm, err := verify.TrustedRoot(verify.TrustOptions{
			File: file, CacheDir: filepath.Join(pull.DefaultCacheDir(), "sigstore"), Offline: offline, Warn: warn,
		})
		if err != nil {
			return nil, err
		}
		return verify.New(tm)
	}
}

func newPromoteCmd() *cobra.Command {
	var o publish.PromoteOptions
	var trustedRoot string
	cmd := &cobra.Command{
		Use:   "promote",
		Short: "Verify a pushed digest's signature, then move the moving tags of its line to it",
		Long: "Verify that --digest was signed by docs-kit's publish workflow for this repository's main\n" +
			"(GITHUB_REPOSITORY), then move <version>, <MAJOR.MINOR> and <MAJOR> (or edge) to it where it is\n" +
			"the newest build of that line.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if o.Project == "" || o.Digest == "" {
				return usage("--project and --digest are required")
			}
			repo := os.Getenv("GITHUB_REPOSITORY")
			if repo == "" {
				return usage("GITHUB_REPOSITORY is unset: promote checks the signature names the repository it runs for, so it runs in GitHub Actions")
			}
			v, err := trust(trustedRoot, false, nil)()
			if err != nil {
				return err
			}
			o.Client, o.Verifier = oci.New(oci.Options{}), v
			o.Policy = verify.Policy{Issuer: githubIssuer, Workflow: publishWorkflow, Refs: []string{publishRefs},
				Repository: "https://github.com/" + repo, Ref: mainRef}
			o.Log = func(s string) { fmt.Fprintln(cmd.ErrOrStderr(), "opm-docs promote:", s) }
			res, err := publish.Promote(cmd.Context(), o)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "opm-docs promote: %s moved %s; left %s\n", res.Build, list(res.Moved), list(res.Skipped))
			return nil
		},
	}
	cmd.Flags().StringVar(&o.Project, "project", "", "the project (required)")
	cmd.Flags().StringVar(&o.Digest, "digest", "", "the pushed manifest digest, sha256:... (required)")
	cmd.Flags().StringVar(&o.Registry, "registry", publish.DefaultRegistry, "the registry prefix")
	cmd.Flags().StringVar(&trustedRoot, "trusted-root", "", "a Sigstore trusted_root.json instead of the TUF-fetched one")
	_ = cmd.Flags().MarkHidden("trusted-root")
	return cmd
}

func list(xs []string) string {
	if len(xs) == 0 {
		return "none"
	}
	return strings.Join(xs, ", ")
}

func newPullCmd() *cobra.Command {
	var o pull.Options
	var locals []string
	var trustedRoot string
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Resolve, verify, unpack and lint the bundles the site shows, and write the lock",
		Long: "Read the pull config, list each tab's tags, verify each bundle's signature before fetching its\n" +
			"layer, unpack it under <out>/<project>/<segment>/, lint it in bundle mode, write each tab's version\n" +
			"history to <out>/<project>/history.json, and write the lock. Each site version's docs bundles (the\n" +
			"anchor, the projects it pins, the projects pulled by their own tag) unpack under\n" +
			"<out>/_versions/<site-version>/<project>/, replaced whole once every one of them passed. A section\n" +
			"(the enhancements) is pulled from its edge tag only, into <out>/<project>/edge/.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			for _, s := range locals {
				l, err := pull.ParseLocal(s)
				if err != nil {
					return usage("%v", err)
				}
				o.Locals = append(o.Locals, l)
			}
			if o.Lock == "" {
				o.Lock = filepath.Join(o.Out, "lock.json")
			}
			warn := func(s string) { fmt.Fprintln(cmd.ErrOrStderr(), "opm-docs pull: warning:", s) }
			o.Warn = warn
			o.Cache = pull.Cache{Dir: pull.DefaultCacheDir()}
			o.Client = oci.New(oci.Options{Anonymous: true})
			o.Verifier = trust(trustedRoot, o.Offline, warn)
			o.Tool = version.Version
			l, err := pull.Run(cmd.Context(), o)
			if err != nil {
				return pullError(cmd, err)
			}
			for i := range l.Bundles {
				e := &l.Bundles[i]
				fmt.Fprintf(cmd.ErrOrStderr(), "opm-docs pull: %s %s %s %s\n", e.Project, e.Segment, e.Version, orLocal(e.Digest))
			}
			for i := range l.Docs {
				e := &l.Docs[i]
				fmt.Fprintf(cmd.ErrOrStderr(), "opm-docs pull: %s %s (%s) %s %s\n", e.Site, e.Project, e.Role, e.Version, orLocal(e.Digest))
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.Config, "config", "bundles.cue", "the pull config")
	f.StringVar(&o.Out, "out", ".bundles", "where bundles unpack, <out>/<project>/<segment>/")
	f.StringVar(&o.Lock, "lock", "", "the lock to write (default <out>/lock.json)")
	f.StringVar(&o.Frozen, "frozen", "", "pull exactly the digests this lock names")
	f.BoolVar(&o.Offline, "offline", false, "fetch nothing: with --frozen, use only the cache")
	f.StringArrayVar(&locals, "local", nil, "take <project>@<segment> (a tab's MAJOR.MINOR or edge, a section's edge, or a site version v<MAJOR>.<MINOR>) from a local bundle tree: <project>@<segment>=<dir> (repeatable)")
	f.StringVar(&trustedRoot, "trusted-root", "", "a Sigstore trusted_root.json instead of the TUF-fetched one")
	_ = f.MarkHidden("trusted-root")
	return cmd
}

func orLocal(d string) string {
	if d == "" {
		return "(local)"
	}
	return d
}

func pullError(cmd *cobra.Command, err error) error {
	var le *pull.LintError
	if errors.As(err, &le) {
		for _, v := range le.Violations {
			fmt.Fprintln(cmd.OutOrStdout(), v)
		}
		return &silentError{le.Error()}
	}
	if pull.IsUsage(err) {
		return asUsage(err)
	}
	return buildError(cmd.OutOrStdout(), err)
}
