package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cs2-highlight-tool-v2/internal/config"
	"cs2-highlight-tool-v2/internal/producews"
)

// TestGetPendingChangelog_FreshInstall 模拟全新安装：dataDir 内无 config.json 时，
// seedFirstInstallChangelog 应该把 LastChangelogVersion 预设为当前版本，
// GetPendingChangelog 之后看到 LastChangelogVersion == 当前版本 -> ShouldShow=false。
func TestGetPendingChangelog_FreshInstall(t *testing.T) {
	dir := t.TempDir()
	a := &App{
		exeDir:   t.TempDir(),
		dataDir:  dir,
		version:  "2.0.2",
		produceW: producews.NewDefault(nil),
	}
	a.seedFirstInstallChangelog()

	pending, err := a.GetPendingChangelog()
	if err != nil {
		t.Fatalf("GetPendingChangelog failed: %v", err)
	}
	if pending.ShouldShow {
		t.Fatalf("fresh install must not show changelog, got %+v", pending)
	}

	cfg, err := config.LoadOrCreate(a.configPath(), a.dataRoot())
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	if cfg.LastChangelogVersion != "2.0.2" {
		t.Fatalf("LastChangelogVersion = %q, want 2.0.2", cfg.LastChangelogVersion)
	}
}

// TestGetPendingChangelog_ExistingUserMissingKey 模拟老用户首次升级到带本功能的版本：
// config.json 已经存在，但缺 last_changelog_version key -> 应该展示 changelog。
func TestGetPendingChangelog_ExistingUserMissingKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// 模拟老 config.json：完全不写 last_changelog_version 字段
	if err := os.WriteFile(path, []byte(`{"cs2_dir":""}`), 0o644); err != nil {
		t.Fatalf("seed legacy config failed: %v", err)
	}
	a := &App{
		exeDir:   t.TempDir(),
		dataDir:  dir,
		version:  "2.0.2",
		produceW: producews.NewDefault(nil),
	}
	// seedFirstInstallChangelog 是 noop（config.json 已存在）
	a.seedFirstInstallChangelog()

	pending, err := a.GetPendingChangelog()
	if err != nil {
		t.Fatalf("GetPendingChangelog failed: %v", err)
	}
	if !pending.ShouldShow {
		t.Fatalf("existing user with missing key must see changelog, got %+v", pending)
	}
	if pending.Version != "2.0.2" {
		t.Errorf("Version = %q, want 2.0.2", pending.Version)
	}
	if pending.BodyZh == "" || pending.BodyEn == "" {
		t.Errorf("expected both bodies populated, got zh=%q en=%q", pending.BodyZh, pending.BodyEn)
	}
}

// TestGetPendingChangelog_SameVersionNoShow 模拟同版本下重复打开：
// LastChangelogVersion 与 current 相同 -> 不展示。
func TestGetPendingChangelog_SameVersionNoShow(t *testing.T) {
	dir := t.TempDir()
	a := &App{
		exeDir:   t.TempDir(),
		dataDir:  dir,
		version:  "2.0.2",
		produceW: producews.NewDefault(nil),
	}
	cfg, err := config.LoadOrCreate(a.configPath(), a.dataRoot())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	cfg.LastChangelogVersion = "2.0.2"
	if err := config.Save(a.configPath(), cfg); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	pending, err := a.GetPendingChangelog()
	if err != nil {
		t.Fatalf("GetPendingChangelog failed: %v", err)
	}
	if pending.ShouldShow {
		t.Fatalf("same-version must not show, got %+v", pending)
	}
}

// TestGetPendingChangelog_MissingNotesFileSilent 模拟开发者忘记写 changelog 文件：
// 当前版本无 embed 文件 -> 静默不展示，且不写回 LastChangelogVersion。
func TestGetPendingChangelog_MissingNotesFileSilent(t *testing.T) {
	dir := t.TempDir()
	a := &App{
		exeDir:   t.TempDir(),
		dataDir:  dir,
		version:  "9.9.9", // notes/v9.9.9.md 必然不存在
		produceW: producews.NewDefault(nil),
	}
	pending, err := a.GetPendingChangelog()
	if err != nil {
		t.Fatalf("GetPendingChangelog failed: %v", err)
	}
	if pending.ShouldShow {
		t.Fatalf("missing notes file must not show, got %+v", pending)
	}
	cfg, err := config.LoadOrCreate(a.configPath(), a.dataRoot())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.LastChangelogVersion != "" {
		t.Fatalf("missing-file path must not write back LastChangelogVersion, got %q", cfg.LastChangelogVersion)
	}
}

// TestAckChangelog_WritesVersion 验证 ack 把传入版本写入 config.json。
func TestAckChangelog_WritesVersion(t *testing.T) {
	dir := t.TempDir()
	a := &App{
		exeDir:   t.TempDir(),
		dataDir:  dir,
		version:  "2.0.2",
		produceW: producews.NewDefault(nil),
	}
	if err := a.AckChangelog("2.0.2"); err != nil {
		t.Fatalf("AckChangelog failed: %v", err)
	}
	cfg, err := config.LoadOrCreate(a.configPath(), a.dataRoot())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.LastChangelogVersion != "2.0.2" {
		t.Fatalf("LastChangelogVersion = %q, want 2.0.2", cfg.LastChangelogVersion)
	}
}

func TestAckChangelog_EmptyVersionRejected(t *testing.T) {
	dir := t.TempDir()
	a := &App{
		exeDir:   t.TempDir(),
		dataDir:  dir,
		version:  "2.0.2",
		produceW: producews.NewDefault(nil),
	}
	err := a.AckChangelog("   ")
	if err == nil {
		t.Fatalf("expected error for empty version, got nil")
	}
	if !strings.Contains(err.Error(), "版本号") {
		t.Errorf("error message = %q, want contain '版本号'", err.Error())
	}
}

// TestGetPendingChangelog_FullCycle 端到端：老用户升级 -> 看到 changelog -> ack -> 下次不再看到。
func TestGetPendingChangelog_FullCycle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("seed legacy config failed: %v", err)
	}
	a := &App{
		exeDir:   t.TempDir(),
		dataDir:  dir,
		version:  "2.0.2",
		produceW: producews.NewDefault(nil),
	}

	pending, err := a.GetPendingChangelog()
	if err != nil || !pending.ShouldShow {
		t.Fatalf("first call must show, got pending=%+v err=%v", pending, err)
	}

	if err := a.AckChangelog(pending.Version); err != nil {
		t.Fatalf("ack failed: %v", err)
	}

	pending2, err := a.GetPendingChangelog()
	if err != nil {
		t.Fatalf("second call err: %v", err)
	}
	if pending2.ShouldShow {
		t.Fatalf("second call must NOT show after ack, got %+v", pending2)
	}
}
