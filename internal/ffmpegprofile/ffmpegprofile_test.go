package ffmpegprofile

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeUserPreset(t *testing.T) {
	if got := NormalizeUserPreset("N1"); got != UserPresetN1 {
		t.Fatalf("NormalizeUserPreset(N1) = %q", got)
	}
	if got := NormalizeUserPreset("bad"); got != UserPresetAuto {
		t.Fatalf("NormalizeUserPreset(bad) = %q", got)
	}
}

func TestResolveProfile_AutoPriority(t *testing.T) {
	caps := CapabilitiesFromEncoders([]string{"h264_nvenc", "libx264"})
	resolved := ResolveProfile(UserPresetAuto, caps)
	if resolved.SelectedProfile.ID != internalPresetN1H264 {
		t.Fatalf("auto selected = %q, want %q", resolved.SelectedProfile.ID, internalPresetN1H264)
	}

	caps2 := CapabilitiesFromEncoders([]string{"hevc_qsv", "libx264"})
	resolved2 := ResolveProfile(UserPresetAuto, caps2)
	if resolved2.SelectedProfile.ID != UserPresetI1 {
		t.Fatalf("auto selected = %q, want %q", resolved2.SelectedProfile.ID, UserPresetI1)
	}
}

func TestResolveProfile_ManualFallback(t *testing.T) {
	caps := CapabilitiesFromEncoders([]string{"h264_amf", "libx264"})
	resolved := ResolveProfile(UserPresetA1, caps)
	if resolved.SelectedProfile.ID != internalPresetA1H264 {
		t.Fatalf("manual a1 selected = %q, want %q", resolved.SelectedProfile.ID, internalPresetA1H264)
	}

	caps2 := CapabilitiesFromEncoders([]string{"libx264"})
	resolved2 := ResolveProfile(UserPresetN1, caps2)
	if resolved2.SelectedProfile.ID != UserPresetC1 {
		t.Fatalf("manual n1 selected = %q, want %q", resolved2.SelectedProfile.ID, UserPresetC1)
	}
}

func TestResolveProfile_ManualWithoutCapabilitiesKeepsUserPreset(t *testing.T) {
	resolved := ResolveProfile(UserPresetN1, CapabilitiesFromEncoders(nil))
	if resolved.SelectedProfile.ID != UserPresetN1 {
		t.Fatalf("manual n1 with empty capabilities selected = %q, want %q", resolved.SelectedProfile.ID, UserPresetN1)
	}
}

func TestBuildRetryChain_AutoAndManual(t *testing.T) {
	caps := CapabilitiesFromEncoders([]string{"hevc_nvenc", "h264_nvenc", "libx264"})
	manual := BuildRetryChain(UserPresetN1, caps)
	if len(manual) != 3 {
		t.Fatalf("manual retry len=%d want 3", len(manual))
	}
	if manual[0].ID != UserPresetN1 || manual[1].ID != internalPresetN1H264 || manual[2].ID != UserPresetC1 {
		t.Fatalf("manual retry chain = %+v", manual)
	}

	auto := BuildRetryChain(UserPresetAuto, caps)
	if len(auto) == 0 || auto[len(auto)-1].ID != UserPresetC1 {
		t.Fatalf("auto retry chain must end with c1: %+v", auto)
	}
}

func TestBuildRetryChain_ManualIncludesUserPresetEvenIfDetectionSaysUnavailable(t *testing.T) {
	caps := CapabilitiesFromEncoders([]string{"h264_nvenc", "libx264"})
	manual := BuildRetryChain(UserPresetN1, caps)
	if len(manual) != 3 {
		t.Fatalf("manual retry len=%d want 3", len(manual))
	}
	if manual[0].ID != UserPresetN1 || manual[1].ID != internalPresetN1H264 || manual[2].ID != UserPresetC1 {
		t.Fatalf("manual retry chain = %+v", manual)
	}
}

func TestBuildEditEncodeArgs(t *testing.T) {
	args, err := BuildEditEncodeArgs(UserPresetN1, EditQualityHigh)
	if err != nil {
		t.Fatalf("BuildEditEncodeArgs: %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "hevc_nvenc") {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestDetectCapabilities_UsesProbeResults(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "ffmpeg.exe")
	if err := os.WriteFile(exe, []byte("stub"), 0755); err != nil {
		t.Fatalf("write exe: %v", err)
	}
	cmdFactory := func(ctx context.Context, command string, args ...string) *exec.Cmd {
		all := append([]string{"-test.run=TestHelperProcessDetect", "--", command}, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], all...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS_FFMPEGPROFILE=1",
			"FFMPEG_PROFILE_AVAILABLE=hevc_nvenc,h264_nvenc,libx264",
		)
		return cmd
	}
	caps, err := DetectCapabilities(context.Background(), exe, cmdFactory)
	if err != nil {
		t.Fatalf("DetectCapabilities: %v", err)
	}
	if !caps.HasEncoder("hevc_nvenc") {
		t.Fatalf("expected hevc_nvenc available: %+v", caps.Encoders)
	}
	if caps.HasEncoder("hevc_amf") {
		t.Fatalf("expected hevc_amf unavailable: %+v", caps.Encoders)
	}
	if _, ok := caps.Errors["hevc_amf"]; !ok {
		t.Fatalf("expected error for hevc_amf")
	}
}

func TestHelperProcessDetect(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS_FFMPEGPROFILE") != "1" {
		return
	}
	if len(os.Args) < 2 {
		os.Exit(2)
	}
	args := os.Args
	sep := -1
	for i, arg := range args {
		if arg == "--" {
			sep = i
			break
		}
	}
	if sep < 0 || sep+1 >= len(args) {
		_, _ = fmt.Fprintln(os.Stderr, "missing separator")
		os.Exit(2)
	}
	ffArgs := args[sep+2:]
	encoder := ""
	for i := 0; i < len(ffArgs)-1; i++ {
		if ffArgs[i] == "-c:v" {
			encoder = strings.ToLower(strings.TrimSpace(ffArgs[i+1]))
			break
		}
	}
	if encoder == "" {
		_, _ = fmt.Fprintln(os.Stderr, "missing encoder")
		os.Exit(2)
	}
	allowed := make(map[string]struct{})
	for _, token := range strings.Split(os.Getenv("FFMPEG_PROFILE_AVAILABLE"), ",") {
		normalized := strings.ToLower(strings.TrimSpace(token))
		if normalized == "" {
			continue
		}
		allowed[normalized] = struct{}{}
	}
	if _, ok := allowed[encoder]; ok {
		os.Exit(0)
	}
	_, _ = fmt.Fprintf(os.Stderr, "encoder not available: %s\n", encoder)
	os.Exit(2)
}
