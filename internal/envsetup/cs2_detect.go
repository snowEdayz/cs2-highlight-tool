package envsetup

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cs2-highlight-tool-v2/internal/config"
)

const steamCS2AppID = "730"

var (
	vdfPairLineRE     = regexp.MustCompile(`^\s*"([^"]+)"\s*"((?:\\.|[^"\\])*)"\s*$`)
	vdfKeyLineRE      = regexp.MustCompile(`^\s*"([^"]+)"\s*$`)
	appManifestDirRE  = regexp.MustCompile(`(?i)"installdir"\s*"((?:\\.|[^"\\])*)"`)
	vdfEscapeReplacer = strings.NewReplacer(`\\`, `\`, `\"`, `"`, `\t`, "\t", `\n`, "\n", `\r`, "\r")
)

func findCS2ExeFromSteamRoot(steamRoot string) (string, error) {
	steamRoot = config.CleanPath(steamRoot)
	if strings.TrimSpace(steamRoot) == "" {
		return "", fmt.Errorf("Steam 安装目录为空")
	}

	libraries, err := loadSteamLibraryRoots(steamRoot)
	if err != nil {
		return "", err
	}

	for _, library := range libraries {
		steamappsDir := filepath.Join(library, "steamapps")
		manifestPath := filepath.Join(steamappsDir, "appmanifest_"+steamCS2AppID+".acf")
		manifestData, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			continue
		}

		installDir, parseErr := parseInstallDirFromAppManifest(manifestData)
		if parseErr != nil {
			continue
		}
		candidate := filepath.Join(steamappsDir, "common", installDir, "game", "bin", "win64", "cs2.exe")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("未在 Steam 库中发现可用的 CS2 安装目录")
}

func loadSteamLibraryRoots(steamRoot string) ([]string, error) {
	steamRoot = config.CleanPath(steamRoot)
	locations := []string{
		filepath.Join(steamRoot, "config", "libraryfolders.vdf"),
		filepath.Join(steamRoot, "steamapps", "libraryfolders.vdf"),
	}

	var lastErr error
	for _, path := range locations {
		data, err := os.ReadFile(path)
		if err != nil {
			lastErr = err
			continue
		}
		parsed := parseSteamLibraryFoldersVDF(data)
		if len(parsed) == 0 {
			lastErr = fmt.Errorf("文件未包含有效库路径: %s", path)
			continue
		}
		if !containsPath(parsed, steamRoot) {
			parsed = append([]string{steamRoot}, parsed...)
		}
		return dedupePathsKeepOrder(parsed), nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("读取 Steam 库配置失败: %w", lastErr)
	}
	return nil, fmt.Errorf("读取 Steam 库配置失败")
}

func parseSteamLibraryFoldersVDF(data []byte) []string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	stack := make([]string, 0, 4)
	pending := ""
	pathsByIndex := make(map[string]string)
	order := make([]string, 0, 8)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
			if line == "" {
				continue
			}
		}

		switch line {
		case "{":
			if pending != "" {
				stack = append(stack, pending)
				pending = ""
			}
			continue
		case "}":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			pending = ""
			continue
		}

		if pair := vdfPairLineRE.FindStringSubmatch(line); pair != nil {
			key := pair[1]
			value := unescapeVDFString(pair[2])
			switch {
			case len(stack) == 1 && strings.EqualFold(stack[0], "libraryfolders") && isDigitsOnly(key):
				if _, exists := pathsByIndex[key]; !exists {
					order = append(order, key)
				}
				pathsByIndex[key] = config.CleanPath(value)
			case len(stack) == 2 && strings.EqualFold(stack[0], "libraryfolders") && isDigitsOnly(stack[1]) && strings.EqualFold(key, "path"):
				index := stack[1]
				if _, exists := pathsByIndex[index]; !exists {
					order = append(order, index)
				}
				pathsByIndex[index] = config.CleanPath(value)
			}
			continue
		}

		if key := vdfKeyLineRE.FindStringSubmatch(line); key != nil {
			pending = key[1]
		}
	}

	paths := make([]string, 0, len(order))
	for _, index := range order {
		path := strings.TrimSpace(pathsByIndex[index])
		if path == "" {
			continue
		}
		paths = append(paths, path)
	}
	return dedupePathsKeepOrder(paths)
}

func parseInstallDirFromAppManifest(data []byte) (string, error) {
	matches := appManifestDirRE.FindSubmatch(data)
	if len(matches) != 2 {
		return "", fmt.Errorf("appmanifest 缺少 installdir")
	}
	installDir := strings.TrimSpace(unescapeVDFString(string(matches[1])))
	if installDir == "" {
		return "", fmt.Errorf("appmanifest installdir 为空")
	}
	return installDir, nil
}

func unescapeVDFString(raw string) string {
	return vdfEscapeReplacer.Replace(raw)
}

func dedupePathsKeepOrder(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		cleaned := config.CleanPath(path)
		if cleaned == "" {
			continue
		}
		key := strings.ToLower(cleaned)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, cleaned)
	}
	return out
}

func containsPath(paths []string, expected string) bool {
	expected = strings.ToLower(config.CleanPath(expected))
	for _, path := range paths {
		if strings.ToLower(config.CleanPath(path)) == expected {
			return true
		}
	}
	return false
}

func isDigitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
