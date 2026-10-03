//go:build unix

package serve

import (
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
)

// staleDir makes a serve directory under tmp with an unlocked owner file.
func staleDir(t *testing.T, tmp, name string, mode os.FileMode) string {
	t.Helper()
	dir := filepath.Join(tmp, name)
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, ownerFile), strconv.Itoa(999999))
	if err := os.Chmod(dir, mode); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSweepStale(t *testing.T) {
	tmp := t.TempDir()
	dead := staleDir(t, tmp, tempPrefix+"dead", 0o700)
	foreign := staleDir(t, tmp, tempPrefix+"foreign-mode", 0o755)
	target := staleDir(t, tmp, "target", 0o700)
	link := filepath.Join(tmp, tempPrefix+"link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	noOwner := filepath.Join(tmp, tempPrefix+"no-owner")
	if err := os.Mkdir(noOwner, 0o700); err != nil {
		t.Fatal(err)
	}
	other := staleDir(t, tmp, "other-dir", 0o700)

	// A running serve holds its owner file's lock.
	held := staleDir(t, tmp, tempPrefix+"held", 0o700)
	f, err := os.OpenFile(filepath.Join(held, ownerFile), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}

	sweepStale(tmp)
	for _, d := range []string{foreign, target, link, noOwner, other, held} {
		if _, err := os.Lstat(d); err != nil {
			t.Errorf("%s was removed", filepath.Base(d))
		}
	}
	if _, err := os.Lstat(dead); err == nil {
		t.Error("the dead run's directory survived")
	}
}

func TestMakeTempHoldsItsLock(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	dir, release, err := makeTemp()
	if err != nil {
		t.Fatal(err)
	}
	sweepStale(os.TempDir())
	if _, err := os.Stat(dir); err != nil {
		t.Fatal("a live run's directory was swept")
	}
	release()
	sweepStale(os.TempDir())
	if _, err := os.Stat(dir); err == nil {
		t.Fatal("a released directory survived the sweep")
	}
}
