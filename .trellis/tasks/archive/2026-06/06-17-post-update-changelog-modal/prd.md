# 更新后首启展示版本更新日志

## Goal

软件自更新到新版本后，用户首次启动时弹出当前版本的更新日志（CHANGELOG）；同一版本只展示一次，关闭后写入"已确认版本"，下次启动不再弹。
让用户清晰感知到这个版本带来了什么变化，提升对自更新机制的信任感。

## What I already know

**后端现状：**
- `internal/release.CurrentAppVersion(wailsConfigData)` 已从 embed 的 `wails.json` 读出当前版本（当前是 `2.0.2`）。
- `internal/release.CompareVersions(a, b)` 已有，支持 semver 比较。
- `internal/app.App.version` 字段已持有当前版本字符串，可直接复用。
- `internal/config/Config` 是单一 `config.json` 的扁平结构体；`LoadOrCreate` 已有"新字段未在 json 中存在则用默认值"的兜底范式（参见 `enable_spec_show_xray_zero` / `sky_blackout` / `kill_feed_lifetime`）。
- 自更新走 `internal/updater.StartApply` → 子进程替换 exe → 重启；重启后是新版本进程，正常启动。
- `runtime.EventsEmit(ctx, name, payload)` 是 push 通道；前端通过 `window.go.app.App.*` 调用同步方法。

**前端现状：**
- Naive UI 提供 `n-modal` / `n-drawer`，无需新增组件库。
- `frontend/package.json` 没有 markdown 渲染库；要支持 markdown 渲染需新增依赖（如 `marked`）。
- i18n 只能改 `zh-CN.json`（CLAUDE.md 硬约束）。
- 入口顺序：`AppShell.vue` → startup wizard → `EnterMainApp()` 后路由跳到 `/import`。

## Assumptions (temporary)

- changelog 内容跟随版本写死，通过 Go `//go:embed` 打进二进制。
- 弹窗在"进入主应用"那一刻触发（不是软件刚启动的 splash 阶段）；首次安装的用户不弹。
- 跨版本累积合并暂不实现 MVP（v1.0 → v1.3 只显示 v1.3 的日志）。

## Open Questions

- ~~Q1: changelog 文件结构~~ **已定：每版本一个 .md 文件，embed 进二进制**
- ~~Q2: 渲染格式~~ **已定：方案 B —— Markdown + `marked` + `DOMPurify` 前端渲染**
- ~~Q3: 触发策略~~ **已定：所有版本变化都弹（含 patch）；理由：本项目每个 v2.0.x 都承担功能更新语义**
- ~~Q4: changelog 内容语言~~ **已定：双语单文件，按 `## 中文` / `## English` 二级标题分段，前端按 locale 选段**
- ~~Q5: 元信息来源~~ **已定：方案 x —— 纯 markdown 双段，无 frontmatter；标题写死成 `更新到 v{ver}` / `Updated to v{ver}`，不存日期**
- ~~Q6: 异常路径~~ **已定：方案 α —— 运行时静默跳过 + 写回 last_changelog_version；并加单测在编译期保证 wails.json 的纯三段数字版本必在 embed 中**

## Requirements

### 内容存储
- 每版本一个 markdown 文件：`internal/changelog/notes/v<X.Y.Z>.md`
- 文件内容双语并存，按 `## 中文` / `## English` 两个二级标题分段
- 文件通过 `//go:embed notes/*.md` 打进二进制；不含 yaml frontmatter
- 模态框标题写死成 `更新到 v{ver}` / `Updated to v{ver}`，不存日期、不存独立标题

### 后端
- 新增 `internal/changelog/` 包，暴露：
  - `Get(version string) (Notes, bool)` —— 读 embed 内容、切分双语段返回；找不到返回 `(Notes{}, false)`
  - `Notes{ Version, BodyZh, BodyEn string }`
