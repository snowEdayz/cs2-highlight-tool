//go:build windows

package envsetup

import (
	"fmt"
	"strings"

	"cs2-highlight-tool-v2/internal/config"

	"golang.org/x/sys/windows/registry"
)

func detectCS2ExeFromSteam() (string, error) {
	steamRoot, err := detectSteamRootFromRegistry()
	if err != nil {
		return "", err
	}
	return findCS2ExeFromSteamRoot(steamRoot)
}

func detectSteamRootFromRegistry() (string, error) {
	candidates := []struct {
		root      registry.Key
		path      string
		valueName string
	}{
		{root: registry.CURRENT_USER, path: `Software\Valve\Steam`, valueName: "SteamPath"},
		{root: registry.LOCAL_MACHINE, path: `SOFTWARE\WOW6432Node\Valve\Steam`, valueName: "InstallPath"},
		{root: registry.LOCAL_MACHINE, path: `SOFTWARE\Valve\Steam`, valueName: "InstallPath"},
	}

	var lastErr error
	for _, candidate := range candidates {
		value, err := readRegistryString(candidate.root, candidate.path, candidate.valueName)
		if err != nil {
			lastErr = err
			continue
		}
		value = config.CleanPath(value)
		if strings.TrimSpace(value) == "" {
			continue
		}
		return value, nil
	}
	if lastErr != nil {
		return "", fmt.Errorf("读取 Steam 注册表失败: %w", lastErr)
	}
	return "", fmt.Errorf("读取 Steam 注册表失败")
}

func readRegistryString(root registry.Key, keyPath, valueName string) (string, error) {
	key, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()
	value, _, err := key.GetStringValue(valueName)
	if err != nil {
		return "", err
	}
	return value, nil
}
