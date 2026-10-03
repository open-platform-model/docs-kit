//go:build linux

package serve

import (
	"os/exec"
	"syscall"
)

// dieWithParent asks the kernel to send the child SIGTERM when serve
// dies, however it dies, so a killed serve never leaves hugo running.
func dieWithParent(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGTERM
}
