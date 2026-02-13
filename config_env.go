package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

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
	PlayTeamVoice      bool    `json:"play_team_voice"`
	KillerPreSeconds   float64 `json:"killer_pre_seconds"`
	KillerPostSeconds  float64 `json:"killer_post_seconds"`
	VictimPreSeconds   float64 `json:"victim_pre_seconds"`
	VictimPostSeconds  float64 `json:"victim_post_seconds"`
}

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
		PlayTeamVoice:      false,
		KillerPreSeconds:   4,
		KillerPostSeconds:  4,
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
		"play_team_voice":     cfg.PlayTeamVoice,
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
