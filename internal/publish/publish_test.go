package publish

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/oci"
	"github.com/open-platform-model/docs-kit/internal/ocitest"
	"github.com/open-platform-model/docs-kit/internal/verify/sigtest"
)

const project = "catalog-opm"

type env struct {
	t        *testing.T
	registry string
	client   *oci.Client
	auth     *sigtest.Authority
	repo     *oci.Repo
}

func newEnv(t *testing.T) *env {
	t.Helper()
	host := ocitest.Registry(t)
	c := oci.New(oci.Options{Anonymous: true, PlainHTTP: true, TagPageSize: 3})
	repo, err := c.Repository(host + "/docs/" + project)
	if err != nil {
		t.Fatal(err)
	}
	return &env{t: t, registry: host + "/docs", client: c, auth: sigtest.New(t), repo: repo}
}

// bundleDir writes a bundle of the given version and revision, built from
// the bundle package's fixture tree with its manifest rewritten.
func (e *env) bundleDir(version string, revision int, body string) string {
	e.t.Helper()
	dir := filepath.Join(e.t.TempDir(), project)
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
		e.t.Fatal(err)
	}
	m, err := bundle.Read(dir)
	if err != nil {
		e.t.Fatal(err)
	}
	m.Version, m.Revision = version, revision
	if version == "edge" {
		m.Source.Ref = "main"
	}
	if err := bundle.Write(dir, m); err != nil {
		e.t.Fatal(err)
	}
	if body != "" {
		if err := os.WriteFile(filepath.Join(dir, "content", "traits", "backup.md"), []byte("---\ntitle: \"Backup\"\ndescription: \"Backup.\"\ntype: reference\n---\n\n"+body+"\n"), 0o600); err != nil {
			e.t.Fatal(err)
		}
	}
	return dir
}

func (e *env) push(dir string) *PushResult {
	e.t.Helper()
	res, err := Push(context.Background(), PushOptions{Dir: dir, Registry: e.registry, Client: e.client})
	if err != nil {
		e.t.Fatal(err)
	}
	return res
}

func (e *env) sign(digest string) {
	e.t.Helper()
	d, err := e.repo.Resolve(context.Background(), digest)
	if err != nil {
		e.t.Fatal(err)
	}
	e.auth.Sign(e.t, e.repo.Graph(), d, sigtest.Publisher("open-platform-model/catalog_opm"))
}

func (e *env) promote(digest string) (*PromoteResult, error) {
	return Promote(context.Background(), PromoteOptions{
		Project: project, Digest: digest, Registry: e.registry, Client: e.client,
		Verifier: e.auth.Verifier(e.t), Policy: sigtest.Policy("open-platform-model/catalog_opm"),
	})
}

// publish pushes, signs and promotes one build, as the workflow does.
func (e *env) publish(version string, revision int, body string) string {
	e.t.Helper()
	res := e.push(e.bundleDir(version, revision, body))
	e.sign(res.Digest)
	if _, err := e.promote(res.Digest); err != nil {
		e.t.Fatal(err)
	}
	return res.Digest
}

func (e *env) at(tag string) string {
	e.t.Helper()
	d, err := e.repo.Resolve(context.Background(), tag)
	if err != nil {
		return ""
	}
	return d.Digest.String()
}

func TestPushFullTagRules(t *testing.T) {
	e := newEnv(t)
	dir := e.bundleDir("4.4.5", 0, "")
	first := e.push(dir)
	if first.Tag != "4.4.5.0" || first.Existing || e.at("4.4.5.0") != first.Digest {
		t.Fatalf("first push %+v", first)
	}
	again := e.push(dir)
	if again.Digest != first.Digest || !again.Existing {
		t.Fatalf("re-run %+v", again)
	}
	_, err := Push(context.Background(), PushOptions{Dir: e.bundleDir("4.4.5", 0, "Changed."), Registry: e.registry, Client: e.client})
	if err == nil || !strings.Contains(err.Error(), "4.4.5.0 is already published as "+first.Digest) || !strings.Contains(err.Error(), "docs revision") {
		t.Fatalf("different build: %v", err)
	}
	if e.at("4.4.5.0") != first.Digest {
		t.Fatal("the full tag moved")
	}
}

func TestPushRefusals(t *testing.T) {
	e := newEnv(t)
	dir := e.bundleDir("4.4.5", 0, "")
	m, _ := bundle.Read(dir)
	m.Source.Dirty = true
	_ = bundle.Write(dir, m)
	if _, err := Push(context.Background(), PushOptions{Dir: dir, Registry: e.registry, Client: e.client}); err == nil || !strings.Contains(err.Error(), "uncommitted") {
		t.Fatalf("dirty: %v", err)
	}
	dir = e.bundleDir("4.4.5", 0, "")
	_ = os.WriteFile(filepath.Join(dir, "content", "traits", "extra.md"), []byte("x"), 0o600)
	if _, err := Push(context.Background(), PushOptions{Dir: dir, Registry: e.registry, Client: e.client}); err == nil || !strings.Contains(err.Error(), "violation") {
		t.Fatalf("unlisted page: %v", err)
	}
	if tags, _ := e.repo.Tags(context.Background()); len(tags) != 0 {
		t.Fatalf("a refused push wrote %v", tags)
	}
}

