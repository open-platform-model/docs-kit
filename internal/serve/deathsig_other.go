//go:build !linux

package serve

import "os/exec"

// dieWithParent has no kernel support here; serve stops its children
// itself on an interrupt, SIGTERM or SIGHUP.
func dieWithParent(*exec.Cmd) {}
