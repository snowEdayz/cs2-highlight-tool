# 用户工作目录初始化与重置

## Goal

软件启动时强制读取注册表 `HKCU\Software\CS2HighlightTool\DataDir`，作为应用数据**唯一**根目录。若未配置或无效，进入"工作目录初始化"流程——在 StartupWizard 顶层叠加一个**不可关闭的 modal**，强制用户选择一个**英文、无空格、可写、非磁盘根**的目录；校验通过后写入注册表，再启动原有组件检查/下载/自更新流程。彻底放弃 `APPDATA`/`LOCALAPPDATA` 作为默认数据根，避免中文 Windows 用户名导致 HLAE/CS2 无法识别路径。同时在 StartupWizard 工具栏提供"重置"按钮：清除当前 DataDir + 删注册表 + 回到初始化 modal。

## Requirements

### 后端 (Go)

- **R1 注册表读写**：新增 `internal/appdata/registry_windows.go` (+ `registry_other.go` 兜底空实现)：
  - `ReadDataDirFromRegistry() (string, error)` —— 读 `HKCU\Software\CS2HighlightTool\DataDir` (REG_SZ)。
  - `WriteDataDirToRegistry(path string) error` —— 写入；自动 CreateKey。
  - `DeleteDataDirFromRegistry() error` —— 仅删除 `DataDir` value，保留子键。
- **R2 路径校验**：新增 `internal/appdata/validate.go`：`ValidateDataDir(path) error` 顺序检查：
  1. 非空、去 `\"'`、`filepath.Clean`。
  2. 不是磁盘根目录（如 `C:\`、`D:\`、UNC 根）。
  3. 字符白名单 `A-Z a-z 0-9 _ - . : \ /`；分别报错"含中文/非 ASCII"、"含空格"、"含非法符号 `< > \" | ? *` 或其他"。
  4. 长度 ≤ 100（chars）。
  5. 父目录可写（尝试 `MkdirAll`）。
  6. **若目录已存在则必须为空**（`os.ReadDir` len == 0）。
  7. 可读写删测试：`<dir>/.cs2ht_init_probe` 写入随机字节 → 读回 → 删除。
- **R3 旧数据清理**：新增 `internal/appdata/cleanup_legacy_windows.go`：`CleanupLegacyData(exeDir string)`：
  - `RemoveAll(filepath.Join(os.UserCacheDir(), "CS2 Highlight Tool"))`。
  - 遍历 `exeDir` 下 legacy children (`config.json`, `hlae`, `plugin`, `ffmpeg`, `updates`, `demo`, `projects`, `outputs`, `logs`) 一并 `RemoveAll`。
  - 失败仅 `s.emitLog("warn", …)`，不阻塞。
  - 在 `SetWorkspaceDir` 成功写注册表**之后**、`runTasks` 之前调用一次。
  - 旧 `MigrateLegacyData` 移除。
- **R4 paths.go 改造**：删除 `DefaultDataDir` 中的 `os.UserCacheDir()` 逻辑；`Resolve(exeDir)` 改为只返回 `Paths{ExeDir: exeDir}`，DataDir 由更高层从注册表获得。
- **R5 App 生命周期重构** (`internal/app/app.go`)：
  - `New()`：不再 `appdata.Resolve` 全套；只 set `exeDir`；调 `appdata.ReadDataDirFromRegistry()`；若读到合法 + ValidateDataDir 通过 → 构造 `envsetup.NewWithDataDir`；否则 `service = nil`，`mode = "workspace_init"`。
  - `Startup(ctx)`：仅启动 `producews`；service 非 nil 时调 `service.Startup`；否则 emit `startup_state_changed` 携 workspace_init 状态。
  - DataDir 存在但 ValidateDataDir 失败 / 路径不存在（外部删除）→ 视同未初始化，删 value，回 workspace_init（满足 corner case "DataDir 启动时外部失踪 → 静默回初始化"）。
- **R6 envsetup 扩展** (`internal/envsetup/state.go`)：新增常量 `modeWorkspaceInit = "workspace_init"`；`StartupState.Mode` 取值集合 ∈ `{workspace_init, startup, main}`。
- **R7 新 Wails 方法** (`internal/app/app_workspace.go`)：
  - `GetWorkspaceState() WorkspaceState` —— 返回 `{Initialized bool, DataDir string, Error string}`。
  - `PickWorkspaceDir() (string, error)` —— 调 wails runtime 系统目录对话框。
  - `ValidateWorkspaceDir(path string) (ok bool, errorMessage string)` —— 给前端实时校验。
  - `SetWorkspaceDir(path string) error` —— 校验 → 写注册表 → cleanup legacy → 构造 service → emit `startup_state_changed` (mode → startup) → 触发 `RunStartupChecks`。
  - `ResetWorkspace() error` —— 读当前 DataDir → `os.RemoveAll(DataDir)` → `DeleteDataDirFromRegistry()` → 重置 service = nil → emit mode = workspace_init。
  - `ExitApp()` —— 调 `runtime.Quit(ctx)`，供 modal 里"退出应用"按钮使用。
