# 支持从 5E 分享链接提取玩家 ID

## Goal

让 5E DEM 导入页支持用户直接粘贴 5E 客户端个人主页分享链接，自动从链接查询参数 `domain` 中提取 5E 查询 ID，例如从 `https://csgo.5eplay.com/app/share_loding_type7?domain=12139xi22eza&tab=77&uuid=...` 提取 `12139xi22eza`，再用该 ID 查询最近战绩，降低用户手动复制 ID 的成本。

## What I Already Know

- 当前 5E 导入页通过 `frontend/src/features/import/composables/useFiveEImport.ts` 将输入框内容 trim 后调用 `ListFiveERecentMatches(playerName, 1)`。
- 后端 `internal/app/app_fivee.go` 会先持久化输入到 `config.json` 的 `fivee_player_name`，再调用 `fivee.ListRecentMatches`。
- 5E 最近战绩接口在 `internal/fivee/client.go` 中使用 `domain` 查询参数，因此分享链接里的 `domain` 正是需要的查询 ID。
- 根 `AGENTS.md` 将 `ListFiveERecentMatches(playerName, page)` 和 `fivee_player_name` 记为稳定契约；输入语义扩展后需要同步更新文档。
- 前端 i18n 规则要求只修改 `zh-CN.json`，不修改 `en-US.json`。

## Requirements

- `ListFiveERecentMatches(playerName, page)` 必须接受原始 5E domain ID。
- `ListFiveERecentMatches(playerName, page)` 必须接受包含 `domain=<id>` 查询参数的 5E 分享链接或包含该链接的分享文案。
- 后端必须在 Wails 边界统一规范化输入，并将规范化后的 domain ID 持久化到 `fivee_player_name`。
- 空输入仍保持当前行为：跳过远端调用并返回空列表。
- 非链接普通输入仍按原输入 trim 后查询，保持兼容。
- 5E 导入页中文提示与占位文案应说明支持粘贴分享链接。
- 不新增或重命名 Wails 方法、事件名、状态枚举，不修改自动生成文件。

## Acceptance Criteria

- [ ] 输入 `12139xi22eza` 时，请求 5E 战绩接口的 `domain` 参数为 `12139xi22eza`。
- [ ] 输入完整分享文本 `【5E对战平台：...】https://csgo.5eplay.com/app/share_loding_type7?domain=12139xi22eza&tab=77&uuid=...` 时，请求 5E 战绩接口的 `domain` 参数为 `12139xi22eza`。
- [ ] 分享链接输入成功查询后，`config.json` 的 `fivee_player_name` 保存为 `12139xi22eza`。
- [ ] 空输入不会发起远端请求。
- [ ] `go test ./...` 通过。
- [ ] `cd frontend && npm run build` 通过。

## Definition of Done

- Tests added/updated for link extraction and app-layer persistence/query behavior.
- Required backend and frontend checks pass.
- Stable contract documentation updated where behavior changed.

## Technical Approach

- 在 `internal/fivee` 添加公开规范化 helper，用于从原始输入中提取 `domain` 查询参数；找不到 `domain` 时返回 trim 后的输入，保持兼容。
- 在 `internal/app.App.ListFiveERecentMatches` 调用该 helper 后再持久化和查询，确保前端初始加载、刷新、分页都复用同一规则。
- 更新 5E 导入页中文 i18n 文案，使用户知道可粘贴分享链接。

## Decision (ADR-lite)

**Context**: 分享链接解析可放在前端或后端。前端解析能即时改变输入框，但分页、初始缓存读取和任何未来调用者仍可能绕过该逻辑。

**Decision**: 在后端 Wails 边界规范化输入，前端仅更新提示文案。

**Consequences**: 单一入口更稳，公开方法签名不变；`fivee_player_name` 仍沿用旧字段名，但其持久化值会变成可查询的 5E domain ID。

## Out of Scope

- 不自动从 5E 客户端进程或本地文件读取账号信息。
- 不新增独立“粘贴链接解析”按钮。
- 不修改英文翻译文件。
- 不调整 5E match ID 批量下载输入逻辑。

## Technical Notes

- Relevant files inspected:
  - `internal/app/app_fivee.go`
  - `internal/app/app_fivee_test.go`
  - `internal/fivee/client.go`
  - `internal/fivee/client_test.go`
  - `frontend/src/features/import/composables/useFiveEImport.ts`
  - `frontend/src/features/import/pages/FiveEImport.vue`
  - `frontend/src/shared/i18n/zh-CN.json`
- Relevant rules read:
  - `AGENTS.md`
  - `internal/AGENTS.md`
  - `frontend/AGENTS.md`
  - `.trellis/spec/backend/index.md`
  - `.trellis/spec/backend/wails-bindings.md`
  - `.trellis/spec/backend/error-handling.md`
  - `.trellis/spec/backend/quality-guidelines.md`
  - `.trellis/spec/frontend/i18n-guidelines.md`
  - `.trellis/spec/frontend/quality-guidelines.md`
  - `.trellis/spec/guides/cross-layer-thinking-guide.md`
