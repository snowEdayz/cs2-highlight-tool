package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx        context.Context
	exeDir     string
	configPath string
	config     *Config
	previewMu  sync.Mutex
	previewURL string
}

type DemoInfo struct {
	DemoPath string          `json:"demo_path"`
	Players  []PlayerSummary `json:"players"`
}

type PlayerSummary struct {
	Name       string         `json:"name"`
	SteamID    string         `json:"steam_id"`
	EntityID   int            `json:"entity_id"`
	TotalKills int            `json:"total_kills"`
	Rounds     []RoundSummary `json:"rounds"`
}

type RoundSummary struct {
	Round int        `json:"round"`
	Kills []KillInfo `json:"kills"`
}

type RecordRequest struct {
	DemoPath       string `json:"demo_path"`
	PlayerSteamID  string `json:"player_steam_id"`
	SelectedRounds []int  `json:"selected_rounds"`
	AutoMode       bool   `json:"auto_mode"`
	DebugMode      bool   `json:"debug_mode"`
}

type RecordResult struct {
	CfgPath    string `json:"cfg_path"`
	OutputPath string `json:"output_path"`
}

type UsageStats struct {
	Run  int `json:"run"`
	Make int `json:"make"`
}

type CountResponse struct {
	Counts int `json:"counts"`
}

