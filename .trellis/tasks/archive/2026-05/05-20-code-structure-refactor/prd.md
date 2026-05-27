# 代码结构与架构优化 — 后端 app/ 层业务逻辑提取

## Goal

将 `internal/app/` 层（Wails 绑定层）中违反"thin wrapper"原则的业务逻辑提取到专用包，使代码结构与 spec 规定的架构意图对齐，提升可维护性和可测试性。

## Requirements

- [ ] 新建 `internal/wanmei/` 包：包含 Wanmei API 客户端、AES/SHA1 加密、HTTP 签名、响应解析、及所有相关类型（`WanmeiMatchItem`、`WanmeiMatchListResult` 等）
- [ ] 新建 `internal/fivee/` 包：包含 5E API 客户端、HTTP 调用、gzip 解析、响应解析、及所有相关类型（`FiveEMatchItem`、`FiveEMatchListResult` 等）
- [ ] 新建 `internal/plugingen/` 包：包含插件 JSON 生成编排逻辑（normalizeSelectedItems、buildProduceHistoryKey、filterItemsByHistory 等），依赖 `internal/clipsjson/`
- [ ] `internal/app/app_wanmei.go`：重构为薄壳（仅参数传递 + 调用 `wanmei.*` 函数）
- [ ] `internal/app/app_fivee.go`：重构为薄壳（仅参数传递 + 调用 `fivee.*` 函数）
- [ ] `internal/app/plugin_generate.go`：重构为薄壳（仅参数传递 + 调用 `plugingen.*` 函数）
- [ ] 已有测试文件（`app_wanmei_test.go`、`app_fivee_test.go`）随逻辑迁移到新包
- [ ] Wails 公共绑定方法名称不变（`ListWanmeiRecentMatches`、`ImportWanmeiMatch`、`ListFiveERecentMatches` 等）

## Acceptance Criteria

- [ ] `go test ./...` 全部通过（包括新包的测试）
- [ ] `cd frontend && npm run build` 通过（类型兼容）
- [ ] `internal/app/` 中无直接的 HTTP 客户端（`net/http` 请求）、加密算法（`crypto/*`）、复杂解析逻辑
- [ ] 新包各自有独立测试覆盖（从旧位置迁移）
- [ ] 新包暴露包级函数（不引入 service struct DI 模式，与 `clipsjson`、`demo` 等包风格一致）

## Definition of Done

- `go test ./...` 通过
- `cd frontend && npm run build` 通过
- `internal/app/` 每个重构文件移除所有加密/HTTP/解析导入
- `.trellis/spec/backend/directory-structure.md` 更新，加入三个新包描述

## Technical Approach

提取策略：包级函数直调（不引入 DI/service struct），与现有 `clipsjson`、`demo`、`download` 包风格一致。

```
app/ (thin binding layer)
  ↓ calls package-level functions
internal/wanmei/      — Wanmei API client + types
internal/fivee/       — 5E API client + types
internal/plugingen/   — plugin JSON generation orchestration
  ↓ depends on
internal/clipsjson/   — low-level Action/Sequence primitives (unchanged)
```

HTTP transport 注入（testability）：新包暴露可选的 transport 参数或 functional option，保留现有测试能力。

## Decision (ADR-lite)

**Context**: `internal/app` 违反 spec 要求的 thin wrapper 原则，含大量 HTTP/加密/解析业务逻辑。

**Decisions**:
1. 后端优先，全覆盖三个违规区域
2. 类型随业务逻辑迁移到新包（`app/` 返回新包类型，Wails 反射自动处理）
3. 新建 `internal/plugingen/` 包承载高层编排逻辑（不合并入 `clipsjson/`）
4. 包级函数调用风格（不引入 service struct，保持与现有包一致）

**Consequences**: 一次性还清技术债；Wails 生成的 TS 类型路径变化但接口名不变；需迁移已有测试。

## Out of Scope

- 前端 composable/component 拆分（留后续任务）
- `MatchSource` 接口抽象（可扩展性留到实际需要时）
- `app_edit.go`、`produce_*` 等其他文件（此次不动）
- 新增任何功能或 UI 变更

## Implementation Plan

- **Step 1**: 新建 `internal/wanmei/`，迁移 `app_wanmei.go` 业务逻辑 + 类型 + 测试；`app_wanmei.go` 变薄壳
- **Step 2**: 新建 `internal/fivee/`，迁移 `app_fivee.go` 业务逻辑 + 类型 + 测试；`app_fivee.go` 变薄壳
- **Step 3**: 新建 `internal/plugingen/`，迁移 `plugin_generate.go` 编排逻辑 + 类型；`plugin_generate.go` 变薄壳
- **Step 4**: 全量 `go test ./...` + 前端构建验证 + 更新 spec 文档

## Technical Notes

- 绑定方法不可重命名（Wails 稳定公共 API 约束）
- `app_wanmei.go` 中 `defaultWanmeiHTTPRequest`/`defaultWanmeiOSSResolveHTTPDo` 是 transport 注入点，迁移时保留可测试性
- `wailsjs/` 自动生成，迁移后运行 `wails dev` 或 `wails build` 重新生成
- 现有测试（`app_wanmei_test.go`、`app_fivee_test.go`）在 `package app` 下，迁移后改为新包的 `package wanmei` / `package fivee`
