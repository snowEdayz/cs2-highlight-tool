package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type MapOverviewMeta struct {
	PosX  float64 `json:"pos_x"`
	PosY  float64 `json:"pos_y"`
	Scale float64 `json:"scale"`
}

type Map2DRenderData struct {
	MapName   string  `json:"map_name"`
	PosX      float64 `json:"pos_x"`
	PosY      float64 `json:"pos_y"`
	Scale     float64 `json:"scale"`
	ImageData string  `json:"image_data"`
}

type mapExtractorRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func (a *App) GetMap2DRenderData(mapName string) (*Map2DRenderData, error) {
	if a.config == nil {
		return nil, fmt.Errorf("配置未加载")
	}
	if a.exeDir == "" {
		return nil, fmt.Errorf("程序目录未初始化")
	}

	normalizedMap := normalizeMapName(mapName)
	if normalizedMap == "" {
		return nil, fmt.Errorf("地图名称为空")
	}

	if err := ensureMapAssets(a.exeDir, a.config, normalizedMap); err != nil {
		return nil, err
	}

	mapsDir := filepath.Join(a.exeDir, "assets", "maps")
	mapDataPath := filepath.Join(mapsDir, "map-data.json")
	allMapData, err := readMapDataFile(mapDataPath)
	if err != nil {
		return nil, err
	}

	meta, ok := allMapData[normalizedMap]
	if !ok || meta.Scale == 0 {
		return nil, fmt.Errorf("地图元数据缺失: %s", normalizedMap)
	}

	imagePath := filepath.Join(mapsDir, normalizedMap+".png")
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("地图图片缺失: %s", imagePath)
	}

	return &Map2DRenderData{
		MapName:   normalizedMap,
		PosX:      meta.PosX,
		PosY:      meta.PosY,
		Scale:     meta.Scale,
		ImageData: "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageBytes),
	}, nil
}

func ensureMapAssets(exeDir string, cfg *Config, mapName string) error {
	mapsDir := filepath.Join(exeDir, "assets", "maps")
	mapImagePath := filepath.Join(mapsDir, mapName+".png")
	mapDataPath := filepath.Join(mapsDir, "map-data.json")

	if _, err := os.Stat(mapImagePath); err == nil {
		if mapData, readErr := readMapDataFile(mapDataPath); readErr == nil {
			if meta, ok := mapData[mapName]; ok && meta.Scale != 0 {
				return nil
			}
		}
	}

	if err := extractMapAssetsFromLocalExtractor(exeDir, cfg); err != nil {
		return err
	}

	if _, err := os.Stat(mapImagePath); err != nil {
		return fmt.Errorf("地图图片缺失: %s", mapImagePath)
	}
	mapData, err := readMapDataFile(mapDataPath)
	if err != nil {
		return err
	}
	meta, ok := mapData[mapName]
	if !ok || meta.Scale == 0 {
		return fmt.Errorf("地图元数据缺失: %s", mapName)
	}
	return nil
}

func extractMapAssetsFromLocalExtractor(exeDir string, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("配置未加载")
	}

	extractorExe, err := ensureMapExtractorAvailable(exeDir)
	if err != nil {
		return err
	}

	pakPath, err := resolvePak01DirVPK(cfg.CS2Exe)
	if err != nil {
		return err
	}

	mapsDir := filepath.Join(exeDir, "assets", "maps")
	if err := os.MkdirAll(mapsDir, 0755); err != nil {
		return fmt.Errorf("创建地图目录失败: %w", err)
	}

	tempDir := filepath.Join(exeDir, "_map_extractor_temp")
	_ = os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时提取目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	printInfo("正在提取地图雷达图...")
	if err := runMapExtractorCommand(
		extractorExe,
		"extract-radar",
		"--pak", pakPath,
		"--out", mapsDir,
	); err != nil {
		return err
	}

	printInfo("正在提取地图 overview 元数据...")
	if err := runMapExtractorCommand(
		extractorExe,
		"extract-overviews",
		"--pak", pakPath,
		"--out", tempDir,
	); err != nil {
		return err
	}

	parsedMapData, err := parseOverviewFiles(tempDir)
	if err != nil {
		return err
	}

	mapDataPath := filepath.Join(mapsDir, "map-data.json")
	existingMapData, _ := readMapDataFile(mapDataPath)
	if existingMapData == nil {
		existingMapData = map[string]MapOverviewMeta{}
	}
	for name, meta := range parsedMapData {
		existingMapData[name] = meta
	}

	if err := writeMapDataFile(mapDataPath, existingMapData); err != nil {
		return err
	}

	printSuccess("地图资源提取完成")
	return nil
}

