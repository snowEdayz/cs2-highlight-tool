# feat: POV HUD 录制（embedded vpk + gameinfo 单出口）

## Goal

为 CS2 highlight tool 增加 POV HUD 录制能力 —— 录制过程中临时启用一个内置 HUD 补丁（`pov.vpk`），
让录制画面包含 POV-style 玩家状态显示，录制完毕自动还原。与上一个任务（archived `pov-vpk-embed-single-gameinfo-backup`）
不同的是：此次**不依赖 PR #11**，POV 运行时代码完全从零按目标架构落地，避免 PR #11 的架构债。

## Background

上一个任务原计划用 R1+R2+R3+R4 一次性"重构"PR #11 引入的 POV 实现：
- ✅ **R4 已完成 + PR #15 已合入 main**：`producegame` 通用 SearchPath helper + `gameinfo_health.go` 双路径覆盖 +
  `pov.vpk` 资产入库 + `wails-bindings.md` spec 更新。
- ❌ **R1/R2/R3 未做**：当时 main 上没有 PR #11 的 POV 运行时代码可重构。

本次任务**绕过 PR #11**，直接按目标架构实现 POV HUD 录制，把 PR #11 当作不存在。

## What I already know（基于当前 main，commit 062663b）

### 现有 produce 环境管理（plugin 链路，POV 链路的基线参照）

- `internal/app/produce_gameconfig.go`：
  - `produceGameInfoBackupSuffix = ".cs2ht_produce.bak"`（gameinfo 唯一备份）
  - `gameInfoSessionState{gameInfoPath, backupPath, modified}` 单备份模型
  - `prepareGameInfoForProduce()` (line 35) — 当前仅检查/注入 `csgo/plugin`：
    `HasPluginSearchPath` 早返回，否则 `InjectPluginSearchPath` + 备份 + 写入
  - `preparePluginDLLForProduce()` (line 84) — DLL 投放，独立备份 `.cs2ht_plugin.bak`
  - `forceRestoreGameInfoForProduce()` (line 205) — 单备份还原
  - `forceRestorePluginDLLForProduce()` (line 227) — DLL 还原 + 清理空目录
  - `forceRestoreProduceEnvironmentForProduce()` (line 273) — DLL → gameinfo 顺序还原
- `internal/app/plugin_generate.go:320-334`：launch 前调用
  `prepareGameInfoForProduce` → `preparePluginDLLForProduce` → `launchHLAEGame`，
  任何一步失败立即 `forceRestoreProduceEnvironmentForProduce`
- `internal/app/produce_session.go:56`：`produceSessionState{ gameInfo, pluginDLL, ... }`（无 `pov` 字段）

### PR #15 已落地的可复用基础

- `internal/producegame/gameinfo.go`：
  - `SearchPathPlugin = "csgo/plugin"`、`SearchPathPOV = "csgo/pov"` 常量
  - 通用 `InjectSearchPath(content, path) → (content, ok)`
  - 通用 `HasSearchPath(content, path) → bool`
  - 通用 `RemoveSearchPath(content, path) → (content, changed)`
  - 老的 `*PluginSearchPath` 三个函数保留为薄封装
- `internal/producegame/assets/pov.vpk`：114 KB 资产已入库（未接 `go:embed`）
- `internal/app/gameinfo_health.go`：`knownInjectedSearchPaths()` 闭包驱动 plugin + pov 残留检测/修复
- `.trellis/spec/backend/wails-bindings.md`：Gameinfo Health Repair Contract 已记录闭包契约 + asymmetric-mechanism gotcha

### 当前 main 上**不存在**的东西（本次需新建）

- ❌ `Config.PovHudEnabled` 持久化字段（grep 全空）
- ❌ 任何 POV 运行时代码（无 `preparePovForProduce` / `povSessionState` / `pov.vpk` 投放逻辑）
- ❌ 前端 POV HUD 开关 UI
- ❌ `povpreset.go`（`go:embed` 接线）

## Open Questions

- 见下方 Q&A loop。

## Requirements (evolving)

- POV 开关字段进 `Config` 并持久化（默认关闭）
- 仅当开关开启时：embed 释放 vpk → 投放到 `csgo/pov.vpk` → 注入 `csgo/pov` 搜索路径
- gameinfo 单备份：`prepareGameInfoForProduce` 成为唯一注入出口，一次备份/一次写入完成 plugin (+pov 当开关开启)
- 录制完毕自动还原：gameinfo 还原 + vpk 文件还原/删除
- POV 录制崩溃留下 `csgo/pov` 残留：`GetGameInfoHealth` 已经能报 `needs_repair`，`RepairGameInfo` 已经能清除（PR #15 已支持）

## Acceptance Criteria (evolving)

