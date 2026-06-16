# refactor: POV VPK 内置 + gameinfo 单备份收敛

## Goal

重构 PR #11 引入的 POV HUD（pov.vpk）实现，修正其架构缺陷：去掉对第三方 Gitee
仓库的运行时在线下载依赖、消除 gameinfo.gi 的双备份隐患、让 POV 注入受用户开关控制、
并将 gameinfo 健康检测扩展到覆盖 `csgo/pov` 残留路径。目标是在不改变用户可见功能
（POV HUD 录制）的前提下，让实现与现有"插件 DLL + gameinfo 单备份"架构对齐。

## Background / 问题陈述

PR #11（feat: 路径中文+空格、POV HUD+VPK下载、4:3拉伸、GPU加速、日志持久化，状态 OPEN）
引入了 POV HUD 模式。经代码审查发现 4 个问题：

1. **在线下载依赖**：`preparePovForProduce` 首次使用时从 Gitee 拉取 pov.vpk
   （`https://gitee.com/zhuang-zhihao589/pov.vpk/releases/download/1/pov.vpk`），
   构成单点故障（网络抖动/仓库迁移/链接失效 → POV 录制失败），并暴露用户使用信号。
2. **gameinfo.gi 双备份**：`povSessionState` 自带 `gameInfoPath/gameInfoBackup/gameInfoModified`
   三个字段，`preparePovForProduce` 又把 gameinfo 备份成第二份 `.cs2ht_pov.bak`。但现有架构中
   gameinfo 生命周期由 `gameInfoSessionState` 单独管理（唯一一份 `.cs2ht_produce.bak`），
   插件 DLL 链路只消费 gameinfo 不备份。第二份备份带来语义歧义（哪个是真源）和顺序耦合风险。
3. **开关判断位置分散**：POV 注入与开关判断耦合在 `preparePovForProduce` 内部，gameinfo 注入
   被两个 prepare 函数分散操作，缺少单一出口。
4. **健康检测覆盖盲区**：`gameinfo_health.go` 的 `GetGameInfoHealth`/`RepairGameInfo` 只检测
   `csgo/plugin` 残留，POV 录制崩溃留下的 `csgo/pov` 残留无法被检测或修复。

## What I already know

### 现有架构（无 POV 时的正确基线）
- `produce_gameconfig.go`：
  - `gameInfoSessionState{gameInfoPath, backupPath, modified}` 单独管理 gameinfo，唯一备份
    后缀 `.cs2ht_produce.bak`。
  - `prepareGameInfoForProduce()` — 备份 + 注入 `csgo/plugin`，一次写入。
  - `preparePluginDLLForProduce()` — 只投放 plugin DLL 到 `csgo/plugin/bin/server.dll`，
    通过 `produceState.gameInfo.gameInfoPath` 复用已注入的 gameinfo，不碰 gameinfo 备份。
  - `forceRestorePluginDLLForProduce()` + `forceRestoreGameInfoForProduce()` — 各还原各的。
- `producegame/gameinfo.go`：
  - `InjectPluginSearchPath` / `HasPluginSearchPath` / `RemovePluginSearchPath` — 仅 `csgo/plugin` 硬编码。
  - `InjectSearchPath(content, searchPath)` — PR #11 已抽出的通用注入函数（plugin/pov 共用）。
- `gameinfo_health.go`：
  - `GetGameInfoHealth` / `RepairGameInfo` — 调 `HasPluginSearchPath` / `RemovePluginSearchPath`，
    仅认 `csgo/plugin`。文案已为中性"gameinfo 健康度检查"。
- `main.go`：已用 `//go:embed all:frontend/dist`，embed 基建已存在。
- `config.go`：已有 `PovHudEnabled bool json:"pov_hud_enabled"` 持久化字段。

### PR #11 引入的 POV 实现（待重构对象）
- `produce_gameconfig.go`：`povSessionState`（含 gameinfo 三字段）、`preparePovForProduce`（下载 +
  投放 vpk + 第二份 gameinfo 备份 + 注入 pov）、`forceRestorePovForProduce`、
  `forceRestoreProduceEnvironmentForProduce` 新增调用 POV 恢复。
- `produce_session.go`：`produceSessionState` 新增 `pov povSessionState` 字段。
- `plugin_generate.go`：`GeneratePluginJSONBatchAndLaunchHLAE` 在 launch 前调
  `preparePovForProduce`。

## Assumptions (temporary)

- pov.vpk 是稳定的 HUD 补丁，版本与 app 版本绑死可接受。
- 用户只在本地 Demo 回放录制时使用 POV，不连 VAC 服务器（HLAE 启动行已强制 `-insecure`）。

### 已核实事实（auto-context）
- **pov.vpk 实际体积 = 116,781 字节（约 114 KB）** — embed 进二进制体积完全可接受。
- **vPK 文件已落地**：用户已将 `pov.vpk` 放置并移入 `internal/producegame/assets/pov.vpk`，
  随仓库分发无版权问题（作者仓库为转载，仅漏放许可文件）。R1 确定为**纯内置**，无在线下载兜底。

