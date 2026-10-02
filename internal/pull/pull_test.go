package pull

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/oci"
	"github.com/open-platform-model/docs-kit/internal/ocitest"
	"github.com/open-platform-model/docs-kit/internal/publish"
	"github.com/open-platform-model/docs-kit/internal/verify"
	"github.com/open-platform-model/docs-kit/internal/verify/sigtest"
)

const (
	project = "catalog-opm"
	owner   = "open-platform-model/catalog_opm"
)

type env struct {
	t        *testing.T
	srv      *ocitest.Server
	stop     func()
	registry string
	client   *oci.Client
	auth     *sigtest.Authority
	work     string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	srv := ocitest.New(t)
	return &env{t: t, srv: srv, stop: srv.Stop, registry: srv.Host + "/docs", client: oci.New(oci.Options{Anonymous: true, PlainHTTP: true}), auth: sigtest.New(t), work: t.TempDir()}
}

// tree writes a bundle directory of a version, from the bundle package's
// fixture.
func tree(t *testing.T, version string, edit func(*bundle.Manifest)) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), project)
	src := "../bundle/testdata/tree"
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dir, rel), 0o750)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, rel), b, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := bundle.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	m.Version = version
	if edit != nil {
		edit(m)
	}
	if err := bundle.Write(dir, m); err != nil {
		t.Fatal(err)
	}
	return dir
}

// publish pushes a build, signs it as signer's main and promotes it.
func (e *env) publish(version, signer string, edit func(*bundle.Manifest)) {
	e.t.Helper()
	ctx := context.Background()
	res, err := publish.Push(ctx, publish.PushOptions{Dir: tree(e.t, version, edit), Registry: e.registry, Client: e.client})
	if err != nil {
		e.t.Fatal(err)
	}
	repo, _ := e.client.Repository(e.registry + "/" + project)
	d, err := repo.Resolve(ctx, res.Digest)
	if err != nil {
		e.t.Fatal(err)
	}
	if signer != "" {
		e.auth.Sign(e.t, repo.Graph(), d, sigtest.Publisher(signer))
	}
	if version == "edge" {
		// An unsigned edge build cannot be promoted; point the tag by hand.
		_, raw, _ := repo.Manifest(ctx, d)
		if err := repo.Tag(ctx, d, raw, "edge"); err != nil {
			e.t.Fatal(err)
		}
		return
	}
	if _, err := publish.Promote(ctx, publish.PromoteOptions{Project: project, Digest: res.Digest, Registry: e.registry, Client: e.client,
		Verifier: e.auth.Verifier(e.t), Policy: sigtest.Policy(signer)}); err != nil {
		e.t.Fatal(err)
	}
}

func (e *env) config(body string) string {
	e.t.Helper()
	p := filepath.Join(e.work, "bundles.cue")
	if body == "" {
		body = `registry: "` + e.registry + `"
tabs: "catalog-opm": {repo: "` + owner + `", root: "/catalogs/opm/", from: "4.4"}
`
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		e.t.Fatal(err)
	}
	return p
}

func (e *env) options(cfg string) Options {
	out := filepath.Join(e.work, "bundles")
	return Options{
		Config: cfg, Out: out, Lock: filepath.Join(out, "lock.json"), Tool: "0.1.0",
		Cache: Cache{Dir: filepath.Join(e.work, "cache")}, Client: e.client,
		Verifier: func() (*verify.Verifier, error) { return e.auth.Verifier(e.t), nil },
	}
}

func segments(l *Lock) string {
	s := make([]string, 0, len(l.Bundles))
	for i := range l.Bundles {
		s = append(s, l.Bundles[i].Segment)
	}
	return strings.Join(s, ",")
}