- [ ] `PovHudEnabled=false`：不投放 vpk、不注入 `csgo/pov`、不产生任何 pov 相关副作用（与无 POV 时行为完全一致）
- [ ] `PovHudEnabled=true`：vpk 从内置释放（无网络请求），gameinfo 同时含 `csgo/plugin` + `csgo/pov`，录完两者都还原
- [ ] gameinfo.gi 整个会话只有一个备份文件 `.cs2ht_produce.bak`
- [ ] `csgo/pov.vpk` 用 `.cs2ht_pov.bak` 备份（仅当 vpk 文件预先存在）
- [ ] 现有插件 gameinfo 流程行为不变（回归不破坏）
- [ ] `go test ./...` 通过；如改前端则 `cd frontend && npm run build` 通过

## Definition of Done

- 单元/集成测试覆盖：单备份还原（plugin+pov 同开）、开关分支、embed 释放幂等
- `go test ./...` 绿
- 改前端则 `cd frontend && npm run build` 绿
- 稳定契约文档（CLAUDE.md / `.trellis/spec/backend/wails-bindings.md`）按需更新

## Decision (ADR-lite)

### D1: Scope 包含前端 toggle UI
**Context**: 后端字段 + 前端开关谁先做？  
**Decision**: 本任务后端 + 前端 toggle UI 一起做。`PovHudEnabled` 字段沿用 `EnableSpecShowXray` / `HideAllUI`
现有 pattern：`Config` 持久化 + `ClipSettings` 镜像字段 + `Get/SaveClipSettings` 双向同步 +
`SettingsPanel.vue` 复选框。  
**Consequences**: ✅ 一次交付完整可用功能。⚠️ 涉及 `zh-CN.json`（按 CLAUDE.md 仅改 zh-CN，en-US 由用户维护）。

### D2: 崩溃残留只清 gameinfo，不清 vpk
**Context**: POV 录制中途崩溃会留下 gameinfo 的 `csgo/pov` 注入 + `csgo/pov.vpk` 文件。  
**Decision**: 健康检查/修复**只覆盖 gameinfo**（PR #15 已落地）。vpk 文件残留不进修复链路。  
**Consequences**: ✅ 一旦 `csgo/pov` 搜索路径从 gameinfo 被移除，CS2 就不会再加载 `csgo/pov.vpk`，
残留 114 KB 文件无害可忽略。✅ 不动 `wails-bindings.md` 的 Gameinfo Health Repair Contract（PR #15 刚改完）。

### D3: vpk 文件"只写不存在，只删自己写"，**无 `.cs2ht_pov.bak`**
**Context**: 如果用户预先在 `csgo/pov.vpk` 放了自己的文件，我们直接覆盖会摧毁用户数据；
但搞备份后缀又会引入第二份备份文件，违反"源文件备份只有一份"原则。  
**Decision**: `preparePovForProduce` 进入时**先 Stat 检查**：
- 如果 `csgo/pov.vpk` 不存在 → embed 释放写入，记录 `vpkInstalled=true`
- 如果 `csgo/pov.vpk` 已存在 → **不动**，记录 `vpkInstalled=false`（沿用用户自带文件）

`forceRestorePovForProduce` 仅在 `vpkInstalled=true` 时删除 `csgo/pov.vpk`，否则跳过。
**完全不引入 `.cs2ht_pov.bak`**。  
**Consequences**: ✅ 用户自带 pov.vpk 不被破坏（虽是极少数情况）。✅ 备份语义零歧义 —— 整个 produce 会话只有
`.cs2ht_produce.bak` 一份备份（属于 gameinfo），无第二份 .bak。✅ `povSessionState` 退化到
`{vpkPath, vpkInstalled}` 两个字段。

## Out of Scope (本次明确不做)

- 不修改 `wails-bindings.md` 的 Gameinfo Health Repair Contract（PR #15 刚更新过，本任务无需再动）
- 不引入 `.cs2ht_pov.bak`（D3 决策）
- 不做 vpk 文件层的健康检查/修复（D2 决策）
- 不动 HLAE 启动参数 / `-insecure` 策略
- 不增加在线下载兜底（vpk 纯内置）
- 不改 plugin DLL 投放流程（仅在 launch sequence 里加 POV 调用点）
- en-US 翻译由用户维护，本任务仅改 zh-CN.json

## Technical Approach

### 两条链路（gameinfo 单出口 + vpk 文件独立链）

