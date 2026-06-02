# workspace-init: auto-append app subdirectory

## Goal

当用户在工作目录初始化弹窗中选择一个父目录（如 `D:\ProgramData`）时，
自动拼接固定子目录名（如 `cs2HighLightTool`），使最终数据目录变为
`D:\ProgramData\cs2HighLightTool`，而不是直接使用用户选择的目录。
目标：更规范的目录结构，降低用户误操作风险。

## What I already know

* 当前流程：`PickWorkspaceDir`（后端打开对话框） → 前端 `ValidateWorkspaceDir` → `SetWorkspaceDir`
* `PickWorkspaceDir` 仅在 `useWorkspaceInit.ts` 中调用，无其他消费方
* 前端路径全程 readonly，用户无法手动输入
* Modal 仅显示一个路径输入框 + 一条 error 文字
* 后端 `filepath.Join` 负责拼路径分隔符（Windows `\`）
* i18n 描述文字 `workspace.init.description` 提到"当前为空"约束（已移除），需更新措辞
* 子目录名用户说为 `cs2HighLightTool`

## Decision

**UI 展示方式**：只显示最终完整路径（方案 A），描述文字说明会自动创建子目录。

## Requirements

* 用户在目录对话框选择父目录后，后端自动 `filepath.Join(selected, "cs2HighLightTool")` 得到最终路径
* 幂等：若路径末段已是 `cs2HighLightTool`（`filepath.Base` 判断），不重复拼接
* Modal 只显示最终路径（readonly input），描述文字改为"选择一个父目录，程序将在其中自动创建 cs2HighLightTool 子目录"
* 更新 `zh-CN.json` 的 `workspace.init.description`；同步删除 `workspace.validate.invalid_nonempty`（已无该校验规则）及 `invalid_length` 提示中的"100"→"200"

## Acceptance Criteria

* [ ] 选 `D:\ProgramData` → input 显示 `D:\ProgramData\cs2HighLightTool`
* [ ] 选 `D:\ProgramData\cs2HighLightTool` → 不重复拼接，仍显示 `D:\ProgramData\cs2HighLightTool`
* [ ] 最终路径通过字符/长度/可写校验后可提交
* [ ] Modal 描述文字清晰反映"选父目录"语义
* [ ] `go test ./internal/...` 通过，前端 `npm run build` 无 TS 错误

## Definition of Done

* 改动通过 `go test ./internal/...` 和前端 `npm run build`
* 前端无 TypeScript 错误

## Out of Scope

* 允许用户自定义子目录名
* 手动输入路径（仍保持只读）

## Technical Notes

* 修改 `internal/app/app_workspace.go: PickWorkspaceDir` — 返回前 `filepath.Join(selected, "cs2HighLightTool")`，加幂等判断
* 或新增后端方法 `AppendAppSubdir(parent) string` 供前端调用（更干净但多一个 Wails binding）
* 前端 `useWorkspaceInit.ts: pick()` 无需改动（如拼接在后端做）
* `zh-CN.json` 描述文字需更新
