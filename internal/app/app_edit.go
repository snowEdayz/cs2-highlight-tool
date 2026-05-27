package app

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cs2-highlight-tool-v2/internal/ffmpegprofile"
)

type EditConcatClip struct {
	VideoPath string  `json:"video_path"`
	Duration  float64 `json:"duration"`
}

type EditConcatTransition struct {
	Type       string  `json:"type"`
	Duration   float64 `json:"duration"`
	AfterIndex int     `json:"after_index,omitempty"`
}

type EditConcatRequest struct {
	Clips       []EditConcatClip       `json:"clips"`
	Transitions []EditConcatTransition `json:"transitions"`
}

type resolvedEditClip struct {
	VideoPath string
	Duration  float64
}

type editEncodeSettings struct {
	FPS         int
	Quality     string
	VideoPreset string
	Caps        ffmpegprofile.Capabilities
}

const (
	defaultEditTransitionDuration = 0.3
	minEditTransitionDuration     = 0.05
	maxEditTransitionDuration     = 5.0
)

func (a *App) ConcatEditClips(request EditConcatRequest) (string, error) {
	resolvedClips, err := a.resolveEditClips(request.Clips)
	if err != nil {
		return "", err
	}

	transitionByIndex, err := normalizeEditTransitions(len(resolvedClips), request.Transitions)
	if err != nil {
		return "", err
	}

	outputDir, ffmpegExe, encode := a.resolveEditOutputPaths()
	if ffmpegExe == "" {
		return "", fmt.Errorf("ffmpeg not found")
	}
	if _, err := os.Stat(ffmpegExe); err != nil {
		return "", fmt.Errorf("ffmpeg not found at %s", ffmpegExe)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output directory failed: %w", err)
	}

	outputPath := filepath.Join(outputDir, fmt.Sprintf("edit_%s.mp4", time.Now().Format("20060102_150405")))
	tracker := newComposeProgressTracker(a, editComposeStageCount(len(resolvedClips), len(transitionByIndex) > 0))

	if len(transitionByIndex) == 0 {
		if _, err := concatSimple(ffmpegExe, resolvedClips, outputPath, encode, tracker); err != nil {
			tracker.fail(err)
			return "", err
		}
	} else {
		if _, err := concatWithTransitions(ffmpegExe, resolvedClips, transitionByIndex, outputPath, encode, tracker); err != nil {
			tracker.fail(err)
			return "", err
		}
	}

	if _, statErr := os.Stat(outputPath); statErr != nil {
		tracker.fail(statErr)
		return "", fmt.Errorf("output video not created: %w", statErr)
	}

	a.addEditedHistoryEntry(outputPath, "edit_timeline")
	tracker.complete()
	return outputPath, nil
}

func (a *App) resolveEditClips(input []EditConcatClip) ([]resolvedEditClip, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("no clips provided")
	}

	ffprobeExe := a.resolveFFprobeExe()
	resolved := make([]resolvedEditClip, 0, len(input))
	for i, clip := range input {
		p := strings.TrimSpace(clip.VideoPath)
		if p == "" {
			return nil, fmt.Errorf("clip %d video path is empty", i+1)
		}
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("clip %d video file not found: %s", i+1, p)
		}

		duration := clip.Duration
		if duration <= 0 {
			if ffprobeExe == "" {
				return nil, fmt.Errorf("clip %d duration is invalid and ffprobe not found", i+1)
			}
			if _, err := os.Stat(ffprobeExe); err != nil {
				return nil, fmt.Errorf("ffprobe not found at %s", ffprobeExe)
			}
			probed, err := probeDurationByFFprobe(ffprobeExe, p)
			if err != nil {
				return nil, fmt.Errorf("clip %d probe duration failed: %w", i+1, err)
			}
			duration = probed
		}
		if duration <= 0 {
			return nil, fmt.Errorf("clip %d duration must be > 0", i+1)
		}

		resolved = append(resolved, resolvedEditClip{
			VideoPath: p,
			Duration:  math.Round(duration*1000) / 1000,
		})
	}
	return resolved, nil
}

