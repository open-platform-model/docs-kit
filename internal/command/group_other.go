//go:build !unix

package command

import "os/exec"

// ownGroup leaves the default: exec kills the command alone.
func ownGroup(*exec.Cmd) {}
