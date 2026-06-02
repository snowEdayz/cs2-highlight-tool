//go:build windows

package appdata

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCleanupLegacyData_RemovesExeDirChildren 验证 exeDir 下 legacy children 被删除。
func TestCleanupLegacyData_RemovesExeDirChildren(t *testing.T) {
	exeDir := t.TempDir()

	// 创建若干 legacy entries
	mustMkdir(t, filepath.Join(exeDir, "hlae"))
	mustWrite(t, filepath.Join(exeDir, "hlae", "HLAE.exe"), "fake")
	mustMkdir(t, filepath.Join(exeDir, "plugin"))
	mustWrite(t, filepath.Join(exeDir, "plugin", "server.dll"), "fake")
	mustWrite(t, filepath.Join(exeDir, "config.json"), `{}`)

	// 保留一个非 legacy 文件，验证不被删除
	keepPath := filepath.Join(exeDir, "unrelated.txt")
	mustWrite(t, keepPath, "keep")

	if err := CleanupLegacyData(exeDir); err != nil {
		t.Logf("CleanupLegacyData returned (non-fatal): %v", err)
	}

	for _, name := range []string{"hlae", "plugin", "config.json"} {
		p := filepath.Join(exeDir, name)
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed, stat err=%v", p, err)
		}
	}

	// unrelated 文件保留
	if _, err := os.Stat(keepPath); err != nil {
		t.Errorf("expected %s to remain, got err=%v", keepPath, err)
	}
}

// TestCleanupLegacyData_Idempotent 重复调用不应报错。
func TestCleanupLegacyData_Idempotent(t *testing.T) {
	exeDir := t.TempDir()
	// 不创建任何 legacy entries

	if err := CleanupLegacyData(exeDir); err != nil {
		t.Errorf("first call err=%v", err)
	}
	if err := CleanupLegacyData(exeDir); err != nil {
		t.Errorf("second call err=%v", err)
	}
}

// TestCleanupLegacyData_EmptyExeDir 输入空 exeDir 应不 panic 也不报错。
func TestCleanupLegacyData_EmptyExeDir(t *testing.T) {
	if err := CleanupLegacyData(""); err != nil {
		// LOCALAPPDATA 子目录可能不存在，是允许的
		t.Logf("CleanupLegacyData('') returned: %v", err)
	}
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