func NewApp() *App {
	exePath, err := os.Executable()
	if err != nil {
		return &App{}
	}
	exeDir := filepath.Dir(exePath)
	return &App{
		exeDir:     exeDir,
		configPath: filepath.Join(exeDir, "config.json"),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	appCtx = ctx

	if a.exeDir == "" {
		exePath, err := os.Executable()
		if err == nil {
			a.exeDir = filepath.Dir(exePath)
			a.configPath = filepath.Join(a.exeDir, "config.json")
		}
	}

	if err := cleanupOutputDirectories(a.exeDir); err != nil {
		printWarning(fmt.Sprintf("清理 outputs 目录失败: %v", err))
	}

	// 启动时仅初始化路径与上下文，环境准备交由前端触发
	printSuccess("启动完成，等待环境初始化")
}

func (a *App) shutdown(ctx context.Context) {
	if a.exeDir == "" {
		exePath, err := os.Executable()
		if err == nil {
			a.exeDir = filepath.Dir(exePath)
		}
	}
	if err := cleanupOutputDirectories(a.exeDir); err != nil {
		printWarning(fmt.Sprintf("清理 outputs 目录失败: %v", err))
	}
}

func cleanupOutputDirectories(exeDir string) error {
	if exeDir == "" {
		return nil
	}
	outputDir := filepath.Join(exeDir, "outputs")
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if err := os.RemoveAll(filepath.Join(outputDir, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func fetchStatsJSON(path string, out interface{}) error {
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest(http.MethodGet, statsBaseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("stats request failed: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (a *App) GetUsageStats() (*UsageStats, error) {
	var stats UsageStats
	if err := fetchStatsJSON("/stats", &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (a *App) IncrementRunCount() (*CountResponse, error) {
	var res CountResponse
	if err := fetchStatsJSON("/run", &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (a *App) IncrementMakeCount() (*CountResponse, error) {
	var res CountResponse
	if err := fetchStatsJSON("/make", &res); err != nil {
		return nil, err
	}
	return &res, nil
}

type releaseAsset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
}

type releaseInfo struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
	HTMLURL string         `json:"html_url"`
}

func getCurrentVersion() string {
	if len(wailsConfigData) == 0 {
		return "0.0.0"
	}
	var data struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(wailsConfigData, &data); err != nil {
		return "0.0.0"
	}
	if data.Version == "" {
		return "0.0.0"
	}
	return data.Version
}

func compareVersions(current, latest string) int {
	parse := func(v string) []int {
		v = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V"))
		parts := strings.Split(v, ".")
		out := make([]int, 0, len(parts))
		for _, p := range parts {
			n, err := strconv.Atoi(p)
			if err != nil {
				n = 0
			}
			out = append(out, n)
		}
		return out
	}
	a := parse(current)
	b := parse(latest)
	max := len(a)
	if len(b) > max {
		max = len(b)
	}
	for len(a) < max {
		a = append(a, 0)
	}
	for len(b) < max {
		b = append(b, 0)
	}
	for i := 0; i < max; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func (a *App) fetchLatestRelease() (*releaseInfo, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	apiURL := updateAPIURLGitHub
	if isChinaIP() {
		apiURL = updateAPIURLGitee
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CS2-Highlight-Tool")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("更新检查失败: %d", resp.StatusCode)
	}

	var info releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &info, nil
}

type UpdateInfo struct {
	Available bool   `json:"available"`
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	URL       string `json:"url"`
}

func (a *App) GetUpdateInfo() (*UpdateInfo, error) {
	current := getCurrentVersion()
	info, err := a.fetchLatestRelease()
	if err != nil || info == nil {
		return &UpdateInfo{Available: false, Current: current}, nil
	}
	latest := info.TagName
	if latest == "" {
		return &UpdateInfo{Available: false, Current: current}, nil
	}
	if compareVersions(current, latest) >= 0 {
		return &UpdateInfo{Available: false, Current: current, Latest: latest}, nil
	}

	downloadURL := info.HTMLURL
	for _, asset := range info.Assets {
		if asset.BrowserDownloadURL != "" {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return &UpdateInfo{Available: false, Current: current, Latest: latest}, nil
	}

	return &UpdateInfo{
		Available: true,
		Current:   current,
		Latest:    latest,
		URL:       downloadURL,
	}, nil
}

func loadStoredHLAEVersion(configPath string) string {
	if configPath == "" {
		return ""
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	var cfg struct {
		HLAEVersion string `json:"hlae_version"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	return cfg.HLAEVersion
}

func (a *App) PrepareEnvironment(debugMode bool) (*Config, error) {
	baseCfg := buildBaseConfig(a.exeDir)
	baseCfg.HLAEVersion = loadStoredHLAEVersion(a.configPath)

	if containsCJK(a.exeDir) {
		return nil, fmt.Errorf("程序路径包含中文: %s", a.exeDir)
	}

	// 先准备 FFmpeg/HLAE，再读取/修复配置
	if err := ensureFFmpegAvailable(a.exeDir, baseCfg, ""); err != nil {
		return nil, err
	}
	if err := checkAndUpdateHLAE(a.exeDir, baseCfg, "", debugMode); err != nil {
		return nil, err
	}
	if err := setupFFmpegIni(a.exeDir, baseCfg); err != nil {
		printWarning(fmt.Sprintf("设置 ffmpeg.ini 失败: %v", err))
	}

	cfg := a.loadConfigAfterEnv(baseCfg)

	os.MkdirAll(cfg.CfgDir, 0755)
	os.MkdirAll(cfg.OutputDir, 0755)
	if err := saveConfig(a.configPath, cfg); err != nil {
		return nil, err
	}

	a.config = cfg
	return a.config, nil
}

func (a *App) loadConfigAfterEnv(baseCfg *Config) *Config {
	if baseCfg == nil {
		baseCfg = buildBaseConfig(a.exeDir)
	}

	cfg := baseCfg
	if _, err := os.Stat(a.configPath); err == nil {
		if loaded, loadErr := loadConfig(a.configPath); loadErr == nil {
			cfg = loaded
		} else {
			printWarning(fmt.Sprintf("读取配置失败，将重新生成: %v", loadErr))
		}
	}

	applyConfigDefaults(cfg, baseCfg)
	return cfg
}

func applyConfigDefaults(cfg *Config, baseCfg *Config) {
	if cfg == nil || baseCfg == nil {
		return
	}

	cfg.CS2Exe = cleanPath(cfg.CS2Exe)
	if cfg.CS2Exe != "" {
		if _, err := os.Stat(cfg.CS2Exe); err != nil {
			cfg.CS2Exe = ""
		}
	}

	cfg.HLAEExe = baseCfg.HLAEExe
	cfg.FFmpegDir = baseCfg.FFmpegDir
	cfg.CfgDir = baseCfg.CfgDir

	if cfg.OutputDir == "" {
		cfg.OutputDir = baseCfg.OutputDir
	} else {
		cfg.OutputDir = cleanPath(cfg.OutputDir)
	}

	if cfg.RecordFPS <= 0 {
		cfg.RecordFPS = baseCfg.RecordFPS
	}
	if cfg.Tickrate <= 0 {
		cfg.Tickrate = baseCfg.Tickrate
	}
	if cfg.KillerPreSeconds <= 0 {
		cfg.KillerPreSeconds = baseCfg.KillerPreSeconds
	}
	if cfg.KillerPostSeconds <= 0 {
		cfg.KillerPostSeconds = baseCfg.KillerPostSeconds
	}
	if cfg.VictimPreSeconds <= 0 {
		cfg.VictimPreSeconds = baseCfg.VictimPreSeconds
	}
	if cfg.VictimPostSeconds <= 0 {
		cfg.VictimPostSeconds = baseCfg.VictimPostSeconds
	}
	if cfg.VideoPreset == "" {
		cfg.VideoPreset = baseCfg.VideoPreset
	}
	if cfg.TransitionDuration < 0 {
		cfg.TransitionDuration = baseCfg.TransitionDuration
	}
	if cfg.TransitionType == "" {
		cfg.TransitionType = baseCfg.TransitionType
	}
	if cfg.LaunchResolution != "4:3" && cfg.LaunchResolution != "16:9" {
		cfg.LaunchResolution = baseCfg.LaunchResolution
	}

	if baseCfg.HLAEVersion != "" {
		cfg.HLAEVersion = baseCfg.HLAEVersion
	}
}

func (a *App) GetConfig() (*Config, error) {
	if a.config == nil {
		cfg := buildBaseConfig(a.exeDir)
		a.config = cfg
		return a.config, nil
	}
	return a.config, nil
}

func (a *App) SaveConfig(cfg Config) (*Config, error) {
	cfg.CS2Exe = cleanPath(cfg.CS2Exe)
	cfg.HLAEExe = cleanPath(cfg.HLAEExe)
	cfg.FFmpegDir = cleanPath(cfg.FFmpegDir)
	cfg.CfgDir = cleanPath(cfg.CfgDir)
	cfg.OutputDir = cleanPath(cfg.OutputDir)

	if cfg.FFmpegDir == "" && a.config != nil {
		cfg.FFmpegDir = a.config.FFmpegDir
	}

	if err := saveConfig(a.configPath, &cfg); err != nil {
		return nil, err
	}
	a.config = &cfg
	return a.config, nil
}

func (a *App) OpenVideoExternal(path string) error {
	if path == "" {
		return fmt.Errorf("视频路径为空")
	}
	path = cleanPath(path)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("视频文件不存在: %s", path)
	}

	switch goruntime.GOOS {
	case "windows":
		return execCommandHidden("cmd", "/c", "start", "", path).Start()
	case "darwin":
		return execCommandHidden("open", path).Start()
	default:
		return execCommandHidden("xdg-open", path).Start()
	}
}

func (a *App) GetVideoPreviewURL(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("视频路径为空")
	}
	path = cleanPath(path)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("视频文件不存在: %s", path)
	}
	if !strings.HasSuffix(strings.ToLower(path), ".mp4") {
		return "", fmt.Errorf("仅支持 mp4 预览")
	}

	baseURL, err := a.ensurePreviewServer()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/video?path=%s", baseURL, url.QueryEscape(path)), nil
}

func (a *App) PickDemo() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 Demo 文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "Demo Files (*.dem)", Pattern: "*.dem"},
		},
	})
}

func (a *App) PickCS2Exe() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 CS2 可执行文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "Executable (*.exe)", Pattern: "*.exe"},
		},
	})
}

func (a *App) PickHLAEExe() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 HLAE.exe",
		Filters: []runtime.FileFilter{
			{DisplayName: "Executable (*.exe)", Pattern: "*.exe"},
		},
	})
}

func (a *App) PickOutputDir() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择输出目录",
	})
}

func (a *App) CheckAndUpdateHLAE(debugMode bool) error {
	if a.config == nil {
		return fmt.Errorf("配置未加载")
	}
	return checkAndUpdateHLAE(a.exeDir, a.config, a.configPath, debugMode)
}

func (a *App) SetupFFmpegIni() error {
	if a.config == nil {
		return fmt.Errorf("配置未加载")
	}
	return setupFFmpegIni(a.exeDir, a.config)
}

func (a *App) CheckEnvironment() error {
	if a.config == nil {
		return fmt.Errorf("配置未加载")
	}
	return checkEnvironment(a.exeDir, a.config, a.configPath)
}

func (a *App) CheckHLAEEnvironment() error {
	if a.config == nil {
		return fmt.Errorf("配置未加载")
	}

	printTitle("\nHLAE 环境检查")

	if err := ensureFFmpegAvailable(a.exeDir, a.config, a.configPath); err != nil {
		return err
	}

	if err := setupFFmpegIni(a.exeDir, a.config); err != nil {
		printWarning(fmt.Sprintf("设置 ffmpeg.ini 失败: %v", err))
	}

	if _, err := os.Stat(a.config.HLAEExe); err != nil {
		printError("✗ HLAE 可执行文件不存在")
		fmt.Printf("  路径: %s\n", a.config.HLAEExe)
		return fmt.Errorf("HLAE 未找到")
	}
	printSuccess("✓ HLAE.exe")

	hookDll := filepath.Join(filepath.Dir(a.config.HLAEExe), "x64", "AfxHookSource2.dll")
	if _, err := os.Stat(hookDll); err != nil {
		printError("✗ AfxHookSource2.dll 不存在")
		fmt.Printf("  路径: %s\n", hookDll)
		return fmt.Errorf("HLAE 组件不完整")
	}
	printSuccess("✓ AfxHookSource2.dll")

	ffmpegExe := resolveFFmpegExe(a.exeDir, a.config)
	if _, err := os.Stat(ffmpegExe); err != nil {
		printError("✗ FFmpeg 可执行文件不存在")
		fmt.Printf("  路径: %s\n", ffmpegExe)
		return fmt.Errorf("FFmpeg 未找到")
	}
	printSuccess("✓ FFmpeg")

	return nil
}

func (a *App) ParseDemo(demoPath string) (*DemoInfo, error) {
	if demoPath == "" {
		return nil, fmt.Errorf("demo 路径为空")
	}
	if _, err := os.Stat(demoPath); err != nil {
		return nil, fmt.Errorf("demo 文件不存在: %s", demoPath)
	}

	printTitle("\n解析 Demo")
	players, killsByPlayer, err := parseDemoKills(demoPath)
	if err != nil {
		return nil, err
	}
	if len(players) == 0 {
		return nil, fmt.Errorf("未找到玩家信息")
	}

	playerList := make([]*PlayerInfo, 0, len(players))
	for _, p := range players {
		playerList = append(playerList, p)
	}
	sort.Slice(playerList, func(i, j int) bool {
		return playerList[i].Name < playerList[j].Name
	})

	result := &DemoInfo{
		DemoPath: demoPath,
		Players:  make([]PlayerSummary, 0, len(playerList)),
	}

	for _, p := range playerList {
		kills := killsByPlayer[int(p.SteamID)]
		roundKills := make(map[int][]KillInfo)
		for _, k := range kills {
			roundKills[k.Round] = append(roundKills[k.Round], k)
		}

		rounds := make([]int, 0, len(roundKills))
		for r := range roundKills {
			rounds = append(rounds, r)
		}
		sort.Ints(rounds)

		summary := PlayerSummary{
			Name:       p.Name,
			SteamID:    fmt.Sprintf("%d", p.SteamID),
			EntityID:   p.EntityID,
			TotalKills: len(kills),
			Rounds:     make([]RoundSummary, 0, len(rounds)),
		}

		for _, r := range rounds {
			summary.Rounds = append(summary.Rounds, RoundSummary{
				Round: r,
				Kills: roundKills[r],
			})
		}

		result.Players = append(result.Players, summary)
	}

	return result, nil
}

func (a *App) DownloadPerfectWorldDemo(matchID string) (string, error) {
	matchID = strings.TrimSpace(matchID)
	if matchID == "" {
		return "", fmt.Errorf("比赛 ID 或 5E 分享链接不能为空")
	}
	if a.exeDir == "" {
		return "", fmt.Errorf("程序目录未初始化")
	}

	demoDir := filepath.Join(a.exeDir, "demos")
	if err := os.MkdirAll(demoDir, 0755); err != nil {
		return "", fmt.Errorf("创建 demos 目录失败: %w", err)
	}

	if isNumericMatchID(matchID) {
		if existing, err := findExistingDemoByKeywords(demoDir, []string{strings.ToLower(matchID)}); err == nil {
			printInfo("已存在匹配的完美世界 Demo，跳过下载")
			return existing, nil
		}
		baseName := fmt.Sprintf("%s_0.dem", matchID)
		downloadURL := fmt.Sprintf(perfectWorldDemoURLFormat, baseName)
		return downloadAndResolveDemo(demoDir, baseName, downloadURL, "完美世界")
	}

	matchCode, err := extract5EMatchCode(matchID)
	if err != nil {
		return "", fmt.Errorf("请输入纯数字完美比赛 ID 或有效的 5E 对局分享链接")
	}
	if existing, err := findExistingDemoByKeywords(demoDir, []string{strings.ToLower(matchCode)}); err == nil {
		printInfo("已存在匹配的 5E Demo，跳过下载")
		return existing, nil
	}
	demoURL, err := fetch5EDemoURL(matchCode)
	if err != nil {
		return "", err
	}
	baseName := fmt.Sprintf("%s.dem", matchCode)
	return downloadAndResolveDemo(demoDir, baseName, demoURL, "5E")
}

func downloadAndResolveDemo(demoDir, baseName, downloadURL, source string) (string, error) {
	demoPath := filepath.Join(demoDir, baseName)
	if _, err := os.Stat(demoPath); err == nil {
		printInfo("已存在 Demo 文件，跳过下载")
		return demoPath, nil
	}

	zipPath := demoPath + ".zip"
	_ = os.Remove(zipPath)

	printTitle(fmt.Sprintf("\n下载%s Demo", source))
	printInfo(fmt.Sprintf("下载地址: %s", downloadURL))
	printInfo(fmt.Sprintf("保存路径: %s", zipPath))

	if err := downloadFile(downloadURL, zipPath); err != nil {
		return "", err
	}

	if !isZipFile(zipPath) {
		if err := os.Rename(zipPath, demoPath); err != nil {
			return "", fmt.Errorf("保存 demo 失败: %w", err)
		}
		printSuccess("Demo 下载完成")
		return demoPath, nil
	}

	if err := unzipFile(zipPath, demoDir); err != nil {
		return "", err
	}
	if err := os.Remove(zipPath); err == nil {
		printInfo("已删除下载的 zip 文件")
	}

	if _, err := os.Stat(demoPath); err == nil {
		printSuccess("Demo 下载完成")
		return demoPath, nil
	}

	found, err := findDemoFile(demoDir, baseName)
	if err != nil {
		return "", err
	}
	printSuccess("Demo 下载完成")
	return found, nil
}

func isNumericMatchID(input string) bool {
	if input == "" {
		return false
	}
	for _, r := range input {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func extract5EMatchCode(input string) (string, error) {
	text := strings.TrimSpace(input)
	if text == "" {
		return "", fmt.Errorf("5E 分享链接为空")
	}

	// 支持复制整段分享文本：可能包含前缀说明 + URL
	reQuery := regexp.MustCompile(`matchcode=([A-Za-z0-9_-]+)`)
	if m := reQuery.FindStringSubmatch(text); len(m) == 2 && m[1] != "" {
		return m[1], nil
	}

	reDirect := regexp.MustCompile(`\bg\d+-n-\d+\b`)
	if m := reDirect.FindString(text); m != "" {
		return m, nil
	}

	start := strings.Index(text, "http")
	if start >= 0 {
		urlText := strings.Fields(text[start:])[0]
		if u, err := url.Parse(urlText); err == nil {
			matchCode := strings.TrimSpace(u.Query().Get("matchcode"))
			if matchCode != "" {
				return matchCode, nil
			}
		}
	}

	return "", fmt.Errorf("未能从输入中提取 5E matchcode")
}

func fetch5EDemoURL(matchCode string) (string, error) {
	matchCode = strings.TrimSpace(matchCode)
	if matchCode == "" {
		return "", fmt.Errorf("5E matchcode 不能为空")
	}

	apiURL := fmt.Sprintf(fiveEMatchAPIURLFormat, url.PathEscape(matchCode))
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建 5E 请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "cs2-highlight-tool")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 5E 对局信息失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("5E 接口返回异常状态码: %d", resp.StatusCode)
	}

	var payload struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Main struct {
				DemoURL string `json:"demo_url"`
			} `json:"main"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("解析 5E 接口响应失败: %w", err)
	}

	demoURL := strings.TrimSpace(payload.Data.Main.DemoURL)
	if demoURL == "" {
		if payload.Message != "" {
			return "", fmt.Errorf("5E 对局未返回 demo 下载地址: %s", payload.Message)
		}
		return "", fmt.Errorf("5E 对局未返回 demo 下载地址")
	}
	return demoURL, nil
}

func isZipFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	var header [4]byte
	if _, err := io.ReadFull(f, header[:]); err != nil {
		return false
	}
	return header[0] == 'P' && header[1] == 'K' && header[2] == 0x03 && header[3] == 0x04
}

func findDemoFile(rootDir, baseName string) (string, error) {
	var bestPath string
	var bestTime time.Time
	foundExact := false
	errFoundExact := errors.New("found_exact")

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".dem") {
			return nil
		}
		if info.Name() == baseName {
			bestPath = path
			foundExact = true
			return errFoundExact
		}
		if bestPath == "" || info.ModTime().After(bestTime) {
			bestPath = path
			bestTime = info.ModTime()
		}
		return nil
	})
	if err != nil && !errors.Is(err, errFoundExact) {
		return "", fmt.Errorf("查找 demo 文件失败: %w", err)
	}
	if foundExact {
		return bestPath, nil
	}
	if bestPath == "" {
		return "", fmt.Errorf("未找到解压后的 demo 文件")
	}
	return bestPath, nil
}

func findExistingDemoByKeywords(rootDir string, keywords []string) (string, error) {
	var bestPath string
	var bestTime time.Time

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		nameLower := strings.ToLower(info.Name())
		if !strings.HasSuffix(nameLower, ".dem") {
			return nil
		}
		for _, keyword := range keywords {
			keyword = strings.TrimSpace(strings.ToLower(keyword))
			if keyword == "" {
				continue
			}
			if !strings.Contains(nameLower, keyword) {
				return nil
			}
		}
		if bestPath == "" || info.ModTime().After(bestTime) {
			bestPath = path
			bestTime = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("扫描已下载 demo 失败: %w", err)
	}
	if bestPath == "" {
		return "", fmt.Errorf("未命中匹配 demo")
	}
	return bestPath, nil
}

func (a *App) ensurePreviewServer() (string, error) {
	a.previewMu.Lock()
	defer a.previewMu.Unlock()

	if a.previewURL != "" {
		return a.previewURL, nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/video", func(w http.ResponseWriter, r *http.Request) {
		videoPath := r.URL.Query().Get("path")
		if videoPath == "" {
			http.Error(w, "missing path", http.StatusBadRequest)
			return
		}
		videoPath = cleanPath(videoPath)
		if !filepath.IsAbs(videoPath) {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		if _, err := os.Stat(videoPath); err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, videoPath)
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	server := &http.Server{Handler: mux}
	go server.Serve(listener)

	a.previewURL = "http://" + listener.Addr().String()
	return a.previewURL, nil
}

func (a *App) RunWorkflow(req RecordRequest) (*RecordResult, error) {
	if a.config == nil {
		return nil, fmt.Errorf("配置未加载")
	}
	if req.DemoPath == "" {
		return nil, fmt.Errorf("demo 路径为空")
	}
	if len(req.SelectedRounds) == 0 {
		return nil, fmt.Errorf("未选择回合")
	}

	absPath, err := filepath.Abs(req.DemoPath)
	if err != nil {
		return nil, fmt.Errorf("无法解析 demo 路径: %v", err)
	}
	req.DemoPath = absPath

	if _, err := os.Stat(req.DemoPath); err != nil {
		return nil, fmt.Errorf("demo 文件不存在: %s", req.DemoPath)
	}

	if err := setupFFmpegIni(a.exeDir, a.config); err != nil {
		printWarning(fmt.Sprintf("设置 ffmpeg.ini 失败: %v", err))
	}

	if err := checkEnvironment(a.exeDir, a.config, a.configPath); err != nil {
		return nil, err
	}

	players, killsByPlayer, err := parseDemoKills(req.DemoPath)
	if err != nil {
		return nil, err
	}

	steamID, err := strconv.ParseUint(req.PlayerSteamID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("无效的 SteamID: %s", req.PlayerSteamID)
	}

	targetPlayer, ok := players[steamID]
	if !ok {
		return nil, fmt.Errorf("未找到选中的玩家")
	}

	targetKills := killsByPlayer[int(targetPlayer.SteamID)]
	if len(targetKills) == 0 {
		return nil, fmt.Errorf("该玩家没有击杀数据")
	}

	roundKills := make(map[int][]KillInfo)
	for _, k := range targetKills {
		roundKills[k.Round] = append(roundKills[k.Round], k)
	}

	var selectedKills []KillInfo
	for _, r := range req.SelectedRounds {
		selectedKills = append(selectedKills, roundKills[r]...)
	}
	if len(selectedKills) == 0 {
		return nil, fmt.Errorf("选择的回合没有击杀数据")
	}

	preTicks := int(a.config.KillerPreSeconds * float64(a.config.Tickrate))
	postTicks := int(a.config.KillerPostSeconds * float64(a.config.Tickrate))
	if preTicks < 0 {
		preTicks = 0
	}
	if postTicks < 0 {
		postTicks = 0
	}
	segments := buildSegments(selectedKills, preTicks, postTicks)
	if len(segments) == 0 {
		return nil, fmt.Errorf("未生成有效片段")
	}

	printTitle("\n生成配置")
	printInfo(fmt.Sprintf("录制片段数: %d", len(segments)))

	demoName := strings.TrimSuffix(filepath.Base(req.DemoPath), filepath.Ext(req.DemoPath))
	cfgName := fmt.Sprintf("auto_%s", demoName)
	cfgPath := filepath.Join(a.config.CfgDir, cfgName+".cfg")

	if err := generateCFG(req.DemoPath, cfgPath, a.config.OutputDir, segments, targetPlayer.Name, targetPlayer.EntityID, a.config); err != nil {
		return nil, err
	}
	printSuccess(fmt.Sprintf("CFG 已生成: %s", cfgPath))

	result := &RecordResult{
		CfgPath: cfgPath,
	}

	if !req.AutoMode {
		return result, nil
	}

	printTitle("\n启动录制")
	if err := launchHLAE(a.config, req.DemoPath, cfgName); err != nil {
		return nil, err
	}

	if err := waitForCS2Completion(60 * time.Minute); err != nil {
		killCS2Processes()
		return nil, err
	}

	if a.config.RecordVictimView {
		victimCfgName := cfgName + "_victim"
		printTitle("\n启动被害者视角录制")
		if err := launchHLAE(a.config, req.DemoPath, victimCfgName); err != nil {
			return nil, err
		}

		if err := waitForCS2Completion(60 * time.Minute); err != nil {
			killCS2Processes()
			return nil, err
		}
	}

	printTitle("\n视频合成")
	finalOutput, err := processRecordings(a.config.OutputDir, demoName, a.exeDir, req.SelectedRounds, a.config, req.DebugMode)
	if err != nil {
		return nil, err
	}
	if err := cleanupOutputDirectories(a.exeDir); err != nil {
		printWarning(fmt.Sprintf("清理 outputs 目录失败: %v", err))
	}

	result.OutputPath = finalOutput
	if req.AutoMode && !req.DebugMode {
		result.CfgPath = ""
	}
	printSuccess("\n✓ 全部完成！")

	return result, nil
}
