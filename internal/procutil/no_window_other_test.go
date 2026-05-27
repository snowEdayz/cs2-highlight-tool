//go:build !windows

package procutil

import (
	"os/exec"
	"testing"
)

func TestConfigureNoWindowProcess_NonWindows(t *testing.T) {
	cmd := exec.Command("echo", "ok")
	ConfigureNoWindowProcess(cmd)
}

func TestConfigureNoWindowProcess_NonWindowsNilCmd(t *testing.T) {
	ConfigureNoWindowProcess(nil)
}
