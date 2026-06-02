//go:build windows

package appdata

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// 注册表存储常量：HKCU\Software\CS2HighlightTool, value name DataDir, REG_SZ.
const (
	registryRoot      = registry.CURRENT_USER
	registryKeyPath   = `Software\CS2HighlightTool`
	registryValueName = "DataDir"
)

// ReadDataDirFromRegistry 读取 HKCU\Software\CS2HighlightTool\DataDir。
// 若注册表 key 或 value 不存在则返回 error。
func ReadDataDirFromRegistry() (string, error) {
	key, err := registry.OpenKey(registryRoot, registryKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("读取注册表失败: %w", err)
	}
	defer key.Close()
	value, _, err := key.GetStringValue(registryValueName)
	if err != nil {
		return "", fmt.Errorf("读取注册表 DataDir 失败: %w", err)
	}
	return value, nil
}

// WriteDataDirToRegistry 写入 HKCU\Software\CS2HighlightTool\DataDir，
// 若 key 不存在则自动创建。
func WriteDataDirToRegistry(path string) error {
	key, _, err := registry.CreateKey(registryRoot, registryKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("创建注册表项失败: %w", err)
	}
	defer key.Close()
	if err := key.SetStringValue(registryValueName, path); err != nil {
		return fmt.Errorf("写入注册表 DataDir 失败: %w", err)
	}
	return nil
}

// DeleteDataDirFromRegistry 仅删除 DataDir value，保留子键。
// 若 key 或 value 已不存在则视为成功。
func DeleteDataDirFromRegistry() error {
	key, err := registry.OpenKey(registryRoot, registryKeyPath, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return fmt.Errorf("打开注册表项失败: %w", err)
	}
	defer key.Close()
	if err := key.DeleteValue(registryValueName); err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return fmt.Errorf("删除注册表 DataDir 失败: %w", err)
	}
	return nil
}
