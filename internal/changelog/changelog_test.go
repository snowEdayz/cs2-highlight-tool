package changelog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// TestGet_ReturnsBothSections 校验 splitBilingual 能正确切出 `## 中文` / `## English` 两段。
func TestGet_ReturnsBothSections(t *testing.T) {
	notes, ok := Get("2.0.2")
	if !ok {
		t.Fatalf("expected v2.0.2 changelog to be present")
	}
	if notes.Version != "2.0.2" {
		t.Errorf("expected version=2.0.2, got %q", notes.Version)
	}
	if notes.BodyZh == "" {
		t.Errorf("expected non-empty BodyZh")
	}
	if notes.BodyEn == "" {
		t.Errorf("expected non-empty BodyEn")
	}
	if !strings.Contains(notes.BodyZh, "### ") {
		t.Errorf("expected BodyZh to keep subheading markdown, got %q", notes.BodyZh)
	}
	if !strings.Contains(notes.BodyEn, "### ") {
		t.Errorf("expected BodyEn to keep subheading markdown, got %q", notes.BodyEn)
	}
}

func TestGet_MissingVersionReturnsFalse(t *testing.T) {
	_, ok := Get("9.9.9")
	if ok {
		t.Fatalf("expected missing version to return ok=false")
	}
}

func TestGet_EmptyVersionReturnsFalse(t *testing.T) {
	_, ok := Get("")
	if ok {
		t.Fatalf("expected empty version to return ok=false")
	}
}

func TestSplitBilingual_HandlesMissingEnglishSection(t *testing.T) {
	zh, en := splitBilingual("## 中文\n- 仅中文\n")
	if zh == "" {
		t.Errorf("expected non-empty zh, got empty")
	}
	if en != "" {
		t.Errorf("expected empty en when section missing, got %q", en)
	}
}

func TestSplitBilingual_HandlesMissingChineseSection(t *testing.T) {
	zh, en := splitBilingual("## English\n- english only\n")
	if zh != "" {
		t.Errorf("expected empty zh when section missing, got %q", zh)
	}
	if en == "" {
		t.Errorf("expected non-empty en")
	}
}

func TestSplitBilingual_OrderIndependent(t *testing.T) {
	zh, en := splitBilingual("## English\n- en first\n\n## 中文\n- zh second\n")
	if !strings.Contains(zh, "zh second") {
		t.Errorf("expected zh to capture content even when after English, got %q", zh)
	}
	if !strings.Contains(en, "en first") {
		t.Errorf("expected en to capture content, got %q", en)
	}
}

// TestEmbeddedNotesCoverCurrentVersion 是发版护栏：
// 读取仓库 wails.json 中声明的版本号，如果是纯三段数字（X.Y.Z），
// 必须能在 notes/ 下找到对应 md 文件；带 -dev/-rc 等非正式后缀时跳过。
func TestEmbeddedNotesCoverCurrentVersion(t *testing.T) {
	wailsPath := locateRepoFile(t, "wails.json")
	data, err := os.ReadFile(wailsPath)
	if err != nil {
		t.Fatalf("read wails.json failed: %v", err)
	}
	var parsed struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse wails.json failed: %v", err)
	}
	version := strings.TrimSpace(parsed.Version)
	if version == "" {
		t.Fatalf("wails.json version field is empty")
	}
	pureSemver := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if !pureSemver.MatchString(version) {
		t.Logf("skipping coverage check for non-pure version %q", version)
		t.Skip()
	}
	if _, ok := Get(version); !ok {
		t.Fatalf("missing internal/changelog/notes/v%s.md — every release tag with a pure semver must ship a changelog file", version)
	}
}

// locateRepoFile 从当前测试包向上查找仓库根的指定文件。
func locateRepoFile(t *testing.T, name string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile)
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate %s relative to %s", name, thisFile)
	return ""
}