```
gameinfo 链路（唯一备份 .cs2ht_produce.bak，由 gameInfoSessionState 管理）：
  prepareGameInfoForProduce()
    → 读 cfg.PovHudEnabled 决定注入集合 paths := {"csgo/plugin"}
       if cfg.PovHudEnabled: paths = {"csgo/plugin", "csgo/pov"}
    → 检查 gameinfo 是否已含所有目标路径（HasSearchPath 全部 true）→ 早返回
    → 1 次备份 (.cs2ht_produce.bak)
    → 循环 InjectSearchPath 注入 → 1 次写入

  forceRestoreGameInfoForProduce()
    → 现有逻辑不变（单备份还原）

vpk 文件链路（独立物理文件，无 backup，由 povSessionState 管理）：
  preparePovForProduce()  // 仅当 cfg.PovHudEnabled
    → Stat csgo/pov.vpk：
        存在 → vpkInstalled=false（沿用用户文件）
        不存在 → 用 embed 字节写入，vpkInstalled=true
    → 失败时同 plugin DLL，触发 forceRestoreProduceEnvironment

  forceRestorePovForProduce()
    → if vpkInstalled: os.Remove(vpkPath)
    → else: 跳过
```

### 关键文件改动清单

**后端（基于 commit 062663b）**：

| 文件 | 改动 |
|---|---|
| `internal/config/config.go` | 新增 `PovHudEnabled bool \`json:"pov_hud_enabled"\``；默认 false |
| `internal/producegame/povpreset.go` | **新建**：`//go:embed assets/pov.vpk` → `var PovVPK []byte` |
| `internal/app/clip_settings.go` | `ClipSettings` 加 `PovHudEnabled bool`；`Get/SaveClipSettings` 同步 |
| `internal/app/produce_gameconfig.go` | 新增 `povSessionState{vpkPath, vpkInstalled}`；改 `prepareGameInfoForProduce` 走 paths 集合；新增 `preparePovForProduce` / `forceRestorePovForProduce`；`forceRestoreProduceEnvironmentForProduce` 加 POV 还原 |
| `internal/app/produce_session.go` | `produceSessionState` 加 `pov povSessionState` 字段 |
| `internal/app/plugin_generate.go:~320` | launch 顺序：`prepareGameInfoForProduce` → `preparePluginDLLForProduce` → `preparePovForProduce`（仅 PovHudEnabled 时） → `launchHLAEGame` |

**前端**：

| 文件 | 改动 |
|---|---|
| `frontend/src/shared/types.ts` | `ClipSettings` 接口加 `pov_hud_enabled: boolean` |
| `frontend/src/features/settings/components/SettingsPanel.vue` | 复选框，复用 `EnableSpecShowXray` / `HideAllUI` 的样式 |
| `frontend/src/shared/i18n/zh-CN.json` | 新增 label & description；en-US 由用户后续维护 |

**测试**：

| 文件 | 用例 |
|---|---|
| `internal/app/produce_session_test.go` | (a) PovHudEnabled=true 时 gameinfo 同时含 plugin+pov，单备份还原后干净；(b) PovHudEnabled=false 时不写 vpk、不注入 pov；(c) preparePovForProduce 在 vpk 已存在时跳过写入，restore 不删 |
| `internal/producegame/povpreset_test.go` | 验证 `PovVPK` 非空且大小符合预期 (~114 KB) |
| `internal/app/clip_settings_test.go`（如有） | Get/SaveClipSettings 的 `pov_hud_enabled` 持久化往返 |

## Implementation Plan (single PR)

整个 MVP 一个 PR：

1. **Config + ClipSettings 字段** — 加 `PovHudEnabled` 到 `Config` + `ClipSettings` + `Get/SaveClipSettings` 同步
2. **povpreset embed** — 新建 `internal/producegame/povpreset.go`，添加最小测试验证字节
3. **后端 produce 流程改造**：
   - `prepareGameInfoForProduce` 接受 PovHud 开关，走通用 paths 集合
   - 新增 `povSessionState` + `preparePovForProduce` + `forceRestorePovForProduce`
   - `produceSessionState.pov` 字段
   - `plugin_generate.go` launch sequence 加 POV prepare/restore 调用
   - 测试覆盖 plugin+pov 同开 / 关闭 / vpk 预存在
4. **前端 UI**：
   - `shared/types.ts` 加字段
   - `SettingsPanel.vue` 加复选框
   - `zh-CN.json` 加 i18n key
5. **验证**：`go test ./...` + `cd frontend && npm run build`，自检后送 PR

## Technical Notes

- **稳定契约**（新增）：`pov_hud_enabled` 配置字段、`ClipSettings.PovHudEnabled` 是新增稳定契约。
- **约束**（来自 CLAUDE.md）：
  - 不手工改自动生成文件（`frontend/wailsjs/**`、`auto-imports.d.ts`、`components.d.ts`）
  - 仅改 `zh-CN.json`，`en-US.json` 由用户维护
  - gameinfo 备份/恢复是稳定契约，`.cs2ht_produce.bak` 仍是唯一备份后缀
  - **不引入任何额外 `.bak` 文件**（D3 决策）