func normalizeEditTransitions(clipCount int, input []EditConcatTransition) (map[int]EditConcatTransition, error) {
	result := make(map[int]EditConcatTransition)
	if clipCount <= 1 {
		if len(input) > 0 {
			return nil, fmt.Errorf("transitions require at least 2 clips")
		}
		return result, nil
	}
	if len(input) == 0 {
		return result, nil
	}

	hasNonZeroAfter := false
	for _, transition := range input {
		if transition.AfterIndex > 0 {
			hasNonZeroAfter = true
			break
		}
	}
	legacySequential := len(input) == clipCount-1 && !hasNonZeroAfter

	if legacySequential {
		for i, transition := range input {
			normalized, err := normalizeTransition(transition)
			if err != nil {
				return nil, fmt.Errorf("transition %d invalid: %w", i+1, err)
			}
			normalized.AfterIndex = i
			result[i] = normalized
		}
		return result, nil
	}

	for i, transition := range input {
		normalized, err := normalizeTransition(transition)
		if err != nil {
			return nil, fmt.Errorf("transition %d invalid: %w", i+1, err)
		}
		if normalized.AfterIndex < 0 || normalized.AfterIndex >= clipCount-1 {
			return nil, fmt.Errorf("transition %d after_index out of range: %d", i+1, normalized.AfterIndex)
		}
		if _, exists := result[normalized.AfterIndex]; exists {
			return nil, fmt.Errorf("duplicate transition for gap index %d", normalized.AfterIndex)
		}
		result[normalized.AfterIndex] = normalized
	}

	return result, nil
}

func normalizeTransition(input EditConcatTransition) (EditConcatTransition, error) {
	transition := input
	transition.Type = strings.ToLower(strings.TrimSpace(transition.Type))
	if transition.Type == "" {
		transition.Type = "fade"
	}
	if transition.Type != "fade" {
		return EditConcatTransition{}, fmt.Errorf("unsupported transition type: %s", transition.Type)
	}

	d := transition.Duration
	if d <= 0 {
		d = defaultEditTransitionDuration
	}
	if d < minEditTransitionDuration || d > maxEditTransitionDuration {
		return EditConcatTransition{}, fmt.Errorf("transition duration out of range: %.3f", d)
	}
	transition.Duration = math.Round(d*1000) / 1000
	return transition, nil
}

func concatSimple(
	ffmpegExe string,
	clips []resolvedEditClip,
	outputPath string,
	encode editEncodeSettings,
	tracker *composeProgressTracker,
) ([]byte, error) {
	listPath := outputPath + ".concat.txt"
	defer os.Remove(listPath)

	var lines []string
	for _, clip := range clips {
		absPath, err := filepath.Abs(strings.TrimSpace(clip.VideoPath))
		if err != nil {
			return nil, fmt.Errorf("resolve clip path failed: %w", err)
		}
		lines = append(lines, fmt.Sprintf("file '%s'", strings.ReplaceAll(absPath, "'", "\\'")))
	}
	if err := os.WriteFile(listPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return nil, fmt.Errorf("write concat list failed: %w", err)
	}

	if tracker != nil {
		tracker.stageStart("合成输出")
	}
	stageDuration := totalClipDuration(clips)
	profiles := buildEditRetryProfiles(encode)
	var lastOut []byte
	var lastErr error
	for _, profile := range profiles {
		videoArgs, err := ffmpegprofile.BuildEditEncodeArgs(profile.ID, encode.Quality)
		if err != nil {
			lastErr = err
			continue
		}
		args := []string{
			"-f", "concat",
			"-safe", "0",
			"-i", listPath,
			"-vf", fmt.Sprintf("settb=AVTB,setpts=PTS-STARTPTS,fps=%d,format=yuv420p", encode.FPS),
			"-af", "asetpts=PTS-STARTPTS,aformat=sample_rates=48000:channel_layouts=stereo",
		}
		args = append(args, videoArgs...)
		args = append(args,
			"-c:a", "aac",
			"-b:a", "192k",
			"-movflags", "+faststart",
			"-y",
			outputPath,
		)
		cmd := ffmpegCommand(ffmpegExe, withFFmpegProgressArgs(args)...)
		configureNoWindowProcess(cmd)
		out, err := runFFmpegCommandWithProgress(cmd, stageDuration, tracker)
		if err == nil {
			if tracker != nil {
				tracker.stageDone()
			}
			return out, nil
		}
		lastOut = out
		lastErr = fmt.Errorf("[%s] %w", profile.ID, err)
	}
	return lastOut, fmt.Errorf("ffmpeg concat failed: %w: %s", lastErr, strings.TrimSpace(string(lastOut)))
}

