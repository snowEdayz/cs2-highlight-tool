package producegame

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectPluginSearchPath_InjectsBeforeGameCsgo(t *testing.T) {
	content := "FileSystem\n{\n\tSearchPaths\n\t{\n\t\tGame\tcsgo\n\t}\n}\n"
	result, ok := InjectPluginSearchPath(content)
	if !ok {
		t.Fatal("expected successful injection, got false")
	}
	if !strings.Contains(result, "Game\tcsgo/plugin") {
		t.Fatalf("expected injected search path in:\n%s", result)
	}
	if !strings.Contains(result, "Game\tcsgo") {
		t.Fatalf("original Game csgo line should remain:\n%s", result)
	}
	idx1 := strings.Index(result, "Game\tcsgo/plugin")
	idx2 := strings.Index(result, "Game\tcsgo\n")
	if idx1 > idx2 {
		t.Fatalf("injected line should appear before original:\n%s", result)
	}
}

func TestInjectPluginSearchPath_NoopWhenAlreadyInjected(t *testing.T) {
	content := "Game\tcsgo/plugin\nGame\tcsgo\n"
	result, ok := InjectPluginSearchPath(content)
	if !ok {
		t.Fatal("expected ok=true when already injected")
	}
	if result != content {
		t.Fatalf("content should be unchanged when already injected, got:\n%s", result)
	}
}

func TestInjectPluginSearchPath_ReturnsFalseWhenNoInjectionPoint(t *testing.T) {
	content := "something unrelated"
	result, ok := InjectPluginSearchPath(content)
	if ok {
		t.Fatal("expected ok=false when no injection point")
	}
	if result != content {
		t.Fatalf("content should be unchanged on failure, got:\n%s", result)
	}
}

func TestResolveGameInfoPath_FindsFileRelativeToCS2Exe(t *testing.T) {
	root := t.TempDir()
	cs2Exe := filepath.Join(root, "game", "bin", "win64", "cs2.exe")
	if err := os.MkdirAll(filepath.Dir(cs2Exe), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(cs2Exe, []byte("exe"), 0644); err != nil {
		t.Fatalf("write cs2 exe: %v", err)
	}

	gameInfoPath := filepath.Join(root, "game", "csgo", "gameinfo.gi")
	if err := os.MkdirAll(filepath.Dir(gameInfoPath), 0755); err != nil {
		t.Fatalf("mkdir gameinfo: %v", err)
	}
	if err := os.WriteFile(gameInfoPath, []byte("gi"), 0644); err != nil {
		t.Fatalf("write gameinfo: %v", err)
	}

	got, err := ResolveGameInfoPath(cs2Exe, "")
	if err != nil {
		t.Fatalf("ResolveGameInfoPath: %v", err)
	}
	if got == "" {
		t.Fatal("expected non-empty path")
	}
}

func TestResolveGameInfoPath_ErrorWhenMissing(t *testing.T) {
	root := t.TempDir()
	cs2Exe := filepath.Join(root, "cs2.exe")
	if err := os.WriteFile(cs2Exe, []byte("exe"), 0644); err != nil {
		t.Fatalf("write cs2 exe: %v", err)
	}

	_, err := ResolveGameInfoPath(cs2Exe, "")
	if err == nil {
		t.Fatal("expected error when gameinfo.gi is missing")
	}
	if !strings.Contains(err.Error(), "gameinfo.gi") {
		t.Fatalf("error should mention gameinfo.gi, got: %v", err)
	}
}

func TestResolveGameInfoPath_UsesCS2DirHint(t *testing.T) {
	root := t.TempDir()
	cs2Exe := filepath.Join(root, "other", "cs2.exe")
	if err := os.MkdirAll(filepath.Dir(cs2Exe), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(cs2Exe, []byte("exe"), 0644); err != nil {
		t.Fatalf("write cs2 exe: %v", err)
	}

	gameRoot := t.TempDir()
	gameInfoPath := filepath.Join(gameRoot, "game", "csgo", "gameinfo.gi")
	if err := os.MkdirAll(filepath.Dir(gameInfoPath), 0755); err != nil {
		t.Fatalf("mkdir gameinfo: %v", err)
	}
	if err := os.WriteFile(gameInfoPath, []byte("gi"), 0644); err != nil {
		t.Fatalf("write gameinfo: %v", err)
	}

	got, err := ResolveGameInfoPath(cs2Exe, gameRoot)
	if err != nil {
		t.Fatalf("ResolveGameInfoPath: %v", err)
	}
	if got == "" {
		t.Fatal("expected non-empty path")
	}
}
