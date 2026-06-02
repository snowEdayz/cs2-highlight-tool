# Local Code Audit: record_quality setting

Date: 2026-06-02

## Goal

Verify the pasted implementation target against the current repository before turning it into implementation requirements.

## Findings

- `edit_quality` already exists as a persisted config field in `internal/config.Config`, with default `high` and supported values `standard|high|ultra`.
- `internal/ffmpegprofile.NormalizeEditQuality` already normalizes the same three values and falls back to `high`.
- `BuildEditEncodeArgs(profileID, quality)` is used for edit composition, not recording. Its C1 values are `standard=18`, `high=16`, `ultra=14`.
- Recording currently uses static `Profile.HLAEParams` from `profileCatalog`, resolved through `ffmpegprofile.HLAEProfileByID` via `internal/clipsjson.buildFFmpegParams`.
- Existing recording high values are:
  - `c1`: `-crf 4`
  - `n1`: `-qp 14`
  - `a1`: `-qp 12`
  - `i1`: `-q:v 12`
  - `n1_h264`: `-qp 16`
  - `a1_h264`: `-qp 14`
  - `i1_h264`: `-q:v 14`
- `auto` is a user-facing preset only. The actual recording profile passed into `clipsjson.Build` is resolved first by `plugingen.ResolvePluginVideoPreset(clipSettings.VideoPreset, cfg)`.
- `internal/clipsjson.BuildOptions` is the right boundary to carry recording quality into plugin JSON generation.
- `frontend/src/features/settings/components/SettingsPanel.vue` has a recording card with `record_fps`, `video_preset`, and `launch_resolution`; this is the expected place for the new control.
- `frontend/src/shared/i18n/zh-CN.json` should be updated, but `en-US.json` should not be changed per `frontend/AGENTS.md`.
- Existing dirty files under `frontend/wailsjs/**` are auto-generated and pre-existing; this task must not edit them manually.

## Corrections To The Pasted Prompt

- Do not blindly reuse all numeric values from `BuildEditEncodeArgs`; recording `c1` currently has a much lower CRF than edit composition. The recording quality function should preserve current static HLAE params when quality is `high`.
- A better name for the new function is still `BuildRecordingEncodeArgs(preset, quality)`, but its contract should be "return recording/HLAE ffmpeg params for the resolved profile and quality" rather than "mirror edit composition args exactly."

## Relevant Existing Tests

- `internal/ffmpegprofile/ffmpegprofile_test.go` covers profile resolution and edit encode args.
- `internal/clipsjson/builder_test.go` covers bootstrap ffmpeg command generation and invalid preset errors.
- `internal/app/app_clips_test.go` covers `GetClipSettings` / `SaveClipSettings` persistence and normalization.
- `internal/config/config_test.go` covers default application and config fallback behavior.
