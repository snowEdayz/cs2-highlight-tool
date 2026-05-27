package envsetup

import (
	"archive/zip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cs2-highlight-tool-v2/internal/release"
)

func createZipForTest(t *testing.T, entries map[string][]byte) string {
	t.Helper()
	zipPath := filepath.Join(t.TempDir(), "payload.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, content := range entries {
		part, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return zipPath
}

func TestInstallPluginFromFile_RejectsNonZip(t *testing.T) {
	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)

	input := filepath.Join(t.TempDir(), pluginDLLFileName)
	if err := os.WriteFile(input, []byte("dll"), 0644); err != nil {
		t.Fatal(err)
	}
	err := svc.installPluginFromFile(input)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "zip") {
		t.Fatalf("error = %v, want zip validation error", err)
	}
}

func TestInstallPluginFromFile_RejectsZipWithoutDLL(t *testing.T) {
	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)

	zipPath := createZipForTest(t, map[string][]byte{
		"readme.txt": []byte("text"),
	})
	err := svc.installPluginFromFile(zipPath)
	if err == nil || !strings.Contains(err.Error(), pluginDLLFileName) {
		t.Fatalf("error = %v, want missing %s error", err, pluginDLLFileName)
	}
}

func TestInstallPluginFromFile_RejectsZipWithoutServerDLL(t *testing.T) {
	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)

	zipPath := createZipForTest(t, map[string][]byte{
		"plugin.dll":    []byte("dll"),
		"changelog.xml": []byte("<changelog><version>v1.2.3</version></changelog>"),
	})
	err := svc.installPluginFromFile(zipPath)
	if err == nil || !strings.Contains(err.Error(), pluginDLLFileName) {
		t.Fatalf("error = %v, want missing %s error", err, pluginDLLFileName)
	}
}

func TestInstallPluginFromFile_RejectsZipWithoutChangelog(t *testing.T) {
	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)

	zipPath := createZipForTest(t, map[string][]byte{
		pluginDLLFileName: []byte("dll"),
	})
	err := svc.installPluginFromFile(zipPath)
	if err == nil || !strings.Contains(err.Error(), "changelog.xml") {
		t.Fatalf("error = %v, want changelog validation error", err)
	}
}

func pluginStepForTest(t *testing.T, state StartupState) ComponentStatus {
	t.Helper()
	for _, step := range state.Steps {
		if step.ID == componentPlugin {
			return step
		}
	}
	t.Fatal("plugin step not found")
	return ComponentStatus{}
}

func writeLocalPluginForTest(t *testing.T, pluginDir, changelog string) string {
	t.Helper()
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	pluginDLL := filepath.Join(pluginDir, pluginDLLFileName)
	if err := os.WriteFile(pluginDLL, []byte("dll"), 0644); err != nil {
		t.Fatal(err)
	}
	if changelog != "" {
		if err := os.WriteFile(filepath.Join(pluginDir, "changelog.xml"), []byte(changelog), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return pluginDLL
}

func TestInstallPluginFromFile_UsesChangelogVersion(t *testing.T) {
	exeDir := t.TempDir()
	svc := New(exeDir, "1.0.0")
	svc.Startup(nil)

	zipPath := createZipForTest(t, map[string][]byte{
		pluginDLLFileName: []byte("dll"),
		"changelog.xml":   []byte("<changelog><version>v1.2.3</version></changelog>"),
	})
	if err := svc.installPluginFromFile(zipPath); err != nil {
		t.Fatalf("installPluginFromFile error: %v", err)
	}

	expected := filepath.Join(exeDir, "plugin", pluginDLLFileName)
	cfg := svc.currentConfig()
	if cfg.PluginDLL != expected {
		t.Fatalf("plugin path = %q, want %q", cfg.PluginDLL, expected)
	}
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("installed dll missing: %v", err)
	}
	step := pluginStepForTest(t, svc.GetStartupState())
	if step.LocalVersion != "1.2.3" {
		t.Fatalf("local version = %q, want %q", step.LocalVersion, "1.2.3")
	}
	data, err := os.ReadFile(filepath.Join(exeDir, "plugin", "version.json"))
	if err != nil {
		t.Fatalf("read version.json: %v", err)
	}
	var saved componentVersionFile
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshal version.json: %v", err)
	}
	if saved.Version != "1.2.3" {
		t.Fatalf("saved version = %q, want %q", saved.Version, "1.2.3")
	}
	logs := svc.logsSnapshot()
	for _, item := range []struct {
		stage  string
		action string
	}{
		{stage: "extract", action: "unzip"},
		{stage: "validate", action: "verify_archive"},
		{stage: "persist_config", action: "write"},
		{stage: "ready", action: "component_ready"},
	} {
		if !hasStructuredLog(logs, componentPlugin, item.stage, item.action) {
			t.Fatalf("missing plugin log stage=%s action=%s", item.stage, item.action)
		}
	}
}

