//go:build !windows

package app

import (
	"os/exec"
	"testing"
)

func TestConfigureNoWindowProcess_NonWindows(t *testing.T) {
	cmd := exec.Command("echo", "ok")
	configureNoWindowProcess(cmd)
}
