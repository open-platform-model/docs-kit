package serve

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/open-platform-model/docs-kit/internal/build"
	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/dialect"
	"github.com/open-platform-model/docs-kit/internal/gittest"
)

func TestMountTarget(t *testing.T) {
	cases := []struct{ kind, root, target, url string }{
		{"tab", "/catalogs/opm/", "content/catalogs/opm/edge", "/catalogs/opm/edge/"},
		{"docs", "/docs/", "content/docs", "/docs/"},
		{"section", "/enhancements/", "content/enhancements", "/enhancements/"},
		{"section", "/", "content", "/"},
	}
	for _, c := range cases {
		got := mountTarget(c.kind, c.root)
		if got != c.target || urlPath(got) != c.url {
			t.Errorf("%s %s: target %q url %q, want %q %q", c.kind, c.root, got, urlPath(got), c.target, c.url)
		}
	}
}

func TestWriteSite(t *testing.T) {
	dir := t.TempDir()
	mounts := []mount{{Source: "/tmp/x/cli/content", Target: "content/docs"}}
	if err := writeSite(dir, mounts); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "hugo.json"))
	if err != nil {
		t.Fatal(err)
	}
	var c hugoConfig
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	want := []mount{{Source: "content", Target: "content"}, mounts[0]}
	if !slices.Equal(c.Module.Mounts, want) || c.Markup.Goldmark.Renderer.Unsafe {
		t.Fatalf("hugo.json:\n%s", b)
	}
	files := make([]string, 0, 3+len(dialect.Figures()))
	files = append(files, "content/_index.md", "layouts/baseof.html", "layouts/_markup/render-blockquote.html")
	for _, f := range dialect.Figures() {
		files = append(files, "layouts/_shortcodes/opm/"+f+".html")
	}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(f))); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}
}

func TestParseHugoVersion(t *testing.T) {
	cases := []struct {
		out   string
		v     [3]int
		ok    bool
		older bool
	}{
		{"hugo v0.167.0-3fff6fb5c267 linux/amd64 BuildDate=2026-09-28T14:50:38Z VendorInfo=gohugoio", [3]int{0, 167, 0}, true, false},
		{"hugo v0.146.0+extended darwin/arm64", [3]int{0, 146, 0}, true, false},
		{"hugo v0.145.9 linux/amd64", [3]int{0, 145, 9}, true, true},
		{"Hugo Static Site Generator v0.54.0/extended linux/amd64", [3]int{0, 54, 0}, true, true},
		{"hugo version unknown", [3]int{}, false, false},
	}
	for _, c := range cases {
		v, ok := parseHugoVersion(c.out)
		if v != c.v || ok != c.ok || (ok && older(v, MinHugo) != c.older) {
			t.Errorf("%q: %v %v", c.out, v, ok)
		}
	}
}