func TestResolveInstalledPluginVersion_FromChangelog(t *testing.T) {
	pluginDir := t.TempDir()
	pluginDLL := writeLocalPluginForTest(t, pluginDir, `<changelog><version> v2.0.1 </version></changelog>`)

	version, err := resolveInstalledPluginVersion(pluginDLL)
	if err != nil {
		t.Fatalf("resolveInstalledPluginVersion error: %v", err)
	}
	if version != "2.0.1" {
		t.Fatalf("version = %q, want %q", version, "2.0.1")
	}
}

func TestResolveInstalledPluginVersion_MissingChangelog(t *testing.T) {
	pluginDir := t.TempDir()
	pluginDLL := writeLocalPluginForTest(t, pluginDir, "")

	_, err := resolveInstalledPluginVersion(pluginDLL)
	if err == nil || !strings.Contains(err.Error(), "changelog.xml") {
		t.Fatalf("error = %v, want changelog error", err)
	}
}

func TestResolveInstalledPluginVersion_InvalidXML(t *testing.T) {
	pluginDir := t.TempDir()
	pluginDLL := writeLocalPluginForTest(t, pluginDir, `<changelog><version`)

	_, err := resolveInstalledPluginVersion(pluginDLL)
	if err == nil || !strings.Contains(err.Error(), "解析 changelog.xml") {
		t.Fatalf("error = %v, want xml parse error", err)
	}
}

func TestResolveInstalledPluginVersion_NoVersion(t *testing.T) {
	pluginDir := t.TempDir()
	pluginDLL := writeLocalPluginForTest(t, pluginDir, `<changelog><entry>ok</entry></changelog>`)

	_, err := resolveInstalledPluginVersion(pluginDLL)
	if err == nil || !strings.Contains(err.Error(), "未找到 version") {
		t.Fatalf("error = %v, want missing version error", err)
	}
}

func TestEnsurePlugin_IgnoresLegacyConfigVersionField(t *testing.T) {
	exeDir := t.TempDir()
	pluginDir := filepath.Join(exeDir, "plugin")
	_ = writeLocalPluginForTest(t, pluginDir, `<changelog><version>1.0.0</version></changelog>`)

	cfgPayload := `{
		"download_source":"custom",
		"plugin_dll":"` + filepath.ToSlash(filepath.Join(pluginDir, pluginDLLFileName)) + `",
		"plugin_version":"9.9.9"
	}`
	if err := os.WriteFile(filepath.Join(exeDir, "config.json"), []byte(cfgPayload), 0644); err != nil {
		t.Fatal(err)
	}

	archiveData := createZipBytesForTest(t, map[string][]byte{
		pluginDLLFileName: []byte("dll"),
		"changelog.xml":   []byte("<changelog><version>2.0.0</version></changelog>"),
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/plugin.zip" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archiveData)
	}))
	defer server.Close()

	svc := New(exeDir, "1.0.0")
	svc.Startup(nil)
	svc.mu.Lock()
	svc.releaseSnapshot = &release.UnifiedLatest{
		Components: map[string]map[string][]release.ComponentCandidate{
			release.ComponentKeyPlugin: {
				release.SourceGitHub: {
					{
						Repo:   "hkslover/cs2-server-plugin",
						Source: release.SourceGitHub,
						OK:     true,
						Info: &release.Info{
							TagName: "v2.0.0",
							Repo:    "hkslover/cs2-server-plugin",
							Source:  release.SourceGitHub,
							Assets: []release.Asset{
								{Name: "plugin_2_0_0.zip", DownloadURL: server.URL + "/plugin.zip", GitHubURL: server.URL + "/plugin.zip"},
							},
						},
					},
				},
			},
		},
	}
	svc.mu.Unlock()

	if err := svc.ensurePlugin(DownloadSourceCustom); err != nil {
		t.Fatalf("ensurePlugin error: %v", err)
	}

	step := pluginStepForTest(t, svc.GetStartupState())
	if step.LocalVersion != "2.0.0" {
		t.Fatalf("local version = %q, want %q", step.LocalVersion, "2.0.0")
	}
}

func TestEnsurePluginWithFallback_FailsWhenChangelogInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}))
	defer server.Close()
	t.Setenv("CS2_RELEASE_API_URL", server.URL)

	exeDir := t.TempDir()
	pluginDir := filepath.Join(exeDir, "plugin")
	_ = writeLocalPluginForTest(t, pluginDir, `<changelog><version>broken`)

	svc := New(exeDir, "1.0.0")
	svc.Startup(nil)
	svc.runComponent(componentPlugin, svc.ensurePluginWithFallback)

	step := pluginStepForTest(t, svc.GetStartupState())
	if step.Status != statusFailed {
		t.Fatalf("step status = %q, want %q", step.Status, statusFailed)
	}
}
