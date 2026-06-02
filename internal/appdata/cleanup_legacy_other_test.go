//go:build !windows

package appdata

import "testing"

// TestCleanupLegacyData_NoopOnNonWindows 非 Windows 平台始终返回 nil。
func TestCleanupLegacyData_NoopOnNonWindows(t *testing.T) {
	if err := CleanupLegacyData("/tmp/nonexistent"); err != nil {
		t.Fatalf("expected nil on non-windows, got: %v", err)
	}
	if err := CleanupLegacyData(""); err != nil {
		t.Fatalf("expected nil on non-windows for empty input, got: %v", err)
	}
}