// fakeProgram writes an executable script named name into a new directory
// and returns its path.
func fakeProgram(t *testing.T, name, script string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+script), 0o700); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFindHugo(t *testing.T) {
	t.Cleanup(func() { lookPath = defaultLookPath })
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	_, err := findHugo(context.Background())
	var ue *UsageError
	if !errors.As(err, &ue) || !strings.Contains(err.Error(), "no hugo on PATH") {
		t.Fatalf("missing hugo: %v", err)
	}
	old := fakeProgram(t, "hugo", "echo 'hugo v0.140.1 linux/amd64'\n")
	lookPath = func(string) (string, error) { return old, nil }
	_, err = findHugo(context.Background())
	if !errors.As(err, &ue) || !strings.Contains(err.Error(), "is Hugo 0.140.1; serve needs 0.146.0 or later") {
		t.Fatalf("old hugo: %v", err)
	}
	ok := fakeProgram(t, "hugo", "echo 'hugo v0.167.0 linux/amd64'\n")
	lookPath = func(string) (string, error) { return ok, nil }
	if p, err := findHugo(context.Background()); err != nil || p != ok {
		t.Fatalf("current hugo: %q %v", p, err)
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// touch moves a file's modification time on, as an editor's save does
// when the size stays the same.
func touch(t *testing.T, path string, at time.Time) {
	t.Helper()
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}

func TestWatcher(t *testing.T) {
	dir := t.TempDir()
	docs := filepath.Join(dir, "docs", "site")
	api := filepath.Join(dir, "api")
	write(t, filepath.Join(docs, "a.md"), "a")
	write(t, filepath.Join(api, "x.cue"), "x")
	write(t, filepath.Join(dir, ".git", "HEAD"), "ref")
	w := newWatcher(map[string][]string{"cli": {docs}, "core": {api}, "all": {dir}})
	if got := w.poll(); len(got) != 0 {
		t.Fatalf("no change: %v", got)
	}
	touch(t, filepath.Join(docs, "a.md"), time.Now().Add(time.Hour))
	if got := w.poll(); !slices.Equal(got, []string{"all", "cli"}) {
		t.Fatalf("changed a.md: %v", got)
	}
	if got := w.poll(); len(got) != 0 {
		t.Fatalf("polled again: %v", got)
	}
	write(t, filepath.Join(dir, ".git", "HEAD"), "ref: refs/heads/other")
	if got := w.poll(); len(got) != 0 {
		t.Fatalf("a change under .git: %v", got)
	}
	write(t, filepath.Join(api, "y.cue"), "y")
	if got := w.poll(); !slices.Equal(got, []string{"all", "core"}) {
		t.Fatalf("added y.cue: %v", got)
	}
	if err := os.Remove(filepath.Join(docs, "a.md")); err != nil {
		t.Fatal(err)
	}
	if got := w.poll(); !slices.Equal(got, []string{"all", "cli"}) {
		t.Fatalf("removed a.md: %v", got)
	}
	w.setRoots(map[string][]string{"cli": {docs, api}})
	if got := w.poll(); len(got) != 0 {
		t.Fatalf("new roots take a baseline: %v", got)
	}
}

func TestWatchRoots(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "docs-kit.cue")
	write(t, cfgPath, `bundles: {
	ops: {
		placement: {kind: "docs", root: "/docs/", owns: ["reference/operator-resources.md"]}
		version: {from: "tag", prefix: "v"}
		sources: [
			{kind: "crd", dir: "./config/crd/bases", samples: "./config/samples", page: "reference/operator-resources.md", title: "t", description: "d"},
			{kind: "markdown", dir: "docs/site"},
		]
	}
	cli: {
		placement: {kind: "docs", root: "/docs/", owns: ["reference/cli/"]}
		version: {from: "tag", prefix: "v"}
		sources: [{kind: "cobra", command: ["go", "run", "./hack/dump"], section: "reference/cli/", title: "t", description: "d"}]
	}
}
`)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	got := watchRoots(dir, cfgPath, cfg.Bundles["ops"])
	want := []string{filepath.Join(dir, "config", "crd", "bases"), filepath.Join(dir, "config", "samples"), filepath.Join(dir, "docs", "site"), cfgPath}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("crd and markdown roots:\n%v\nwant\n%v", got, want)
	}
	got = watchRoots(dir, cfgPath, cfg.Bundles["cli"])
	if want := []string{dir, cfgPath}; !slices.Equal(got, want) {
		t.Fatalf("a repository command watches the tree: %v", got)
	}
}

// fakeServer is a server whose build writes body as content/page.md, or
// fails when body is "".
type fakeServer struct {
	*server
	mu     sync.Mutex
	body   string
	builds []string
}

func newFakeServer(t *testing.T) *fakeServer {
	t.Helper()
	tmp := t.TempDir()
	f := &fakeServer{server: &server{o: Options{Stderr: io.Discard}, stage: filepath.Join(tmp, "stage"), live: filepath.Join(tmp, "live")}}
	f.build = func(_ context.Context, project string) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.builds = append(f.builds, project)
		content := filepath.Join(f.stage, project, "content")
		if err := os.RemoveAll(filepath.Join(f.stage, project)); err != nil {
			return err
		}
		if f.body == "" {
			return errors.New("build failed")
		}
		write(t, filepath.Join(content, "page.md"), f.body)
		return nil
	}
	return f
}

func (f *fakeServer) set(body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.body = body
}

func (f *fakeServer) served(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(f.live, "cli", "content", p))
	if err != nil {
		return ""
	}
	return string(b)
}

func TestRebuildKeepsLastGood(t *testing.T) {
	f := newFakeServer(t)
	write(t, filepath.Join(f.live, "cli", "content", "stale.md"), "old")
	f.set("one")
	f.rebuild(context.Background(), "cli")
	if f.served(t, "page.md") != "one" || f.served(t, "stale.md") != "" {
		t.Fatalf("first build not served: %q %q", f.served(t, "page.md"), f.served(t, "stale.md"))
	}
	f.set("")
	var reported error
	f.report = func(_ io.Writer, err error) { reported = err }
	f.rebuild(context.Background(), "cli")
	if f.served(t, "page.md") != "one" || reported == nil {
		t.Fatalf("a failed build changed the served tree: %q (reported %v)", f.served(t, "page.md"), reported)
	}
	f.set("two")
	f.rebuild(context.Background(), "cli")
	if f.served(t, "page.md") != "two" {
		t.Fatalf("rebuild not served: %q", f.served(t, "page.md"))
	}
}

