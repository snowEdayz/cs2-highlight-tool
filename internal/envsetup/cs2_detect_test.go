package envsetup

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestParseSteamLibraryFoldersVDF_NewFormat(t *testing.T) {
	content := `
"libraryfolders"
{
	"0"
	{
		"path" "C:\\Program Files (x86)\\Steam"
		"apps"
		{
			"730" "1"
		}
	}
	"1"
	{
		"path" "D:\\SteamLibrary"
	}
}
`
	paths := parseSteamLibraryFoldersVDF([]byte(content))
	if len(paths) != 2 {
		t.Fatalf("paths count = %d, want 2", len(paths))
	}
	if paths[0] != `C:\Program Files (x86)\Steam` {
		t.Fatalf("paths[0] = %q", paths[0])
	}
	if paths[1] != `D:\SteamLibrary` {
		t.Fatalf("paths[1] = %q", paths[1])
	}
}

func TestParseSteamLibraryFoldersVDF_OldFormat(t *testing.T) {
	content := `
"libraryfolders"
{
	"TimeNextStatsReport" "0"
	"0" "C:\\Program Files (x86)\\Steam"
	"1" "D:\\SteamLibrary"
}
`
	paths := parseSteamLibraryFoldersVDF([]byte(content))
	if len(paths) != 2 {
		t.Fatalf("paths count = %d, want 2", len(paths))
	}
	if paths[0] != `C:\Program Files (x86)\Steam` {
		t.Fatalf("paths[0] = %q", paths[0])
	}
	if paths[1] != `D:\SteamLibrary` {
		t.Fatalf("paths[1] = %q", paths[1])
	}
}

func TestParseInstallDirFromAppManifest(t *testing.T) {
	content := `
"AppState"
{
	"appid" "730"
	"installdir" "Counter-Strike Global Offensive"
}
`
	dir, err := parseInstallDirFromAppManifest([]byte(content))
	if err != nil {
		t.Fatalf("parseInstallDirFromAppManifest: %v", err)
	}
	if dir != "Counter-Strike Global Offensive" {
		t.Fatalf("installdir = %q", dir)
	}
}

func TestFindCS2ExeFromSteamRoot_PrefersLibraryOrder(t *testing.T) {
	steamRoot := t.TempDir()
	libA := filepath.Join(steamRoot, "libraryA")
	libB := filepath.Join(steamRoot, "libraryB")

	writeLibraryFoldersForTest(t, filepath.Join(steamRoot, "config", "libraryfolders.vdf"), []string{libA, libB})
	writeInstalledCS2ForTest(t, libA, "CS2-A")
	writeInstalledCS2ForTest(t, libB, "CS2-B")

	got, err := findCS2ExeFromSteamRoot(steamRoot)
	if err != nil {
		t.Fatalf("findCS2ExeFromSteamRoot: %v", err)
	}
	want := filepath.Join(libA, "steamapps", "common", "CS2-A", "game", "bin", "win64", "cs2.exe")
	if got != want {
		t.Fatalf("cs2 exe = %q, want %q", got, want)
	}
}

func TestFindCS2ExeFromSteamRoot_ReturnsErrorWhenMissing(t *testing.T) {
	steamRoot := t.TempDir()
	libA := filepath.Join(steamRoot, "libraryA")
	writeLibraryFoldersForTest(t, filepath.Join(steamRoot, "config", "libraryfolders.vdf"), []string{libA})

	_, err := findCS2ExeFromSteamRoot(steamRoot)
	if err == nil {
		t.Fatal("expected error when cs2 is missing")
	}
}

func writeLibraryFoldersForTest(t *testing.T, path string, libraries []string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir libraryfolders dir: %v", err)
	}
	content := "\"libraryfolders\"\n{\n"
	for idx, library := range libraries {
		escaped := escapeVDFPathForTest(library)
		content += "\t\"" + strconv.Itoa(idx) + "\"\n"
		content += "\t{\n"
		content += "\t\t\"path\" \"" + escaped + "\"\n"
		content += "\t}\n"
	}
	content += "}\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write libraryfolders: %v", err)
	}
}

func writeInstalledCS2ForTest(t *testing.T, libraryRoot string, installDir string) {
	t.Helper()
	manifestPath := filepath.Join(libraryRoot, "steamapps", "appmanifest_730.acf")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		t.Fatalf("mkdir steamapps: %v", err)
	}
	manifest := "\"AppState\"\n{\n\t\"appid\" \"730\"\n\t\"installdir\" \"" + installDir + "\"\n}\n"
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("write appmanifest: %v", err)
	}

	cs2Exe := filepath.Join(libraryRoot, "steamapps", "common", installDir, "game", "bin", "win64", "cs2.exe")
	if err := os.MkdirAll(filepath.Dir(cs2Exe), 0755); err != nil {
		t.Fatalf("mkdir cs2 dir: %v", err)
	}
	if err := os.WriteFile(cs2Exe, []byte("cs2"), 0644); err != nil {
		t.Fatalf("write cs2 exe: %v", err)
	}
}

func escapeVDFPathForTest(path string) string {
	path = filepath.Clean(path)
	path = filepath.ToSlash(path)
	return path
}
