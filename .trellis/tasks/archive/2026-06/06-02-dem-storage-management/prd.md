# 添加 Dem 目录存储管理

## Goal

在设置页面补齐 Demo 文件存储管理能力，让用户能像管理视频输出目录一样查看、打开并清理工作数据目录下的 DEM 文件目录。

## What I Already Know

* 当前设置页已有“输出目录”卡片，展示视频数量、总占用空间、目录位置，并提供刷新、打开文件夹、一键清理按钮。
* 现有后端接口为 `GetOutputsStorageStats`、`OpenOutputsDirectory`、`ClearOutputsDirectory`，实现位于 `internal/app/outputs_storage.go`。
* 现有统计结构 `OutputsStorageStats` 返回 `output_dir`、`video_count`、`total_size_bytes`。
* 现有 Demo 文件统一进入 `<dataDir>/demo` 下的子目录：手动导入为 `demo/raw`，完美为 `demo/wanmei/<matchID>`，5E 为 `demo/5e/<matchID>`。
* 根与子目录 `AGENTS.md` 均要求前后端契约变更同步文档，后端改动跑 `go test ./...`，前端改动跑 `cd frontend && npm run build`。

## Requirements

* 将设置页现有“输出目录”标题改为“视频输出目录”。
* 在设置页新增一个同风格的“Dem目录”卡片。
* Dem目录统计 `<dataDir>/demo` 下递归的 `.dem` 文件数量，扩展名大小写不敏感。
* Dem目录总占用空间统计 `<dataDir>/demo` 下递归所有文件的总字节数，与视频输出目录保持一致。
* Dem目录显示目录位置，并提供刷新、打开文件夹、一键清理按钮。
* 清理 Dem目录时删除 `<dataDir>/demo` 下所有直接子项，保留 `demo` 目录本身。
* 前端界面风格、按钮布局、错误提示与确认弹窗行为沿用视频输出目录。
* i18n 只更新 `zh-CN.json`；`en-US.json` 不由本任务维护。

## Acceptance Criteria

* [ ] 设置页原“输出目录”文案显示为“视频输出目录”。
* [ ] 设置页出现“Dem目录”卡片，包含 DEM 数量、占用空间、目录位置、刷新、打开文件夹、一键清理。
* [ ] 后端能递归统计 `<dataDir>/demo` 下 `.dem` 文件数量，并统计全部文件总大小。
* [ ] 清理 Dem目录后 `<dataDir>/demo` 保留为空目录，统计归零。
* [ ] 后端单元测试覆盖 DEM 统计和清理行为。
* [ ] `go test ./...` 通过。
* [ ] `cd frontend && npm run build` 通过。

## Out of Scope

* 不改变 Demo 导入、解析、列表选择逻辑。
* 不清理 `<dataDir>/projects`、`outputs`、`temp` 等其他目录。
* 不更新 `en-US.json`。
* 不手工修改 `frontend/wailsjs/**` 自动生成文件。

## Technical Notes

* 预计新增 Wails 暴露方法：`GetDemoStorageStats`、`OpenDemoDirectory`、`ClearDemoDirectory`。
* 预计新增统计结构可与输出目录结构类似，但前端字段应表达 `demo_count`。
* 需要更新根 `AGENTS.md` 稳定契约，记录新增 Wails 方法及返回语义。
* 需要更新 `frontend/src/shared/types/clips.ts` 或共享类型入口中的存储统计类型。
* 需要让 `SettingsPanel.vue` 同时加载输出目录与 Dem目录统计。