func TestWatchLoop(t *testing.T) {
	f := newFakeServer(t)
	src := t.TempDir()
	page := filepath.Join(src, "docs", "site", "a.md")
	write(t, page, "a")
	f.set("one")
	f.o.Build.Source = src // no config there: roots stay as they are
	w := newWatcher(map[string][]string{"cli": {filepath.Dir(page)}})
	ticks := make(chan time.Time)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); f.watch(ctx, ticks, w) }()
	ticks <- time.Now() // nothing changed
	touch(t, page, time.Now().Add(time.Hour))
	ticks <- time.Now()
	ticks <- time.Now() // waits for the previous round
	cancel()
	<-done
	f.mu.Lock()
	defer f.mu.Unlock()
	if !slices.Equal(f.builds, []string{"cli"}) {
		t.Fatalf("builds %v, want one of cli", f.builds)
	}
	if f.served(t, "page.md") != "one" {
		t.Fatal("rebuild not served")
	}
}

const docsConfig = `bundles: cli: {
	placement: {kind: "docs", root: "/docs/"}
	version: {from: "tag", prefix: "v"}
	sources: [{kind: "markdown", dir: "docs/site"}]
}
`

func installPage(body string) string {
	return "---\ntitle: \"Install\"\ndescription: \"Install the cli.\"\ntype: how-to\n---\n\n" + body + "\n\n> [!NOTE]\n> A note.\n\n{{< opm/module-to-cluster >}}\n"
}

func cliRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	t.Setenv("GITHUB_REPOSITORY", "")
	r := gittest.New(t, "https://github.com/example/cli.git")
	r.Write(map[string]string{"docs-kit.cue": docsConfig, "docs/site/start/install.md": installPage("First words.")})
	r.Commit("docs")
	return r
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", Bind+":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// get fetches url until its body holds want or the deadline passes.
func get(t *testing.T, url, want string, deadline time.Time) string {
	t.Helper()
	last := ""
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
		if resp, err := http.DefaultClient.Do(req); err == nil {
			b, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			last = string(b)
			if resp.StatusCode == http.StatusOK && strings.Contains(last, want) {
				return last
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("%s never held %q; last body:\n%s", url, want, last)
	return ""
}

// TestServeWithHugo runs the whole preview against the hugo on PATH. It
// skips without one, unless OPM_DOCS_REQUIRE_HUGO is set (CI's serve job,
// which installs a pinned Hugo).
func TestServeWithHugo(t *testing.T) {
	if _, err := exec.LookPath("hugo"); err != nil {
		if os.Getenv("OPM_DOCS_REQUIRE_HUGO") != "" {
			t.Fatal("OPM_DOCS_REQUIRE_HUGO is set and there is no hugo on PATH")
		}
		t.Skip("no hugo on PATH")
	}
	r := cliRepo(t)
	port := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errc := make(chan error, 1)
	var logs syncBuffer
	go func() {
		errc <- Run(ctx, Options{
			Build: build.Options{Source: r.Dir, Tool: "0.0.0-test", Stderr: &logs},
			Port:  port, Stderr: &logs, Interval: 100 * time.Millisecond,
		}, nil)
	}()
	url := "http://" + Bind + ":" + strconv.Itoa(port) + "/docs/start/install/"
	body := get(t, url, "First words.", time.Now().Add(60*time.Second))
	for _, want := range []string{`class="alert alert-note"`, "Figure <code>module-to-cluster</code>", "Install the cli."} {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q:\n%s", want, body)
		}
	}
	r.Write(map[string]string{"docs/site/start/install.md": installPage("Second words.")})
	get(t, url, "Second words.", time.Now().Add(30*time.Second))
	// A page breaking the dialect fails the rebuild; the last good page stays.
	r.Write(map[string]string{"docs/site/start/install.md": installPage("Third words.") + "\n![a picture](x.png)\n"})
	deadline := time.Now().Add(30 * time.Second)
	for !strings.Contains(logs.String(), "still serving the last good build") {
		if time.Now().After(deadline) {
			t.Fatalf("no failed rebuild reported:\n%s", logs.String())
		}
		time.Sleep(100 * time.Millisecond)
	}
	get(t, url, "Second words.", time.Now().Add(5*time.Second))
	cancel()
	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("stopped serve returned %v\n%s", err, logs.String())
		}
	case <-time.After(20 * time.Second):
		t.Fatal("serve did not stop")
	}
}

type syncBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}
