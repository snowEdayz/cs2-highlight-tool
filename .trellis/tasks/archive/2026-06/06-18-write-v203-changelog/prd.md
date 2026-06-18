# write v2.0.3 changelog

## Goal

根据本地 `main` 与 `origin/main` 的差异，提炼真实面向用户的新增功能，并补充到 `internal/changelog/notes/v2.0.3.md`。

## Requirements

* 先更新远端引用并比较 `origin/main..HEAD`。
* 排除 Trellis 任务归档、journal 等非产品功能提交。
* 将功能提交 `842620d feat: add shoulder camera setting` 转写成用户可读的中文与英文 changelog 项。
* 将用户手动维护的 `frontend/src/shared/i18n/en-US.json` 翻译改动纳入本任务提交。
* 不触碰自动生成文件。

## Acceptance Criteria

* [ ] 中文 `新增` 段包含越肩视角录制设置说明。
* [ ] English `Added` 段包含对应英文说明。
* [ ] `frontend/src/shared/i18n/en-US.json` 中用户提供的英文翻译随本任务提交。
* [ ] 文案与代码行为一致：设置默认关闭，开启后录制脚本写入越肩视角命令。
* [ ] 运行后端与前端相关 Required Checks。

## Definition of Done

* changelog 已更新。
* `go test ./...` 已执行并汇报结果。
* `cd frontend && npm run build` 已执行并汇报结果。

## Technical Approach

比较本地相对远端的提交，确认 `21fac80` 与 `40512bb` 仅为 Trellis 归档/日志记录；把 `842620d` 的 `use_shoulder_camera` 设置写入版本说明，并把用户手动补充的英文 i18n 文件一并提交。

## Out of Scope

* 不修改功能代码或生成绑定文件。
* 不由 AI 重写 `en-US.json` 内容，仅纳入用户已经手动维护的变更。

## Technical Notes

* 已读取 `AGENTS.md` 与 `internal/AGENTS.md`。
* 已读取 `.trellis/spec/backend/index.md` 与 shared thinking guide index。
