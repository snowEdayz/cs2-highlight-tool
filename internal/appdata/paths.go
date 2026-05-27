package appdata

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"cs2-highlight-tool-v2/internal/download"
)

const AppDataDirName = "CS2 Highlight Tool"

type Paths struct {
	ExeDir  string
	DataDir string
}

func Resolve(exeDir string) Paths {
	exeDir = filepath.Clean(strings.TrimSpace(exeDir))
	return Paths{
		ExeDir:  exeDir,
		DataDir: DefaultDataDir(exeDir),
	}
}

func DefaultDataDir(exeDir string) string {
	cacheDir, _ := os.UserCacheDir()
	configDir, _ := os.UserConfigDir()
	return defaultDataDirForGOOS(runtime.GOOS, cacheDir, configDir, exeDir)
}

func defaultDataDirForGOOS(goos string, cacheDir string, configDir string, exeDir string) string {
	exeDir = filepath.Clean(strings.TrimSpace(exeDir))
	cacheDir = strings.TrimSpace(cacheDir)
	configDir = strings.TrimSpace(configDir)
	if goos == "windows" {
		if cacheDir != "" {
			return filepath.Join(cacheDir, AppDataDirName)
		}
		return exeDir
	}
	if configDir != "" {
		return filepath.Join(configDir, AppDataDirName)
	}
	return exeDir
}

func MigrateLegacyData(exeDir string, dataDir string) error {
	exeDir = filepath.Clean(strings.TrimSpace(exeDir))
	dataDir = filepath.Clean(strings.TrimSpace(dataDir))
	if exeDir == "" || dataDir == "" || samePath(exeDir, dataDir) {
		return nil
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("创建应用数据目录失败: %w", err)
	}
	for _, name := range legacyDataEntries() {
		src := filepath.Join(exeDir, name)
		dst := filepath.Join(dataDir, name)
		if err := migrateEntry(src, dst); err != nil {
			return fmt.Errorf("迁移旧数据 %s 失败: %w", name, err)
		}
	}
	return nil
}

func legacyDataEntries() []string {
	return []string{
		"config.json",
		"hlae",
		"plugin",
		"ffmpeg",
		"updates",
		"demo",
		"projects",
		"outputs",
		"logs",
	}
}

func migrateEntry(src string, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode()); err != nil {
			return err
		}
		return download.CopyDirContents(src, dst)
	}
	return download.CopyFile(src, dst)
}

func samePath(a string, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
