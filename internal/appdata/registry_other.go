//go:build !windows

package appdata

import "errors"

// errRegistryUnsupported 用于非 Windows 平台兜底，让上层走 UserConfigDir。
var errRegistryUnsupported = errors.New("registry not supported on this platform")

// ReadDataDirFromRegistry 在非 Windows 平台返回 errRegistryUnsupported。
func ReadDataDirFromRegistry() (string, error) {
	return "", errRegistryUnsupported
}

// WriteDataDirToRegistry 在非 Windows 平台返回 errRegistryUnsupported。
func WriteDataDirToRegistry(path string) error {
	return errRegistryUnsupported
}

// DeleteDataDirFromRegistry 在非 Windows 平台返回 nil（无注册表可清理）。
func DeleteDataDirFromRegistry() error {
	return nil
}
