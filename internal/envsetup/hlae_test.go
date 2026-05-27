package envsetup

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cs2-highlight-tool-v2/internal/release"
)

func createZipBytesForTest(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
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
	return buf.Bytes()
}

func TestWriteHLAEFfmpegIni(t *testing.T) {
	hlaeDir := t.TempDir()
	ffmpegExe := `C:\tools\ffmpeg\bin\ffmpeg.exe`

	if err := writeHLAEFfmpegIni(hlaeDir, ffmpegExe); err != nil {
		t.Fatalf("writeHLAEFfmpegIni failed: %v", err)
	}

	iniPath := filepath.Join(hlaeDir, "ffmpeg", "ffmpeg.ini")
	data, err := os.ReadFile(iniPath)
	if err != nil {
		t.Fatalf("failed to read ini: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "[Ffmpeg]") {
		t.Fatalf("ini missing [Ffmpeg] section, got: %q", content)
	}
	if !strings.Contains(content, "Path="+ffmpegExe) {
		t.Fatalf("ini missing correct Path, got: %q", content)
	}
}

func TestWriteHLAEFfmpegIni_CreatesDir(t *testing.T) {
	root := t.TempDir()
	hlaeDir := filepath.Join(root, "hlae")

	ffmpegExe := `/some/path/ffmpeg.exe`
	if err := writeHLAEFfmpegIni(hlaeDir, ffmpegExe); err != nil {
		t.Fatalf("writeHLAEFfmpegIni failed: %v", err)
	}

	iniPath := filepath.Join(hlaeDir, "ffmpeg", "ffmpeg.ini")
	if _, err := os.Stat(iniPath); err != nil {
		t.Fatalf("ini file not created: %v", err)
	}
}

func TestValidateHLAEFfmpegIni_Valid(t *testing.T) {
	hlaeDir := t.TempDir()
	ffmpegExe := filepath.Join("C:", "tools", "ffmpeg", "bin", "ffmpeg.exe")

	if err := writeHLAEFfmpegIni(hlaeDir, ffmpegExe); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	if err := validateHLAEFfmpegIni(hlaeDir, ffmpegExe); err != nil {
		t.Fatalf("validate should pass: %v", err)
	}
}

func TestValidateHLAEFfmpegIni_Missing(t *testing.T) {
	hlaeDir := t.TempDir()

	err := validateHLAEFfmpegIni(hlaeDir, "whatever")
	if err == nil {
		t.Fatal("validate should fail when ini is missing")
	}
	if !strings.Contains(err.Error(), "ffmpeg.ini") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateHLAEFfmpegIni_WrongPath(t *testing.T) {
	hlaeDir := t.TempDir()
	if err := writeHLAEFfmpegIni(hlaeDir, "/old/path/ffmpeg.exe"); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	err := validateHLAEFfmpegIni(hlaeDir, "/new/path/ffmpeg.exe")
	if err == nil {
		t.Fatal("validate should fail when path does not match")
	}
	if !strings.Contains(err.Error(), "不匹配") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateHLAEFfmpegIni_EmptyFile(t *testing.T) {
	hlaeDir := t.TempDir()
	iniDir := filepath.Join(hlaeDir, "ffmpeg")
	if err := os.MkdirAll(iniDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(iniDir, "ffmpeg.ini"), []byte("[Ffmpeg]\r\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := validateHLAEFfmpegIni(hlaeDir, "/some/path")
	if err == nil {
		t.Fatal("validate should fail when Path line is missing")
	}
	if !strings.Contains(err.Error(), "未找到") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTryWriteHLAEFfmpegIni_Integration(t *testing.T) {
	exeDir := t.TempDir()
	hlaeDir := filepath.Join(exeDir, "hlae")
	if err := os.MkdirAll(hlaeDir, 0755); err != nil {
		t.Fatal(err)
	}

	svc := New(exeDir, "1.0.0")
	svc.Startup(nil)

	ffmpegExe := filepath.Join(exeDir, "ffmpeg", "bin", "ffmpeg.exe")

	// tryWriteHLAEFfmpegIni should write the ini
	svc.tryWriteHLAEFfmpegIni(ffmpegExe)

	if err := validateHLAEFfmpegIni(hlaeDir, ffmpegExe); err != nil {
		t.Fatalf("ini should be valid after tryWrite: %v", err)
	}

	// calling again should be a no-op (already valid)
	svc.tryWriteHLAEFfmpegIni(ffmpegExe)

	if err := validateHLAEFfmpegIni(hlaeDir, ffmpegExe); err != nil {
		t.Fatalf("ini should still be valid: %v", err)
	}
}

func writeLocalHLAEForTest(t *testing.T, hlaeDir, changelog string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(hlaeDir, "x64"), 0755); err != nil {
		t.Fatal(err)
	}
	hlaeExe := filepath.Join(hlaeDir, "HLAE.exe")
	if err := os.WriteFile(hlaeExe, []byte("exe"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hlaeDir, "x64", "AfxHookSource2.dll"), []byte("hook"), 0644); err != nil {
		t.Fatal(err)
	}
	if changelog != "" {
		if err := os.WriteFile(filepath.Join(hlaeDir, "changelog.xml"), []byte(changelog), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return hlaeExe
}

func hlaeStepForTest(t *testing.T, state StartupState) ComponentStatus {
	t.Helper()
	for _, step := range state.Steps {
		if step.ID == componentHLAE {
			return step
		}
	}
	t.Fatal("hlae step not found")
	return ComponentStatus{}
}

func TestResolveInstalledHLAEVersion_FromChangelog(t *testing.T) {
	hlaeDir := t.TempDir()
	hlaeExe := writeLocalHLAEForTest(t, hlaeDir, `<changelog><version> v2.189.7 </version></changelog>`)

	version, err := resolveInstalledHLAEVersion(hlaeExe)
	if err != nil {
		t.Fatalf("resolveInstalledHLAEVersion error: %v", err)
	}
	if version != "2.189.7" {
		t.Fatalf("version = %q, want %q", version, "2.189.7")
	}
}

func TestResolveInstalledHLAEVersion_MissingChangelog(t *testing.T) {
	hlaeDir := t.TempDir()
	hlaeExe := writeLocalHLAEForTest(t, hlaeDir, "")

	_, err := resolveInstalledHLAEVersion(hlaeExe)
	if err == nil || !strings.Contains(err.Error(), "changelog.xml") {
		t.Fatalf("error = %v, want changelog error", err)
	}
}

func TestResolveInstalledHLAEVersion_InvalidXML(t *testing.T) {
	hlaeDir := t.TempDir()
	hlaeExe := writeLocalHLAEForTest(t, hlaeDir, `<changelog><version`)

	_, err := resolveInstalledHLAEVersion(hlaeExe)
	if err == nil || !strings.Contains(err.Error(), "解析 changelog.xml") {
		t.Fatalf("error = %v, want xml parse error", err)
	}
}

func TestResolveInstalledHLAEVersion_NoVersion(t *testing.T) {
	hlaeDir := t.TempDir()
	hlaeExe := writeLocalHLAEForTest(t, hlaeDir, `<changelog><entry>ok</entry></changelog>`)

	_, err := resolveInstalledHLAEVersion(hlaeExe)
	if err == nil || !strings.Contains(err.Error(), "未找到 version") {
		t.Fatalf("error = %v, want missing version error", err)
	}
}

func TestInstallHLAEFromArchive_UsesChangelogVersion(t *testing.T) {
	exeDir := t.TempDir()
	svc := New(exeDir, "1.0.0")
	svc.Startup(nil)

	zipPath := createZipForTest(t, map[string][]byte{
		"release/hlae.exe":               []byte("hlae"),
		"release/x64/AfxHookSource2.dll": []byte("hook"),
		"release/changelog.xml":          []byte("<changelog><version>v2.189.7</version></changelog>"),
	})
	if err := svc.installHLAEFromArchive(zipPath); err != nil {
		t.Fatalf("installHLAEFromArchive error: %v", err)
	}

	step := hlaeStepForTest(t, svc.GetStartupState())
	if step.LocalVersion != "2.189.7" {
		t.Fatalf("local version = %q, want %q", step.LocalVersion, "2.189.7")
	}

	data, err := os.ReadFile(filepath.Join(exeDir, "hlae", "version.json"))
	if err != nil {
		t.Fatalf("read version.json: %v", err)
	}
	var saved componentVersionFile
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshal version.json: %v", err)
	}
	if saved.Version != "2.189.7" {
		t.Fatalf("saved version = %q, want %q", saved.Version, "2.189.7")
	}
}

func TestEnsureHLAE_IgnoresLegacyConfigVersionField(t *testing.T) {
	exeDir := t.TempDir()
	hlaeDir := filepath.Join(exeDir, "hlae")
	_ = writeLocalHLAEForTest(t, hlaeDir, `<changelog><version>1.0.0</version></changelog>`)

	cfgPayload := `{
		"download_source":"custom",
		"hlae_exe":"` + filepath.ToSlash(filepath.Join(hlaeDir, "HLAE.exe")) + `",
		"hlae_version":"9.9.9"
	}`
	if err := os.WriteFile(filepath.Join(exeDir, "config.json"), []byte(cfgPayload), 0644); err != nil {
		t.Fatal(err)
	}

	archiveData := createZipBytesForTest(t, map[string][]byte{
		"release/hlae.exe":               []byte("hlae"),
		"release/x64/AfxHookSource2.dll": []byte("hook"),
		"release/changelog.xml":          []byte("<changelog><version>2.0.0</version></changelog>"),
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hlae.zip" {
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
			release.ComponentKeyHLAE: {
				release.SourceGitHub: {
					{
						Repo:   "advancedfx/advancedfx",
						Source: release.SourceGitHub,
						OK:     true,
						Info: &release.Info{
							TagName: "v2.0.0",
							Repo:    "advancedfx/advancedfx",
							Source:  release.SourceGitHub,
							Assets: []release.Asset{
								{Name: "hlae_2_0_0.zip", DownloadURL: server.URL + "/hlae.zip", GitHubURL: server.URL + "/hlae.zip"},
							},
						},
					},
				},
			},
		},
	}
	svc.mu.Unlock()

	if err := svc.ensureHLAE(DownloadSourceCustom); err != nil {
		t.Fatalf("ensureHLAE error: %v", err)
	}

	step := hlaeStepForTest(t, svc.GetStartupState())
	if step.LocalVersion != "2.0.0" {
		t.Fatalf("local version = %q, want %q", step.LocalVersion, "2.0.0")
	}
}

func TestEnsureHLAEWithFallback_FailsWhenChangelogInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}))
	defer server.Close()
	t.Setenv("CS2_RELEASE_API_URL", server.URL)

	exeDir := t.TempDir()
	hlaeDir := filepath.Join(exeDir, "hlae")
	_ = writeLocalHLAEForTest(t, hlaeDir, `<changelog><version>broken`)

	svc := New(exeDir, "1.0.0")
	svc.Startup(nil)
	svc.runComponent(componentHLAE, svc.ensureHLAEWithFallback)

	step := hlaeStepForTest(t, svc.GetStartupState())
	if step.Status != statusFailed {
		t.Fatalf("step status = %q, want %q", step.Status, statusFailed)
	}
}
