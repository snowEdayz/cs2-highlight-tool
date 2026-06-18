# Add Shoulder Camera Setting

## Goal

Add a settings-page toggle for using the over-shoulder camera view. When enabled, generated plugin JSON bootstrap actions must include the requested camera command before `r_show_build_info 0`.

## Requirements

* Add a persisted `use_shoulder_camera` boolean setting to clip settings/config, defaulting to `false`.
* Expose the setting through existing `GetClipSettings` / `SaveClipSettings` flows without adding new Wails methods.
* Add a settings-page switch labeled `使用越肩视角`.
* When `use_shoulder_camera=true`, generated bootstrap actions must append this exact command before `r_show_build_info 0`:
  `cam_command 1;cam_idealdist 30;cam_idealyaw 0;cam_idealpitch 0;c_thirdpersonshoulder 1;c_thirdpersonshoulderaimdist 300;c_thirdpersonshoulderdist 40;c_thirdpersonshoulderheight 2;c_thirdpersonshoulderoffset 20;`
* When disabled, do not emit any `cam_command` / `c_thirdpersonshoulder` bootstrap command.
* Keep generated files under `frontend/wailsjs/**` untouched.
* Update Chinese i18n only; `en-US.json` is intentionally out of scope.
* Update AGENTS stable contract docs because `GetClipSettings` / `SaveClipSettings` fields and `config.json` fields change.

## Acceptance Criteria

* [x] `GetClipSettings` returns `use_shoulder_camera=false` for default/legacy config.
* [x] `SaveClipSettings` persists and reloads `use_shoulder_camera=true`.
* [x] `config.LoadOrCreate` backfills/saves `use_shoulder_camera` for legacy configs.
* [x] `clipsjson.Build` emits the over-shoulder camera command before `r_show_build_info 0` only when enabled.
* [x] Settings UI shows a switch for `使用越肩视角` and auto-saves through the existing settings flow.
* [x] `go test ./...` passes.
* [x] `cd frontend && npm run build` passes.

## Definition of Done

* Backend and frontend code updated consistently.
* Focused tests added/updated for persistence and bootstrap action generation.
* Required checks run and reported.
* No manual edits to generated Wails files.

## Technical Approach

Use the existing `hide_all_ui` pattern:

* Add `UseShoulderCamera` / `use_shoulder_camera` to `config.Config`, `app.ClipSettings`, and `clipsjson.BuildOptions`.
* Pass the setting from `GeneratePluginJSON` into `clipsjson.Build`.
* Insert the command at the start of `buildBootstrapSequence` when enabled so it appears before `r_show_build_info 0`.
* Add the frontend field to `ClipSettings`, the reactive defaults, the settings panel switch, and `zh-CN.json`.

## Decision (ADR-lite)

**Context**: This is a persisted user preference that affects every generated plugin JSON bootstrap sequence.

**Decision**: Extend the existing clip settings contract rather than adding a per-request flag or a new API.

**Consequences**: The option is simple and consistent with current settings behavior. It changes a stable settings/config contract, so docs and tests must be updated.

## Out of Scope

* Per-clip or per-batch shoulder camera overrides.
* Customizing individual camera command values in the UI.
* Updating `frontend/shared/i18n/en-US.json`.
* Regenerating Wails bindings.

## Technical Notes

* Root `AGENTS.md`, `internal/AGENTS.md`, and `frontend/AGENTS.md` require generated files not be manually edited.
* Existing related fields: `hide_all_ui`, `sky_blackout`, `block_kill_feed`.
* Main files expected to change:
  * `internal/config/config.go`
  * `internal/config/config_test.go`
  * `internal/app/clip_settings.go`
  * `internal/app/plugin_generate.go`
  * `internal/app/app_clips_test.go`
  * `internal/clipsjson/builder.go`
  * `internal/clipsjson/builder_test.go`
  * `frontend/src/shared/types/clips.ts`
  * `frontend/src/features/settings/components/SettingsPanel.vue`
  * `frontend/src/shared/i18n/zh-CN.json`
  * `AGENTS.md`
  * `frontend/AGENTS.md`
