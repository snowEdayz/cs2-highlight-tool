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

func TestHasPluginSearchPathDetectsStandaloneLine(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "tab separated",
			content: "FileSystem\n{\n\t\tGame\tcsgo/plugin\n\t\tGame\tcsgo\n}\n",
			want:    true,
		},
		{
			name:    "space separated",
			content: "FileSystem\n{\n\t\tGame csgo/plugin\n\t\tGame\tcsgo\n}\n",
			want:    true,
		},
		{
			name:    "comment only",
			content: "// Game\tcsgo/plugin\nGame\tcsgo\n",
			want:    false,
		},
		{
			name:    "healthy",
			content: "Game\tcsgo\n",
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasPluginSearchPath(tt.content); got != tt.want {
				t.Fatalf("HasPluginSearchPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemovePluginSearchPathRemovesOnlyStandaloneInjectedLines(t *testing.T) {
	content := "FileSystem\n{\n\tSearchPaths\n\t{\n\t\t// Game\tcsgo/plugin\n\t\tGame\tcsgo/plugin\n\t\tGame csgo/plugin\n\t\tGame\tcsgo\n\t}\n}\n"

	result, changed := RemovePluginSearchPath(content)
	if !changed {
		t.Fatal("expected RemovePluginSearchPath to report changed=true")
	}
	if strings.Contains(result, "\t\tGame\tcsgo/plugin\n") {
		t.Fatalf("tab injected line should be removed:\n%s", result)
	}
	if strings.Contains(result, "\t\tGame csgo/plugin\n") {
		t.Fatalf("space injected line should be removed:\n%s", result)
	}
	if !strings.Contains(result, "// Game\tcsgo/plugin") {
		t.Fatalf("comment should remain:\n%s", result)
	}
	if !strings.Contains(result, "Game\tcsgo") {
		t.Fatalf("original csgo search path should remain:\n%s", result)
	}
}

func TestRemovePluginSearchPathNoopWhenHealthy(t *testing.T) {
	content := "FileSystem\n{\n\tSearchPaths\n\t{\n\t\tGame\tcsgo\n\t}\n}\n"

	result, changed := RemovePluginSearchPath(content)
	if changed {
		t.Fatal("expected changed=false for healthy content")
	}
	if result != content {
		t.Fatalf("healthy content should be unchanged, got:\n%s", result)
	}
}

func TestInjectSearchPath_InjectsArbitraryPathBeforeGameCsgo(t *testing.T) {
	content := "FileSystem\n{\n\tSearchPaths\n\t{\n\t\tGame\tcsgo\n\t}\n}\n"
	result, ok := InjectSearchPath(content, SearchPathPOV)
	if !ok {
		t.Fatal("expected successful injection, got false")
	}
	if !strings.Contains(result, "Game\tcsgo/pov.vpk") {
		t.Fatalf("expected injected pov search path in:\n%s", result)
	}
	if !strings.Contains(result, "Game\tcsgo") {
		t.Fatalf("original Game csgo line should remain:\n%s", result)
	}
	idx1 := strings.Index(result, "Game\tcsgo/pov.vpk")
	idx2 := strings.Index(result, "Game\tcsgo\n")
	if idx1 > idx2 {
		t.Fatalf("injected line should appear before original:\n%s", result)
	}
}

func TestInjectSearchPath_NoopWhenAlreadyInjected(t *testing.T) {
	content := "Game\tcsgo/pov.vpk\nGame\tcsgo\n"
	result, ok := InjectSearchPath(content, SearchPathPOV)
	if !ok {
		t.Fatal("expected ok=true when already injected")
	}
	if result != content {
		t.Fatalf("content should be unchanged when already injected, got:\n%s", result)
	}
}

func TestInjectSearchPath_ReturnsFalseWhenNoInjectionPoint(t *testing.T) {
	content := "something unrelated"
	result, ok := InjectSearchPath(content, SearchPathPOV)
	if ok {
		t.Fatal("expected ok=false when no injection point")
	}
	if result != content {
		t.Fatalf("content should be unchanged on failure, got:\n%s", result)
	}
}

func TestHasSearchPathDetectsStandaloneLine(t *testing.T) {
	tests := []struct {
		name       string
		searchPath string
		content    string
		want       bool
	}{
		{
			name:       "pov tab separated",
			searchPath: SearchPathPOV,
			content:    "FileSystem\n{\n\t\tGame\tcsgo/pov.vpk\n\t\tGame\tcsgo\n}\n",
			want:       true,
		},
		{
			name:       "pov space separated",
			searchPath: SearchPathPOV,
			content:    "FileSystem\n{\n\t\tGame csgo/pov.vpk\n\t\tGame\tcsgo\n}\n",
			want:       true,
		},
		{
			name:       "pov comment only ignored",
			searchPath: SearchPathPOV,
			content:    "// Game\tcsgo/pov.vpk\nGame\tcsgo\n",
			want:       false,
		},
		{
			name:       "plugin present but pov absent",
			searchPath: SearchPathPOV,
			content:    "Game\tcsgo/plugin\nGame\tcsgo\n",
			want:       false,
		},
		{
			name:       "healthy",
			searchPath: SearchPathPOV,
			content:    "Game\tcsgo\n",
			want:       false,
		},
		{
			name:       "plugin still detected via generic helper",
			searchPath: SearchPathPlugin,
			content:    "Game\tcsgo/plugin\nGame\tcsgo\n",
			want:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasSearchPath(tt.content, tt.searchPath); got != tt.want {
				t.Fatalf("HasSearchPath(%q) = %v, want %v", tt.searchPath, got, tt.want)
			}
		})
	}
}

func TestRemoveSearchPathRemovesOnlyMatchingEntries(t *testing.T) {
	// Content with both plugin and pov injected: removing pov must leave plugin
	// and the original csgo line intact.
	content := "FileSystem\n{\n\tSearchPaths\n\t{\n\t\t// Game\tcsgo/pov.vpk\n\t\tGame\tcsgo/pov.vpk\n\t\tGame csgo/pov.vpk\n\t\tGame\tcsgo/plugin\n\t\tGame\tcsgo\n\t}\n}\n"

	result, changed := RemoveSearchPath(content, SearchPathPOV)
	if !changed {
		t.Fatal("expected RemoveSearchPath(pov) to report changed=true")
	}
	if strings.Contains(result, "\t\tGame\tcsgo/pov.vpk\n") {
		t.Fatalf("tab pov injected line should be removed:\n%s", result)
	}
	if strings.Contains(result, "\t\tGame csgo/pov.vpk\n") {
		t.Fatalf("space pov injected line should be removed:\n%s", result)
	}
	if !strings.Contains(result, "// Game\tcsgo/pov.vpk") {
		t.Fatalf("comment should remain:\n%s", result)
	}
	if !strings.Contains(result, "Game\tcsgo/plugin") {
		t.Fatalf("plugin search path should remain when removing pov:\n%s", result)
	}
	if !strings.Contains(result, "Game\tcsgo") {
		t.Fatalf("original csgo search path should remain:\n%s", result)
	}
}

func TestRemoveSearchPathNoopWhenAbsent(t *testing.T) {
	content := "FileSystem\n{\n\tSearchPaths\n\t{\n\t\tGame\tcsgo/plugin\n\t\tGame\tcsgo\n\t}\n}\n"

	result, changed := RemoveSearchPath(content, SearchPathPOV)
	if changed {
		t.Fatal("expected changed=false when pov absent")
	}
	if result != content {
		t.Fatalf("content should be unchanged when target absent, got:\n%s", result)
	}
}

func TestKnownInjectedSearchPathsAreHandledByHelpers(t *testing.T) {
	// Regression guard: every path this tool may inject can be detected and
	// removed through the generic helpers.
	paths := []string{SearchPathPlugin, SearchPathPOV}
	base := "FileSystem\n{\n\tSearchPaths\n\t{\n\t\tGame\tcsgo\n\t}\n}\n"
	for _, p := range paths {
		injected, ok := InjectSearchPath(base, p)
		if !ok {
			t.Fatalf("InjectSearchPath(%q) failed", p)
		}
		if !HasSearchPath(injected, p) {
			t.Fatalf("HasSearchPath(%q) should detect injected content", p)
		}
		removed, changed := RemoveSearchPath(injected, p)
		if !changed {
			t.Fatalf("RemoveSearchPath(%q) should report changed", p)
		}
		if HasSearchPath(removed, p) {
			t.Fatalf("RemoveSearchPath(%q) should leave no residual", p)
		}
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
