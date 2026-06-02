package appdata

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveExeOnly_ReturnsExeDirWithEmptyDataDir(t *testing.T) {
	exeDir := filepath.Join("C:", "Users", "me", "Downloads")

	got := ResolveExeOnly(exeDir)

	if got.ExeDir != filepath.Clean(exeDir) {
		t.Fatalf("ExeDir = %q, want %q", got.ExeDir, filepath.Clean(exeDir))
	}
	if got.DataDir != "" {
		t.Fatalf("DataDir = %q, want empty", got.DataDir)
	}
}

func TestResolveExeOnly_TrimsWhitespace(t *testing.T) {
	got := ResolveExeOnly("  /tmp/foo  ")
	if !strings.HasSuffix(got.ExeDir, "foo") {
		t.Fatalf("ExeDir = %q, want suffix 'foo'", got.ExeDir)
	}
}

func TestSamePath_TrueForEqualPaths(t *testing.T) {
	a := filepath.Join("a", "b", "c")
	b := filepath.Join("a", "b", "c")
	if !samePath(a, b) {
		t.Fatalf("samePath(%q, %q) = false, want true", a, b)
	}
}

func TestSamePath_FalseForDifferentPaths(t *testing.T) {
	if samePath("/a/b", "/a/c") {
		t.Fatalf("samePath of different paths should be false")
	}
}
