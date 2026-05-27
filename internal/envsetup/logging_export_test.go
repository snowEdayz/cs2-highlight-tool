package envsetup

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSanitizeURLForExport(t *testing.T) {
	raw := "https://example.com/api/release?version=1.2.3&token=secret&source=github&extra=drop"
	got := sanitizeURLForExport(raw)
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("sanitizeURLForExport parse error: %v", err)
	}
	query := parsed.Query()
	if query.Get("version") != "1.2.3" {
		t.Fatalf("version query mismatch: %q", query.Get("version"))
	}
	if query.Get("source") != "github" {
		t.Fatalf("source query mismatch: %q", query.Get("source"))
	}
	if query.Get("token") != "***" {
		t.Fatalf("token should be masked, got %q", query.Get("token"))
	}
	if query.Get("extra") != "" {
		t.Fatalf("non-whitelisted query key should be removed, got %q", query.Get("extra"))
	}
}

func TestSanitizePathForExport_ReplacesHomePrefix(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		t.Skip("home directory unavailable")
	}
	path := filepath.Join(home, "Desktop", "test", "startup.log")
	got := sanitizePathForExport(path)
	if !strings.HasPrefix(got, "~") {
		t.Fatalf("expected path prefixed with ~, got %q", got)
	}
	if strings.Contains(got, home) {
		t.Fatalf("home path should be hidden, got %q", got)
	}
}

func TestSanitizeErrorForExport_MasksCredentials(t *testing.T) {
	raw := "download failed auth=abc token=xyz Authorization: Bearer test-token"
	got := sanitizeErrorForExport(raw)
	for _, token := range []string{"abc", "xyz", "test-token"} {
		if strings.Contains(got, token) {
			t.Fatalf("sensitive token leaked in error: %q", got)
		}
	}
	if !strings.Contains(got, "Authorization:***") {
		t.Fatalf("expected authorization masking, got %q", got)
	}
}

func TestBuildStartupLogReport_ContainsTimelineAndSummaries(t *testing.T) {
	home, _ := os.UserHomeDir()
	now := time.Now().UTC()
	state := StartupState{
		Mode:         "startup",
		Phase:        phaseRunningTasks,
		Running:      true,
		CanEnterMain: false,
		SourceStep: SourceStepState{
			Status:      statusFailed,
			Source:      "github",
			CountryCode: "US",
			Message:     "ip lookup timeout",
			Error:       "auth=token123",
		},
		SelfUpdate: SelfUpdateState{
			Status:    statusFailed,
			Available: true,
			Current:   "1.0.0",
			Latest:    "1.1.0",
			URL:       "https://example.com/release?token=abc&version=1.1.0",
			AssetURL:  "https://example.com/download/app.exe?auth=abcd",
			Error:     "Authorization: Bearer secret",
		},
		Steps: []ComponentStatus{
			{
				ID:            componentPlugin,
				Name:          "插件 DLL",
				Status:        statusFailed,
				LocalVersion:  "1.0.0",
				RemoteVersion: "1.2.0",
				Path:          filepath.Join(home, "Desktop", "plugin", "cs2-highlight-plugin.dll"),
				ManualURL:     "https://example.com/plugin?token=abc&version=1.2.0",
				Error:         "download failed token=abc",
			},
		},
	}
	logs := []LogMessage{
		{
			Time:      now.Add(10 * time.Millisecond).Format(time.RFC3339Nano),
			Level:     "info",
			Message:   "download started",
			Component: componentPlugin,
			Stage:     "download",
			Action:    "download_asset",
			Source:    "github",
			Attempt:   1,
			Meta: map[string]string{
				"url":  "https://example.com/plugin.zip?token=abc&version=1.2.0",
				"path": filepath.Join(home, "Downloads", "plugin.zip"),
			},
		},
		{
			Time:      now.Add(20 * time.Millisecond).Format(time.RFC3339Nano),
			Level:     "error",
			Message:   "download failed",
			Component: componentPlugin,
			Stage:     "download",
			Action:    "download_asset",
			Source:    "github",
			Attempt:   1,
			ElapsedMS: 42,
			Error:     "token=abc",
		},
	}

	report := buildStartupLogReport(now, state, logs)
	for _, section := range []string{"[Timeline]", "[Failure Summary]", "[Performance Summary]"} {
		if !strings.Contains(report, section) {
			t.Fatalf("missing section %s", section)
		}
	}
	if strings.Contains(report, "token123") || strings.Contains(report, "token=abc") || strings.Contains(report, "secret") {
		t.Fatalf("report leaked sensitive fields:\n%s", report)
	}
	if home != "" && strings.Contains(report, home) {
		t.Fatalf("report leaked home path: %s", home)
	}
	if !strings.Contains(report, "component=plugin") {
		t.Fatalf("timeline should include component field:\n%s", report)
	}
	if !strings.Contains(report, "plugin: total=") {
		t.Fatalf("performance summary should include plugin stats:\n%s", report)
	}
}

func TestEmitLogHelpers_WriteStructuredFields(t *testing.T) {
	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)

	start := svc.logStepStart(componentPlugin, "download", "download_asset", "github", 2, map[string]string{
		"url": "https://example.com/plugin.zip",
	})
	svc.logStepDone(componentPlugin, "download", "download_asset", "github", 2, start, map[string]string{
		"target": "plugin.zip",
	})
	failStart := svc.logStepStart(componentPlugin, "extract", "unzip", "github", 3, nil)
	svc.logStepFail(componentPlugin, "extract", "unzip", "github", 3, failStart, fmt.Errorf("zip broken"), nil)

	logs := svc.logsSnapshot()
	if len(logs) < 4 {
		t.Fatalf("expected at least 4 logs, got %d", len(logs))
	}
	if !hasStructuredLog(logs, componentPlugin, "download", "download_asset") {
		t.Fatal("missing structured download log")
	}
	if !hasStructuredLog(logs, componentPlugin, "extract", "unzip") {
		t.Fatal("missing structured extract log")
	}
	last := logs[len(logs)-1]
	if strings.TrimSpace(last.Error) == "" {
		t.Fatalf("expected failure log with error, got %+v", last)
	}
}

func TestEmitLogWithFields_RingBufferLimit(t *testing.T) {
	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)

	total := startupLogBufferLimit + 25
	for i := 0; i < total; i++ {
		svc.emitLog("info", fmt.Sprintf("line-%d", i))
	}
	logs := svc.logsSnapshot()
	if len(logs) != startupLogBufferLimit {
		t.Fatalf("log length = %d, want %d", len(logs), startupLogBufferLimit)
	}
	wantFirst := fmt.Sprintf("line-%d", total-startupLogBufferLimit)
	if logs[0].Message != wantFirst {
		t.Fatalf("oldest log = %q, want %q", logs[0].Message, wantFirst)
	}
}
