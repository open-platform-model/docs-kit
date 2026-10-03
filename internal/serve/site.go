package serve

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/build"
	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/render"
)

// The opmodel.dev preview interface site mode relies on (docs-kit
// "Site decisions"): two tasks and one variable.
const (
	SitePullTask  = "bundles:pull"
	SiteServeTask = "serve"
	SiteLocalVar  = "OPM_BUNDLES_LOCAL"
)

// reSiteVersion is C16's #SiteVersion.
var reSiteVersion = regexp.MustCompile(`^v(0|[1-9]\d*)\.(0|[1-9]\d*)$`)

// RunSite builds the selected projects once and previews them through the
// opmodel.dev checkout o.Site: `task bundles:pull`, then `task serve`, both
// run there with OPM_BUNDLES_LOCAL naming each built tree. It does not
// watch. It returns nil when ctx ends while the site serves.
func RunSite(ctx context.Context, o Options) error {
	cfg, projects, err := build.Selected(o.Build)
	if err != nil {
		return err
	}
	task, err := checkSite(o, cfg, projects)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "opm-docs-serve-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if strings.ContainsAny(tmp, " \t\n:,") {
		return usage("the build directory %s holds a space, ':' or ','; the site cannot mount it: set TMPDIR to a path without one", tmp)
	}
	bo := o.Build
	bo.Out = tmp
	bo.Release, bo.Revision, bo.Patches, bo.Check = "", 0, nil, false
	results, err := build.Run(ctx, bo)
	if err != nil {
		return &BuildError{err}
	}
	var pairs []string
	for _, r := range results {
		seg := "edge"
		if cfg.Bundles[r.Project].Placement.Kind == render.KindDocs {
			seg = o.Version
		}
		pairs = append(pairs, r.Project+"@"+seg+"="+abs(r.Dir))
	}
	local := strings.Join(pairs, " ")
	fmt.Fprintf(o.Stderr, "opm-docs serve: %s=%q in %s\n", SiteLocalVar, local, o.Site)
	env := append(withoutVar(os.Environ(), SiteLocalVar), SiteLocalVar+"="+local)
	for _, t := range []string{SitePullTask, SiteServeTask} {
		cmd := exec.CommandContext(ctx, task, t)
		cmd.Dir, cmd.Env = o.Site, env
		cmd.Stdin, cmd.Stdout, cmd.Stderr = o.Stdin, o.Stdout, o.Stderr
		stopped, err := runChild(ctx, cmd)
		if stopped {
			return nil
		}
		if err != nil {
			return fmt.Errorf("the site's preview failed in %s: %w", o.Site, err)
		}
	}
	return nil
}

// withoutVar drops name from env.
func withoutVar(env []string, name string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if !strings.HasPrefix(kv, name+"=") {
			out = append(out, kv)
		}
	}
	return out
}

// checkSite refuses what site mode cannot run: a docs bundle without a
// site version, a malformed one, no task, a directory that is no
// checkout. It returns the path of task.
func checkSite(o Options, cfg *config.Config, projects []string) (string, error) {
	docs := ""
	for _, p := range projects {
		if cfg.Bundles[p].Placement.Kind == render.KindDocs {
			docs = p
			break
		}
	}
	switch {
	case docs != "" && o.Version == "":
		return "", usage("%s is a docs bundle: pass --version, the site version it previews in (such as v1.0)", docs)
	case o.Version != "" && !reSiteVersion.MatchString(o.Version):
		return "", usage("--version %s: a site version is v<MAJOR>.<MINOR>, such as v1.0", o.Version)
	}
	task, err := lookPath("task")
	if err != nil {
		return "", usage("--site needs task (https://taskfile.dev) on PATH: the site's preview runs `task %s` and `task %s`", SitePullTask, SiteServeTask)
	}
	if !statOK(filepath.Join(o.Site, "Taskfile.yml")) {
		return "", usage("--site %s: no Taskfile.yml there; pass the root of an opmodel.dev checkout", o.Site)
	}
	return task, nil
}

// statOK reports an existing path.
func statOK(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
