//go:build windows

package procutil

import (
	"os/exec"
	"syscall"
	"testing"
)

func TestConfigureNoWindowProcess_Windows(t *testing.T) {
	cmd := exec.Command("cmd.exe", "/c", "echo", "ok")
	ConfigureNoWindowProcess(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatalf("SysProcAttr should be set")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatalf("HideWindow should be true")
	}
	if cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatalf("CREATE_NO_WINDOW should be set")
	}
}

func TestConfigureNoWindowProcess_WindowsKeepExistingFlags(t *testing.T) {
	cmd := exec.Command("cmd.exe", "/c", "echo", "ok")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000010}

	ConfigureNoWindowProcess(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatalf("SysProcAttr should be set")
	}
	if cmd.SysProcAttr.CreationFlags&0x00000010 == 0 {
		t.Fatalf("existing creation flags should be preserved")
	}
	if cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatalf("CREATE_NO_WINDOW should be set")
	}
}

func TestConfigureNoWindowProcess_WindowsNilCmd(t *testing.T) {
	ConfigureNoWindowProcess(nil)
}
