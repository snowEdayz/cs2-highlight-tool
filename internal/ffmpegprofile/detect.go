package ffmpegprofile

import (
	"context"
	"cs2-highlight-tool-v2/internal/procutil"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type Capabilities struct {
	Encoders map[string]bool
	Errors   map[string]string
	TestedAt time.Time
}

type CommandContextFunc func(ctx context.Context, command string, args ...string) *exec.Cmd

func CapabilitiesFromEncoders(encoders []string) Capabilities {
	caps := Capabilities{Encoders: make(map[string]bool, len(encoders))}
	for _, encoder := range encoders {
		normalized := strings.ToLower(strings.TrimSpace(encoder))
		if normalized == "" {
			continue
		}
		caps.Encoders[normalized] = true
	}
	return caps
}

func (c Capabilities) AvailableEncoders() []string {
	if len(c.Encoders) == 0 {
		return nil
	}
	out := make([]string, 0, len(c.Encoders))
	for encoder, ok := range c.Encoders {
		if !ok {
			continue
		}
		out = append(out, encoder)
	}
	sort.Strings(out)
	return out
}

func (c Capabilities) HasEncoder(encoder string) bool {
	if len(c.Encoders) == 0 {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(encoder))
	if normalized == "" {
		return false
	}
	return c.Encoders[normalized]
}

func (c Capabilities) IsEmpty() bool {
	for _, ok := range c.Encoders {
		if ok {
			return false
		}
	}
	return true
}

func DetectCapabilities(ctx context.Context, ffmpegExe string, cmdFactory CommandContextFunc) (Capabilities, error) {
	exe := strings.TrimSpace(ffmpegExe)
	if exe == "" {
		return Capabilities{}, fmt.Errorf("ffmpeg 路径为空")
	}
	if _, err := os.Stat(exe); err != nil {
		return Capabilities{}, fmt.Errorf("ffmpeg 不存在: %s", exe)
	}
	if cmdFactory == nil {
		cmdFactory = exec.CommandContext
	}
	if ctx == nil {
		ctx = context.Background()
	}

	caps := Capabilities{
		Encoders: make(map[string]bool),
		Errors:   make(map[string]string),
		TestedAt: time.Now(),
	}
	probeOrder := []string{"hevc_nvenc", "hevc_amf", "hevc_qsv", "h264_nvenc", "h264_amf", "h264_qsv", "libx264"}

	for _, encoder := range probeOrder {
		normalized := strings.ToLower(strings.TrimSpace(encoder))
		if normalized == "" {
			continue
		}
		available, err := probeEncoder(ctx, cmdFactory, exe, normalized)
		caps.Encoders[normalized] = available
		if err != nil {
			caps.Errors[normalized] = err.Error()
		}
	}

	return caps, nil
}

func probeEncoder(ctx context.Context, cmdFactory CommandContextFunc, ffmpegExe string, encoder string) (bool, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := cmdFactory(
		probeCtx,
		ffmpegExe,
		"-hide_banner",
		"-loglevel", "error",
		"-nostdin",
		"-f", "lavfi",
		"-i", "color=size=64x64:rate=30:duration=0.1",
		"-an",
		"-frames:v", "1",
		"-c:v", encoder,
		"-f", "null",
		"-",
	)
	procutil.ConfigureNoWindowProcess(cmd)
	out, err := cmd.CombinedOutput()
	if probeCtx.Err() == context.DeadlineExceeded {
		return false, fmt.Errorf("probe 超时")
	}
	if err != nil {
		text := strings.TrimSpace(string(out))
		if text == "" {
			text = err.Error()
		}
		return false, fmt.Errorf("probe 失败: %s", text)
	}
	return true, nil
}
