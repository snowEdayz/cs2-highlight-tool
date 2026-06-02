//go:build windows

package appdata

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CleanupLegacyData 清理旧版本遗留的应用数据：
//  1. 删除 LOCALAPPDATA 下 `<UserCacheDir>/CS2 Highlight Tool` 旧路径
//  2. 遍历 exeDir 下已知 legacy children，逐个 RemoveAll
//
// 任一子项失败仅返回聚合错误，不中断后续清理。
func CleanupLegacyData(exeDir string) error {
	var errs []string

	// 1) 清理 LOCALAPPDATA 兜底
	if cacheDir, err := os.UserCacheDir(); err == nil && strings.TrimSpace(cacheDir) != "" {
		legacyRoot := filepath.Join(cacheDir, AppDataDirName)
		if err := os.RemoveAll(legacyRoot); err != nil {
			errs = append(errs, fmt.Sprintf("清理 %s 失败: %v", legacyRoot, err))
		}
	}

	// 2) 清理 exeDir 下 legacy children
	exeDir = filepath.Clean(strings.TrimSpace(exeDir))
	if exeDir != "" {
		for _, name := range []string{"config.json", "hlae", "plugin", "ffmpeg", "updates", "demo", "projects", "outputs", "logs"} {
			target := filepath.Join(exeDir, name)
			if err := os.RemoveAll(target); err != nil {
				errs = append(errs, fmt.Sprintf("清理 %s 失败: %v", target, err))
			}
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