## Open Questions

- 无（全部已解决）。

## Decision (ADR-lite)

**Context**: PR #11 的 POV 实现存在 4 个问题：在线下载单点故障、gameinfo 双备份、
开关判断分散、健康检测盲区。其中 R1/R2/R3 共享同一组函数（`prepareGameInfoForProduce` /
`preparePovForProduce` / `povSessionState`），分阶段会产生半成品中间态。

**Decision**:
- R1/R2/R3/R4 **全部进 MVP**，一次性收敛。理由：R2 是地基，R3/R1 围绕它落地，
  R4 独立但成本极小，合并做最干净、回归风险最低。
- vpk 采用**纯内置**（`go:embed`），无在线下载兜底。版本跟 app 走。
- gameinfo **唯一备份**（沿用 `.cs2ht_produce.bak`），删除 `.cs2ht_pov.bak`。
- POV 的 `povSessionState` 退化为**仅管 vpk 文件**，不再持有 gameinfo 字段。

**Consequences**:
- ✅ 去网络依赖，无单点故障；gameinfo 备份语义无歧义；开关判断集中；
  崩溃残留可被健康检测修复。
- ⚠️ vpk 更新需发新 app 版本（可接受，HUD 补丁稳定）。
- ⚠️ 这是对未合并 PR #11 的**重构**，需在 PR #11 合入前/合入时同步应用，避免架构债沉淀。

## Technical Approach

### 架构目标：两条独立链路，gameinfo 单一出口

```
gameinfo 链路（唯一备份 .cs2ht_produce.bak，由 gameInfoSessionState 管理）:
  prepareGameInfoForProduce()
    → 读 cfg.PovHudEnabled 决定注入集合 {csgo/plugin} 或 {csgo/plugin, csgo/pov}
    → 1 次备份，1 次写入
  forceRestoreGameInfoForProduce()
    → 1 次还原（那唯一一份），还原成干净原始态

vpk 文件链路（物理文件，独立，由 povSessionState 管理）:
  preparePovForProduce()
    → 仅当 cfg.PovHudEnabled：embed 释放 → dataDir/presets/pov/ → 复制到 csgo/pov.vpk
    → vpk 自身备份（若 csgo/pov.vpk 已存在则备份为 .cs2ht_pov.bak ← 仅 vpk 文件用此后缀）
  forceRestorePovForProduce()
    → 仅还原/删除 vpk 文件，完全不碰 gameinfo
```

### 按需求拆解的实现要点

**R1 内置 vpk**（`internal/producegame/`）
- 新增 `povpreset.go`：
  ```go
  import _ "embed"
  //go:embed assets/pov.vpk
  var povVPK []byte
  ```
- `preparePovForProduce` 中删除 Gitee `download.File` 分支，改为：
  ```go
  if _, err := os.Stat(povSrc); err != nil {
      os.MkdirAll(povDir, 0755)
      os.WriteFile(povSrc, producegame.PovVPK, 0644)
  }
  ```
- 资产已就位：`internal/producegame/assets/pov.vpk`（114 KB）。

**R2 gameinfo 单备份收敛**（`internal/app/produce_gameconfig.go`）
- `povSessionState` 删除 `gameInfoPath / gameInfoBackup / gameInfoModified` 三字段，
  仅保留 `vpkPath / vpkInstalled / vpkBackup`。
- `prepareGameInfoForProduce` 改为收集注入集合后统一注入：
  ```go
  searchPaths := []string{"csgo/plugin"}
  if cfg.PovHudEnabled { searchPaths = append(searchPaths, "csgo/pov") }
  ```
  循环 `InjectSearchPath`，一次备份、一次写入。
- 删除 `preparePovForProduce` 里所有 gameinfo 备份/注入/回滚代码。
- `forceRestorePovForProduce` 删除 gameinfo 还原分支，只还原 vpk。
- **注意**：`.cs2ht_pov.bak` 后缀**保留给 vpk 文件自身备份**（vpk 是独立物理文件，需独立备份），
  只是不再用于 gameinfo。

**R3 开关判断**（落在 R2 的 `prepareGameInfoForProduce`）
- 注入 `csgo/pov` 前判断 `cfg.PovHudEnabled`；`preparePovForProduce` 入口也复用同一判断
  （双重保险，确保开关关闭时既不注入也不投放 vpk）。

**R4 健康检测扩展**（`internal/producegame/gameinfo.go` + `internal/app/gameinfo_health.go`）
- `gameinfo.go` 新增对称通用函数：
  ```go
  func HasSearchPath(content, searchPath string) bool
  func RemoveSearchPath(content, searchPath string) (string, bool)
  ```
  `HasPluginSearchPath`/`RemovePluginSearchPath` 保留为 `csgo/plugin` 的薄封装（兼容现有调用 + 测试）。