- **R8 启动门控**：未初始化时拒绝执行 `RunStartupChecks` / `EnterMainApp` / 自更新检测，返回明确中文错误信息。

### 前端 (Vue 3 + TS)

- **R9 类型扩展** (`frontend/src/shared/types.ts`)：`StartupState.mode` 取值加 `"workspace_init"`；新增 `WorkspaceState` interface。
- **R10 新功能模块** `frontend/src/features/workspace-init/`：
  - `components/WorkspaceInitModal.vue`：`n-modal` `:closable="false"` `:mask-closable="false"` `:close-on-esc="false"`；"浏览"按钮 + 只读 `n-input`；下方校验错误红字；底部"退出应用"（次按钮）+ "下一步"（主按钮，仅校验通过亮起）。
  - `composables/useWorkspaceInit.ts`：调 `PickWorkspaceDir` / `ValidateWorkspaceDir` / `SetWorkspaceDir` / `ExitApp`。
- **R11 AppShell.vue 三态分支**：`mode === "main"` → MainApp；`mode === "workspace_init"` → StartupWizard 骨架 + `<WorkspaceInitModal />`；其他 → StartupWizard 正常态。
- **R12 StartupWizard 重置按钮**：工具栏新增"工作目录重置"按钮：
  - `disabled` 当 `state.running === true || ["downloading", "installing"].includes(state.self_update.status)`。
  - 点击 → `n-dialog`：标题"⚠️ 确认重置工作目录"，正文含完整 DataDir 路径 + "操作不可撤销，将清除该目录下的所有应用数据并重新进行环境配置"。
  - 确认 → `ResetWorkspace()` → 失败弹 error toast。
- **R13 i18n** (`zh-CN.json`)：新增 keys：`workspace.init.*`、`workspace.reset.*`、`workspace.validate.*`（错误分类）、`workspace.exit_button`。

## Acceptance Criteria

