# workspace-validate: 修复错误原因无法传至前端

## 根本原因

`ValidateWorkspaceDir(path string) (bool, string)` 在 Wails v2 中生成绑定类型为
`Promise<boolean|string>`（联合类型），运行时只传回 bool，string 丢失。
前端 errMsg 永远为空，触发兜底 "路径无效"，用户看不到具体原因。

## 修复方案

将返回类型改为单一 struct：

```go
type WorkspaceValidateResult struct {
    OK           bool   `json:"ok"`
    ErrorMessage string `json:"errorMessage"`
}
func (a *App) ValidateWorkspaceDir(path string) WorkspaceValidateResult
```

前端 `useWorkspaceInit.ts` 已有 object 分支读取 `obj.ok` / `obj.errorMessage`，
改完后端后无需改前端逻辑。

## Acceptance Criteria

* [ ] 选含空格路径 → 红色文字显示"路径不能包含空格"
* [ ] 选含中文路径 → 显示"路径不能包含中文或非 ASCII 字符"
* [ ] 选根目录 → 显示"路径不能是磁盘根目录"
* [ ] 合法路径 → 无错误，确认按钮可用
* [ ] `go test ./internal/app/...` 通过

## Out of Scope

* 进一步细化"哪个字符出问题"（后续任务）
