package appdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDataDirForWindowsUsesLocalAppData(t *testing.T) {
	exeDir := filepath.Join("C:", "Users", "me", "Downloads")
	localAppData := filepath.Join("C:", "Users", "me", "AppData", "Local")
	roamingAppData := filepath.Join("C:", "Users", "me", "AppData", "Roaming")

	got := defaultDataDirForGOOS("windows", localAppData, roamingAppData, exeDir)
	want := filepath.Join(localAppData, AppDataDirName)
	if got != want {
		t.Fatalf("default data dir = %q, want %q", got, want)
	}
}

func TestDefaultDataDirFallsBackToExecutableDir(t *testing.T) {
	exeDir := t.TempDir()

	got := defaultDataDirForGOOS("windows", "", "", exeDir)
	if got != exeDir {
		t.Fatalf("fallback data dir = %q, want %q", got, exeDir)
	}
}

func TestMigrateLegacyDataMovesKnownEntriesWithoutOverwriting(t *testing.T) {
	exeDir := t.TempDir()
	dataDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(exeDir, "config.json"), []byte(`{"fivee_player_name":"old"}`), 0o644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(exeDir, "hlae"), 0o755); err != nil {
		t.Fatalf("mkdir legacy hlae: %v", err)
	}
	if err := os.WriteFile(filepath.Join(exeDir, "hlae", "HLAE.exe"), []byte("legacy hlae"), 0o644); err != nil {
		t.Fatalf("write legacy hlae: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(exeDir, "plugin"), 0o755); err != nil {
		t.Fatalf("mkdir legacy plugin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(exeDir, "plugin", "server.dll"), []byte("legacy plugin"), 0o644); err != nil {
		t.Fatalf("write legacy plugin: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "plugin"), 0o755); err != nil {
		t.Fatalf("mkdir target plugin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "plugin", "server.dll"), []byte("target plugin"), 0o644); err != nil {
		t.Fatalf("write target plugin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(exeDir, "unrelated.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatalf("write unrelated file: %v", err)
	}

	if err := MigrateLegacyData(exeDir, dataDir); err != nil {
		t.Fatalf("migrate legacy data: %v", err)
	}

	assertFileContent(t, filepath.Join(dataDir, "config.json"), `{"fivee_player_name":"old"}`)
	assertFileContent(t, filepath.Join(dataDir, "hlae", "HLAE.exe"), "legacy hlae")
	assertFileContent(t, filepath.Join(dataDir, "plugin", "server.dll"), "target plugin")
	assertFileContent(t, filepath.Join(exeDir, "plugin", "server.dll"), "legacy plugin")
	assertFileContent(t, filepath.Join(exeDir, "unrelated.txt"), "keep")
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("%s content = %q, want %q", path, string(data), want)
	}
}
