# 增加工具设置启动分辨率 1280x960

## Goal

在工具设置页面的“启动分辨率”下拉选项中新增 4:3 的 `1280x960` 启动分辨率，让用户可以在现有 `16:9` 与 `4:3 (1440x1080)` 之外选择更低的 4:3 分辨率启动 CS2。

## What I Already Know

* 用户明确要求：工具设置页面当前启动分辨率只支持两种，需要新增 `4:3 1280x960`。
* 现有前端设置页在 `frontend/src/features/settings/components/SettingsPanel.vue` 中定义 `resolutionOptions`。
* 现有前端类型 `frontend/src/shared/types/clips.ts` 将 `launch_resolution` 限定为 `"16:9" | "4:3"`。
* 现有中文文案位于 `frontend/src/shared/i18n/zh-CN.json`；根据项目规则，本次 i18n 只维护 `zh-CN.json`，不主动修改 `en-US.json`。
* 后端 `internal/config/config.go` 和 `internal/app/clip_settings.go` 会校验 `launch_resolution`，不支持的值会回退到默认 `4:3`。
* HLAE 启动命令由 `internal/app/hlae_launch.go` 的 `buildHLAECommandLine` 生成；现有 `"4:3"` 会追加 `-w 1440 -h 1080`。
* 当前工作区已有未提交的自动生成文件变更：`frontend/wailsjs/go/app/App.d.ts`、`frontend/wailsjs/go/app/App.js`、`frontend/wailsjs/go/models.ts`。本任务不得手工修改这些文件。

## Requirements

* 保留现有两个启动分辨率选项：
  * `16:9`
  * `4:3 (1440x1080)`
* 新增一个独立选项：中文文案使用 `4:3 (1280x960)`。
* 新选项保存后不得被前端类型或后端校验逻辑回退。
* 使用新选项启动 HLAE 时，命令行应包含 `-w 1280 -h 960`。
* 现有 `"4:3"` 配置值继续代表 `1440x1080`，避免改变已有用户配置语义。
* 若本次扩展 `GetClipSettings` / `SaveClipSettings` 的 `launch_resolution` 允许值，应同步更新根级 `AGENTS.md` 的稳定契约说明。

## Acceptance Criteria

* [ ] 工具设置页面“启动分辨率”下拉中显示第三个选项，中文文案为 `4:3 (1280x960)`。
* [ ] 保存 `4:3 (1280x960)` 后，重新加载设置仍保持该选择。
* [ ] `SaveClipSettings` 接收新增值，不回退到默认值。
* [ ] HLAE 启动命令对新增值输出 `-w 1280 -h 960`。
* [ ] 原有 `"4:3"` 仍输出 `-w 1440 -h 1080`。
* [ ] 原有 `"16:9"` 行为不变，不追加 4:3 宽高参数。
* [ ] 后端相关测试覆盖新增校验与命令生成行为。
* [ ] 前端构建通过。

## Definition of Done

* 前端和后端的 `launch_resolution` 允许值保持一致。
* 相关测试更新并通过。
* 执行 Required Checks：
  * `go test ./...`
  * `cd frontend && npm run build`
* 不手工修改 `frontend/wailsjs/**` 自动生成文件。
* 若公共契约文档需要更新，按 `AGENTS.md` 维护触发器同步更新。

## Technical Approach

建议新增一个显式持久化值，例如 `4:3_1280x960`，用于区分现有 `"4:3"` 的 `1440x1080` 语义。前端下拉中文文案展示为 `4:3 (1280x960)`，保存值为新增枚举；后端配置与设置归一化逻辑接受该值；HLAE 命令生成根据该值追加 `-w 1280 -h 960`。

## Decision (ADR-lite)

**Context**: 当前 `"4:3"` 已经在前后端和配置文件中代表 `1440x1080`，直接把它改成 `1280x960` 会破坏已有用户配置和测试语义。

**Decision**: 新增独立枚举值表示 `4:3 (1280x960)`，保留 `"4:3"` 表示 `4:3 (1440x1080)`。

**Consequences**: 需要同步更新前端类型、设置选项、后端校验、HLAE 命令生成和测试；好处是行为清晰且兼容已有配置。

## Out of Scope

* 不新增任意自定义分辨率输入框。
* 不改变默认启动分辨率。
* 不重命名现有 `"4:3"` 配置值。
* 不修改自动生成的 Wails 绑定文件。
* 不维护 `en-US.json`，该文件由用户自行维护。

## Technical Notes

* 已阅读根级 `AGENTS.md` 与 `frontend/AGENTS.md`。
* 相关文件候选：
  * `frontend/src/features/settings/components/SettingsPanel.vue`
  * `frontend/src/shared/types/clips.ts`
  * `frontend/src/shared/i18n/zh-CN.json`
  * `internal/config/config.go`
  * `internal/app/clip_settings.go`
  * `internal/app/hlae_launch.go`
  * `internal/config/config_test.go`
  * `internal/app/app_clips_test.go`
  * `AGENTS.md`
