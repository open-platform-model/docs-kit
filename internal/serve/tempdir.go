package serve

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// tempPrefix names every serve run's temporary directory.
const tempPrefix = "opm-docs-serve-"

// ownerFile holds the pid of the run that owns a temporary directory.
const ownerFile = "owner.pid"

// makeTemp makes a run's temporary directory and records the run's pid
// in it, after sweeping the directories of runs that died without
// removing theirs (a SIGKILL cannot be caught).
func makeTemp() (string, error) {
	sweepStale(os.TempDir())
	dir, err := os.MkdirTemp("", tempPrefix+"*")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, ownerFile), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

// sweepStale removes each serve directory under tmp whose owner is no
// longer running. A directory without a readable owner is left alone.
func sweepStale(tmp string) {
	entries, err := os.ReadDir(tmp)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), tempPrefix) {
			continue
		}
		dir := filepath.Join(tmp, e.Name())
		b, err := os.ReadFile(filepath.Join(dir, ownerFile))
		if err != nil {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
		if err != nil || pid <= 0 || pid == os.Getpid() || alive(pid) {
			continue
		}
		_ = os.RemoveAll(dir)
	}
}
