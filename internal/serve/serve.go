// Package serve previews a repository's bundles: it builds them as edge
// bundles and serves them on a skeleton Hugo site embedded in opm-docs,
// rebuilding a bundle when a file under its sources changes, or hands them
// to an opmodel.dev checkout's own preview (site mode).
package serve

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/open-platform-model/docs-kit/internal/build"
	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/config"
)

// Options configures serve.
type Options struct {
	// Build is the build every bundle uses: Config, Projects, Source, Tool
	// and Stderr. serve sets Out and always builds edge.
	Build build.Options
	Port  int // the skeleton's port on 127.0.0.1
	// Site is an opmodel.dev checkout: serve through its own preview
	// instead of the skeleton (RunSite). SiteVersion is the site version
	// a docs bundle previews in.
	Site        string
	SiteVersion string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	// Interval is the polling period; 0 is one second.
	Interval time.Duration
}

// Bind is the only address the skeleton listens on.
const Bind = "127.0.0.1"

// BuildError reports the first build failing; Err is build's error.
type BuildError struct{ Err error }

func (e *BuildError) Error() string { return e.Err.Error() }
func (e *BuildError) Unwrap() error { return e.Err }

// Report prints a failed rebuild while watching; the command sets it to
// print lint violations as build does. nil prints the error.
type Report func(w io.Writer, err error)

// server holds one serve run's directories: stage/<project> is where
// each build goes, live/<project>/content is what Hugo mounts.
type server struct {
	o      Options
	stage  string
	live   string
	build  func(ctx context.Context, project string) error // builds into stage/<project>
	report Report
	// restart asks for hugo to start again (pages added or removed); nil
	// in tests without hugo.
	restart func()
}

// buildRun is build.Run; tests replace it.
var buildRun = build.Run

// Run builds the selected projects and serves them on the skeleton with
// the host's hugo until ctx ends (nil) or hugo exits (an error). A first
// build that fails returns a *BuildError, unless ctx ended during it; a
// rebuild that fails while watching is reported and the last good bundle
// stays served.
func Run(ctx context.Context, o Options, report Report) error {
	cfg, projects, err := build.Selected(o.Build)
	if err != nil {
		return err
	}
	hugo, err := findHugo(ctx)
	if err != nil {
		return err
	}
	tmp, err := makeTemp()
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	s := &server{o: o, stage: filepath.Join(tmp, "stage"), live: filepath.Join(tmp, "live"), report: report}
	s.build = s.buildOne
	var mounts []mount
	for _, p := range projects {
		if err := s.build(ctx, p); err != nil {
			if ctx.Err() != nil {
				return nil // stopped during the first build
			}
			return &BuildError{err}
		}
		if _, err := syncTree(filepath.Join(s.stage, p, bundle.ContentDir), filepath.Join(s.live, p, bundle.ContentDir)); err != nil {
			return err
		}
		m, err := bundle.Read(filepath.Join(s.stage, p))
		if err != nil {
			return err
		}
		mounts = append(mounts, mount{Source: filepath.Join(s.live, p, bundle.ContentDir), Target: mountTarget(m.Placement.Kind, m.Placement.Root)})
	}
	siteDir := filepath.Join(tmp, "site")
	if err := writeSite(siteDir, mounts); err != nil {
		return err
	}
	base := "http://" + Bind + ":" + strconv.Itoa(o.Port)
	ready := func() {
		for i, p := range projects {
			fmt.Fprintf(o.Stderr, "opm-docs serve: %s at %s%s\n", p, base, urlPath(mounts[i].Target))
		}
		fmt.Fprintf(o.Stderr, "opm-docs serve: watching the sources; Ctrl-C stops\n")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	restart := make(chan struct{}, 1)
	s.restart = func() {
		select {
		case restart <- struct{}{}:
		default:
		}
	}
	w := newWatcher(roots(o.Build.Source, cfg, projects))
	interval := o.Interval
	if interval == 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.watch(ctx, ticker.C, w)
	}()
	err = s.hugo(ctx, hugo, siteDir, restart, ready)
	cancel()
	<-done
	return err
}

// hugo runs `hugo server` until ctx ends (nil) or it exits on its own (an
// error), starting it again on each restart: Hugo's own watcher follows
// changed files, not a page added in a new directory or removed. ready
// runs once, when the first hugo accepts connections.
func (s *server) hugo(ctx context.Context, hugo, siteDir string, restart <-chan struct{}, ready func()) error {
	addr := Bind + ":" + strconv.Itoa(s.o.Port)
	first := true
	for {
		hctx, hcancel := context.WithCancel(ctx)
		cmd := exec.CommandContext(hctx, hugo, "server",
			"--source", siteDir, "--bind", Bind, "--port", strconv.Itoa(s.o.Port), "--baseURL", "http://"+addr+"/", "--appendPort=false", "--renderToMemory")
		cmd.Env = withoutPrefix(os.Environ(), "HUGO_")
		cmd.Stdout, cmd.Stderr = s.o.Stderr, s.o.Stderr
		type result struct {
			stopped bool
			err     error
		}
		exited := make(chan result, 1)
		go func() {
			stopped, err := runChild(hctx, cmd)
			exited <- result{stopped, err}
		}()
		if first {
			first = false
			go func() {
				if serving(hctx, addr) {
					ready()
				}
			}()
		}
		select {
		case r := <-exited:
			hcancel()
			if ended(ctx) || r.stopped {
				return nil
			}
			err := r.err
			if err == nil {
				err = errors.New("hugo server stopped")
			}
			return fmt.Errorf("%w; the preview is down (when the port is in use, pass another with --port)", err)
		case <-ctx.Done():
			hcancel()
			<-exited
			return nil
		case <-restart:
			hcancel()
			<-exited
			if ended(ctx) {
				return nil
			}
			fmt.Fprintf(s.o.Stderr, "opm-docs serve: pages were added or removed; restarting hugo\n")
		}
	}
}