func TestEdge(t *testing.T) {
	e := newEnv(t)
	res := e.push(e.bundleDir("edge", 0, ""))
	if res.Tag != "" || e.at("edge") != "" {
		t.Fatalf("edge push %+v tagged %q", res, e.at("edge"))
	}
	e.sign(res.Digest)
	pr, err := e.promote(res.Digest)
	if err != nil || !slices.Equal(pr.Moved, []string{"edge"}) || e.at("edge") != res.Digest {
		t.Fatalf("promote edge %+v %v", pr, err)
	}
}

func TestUnsignedNotPromoted(t *testing.T) {
	e := newEnv(t)
	res := e.push(e.bundleDir("4.4.5", 0, ""))
	_, err := e.promote(res.Digest)
	if err == nil || !strings.Contains(err.Error(), res.Digest) || !strings.Contains(err.Error(), "no signature") {
		t.Fatalf("err = %v", err)
	}
	for _, tag := range []string{"4.4.5", "4.4", "4"} {
		if e.at(tag) != "" {
			t.Fatalf("%s moved", tag)
		}
	}
	// A signature by another repository does not promote either.
	d, _ := e.repo.Resolve(context.Background(), res.Digest)
	e.auth.Sign(t, e.repo.Graph(), d, sigtest.Publisher("open-platform-model/cli"))
	if _, err := e.promote(res.Digest); err == nil || !strings.Contains(err.Error(), "only repository allowed") {
		t.Fatalf("err = %v", err)
	}
}

func TestPromotionLines(t *testing.T) {
	e := newEnv(t)
	d445 := e.publish("4.4.5", 0, "")
	for _, tag := range []string{"4.4.5", "4.4", "4"} {
		if e.at(tag) != d445 {
			t.Fatalf("first release: %s at %s", tag, e.at(tag))
		}
	}
	d446 := e.publish("4.4.6", 0, "")
	for _, tag := range []string{"4.4.6", "4.4", "4"} {
		if e.at(tag) != d446 {
			t.Fatalf("new patch: %s at %s", tag, e.at(tag))
		}
	}
	// A revision of the older patch moves only its release tag.
	d4451 := e.publish("4.4.5", 1, "Fixed.")
	if e.at("4.4.5") != d4451 || e.at("4.4") != d446 || e.at("4") != d446 {
		t.Fatalf("revision of an older patch: 4.4.5=%s 4.4=%s 4=%s", e.at("4.4.5"), e.at("4.4"), e.at("4"))
	}
	// A re-run of a promote is a no-op.
	if pr, err := e.promote(d446); err != nil || len(pr.Moved) != 3 || e.at("4") != d446 {
		t.Fatalf("re-run %+v %v", pr, err)
	}
}

func TestConcurrentReleases(t *testing.T) {
	e := newEnv(t)
	e.publish("4.4.5", 0, "")
	// 4.4.6.0 is pushed and signed, but its promote is slow...
	slow := e.push(e.bundleDir("4.4.6", 0, ""))
	e.sign(slow.Digest)
	// ...and 4.5.0.0 publishes in full meanwhile.
	d450 := e.publish("4.5.0", 0, "")
	pr, err := e.promote(slow.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if e.at("4") != d450 || e.at("4.4") != slow.Digest || e.at("4.4.6") != slow.Digest || !slices.Contains(pr.Skipped, "4") {
		t.Fatalf("4=%s 4.4=%s 4.4.6=%s %+v", e.at("4"), e.at("4.4"), e.at("4.4.6"), pr)
	}
}

func TestPrereleaseTagCollision(t *testing.T) {
	e := newEnv(t)
	// Build 1.0.0-beta revision 5 owns the full tag 1.0.0-beta.5...
	owner := e.publish("1.0.0-beta", 5, "")
	// ...which is also the release tag of version 1.0.0-beta.5.
	res := e.push(e.bundleDir("1.0.0-beta.5", 0, ""))
	e.sign(res.Digest)
	_, err := e.promote(res.Digest)
	if err == nil || !strings.Contains(err.Error(), "1.0.0-beta.5 is the full tag of build 1.0.0-beta.5") || !strings.Contains(err.Error(), "never moves") {
		t.Fatalf("err = %v", err)
	}
	if e.at("1.0.0-beta.5") != owner {
		t.Fatal("a full tag moved")
	}
}

func TestEdgeNeverMovesBack(t *testing.T) {
	e := newEnv(t)
	edge := func(created, body string) string {
		dir := e.bundleDir("edge", 0, body)
		m, _ := bundle.Read(dir)
		m.Created = created
		if err := bundle.Write(dir, m); err != nil {
			t.Fatal(err)
		}
		res := e.push(dir)
		e.sign(res.Digest)
		if _, err := e.promote(res.Digest); err != nil {
			t.Fatal(err)
		}
		return res.Digest
	}
	newer := edge("2026-10-02T12:00:00Z", "Newer.")
	edge("2026-10-01T12:00:00Z", "Older.")
	if e.at("edge") != newer {
		t.Fatal("edge moved back to an older commit")
	}
	newest := edge("2026-10-03T12:00:00Z", "Newest.")
	if e.at("edge") != newest {
		t.Fatal("edge did not move forward")
	}
}