- 在 `internal/config/Config` 新增字段 `LastChangelogVersion string \`json:"last_changelog_version,omitempty"\``
- `Config.LoadOrCreate`：
  - 当 `config.json` 不存在（全新安装）→ `Default()` 把 `LastChangelogVersion` 直接置为当前 `App.version`
  - 当 `config.json` 存在但缺 `last_changelog_version` key → 视为"未确认"，按正常流程允许弹一次（沿用现有 `strings.Contains(string(data), "...")` 的兜底范式不适用此场景，因为我们恰恰要在缺失时返回 ""，不主动覆盖）
- 新增 Wails 方法：
  - `GetPendingChangelog() PendingChangelog` —— 返回 `{ Version, BodyZh, BodyEn, ShouldShow bool }`
    - 当 current == last → ShouldShow=false
    - 当 changelog 文件缺失 → ShouldShow=false，但仍然不写回（写回交给前端 ack）
  - `AckChangelog(version string) error` —— 用户关闭弹窗时调用，把 `LastChangelogVersion` 写回 config

### 前端
- 新增 `frontend/src/features/changelog/` 模块：
  - `ChangelogModal.vue` —— Naive UI `n-modal` 展示 markdown，关闭时调 `AckChangelog`
  - `useChangelog.ts` —— 在 `EnterMainApp` 成功后调 `GetPendingChangelog`，`ShouldShow=true` 时挂载 modal
- 新增前端依赖：`marked` + `dompurify` + `@types/dompurify`
- 按当前 i18n locale 选择 `BodyZh` / `BodyEn`；缺一种语言时回退到另一种
- 触发位置：`AppShell.vue` 或 startup wizard 结束跳转 `/import` 后；具体由前端实现侧决定

### 触发策略
- 任意版本变化都触发弹窗（含 patch），理由：本项目每个 patch 都承担功能更新语义
- 跨多版本时只展示最新版的 changelog（v1.0 → v1.3 只弹 v1.3）
- 首次安装：直接把 `LastChangelogVersion` 种为当前版本 → 不弹
- 缺 embed 文件：静默跳过

### 防御性测试
- `internal/changelog/changelog_test.go`：
  - `TestEmbeddedNotesCoverCurrentVersion` —— 读 `wails.json` 取版本号，若是纯三段数字格式（`X.Y.Z`），断言 `notes/vX.Y.Z.md` 存在
  - 带 `-dev`/`-rc`/非纯数字版本跳过

## Acceptance Criteria

- [ ] 从 v2.0.2 升级到 v2.0.3，重启后首次进入主应用弹出 v2.0.3 changelog；关闭后 `config.json` 中 `last_changelog_version = "2.0.3"`
- [ ] 同一版本下重启 App 不再弹窗
- [ ] 全新安装的用户（首次创建 `config.json`）不弹窗，`last_changelog_version` 直接 = 当前版本
- [ ] 切换 UI locale（中/英）时，已展示的弹窗按 locale 渲染对应段
- [ ] 当前版本在 embed 资源里无对应 changelog 文件时，不弹窗、不报错（容错跳过）；前端可补 ack 也可不 ack（不影响后续逻辑）
- [ ] `go test ./internal/changelog ./internal/config ./internal/app` 全通过
- [ ] `cd frontend && npm run build` 通过（含 marked / dompurify 类型）
- [ ] `internal/changelog/changelog_test.go::TestEmbeddedNotesCoverCurrentVersion` 在仓库当前 `wails.json` 版本下能找到对应 md 文件

## Definition of Done

- 单元测试：版本比较 + "是否需要展示"判定 + config 序列化新字段。
- 前端构建（vue-tsc + vite）green。
- `zh-CN.json` 增量更新；`en-US.json` 不动。
- 文档/CHANGELOG 例子放在仓库 `internal/changelog/` 下，README 或 CLAUDE.md 记录"发版要更新 changelog 文件"的提示。

## Out of Scope (explicit)

