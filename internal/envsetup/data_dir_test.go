package envsetup

import (
	"path/filepath"
	"testing"
)

func TestNewWithDataDirUsesDataDirForManagedPaths(t *testing.T) {
	exeDir := t.TempDir()
	dataDir := t.TempDir()

	svc := NewWithDataDir(exeDir, dataDir, "1.0.0")

	if svc.exeDir != exeDir {
		t.Fatalf("exeDir = %q, want %q", svc.exeDir, exeDir)
	}
	if svc.dataDir != dataDir {
		t.Fatalf("dataDir = %q, want %q", svc.dataDir, dataDir)
	}
	if svc.configPath != filepath.Join(dataDir, "config.json") {
		t.Fatalf("configPath = %q, want dataDir config", svc.configPath)
	}

	state := svc.GetStartupState()
	hlae := findStepByID(state.Steps, componentHLAE)
	if hlae == nil || hlae.Path != filepath.Join(dataDir, "hlae", "HLAE.exe") {
		t.Fatalf("hlae path = %#v, want under dataDir", hlae)
	}
	plugin := findStepByID(state.Steps, componentPlugin)
	if plugin == nil || plugin.Path != filepath.Join(dataDir, "plugin", "server.dll") {
		t.Fatalf("plugin path = %#v, want under dataDir", plugin)
	}
	ffmpeg := findStepByID(state.Steps, componentFFmpeg)
	if ffmpeg == nil || ffmpeg.Path != filepath.Join(dataDir, "ffmpeg", "bin", "ffmpeg.exe") {
		t.Fatalf("ffmpeg path = %#v, want under dataDir", ffmpeg)
	}
}