func ensureMapExtractorAvailable(exeDir string) (string, error) {
	extractorExe := filepath.Join(exeDir, "tools", "cs2-map-extractor.exe")
	if _, err := os.Stat(extractorExe); err == nil {
		return extractorExe, nil
	}

	printWarning("未找到地图提取工具，开始自动下载...")
	if err := downloadMapExtractor(exeDir, extractorExe); err != nil {
		return "", err
	}
	if _, err := os.Stat(extractorExe); err != nil {
		return "", fmt.Errorf("地图提取工具下载后仍不存在: %s", extractorExe)
	}
	printSuccess("地图提取工具已准备就绪")
	return extractorExe, nil
}

func downloadMapExtractor(exeDir, targetPath string) error {
	release, err := getLatestMapExtractorRelease()
	if err != nil {
		return err
	}
	if release == nil || len(release.Assets) == 0 {
		return fmt.Errorf("地图提取工具 release 资源为空")
	}

	var assetURL string
	for _, asset := range release.Assets {
		name := strings.ToLower(strings.TrimSpace(asset.Name))
		if name == "cs2-map-extractor.exe" {
			assetURL = asset.BrowserDownloadURL
			break
		}
	}
	if assetURL == "" {
		for _, asset := range release.Assets {
			name := strings.ToLower(strings.TrimSpace(asset.Name))
			if strings.Contains(name, "cs2-map-extractor") && strings.HasSuffix(name, ".exe") {
				assetURL = asset.BrowserDownloadURL
				break
			}
		}
	}
	if assetURL == "" {
		return fmt.Errorf("release 中未找到可用的 cs2-map-extractor.exe 资产")
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("创建 tools 目录失败: %w", err)
	}

	tempPath := filepath.Join(exeDir, "cs2-map-extractor_temp.exe")
	_ = os.Remove(tempPath)
	_ = os.Remove(targetPath)

	printInfo("正在下载地图提取工具...")
	if err := downloadFile(assetURL, tempPath); err != nil {
		return fmt.Errorf("下载地图提取工具失败: %w", err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		return fmt.Errorf("保存地图提取工具失败: %w", err)
	}
	return nil
}

func getLatestMapExtractorRelease() (*mapExtractorRelease, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	apiURL := mapExtractorReleaseAPIURLGitHub
	if isChinaIP() {
		apiURL = mapExtractorReleaseAPIURLGitee
	}

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建地图提取工具版本请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "CS2-Highlight-Tool")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求地图提取工具版本失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("地图提取工具版本 API 返回异常状态码: %d", resp.StatusCode)
	}

	var release mapExtractorRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析地图提取工具版本信息失败: %w", err)
	}
	return &release, nil
}

func runMapExtractorCommand(exePath string, args ...string) error {
	cmd := execCommandHidden(exePath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputText := strings.TrimSpace(string(output))
		if outputText == "" {
			return fmt.Errorf("地图提取工具执行失败: %w", err)
		}
		if len(outputText) > 1000 {
			outputText = outputText[:1000] + "..."
		}
		return fmt.Errorf("地图提取工具执行失败: %w, 输出: %s", err, outputText)
	}
	return nil
}

