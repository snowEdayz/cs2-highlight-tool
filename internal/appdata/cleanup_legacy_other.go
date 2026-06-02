//go:build !windows

package appdata

// CleanupLegacyData 在非 Windows 平台为 no-op。
func CleanupLegacyData(exeDir string) error {
	return nil
}