func concatWithTransitions(
	ffmpegExe string,
	clips []resolvedEditClip,
	transitionByIndex map[int]EditConcatTransition,
	outputPath string,
	encode editEncodeSettings,
	tracker *composeProgressTracker,
) ([]byte, error) {
	if len(clips) < 2 {
		return nil, fmt.Errorf("at least 2 clips are required for transitions")
	}

	workDir := outputPath + ".work"
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("create transition work directory failed: %w", err)
	}
	defer os.RemoveAll(workDir)

	normalizedPaths := make([]string, 0, len(clips))
	normalizedDurations := make([]float64, 0, len(clips))
	for i, clip := range clips {
		normalizedPath := filepath.Join(workDir, fmt.Sprintf("clip_%03d.mp4", i))
		stageName := fmt.Sprintf("预处理片段 %d/%d", i+1, len(clips))
		if err := normalizeTransitionInputClip(ffmpegExe, clip.VideoPath, normalizedPath, encode, clip.Duration, tracker, stageName); err != nil {
			return nil, fmt.Errorf("normalize clip %d failed: %w", i+1, err)
		}
		normalizedPaths = append(normalizedPaths, normalizedPath)
		normalizedDurations = append(normalizedDurations, clip.Duration)
	}

	currentPath := normalizedPaths[0]
	currentDuration := normalizedDurations[0]
	for gapIndex := 0; gapIndex < len(normalizedPaths)-1; gapIndex++ {
		nextPath := normalizedPaths[gapIndex+1]
		nextDuration := normalizedDurations[gapIndex+1]
		stagePath := filepath.Join(workDir, fmt.Sprintf("stage_%03d.mp4", gapIndex))

		if transition, ok := transitionByIndex[gapIndex]; ok {
			if transition.Duration >= currentDuration || transition.Duration >= nextDuration {
				return nil, fmt.Errorf(
					"transition duration %.3f exceeds clip durations at gap %d (left=%.3f right=%.3f)",
					transition.Duration,
					gapIndex,
					currentDuration,
					nextDuration,
				)
			}
			stageName := fmt.Sprintf("应用转场 %d/%d", gapIndex+1, len(normalizedPaths)-1)
			stageDuration := currentDuration + nextDuration - transition.Duration
			if err := applyFadeTransition(ffmpegExe, currentPath, nextPath, currentDuration, transition.Duration, stagePath, encode, stageDuration, tracker, stageName); err != nil {
				return nil, fmt.Errorf("apply transition at gap %d failed: %w", gapIndex, err)
			}
			currentDuration = currentDuration + nextDuration - transition.Duration
		} else {
			stageName := fmt.Sprintf("拼接片段 %d/%d", gapIndex+1, len(normalizedPaths)-1)
			stageDuration := currentDuration + nextDuration
			if err := concatHardCutPair(ffmpegExe, currentPath, nextPath, stagePath, encode, stageDuration, tracker, stageName); err != nil {
				return nil, fmt.Errorf("concat hard cut at gap %d failed: %w", gapIndex, err)
			}
			currentDuration = currentDuration + nextDuration
		}
		currentPath = stagePath
	}

	if tracker != nil {
		tracker.stageStart("写出最终文件")
	}
	args := []string{
		"-y",
		"-i", currentPath,
		"-c", "copy",
		"-movflags", "+faststart",
		outputPath,
	}
	cmd := ffmpegCommand(ffmpegExe, withFFmpegProgressArgs(args)...)
	configureNoWindowProcess(cmd)
	out, err := runFFmpegCommandWithProgress(cmd, currentDuration, tracker)
	if err != nil {
		return out, fmt.Errorf("finalize output failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if tracker != nil {
		tracker.stageDone()
	}
	return out, nil
}

