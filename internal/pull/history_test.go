package pull

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/verify"
	"github.com/open-platform-model/docs-kit/schema"
)

var update = flag.Bool("update", false, "rewrite testdata/history/history.golden.json")

// historyTree is a bundle tree of version whose data/catalog.json is the
// fixture of segment seg, built by opm-docs tool.
func historyTree(t *testing.T, version, seg, tool string) string {
	t.Helper()
	dir := tree(t, version, func(m *bundle.Manifest) {
		m.Tool = tool
		if version == "edge" {
			m.Source.Ref = "main"
		}
	})
	b, err := os.ReadFile(filepath.Join("testdata", "history", seg, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data", "catalog.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// historyLocals are the three fixture segments: 4.5 and 4.6 built by one
// opm-docs minor, edge by the next.
func historyLocals(t *testing.T) []Local {
	return []Local{
		{project, "4.5", historyTree(t, "4.5.0", "4.5", "0.3.0")},
		{project, "4.6", historyTree(t, "4.6.1", "4.6", "0.3.2")},
		{project, "edge", historyTree(t, "edge", "edge", "0.4.0")},
	}
}

func historyFile(o Options) string { return filepath.Join(o.Out, project, "history.json") }

func TestLocalTreesGetAHistory(t *testing.T) {
	e := newEnv(t)
	o := e.options(e.config(""))
	e.stop()
	o.Verifier = func() (*verify.Verifier, error) { return nil, errors.New("a local pull fetched the trusted root") }
	o.Tool = "0.3.0"
	o.Locals = historyLocals(t)
	l, err := Run(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(historyFile(o))
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "history", "history.golden.json")
	if *update {
		_ = os.WriteFile(golden, got, 0o600)
	}
	want, _ := os.ReadFile(golden)
	if !bytes.Equal(got, want) {
		t.Fatalf("history.json differs from %s (go test -run %s -update):\n%s", golden, t.Name(), got)
	}
	sum := sha256.Sum256(got)
	if len(l.History) != 1 || l.History[0] != (HistoryEntry{Project: project, Digest: "sha256:" + hex.EncodeToString(sum[:]), Path: "catalog-opm/history.json"}) {
		t.Fatalf("lock history %+v", l.History)
	}
	data, _ := os.ReadFile(o.Lock)
	if !strings.Contains(string(data), "  ],\n  \"history\": [\n    {\n      \"project\": \"catalog-opm\",\n      \"digest\": \"sha256:") {
		t.Fatalf("lock:\n%s", data)
	}
	// A second run over the same trees writes the same bytes.
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	again, _ := os.ReadFile(historyFile(o))
	if !bytes.Equal(got, again) {
		t.Fatal("two pulls of the same trees wrote different histories")
	}
}

func TestOneSegmentRemovesTheHistory(t *testing.T) {
	e := newEnv(t)
	o := e.options(e.config(""))
	e.stop()
	o.Locals = historyLocals(t)
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(historyFile(o)); err != nil {
		t.Fatal(err)
	}
	o.Locals = o.Locals[:1]
	l, err := Run(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(historyFile(o)); !os.IsNotExist(err) {
		t.Fatalf("history.json left with one segment: %v", err)
	}
	data, _ := os.ReadFile(o.Lock)
	if l.History != nil || strings.Contains(string(data), "history") {
		t.Fatalf("lock:\n%s", data)
	}
}

// A segment whose manifest lists no cue-catalog data takes no part.
func TestSegmentWithoutCatalogTakesNoPart(t *testing.T) {
	e := newEnv(t)
	o := e.options(e.config(""))
	e.stop()
	bare := historyTree(t, "4.6.1", "4.6", "0.3.0")
	_ = os.Remove(filepath.Join(bare, "data", "catalog.json"))
	m, _ := bundle.Read(bare)
	m.Data = nil
	if err := bundle.Write(bare, m); err != nil {
		t.Fatal(err)
	}
	o.Locals = []Local{{project, "4.5", historyTree(t, "4.5.0", "4.5", "0.3.0")}, {project, "4.6", bare}}
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(historyFile(o)); !os.IsNotExist(err) {
		t.Fatalf("history.json written from one catalog segment: %v", err)
	}
}

// --frozen --offline writes the same history and lock as the online pull
// it replays.
func TestFrozenOfflineHistory(t *testing.T) {
	e := newEnv(t)
	e.publishDir(historyTree(t, "4.5.0", "4.5", "0.3.0"), "4.5.0", owner)
	e.publishDir(historyTree(t, "4.6.1", "4.6", "0.3.2"), "4.6.1", owner)
	e.publishDir(historyTree(t, "edge", "edge", "0.4.0"), "edge", owner)
	o := e.options(e.config(""))
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	want, _ := os.ReadFile(historyFile(o))
	wantLock, _ := os.ReadFile(o.Lock)
	golden, _ := os.ReadFile(filepath.Join("testdata", "history", "history.golden.json"))
	if !bytes.Equal(want, bytes.Replace(golden, []byte(`"tool": "0.3.0"`), []byte(`"tool": "0.1.0"`), 1)) {
		t.Fatalf("the pulled history differs from the local one:\n%s", want)
	}
	saved := filepath.Join(e.work, "saved.lock")
	_ = os.WriteFile(saved, wantLock, 0o600)
	_ = os.RemoveAll(o.Out)
	o.Frozen, o.Offline = saved, true
	e.stop()
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(historyFile(o))
	gotLock, _ := os.ReadFile(o.Lock)
	if !bytes.Equal(got, want) || !bytes.Equal(gotLock, wantLock) {
		t.Fatalf("offline history or lock differs:\n%s\n%s", got, gotLock)
	}
}

// A lock without history still validates; a lock entry missing a field
// does not.
func TestLockHistoryKeyIsOptional(t *testing.T) {
	base := `{"schema": "docs.opmodel.dev/lock/v1", "tool": "0.3.0", "config": "sha256:` + strings.Repeat("a", 64) + `", "bundles": []%s}`
	if _, err := schema.ValidateJSON("#Lock", "lock.json", []byte(strings.Replace(base, "%s", "", 1))); err != nil {
		t.Fatal(err)
	}
	with := `, "history": [{"project": "catalog-opm", "digest": "sha256:` + strings.Repeat("b", 64) + `", "path": "catalog-opm/history.json"}]`
	if _, err := schema.ValidateJSON("#Lock", "lock.json", []byte(strings.Replace(base, "%s", with, 1))); err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(with, `, "path": "catalog-opm/history.json"`, "", 1)
	if _, err := schema.ValidateJSON("#Lock", "lock.json", []byte(strings.Replace(base, "%s", bad, 1))); err == nil {
		t.Fatal("a history entry without a path validated")
	}
}

// Encode sorts history entries by project.
func TestLockSortsHistory(t *testing.T) {
	l := &Lock{Schema: LockSchema, Tool: "0.3.0", Config: "sha256:" + strings.Repeat("a", 64), History: []HistoryEntry{
		{Project: "zeta", Digest: "sha256:" + strings.Repeat("b", 64), Path: "zeta/history.json"},
		{Project: "alpha", Digest: "sha256:" + strings.Repeat("c", 64), Path: "alpha/history.json"},
	}}
	b, err := l.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Index(b, []byte("alpha")) > bytes.Index(b, []byte("zeta")) {
		t.Fatalf("lock:\n%s", b)
	}
	if l.History[0].Project != "zeta" {
		t.Fatal("Encode reordered the caller's entries")
	}
}

// A history that refuses a bundle's data fails the pull before anything
// is swapped in: the previous segments, history.json and lock stay as
// they were, and the error names the segment's data file.
func TestRefusedHistoryKeepsThePreviousPull(t *testing.T) {
	e := newEnv(t)
	o := e.options(e.config(""))
	e.stop()
	o.Locals = historyLocals(t)
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	snapshot := func() map[string]string {
		files := map[string]string{}
		_ = filepath.WalkDir(o.Out, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				b, _ := os.ReadFile(p)
				files[p] = string(b)
			}
			return err
		})
		return files
	}
	before := snapshot()
	bad := historyTree(t, "4.6.2", "4.6", "0.3.2")
	catalog := filepath.Join(bad, "data", "catalog.json")
	b, _ := os.ReadFile(catalog)
	_ = os.WriteFile(catalog, bytes.Replace(b, []byte(`"page": "traits/backup-v1alpha1"`), []byte(`"page": "../escape"`), 1), 0o600)
	o.Locals = []Local{o.Locals[0], {project, "4.6", bad}, o.Locals[2]}
	_, err := Run(context.Background(), o)
	if err == nil || IsUsage(err) || !strings.Contains(err.Error(), "catalog-opm 4.6 data/catalog.json") || strings.Contains(err.Error(), "report it") {
		t.Fatalf("err = %v", err)
	}
	after := snapshot()
	if len(after) != len(before) {
		t.Fatalf("files %d, want %d", len(after), len(before))
	}
	for p, want := range before {
		if after[p] != want {
			t.Errorf("%s changed", p)
		}
	}
	if left, _ := filepath.Glob(filepath.Join(o.Out, project, ".incoming-*")); len(left) != 0 {
		t.Fatalf("left %v", left)
	}
}

// A lock outside --out cannot record a history path; the pull says so
// before anything is swapped in.
func TestHistoryNeedsTheLockInOut(t *testing.T) {
	e := newEnv(t)
	o := e.options(e.config(""))
	e.stop()
	o.Lock = filepath.Join(e.work, "elsewhere", "lock.json")
	o.Locals = historyLocals(t)
	if _, err := Run(context.Background(), o); !IsUsage(err) || !strings.Contains(err.Error(), "must sit in --out") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(o.Out, project, "4.5")); !os.IsNotExist(err) {
		t.Fatal("a segment was swapped in")
	}
}
