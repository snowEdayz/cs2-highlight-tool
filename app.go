package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

	// 启动时仅初始化路径与上下文，环境准备交由前端触发
	printSuccess("启动完成，等待环境初始化")
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
	if cfg.VideoPreset == "" {
		cfg.VideoPreset = baseCfg.VideoPreset
	}
	if cfg.TransitionDuration < 0 {
		cfg.TransitionDuration = baseCfg.TransitionDuration
	}
	if cfg.TransitionType == "" {
		cfg.TransitionType = baseCfg.TransitionType
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

	preTicks := int(5 * float64(a.config.Tickrate))
	postTicks := int(5 * float64(a.config.Tickrate))
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

	printTitle("\n视频合成")
	finalOutput, err := processRecordings(a.config.OutputDir, demoName, a.exeDir, req.SelectedRounds, a.config, req.DebugMode)
	if err != nil {
		return nil, err
	}

	result.OutputPath = finalOutput
	printSuccess("\n✓ 全部完成！")

	return result, nil
}
