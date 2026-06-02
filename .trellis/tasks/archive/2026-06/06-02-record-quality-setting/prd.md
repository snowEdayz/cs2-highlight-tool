# 新增录制质量配置

## Goal

在工具设置的录制配置区域新增 `录制质量` 下拉选项，让用户用 `标准 / 高 / 极高` 调整录制阶段 HLAE ffmpeg 参数中的质量值。默认值必须保持为 `高`，并且 `高` 生成的 plugin JSON 录制参数必须与当前版本一致，避免升级后改变既有录制画质、码率和文件体积表现。

## What I Already Know

- 这是一个跨层变更：配置持久化 -> Wails `ClipSettings` -> 前端设置面板 -> plugin JSON 构建 -> HLAE recording ffmpeg params。
- 质量枚举可复用现有 `standard|high|ultra` 和 `ffmpegprofile.NormalizeEditQuality`。
- 当前 `edit_quality` 仅作用于剪辑合成阶段的 `BuildEditEncodeArgs`。
- 当前录制阶段使用 `profileCatalog` 中静态 `HLAEParams`，通过 `clipsjson.buildFFmpegParams(opts.VideoPreset)` 写入 `mirv_streams settings add ffmpeg ...`。
- 前端只维护 `zh-CN.json`；`en-US.json` 由用户自行维护。
- 根级 `AGENTS.md` 的稳定契约需要在实现时补充 `record_quality` 字段约定。
- `frontend/wailsjs/**` 是自动生成文件，当前已有未提交变更，本任务不得手工修改。

## Research References

- [`research/local-code-audit.md`](research/local-code-audit.md) — 核对贴文与当前代码，确认录制侧静态参数、默认值和受影响边界。

## Requirements

- 新增持久化配置字段 `record_quality`，取值 `standard|high|ultra`，默认 `high`。
- `GetClipSettings` / `SaveClipSettings` 暴露并保存 `record_quality`。
- `SaveClipSettings` 对无效 `record_quality` 回退到默认 `high`；大小写和空白应按现有质量字段方式归一化。
- 前端 `ClipSettings` 类型新增 `record_quality` 字段。
- 设置面板的 `录制配置` 卡片新增 `录制质量` 下拉，选项复用 `标准 / 高 / 极高` 文案。
- `GeneratePluginJSON` / material generation / batch launch 共享的 plugin JSON 构建路径必须传递 `record_quality`。
- `clipsjson.BuildOptions` 新增 `RecordQuality`，并用它生成录制阶段 ffmpeg 参数。
- `ffmpegprofile` 新增录制参数构建能力，输入 resolved profile ID 和 quality，返回 HLAE 参数字符串。
- `record_quality=high` 必须生成与当前静态 `HLAEParams` 完全等价的录制参数。
- `record_quality=standard|ultra` 只调整质量相关参数，不改变编码器、像素格式、GOP、preset、voice/xray、record fps、输出目录等非质量行为。
- `record_quality` 必须同时覆盖软件编码和硬件编码：软件编码 `c1/libx264` 调整 `crf`，硬件编码 `n1/a1/i1` 及 H264 fallback 调整各自的 `qp` 或 `q:v`。
- 实现不得只覆盖 CPU/software preset；NVENC、AMF、QSV 的 HEVC preset 与 H264 fallback 都必须有明确映射和测试断言。

## Proposed Quality Mapping

This mapping is anchored on current recording defaults, not edit-composition C1 values.

| profile | parameter | standard | high | ultra |
| --- | --- | ---: | ---: | ---: |
| `c1` | `crf` | 10 | 4 | 2 |
| `n1` | `qp` | 20 | 14 | 10 |
| `a1` | `qp` | 20 | 12 | 8 |
| `i1` | `q:v` | 20 | 12 | 8 |
| `n1_h264` | `qp` | 22 | 16 | 12 |
| `a1_h264` | `qp` | 22 | 14 | 10 |
| `i1_h264` | `q:v` | 22 | 14 | 10 |

## Data Flow

1. User changes settings in `SettingsPanel.vue`.
2. Frontend calls `SaveClipSettings` with `record_quality`.
3. Backend persists `config.json.record_quality`.
4. Plugin generation loads normalized `ClipSettings`.
5. `plugin_generate.go` passes `RecordQuality` into `clipsjson.BuildOptions`.
6. `clipsjson.Build` calls `ffmpegprofile.BuildRecordingEncodeArgs(resolvedPreset, recordQuality)`.
7. Generated plugin JSON bootstrap includes `mirv_streams settings add ffmpeg <preset> <quality-adjusted params>`.

## Acceptance Criteria

- [ ] 设置面板录制配置区域显示 `录制质量` 下拉。
- [ ] 下拉选项为 `标准 / 高 / 极高`。
- [ ] 保存 `极高` 后重新加载设置仍保持 `极高`。
- [ ] `SaveClipSettings` 接收 `standard|high|ultra`，无效值回退 `high`。
- [ ] `record_quality=high` 生成的 HLAE ffmpeg 参数与当前静态参数一致。
- [ ] `record_quality=ultra` 对 `n1` 生成 `-qp 10`，而不是当前 `-qp 14`。
- [ ] `record_quality=standard` 对 `n1` 生成 `-qp 20`。
- [ ] `c1` 使用录制侧 CRF 映射 `10/4/2`，不误用剪辑合成 `18/16/14`。
- [ ] 硬件加速 preset `n1/a1/i1` 分别按质量生成预期 `qp` / `q:v`。
- [ ] 硬件 H264 fallback preset `n1_h264/a1_h264/i1_h264` 分别按质量生成预期 `qp` / `q:v`。
- [ ] `auto` preset 仍先解析到具体 profile 后再套用录制质量。
- [ ] 不手工修改 `frontend/wailsjs/**`、`frontend/src/auto-imports.d.ts`、`frontend/src/components.d.ts`。

## Tests Required

- `internal/ffmpegprofile/ffmpegprofile_test.go`: 覆盖每个 recording profile 与 `standard/high/ultra` 的质量参数。
- `internal/clipsjson/builder_test.go`: 覆盖 `BuildOptions.RecordQuality` 会改变 bootstrap ffmpeg command，并保持 invalid preset error。
- `internal/config/config_test.go`: 覆盖 `record_quality` 默认值、有效值保留、无效值回退。
- `internal/app/app_clips_test.go`: 覆盖 `GetClipSettings` / `SaveClipSettings` 对 `RecordQuality` 的持久化与回退。
- Required checks:
  - `go test ./...`
  - `cd frontend && npm run build`

## Out Of Scope

- 不新增更多质量档位或自定义数字输入。
- 不修改 `edit_quality` 的现有行为。
- 不修改 FFmpeg 能力探测、preset 自动选择、fallback 顺序。
- 不修改录制 FPS、启动分辨率、输出目录或素材命名规则。
- 不生成或手工编辑 Wails 自动绑定文件。

## Open Questions

- 已确认：同一个 `录制质量` UI 选项需要同时适配软件编码与硬件编码。`c1/libx264` 使用 CRF 映射，`n1/a1/i1` 及 H264 fallback 使用 QP / `q:v` 映射。