- 跨版本日志合并展示（v1.0 → v1.3 显示 v1.1+v1.2+v1.3 三段）。
- 从远端拉取 changelog（GitHub release API / 自有服务）。
- 用户主动"查看历史更新日志"入口（未来可加，本期不做）。
- 富媒体 changelog（图片、视频、可点击链接跳浏览器除外）。

## Research References

- [`research/changelog-modal-conventions.md`](research/changelog-modal-conventions.md) —— 7 个开源桌面 App 的实现对照；核心结论：(1) 业界主流是远程拉取，但同栈的 Wails 项目 SatisfactoryModManager 用 `marked + DOMPurify` 渲染 markdown 给我们提供了直接对标；(2) VS Code 的 `postUpdateWidget/lastKnownVersion` 是首次安装静默 + 只对 major/minor 跳变触发的教科书写法，PRD 应照搬此语义。

## Implementation Plan

- 单 PR 交付：后端包 + config 字段 + 两个 Wails 方法 + 前端模块 + i18n + 占位 changelog 文件 + 单测，一次性合入

## Decision (ADR-lite)

**Context**：发版后用户对自更新缺乏感知；想要一个"已经更新到 X 版本，本版本做了 Y / Z"的弹窗，且不打扰用户（同版本不重复弹）。

**Decision**：
- 内容：每版本一个双语 markdown 文件，`//go:embed` 进二进制
- 渲染：前端 `marked + dompurify`
- 展示策略：任意版本变化都弹（含 patch），首次安装静默，缺文件静默
- 持久化：`config.json` 新增 `last_changelog_version`

**Consequences**：
- 离线启动 / 国内网络抖动场景下弹窗仍然可靠（不依赖远端 release body）
- 发版流程多一步：每个 tag 必须配套一个 `notes/v{ver}.md`；单元测试在 CI 层兜底
- 前端首次引入 markdown 渲染依赖（marked + dompurify ~30KB），未来可复用于其他 in-app 富文本场景
- 不实现跨版本累积；后续如有需要可在 `GetPendingChangelog` 改造时扩展（返回 `[]Notes`）

## Technical Notes

- Embed 路径：`internal/changelog/notes/v<X.Y.Z>.md`
- 包暴露：
  ```go
  package changelog

  //go:embed notes/*.md
  var notesFS embed.FS

  type Notes struct { Version, BodyZh, BodyEn string }
  func Get(version string) (Notes, bool)  // 切分 "## 中文" / "## English"
  ```
- 切分实现：用 `regexp.MustCompile(`(?m)^## (中文|English)\s*$`)` 找出 header 位置后切片，trim 空白后赋值；任一段缺失时该字段为空字符串
- Config 兼容：`LoadOrCreate` 区分"config.json 不存在"（首装，种当前版本）vs "key 不存在但 json 存在"（升级到含本功能版本的老用户）。后者需要可以正常弹一次，所以不在 `LoadOrCreate` 默认填值；用一个 `ShouldShowChangelog(cfg, current)` 辅助函数判定
- 前端 markdown 渲染：参照 SatisfactoryModManager `frontend/src/lib/utils/markdown.ts` 的 marked + DOMPurify 组合
- 触发点：`AppShell.vue` 在路由首次稳定在主应用区（非 `/startup` / `/workspace-init`）后调一次 `GetPendingChangelog`，弹窗交互结束后调 `AckChangelog`
- 涉及代码改动：
  - `internal/changelog/`（新包）
  - `internal/config/config.go`（新增字段 + 兼容）
  - `internal/app/app.go`（暴露两个 Wails 方法）+ 可能拆一个 `app_changelog.go`
  - `frontend/src/features/changelog/`（新模块）
  - `frontend/src/app/AppShell.vue` 或路由钩子（挂载点）
  - `frontend/src/shared/i18n/zh-CN.json`（弹窗标题、关闭按钮文案）
  - `frontend/package.json`（新增 marked + dompurify）
  - 初始 `internal/changelog/notes/v2.0.2.md`（占位，避免首次构建测试挂）

