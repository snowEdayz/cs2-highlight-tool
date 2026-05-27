# 优化 internal/app 架构以提升可维护性

## Goal

在不改变现有对前端暴露行为（Wails 方法签名/事件契约）的前提下，优化后端 `internal/app` 的结构，降低耦合与单文件复杂度，让后续功能迭代与测试维护更容易。

## What I already know

- 用户反馈：后端 `internal/app` 目录文件较多，希望优化架构以利于维护。
- 当前 `internal/app` 共有 **31 个 Go 文件**，约 **7657 行**。
- 大文件集中在：`plugin_generate.go`（722 行）、`app_edit.go`（592 行）、`produce_takefile.go`（436 行）、`produce_gameconfig.go`（347 行）、`produce_merge.go`（332 行）等。
- `.trellis/spec/backend/directory-structure.md` 约定：`internal/app` 应是 Wails Binding Layer（前端可调用边界），倾向“薄层委托”，业务逻辑下沉到下层包。
- 根 `AGENTS.md` / `internal/AGENTS.md` 要求保持稳定契约：Wails 暴露方法、关键事件名、状态枚举语义不可随意破坏。

## Assumptions (temporary)

- 本次任务优先做“结构与职责优化”，而非功能新增。
- 前端调用方式与返回字段语义保持兼容。
- 可以通过新增/拆分 `internal` 下服务包来承接 `app` 中过重逻辑。

## Open Questions

- 暂无（已完成当前规划问题收敛）。

## Requirements (evolving)

- 在不破坏 Wails 公共接口契约的前提下优化 `internal/app` 可维护性。
- 明确 `internal/app` 与下层业务包职责边界，减少 `app` 层业务逻辑堆积。
- 保持现有流程行为一致（尤其是 startup / produce / edit / import）。
- 已确认采用**中度重构**：将 `internal/app` 中重业务逻辑下沉到 `internal/*` 业务服务包，`app` 保持 Wails 暴露边界与薄封装。
- 已确认首批覆盖范围为 **`produce + plugin + edit`**，不做全量一次性重构。
- 已确认迁移节奏为 **三阶段落地**（避免一次性大改风险）。
- 已确认每个阶段都必须可独立通过 `go test ./...`，并保持可发布状态。
## Acceptance Criteria (evolving)

- [ ] `internal/app` 结构较当前更清晰，职责分层可解释（文档化）。
- [ ] Wails 暴露方法名、关键事件名、返回字段语义保持兼容。
- [ ] 三阶段每一阶段提交点都可独立通过 `go test ./...`。
- [ ] 最终结果 `go test ./...` 通过。
- [ ] 如涉及架构/契约变化，更新 AGENTS 相关文档。

## Definition of Done (team quality bar)

- Tests added/updated (unit/integration where appropriate)
- Lint / typecheck / CI green
- Docs/notes updated if behavior changes
- Rollout/rollback considered if risky

## Out of Scope (explicit)

- 不做前端 UI 交互逻辑改版。
- 不在本任务中引入与需求无关的新功能。
- 本次不处理 `import` 域架构下沉（wanmei/fivee/demo 保持现状）。
## Technical Notes

- 已阅读：
  - `.trellis/spec/backend/index.md`
  - `.trellis/spec/backend/directory-structure.md`
  - `docs/ARCHITECTURE.md`
  - `internal/AGENTS.md`
- 现状快速扫描：`internal/app` 中同时存在薄封装方法与较重业务编排逻辑，具备进一步分层优化空间。
- 体量分布（非测试代码）：`produce+plugin` 约 2964 行，`edit` 约 956 行，`import` 约 318 行，适合先围绕高复杂域做首批下沉。
