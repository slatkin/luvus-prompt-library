// detach_windows.go — start an edit-external editor detached on Windows
// (design D15). This file exists only so that the program compiles there.
//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

const detachedProcess = 0x00000008
const createNewProcessGroup = 0x00000200

func setDetach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: detachedProcess | createNewProcessGroup,
	}
}
