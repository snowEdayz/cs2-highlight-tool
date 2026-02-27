package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func normalizeVictimSegments(victimSegments []Segment, mergeWindowTicks int) []Segment {
	if len(victimSegments) == 0 {
		return nil
	}
	if mergeWindowTicks < 1 {
		mergeWindowTicks = 1
	}

	normalized := make([]Segment, 0, len(victimSegments))
	lastKeptTick := -1
	for _, seg := range victimSegments {
		killTick := seg.StartTick
		if len(seg.Kills) > 0 {
			killTick = seg.Kills[0].Tick
		}
		if lastKeptTick >= 0 && killTick-lastKeptTick <= mergeWindowTicks {
			// 同一小时间窗口内多人死亡，仅保留第一个视角，避免频繁切换导致异常。
			continue
		}
		normalized = append(normalized, seg)
		lastKeptTick = killTick
	}
	return normalized
}

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

func generateCFG(demoPath, cfgPath, outputDir string, segments []Segment, targetName string, targetSlot int, cfg *Config) error {
	os.MkdirAll(filepath.Dir(cfgPath), 0755)
	os.MkdirAll(outputDir, 0755)

	// 始终基于配置重建击杀者片段，确保 killer_pre_seconds / killer_post_seconds 生效。
	killerPreTicks := int(cfg.KillerPreSeconds * float64(cfg.Tickrate))
	killerPostTicks := int(cfg.KillerPostSeconds * float64(cfg.Tickrate))
	if killerPreTicks < 0 {
		killerPreTicks = 0
	}
	if killerPostTicks < 0 {
		killerPostTicks = 0
	}
	segments = buildSegments(segmentsToKills(segments), killerPreTicks, killerPostTicks)
	if len(segments) == 0 {
		return fmt.Errorf("未生成有效片段")
	}

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
	if cfg.PlayTeamVoice {
		lines = append(lines, "tv_listen_voice_indices -1")
	} else {
		lines = append(lines, "tv_listen_voice_indices 0")
	}

	lines = append(lines, "mirv_streams record screen enabled 1")
	lines = append(lines, fmt.Sprintf("mirv_streams record fps %d", cfg.RecordFPS))
	lines = append(lines, "mirv_streams record startMovieWav 1")
	// 使用预设配置生成 FFmpeg 命令
	lines = append(lines, fmt.Sprintf(`mirv_streams settings add ffmpeg %s "%s {QUOTE}{AFX_STREAM_PATH}.mp4{QUOTE}"`, presetName, ffmpegParams))
	lines = append(lines, fmt.Sprintf("mirv_streams record screen settings %s", presetName))

	baseOutput := windowsToUnixPath(filepath.Join(outputDir, demoName))
	lines = append(lines, fmt.Sprintf(`mirv_streams record name "%s"`, baseOutput))

	const (
		initialTick = 64
		specDelay   = 4
	)
	// demo UI 关闭flag
	var demoUIFlag int = 0
	// 转场模式：每个片段单独录制
	for idx, seg := range segments {
		var jumpTick int
		if idx == 0 {
			jumpTick = initialTick
		} else {
			jumpTick = segments[idx-1].EndTick + 10
		}

		if demoUIFlag == 0 {
			lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "demoui 3"`, jumpTick-5))
			demoUIFlag = 1
		}

		lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "demo_gototick %d"`, jumpTick, seg.StartTick-10))
		lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "spec_mode 1"`, seg.StartTick+1))

		if targetSlot > 0 {
			lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "spec_player %d"`, seg.StartTick+specDelay, targetSlot))
		} else {
			lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "spec_player %s"`, seg.StartTick+specDelay, targetName))
		}

		lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "mirv_streams record start"`, seg.StartTick+specDelay+2))
		lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "mirv_streams record end"`, seg.EndTick+1))
	}
	lines = append(lines, fmt.Sprintf(`mirv_cmd addAtTick %d "quit"`, segments[len(segments)-1].EndTick+10+2))

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
		mergeWindowTicks := nearPreTicks + nearPostTicks
		victimSegments = normalizeVictimSegments(victimSegments, mergeWindowTicks)
		if len(victimSegments) == 0 {
			return fmt.Errorf("未生成有效片段")
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
				if demoUIFlag == 1 {
					victimLines = append(victimLines, fmt.Sprintf(`mirv_cmd addAtTick %d "demoui 3"`, jumpTick-5))
					demoUIFlag = 0
				}
				victimLines = append(victimLines, fmt.Sprintf(`mirv_cmd addAtTick %d "demo_gototick %d"`, jumpTick, startTick))
			}
			victimLines = append(victimLines, fmt.Sprintf(`mirv_cmd addAtTick %d "spec_mode 1"`, startTick+1))
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
		victimLines = append(victimLines, fmt.Sprintf(`mirv_cmd addAtTick %d "quit"`, prevEndTick+10+2))

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
