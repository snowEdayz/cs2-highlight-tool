package updater

import (
	"cs2-highlight-tool-v2/internal/download"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func StartApply(dataDir string, assetURL string, latest string, downloadFile func(url, targetPath string) error) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	updateDir := filepath.Join(dataDir, "updates", sanitizeFileName(latest))
	if err := os.MkdirAll(updateDir, 0755); err != nil {
		return err
	}
	newExe := filepath.Join(updateDir, filepath.Base(exePath)+".new")
	if err := downloadFile(assetURL, newExe); err != nil {
		return fmt.Errorf("下载软件更新失败: %w", err)
	}

	updaterPath := filepath.Join(updateDir, "updater-"+filepath.Base(exePath))
	if err := download.CopyFile(exePath, updaterPath); err != nil {
		return fmt.Errorf("创建临时 updater 失败: %w", err)
	}

	cmd := exec.Command(updaterPath,
		"--apply-update",
		"--pid", strconv.Itoa(os.Getpid()),
		"--src", newExe,
		"--dst", exePath,
		"--restart",
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 updater 失败: %w", err)
	}
	return nil
}

func RunApplyMode(args []string) error {
	fs := flag.NewFlagSet("apply-update", flag.ContinueOnError)
	pid := fs.Int("pid", 0, "process id to wait for")
	src := fs.String("src", "", "new executable path")
	dst := fs.String("dst", "", "target executable path")
	restart := fs.Bool("restart", false, "restart target executable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *src == "" || *dst == "" {
		return fmt.Errorf("missing --src or --dst")
	}
	_ = pid
	return applyUpdateFiles(*src, *dst, *restart)
}

func applyUpdateFiles(src, dst string, restart bool) error {
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("new executable not found: %w", err)
	}
	oldPath := dst + ".old"
	var lastErr error
	for i := 0; i < 120; i++ {
		_ = os.Remove(oldPath)
		if err := os.Rename(dst, oldPath); err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if err := os.Rename(src, dst); err != nil {
			_ = os.Rename(oldPath, dst)
			return fmt.Errorf("replace executable failed: %w", err)
		}
		if restart {
			_ = exec.Command(dst).Start()
		}
		_ = os.Remove(oldPath)
		return nil
	}
	return fmt.Errorf("等待主程序退出后仍无法替换文件: %w", lastErr)
}

func sanitizeFileName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return replacer.Replace(value)
}
