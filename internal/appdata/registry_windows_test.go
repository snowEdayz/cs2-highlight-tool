//go:build windows

package appdata

import (
	"testing"

	"golang.org/x/sys/windows/registry"
)

// TestRegistry_RoundTrip 验证写入/读取/删除 DataDir 注册表 value。
// 使用真实 HKCU 注册表，测试结束时清理。
func TestRegistry_RoundTrip(t *testing.T) {
	// 测试前先清理（防止上次测试残留）
	_ = DeleteDataDirFromRegistry()

	// 1. 未写入时应读取失败
	if _, err := ReadDataDirFromRegistry(); err == nil {
		t.Fatalf("expected error before write, got nil")
	}

	// 2. 写入测试值
	want := `D:\TestDataDir`
	if err := WriteDataDirToRegistry(want); err != nil {
		t.Fatalf("WriteDataDirToRegistry failed: %v", err)
	}

	// 测试结束时确保清理
	t.Cleanup(func() {
		_ = DeleteDataDirFromRegistry()
	})

	// 3. 读取应得到相同值
	got, err := ReadDataDirFromRegistry()
	if err != nil {
		t.Fatalf("ReadDataDirFromRegistry failed: %v", err)
	}
	if got != want {
		t.Fatalf("ReadDataDirFromRegistry = %q, want %q", got, want)
	}

	// 4. 删除
	if err := DeleteDataDirFromRegistry(); err != nil {
		t.Fatalf("DeleteDataDirFromRegistry failed: %v", err)
	}

	// 5. 删除后读取应失败
	if _, err := ReadDataDirFromRegistry(); err == nil {
		t.Fatalf("expected error after delete, got nil")
	}

	// 6. 再次删除应幂等（不报错）
	if err := DeleteDataDirFromRegistry(); err != nil {
		t.Fatalf("DeleteDataDirFromRegistry idempotent call failed: %v", err)
	}
}

// TestRegistry_OverwriteExisting 验证重复写入会覆盖旧值。
func TestRegistry_OverwriteExisting(t *testing.T) {
	_ = DeleteDataDirFromRegistry()
	t.Cleanup(func() {
		_ = DeleteDataDirFromRegistry()
	})

	if err := WriteDataDirToRegistry(`C:\First`); err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	if err := WriteDataDirToRegistry(`D:\Second`); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	got, err := ReadDataDirFromRegistry()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if got != `D:\Second` {
		t.Fatalf("after overwrite = %q, want %q", got, `D:\Second`)
	}
}

// TestRegistry_DeleteWithoutKey 验证 key 不存在时 Delete 不报错。
func TestRegistry_DeleteWithoutKey(t *testing.T) {
	// 直接尝试删除一个完全不存在的 key 路径下的 value
	// 这里通过先删除 value（key 可能仍存在）测试当前实现的幂等性
	_ = DeleteDataDirFromRegistry()
	if err := DeleteDataDirFromRegistry(); err != nil {
		t.Fatalf("DeleteDataDirFromRegistry should be idempotent, got: %v", err)
	}
}

// TestRegistry_PathConstants 验证我们写入的位置可被独立读取（防止 key 路径漂移）。
func TestRegistry_PathConstants(t *testing.T) {
	_ = DeleteDataDirFromRegistry()
	t.Cleanup(func() {
		_ = DeleteDataDirFromRegistry()
	})

	want := `D:\PathConstantCheck`
	if err := WriteDataDirToRegistry(want); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// 直接打开常量路径校验
	key, err := registry.OpenKey(registryRoot, registryKeyPath, registry.QUERY_VALUE)
	if err != nil {
		t.Fatalf("OpenKey direct failed: %v", err)
	}
	defer key.Close()
	got, _, err := key.GetStringValue(registryValueName)
	if err != nil {
		t.Fatalf("GetStringValue direct failed: %v", err)
	}
	if got != want {
		t.Fatalf("direct read = %q, want %q", got, want)
	}
}
