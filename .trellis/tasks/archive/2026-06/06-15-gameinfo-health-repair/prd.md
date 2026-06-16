# Gameinfo Health Repair

## Goal

Add a visible gameinfo health and repair flow so users can recover CS2 startup after this tool exits abnormally and leaves `Game\tcsgo/plugin` in `gameinfo.gi`. The normal produce flow should still inject the plugin search path only while recording, then restore it afterward.

## What I Already Know

* Current produce launch path calls `prepareGameInfoForProduce()` before launching HLAE and `forceRestoreProduceEnvironmentForProduce()` on failure, session end, and app shutdown.
* Current gameinfo injection inserts `Game\tcsgo/plugin` before the existing `Game\tcsgo` line.
* Current normal restore uses an in-memory session state plus a backup file named `gameinfo.gi.cs2ht_produce.bak`.
* If the process exits abnormally, the in-memory restore state is lost and `Game\tcsgo/plugin` can remain in `gameinfo.gi`.
* A stale plugin search path can prevent users from launching CS2 normally or through match platforms because the plugin is intended for insecure/HLAE recording mode only.
* The preferred product direction is a visible status bar wrench entry, not a settings-page-only repair action.

## Requirements

* Preserve the existing backup-based restore path for the normal produce flow.
* Add a backup-independent repair path that detects and removes stale `Game\tcsgo/plugin` lines from `gameinfo.gi`.
* Detect gameinfo health at app startup and expose the result to the frontend.
* Add a status bar wrench entry near the existing language/settings controls.
* Show a status dot beside the wrench:
  * green when `gameinfo.gi` is normal,
  * red when the plugin search path is present and needs repair,
  * gray when the status cannot be determined yet or CS2/gameinfo is not configured.
* When unhealthy on app startup, automatically open a small local popover anchored near the top-right wrench entry once per app launch.
* The unhealthy prompt must not be a global modal; it should be visually tied to the wrench area and dismissible by normal popover behavior.
* Repair should remove only the injected plugin search path line and save the file.
* After repair, refresh the displayed health state.

## Acceptance Criteria

* [ ] App can report gameinfo health without starting a produce session.
* [ ] App can repair a stale `Game\tcsgo/plugin` line even when no backup/session state exists.
* [ ] Existing produce injection still creates a backup and existing normal produce restore still uses it.
* [ ] Existing already-injected behavior remains idempotent during prepare.
* [ ] Startup UI shows a visible wrench health entry with green/red/gray state.
* [ ] Red state automatically opens a local wrench popover once per app launch, not a global modal.
* [ ] Red state provides a one-click repair action and clear error feedback on failure.
* [ ] Backend tests cover detecting healthy/unhealthy/missing gameinfo and repairing stale injected lines.
* [ ] Frontend build passes.
* [ ] Backend tests pass.

## Definition of Done

* Tests added/updated for backend repair helpers and app-layer methods.
* Frontend build passes with the new status bar entry.
* `go test ./...` passes.
* `cd frontend && npm run build` passes.
* AGENTS docs updated if new Wails methods or stable frontend/backend contracts are introduced.

## Technical Approach

Use a dual recovery model:

1. Normal produce flow keeps the existing backup file and session-state recovery for precise restoration.
2. Health/repair flow adds stateless detection and repair by reading `gameinfo.gi`, removing exact plugin search path lines, and saving the file.

Likely backend additions:

* Add helper functions under `internal/producegame`:
  * detect whether content contains the plugin search path as a standalone search path line.
  * remove standalone `Game\tcsgo/plugin` / `Game csgo/plugin` lines.
* Add Wails app-layer methods, names to be finalized, for:
  * reading current gameinfo health,
  * repairing gameinfo.

Likely frontend additions:

* Add a compact wrench button to the app top/status bar.
* Use a small color dot to show status.
* Show a popover/dropdown with current state, gameinfo path when available, repair action, and error text.

## Decision (ADR-lite)

**Context**: Backup-based recovery is exact but depends on process lifetime. Abnormal exit loses in-memory state and can leave `gameinfo.gi` modified.

**Decision**: Keep backup restore for normal produce sessions and add stateless health/repair as a crash recovery fallback.

**Consequences**: The normal flow remains precise, while users get a visible recovery path after crashes. The repair helper must be careful to remove only the injected search path line, not arbitrary text.

## Out of Scope

* Removing the existing backup mechanism.
* Automatically modifying `gameinfo.gi` without user confirmation.
* Adding a full settings page for this feature.
* Supporting arbitrary custom plugin search paths beyond this tool's `csgo/plugin` line.

## Open Questions

* None.

## Technical Notes

* Relevant files inspected:
  * `internal/producegame/gameinfo.go`
  * `internal/app/produce_gameconfig.go`
  * `internal/app/plugin_generate.go`
  * `internal/app/produce_session.go`
  * `internal/app/app.go`
  * `internal/envsetup/plugin.go`
  * `internal/config/config.go`
* Existing backup suffixes:
  * `gameinfo.gi.cs2ht_produce.bak`
  * `server.dll.cs2ht_plugin.bak`
* Existing required checks for this cross-layer task:
  * `go test ./...`
  * `cd frontend && npm run build`
