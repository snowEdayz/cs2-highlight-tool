//go:build !windows

package app

import (
	"cs2-highlight-tool-v2/internal/procutil"
	"os/exec"
)

func configureNoWindowProcess(cmd *exec.Cmd) {
	procutil.ConfigureNoWindowProcess(cmd)
}