- `gameinfo_health.go`：检测与修复同时覆盖两条路径：
  ```go
  if producegame.HasSearchPath(content, "csgo/plugin") ||
     producegame.HasSearchPath(content, "csgo/pov") { ... }
  ```
  修复时两个路径都 `RemoveSearchPath`。文案零改动（已是中性）。

## Implementation Plan (small PRs)

- **PR1 — 基建 + R4（低风险独立项）**
  - `producegame/gameinfo.go`：新增 `HasSearchPath`/`RemoveSearchPath`，plugin 版改为薄封装。
  - `gameinfo_health.go`：检测/修复覆盖 plugin + pov 双路径。
  - 测试：新增 pov 路径残留检测/修复用例。
  - *可独立合入，不依赖 R1/R2/R3。*

- **PR2 — R2 + R3（gameinfo 单备份收敛 + 开关判断）**
  - `povSessionState` 去掉 gameinfo 字段；`prepareGameInfoForProduce` 成为唯一注入出口（含开关判断）。
  - `preparePovForProduce`/`forceRestorePovForProduce` 剥离 gameinfo 逻辑。
  - 测试：插件+pov 同开时 gameinfo 单备份还原、开关关闭分支。
  - *依赖 PR1 的 `HasSearchPath`（可选，也可内部临时用 plugin 版）。*

- **PR3 — R1（内置 vpk）**
  - 新增 `producegame/povpreset.go`（embed）；资产已就位。
  - `preparePovForProduce` 删除下载分支，改 embed 释放。
  - 测试：内置释放、多会话复用。
  - *依赖 PR2 的 `preparePovForProduce` 新结构。*

## Acceptance Criteria (evolving)

- [ ] POV 开启时：vpk 从内置释放，全程无任何网络请求（除非策略选内置兜底）。
- [ ] POV 关闭时：不投放 vpk、不注入 `csgo/pov`、不产生任何 pov 相关副作用。
- [ ] gameinfo.gi 整个会话只有一个备份文件 `.cs2ht_produce.bak`，无 `.cs2ht_pov.bak`。
- [ ] 插件 + POV 同时开启时：gameinfo 同时含 `csgo/plugin` 与 `csgo/pov`，恢复后还原成干净原始态。
- [ ] POV 录制中途崩溃留下 `csgo/pov` 残留：`GetGameInfoHealth` 报 `needs_repair`，
      `RepairGameInfo` 能清除。
- [ ] 现有插件 gameinfo 流程行为不变（回归不破坏）。
- [ ] `go test ./...` 通过；`cd frontend && npm run build` 通过（前端无改动则仅后端校验）。

## Definition of Done

- 单元/集成测试覆盖：gameinfo 单备份还原、开关分支、健康检测双路径覆盖。
- `go test ./...` 绿；如改前端则 `npm run build` 绿。
- AGENTS.md 稳定契约更新（如涉及 pov 持久化字段/行为契约变化）。
- `.cs2ht_pov.bak` 不再用于 gameinfo（grep 确认仅 vpk 文件备份保留此后缀）。
- **【spec 维护】** `.trellis/spec/backend/wails-bindings.md` 的 "Gameinfo Health Repair Contract"
  场景当前硬编码为仅检测 `csgo/plugin`（Contract 第 3/4 条、Tests 第 6 条、Good Case 第 5 条）。
  R4 扩展覆盖 `csgo/pov` 后，**必须同步更新该 spec 场景**（按 AGENTS.md "维护触发器"规则：
  下载源/回退/检测策略变化触发 spec 更新），否则 spec 与实现漂移。

## Out of Scope (explicit)

- 不改变 POV HUD 的用户可见行为（仍是临时装 vpk + 改 gameinfo，录完还原）。
- 不引入 pov.vpk 的组件管理流程（不走 envsetup 的 component 安装链路）。
- 不重构插件 DLL 流程的内部结构（仅当 R2 要求时让 plugin/pov 共用 gameinfo 出口）。
- 不改 HLAE `-insecure` 启动策略。

## Technical Notes

- 关键文件：
  - `internal/app/produce_gameconfig.go`（核心重构点）
  - `internal/app/produce_session.go`（`produceSessionState.pov` 结构调整）
  - `internal/app/plugin_generate.go`（`preparePovForProduce` 调用点）
  - `internal/app/gameinfo_health.go`（健康检测扩展）
  - `internal/producegame/gameinfo.go`（`InjectSearchPath` 已有，需补 `Has/RemoveSearchPath`；
    **新增 embed：`internal/producegame/assets/pov.vpk` 已就位**）
  - `internal/app/produce_session_test.go`（测试更新）
- 约束（来自 AGENTS.md）：
  - 不手工改自动生成文件（`frontend/wailsjs/**` 等）。
  - `pov_hud_enabled` 是已有稳定契约字段，本任务不改其语义，仅改实现。
  - gameinfo 备份/恢复是稳定契约，收敛后 `.cs2ht_produce.bak` 仍是唯一备份后缀。