func collectRadarImages(sourceDir, mapsDir string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info == nil || info.IsDir() {
			return nil
		}

		name := strings.ToLower(info.Name())
		if !strings.HasSuffix(name, "_radar_psd.png") {
			return nil
		}
		if strings.Contains(name, "_preview") || strings.Contains(name, "_vanity") {
			return nil
		}

		base := strings.TrimSuffix(name, "_radar_psd.png")
		mapName := normalizeMapName(base)
		if mapName == "" {
			return nil
		}

		dst := filepath.Join(mapsDir, mapName+".png")
		return copyFile(path, dst)
	})
}

func parseOverviewFiles(rootDir string) (map[string]MapOverviewMeta, error) {
	result := map[string]MapOverviewMeta{}
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info == nil || info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(info.Name())) != ".txt" {
			return nil
		}

		meta, ok, err := parseOverviewFile(path)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		mapName := normalizeMapName(strings.TrimSuffix(info.Name(), filepath.Ext(info.Name())))
		if mapName == "" {
			return nil
		}
		result[mapName] = meta
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("解析 overview txt 失败: %w", err)
	}
	return result, nil
}

func parseOverviewFile(path string) (MapOverviewMeta, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MapOverviewMeta{}, false, fmt.Errorf("读取 overview 文件失败: %w", err)
	}

	values := map[string]float64{}
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || line == "{" || line == "}" || strings.HasPrefix(line, "//") {
			continue
		}

		line = strings.ReplaceAll(line, "\t", " ")
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.Trim(strings.ToLower(fields[0]), "\"")
		if key != "pos_x" && key != "pos_y" && key != "scale" {
			continue
		}

		valueText := strings.Trim(fields[1], "\"")
		value, parseErr := strconv.ParseFloat(valueText, 64)
		if parseErr != nil {
			continue
		}
		values[key] = value
	}

	if _, ok := values["pos_x"]; !ok {
		return MapOverviewMeta{}, false, nil
	}
	if _, ok := values["pos_y"]; !ok {
		return MapOverviewMeta{}, false, nil
	}
	if _, ok := values["scale"]; !ok {
		return MapOverviewMeta{}, false, nil
	}

	return MapOverviewMeta{
		PosX:  values["pos_x"],
		PosY:  values["pos_y"],
		Scale: values["scale"],
	}, true, nil
}

func readMapDataFile(path string) (map[string]MapOverviewMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取地图元数据失败: %w", err)
	}

	result := map[string]MapOverviewMeta{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("解析地图元数据失败: %w", err)
	}
	return result, nil
}

func writeMapDataFile(path string, data map[string]MapOverviewMeta) error {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化地图元数据失败: %w", err)
	}
	if err := os.WriteFile(path, jsonBytes, 0644); err != nil {
		return fmt.Errorf("写入地图元数据失败: %w", err)
	}
	return nil
}

func resolvePak01DirVPK(cs2Exe string) (string, error) {
	cs2Exe = cleanPath(cs2Exe)
	if cs2Exe == "" {
		return "", fmt.Errorf("CS2 路径为空")
	}
	if _, err := os.Stat(cs2Exe); err != nil {
		return "", fmt.Errorf("CS2 可执行文件不存在: %s", cs2Exe)
	}

	cs2Dir := filepath.Dir(cs2Exe)
	candidates := []string{
		filepath.Join(cs2Dir, "..", "..", "csgo", "pak01_dir.vpk"),
		filepath.Join(cs2Dir, "..", "csgo", "pak01_dir.vpk"),
		filepath.Join(cs2Dir, "csgo", "pak01_dir.vpk"),
	}
	for _, c := range candidates {
		absPath, _ := filepath.Abs(c)
		if _, err := os.Stat(absPath); err == nil {
			return absPath, nil
		}
	}

	return "", fmt.Errorf("未找到 pak01_dir.vpk，请确认 CS2.exe 路径正确")
}