- [ ] 首次启动且注册表 value 不存在 → 显示 StartupWizard 骨架 + 不可关闭 modal；`GetStartupState` 返回 `mode = "workspace_init"`；`RunStartupChecks` 调用返回 error。
- [ ] 选择含中文路径 / 含空格 / 选磁盘根 `C:\` / 长度 > 100 / 已存在非空目录 → 校验返回分类错误信息，"下一步"按钮保持 disabled。
- [ ] 选择合法目录 → 写入注册表成功 → modal 关闭 → 自动 `RunStartupChecks` 开始原有流程 → `CleanupLegacyData` 在后台清空 LOCALAPPDATA 旧路径并记录到启动日志。
- [ ] 已初始化用户再次启动 → 直接进入 `startup` mode，不弹 modal；HLAE/Plugin/FFmpeg/config 路径均位于 DataDir 下。
- [ ] 已初始化但 DataDir 目录被外部删除/失效 → 启动检测后 silent 回到 workspace_init mode，注册表 value 被同步删除。
- [ ] `StartupState.Steps[].Path` / playdemo 命令 / `mirv_streams record name` / 生成的 plugin JSON 路径 / HLAE 启动路径 / Hook DLL 路径 / FFmpeg 路径 / 插件 DLL 路径，**实际取值**全部基于 DataDir。
- [ ] StartupWizard 重置按钮：`state.running === true` 时灰显；所有组件 ready 时可点；任意组件 failed/needs_action 时可点；selfUpdate downloading/installing 时灰显。
- [ ] 重置 → `n-dialog` 弹出含 DataDir 路径 → 确认 → DataDir RemoveAll + 注册表 value 删除 + state.mode 切回 workspace_init + 弹回初始化 modal。
- [ ] 初始化 modal 的"退出应用"按钮 → 调 `ExitApp()` → 进程退出（Wails `runtime.Quit`）。
- [ ] 重置过程 RemoveAll 失败（如文件被占用）→ 前端 error toast；状态不切换（仍在主程序/wizard）。
- [ ] 非 Windows 平台编译通过；逻辑路径走 `appdata` 的 `_other.go` 实现：注册表方法返回"not supported on this platform"错误，但 `App` 兜底使用 `os.UserConfigDir() + "CS2 Highlight Tool"` 作为 DataDir 以保留 `wails dev`。
- [ ] `go test ./...` + `cd frontend && npm run build` 全绿。

## Definition of Done

- 后端单元测试覆盖：注册表读写（mock 或 真实写到测试 key）+ ValidateDataDir 全部分支 + CleanupLegacyData 幂等性 + App 启动两种分支（已/未初始化）+ ResetWorkspace。
- `cd frontend && npm run build` 无 TS 错误、无 missing i18n key。
- zh-CN.json 文案完整；en-US.json 不动（用户维护）。
- `CLAUDE.md` "Stable Wails Public Methods" 段增补：`GetWorkspaceState`、`PickWorkspaceDir`、`ValidateWorkspaceDir`、`SetWorkspaceDir`、`ResetWorkspace`、`ExitApp`。
- 手动验证（在中文用户名 Windows VM 上）：首次启动 → 选 `D:\CS2HT` → 下载 HLAE/Plugin/FFmpeg → 启动 CS2 + HLAE 录制走通。

## Decision (ADR-lite)

- **Context**: 中文 Windows 用户名导致 `LOCALAPPDATA` 路径含中文 → HLAE/CS2 子进程无法解析 hookdll/插件路径。已有 `appdata` + `envsetup.NewWithDataDir` 抽象层，剩下的是改造前置 DataDir 解析入口 + 加 UI 流程。
- **Decision**: 注册表 `HKCU\Software\CS2HighlightTool\DataDir` 作为唯一数据根来源；mode = "workspace_init" 作为后端状态；前端在该状态下 StartupWizard + 不可关闭 modal 强制选目录；应用独占 DataDir 语义（重置 = RemoveAll 整目录）；不迁移旧数据 + 主动清理 LOCALAPPDATA 旧位置；仅 Windows 走该流程，其他平台保留 UserConfigDir 兜底。
- **Consequences**:
  - (+) 现有 `envsetup.NewWithDataDir` 完全复用，业务层零修改。
  - (+) 重置语义干净，无"已知子项清单"漂移风险。
  - (+) cleanup_legacy 一次性清理 LOCALAPPDATA 避免磁盘占用。
  - (-) 升级用户首次启动需重下 HLAE/Plugin/FFmpeg ~几百 MB（接受的代价）。
  - (-) 主程序后无重置入口，用户需重启应用回到 wizard（接受的限制）。
  - (?) 未来如需"换目录保留数据"或"DataDir 占用大小"展示可独立加 feature，不破坏现有结构。

## Out of Scope

- "只换目录、保留旧数据"模式（用户明确不要）。
- 管理员权限 / HKLM 注册表写入。
- LOCALAPPDATA → DataDir 自动迁移（按决议主动清理而非迁移）。
- DataDir picker 默认建议路径预填（按 expansion sweep 不勾选）。
- 重置部分失败的精细子项报错（按 expansion sweep 不勾选；走简单 toast）。
- Settings 页 / MainApp 内的重置入口。
- 跨平台 GUI 一致的 DataDir 强制流程（非 Windows 用 UserConfigDir 兜底）。
- 注册表 value 加密 / 防篡改。

## Implementation Plan (phased commits)

- **Phase 1 — 后端核心**：
  - 新增 `appdata/registry_{windows,other}.go` + tests
  - 新增 `appdata/validate.go` + tests
  - 新增 `appdata/cleanup_legacy_{windows,other}.go` + tests
  - 改造 `appdata/paths.go`（删 cacheDir 默认 + 删 MigrateLegacyData + 保留 samePath）
  - 改造 `internal/app/app.go` + 新增 `app_workspace.go`
  - 扩展 `envsetup/state.go` mode 常量
  - `go test ./...` 全绿
- **Phase 2 — 前端 UI**：
  - 类型扩展 (`shared/types.ts`)
  - 新增 `features/workspace-init/{components,composables}`
  - AppShell.vue 三态分支
  - StartupWizard.vue 重置按钮 + n-dialog
  - zh-CN.json 文案
  - `cd frontend && npm run build` 全绿
- **Phase 3 — 验收 + 文档**：
  - 更新 `CLAUDE.md` Stable Wails Public Methods 段
  - 端到端手动验证（如有 Windows 环境）

## Technical Notes

- 复用 `golang.org/x/sys/windows/registry`（已是依赖，被 `cs2_detect_windows.go` 使用）。`SetStringValue` / `GetStringValue` / `DeleteValue` / `CreateKey`。
- `samePath` 已存在于 `internal/appdata/paths.go`，保留。
- 启动事件序列（前端）：`onMounted` → `GetStartupState` → 若 `mode === workspace_init` 不调 `RunStartupChecks`；等用户走完 modal → 后端会主动 emit `startup_state_changed` 切到 startup mode。前端在 `EventsOn("startup_state_changed")` 监听变化做模态显隐。
- 非 Windows build 通过 `//go:build windows` / `//go:build !windows` 分别提供 registry / cleanup 实现；前者真实操作，后者返回 error 让 App 走 UserConfigDir 兜底。
- 重置流程中 `os.RemoveAll(dataDir)` 之前要确保 service 已 `Stop`（producews 不动，因为重置按钮仅当 !running 才可点；但保险起见 `service = nil` 切断后续 emit）。

## Research References

(无外部 research — 注册表 API 与 Wails event 模型在 stdlib/已有先例之中。)
