package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOutputsStorageStatsCountsVideosAndAllFileBytes(t *testing.T) {
	dataDir := t.TempDir()
	app := &App{exeDir: t.TempDir(), dataDir: dataDir}
	outputsDir := filepath.Join(dataDir, "outputs")
	writeTestFile(t, filepath.Join(outputsDir, "clip_a.mp4"), 10)
	writeTestFile(t, filepath.Join(outputsDir, "clip_b.MKV"), 20)
	writeTestFile(t, filepath.Join(outputsDir, "session", "take.wav"), 30)
	writeTestFile(t, filepath.Join(outputsDir, "session", "clip_c.mov"), 40)

	stats, err := app.GetOutputsStorageStats()
	if err != nil {
		t.Fatalf("GetOutputsStorageStats: %v", err)
	}

	if stats.OutputDir != outputsDir {
		t.Fatalf("OutputDir=%q want %q", stats.OutputDir, outputsDir)
	}
	if stats.VideoCount != 3 {
		t.Fatalf("VideoCount=%d want 3", stats.VideoCount)
	}
	if stats.TotalSizeBytes != 100 {
		t.Fatalf("TotalSizeBytes=%d want 100", stats.TotalSizeBytes)
	}
}

func TestClearOutputsDirectoryRemovesChildrenAndKeepsDirectory(t *testing.T) {
	dataDir := t.TempDir()
	app := &App{exeDir: t.TempDir(), dataDir: dataDir}
	outputsDir := filepath.Join(dataDir, "outputs")
	writeTestFile(t, filepath.Join(outputsDir, "clip_a.mp4"), 10)
	writeTestFile(t, filepath.Join(outputsDir, "batch", "clip_b.mp4"), 20)

	stats, err := app.ClearOutputsDirectory()
	if err != nil {
		t.Fatalf("ClearOutputsDirectory: %v", err)
	}

	if stats.OutputDir != outputsDir {
		t.Fatalf("OutputDir=%q want %q", stats.OutputDir, outputsDir)
	}
	if stats.VideoCount != 0 || stats.TotalSizeBytes != 0 {
		t.Fatalf("stats after clear mismatch: %+v", stats)
	}
	if info, err := os.Stat(outputsDir); err != nil {
		t.Fatalf("outputs directory should remain: %v", err)
	} else if !info.IsDir() {
		t.Fatalf("outputs path should be directory")
	}
	entries, err := os.ReadDir(outputsDir)
	if err != nil {
		t.Fatalf("ReadDir outputs: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("outputs directory should be empty, got %d entries", len(entries))
	}
}

func writeTestFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
