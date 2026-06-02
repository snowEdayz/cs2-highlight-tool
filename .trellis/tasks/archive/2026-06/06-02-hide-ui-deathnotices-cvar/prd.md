# 隐藏所有 UI 时写入 deathnotices 参数

## Goal

在录制配置中新增默认关闭的“隐藏所有 UI”开关。开启后，生成插件 JSON 的启动序列写入 `cl_draw_only_deathnotices 1`；未开启时不写入任何 `cl_draw_only_deathnotices` 命令。

## What I Already Know

- 用户明确要求该参数和关闭 X 光、队内语音类似，属于录制参数配置。
- 现有 X 光配置字段为 `enable_spec_show_xray_zero`，队内语音配置字段为 `enable_voice`。
- 现有启动序列在 `internal/clipsjson/builder.go` 中集中写入 `spec_show_xray` 和 `tv_listen_voice_indices*`。
- 本需求只要求全局配置，不要求按单个击杀片段覆盖。
- 关闭状态必须“什么都不写”，不能输出 `cl_draw_only_deathnotices 0`。

## Requirements

- 新增配置字段 `hide_all_ui`，默认值为 `false`。
- `GetClipSettings` / `SaveClipSettings` 返回和保存该字段。
- 设置面板提供一个开关，文案为“隐藏所有 UI”。
- 生成插件 JSON 时：
  - `hide_all_ui=true`：bootstrap sequence 写入 `cl_draw_only_deathnotices 1`。
  - `hide_all_ui=false`：不写入任何 `cl_draw_only_deathnotices` 命令。
- 不修改 `frontend/wailsjs/**` 等自动生成文件。
- i18n 只更新 `zh-CN.json`。
- 同步更新根级和前端 AGENTS 中的稳定契约说明。

## Acceptance Criteria

- [x] 新配置默认关闭。
- [x] 保存开启后再次读取仍为开启。
- [x] 开启时生成 JSON 包含 `cl_draw_only_deathnotices 1`。
- [x] 关闭时生成 JSON 不包含 `cl_draw_only_deathnotices`。
- [x] 前端构建通过。
- [x] 后端测试通过。

## Out of Scope

- 不新增单个击杀片段级别的隐藏 UI override。
- 不在关闭时写入 `cl_draw_only_deathnotices 0`。
- 不修改英文翻译文件。

## Technical Notes

- 相关后端文件：
  - `internal/config/config.go`
  - `internal/app/clip_settings.go`
  - `internal/app/plugin_generate.go`
  - `internal/clipsjson/builder.go`
- 相关前端文件：
  - `frontend/src/shared/types/clips.ts`
  - `frontend/src/features/settings/components/SettingsPanel.vue`
  - `frontend/src/features/clips/pages/ClipsPage.vue`
  - `frontend/src/shared/i18n/zh-CN.json`
- 验证命令：
  - `go test ./...`
  - `cd frontend && npm run build`