// ended reports a context that is done: the author stopped serve.
func ended(ctx context.Context) bool { return ctx.Err() != nil }

// serving waits until addr accepts a connection (true), or ctx ends or a
// minute passes (false).
func serving(ctx context.Context, addr string) bool {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	for ctx.Err() == nil {
		c, err := (&net.Dialer{Timeout: time.Second}).DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = c.Close()
			return true
		}
		select {
		case <-ctx.Done():
		case <-time.After(100 * time.Millisecond):
		}
	}
	return false
}

// withoutPrefix drops every variable whose name starts with prefix: a
// HUGO_* variable of the author's shell would reconfigure the skeleton.
func withoutPrefix(env []string, prefix string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if !strings.HasPrefix(kv, prefix) {
			out = append(out, kv)
		}
	}
	return out
}

// roots is every project's watch roots.
func roots(source string, cfg *config.Config, projects []string) map[string][]root {
	out := map[string][]root{}
	for _, p := range projects {
		out[p] = watchRoots(source, cfg.Path, cfg.Bundles[p])
	}
	return out
}

// buildOne builds one project as an edge bundle into stage/<project>.
func (s *server) buildOne(ctx context.Context, project string) error {
	o := s.o.Build
	o.Projects = []string{project}
	o.Out = s.stage
	o.Release, o.Revision, o.Patches, o.Check = "", 0, nil, false
	_, err := buildRun(ctx, o)
	return err
}

// watch rebuilds each project whose files changed, at every tick, until
// ctx ends. After a round it reloads the config's roots, so a source
// added to docs-kit.cue is watched; a config that no longer loads keeps
// the previous roots.
func (s *server) watch(ctx context.Context, ticks <-chan time.Time, w *watcher) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
		}
		changed := w.poll()
		for _, p := range changed {
			if ctx.Err() != nil {
				return
			}
			s.rebuild(ctx, p)
		}
		if len(changed) == 0 {
			continue
		}
		o := s.o.Build
		o.Projects = nil
		if cfg, _, err := build.Selected(o); err == nil {
			r := map[string][]root{}
			for p := range w.roots {
				if b, ok := cfg.Bundles[p]; ok {
					r[p] = watchRoots(s.o.Build.Source, cfg.Path, b)
				} else {
					r[p] = w.roots[p]
				}
			}
			w.setRoots(r)
		}
	}
}

// rebuild builds one project into the stage and, when it passes, copies
// its content over the served tree, so Hugo's own watcher reloads the
// changed pages. A failed build is reported and changes nothing served.
func (s *server) rebuild(ctx context.Context, project string) {
	fmt.Fprintf(s.o.Stderr, "opm-docs serve: %s changed; rebuilding\n", project)
	if err := s.build(ctx, project); err != nil {
		if ctx.Err() != nil {
			return
		}
		if s.report != nil {
			s.report(s.o.Stderr, err)
		} else {
			fmt.Fprintln(s.o.Stderr, err)
		}
		fmt.Fprintf(s.o.Stderr, "opm-docs serve: %s: the build failed; still serving the last good build\n", project)
		return
	}
	changed, err := syncTree(filepath.Join(s.stage, project, bundle.ContentDir), filepath.Join(s.live, project, bundle.ContentDir))
	if err != nil {
		fmt.Fprintf(s.o.Stderr, "opm-docs serve: %s: copying the build over the served pages failed, so they may be partial: %v; save a page to rebuild\n", project, err)
		return
	}
	fmt.Fprintf(s.o.Stderr, "opm-docs serve: %s rebuilt\n", project)
	if changed && s.restart != nil {
		s.restart()
	}
}

// syncTree makes dst hold exactly src's files: it writes each file whose
// bytes differ and removes what src lacks, so only changed pages touch
// Hugo's watcher. It reports whether a file was added or removed.
func syncTree(src, dst string) (bool, error) {
	setChanged := false
	want := map[string]bool{}
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		to := filepath.Join(dst, rel)
		want[rel] = true
		if d.IsDir() {
			return os.MkdirAll(to, 0o750)
		}
		b, err := os.ReadFile(p) //nolint:gosec // a file of the bundle serve just built
		if err != nil {
			return err
		}
		old, err := os.ReadFile(to)
		if err == nil && bytes.Equal(old, b) {
			return nil
		}
		if err != nil {
			setChanged = true
		}
		return os.WriteFile(to, b, 0o600) //nolint:gosec // to is a path of the served copy, under dst
	})
	if err != nil {
		return setChanged, err
	}
	var stale []string
	err = filepath.WalkDir(dst, func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dst, p)
		if err != nil {
			return err
		}
		if !want[rel] {
			stale = append(stale, p)
		}
		return nil
	})
	if err != nil {
		return setChanged, err
	}
	// Deepest first, so a directory is removed after its files.
	for i := len(stale) - 1; i >= 0; i-- {
		setChanged = true
		if err := os.RemoveAll(stale[i]); err != nil {
			return setChanged, err
		}
	}
	return setChanged, nil
}
