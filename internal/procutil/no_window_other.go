//go:build !windows

package procutil

import "os/exec"

func ConfigureNoWindowProcess(cmd *exec.Cmd) {
	_ = cmd
}
