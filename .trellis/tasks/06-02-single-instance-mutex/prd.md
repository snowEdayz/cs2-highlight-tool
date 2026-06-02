# PRD: 单例运行 (Single Instance Mutex)

## 需求
- CS2 Highlight Tool 启动时检查是否已有实例在运行
- 如果已有实例，直接静默退出（不弹框、不通知用户）
- 自更新流程不受影响

## 方案
Windows Named Mutex（命名内核互斥体）：
- 使用 `CreateMutexW` 创建名为 `Local\CS2HighlightTool` 的 Mutex
- 如果返回 `ERROR_ALREADY_EXISTS`，则说明已有实例运行
- 进程退出时内核自动释放 Mutex

## 文件改动

### 新增 2 个文件
- `internal/app/single_instance_windows.go` — Windows 实现
- `internal/app/single_instance_other.go` — 非 Windows 桩

### 修改 1 个文件
- `main.go` — 在 `--apply-update` 跳过逻辑之后、`wails.Run()` 之前加入检查

## 更新流程兼容性验证（已确认）
1. `--apply-update` 分支在最前面，跳过互斥检查 ✓
2. updater 进程（Instance B）是 `--apply-update` 模式，跳过检查 ✓
3. 原进程退出 → Mutex 自动释放 → 新版本正常创建新 Mutex ✓
4. 更新进行中时，用户二次启动被正确拦截 ✓
