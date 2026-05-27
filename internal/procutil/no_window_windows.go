//go:build windows

package procutil

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

func ConfigureNoWindowProcess(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}

	attr := cmd.SysProcAttr
	if attr == nil {
		attr = &syscall.SysProcAttr{}
	}
	attr.HideWindow = true
	attr.CreationFlags |= createNoWindow
	cmd.SysProcAttr = attr
}
