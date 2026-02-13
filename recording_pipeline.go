package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

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
	// cmdLine := fmt.Sprintf(`-insecure -novid -low -high +sv_lan 1 -worldwide -console +playdemo "%s" +exec %s`, demoPath, cfgName)
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

	cmd := execCommandHidden(cfg.HLAEExe, args...)

	// 设置环境变量
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("USRLOCALCSGO=%s", filepath.Dir(cfg.CfgDir)))

	printInfo("正在启动 HLAE 和 CS2...")
	return cmd.Start()
}

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
