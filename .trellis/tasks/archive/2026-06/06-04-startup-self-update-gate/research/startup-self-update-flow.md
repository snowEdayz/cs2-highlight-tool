# Startup Self-Update Flow Research

## Relevant Contracts

* Root `AGENTS.md` says startup prepares environment through unified update source check, component checks/installs, and self-update detection before entering main.
* `internal/AGENTS.md` requires stable component IDs (`hlae`, `plugin`, `ffmpeg`, `cs2`) and status/phase semantics.
* `frontend/AGENTS.md` requires startup state to come from `GetStartupState` plus events, and button availability to stay consistent with `running`, `self_update`, and `can_enter_main`.

## Current Backend Flow

* `internal/envsetup/service_startup.go:16` is the main `RunStartupChecks()` entry point.
* `RunStartupChecks()` prepares directories, loads config, resolves source, fetches the unified Release snapshot, sets `phase=running_tasks`, then calls `runTasksDefault`.
* `runTasksDefault` sets `SelfUpdate.Status=checking`, then starts `checkSelfUpdate` in one goroutine and all component jobs in separate goroutines.
* The component jobs are HLAE, plugin, FFmpeg, and CS2 path checks.
* `runTasksDefault` waits for all goroutines before clearing startup running state through the caller's defer.

## Current Self-Update Flow

* `internal/envsetup/selfupdate.go:15` checks software updates from the unified snapshot.
* On release lookup failure, it sets `SelfUpdate.Status=failed`, logs that the startup flow will continue, and returns.
* On up-to-date, it sets `SelfUpdate.Status=ready`.
* On newer version, it sets `SelfUpdate.Status=needs_action`, `Available=true`, `Latest`, `URL`, and `AssetURL`, and sets `CanEnterMain=false`.
* `internal/envsetup/service_state.go:238` already prevents entering main when `SelfUpdate.Available` is true.

## Current Frontend Flow

* `frontend/src/features/startup/composables/useStartupWizard.ts:42` computes `busy` from `state.running` or self-update downloading/installing.
* `frontend/src/features/startup/components/StartupWizard.vue:63` shows the update button when `task.kind === "self_update"` and `state.self_update.available`.
* The same button is disabled when `busy` is true.

## Root Cause

The software update check and component setup run concurrently. When a new software version is detected, the UI can show the update button, but `state.running` remains true until all component jobs finish. Because the button is disabled by `busy`, the user cannot apply the software update while the component flow is still running. This also causes unnecessary component downloads/installs on an outdated app version.

## Recommended Design

1. Keep unified source detection and Release snapshot fetch before all checks, because self-update and components both consume that snapshot.
2. In `runTasksDefault`, run self-update check synchronously before constructing/starting component jobs.
3. After `checkSelfUpdate`, inspect `s.state.SelfUpdate`.
4. If `Available=true` and `Status=needs_action`, log that component checks are deferred until the app is updated, call `refreshCanEnterMain()` or otherwise ensure `CanEnterMain=false`, and return without starting component goroutines.
5. If `Status=ready` or `Status=failed`, proceed with the existing component job list and concurrency.

## Test Strategy

* Add a backend unit test in `internal/envsetup/service_startup_test.go` for a unified payload containing an app/self-update dependency newer than `svc.version`.
* In that test, inject a `runTasksFn` or component hook strategy that proves component jobs are not called. If `runTasksFn` wraps all tasks and makes this hard, factor the gating into a helper or adjust the test at the `runTasksDefault` level with local installed fixtures and no download side effects.
* Assert final state:
  * `SelfUpdate.Available == true`
  * `SelfUpdate.Status == statusNeedsAction`
  * `Running == false`
  * `CanEnterMain == false`
  * component steps remain pending and not downloading/installing
* Preserve existing test `TestRunStartupChecks_DoesNotBlockOnFFmpegDetectWhenSelfUpdateFails`: failed self-update check should still continue component work.

## Open Product Choice

When software update check fails because the unified snapshot lacks app/self-update data or asset selection fails, the current behavior continues component checks. The PRD assumes this should remain unchanged so transient update-service failures do not block first launch.
