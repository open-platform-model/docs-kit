//go:build unix

package serve

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// lockOwner takes the run's exclusive lock on its owner file. A flock is
// released by the kernel when the process dies, however it dies, and it
// holds across pid namespaces sharing one /tmp (toolbox), where a pid
// says nothing.
func lockOwner(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) //nolint:gosec // a file descriptor fits an int
}

// sweepStale removes each serve directory under tmp that no running serve
// holds: a real directory (not a symbolic link) owned by this user with
// mode 0700, holding an owner file whose lock can be taken. Anything else
// is left alone.
func sweepStale(tmp string) {
	entries, err := os.ReadDir(tmp)
	if err != nil {
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), tempPrefix) {
			sweepOne(filepath.Join(tmp, e.Name()))
		}
	}
}

func sweepOne(dir string) {
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&(fs.ModeSymlink|fs.ModePerm) != 0o700 || !ownedByUs(info) {
		return
	}
	owner := filepath.Join(dir, ownerFile)
	if fi, err := os.Lstat(owner); err != nil || !fi.Mode().IsRegular() {
		return
	}
	f, err := os.OpenFile(owner, os.O_RDWR, 0)
	if err != nil {
		return
	}
	defer f.Close()
	if syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil { //nolint:gosec // a file descriptor fits an int
		return // a running serve holds it
	}
	_ = os.RemoveAll(dir)
}

func ownedByUs(info fs.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(st.Uid) == os.Getuid()
}
