package serve

import (
	"os"
	"path/filepath"
	"strconv"
)

// tempPrefix names every serve run's temporary directory.
const tempPrefix = "opm-docs-serve-"

// ownerFile is the lock a run holds on its temporary directory for its
// whole life; it also records the run's pid, for a reader only.
const ownerFile = "owner.pid"

// makeTemp makes a run's temporary directory and locks it, after sweeping
// the directories of runs that died without removing theirs (a SIGKILL
// cannot be caught). release drops the lock; call it after removing the
// directory.
func makeTemp() (dir string, release func(), err error) {
	sweepStale(os.TempDir())
	dir, err = os.MkdirTemp("", tempPrefix+"*")
	if err != nil {
		return "", nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, ownerFile), os.O_CREATE|os.O_RDWR|os.O_EXCL, 0o600)
	if err == nil {
		err = lockOwner(f)
		if err == nil {
			_, err = f.WriteString(strconv.Itoa(os.Getpid()))
		}
		if err != nil {
			_ = f.Close()
		}
	}
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, err
	}
	return dir, func() { _ = f.Close() }, nil
}
