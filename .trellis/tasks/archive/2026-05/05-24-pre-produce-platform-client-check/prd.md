# 制作前检测并关闭对战平台客户端进程

## Goal

在点击"开始制作"之前，检测用户电脑上是否有对战平台客户端（如完美世界竞技平台、5E 客户端）正在运行。HLAE 通过 DLL 注入启动 CS2，该行为可能与对战平台的反作弊系统不兼容导致蓝屏，因此必须确保所有对战平台客户端关闭后才允许开始录制。

## Requirements

- 点击"开始制作"时，先后端检查所有平台客户端进程状态
- 如果**全部未运行**，直接跳过弹窗，继续原有制作流程（无感）
- 如果**任意一个运行中**，弹出检测对话框，显示各客户端的运行状态
- 对话框中每个运行中的客户端有单独的"退出"按钮，同时有"全部退出"按钮（一键发送所有退出信号）
- 点击退出后：显示 spinner，自动轮询（每 ~500ms，最多 5 秒），退出成功后状态自动更新为"未运行"
- 超时后客户端仍在运行：显示"请手动关闭"提示，不再自动重试
- 全部关闭后："继续制作"按钮变为可点击，等用户手动点击确认后才执行制作流程
- 封装良好：平台客户端列表为配置，后续添加新客户端只改列表，不改业务逻辑
- 非 Windows 平台：返回空列表（不阻断流程）

## Acceptance Criteria

- [ ] 所有平台客户端未运行时，点击"开始制作"无弹窗，直接执行原流程
- [ ] 有客户端运行时，弹出对话框，正确显示各进程运行状态
- [ ] 点击单个"退出"按钮后，对应行显示 spinner，5 秒内退出成功则状态更新
- [ ] "全部退出"按钮同时对所有运行中的客户端发送退出信号
- [ ] 超时（5 秒后仍在运行）显示"请手动关闭"提示
- [ ] 所有客户端未运行时，"继续制作"按钮可点击；有任意运行中则禁用
- [ ] 用户点击"继续制作"后，执行 `GeneratePluginJSONBatchAndLaunchHLAE`
- [ ] go build（windows + 非 windows）均通过
- [ ] `cd frontend && npm run build` 通过

## Definition of Done

- 后端 Windows / non-Windows build 均通过（go build）
- Frontend type-check 通过（npm run build）
- zh-CN.json 已更新，新文案已添加
- 现有 `go test ./...` 通过

## Technical Approach

### 后端（internal/app）

新文件组：
- `platform_client.go` — `PlatformClientConfig` 结构体 + 全局客户端列表 + `PlatformClientStatus` 类型
- `platform_client_windows.go`（`//go:build windows`）— 进程枚举 + WM_CLOSE 实现
- `platform_client_other.go`（`//go:build !windows`）— 返回空/not-supported 存根

客户端配置列表（可扩展）：
```go
var platformClients = []PlatformClientConfig{
    {ExeName: "完美世界竞技平台.exe", DisplayName: "完美世界竞技平台"},
    {ExeName: "5EClient.exe",        DisplayName: "5E 对战平台"},
}
```

新 Wails 方法（加到 app.go 或单独文件）：
- `CheckPlatformClients() []PlatformClientStatus` — 快照检查，每个客户端返回 `{exe_name, display_name, running, pid}`
- `RequestClosePlatformClient(exeName string) PlatformClientCloseResult` — 发 WM_CLOSE，等待 grace period（约 3 秒），返回是否已退出

### 前端（frontend/src）

- 新文件 `features/produce/composables/usePlatformClientCheck.ts` — 调用 `CheckPlatformClients`、`RequestClosePlatformClient`，管理各客户端状态、spinner、超时逻辑
- 新组件 `features/produce/components/PlatformClientCheckModal.vue` — `n-modal` 对话框，展示状态列表 + 退出按钮 + 继续按钮
- 修改 `useProducePage.ts` 中 `generateAndLaunch()`：在调用后端前先触发平台客户端检查

## Decision (ADR-lite)

**Context**: 需要在开始录制前强制检查并关闭对战平台客户端，避免 HLAE DLL 注入导致蓝屏。

**Decision**: 前端 modal 拦截 + 后端轮询检测；前端驱动 UX（spinner、按钮状态），后端提供无状态检查/关闭 API。

**Consequences**: 后端无需维护检测状态，前端控制 UX 节奏，可测试性更好。代价是需要前端轮询（可接受，因为最多 5 秒）。

## Out of Scope

- 强制终止平台客户端进程（`TerminateProcess`）——仅优雅退出
- 非 Windows 平台的实际实现
- 自动检测平台客户端安装路径

## Technical Notes

### 现有进程操作基础设施可复用
- `cs2_process_windows.go`：`CreateToolhelp32Snapshot` + `WM_CLOSE` + `waitForProcessExit` 模式
- `cs2_process_other.go`：`!windows` 编译存根模式
- 新代码直接沿用同样模式，按进程名（大小写不敏感）匹配 PID

### 涉及文件
**新建：**
- `internal/app/platform_client.go`
- `internal/app/platform_client_windows.go`
- `internal/app/platform_client_other.go`
- `frontend/src/features/produce/composables/usePlatformClientCheck.ts`
- `frontend/src/features/produce/components/PlatformClientCheckModal.vue`

**修改：**
- `internal/app/app.go` 或新文件暴露 Wails 方法
- `frontend/src/features/produce/composables/useProducePage.ts` — `generateAndLaunch()` 拦截
- `frontend/src/features/produce/pages/ProducePage.vue` — 引入 modal 组件
- `frontend/src/shared/types.ts` — 新增 `PlatformClientStatus`、`PlatformClientCloseResult` 类型
- `frontend/src/shared/i18n/zh-CN.json` — 新增文案
- `frontend/wailsjs/go/app/App.d.ts` — 新增绑定（wails generate 后更新）
- `frontend/wailsjs/go/models.ts` — 新增类型绑定（wails generate 后更新）