func normalizeTransitionInputClip(
	ffmpegExe string,
	inputPath string,
	outputPath string,
	encode editEncodeSettings,
	stageDuration float64,
	tracker *composeProgressTracker,
	stageName string,
) error {
	if tracker != nil {
		tracker.stageStart(stageName)
	}
	profiles := buildEditRetryProfiles(encode)
	var lastOut []byte
	var lastErr error
	for _, profile := range profiles {
		videoArgs, err := ffmpegprofile.BuildEditEncodeArgs(profile.ID, encode.Quality)
		if err != nil {
			lastErr = err
			continue
		}
		args := []string{
			"-y",
			"-i", inputPath,
			"-map", "0:v:0",
			"-map", "0:a:0",
			"-vf", fmt.Sprintf("settb=AVTB,setpts=PTS-STARTPTS,fps=%d,format=yuv420p", encode.FPS),
			"-af", "asetpts=PTS-STARTPTS,aformat=sample_rates=48000:channel_layouts=stereo",
		}
		args = append(args, videoArgs...)
		args = append(args,
			"-c:a", "aac",
			"-b:a", "192k",
			"-movflags", "+faststart",
			"-shortest",
			outputPath,
		)
		cmd := ffmpegCommand(ffmpegExe, withFFmpegProgressArgs(args)...)
		configureNoWindowProcess(cmd)
		out, err := runFFmpegCommandWithProgress(cmd, stageDuration, tracker)
		if err == nil {
			if tracker != nil {
				tracker.stageDone()
			}
			return nil
		}
		lastOut = out
		lastErr = fmt.Errorf("[%s] %w", profile.ID, err)
	}
	return fmt.Errorf("ffmpeg normalize failed: %w: %s", lastErr, strings.TrimSpace(string(lastOut)))
}

func applyFadeTransition(
	ffmpegExe string,
	leftPath string,
	rightPath string,
	leftDuration float64,
	transitionDuration float64,
	outputPath string,
	encode editEncodeSettings,
	stageDuration float64,
	tracker *composeProgressTracker,
	stageName string,
) error {
	if tracker != nil {
		tracker.stageStart(stageName)
	}
	offset := math.Round((leftDuration-transitionDuration)*1000) / 1000
	filter := fmt.Sprintf(
		"[0:v]settb=AVTB,setpts=PTS-STARTPTS,fps=%d,format=yuv420p[v0];"+
			"[1:v]settb=AVTB,setpts=PTS-STARTPTS,fps=%d,format=yuv420p[v1];"+
			"[v0][v1]xfade=transition=fade:duration=%.3f:offset=%.3f[v];"+
			"[0:a]asetpts=PTS-STARTPTS,aformat=sample_rates=48000:channel_layouts=stereo[a0];"+
			"[1:a]asetpts=PTS-STARTPTS,aformat=sample_rates=48000:channel_layouts=stereo[a1];"+
			"[a0][a1]acrossfade=d=%.3f:c1=tri:c2=tri[a]",
		encode.FPS,
		encode.FPS,
		transitionDuration,
		offset,
		transitionDuration,
	)

	profiles := buildEditRetryProfiles(encode)
	var lastOut []byte
	var lastErr error
	for _, profile := range profiles {
		videoArgs, err := ffmpegprofile.BuildEditEncodeArgs(profile.ID, encode.Quality)
		if err != nil {
			lastErr = err
			continue
		}
		args := []string{
			"-y",
			"-i", leftPath,
			"-i", rightPath,
			"-filter_complex", filter,
			"-map", "[v]",
			"-map", "[a]",
		}
		args = append(args, videoArgs...)
		args = append(args,
			"-c:a", "aac",
			"-b:a", "192k",
			"-movflags", "+faststart",
			outputPath,
		)
		cmd := ffmpegCommand(ffmpegExe, withFFmpegProgressArgs(args)...)
		configureNoWindowProcess(cmd)
		out, err := runFFmpegCommandWithProgress(cmd, stageDuration, tracker)
		if err == nil {
			if tracker != nil {
				tracker.stageDone()
			}
			return nil
		}
		lastOut = out
		lastErr = fmt.Errorf("[%s] %w", profile.ID, err)
	}
	return fmt.Errorf("ffmpeg transition failed: %w: %s", lastErr, strings.TrimSpace(string(lastOut)))
}

