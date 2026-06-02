# Add outputs storage stats and cleanup

## Goal

Add an output directory management section to Settings so users can see how much disk space generated videos are using, open the managed outputs folder, and clear generated outputs when they need to reclaim space.

## What I Already Know

* The app now stores runtime data under the user data directory. The managed video output directory is `<dataDir>/outputs`.
* `config.RecordOutputDir` is normalized to the managed output directory by `App.fixedRecordOutputDir()`.
* Generated HLAE clips and edited videos both write under `RecordOutputDir`.
* Settings UI lives in `frontend/src/features/settings/components/SettingsPanel.vue`.
* Frontend backend calls go through `window.go.app.App.*`; generated Wails bindings under `frontend/wailsjs/**` must not be edited manually.
* i18n changes should be made only in `frontend/src/shared/i18n/zh-CN.json`; `en-US.json` is user-maintained.

## Confirmed Decisions

* The Settings page should display a lightweight snapshot, not a live filesystem watcher.
* Refresh should happen when Settings opens and after cleanup.
* Opening the folder should use a backend Wails method so Windows Explorer can open the real managed directory.
* Cleanup should not delete the `outputs` directory itself, only its contents.
* Cleanup should delete every child under `<dataDir>/outputs` because this directory is managed by the app and only contains generated clip folders and video files.
* Frontend must show a confirmation dialog before calling cleanup.

## Requirements

* Backend exposes a stable Settings-facing API to get output storage usage.
* Backend exposes an API to open the managed outputs folder.
* Backend exposes an API to clear every child under the managed outputs directory and return a refreshed usage snapshot.
* Usage snapshot includes at least:
  * output directory path
  * video file count
  * total size in bytes
* Frontend Settings page shows:
  * video count
  * formatted total size
  * output directory path
  * refresh/open folder/clear buttons
* Cleanup action requires a confirmation prompt before deletion.
* Errors should be shown in the existing Settings alert/message style.

## Acceptance Criteria

* [x] Settings displays current outputs count and total size.
* [x] Clicking open folder opens `<dataDir>/outputs` after ensuring it exists.
* [x] Clicking clear asks for confirmation, deletes the intended files, and refreshes the displayed stats.
* [x] Backend tests cover stats and cleanup behavior.
* [x] `go test ./...` passes.
* [x] `cd frontend && npm run build` passes.

## Definition of Done

* Tests added or updated where appropriate.
* Backend and frontend contracts stay in sync.
* Generated Wails files are not edited manually.
* AGENTS docs are updated if new Wails methods are treated as stable public interfaces.

## Out of Scope

* Moving `outputs` to a user-selectable custom location.
* Per-file deletion UI.
* Background disk usage monitoring.
* Cleaning demo files, projects, logs, updates, or component directories.

## Technical Notes

* Relevant backend files:
  * `internal/app/app.go`
  * `internal/app/clip_settings.go`
  * `internal/app/produce_takefile.go`
  * `internal/config/config.go`
  * `internal/appdata/paths.go`
* Relevant frontend files:
  * `frontend/src/features/settings/components/SettingsPanel.vue`
  * `frontend/src/shared/types/clips.ts`
  * `frontend/src/shared/i18n/zh-CN.json`
* Directory rules read:
  * `AGENTS.md`
  * `internal/AGENTS.md`
  * `frontend/AGENTS.md`
