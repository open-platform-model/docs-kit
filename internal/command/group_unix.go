//go:build unix

package command

import (
	"os/exec"
	"syscall"
)

// ownGroup runs the command in a process group of its own and, on timeout
// or cancellation, kills the whole group, so a child it started (a
// compiled program under go run) does not outlive it.
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
