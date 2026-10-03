package serve

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/build"
	"github.com/open-platform-model/docs-kit/internal/gittest"
)

const siteConfig = docsConfig + `bundles: "catalog-x": {
	placement: {kind: "tab", root: "/catalogs/x/"}
	version: {from: "tag", prefix: "x-v"}
	sources: [{kind: "markdown", dir: "docs/catalog"}]
}
`

const catalogLanding = "---\ntitle: \"Catalog x\"\ndescription: \"The x catalog.\"\n---\n\nThe landing.\n"

// fakeTask puts a task on PATH that logs each run as
// "<args>|<dir>|<OPM_BUNDLES_LOCAL>|<every tree has a manifest>|
// <OPM_BUNDLES_FROZEN and OPM_BUNDLES>" and
// exits with status when its argument is fail.
func fakeTask(t *testing.T, fail string) string {
	t.Helper()
	log := filepath.Join(t.TempDir(), "task.log")
	p := fakeProgram(t, "task", `trees=yes
for pair in $OPM_BUNDLES_LOCAL; do [ -f "${pair#*=}/manifest.json" ] || trees=no; done
printf '%s|%s|%s|%s|%s\n' "$*" "$PWD" "$OPM_BUNDLES_LOCAL" "$trees" "$OPM_BUNDLES_FROZEN$OPM_BUNDLES" >> "`+log+`"
[ "$1" = "`+fail+`" ] && exit 3
exit 0
`)
	t.Setenv("PATH", filepath.Dir(p)+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func siteRepo(t *testing.T) (r *gittest.Repo, siteDir string) {
	t.Helper()
	t.Setenv("GITHUB_REPOSITORY", "")
	r = gittest.New(t, "https://github.com/example/cli.git")
	r.Write(map[string]string{"docs-kit.cue": siteConfig, "docs/site/start/install.md": installPage("Words."), "docs/catalog/_index.md": catalogLanding})
	r.Commit("docs")
	siteDir = t.TempDir()
	write(t, filepath.Join(siteDir, "Taskfile.yml"), "version: \"3\"\n")
	write(t, filepath.Join(siteDir, "site", "bundles.cue"), "tabs: {}\n")
	return r, siteDir
}

func siteOptions(r *gittest.Repo, site, version string) Options {
	return Options{
		Build: build.Options{Source: r.Dir, Tool: "0.0.0-test", Stderr: io.Discard},
		Site:  site, SiteVersion: version, Stdout: io.Discard, Stderr: io.Discard,
	}
}

func TestRunSite(t *testing.T) {
	log := fakeTask(t, "")
	r, site := siteRepo(t)
	t.Setenv("OPM_BUNDLES_FROZEN", "site/bundles.frozen.json")
	t.Setenv("OPM_BUNDLES", "elsewhere")
	if err := RunSite(context.Background(), siteOptions(r, site, "v1.0")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 {
		t.Fatalf("task ran %d times:\n%s", len(lines), b)
	}
	for i, task := range []string{"bundles:pull", "serve"} {
		f := strings.Split(lines[i], "|")
		if f[0] != task || f[1] != site {
			t.Errorf("run %d: task %q in %q, want %q in %q", i, f[0], f[1], task, site)
		}
		pairs := strings.Fields(f[2])
		if len(pairs) != 2 || !strings.HasPrefix(pairs[0], "catalog-x@edge=/") || !strings.HasPrefix(pairs[1], "cli@v1.0=/") ||
			!strings.HasSuffix(pairs[0], "/catalog-x") || !strings.HasSuffix(pairs[1], "/cli") {
			t.Errorf("run %d: OPM_BUNDLES_LOCAL=%q", i, f[2])
		}
		if f[3] != "yes" {
			t.Errorf("run %d: a named tree has no manifest.json", i)
		}
		if f[4] != "" {
			t.Errorf("run %d: the author's OPM_BUNDLES_FROZEN or OPM_BUNDLES reached the site: %q", i, f[4])
		}
	}
}

func TestRunSiteRefuses(t *testing.T) {
	r, site := siteRepo(t)
	onlyTaskfile := t.TempDir()
	write(t, filepath.Join(onlyTaskfile, "Taskfile.yml"), "version: \"3\"\n")
	cases := []struct {
		name, site, version, want string
	}{
		{"docs bundle without a site version", site, "", "pass --site-version"},
		{"malformed site version", site, "1.0", "--site-version 1.0: a site version is v<MAJOR>.<MINOR>"},
		{"no checkout", t.TempDir(), "v1.0", "no Taskfile.yml there"},
		{"no bundles.cue", onlyTaskfile, "v1.0", "no site/bundles.cue there"},
	}
	fakeTask(t, "")
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := RunSite(context.Background(), siteOptions(r, c.site, c.version))
			var ue *UsageError
			if !errors.As(err, &ue) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want a usage error naming %q", err, c.want)
			}
		})
	}
	t.Run("no task", func(t *testing.T) {
		t.Cleanup(func() { lookPath = defaultLookPath })
		lookPath = func(string) (string, error) { return "", errors.New("not found") }
		err := RunSite(context.Background(), siteOptions(r, site, "v1.0"))
		var ue *UsageError
		if !errors.As(err, &ue) || !strings.Contains(err.Error(), "needs task") {
			t.Fatalf("got %v", err)
		}
	})
}

func TestRunSitePullFails(t *testing.T) {
	log := fakeTask(t, "bundles:pull")
	r, site := siteRepo(t)
	err := RunSite(context.Background(), siteOptions(r, site, "v1.0"))
	if err == nil || !strings.Contains(err.Error(), "`") || !strings.Contains(err.Error(), "bundles:pull` exited with status 3") {
		t.Fatalf("got %v", err)
	}
	b, _ := os.ReadFile(log)
	if strings.Count(string(b), "\n") != 1 {
		t.Fatalf("task serve ran after a failed pull:\n%s", b)
	}
}

func TestRunSiteStoppedDuringBuild(t *testing.T) {
	log := fakeTask(t, "")
	r, site := siteRepo(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { buildRun = build.Run })
	buildRun = func(context.Context, build.Options) ([]build.Result, error) {
		cancel()
		return nil, errors.New("git: signal: interrupt")
	}
	if err := RunSite(ctx, siteOptions(r, site, "v1.0")); err != nil {
		t.Fatalf("stopped during the build: %v", err)
	}
	if _, err := os.Stat(log); err == nil {
		t.Fatal("task ran after the build was stopped")
	}
}
