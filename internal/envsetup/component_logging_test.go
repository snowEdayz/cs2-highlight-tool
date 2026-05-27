package envsetup

import (
	"os"
	"path/filepath"
	"testing"

	"cs2-highlight-tool-v2/internal/config"
)

func TestInstallHLAEFromArchive_LogsStages(t *testing.T) {
	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)

	zipPath := createZipForTest(t, map[string][]byte{
		"release/hlae.exe":               []byte("hlae"),
		"release/x64/AfxHookSource2.dll": []byte("hook"),
		"release/changelog.xml":          []byte("<changelog><version>2.0.0</version></changelog>"),
		"release/other/readme.txt":       []byte("doc"),
	})
	if err := svc.installHLAEFromArchive(zipPath); err != nil {
		t.Fatalf("installHLAEFromArchive error: %v", err)
	}
	logs := svc.logsSnapshot()
	for _, item := range []struct {
		stage  string
		action string
	}{
		{stage: "extract", action: "unzip"},
		{stage: "validate", action: "verify_archive"},
		{stage: "persist_config", action: "write"},
		{stage: "ready", action: "component_ready"},
	} {
		if !hasStructuredLog(logs, componentHLAE, item.stage, item.action) {
			t.Fatalf("missing hlae log stage=%s action=%s", item.stage, item.action)
		}
	}
}

func TestInstallFFmpegFromArchive_LogsStages(t *testing.T) {
	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)

	zipPath := createZipForTest(t, map[string][]byte{
		"package/bin/ffmpeg.exe": []byte("ffmpeg"),
	})
	if err := svc.installFFmpegFromArchive(zipPath); err != nil {
		t.Fatalf("installFFmpegFromArchive error: %v", err)
	}
	logs := svc.logsSnapshot()
	for _, item := range []struct {
		stage  string
		action string
	}{
		{stage: "extract", action: "unarchive"},
		{stage: "validate", action: "verify_archive"},
		{stage: "persist_config", action: "write"},
		{stage: "ready", action: "component_ready"},
	} {
		if !hasStructuredLog(logs, componentFFmpeg, item.stage, item.action) {
			t.Fatalf("missing ffmpeg log stage=%s action=%s", item.stage, item.action)
		}
	}
}

func TestEnsureCS2Path_LogsStages(t *testing.T) {
	exeDir := t.TempDir()
	svc := New(exeDir, "1.0.0")
	svc.Startup(nil)

	cs2Root := filepath.Join(exeDir, "cs2")
	cs2Path := filepath.Join(cs2Root, "game", "bin", "win64", "cs2.exe")
	if err := os.MkdirAll(filepath.Dir(cs2Path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cs2Path, []byte("cs2"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.persistConfig(func(next *config.Config) error {
		next.CS2Dir = cs2Root
		next.CS2Exe = ""
		return nil
	}); err != nil {
		t.Fatalf("persist config error: %v", err)
	}

	if err := svc.ensureCS2Path(); err != nil {
		t.Fatalf("ensureCS2Path error: %v", err)
	}
	logs := svc.logsSnapshot()
	for _, item := range []struct {
		stage  string
		action string
	}{
		{stage: "check", action: "resolve_path"},
		{stage: "persist_config", action: "write"},
		{stage: "ready", action: "component_ready"},
	} {
		if !hasStructuredLog(logs, componentCS2, item.stage, item.action) {
			t.Fatalf("missing cs2 log stage=%s action=%s", item.stage, item.action)
		}
	}
}