func TestResolveTabs(t *testing.T) {
	e := newEnv(t)
	e.publish("4.3.2", owner, nil)
	e.publish("4.4.5", owner, nil)
	e.publish("edge", owner, func(m *bundle.Manifest) { m.Source.Ref = "main" })
	cfg := e.config("")
	l, err := Run(context.Background(), e.options(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if segments(l) != "4.4,edge" {
		t.Fatalf("segments %s", segments(l))
	}
	if _, err := os.Stat(filepath.Join(e.work, "bundles", project, "4.3")); !os.IsNotExist(err) {
		t.Fatal("pulled 4.3, below from")
	}
	first, _ := os.ReadFile(filepath.Join(e.work, "bundles", "lock.json"))
	// A new minor appears with an unchanged config.
	e.publish("4.5.0", owner, nil)
	l, err = Run(context.Background(), e.options(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if segments(l) != "4.4,4.5,edge" {
		t.Fatalf("segments after a new minor %s", segments(l))
	}
	if e0 := l.Bundles[0]; e0.Tag != "4.4" || e0.Version != "4.4.5" || e0.Signer == nil || e0.Signer.Repository != "https://github.com/"+owner || e0.Dir != "catalog-opm/4.4" {
		t.Fatalf("entry %+v", e0)
	}
	again, err := Run(context.Background(), e.options(cfg))
	if err != nil {
		t.Fatal(err)
	}
	a, _ := again.Encode()
	b, _ := l.Encode()
	if !bytes.Equal(a, b) || bytes.Equal(first, a) {
		t.Fatal("two pulls of one resolution wrote different locks, or a new minor changed nothing")
	}
}

func TestOfflineFrozen(t *testing.T) {
	e := newEnv(t)
	e.publish("4.4.5", owner, nil)
	cfg := e.config("")
	o := e.options(cfg)
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	saved := filepath.Join(e.work, "saved.lock")
	want, _ := os.ReadFile(o.Lock)
	_ = os.WriteFile(saved, want, 0o600)
	_ = os.RemoveAll(o.Out)
	o.Frozen, o.Offline = saved, true
	e.stop() // offline: the registry is gone
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(o.Lock)
	if !bytes.Equal(got, want) {
		t.Fatalf("offline lock differs:\n%s\nwant:\n%s", got, want)
	}
	// An empty cache fails naming the first missing digest.
	o.Cache = Cache{Dir: filepath.Join(e.work, "empty")}
	if _, err := Run(context.Background(), o); err == nil || !strings.Contains(err.Error(), "not in the cache") {
		t.Fatalf("err = %v", err)
	}
}

func TestFrozenFromAnotherConfig(t *testing.T) {
	e := newEnv(t)
	e.publish("4.4.5", owner, nil)
	o := e.options(e.config(""))
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	saved := filepath.Join(e.work, "saved.lock")
	b, _ := os.ReadFile(o.Lock)
	_ = os.WriteFile(saved, b, 0o600)
	o.Config = e.config(`registry: "` + e.registry + `"
tabs: "catalog-opm": {repo: "open-platform-model/cli", root: "/catalogs/opm/", from: "4.4"}
`)
	o.Frozen = saved
	_, err := Run(context.Background(), o)
	if !IsUsage(err) || strings.Count(err.Error(), "sha256:") != 2 {
		t.Fatalf("err = %v", err)
	}
}

func TestRefusals(t *testing.T) {
	cases := []struct {
		name  string
		setup func(e *env)
		want  []string
	}{
		{"signed by another repository", func(e *env) { e.publish("4.4.5", "open-platform-model/cli", nil) },
			[]string{"catalog-opm", "4.4", "sha256:", "only repository allowed"}},
		{"unsigned edge", func(e *env) {
			e.publish("4.4.5", owner, nil)
			e.publish("edge", "", func(m *bundle.Manifest) { m.Source.Ref = "main" })
		}, []string{"edge", "sha256:", "no signature"}},
		{"placement mismatch", func(e *env) {
			e.publish("4.4.5", owner, func(m *bundle.Manifest) { m.Placement.Root = "/catalogs/other/" })
		}, []string{"catalog-opm", "sha256:", "/catalogs/other/", "/catalogs/opm/"}},
		{"nothing to show", func(e *env) { e.publish("4.3.2", owner, nil) }, []string{"catalog-opm", "empty"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t)
			c.setup(e)
			_, err := Run(context.Background(), e.options(e.config("")))
			if err == nil {
				t.Fatal("pulled")
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not name %q", err, w)
				}
			}
		})
	}
}

func TestMissingEdgeWarns(t *testing.T) {
	e := newEnv(t)
	e.publish("4.4.5", owner, nil)
	o := e.options(e.config(""))
	var warnings []string
	o.Warn = func(s string) { warnings = append(warnings, s) }
	l, err := Run(context.Background(), o)
	if err != nil || segments(l) != "4.4" || len(warnings) != 1 || !strings.Contains(warnings[0], "edge") {
		t.Fatalf("%v %s %v", err, segments(l), warnings)
	}
	// The counter does see a layer fetch: the refusal tests rely on it.
	if n := e.srv.BlobGets(e.layerOf("4.4").Digest.String()); n != 1 {
		t.Fatalf("layer fetched %d time(s), want 1", n)
	}
}

func TestLocalOnly(t *testing.T) {
	e := newEnv(t)
	o := e.options(e.config(""))
	e.stop() // an all-local pull needs no registry
	o.Verifier = func() (*verify.Verifier, error) { return nil, errors.New("a local pull fetched the trusted root") }
	o.Locals = []Local{
		{project, "4.4", tree(t, "4.4.5", nil)},
		{project, "4.5", tree(t, "4.5.0", nil)},
		{project, "edge", tree(t, "edge", func(m *bundle.Manifest) { m.Source.Ref = "main" })},
	}
	// A segment directory from an earlier pull is swept.
	_ = os.MkdirAll(filepath.Join(o.Out, project, "4.3"), 0o750)
	_ = os.WriteFile(filepath.Join(o.Out, project, "history.json"), []byte("{}"), 0o600)
	l, err := Run(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if segments(l) != "4.4,4.5,edge" {
		t.Fatalf("segments %s", segments(l))
	}
	for _, en := range l.Bundles {
		if !en.Local || en.Digest != "" || en.Signer != nil {
			t.Fatalf("entry %+v", en)
		}
	}
	data, _ := os.ReadFile(o.Lock)
	if !strings.Contains(string(data), "\"segment\": \"4.4\",\n      \"local\": true,") || strings.Contains(string(data), "\"tag\"") {
		t.Fatalf("lock:\n%s", data)
	}
	if _, err := os.Stat(filepath.Join(o.Out, project, "4.3")); !os.IsNotExist(err) {
		t.Fatal("stale segment left")
	}
	if _, err := os.Stat(filepath.Join(o.Out, project, "history.json")); err != nil {
		t.Fatal("history.json removed")
	}
	o.Locals = []Local{{project, "4.5", tree(t, "4.4.5", nil)}}
	if _, err := Run(context.Background(), o); !IsUsage(err) || !strings.Contains(err.Error(), "4.4.5") {
		t.Fatalf("segment mismatch: %v", err)
	}
}

func TestParseLocal(t *testing.T) {
	if l, err := ParseLocal("catalog-opm@4.4=out/x"); err != nil || l.Dir != "out/x" || l.Segment != "4.4" {
		t.Fatalf("%+v %v", l, err)
	}
	for _, bad := range []string{"catalog-opm=dir", "catalog-opm@4=dir", "@4.4=dir", "catalog-opm@4.4="} {
		if _, err := ParseLocal(bad); err == nil {
			t.Errorf("%s parsed", bad)
		}
	}
}

// layerOf returns the layer descriptor of the manifest a tag names.
func (e *env) layerOf(tag string) ocispec.Descriptor {
	e.t.Helper()
	repo, _ := e.client.Repository(e.registry + "/" + project)
	d, err := repo.Resolve(context.Background(), tag)
	if err != nil {
		e.t.Fatal(err)
	}
	m, _, err := repo.Manifest(context.Background(), d)
	if err != nil {
		e.t.Fatal(err)
	}
	return m.Layers[0]
}

// TestRefusedSignatureFetchesNoLayer: a bundle whose signature fails the
// policy never has its layer fetched.
func TestRefusedSignatureFetchesNoLayer(t *testing.T) {
	e := newEnv(t)
	e.publish("4.4.5", "open-platform-model/cli", nil)
	layer := e.layerOf("4.4")
	if _, err := Run(context.Background(), e.options(e.config(""))); err == nil {
		t.Fatal("pulled")
	}
	if n := e.srv.BlobGets(layer.Digest.String()); n != 0 {
		t.Fatalf("the layer of a refused bundle was fetched %d time(s)", n)
	}
}

// TestOversizedLayerNotFetched: a signed bundle whose layer descriptor is
// larger than a bundle may be is refused before a byte of it moves.
func TestOversizedLayerNotFetched(t *testing.T) {
	e := newEnv(t)
	e.publish("4.4.5", owner, nil)
	ctx := context.Background()
	repo, _ := e.client.Repository(e.registry + "/" + project)
	d, _ := repo.Resolve(ctx, "4.4")
	m, _, err := repo.Manifest(ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	m.Layers[0].Size = bundle.MaxLayerSize + 1
	raw, _ := json.Marshal(m)
	big := content.NewDescriptorFromBytes(ocispec.MediaTypeImageManifest, raw)
	if err := repo.Graph().PushReference(ctx, big, bytes.NewReader(raw), "4.4"); err != nil {
		t.Fatal(err)
	}
	e.auth.Sign(t, repo.Graph(), big, sigtest.Publisher(owner))
	_, err = Run(ctx, e.options(e.config("")))
	if err == nil || !strings.Contains(err.Error(), "more than the 33554432 a bundle may hold") {
		t.Fatalf("err = %v", err)
	}
	if n := e.srv.BlobGets(m.Layers[0].Digest.String()); n != 0 {
		t.Fatalf("an oversized layer was fetched %d time(s)", n)
	}
}

// TestFrozenLockOutsideTheConfig: a frozen lock may only name what the
// config would pull: the configured repository and a segment it shows.
func TestFrozenLockOutsideTheConfig(t *testing.T) {
	e := newEnv(t)
	e.publish("4.4.5", owner, nil)
	o := e.options(e.config(""))
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	good, _ := os.ReadFile(o.Lock)
	for _, c := range []struct{ name, from, to, want string }{
		{"another repository", `"repository": "` + e.registry + `/catalog-opm"`, `"repository": "ghcr.io/evil/catalog-opm"`, "names the repository ghcr.io/evil/catalog-opm"},
		{"a segment below from", `"segment": "4.4"`, `"segment": "4.3"`, "catalog-opm@4.3 is not a segment the config shows"},
	} {
		t.Run(c.name, func(t *testing.T) {
			bad := strings.Replace(string(good), c.from, c.to, 1)
			if bad == string(good) {
				t.Fatal("replacement did not apply")
			}
			p := filepath.Join(t.TempDir(), "lock.json")
			_ = os.WriteFile(p, []byte(bad), 0o600)
			oo := o
			oo.Frozen = p
			if _, err := Run(context.Background(), oo); !IsUsage(err) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

// TestRefusedBundleKeepsThePreviousSegment: a bundle that fails its lint
// is never swapped in.
func TestRefusedBundleKeepsThePreviousSegment(t *testing.T) {
	e := newEnv(t)
	o := e.options(e.config(""))
	o.Locals = []Local{{project, "4.4", tree(t, "4.4.5", nil)}}
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(o.Out, project, "4.4", "content", "traits", "backup.md")
	before, _ := os.ReadFile(page)
	bad := tree(t, "4.4.6", nil)
	_ = os.WriteFile(filepath.Join(bad, "content", "traits", "backup.md"), []byte("---\ntitle: \"x\"\ndescription: \"x\"\ntype: reference\n---\n\n[x](relative.md)\n"), 0o600)
	o.Locals = []Local{{project, "4.4", bad}}
	var le *LintError
	if _, err := Run(context.Background(), o); !errors.As(err, &le) {
		t.Fatalf("err = %v", err)
	}
	after, err := os.ReadFile(page)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("the previous segment changed: %v", err)
	}
	if left, _ := filepath.Glob(filepath.Join(o.Out, project, ".incoming-*")); len(left) != 0 {
		t.Fatalf("left %v", left)
	}
}