func concatHardCutPair(
	ffmpegExe string,
	leftPath string,
	rightPath string,
	outputPath string,
	encode editEncodeSettings,
	stageDuration float64,
	tracker *composeProgressTracker,
	stageName string,
) error {
	if tracker != nil {
		tracker.stageStart(stageName)
	}
	listPath := outputPath + ".list.txt"
	defer os.Remove(listPath)

	leftAbs, err := filepath.Abs(leftPath)
	if err != nil {
		return fmt.Errorf("resolve left clip path failed: %w", err)
	}
	rightAbs, err := filepath.Abs(rightPath)
	if err != nil {
		return fmt.Errorf("resolve right clip path failed: %w", err)
	}
	content := []string{
		fmt.Sprintf("file '%s'", strings.ReplaceAll(leftAbs, "'", "\\'")),
		fmt.Sprintf("file '%s'", strings.ReplaceAll(rightAbs, "'", "\\'")),
	}
	if err := os.WriteFile(listPath, []byte(strings.Join(content, "\n")), 0644); err != nil {
		return fmt.Errorf("write pair concat list failed: %w", err)
	}

	args := []string{
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", listPath,
		"-c", "copy",
		"-movflags", "+faststart",
		outputPath,
	}
	cmd := ffmpegCommand(ffmpegExe, withFFmpegProgressArgs(args)...)
	configureNoWindowProcess(cmd)
	out, err := runFFmpegCommandWithProgress(cmd, stageDuration, tracker)
	if err == nil {
		if tracker != nil {
			tracker.stageDone()
		}
		return nil
	}

	profiles := buildEditRetryProfiles(encode)
	var fallbackOut []byte
	var fallbackErr error
	for _, profile := range profiles {
		videoArgs, buildErr := ffmpegprofile.BuildEditEncodeArgs(profile.ID, encode.Quality)
		if buildErr != nil {
			fallbackErr = buildErr
			continue
		}
		args := []string{
			"-y",
			"-i", leftPath,
			"-i", rightPath,
			"-filter_complex", "[0:v][0:a][1:v][1:a]concat=n=2:v=1:a=1[v][a]",
			"-map", "[v]",
			"-map", "[a]",
		}
		args = append(args, videoArgs...)
		args = append(args,
			"-c:a", "aac",
			"-b:a", "192k",
			"-movflags", "+faststart",
			outputPath,
		)
		fallbackCmd := ffmpegCommand(ffmpegExe, withFFmpegProgressArgs(args)...)
		configureNoWindowProcess(fallbackCmd)
		currentOut, currentErr := runFFmpegCommandWithProgress(fallbackCmd, stageDuration, tracker)
		if currentErr == nil {
			if tracker != nil {
				tracker.stageDone()
			}
			return nil
		}
		fallbackOut = currentOut
		fallbackErr = fmt.Errorf("[%s] %w", profile.ID, currentErr)
	}
	return fmt.Errorf(
		"ffmpeg hard-cut concat failed: %w: %s; fallback failed: %w: %s",
		err,
		strings.TrimSpace(string(out)),
		fallbackErr,
		strings.TrimSpace(string(fallbackOut)),
	)
}

func editComposeStageCount(clipCount int, withTransitions bool) int {
	if clipCount <= 0 {
		return 1
	}
	if !withTransitions {
		return 1
	}
	return clipCount + (clipCount - 1) + 1
}

func totalClipDuration(clips []resolvedEditClip) float64 {
	total := 0.0
	for _, clip := range clips {
		total += clip.Duration
	}
	return total
}
