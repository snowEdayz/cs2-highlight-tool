package app

import (
	"path/filepath"
	"strings"
	"testing"

	"cs2-highlight-tool-v2/internal/envsetup"
	"cs2-highlight-tool-v2/internal/producews"
)


// TestAppendAppSubdir 验证自动追加 cs2HighLightTool 的幂等逻辑。
// 使用 filepath.Join 构造期望值，确保跨平台路径分隔符一致。
func TestAppendAppSubdir(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{
			filepath.Join("/home", "user", "data"),
			filepath.Join("/home", "user", "data", "cs2HighLightTool"),
		},
		{
			filepath.Join("/home", "user", "data", "cs2HighLightTool"),
			filepath.Join("/home", "user", "data", "cs2HighLightTool"),
		},
	}
	for _, c := range cases {
		got := appendAppSubdir(c.input)
		if got != c.want {
			t.Errorf("appendAppSubdir(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// TestGetWorkspaceState_Uninitialized 验证未注册 dataDir 时返回 Initialized=false。
func TestGetWorkspaceState_Uninitialized(t *testing.T) {
	a := &App{
		exeDir:   t.TempDir(),
		produceW: producews.NewDefault(nil),
	}
	ws := a.GetWorkspaceState()
	if ws.Initialized {
		t.Fatalf("expected Initialized=false, got %#v", ws)
	}
	if ws.DataDir != "" {
		t.Fatalf("expected empty DataDir, got %q", ws.DataDir)
	}
}

// TestGetWorkspaceState_Initialized 验证设置 service+dataDir 后 Initialized=true。
func TestGetWorkspaceState_Initialized(t *testing.T) {
	exeDir := t.TempDir()
	dataDir := t.TempDir()
	a := &App{
		exeDir:   exeDir,
		dataDir:  dataDir,
		produceW: producews.NewDefault(nil),
	}
	a.service = envsetup.NewWithDataDir(exeDir, dataDir, "test")

	ws := a.GetWorkspaceState()
	if !ws.Initialized {
		t.Fatalf("expected Initialized=true, got %#v", ws)
	}
	if ws.DataDir != dataDir {
		t.Fatalf("DataDir = %q, want %q", ws.DataDir, dataDir)
	}
}

// TestGetStartupState_NoServiceReturnsWorkspaceInit 验证未初始化时 GetStartupState 返回 mode=workspace_init。
func TestGetStartupState_NoServiceReturnsWorkspaceInit(t *testing.T) {
	a := &App{
		exeDir:   t.TempDir(),
		produceW: producews.NewDefault(nil),
	}
	state := a.GetStartupState()
	if state.Mode != envsetup.ModeWorkspaceInit {
		t.Fatalf("Mode = %q, want %q", state.Mode, envsetup.ModeWorkspaceInit)
	}
}

// TestRunStartupChecks_NoServiceReturnsWorkspaceInit 验证未初始化时 RunStartupChecks 返回 mode=workspace_init。
func TestRunStartupChecks_NoServiceReturnsWorkspaceInit(t *testing.T) {
	a := &App{
		exeDir:   t.TempDir(),
		produceW: producews.NewDefault(nil),
	}
	state := a.RunStartupChecks()
	if state.Mode != envsetup.ModeWorkspaceInit {
		t.Fatalf("Mode = %q, want %q", state.Mode, envsetup.ModeWorkspaceInit)
	}
}

// TestEnterMainApp_NoServiceReturnsError 验证未初始化时 EnterMainApp 返回明确错误。
func TestEnterMainApp_NoServiceReturnsError(t *testing.T) {
	a := &App{
		exeDir:   t.TempDir(),
		produceW: producews.NewDefault(nil),
	}
	err := a.EnterMainApp()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "工作目录") {
		t.Fatalf("error = %q, want contain '工作目录'", got)
	}
}

// TestApplySelfUpdate_NoServiceReturnsWorkspaceInit 验证未初始化时 ApplySelfUpdate 返回 mode=workspace_init。
func TestApplySelfUpdate_NoServiceReturnsWorkspaceInit(t *testing.T) {
	a := &App{
		exeDir:   t.TempDir(),
		produceW: producews.NewDefault(nil),
	}
	state := a.ApplySelfUpdate()
	if state.Mode != envsetup.ModeWorkspaceInit {
		t.Fatalf("Mode = %q, want %q", state.Mode, envsetup.ModeWorkspaceInit)
	}
}

// TestExportStartupLogs_NoServiceReturnsError 验证未初始化时 ExportStartupLogs 返回错误。
func TestExportStartupLogs_NoServiceReturnsError(t *testing.T) {
	a := &App{
		exeDir:   t.TempDir(),
		produceW: producews.NewDefault(nil),
	}
	_, err := a.ExportStartupLogs()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// TestValidateWorkspaceDir_Wrapper 验证包装方法正确传递校验结果。
func TestValidateWorkspaceDir_Wrapper(t *testing.T) {
	a := &App{
		exeDir:   t.TempDir(),
		produceW: producews.NewDefault(nil),
	}

	// 失败用例：含空格
	ok, msg := a.ValidateWorkspaceDir(`C:\Program Files\App`)
	if ok {
		t.Fatalf("expected validation to fail for path with space, ok=%v msg=%q", ok, msg)
	}
	if !strings.Contains(msg, "空格") {
		t.Fatalf("error msg = %q, want contain '空格'", msg)
	}

	// 失败用例：磁盘根
	ok, msg = a.ValidateWorkspaceDir(`/`)
	if ok {
		t.Fatalf("expected validation to fail for disk root, ok=%v msg=%q", ok, msg)
	}
}

// TestApp_DataPath_RespectsDataDir 验证 dataPath 函数优先使用 dataDir。
func TestApp_DataPath_RespectsDataDir(t *testing.T) {
	exeDir := t.TempDir()
	dataDir := t.TempDir()
	a := &App{
		exeDir:  exeDir,
		dataDir: dataDir,
	}
	got := a.dataPath("config.json")
	want := filepath.Join(dataDir, "config.json")
	if got != want {
		t.Fatalf("dataPath = %q, want %q", got, want)
	}
}

// TestApp_DataPath_FallsBackToExeDir 验证 dataDir 为空时回退到 exeDir。
func TestApp_DataPath_FallsBackToExeDir(t *testing.T) {
	exeDir := t.TempDir()
	a := &App{exeDir: exeDir}
	got := a.dataPath("foo")
	want := filepath.Join(exeDir, "foo")
	if got != want {
		t.Fatalf("dataPath = %q, want %q", got, want)
	}
}
