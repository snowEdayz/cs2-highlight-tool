//go:build windows

package app

import (
	"os/exec"
	"testing"
)

func TestConfigureNoWindowProcess_Windows(t *testing.T) {
	cmd := exec.Command("cmd.exe", "/c", "echo", "ok")
	configureNoWindowProcess(cmd)
	if cmd.SysProcAttr == nil {
		t.Fatalf("SysProcAttr should be set")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatalf("HideWindow should be true")
	}
	if cmd.SysProcAttr.CreationFlags&0x08000000 == 0 {
		t.Fatalf("CREATE_NO_WINDOW should be set")
	}
}
