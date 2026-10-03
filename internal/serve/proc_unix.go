//go:build unix

package serve

import (
	"errors"
	"syscall"
)

// alive reports whether a process with pid runs (or exists under another
// user, which counts as running).
func alive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
