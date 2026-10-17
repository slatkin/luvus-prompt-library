// detach_other.go — start an edit-external editor detached on every platform
// except Windows (design D15). Setsid detaches the child from the caller's
// session, matching Node's `detached: true`.
//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func setDetach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
