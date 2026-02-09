package main

import (
	"archive/zip"
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/bodgit/sevenzip"
	"github.com/fatih/color"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ==================== 颜色输出 ====================

var (
	colorRed     = color.New(color.FgRed)
	colorGreen   = color.New(color.FgGreen)
	colorYellow  = color.New(color.FgYellow)
	colorBlue    = color.New(color.FgBlue)
	colorMagenta = color.New(color.FgMagenta)
	colorCyan    = color.New(color.FgCyan)
	colorWhite   = color.New(color.FgWhite)
	colorBold    = color.New(color.Bold)

	// 组合颜色
	colorMagentaBold = color.New(color.FgMagenta, color.Bold)
	colorCyanBold    = color.New(color.FgCyan, color.Bold)
	colorGreenBold   = color.New(color.FgGreen, color.Bold)
)

//go:embed frontend/dist
var assets embed.FS

//go:embed wails.json
var wailsConfigData []byte

var appCtx context.Context

const (
	ffmpegDownloadURLGlobal = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.7z"
	ffmpegDownloadURLCN     = "https://gitee.com/hkslover/ffmpeg_release/releases/download/v8.0.1/ffmpeg-8.0.1-essentials_build.7z"
)

type LogMessage struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

func emitLog(level, message string) {
	if appCtx == nil {
		return
	}
	runtime.EventsEmit(appCtx, "log", LogMessage{
		Level:   level,
		Message: message,
		Time:    time.Now().Format(time.RFC3339),
	})
}

func printSuccess(text string) {
	emitLog("success", text)
	colorGreen.Println(text)
}

func printError(text string) {
	emitLog("error", text)
	colorRed.Println(text)
}

func printWarning(text string) {
	emitLog("warning", text)
	colorYellow.Println(text)
}

func printInfo(text string) {
	emitLog("info", text)
	colorCyan.Println(text)
}

func printTitle(text string) {
	emitLog("title", text)
	colorCyanBold.Println(text)
}

func execCommandHidden(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

// 等待用户输入（用于拖拽模式）
func waitForExit() {
	fmt.Println()
	colorYellow.Print("按 Enter 键退出...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// ==================== HLAE 更新 ====================

// Gitee Release API 响应结构
type GiteeRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func isChinaIP() bool {
	conn, err := net.DialTimeout("tcp", "www.google.com:80", 3*time.Second)
	if err == nil {
		conn.Close()
		return false
	}

	connCN, errCN := net.DialTimeout("tcp", "www.baidu.com:80", 3*time.Second)
	if errCN == nil {
		connCN.Close()
		return true
	}

	return false
}

func getFFmpegDownloadURL() string {
	if isChinaIP() {
		return ffmpegDownloadURLCN
	}
	return ffmpegDownloadURLGlobal
}

// 获取最新 HLAE 版本信息
func getLatestHLAEVersion() (*GiteeRelease, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	useGitee := isChinaIP()

	apiURL := "https://gitee.com/api/v5/repos/hkslover/advancedfx/releases/latest"
	if !useGitee {
		apiURL = "https://api.github.com/repos/advancedfx/advancedfx/releases/latest"
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "CS2-Highlight-Tool")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("版本 API 返回错误: %d", resp.StatusCode)
	}

	var release GiteeRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &release, nil
}

// 下载文件
func downloadFile(url, filepath string) error {
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	printInfo(fmt.Sprintf("开始下载: %s", url))

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	// 显示下载进度
	totalSize := resp.ContentLength
	downloaded := int64(0)
	buffer := make([]byte, 32*1024)
	lastPrint := time.Now()

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			_, writeErr := out.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("写入文件失败: %w", writeErr)
			}
			downloaded += int64(n)

			// 每秒更新一次进度
			if time.Since(lastPrint) > time.Second {
				if totalSize > 0 {
					percent := float64(downloaded) / float64(totalSize) * 100
					printInfo(fmt.Sprintf("下载进度: %.1f%% (%.2f MB / %.2f MB)",
						percent,
						float64(downloaded)/(1024*1024),
						float64(totalSize)/(1024*1024)))
				}
				lastPrint = time.Now()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取数据失败: %w", err)
		}
	}

	printSuccess(fmt.Sprintf("下载完成: %.2f MB", float64(downloaded)/(1024*1024)))
	return nil
}

// 解压 ZIP 文件
func unzipFile(zipPath, destDir string) error {
	printInfo("正在解压文件...")

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("打开 ZIP 文件失败: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// 防止 Zip Slip 漏洞
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("非法的文件路径: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("创建文件失败: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("读取压缩文件失败: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("解压文件失败: %w", err)
		}
	}

	printSuccess("解压完成")
	return nil
}

// 解压 7z 文件
func extract7z(archivePath, destDir string) error {
	printInfo("正在解压文件...")

	r, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("打开 7z 文件失败: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// 防止 Zip Slip 漏洞
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("非法的文件路径: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("创建文件失败: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("读取压缩文件失败: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("解压文件失败: %w", err)
		}
	}

	printSuccess("解压完成")
	return nil
}

func resolveFFmpegDir(exeDir string, cfg *Config) string {
	if cfg != nil && cfg.FFmpegDir != "" {
		return cfg.FFmpegDir
	}
	return filepath.Join(exeDir, "ffmpeg", "bin")
}

func resolveFFmpegExe(exeDir string, cfg *Config) string {
	return filepath.Join(resolveFFmpegDir(exeDir, cfg), "ffmpeg.exe")
}

func findFFmpegRootDir(baseDir string) (string, error) {
	var found string
	err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		candidate := filepath.Join(path, "bin", "ffmpeg.exe")
		if _, statErr := os.Stat(candidate); statErr == nil {
			found = path
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("解压后未找到 ffmpeg.exe")
	}
	return found, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

func moveDir(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyDir(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

func ensureFFmpegAvailable(exeDir string, cfg *Config, configPath string) error {
	expectedFFmpegRoot := filepath.Join(exeDir, "ffmpeg")
	expectedFFmpegBin := filepath.Join(expectedFFmpegRoot, "bin")
	if cfg.FFmpegDir != expectedFFmpegBin {
		cfg.FFmpegDir = expectedFFmpegBin
		if configPath != "" {
			if err := saveConfig(configPath, cfg); err != nil {
				return fmt.Errorf("保存配置失败: %w", err)
			}
		}
	}

	ffmpegExe := resolveFFmpegExe(exeDir, cfg)
	if _, err := os.Stat(ffmpegExe); err == nil {
		return nil
	}

	printWarning("未找到 FFmpeg，开始下载...")
	printInfo("正在下载 FFmpeg...")
	tempArchive := filepath.Join(exeDir, "ffmpeg_temp.7z")
	downloadURL := getFFmpegDownloadURL()
	if err := downloadFile(downloadURL, tempArchive); err != nil {
		return fmt.Errorf("下载 FFmpeg 失败: %w", err)
	}
	defer os.Remove(tempArchive)

	extractDir := filepath.Join(exeDir, "_ffmpeg_extract")
	_ = os.RemoveAll(extractDir)
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}

	printInfo("正在解压 FFmpeg...")
	if err := extract7z(tempArchive, extractDir); err != nil {
		return fmt.Errorf("解压 FFmpeg 失败: %w", err)
	}

	sourceDir, err := findFFmpegRootDir(extractDir)
	if err != nil {
		return err
	}
	fallbackBin := filepath.Join(sourceDir, "bin")

	targetDir := expectedFFmpegRoot
	if _, err := os.Stat(targetDir); err == nil {
		if err := os.RemoveAll(targetDir); err != nil {
			printWarning(fmt.Sprintf("清理旧 FFmpeg 目录失败，改用临时目录: %v", err))
			cfg.FFmpegDir = fallbackBin
			if configPath != "" {
				_ = saveConfig(configPath, cfg)
			}
			if _, statErr := os.Stat(resolveFFmpegExe(exeDir, cfg)); statErr == nil {
				printSuccess("FFmpeg 已准备就绪")
				return nil
			}
			return fmt.Errorf("FFmpeg 解压后未找到 ffmpeg.exe")
		}
	}

	if err := moveDir(sourceDir, targetDir); err != nil {
		printWarning(fmt.Sprintf("移动 FFmpeg 目录失败，改用临时目录: %v", err))
		cfg.FFmpegDir = fallbackBin
		if configPath != "" {
			_ = saveConfig(configPath, cfg)
		}
		if _, statErr := os.Stat(resolveFFmpegExe(exeDir, cfg)); statErr == nil {
			printSuccess("FFmpeg 已准备就绪")
			return nil
		}
		return fmt.Errorf("FFmpeg 解压后未找到 ffmpeg.exe")
	}
	_ = os.RemoveAll(extractDir)

	if _, err := os.Stat(resolveFFmpegExe(exeDir, cfg)); err != nil {
		return fmt.Errorf("FFmpeg 解压后未找到 ffmpeg.exe")
	}

	printSuccess("FFmpeg 已准备就绪")
	return nil
}

// 检查并更新 HLAE
func checkAndUpdateHLAE(exeDir string, cfg *Config, configPath string, debugMode bool) error {
	printTitle("\nHLAE 版本检查")

	// 获取最新版本信息
	release, err := getLatestHLAEVersion()
	if err != nil {
		printWarning(fmt.Sprintf("无法获取最新版本信息: %v", err))
		printWarning("将继续使用现有版本")
		return nil
	}

	if len(release.Assets) == 0 {
		printWarning("未找到可用的下载文件")
		return nil
	}

	// 查找 hlae_*.zip 文件（排除 .asc 签名文件）
	var latestAsset *struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	}

	for i := range release.Assets {
		asset := &release.Assets[i]
		// 查找 hlae_开头的 .zip 文件，但排除 .zip.asc
		if strings.HasPrefix(asset.Name, "hlae_") &&
			strings.HasSuffix(asset.Name, ".zip") &&
			!strings.HasSuffix(asset.Name, ".zip.asc") {
			latestAsset = asset
			break
		}
	}

	if latestAsset == nil {
		printWarning("未找到 HLAE ZIP 安装包")
		return nil
	}

	latestVersion := latestAsset.Name

	printInfo(fmt.Sprintf("最新版本: %s", latestVersion))
	printInfo(fmt.Sprintf("当前版本: %s", cfg.HLAEVersion))

	// 检查是否需要更新
	hlaeExe := filepath.Join(exeDir, "hlae", "HLAE.exe")
	hlaeMissing := false
	if _, err := os.Stat(hlaeExe); err != nil {
		hlaeMissing = true
	}
	if cfg.HLAEVersion == latestVersion && !hlaeMissing {
		printSuccess("HLAE 已是最新版本")
		return nil
	}

	// 需要更新
	printWarning("检测到新版本，开始更新...")

	// 直接使用 Gitee 下载链接（无需镜像加速）
	downloadURL := latestAsset.BrowserDownloadURL
	printInfo(fmt.Sprintf("下载地址: %s", downloadURL))

	// 下载到临时文件
	printInfo("正在下载 HLAE...")
	tempZip := filepath.Join(exeDir, "hlae_temp.zip")
	if err := downloadFile(downloadURL, tempZip); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}

	// Debug 模式下保留临时文件
	if !debugMode {
		defer os.Remove(tempZip)
	} else {
		printInfo(fmt.Sprintf("Debug 模式：保留临时文件 %s", tempZip))
	}

	// 解压到 hlae 目录
	hlaeDir := filepath.Join(exeDir, "hlae")

	// 如果 hlae 目录已存在，先删除
	if _, err := os.Stat(hlaeDir); err == nil {
		printInfo("删除旧版本...")
		if err := os.RemoveAll(hlaeDir); err != nil {
			return fmt.Errorf("删除旧版本失败: %w", err)
		}
	}

	// 创建 hlae 目录
	if err := os.MkdirAll(hlaeDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 解压
	printInfo("正在解压 HLAE...")
	if err := unzipFile(tempZip, hlaeDir); err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}

	// 更新配置文件中的版本号
	cfg.HLAEVersion = latestVersion
	if configPath != "" {
		if err := saveConfig(configPath, cfg); err != nil {
			return fmt.Errorf("保存配置失败: %w", err)
		}
	}

	printSuccess(fmt.Sprintf("HLAE 更新完成: %s", latestVersion))
	return nil
}

// ==================== 数据结构 ====================

type Config struct {
	CS2Exe             string  `json:"cs2_exe"`
	HLAEExe            string  `json:"hlae_exe"`
	HLAEVersion        string  `json:"hlae_version"` // 当前 HLAE 版本
	FFmpegDir          string  `json:"ffmpeg_dir"`
	CfgDir             string  `json:"cfg_dir"`
	OutputDir          string  `json:"output_dir"`
	RecordFPS          int     `json:"record_fps"`
	Tickrate           int     `json:"tickrate"`
	VideoPreset        string  `json:"video_preset"` // 录制预设: "c1" (libx264) 或 "n1" (hevc_nvenc)
	TransitionDuration float64 `json:"transition_duration"`
	TransitionType     string  `json:"transition_type"`
	LaunchResolution   string  `json:"launch_resolution"`
	RecordVictimView   bool    `json:"record_victim_view"`
	KillerPreSeconds   float64 `json:"killer_pre_seconds"`
	KillerPostSeconds  float64 `json:"killer_post_seconds"`
	VictimPreSeconds   float64 `json:"victim_pre_seconds"`
	VictimPostSeconds  float64 `json:"victim_post_seconds"`
}

type KillInfo struct {
	Round      int    `json:"round"`
	Tick       int    `json:"tick"`
	VictimName string `json:"victim_name"`
	VictimID   int    `json:"victim_entity_id"`
	KillerName string `json:"killer_name"`
	WeaponName string `json:"weapon_name"`
	IsHeadshot bool   `json:"is_headshot"`
	IsWallbang bool   `json:"is_wallbang"`
}

type Segment struct {
	StartTick int        `json:"start_tick"`
	EndTick   int        `json:"end_tick"`
	Kills     []KillInfo `json:"kills"`
}

type PlayerInfo struct {
	Name     string `json:"name"`
	SteamID  uint64 `json:"steam_id"`
	EntityID int    `json:"entity_id"`
}

// ==================== 配置加载 ====================

func buildBaseConfig(exeDir string) *Config {
	return &Config{
		CS2Exe:             "",                                        // 空，让用户输入
		HLAEExe:            filepath.Join(exeDir, "hlae", "HLAE.exe"), // exe目录/hlae/HLAE.exe
		HLAEVersion:        "",                                        // 空，将自动下载最新版本
		FFmpegDir:          filepath.Join(exeDir, "ffmpeg", "bin"),    // exe目录/ffmpeg/bin
		CfgDir:             filepath.Join(exeDir, "cfg"),              // exe目录/cfg
		OutputDir:          filepath.Join(exeDir, "outputs"),          // exe目录/outputs
		RecordFPS:          60,
		Tickrate:           64,
		VideoPreset:        "n1",
		TransitionDuration: 1,
		TransitionType:     "fade",
		LaunchResolution:   "4:3",
		RecordVictimView:   false,
		KillerPreSeconds:   5,
		KillerPostSeconds:  5,
		VictimPreSeconds:   2,
		VictimPostSeconds:  2,
	}
}

// 创建默认配置文件
func createDefaultConfig(path string, exeDir string) (*Config, error) {
	printTitle("\n创建默认配置")
	printInfo("未找到配置文件，正在创建默认配置...")

	cfg := buildBaseConfig(exeDir)

	// 创建必要的目录
	os.MkdirAll(cfg.CfgDir, 0755)
	os.MkdirAll(cfg.OutputDir, 0755)

	// 保存配置文件
	if err := saveConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("保存默认配置失败: %w", err)
	}

	printSuccess(fmt.Sprintf("✓ 默认配置已创建: %s", path))
	printInfo("配置说明:")
	fmt.Printf("  cfg_dir:     %s\n", cfg.CfgDir)
	fmt.Printf("  hlae_exe:    %s\n", cfg.HLAEExe)
	fmt.Printf("  output_dir:  %s\n", cfg.OutputDir)
	fmt.Printf("  record_fps:  %d\n", cfg.RecordFPS)
	fmt.Printf("  tickrate:    %d\n", cfg.Tickrate)
	fmt.Printf("  video_preset: %s\n", cfg.VideoPreset)

	return cfg, nil
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 标准化路径：支持正斜杠和单反斜杠，转换为系统路径格式
	cfg.CS2Exe = filepath.FromSlash(cfg.CS2Exe)
	cfg.HLAEExe = filepath.FromSlash(cfg.HLAEExe)
	cfg.FFmpegDir = filepath.FromSlash(cfg.FFmpegDir)
	cfg.CfgDir = filepath.FromSlash(cfg.CfgDir)
	cfg.OutputDir = filepath.FromSlash(cfg.OutputDir)

	return &cfg, nil
}

// 加载或创建配置文件
func loadOrCreateConfig(path string, exeDir string) (*Config, error) {
	// 检查配置文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// 配置文件不存在，创建默认配置
		return createDefaultConfig(path, exeDir)
	}

	// 配置文件存在，正常加载
	return loadConfig(path)
}

func saveConfig(path string, cfg *Config) error {
	// 转换路径为正斜杠（JSON标准格式）
	configData := map[string]interface{}{
		"cs2_exe":             windowsToUnixPath(cfg.CS2Exe),
		"hlae_exe":            windowsToUnixPath(cfg.HLAEExe),
		"hlae_version":        cfg.HLAEVersion,
		"ffmpeg_dir":          windowsToUnixPath(cfg.FFmpegDir),
		"cfg_dir":             windowsToUnixPath(cfg.CfgDir),
		"output_dir":          windowsToUnixPath(cfg.OutputDir),
		"record_fps":          cfg.RecordFPS,
		"tickrate":            cfg.Tickrate,
		"video_preset":        cfg.VideoPreset,
		"transition_duration": cfg.TransitionDuration,
		"transition_type":     cfg.TransitionType,
		"launch_resolution":   cfg.LaunchResolution,
		"record_victim_view":  cfg.RecordVictimView,
		"killer_pre_seconds":  cfg.KillerPreSeconds,
		"killer_post_seconds": cfg.KillerPostSeconds,
		"victim_pre_seconds":  cfg.VictimPreSeconds,
		"victim_post_seconds": cfg.VictimPostSeconds,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

func autoSetupPaths(exeDir string, cfg *Config, configPath string) error {
	needsSave := false

	// 自动设置 hlae_exe 为 exe目录/hlae/HLAE.exe
	expectedHLAE := filepath.Join(exeDir, "hlae", "HLAE.exe")
	if cfg.HLAEExe != expectedHLAE {
		cfg.HLAEExe = expectedHLAE
		needsSave = true
	}

	// 自动设置 cfg_dir 为 exe目录/cfg
	expectedCfgDir := filepath.Join(exeDir, "cfg")
	if cfg.CfgDir != expectedCfgDir {
		cfg.CfgDir = expectedCfgDir
		needsSave = true
	}

	// 自动设置 ffmpeg_dir 为 exe目录/ffmpeg/bin
	expectedFFmpegDir := filepath.Join(exeDir, "ffmpeg", "bin")
	if cfg.FFmpegDir != expectedFFmpegDir {
		cfg.FFmpegDir = expectedFFmpegDir
		needsSave = true
	}

	// 自动设置 output_dir 为 exe目录/outputs
	expectedOutputDir := filepath.Join(exeDir, "outputs")
	if cfg.OutputDir != expectedOutputDir {
		cfg.OutputDir = expectedOutputDir
		needsSave = true
	}

	// 如果有变化，保存配置文件
	if needsSave {
		// 创建必要的目录
		os.MkdirAll(expectedCfgDir, 0755)
		os.MkdirAll(expectedOutputDir, 0755)

		if err := saveConfig(configPath, cfg); err != nil {
			return fmt.Errorf("保存配置失败: %w", err)
		}
		printInfo("已自动更新配置文件路径")
	}

	return nil
}

// 清理和标准化用户输入的路径
func cleanPath(input string) string {
	// 去除首尾空白
	input = strings.TrimSpace(input)

	// 去除可能的引号
	input = strings.Trim(input, "\"'")

	// 去除首尾空白（再次，因为引号内可能有空格）
	input = strings.TrimSpace(input)

	// 标准化路径分隔符
	input = filepath.FromSlash(input)

	return input
}

// 检查并设置 CS2 路径
func checkAndSetupCS2Path(cfg *Config, configPath string) error {
	// 检查 CS2 路径是否有效
	if cfg.CS2Exe != "" {
		if _, err := os.Stat(cfg.CS2Exe); err == nil {
			// 路径有效，无需设置
			return nil
		}
	}

	// CS2 路径无效或为空，提示用户输入
	printTitle("\nCS2 路径设置")
	printWarning("未找到有效的 CS2 可执行文件路径")
	fmt.Println()
	printInfo("请输入 CS2 的完整路径")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		colorYellow.Print("请输入 CS2 路径: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("读取输入失败: %w", err)
		}

		// 清理路径
		cs2Path := cleanPath(input)

		if cs2Path == "" {
			printError("路径不能为空，请重新输入")
			continue
		}

		// 验证路径是否存在
		if _, err := os.Stat(cs2Path); err != nil {
			printError(fmt.Sprintf("路径不存在: %s", cs2Path))
			printWarning("请检查路径是否正确，然后重新输入")
			fmt.Println()
			continue
		}

		// 验证是否是 cs2.exe
		if !strings.HasSuffix(strings.ToLower(filepath.Base(cs2Path)), "cs2.exe") {
			printWarning("警告: 文件名不是 cs2.exe，是否继续？(y/n)")
			colorYellow.Print("请输入: ")
			confirm, _ := reader.ReadString('\n')
			confirm = strings.TrimSpace(strings.ToLower(confirm))
			if confirm != "y" && confirm != "yes" {
				continue
			}
		}

		// 保存路径
		cfg.CS2Exe = cs2Path
		if err := saveConfig(configPath, cfg); err != nil {
			return fmt.Errorf("保存配置失败: %w", err)
		}

		printSuccess(fmt.Sprintf("✓ CS2 路径已保存: %s", cs2Path))
		return nil
	}
}

// 设置 HLAE 的 ffmpeg.ini 文件
func setupFFmpegIni(exeDir string, cfg *Config) error {
	// ffmpeg.exe 路径（ffmpeg/bin）
	ffmpegExe := resolveFFmpegExe(exeDir, cfg)

	// ffmpeg.ini 路径（hlae 目录下的 ffmpeg 子目录）
	ffmpegIniDir := filepath.Join(filepath.Dir(cfg.HLAEExe), "ffmpeg")
	ffmpegIniPath := filepath.Join(ffmpegIniDir, "ffmpeg.ini")

	// 创建 ffmpeg 目录
	if err := os.MkdirAll(ffmpegIniDir, 0755); err != nil {
		return fmt.Errorf("创建 ffmpeg 目录失败: %w", err)
	}

	// 生成 ini 文件内容
	iniContent := fmt.Sprintf("[Ffmpeg]\nPath=%s\n", ffmpegExe)

	// 如果内容一致则跳过
	if data, err := os.ReadFile(ffmpegIniPath); err == nil {
		if strings.TrimSpace(string(data)) == strings.TrimSpace(iniContent) {
			return nil
		}
	}

	// 写入文件
	if err := os.WriteFile(ffmpegIniPath, []byte(iniContent), 0644); err != nil {
		return fmt.Errorf("写入 ffmpeg.ini 失败: %w", err)
	}

	printInfo(fmt.Sprintf("已创建 ffmpeg.ini: %s", ffmpegIniPath))
	return nil
}

// ==================== 环境检查 ====================

func checkEnvironment(exeDir string, cfg *Config, configPath string) error {
	printTitle("\n环境检查")

	if containsCJK(exeDir) {
		return fmt.Errorf("程序路径包含中文: %s", exeDir)
	}

	if err := ensureFFmpegAvailable(exeDir, cfg, configPath); err != nil {
		return err
	}
	if err := setupFFmpegIni(exeDir, cfg); err != nil {
		printWarning(fmt.Sprintf("设置 ffmpeg.ini 失败: %v", err))
	}

	// 检查 CS2
	if _, err := os.Stat(cfg.CS2Exe); err != nil {
		printError("✗ CS2 可执行文件不存在")
		fmt.Printf("  路径: %s\n", cfg.CS2Exe)
		return fmt.Errorf("CS2 未找到")
	}
	printSuccess("✓ CS2")

	// 检查 HLAE
	if _, err := os.Stat(cfg.HLAEExe); err != nil {
		printError("✗ HLAE 可执行文件不存在")
		fmt.Printf("  路径: %s\n", cfg.HLAEExe)
		return fmt.Errorf("HLAE 未找到")
	}
	hookDll := filepath.Join(filepath.Dir(cfg.HLAEExe), "x64", "AfxHookSource2.dll")
	if _, err := os.Stat(hookDll); err != nil {
		printError("✗ AfxHookSource2.dll 不存在")
		fmt.Printf("  路径: %s\n", hookDll)
		return fmt.Errorf("HLAE 组件不完整")
	}
	printSuccess("✓ HLAE")

	// 检查 FFmpeg（ffmpeg/bin）
	ffmpegExe := resolveFFmpegExe(exeDir, cfg)
	if _, err := os.Stat(ffmpegExe); err != nil {
		printError("✗ FFmpeg 可执行文件不存在")
		fmt.Printf("  路径: %s\n", ffmpegExe)
		return fmt.Errorf("FFmpeg 未找到")
	}
	printSuccess("✓ FFmpeg")

	return nil
}

// ==================== Demo 解析 ====================

func parseDemoKills(demoPath string) (map[uint64]*PlayerInfo, map[int][]KillInfo, error) {
	f, err := os.Open(demoPath)
	if err != nil {
		return nil, nil, fmt.Errorf("打开 demo 失败: %w", err)
	}
	defer f.Close()

	parser := dem.NewParser(f)
	defer parser.Close()

	players := make(map[uint64]*PlayerInfo)
	kills := make(map[int][]KillInfo)
	currentRound := 0

	// 注册回合开始事件
	parser.RegisterEventHandler(func(e events.RoundStart) {
		currentRound = parser.GameState().TotalRoundsPlayed()
	})

	// 注册击杀事件
	parser.RegisterEventHandler(func(e events.Kill) {
		if e.Killer == nil || e.Victim == nil {
			return
		}

		// 跳过热身
		if parser.GameState().IsWarmupPeriod() {
			return
		}

		// 记录玩家信息
		if _, exists := players[e.Killer.SteamID64]; !exists {
			players[e.Killer.SteamID64] = &PlayerInfo{
				Name:     e.Killer.Name,
				SteamID:  e.Killer.SteamID64,
				EntityID: e.Killer.EntityID,
			}
		}

		// 获取武器名称
		weaponName := "Unknown"
		if e.Weapon != nil {
			weaponName = e.Weapon.String()
		}

		// 记录击杀
		killInfo := KillInfo{
			Round:      currentRound,
			Tick:       parser.GameState().IngameTick(),
			VictimName: e.Victim.Name,
			VictimID:   e.Victim.EntityID,
			KillerName: e.Killer.Name,
			WeaponName: weaponName,
			IsHeadshot: e.IsHeadshot,
			IsWallbang: e.PenetratedObjects > 0,
		}

		killsByRound := kills[int(e.Killer.SteamID64)]
		killsByRound = append(killsByRound, killInfo)
		kills[int(e.Killer.SteamID64)] = killsByRound
	})

	// 解析整个 demo
	if err := parser.ParseToEnd(); err != nil {
		return nil, nil, fmt.Errorf("解析 demo 失败: %w", err)
	}

	return players, kills, nil
}

// ==================== 用户交互 ====================

func promptChoice(options []string, prompt string) int {
	reader := bufio.NewReader(os.Stdin)
	for {
		if prompt != "" {
			fmt.Print(prompt)
		}
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		idx, err := strconv.Atoi(input)
		if err == nil && idx >= 0 && idx < len(options) {
			return idx
		}
		printWarning("无效输入，请重试")
		if prompt != "" {
			fmt.Print(prompt)
		}
	}
}

func promptRounds(validRounds []int) []int {
	reader := bufio.NewReader(os.Stdin)
	for {
		colorYellow.Print("\n选择回合 (输入回合编号): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "all" {
			return validRounds
		}

		parts := strings.Split(input, ",")
		var selected []int
		valid := true

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			num, err := strconv.Atoi(part)
			if err != nil {
				valid = false
				break
			}

			found := false
			for _, r := range validRounds {
				if r == num {
					found = true
					break
				}
			}

			if !found {
				valid = false
				break
			}

			selected = append(selected, num)
		}

		if valid && len(selected) > 0 {
			return selected
		}

		printWarning("无效输入，请重试")
	}
}

// ==================== 片段构建 ====================

func buildSegments(kills []KillInfo, preTicks, postTicks int) []Segment {
	if len(kills) == 0 {
		return nil
	}

	// 按 tick 排序
	sort.Slice(kills, func(i, j int) bool {
		return kills[i].Tick < kills[j].Tick
	})

	segments := []Segment{}

	for _, k := range kills {
		startTick := k.Tick - preTicks
		if startTick < 0 {
			startTick = 0
		}
		endTick := k.Tick + postTicks

		// 合并重叠的片段
		if len(segments) > 0 && startTick <= segments[len(segments)-1].EndTick {
			lastSeg := &segments[len(segments)-1]
			if endTick > lastSeg.EndTick {
				lastSeg.EndTick = endTick
			}
			lastSeg.Kills = append(lastSeg.Kills, k)
		} else {
			segments = append(segments, Segment{
				StartTick: startTick,
				EndTick:   endTick,
				Kills:     []KillInfo{k},
			})
		}
	}

	return segments
}

func buildVictimSegments(kills []KillInfo, preTicks, postTicks int) []Segment {
	if len(kills) == 0 {
		return nil
	}

	sortedKills := make([]KillInfo, len(kills))
	copy(sortedKills, kills)
	sort.Slice(sortedKills, func(i, j int) bool {
		return sortedKills[i].Tick < sortedKills[j].Tick
	})

	segments := make([]Segment, 0, len(sortedKills))
	for _, k := range sortedKills {
		startTick := k.Tick - preTicks
		if startTick < 0 {
			startTick = 0
		}
		endTick := k.Tick + postTicks
		segments = append(segments, Segment{
			StartTick: startTick,
			EndTick:   endTick,
			Kills:     []KillInfo{k},
		})
	}

	return segments
}

func segmentsToKills(segments []Segment) []KillInfo {
	if len(segments) == 0 {
		return nil
	}
	var kills []KillInfo
	for _, seg := range segments {
		if len(seg.Kills) > 0 {
			kills = append(kills, seg.Kills...)
		}
	}
	return kills
}

// ==================== CFG 生成 ====================

func buildFFmpegParams(preset string) (string, string, error) {
	// 只支持两个预设配置
	// c1: libx264 软件编码（高质量）
	// n1: hevc_nvenc 硬件编码（NVIDIA GPU，HEVC/H.265）

	switch preset {
	case "c1":
		params := "-c:v libx264 -preset 1 -crf 4 -qmax 20 -g 120 -keyint_min 1 -pix_fmt yuv420p -x264-params ref=3:me=hex:subme=3:merange=12:b-adapt=1:aq-mode=2:aq-strength=0.9:no-fast-pskip=1"
		return "c1", params, nil
	case "n1":
		params := "-c:v hevc_nvenc -g 120 -preset medium -tune hq -rc constqp -qp 14 -pix_fmt yuv420p"
		return "n1", params, nil
	default:
		return "", "", fmt.Errorf("不支持的预设: %s (仅支持 c1 或 n1)", preset)
	}
}

func windowsToUnixPath(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

func containsCJK(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func generateCFG(demoPath, cfgPath, outputDir string, segments []Segment, targetName string, targetSlot int, cfg *Config) error {
	os.MkdirAll(filepath.Dir(cfgPath), 0755)
	os.MkdirAll(outputDir, 0755)

	demoName := strings.TrimSuffix(filepath.Base(demoPath), filepath.Ext(demoPath))

	// 获取预设配置
	presetName, ffmpegParams, err := buildFFmpegParams(cfg.VideoPreset)
	if err != nil {
		return err
	}

	var lines []string
	lines = append(lines, "r_show_build_info 0")
	lines = append(lines, "cl_trueview_show_status 0")
	lines = append(lines, "engine_no_focus_sleep 0")
	lines = append(lines, "cl_demo_predict 0")
	lines = append(lines, "spec_show_xray 0")
	lines = append(lines, "mirv_streams record screen enabled 1")
	lines = append(lines, fmt.Sprintf("mirv_streams record fps %d", cfg.RecordFPS))
	lines = append(lines, "mirv_streams record startMovieWav 1")
	// 使用预设配置生成 FFmpeg 命令
	lines = append(lines, fmt.Sprintf(`mirv_streams settings add ffmpeg %s "%s {QUOTE}{AFX_STREAM_PATH}.mp4{QUOTE}"`, presetName, ffmpegParams))
	lines = append(lines, fmt.Sprintf("mirv_streams record screen settings %s", presetName))
	lines = append(lines, "mirv_cmd clear")

	baseOutput := windowsToUnixPath(filepath.Join(outputDir, demoName))
	lines = append(lines, fmt.Sprintf(`mirv_streams record name "%s"`, baseOutput))

	const (
		initialTick = 64
		specDelay   = 4
	)

	// 转场模式：每个片段单独录制
	for idx, seg := range segments {
		var jumpTick int
		if idx == 0 {
			jumpTick = initialTick
		} else {
			jumpTick = segments[idx-1].EndTick + 10
		}

		lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "demo_gototick %d"`, jumpTick, seg.StartTick))
		lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "spec_mode 1"`, seg.StartTick))

		if targetSlot > 0 {
			lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "spec_player %d"`, seg.StartTick+specDelay, targetSlot))
		} else {
			lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "spec_player %s"`, seg.StartTick+specDelay, targetName))
		}

		lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "mirv_streams record start"`, seg.StartTick+specDelay+2))
		lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "mirv_streams record end"`, seg.EndTick+1))
	}
	lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "quit"`, segments[len(segments)-1].EndTick+10))

	if cfg.RecordVictimView {
		victimPreTicks := int(cfg.VictimPreSeconds * float64(cfg.Tickrate))
		victimPostTicks := int(cfg.VictimPostSeconds * float64(cfg.Tickrate))
		if victimPreTicks < 0 {
			victimPreTicks = 0
		}
		if victimPostTicks < 0 {
			victimPostTicks = 0
		}

		victimSegments := buildVictimSegments(segmentsToKills(segments), victimPreTicks, victimPostTicks)
		if len(victimSegments) == 0 {
			return fmt.Errorf("未生成有效片段")
		}

		victimCfgName := strings.TrimSuffix(filepath.Base(cfgPath), filepath.Ext(cfgPath)) + "_victim"
		victimCfgPath := filepath.Join(filepath.Dir(cfgPath), victimCfgName+".cfg")

		var victimLines []string
		victimLines = append(victimLines, "mirv_cmd clear")
		victimLines = append(victimLines, "r_show_build_info 0")
		victimLines = append(victimLines, "cl_trueview_show_status 0")
		victimLines = append(victimLines, "engine_no_focus_sleep 0")
		victimLines = append(victimLines, "cl_demo_predict 0")
		victimLines = append(victimLines, "spec_show_xray 0")
		victimLines = append(victimLines, "mirv_streams record screen enabled 1")
		victimLines = append(victimLines, fmt.Sprintf("mirv_streams record fps %d", cfg.RecordFPS))
		victimLines = append(victimLines, "mirv_streams record startMovieWav 1")
		victimLines = append(victimLines, fmt.Sprintf(`mirv_streams settings add ffmpeg %s "%s {QUOTE}{AFX_STREAM_PATH}.mp4{QUOTE}"`, presetName, ffmpegParams))
		victimLines = append(victimLines, fmt.Sprintf("mirv_streams record screen settings %s", presetName))

		baseOutputVictim := baseOutput + "_victim"
		victimLines = append(victimLines, fmt.Sprintf(`mirv_streams record name "%s"`, baseOutputVictim))
		const victimSpecDelay = 4
		prevEndTick := -1
		prevStartTick := -1
		lastRecordEndIdx := -1
		nearPreTicks := victimPreTicks
		nearPostTicks := victimPostTicks
		if nearPreTicks > 8 {
			nearPreTicks = 8
		}
		if nearPostTicks > 8 {
			nearPostTicks = 8
		}
		if nearPreTicks < 1 {
			nearPreTicks = 1
		}
		if nearPostTicks < 1 {
			nearPostTicks = 1
		}
		for _, seg := range victimSegments {
			killTick := seg.StartTick
			if len(seg.Kills) > 0 {
				killTick = seg.Kills[0].Tick
			}
			victimSlot := 0
			if len(seg.Kills) > 0 {
				victimSlot = seg.Kills[0].VictimID
			}
			if victimSlot <= 0 {
				return fmt.Errorf("未找到被击杀玩家")
			}

			startTick := seg.StartTick
			endTick := seg.EndTick
			useJump := true
			jumpTick := initialTick
			if prevEndTick >= 0 {
				if startTick <= prevEndTick+10 {
					useJump = false
					desiredStart := killTick - nearPreTicks
					if desiredStart < 0 {
						desiredStart = 0
					}
					desiredEnd := killTick + nearPostTicks
					newPrevEnd := desiredStart - 2
					if newPrevEnd < prevStartTick {
						newPrevEnd = prevStartTick
					}
					if lastRecordEndIdx >= 0 && newPrevEnd < prevEndTick {
						victimLines[lastRecordEndIdx] = fmt.Sprintf(`mirv_cmd addAtTick %d "mirv_streams record end"`, newPrevEnd+1)
						prevEndTick = newPrevEnd
					}
					if desiredStart < prevEndTick+2 {
						startTick = prevEndTick + 2
					} else {
						startTick = desiredStart
					}
					endTick = desiredEnd
					if endTick < startTick {
						endTick = startTick
					}
				} else {
					jumpTick = prevEndTick + 10
				}
			}

			if useJump {
				victimLines = append(victimLines, fmt.Sprintf(`mirv_cmd addAtTick %d "demo_gototick %d"`, jumpTick, startTick))
			}
			victimLines = append(victimLines, fmt.Sprintf(`mirv_cmd addAtTick %d "spec_mode 1"`, startTick))
			victimLines = append(victimLines, fmt.Sprintf(`mirv_cmd addAtTick %d "spec_player %d"`, startTick+victimSpecDelay, victimSlot))
			victimLines = append(victimLines, fmt.Sprintf(`mirv_cmd addAtTick %d "mirv_streams record start"`, startTick+victimSpecDelay+2))
			victimLines = append(victimLines, fmt.Sprintf(`mirv_cmd addAtTick %d "mirv_streams record end"`, endTick+1))
			lastRecordEndIdx = len(victimLines) - 1
			prevStartTick = startTick
			prevEndTick = endTick
		}

		if prevEndTick < 0 {
			prevEndTick = victimSegments[len(victimSegments)-1].EndTick
		}
		victimLines = append(victimLines, fmt.Sprintf(`mirv_cmd addAtTick %d "quit"`, prevEndTick+10))

		victimContent := strings.Join(victimLines, "\n") + "\n"
		if err := os.WriteFile(victimCfgPath, []byte(victimContent), 0644); err != nil {
			return fmt.Errorf("写入被害者 CFG 失败: %w", err)
		}
	}

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入 CFG 失败: %w", err)
	}

	return nil
}

// ==================== HLAE 启动 ====================

func launchHLAE(cfg *Config, demoPath, cfgName string) error {
	// 检查文件存在
	if _, err := os.Stat(cfg.HLAEExe); err != nil {
		return fmt.Errorf("HLAE 不存在: %s", cfg.HLAEExe)
	}
	if _, err := os.Stat(cfg.CS2Exe); err != nil {
		return fmt.Errorf("CS2 不存在: %s", cfg.CS2Exe)
	}

	hookDll := filepath.Join(filepath.Dir(cfg.HLAEExe), "x64", "AfxHookSource2.dll")
	if _, err := os.Stat(hookDll); err != nil {
		return fmt.Errorf("AfxHookSource2.dll 不存在: %s", hookDll)
	}

	cmdLine := fmt.Sprintf(`-insecure -novid -low -high +sv_lan 1 -coop_fullscreen -worldwide -console +playdemo "%s" +exec %s`, demoPath, cfgName)
	if cfg.LaunchResolution == "4:3" {
		cmdLine += " -w 1440 -h 1080"
	}

	args := []string{
		"-noGui", "-autoStart", "-noConfig",
		"-afxDisableSteamStorage", "-customLoader",
		"-hookDllPath", hookDll,
		"-programPath", cfg.CS2Exe,
		"-cmdLine", cmdLine,
	}

	cmd := exec.Command(cfg.HLAEExe, args...)

	// 设置环境变量
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("USRLOCALCSGO=%s", filepath.Dir(cfg.CfgDir)))

	printInfo("正在启动 HLAE 和 CS2...")
	return cmd.Start()
}

// ==================== 进程监控 ====================

func waitForCS2Completion(timeout time.Duration) error {
	printInfo("等待 CS2 启动...")
	startTime := time.Now()

	// 等待启动
	var cs2Started bool
	for time.Since(startTime) < 60*time.Second {
		if isCS2Running() {
			printSuccess("CS2 已启动")
			cs2Started = true
			break
		}
		time.Sleep(time.Second)
	}

	if !cs2Started {
		return fmt.Errorf("CS2 未在 60 秒内启动")
	}

	// 等待退出
	printInfo("等待录制完成...")
	for time.Since(startTime) < timeout {
		if !isCS2Running() {
			elapsed := int(time.Since(startTime).Seconds())
			printSuccess(fmt.Sprintf("录制完成 (用时: %d 秒)", elapsed))
			time.Sleep(3 * time.Second) // 等待文件写入
			return nil
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("超时: CS2 运行超过 %v", timeout)
}

func isCS2Running() bool {
	cmd := execCommandHidden("tasklist", "/FI", "IMAGENAME eq cs2.exe", "/NH")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "cs2.exe")
}

func killCS2Processes() {
	execCommandHidden("taskkill", "/F", "/IM", "cs2.exe").Run()
}

// ==================== 视频处理 ====================

func getVideoDuration(videoPath, ffmpegExe string) float64 {
	cmd := execCommandHidden(ffmpegExe, "-i", videoPath, "-f", "null", "-")
	output, _ := cmd.CombinedOutput()

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Duration:") {
			parts := strings.Split(line, "Duration:")
			if len(parts) > 1 {
				timePart := strings.Split(parts[1], ",")[0]
				timePart = strings.TrimSpace(timePart)

				// 解析 HH:MM:SS.ms
				timeFields := strings.Split(timePart, ":")
				if len(timeFields) == 3 {
					h, _ := strconv.ParseFloat(timeFields[0], 64)
					m, _ := strconv.ParseFloat(timeFields[1], 64)
					s, _ := strconv.ParseFloat(timeFields[2], 64)
					return h*3600 + m*60 + s
				}
			}
		}
	}
	return 0
}

func mergeAudioVideo(videoPath, audioPath, outputPath, ffmpegExe string) error {
	cmd := execCommandHidden(ffmpegExe,
		"-y",
		"-i", videoPath,
		"-i", audioPath,
		"-c:v", "copy",
		"-c:a", "aac",
		"-b:a", "192k",
		outputPath,
	)
	return cmd.Run()
}

func createTransitionsVideo(segments []string, outputPath, ffmpegExe string, duration float64, transType string, preset string) error {
	numSegs := len(segments)

	encodeArgs := buildTransitionEncodeArgs(preset)

	// 单个片段时，使用 FFmpeg 重新编码以确保格式正确
	if numSegs == 1 {
		args := []string{
			"-y",
			"-i", segments[0],
		}
		args = append(args, encodeArgs...)
		args = append(args, outputPath)
		cmd := execCommandHidden(ffmpegExe, args...)
		return cmd.Run()
	}

	// 获取时长
	durations := make([]float64, numSegs)
	for i, seg := range segments {
		durations[i] = getVideoDuration(seg, ffmpegExe)
	}

	// 构建转场命令
	var args []string
	args = append(args, "-y")

	for _, seg := range segments {
		args = append(args, "-i", seg)
	}

	// 构建 filter_complex
	var filters []string

	if numSegs == 2 {
		offset := durations[0] - duration
		filters = append(filters, fmt.Sprintf("[0:v][1:v]xfade=transition=%s:duration=%.2f:offset=%.2f[v]", transType, duration, offset))
		filters = append(filters, fmt.Sprintf("[0:a][1:a]acrossfade=d=%.2f[a]", duration))
	} else {
		// 多片段转场
		offsets := make([]float64, numSegs-1)
		current := 0.0
		for i := 0; i < numSegs-1; i++ {
			current += durations[i] - duration
			offsets[i] = current
		}

		// 视频转场
		filters = append(filters, fmt.Sprintf("[0:v][1:v]xfade=transition=%s:duration=%.2f:offset=%.2f[v01]", transType, duration, offsets[0]))
		for i := 1; i < numSegs-1; i++ {
			var prev, curr string
			if i == 1 {
				prev = "v01"
			} else {
				prev = fmt.Sprintf("v%d%d", i-1, i)
			}
			if i < numSegs-2 {
				curr = fmt.Sprintf("v%d%d", i, i+1)
			} else {
				curr = "v"
			}
			filters = append(filters, fmt.Sprintf("[%s][%d:v]xfade=transition=%s:duration=%.2f:offset=%.2f[%s]", prev, i+1, transType, duration, offsets[i], curr))
		}

		// 音频同步
		filters = append(filters, "[0:a]acopy[a0]")
		for i := 1; i < numSegs; i++ {
			filters = append(filters, fmt.Sprintf("[%d:a]atrim=start=%.2f,asetpts=PTS-STARTPTS[a%d]", i, duration, i))
		}
		audioInputs := ""
		for i := 0; i < numSegs; i++ {
			audioInputs += fmt.Sprintf("[a%d]", i)
		}
		filters = append(filters, fmt.Sprintf("%sconcat=n=%d:v=0:a=1[a]", audioInputs, numSegs))
	}

	args = append(args, "-filter_complex", strings.Join(filters, ";"))
	args = append(args, "-map", "[v]", "-map", "[a]")
	args = append(args, encodeArgs...)
	args = append(args, outputPath)

	cmd := execCommandHidden(ffmpegExe, args...)
	return cmd.Run()
}

func buildTransitionEncodeArgs(preset string) []string {
	switch preset {
	case "n1":
		return []string{
			"-c:v", "h264_nvenc",
			"-g", "120",
			"-preset", "p4",
			"-tune", "hq",
			"-rc", "vbr",
			"-cq", "19",
			"-pix_fmt", "yuv420p",
			"-profile:v", "high",
			"-c:a", "aac",
			"-b:a", "192k",
		}
	default:
		return []string{
			"-c:v", "libx264",
			"-crf", "23",
			"-preset", "medium",
			"-pix_fmt", "yuv420p",
			"-c:a", "aac",
			"-b:a", "192k",
		}
	}
}

func collectVideoFiles(baseDir string) ([]string, error) {
	if _, err := os.Stat(baseDir); err != nil {
		return nil, fmt.Errorf("输出目录不存在: %s", baseDir)
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("读取输出目录失败: %w", err)
	}

	var videoFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(strings.ToLower(entry.Name()), "take") && strings.HasSuffix(strings.ToLower(entry.Name()), ".mp4") {
			videoFiles = append(videoFiles, filepath.Join(baseDir, entry.Name()))
		}
	}

	if len(videoFiles) == 0 {
		return nil, fmt.Errorf("未找到录制视频文件")
	}

	sort.Strings(videoFiles)
	return videoFiles, nil
}

func processRecordings(outputDir, demoName, exeDir string, selectedRounds []int, cfg *Config, debugMode bool) (string, error) {
	// ffmpeg.exe 在 ffmpeg/bin
	ffmpegExe := resolveFFmpegExe(exeDir, cfg)

	baseDir := filepath.Join(outputDir, demoName)
	videoFiles, err := collectVideoFiles(baseDir)
	if err != nil {
		return "", err
	}
	printInfo(fmt.Sprintf("找到 %d 个击杀者片段", len(videoFiles)))

	if cfg.RecordVictimView {
		victimBaseDir := baseDir + "_victim"
		victimFiles, err := collectVideoFiles(victimBaseDir)
		if err != nil {
			return "", err
		}
		printInfo(fmt.Sprintf("找到 %d 个被害者片段", len(victimFiles)))
		videoFiles = append(videoFiles, victimFiles...)
	}

	// 创建临时目录
	tempDir := filepath.Join(outputDir, "_temp")
	os.MkdirAll(tempDir, 0755)

	// 合成各个片段
	printInfo("合成视频片段...")
	var mergedSegments []string
	for i, videoFile := range videoFiles {
		// 视频文件: baseDir/take0000.mp4
		// 音频文件: baseDir/take0000/audio.wav
		baseDir := filepath.Dir(videoFile)
		baseName := strings.TrimSuffix(filepath.Base(videoFile), filepath.Ext(videoFile))
		audioFile := filepath.Join(baseDir, baseName, "audio.wav")

		if _, err := os.Stat(videoFile); err != nil {
			printWarning(fmt.Sprintf("片段 %d 视频文件不存在: %s", i+1, videoFile))
			continue
		}
		if _, err := os.Stat(audioFile); err != nil {
			printWarning(fmt.Sprintf("片段 %d 音频文件不存在: %s", i+1, audioFile))
			continue
		}

		tempOutput := filepath.Join(tempDir, fmt.Sprintf("seg_%03d.mp4", i))

		if err := mergeAudioVideo(videoFile, audioFile, tempOutput, ffmpegExe); err != nil {
			printError(fmt.Sprintf("合成片段 %d 失败: %v", i+1, err))
			continue
		}

		mergedSegments = append(mergedSegments, tempOutput)
	}

	if len(mergedSegments) == 0 {
		return "", fmt.Errorf("没有成功合成的片段")
	}

	// 构建回合信息字符串
	roundsStr := ""
	if len(selectedRounds) == 1 {
		roundsStr = fmt.Sprintf("_R%d", selectedRounds[0])
	} else if len(selectedRounds) <= 5 {
		roundsStr = "_R"
		for i, r := range selectedRounds {
			if i > 0 {
				roundsStr += "-"
			}
			roundsStr += fmt.Sprintf("%d", r)
		}
	} else {
		roundsStr = fmt.Sprintf("_R%d-%d", selectedRounds[0], selectedRounds[len(selectedRounds)-1])
	}

	// 添加转场效果
	finalOutput := filepath.Join(outputDir, demoName+roundsStr+".mp4")
	printInfo("添加转场效果...")

	if err := createTransitionsVideo(mergedSegments, finalOutput, ffmpegExe, cfg.TransitionDuration, cfg.TransitionType, cfg.VideoPreset); err != nil {
		return "", fmt.Errorf("转场合成失败: %w", err)
	}

	// 获取文件大小
	info, _ := os.Stat(finalOutput)
	sizeMB := float64(info.Size()) / (1024 * 1024)

	fmt.Println()
	colorGreenBold.Printf("✓ 完成: %s\n", finalOutput)
	colorCyan.Printf("  文件大小: %.1f MB\n", sizeMB)

	// Debug 模式下保留临时文件
	if debugMode {
		printInfo("Debug 模式：保留所有临时文件")
		printInfo(fmt.Sprintf("  临时目录: %s", tempDir))
		printInfo(fmt.Sprintf("  录制文件目录: %s", baseDir))
		cfgPath := filepath.Join(cfg.CfgDir, "auto_"+demoName+".cfg")
		printInfo(fmt.Sprintf("  配置文件: %s", cfgPath))
		return finalOutput, nil
	}

	// 清理临时文件
	os.RemoveAll(tempDir)

	// 清理生成的 cfg 文件
	cfgPath := filepath.Join(cfg.CfgDir, "auto_"+demoName+".cfg")
	if err := os.Remove(cfgPath); err == nil {
		printInfo("已删除生成的 cfg 文件: " + cfgPath)
	}
	if cfg.RecordVictimView {
		victimCfgPath := filepath.Join(cfg.CfgDir, "auto_"+demoName+"_victim.cfg")
		if err := os.Remove(victimCfgPath); err == nil {
			printInfo("已删除生成的 cfg 文件: " + victimCfgPath)
		}
	}

	// 清理视频文件和对应的音频目录
	for _, videoFile := range videoFiles {
		// 删除视频文件
		os.Remove(videoFile)
		// 删除对应的音频目录
		baseDir := filepath.Dir(videoFile)
		baseName := strings.TrimSuffix(filepath.Base(videoFile), filepath.Ext(videoFile))
		audioDir := filepath.Join(baseDir, baseName)
		os.RemoveAll(audioDir)
	}
	printInfo("已清理临时文件")

	// 清理 demo 输出目录（包含全部 take 和音频子目录）
	if err := os.RemoveAll(baseDir); err == nil {
		printInfo("已清理输出目录: " + baseDir)
	}

	return finalOutput, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// ==================== 主函数 ====================

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "CS2 击杀集锦制作工具",
		Width:            1200,
		Height:           800,
		DisableResize:    false,
		Assets:           assets,
		BackgroundColour: &options.RGBA{R: 20, G: 20, B: 20, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind:             []interface{}{app},
	})
	if err != nil {
		printError(fmt.Sprintf("启动失败: %v", err))
	}
}
