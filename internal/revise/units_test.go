package revise

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/verify"
	"github.com/open-platform-model/docs-kit/internal/verify/sigtest"
)

// docsKitUnits is the demo config with one unit, read from the repository's
// release manifest.
var docsKitUnits = strings.Replace(docsKit, "\tsources: [", "\tunits: \"opm-controller-module\": key: \"mod\"\n\tsources: [", 1)

// setUnits commits a new module version on main.
func (e *env) setUnits(version string) {
	e.t.Helper()
	e.r.Write(map[string]string{".release-please-manifest.json": "{\"mod\": \"" + version + "\"}\n"})
	e.r.CommitAt(fixDateA, "release the module")
	e.syncMain()
}

func (e *env) refresh() (*Result, *bundle.Manifest, error) {
	e.t.Helper()
	out := filepath.Join(e.t.TempDir(), "out")
	res, err := Run(context.Background(), Options{
		Repo: e.r.Dir, Project: project, Tag: tag, RefreshUnits: true, Out: out,
		Registry: e.registry, Tool: e.tool, Client: e.client,
		Verifier: func() (*verify.Verifier, error) { return e.auth.Verifier(e.t), nil }, Policy: sigtest.Policy(owner),
	})
	if err != nil {
		return nil, nil, err
	}
	m, err := bundle.Read(res.Dir)
	if err != nil {
		e.t.Fatal(err)
	}
	return res, m, nil
}

func (e *env) refreshRefused(want string) {
	e.t.Helper()
	if _, _, err := e.refresh(); err == nil || !strings.Contains(err.Error(), want) {
		e.t.Fatalf("refresh: %v, want an error with %q", err, want)
	}
}

func TestRefreshUnits(t *testing.T) {
	e := newEnvWith(t, docsKitUnits)
	e.publishRelease()
	e.refreshRefused("units already current in 1.2.3.0")

	// A module release: revision 1 has the same pages and the new unit.
	e.setUnits("0.2.0")
	before := e.tags()
	res, m, err := e.refresh()
	if err != nil {
		t.Fatal(err)
	}
	if res.Revision != 1 || len(res.Patches) != 0 || m.Revision != 1 || m.Version != "1.2.3" || m.Units["opm-controller-module"] != "0.2.0" || len(m.Source.Patches) != 0 || m.Source.Commit != e.release {
		t.Fatalf("result %+v manifest %+v", res, m)
	}
	if got := e.tags(); strings.Join(got, ",") != strings.Join(before, ",") {
		t.Fatalf("refresh wrote to the registry: %v -> %v", before, got)
	}
	e.publishBuilt(res.Dir)
	e.refreshRefused("units already current in 1.2.3.1")

	refreshThenFix(t, e)
}

// refreshThenFix runs a fix and another refresh on top of a published
// refresh revision 1: the fix keeps the refreshed units, the refresh keeps
// the fix.
func refreshThenFix(t *testing.T, e *env) {
	t.Helper()
	// A fix on top of a refreshed revision keeps the refreshed units.
	fixA := e.fix(fixDateA, map[string][2]string{landing: {"Every member is listed below.", "Every member is listed here."}})
	fm, dir := e.built(fixA, 2, fixA)
	if fm.Units["opm-controller-module"] != "0.2.0" {
		t.Fatalf("a fix revision lost the refreshed units: %v", fm.Units)
	}
	e.publishBuilt(dir)

	// A refresh on top of a fix keeps the fix.
	e.setUnits("0.3.0")
	res, m, err := e.refresh()
	if err != nil {
		t.Fatal(err)
	}
	if res.Revision != 3 || len(m.Source.Patches) != 1 || m.Source.Patches[0] != fixA || m.Units["opm-controller-module"] != "0.3.0" {
		t.Fatalf("result %+v manifest %+v", res, m)
	}
	if !strings.Contains(e.page(res.Dir, "_index.md"), "listed here") {
		t.Fatal("the refreshed revision lost the fix")
	}
}

func TestRefreshUnitsRefusals(t *testing.T) {
	e := newEnv(t) // the demo config declares no units
	e.publishRelease()
	e.refreshRefused("declares no units")

	e = newEnvWith(t, docsKitUnits)
	e.refreshRefused("publish the release first")
	e.publishRelease()
	e.setUnits("0.2.0")
	res, _, err := e.refresh()
	if err != nil {
		t.Fatal(err)
	}
	// Pushed and never signed, the units still those of main: nothing to refresh.
	first := e.push(res.Dir).Digest
	e.refreshRefused("units already current in 1.2.3.1")

	// Another module release on top of an unsigned revision is refused.
	e.setUnits("0.3.0")
	e.refreshRefused("1.2.3.1 (" + first + ") is not signed by docs-kit's publish workflow")

	// Signed and promoted, the module moved again: a new revision.
	e.sign(first)
	e.promote(first)
	if res, _, err = e.refresh(); err != nil || res.Revision != 2 {
		t.Fatalf("after promote: %+v, %v", res, err)
	}
}

// The release tag's own docs-kit.cue declares no units, as the live
// controller tags do; main's does. The refresh still records them, and a
// fix after it carries them on.
func TestRefreshUnitsTagWithoutUnits(t *testing.T) {
	e := newEnv(t) // the tag declares no units
	e.publishRelease()
	e.r.Write(map[string]string{"docs-kit.cue": docsKitUnits})
	e.r.CommitAt(fixDateA, "declare the unit on main")
	e.syncMain()
	res, m, err := e.refresh()
	if err != nil {
		t.Fatal(err)
	}
	if res.Revision != 1 || m.Units["opm-controller-module"] != "0.1.1" {
		t.Fatalf("result %+v units %v", res, m.Units)
	}
	e.publishBuilt(res.Dir)
	e.refreshRefused("units already current in 1.2.3.1")
	fixA := e.fix(fixDateB, map[string][2]string{landing: {"Every member is listed below.", "Every member is listed here."}})
	fm, _ := e.built(fixA, 2, fixA)
	if fm.Units["opm-controller-module"] != "0.1.1" {
		t.Fatalf("a fix revision lost the units: %v", fm.Units)
	}
}
