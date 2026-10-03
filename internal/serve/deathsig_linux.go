//go:build linux

package serve

import (
	"os/exec"
	"syscall"
)

// dieWithParent asks the kernel to send the child SIGTERM when its parent
// dies, so a killed serve does not leave hugo running. Best effort: the
// kernel ties the signal to the OS thread that started the child, not to
// the process (golang/go#27505), so it also fires if that thread exits
// first. Go keeps its threads unless a goroutine locked one, which serve
// never does; the interrupt serve sends on a clean stop does not depend on
// this.
func dieWithParent(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGTERM
}
