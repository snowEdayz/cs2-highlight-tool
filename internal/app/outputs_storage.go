package app

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
)

type OutputsStorageStats struct {
	OutputDir      string `json:"output_dir"`
	VideoCount     int    `json:"video_count"`
	TotalSizeBytes int64  `json:"total_size_bytes"`
}

func (a *App) GetOutputsStorageStats() (*OutputsStorageStats, error) {
	return a.outputsStorageStats()
}

func (a *App) ClearOutputsDirectory() (*OutputsStorageStats, error) {
	outputDir := a.fixedRecordOutputDir()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, fmt.Errorf("读取输出目录失败: %w", err)
	}
	for _, entry := range entries {
		childPath := filepath.Join(outputDir, entry.Name())
		if err := os.RemoveAll(childPath); err != nil {
			return nil, fmt.Errorf("清理输出目录失败: %w", err)
		}
	}
	return a.outputsStorageStats()
}

func (a *App) OpenOutputsDirectory() error {
	outputDir := a.fixedRecordOutputDir()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", outputDir)
	case "darwin":
		cmd = exec.Command("open", outputDir)
	default:
		cmd = exec.Command("xdg-open", outputDir)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("打开输出目录失败: %w", err)
	}
	return nil
}

func (a *App) outputsStorageStats() (*OutputsStorageStats, error) {
	outputDir := a.fixedRecordOutputDir()
	stats := &OutputsStorageStats{OutputDir: outputDir}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	err := filepath.WalkDir(outputDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		stats.TotalSizeBytes += info.Size()
		if isOutputVideoFile(path) {
			stats.VideoCount++
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("统计输出目录失败: %w", err)
	}
	return stats, nil
}

func isOutputVideoFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".mov", ".mkv", ".avi":
		return true
	default:
		return false
	}
}
