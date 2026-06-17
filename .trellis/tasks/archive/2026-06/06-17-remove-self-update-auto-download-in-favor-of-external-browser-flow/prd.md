# Remove Self-Update Auto Download in Favor of External Browser Flow

## Goal

Remove the in-app self-update **download/apply/restart** logic for the application's own binary. When a new app version is detected at startup, the UI shows the available-update state as today, but the action button no longer downloads/replaces the exe — instead it opens the GitHub release page in the user's default browser, and the user manually downloads and replaces the exe.

The motivation: the existing self-update implementation is broken on Windows (the updater child process is killed by the Wails/WebView2 Job object before it can rename the exe; the `--pid` arg is parsed but ignored; failures are silent). Rather than rewrite the updater (brittle on Windows: file locks, AV interference, Job object inheritance, UPX-packed binary handling), we delete the feature entirely. Browser-based manual update is the convention for many similar Windows tools (HLAE itself, cs-demo-manager, etc.) and removes an entire class of bugs.

## Scope (Narrow)

**In scope (delete or simplify):**
- `internal/updater/` — whole package
- `main.go` — the `--apply-update` entry block
- `internal/envsetup/selfupdate.go` — `downloadAndApplySelfUpdate` and its only caller `Service.ApplySelfUpdate`
- `internal/app/app_startup.go` — Wails-exposed `App.ApplySelfUpdate`
- `frontend/wailsjs/go/app/App.*` — regenerated; the binding will be gone
- Frontend "立即更新" button → "前往下载" button calling `OpenManualDownload("self_update")` (already implemented)
- `SelfUpdateState.AssetURL` field — no longer used anywhere
- The `downloading` / `installing` branches in `normalizeSelfUpdateStatus` / `statusText` / `progressFor` for the `self_update` task

**Explicitly OUT of scope (must keep working):**
- Component auto-download (HLAE / plugin / ffmpeg) inside the app — unchanged
- Self-update **version detection** (`checkSelfUpdate`) — unchanged; still runs at startup, still surfaces `available: true` when newer version exists
- The startup gate that blocks `CanEnterMain` when an update is available — unchanged (user must acknowledge the update prompt by opening the browser; whether we still block main entry after they click is a question below)
- GeoIP / source resolution / release-info caching — unchanged
- The post-update changelog modal (v2.0.3 feature) — unchanged; still triggers on first launch after version bump

## What I already know (from repo inspection)

- `internal/envsetup/service_actions.go:140-165` — `Service.OpenManualDownload(componentID string)` **already special-cases `componentID == "self_update"`** and opens `state.SelfUpdate.URL` via `runtime.BrowserOpenURL`. The frontend just needs to call this instead of `ApplySelfUpdate`. No new Wails method needed.
- `internal/envsetup/release_fallback.go:29-38` — `infoManualURL("self_update", ...)` returns `info.HTMLURL` first → the GitHub release **page** URL (e.g. `https://github.com/hkslover/cs2-highlight-tool/releases/tag/v2.0.4`), which is exactly the right landing page (release notes + asset list).
- `internal/envsetup/state.go:79-87` — `SelfUpdateState` shape: `Status, Available, Current, Latest, URL, AssetURL, Error`. After this change, `AssetURL` is dead and `Status` will only ever be `pending` / `checking` / `ready` / `failed` / `needs_action`.
- `frontend/src/features/startup/components/StartupWizard.vue:63-71` — the "立即更新" button calls `applySelfUpdate` from the composable.
- `frontend/src/features/startup/composables/useStartupWizard.ts:118-120` — `applySelfUpdate` simply does `callBackend("ApplySelfUpdate")`.
- `frontend/src/features/startup/composables/startup-display.ts:18-35` — `normalizeSelfUpdateStatus` covers `downloading`/`installing`; can simplify.
- i18n keys at `zh-CN.json:365`: `"update_now": "立即更新"` — needs new copy. The en-US.json is maintained separately per CLAUDE.md.
- Tests touching `ApplySelfUpdate`: `internal/app/app_workspace_test.go:111-117` (`TestApplySelfUpdate_NoServiceReturnsWorkspaceInit`) — must be deleted along with the method.
- v2.0.2 broken updater leaves leftover `dataDir/updates/<version>/` directories on disk (best-effort, not always); whether to clean these up on first v2.0.3 launch is a question below.

## Requirements (evolving)

1. Delete `internal/updater/` package entirely.
2. Delete `--apply-update` entry block from `main.go`.
3. Delete `Service.ApplySelfUpdate` (envsetup) and `App.ApplySelfUpdate` (Wails binding). Delete `downloadAndApplySelfUpdate` and dead helpers (`firstGitHubURL`, `isGitHubURL`) in `internal/envsetup/selfupdate.go`.
4. Delete `internal/app/app_workspace_test.go:111-117` (`TestApplySelfUpdate_NoServiceReturnsWorkspaceInit`).
5. Drop `SelfUpdateState.AssetURL` field (Go struct + frontend TS type).
6. Simplify `normalizeSelfUpdateStatus` to drop `downloading` / `installing` cases.
7. Frontend "立即更新" button: relabel + change handler to call `OpenManualDownload("self_update")` (reuses existing Wails method).
8. Regenerate `frontend/wailsjs/go/app/App.*` via `wails build` (or manual edit consistent with auto-generation — but per CLAUDE.md these are auto-generated; do not hand-edit).
9. Bump `wails.json` `version` and `info.productVersion` to the new release version.
10. Add a new `internal/changelog/notes/<version>.md` describing this change (bilingual, matching v2.0.3.md format).
11. Keep the existing startup gate: when an update is available, `CanEnterMain = false` — user is locked at the startup wizard until they update. (Decided.)
12. UI: only the relabeled primary button; do NOT add an "打开安装目录" helper button. (Decided.)
13. Do NOT clean up leftover `dataDir/updates/` directories — leave them on disk untouched. (Decided.)
14. Version number: **reuse `v2.0.3`** (previously deleted tag/release). The existing `internal/changelog/notes/v2.0.3.md` already covers other v2.0.3 features; append one line under both 中文 / English `### 修复` for "改为浏览器跳转下载". (Decided.)

## Acceptance Criteria

- [ ] `go build ./...` and `go test ./...` pass.
- [ ] `cd frontend && npm run build` passes (regenerated wailsjs has no `ApplySelfUpdate`).
- [ ] On startup with a newer release available, the wizard shows the self_update task with status `needs_action` and a single primary button.
- [ ] Clicking the button calls `OpenManualDownload("self_update")` on the backend and opens the GitHub release **page** (not direct asset download) in the system default browser.
- [ ] After clicking, the UI does NOT show any "downloading" / "installing" state for the self_update task (those statuses are unreachable).
- [ ] No reference to `--apply-update`, `internal/updater`, `ApplySelfUpdate`, or `AssetURL` (for self_update) remains in the codebase.
- [ ] Existing component download flows (HLAE/plugin/ffmpeg) still work in-app — manual test or covered by existing tests.
- [ ] CN GeoIP users: clicking the button opens a URL that loads (GitHub release HTML page works without gh-proxy).
- [ ] If the user has a previously-downloaded `dataDir/updates/<version>/` from v2.0.2 / earlier, the new version either ignores it (no crash) or cleans it up (decision pending).

## Definition of Done

- Tests added/updated (Go + frontend if any).
- Lint / type-check / CI green.
- Changelog notes file added.
- `wails.json` version bumped.
- PR opened against `main`; after merge, push the new version tag to trigger the release workflow.

## Open Questions

_All resolved — see Decisions below._

## Decisions (ADR-lite)

**Context**: v2.0.2 self-updater is broken on Windows. Fix-in-place is fragile; user chose to remove the feature.

**Decisions**:
- Delete `internal/updater/`, `--apply-update` entry, `ApplySelfUpdate` (both Service and App), `downloadAndApplySelfUpdate`.
- Frontend "立即更新" button → "前往下载" calling existing `OpenManualDownload("self_update")` (which already opens `runtime.BrowserOpenURL` on the release HTML page URL). No new Wails method.
- Keep `CanEnterMain = false` gate when update available — user must acknowledge by clicking the browser button (which still leaves the wizard open, but they can come back later).
- No "打开安装目录" helper button.
- No cleanup of `dataDir/updates/` leftover folders.
- Release as **v2.0.3** (reuse deleted tag). Append a fix line to existing `internal/changelog/notes/v2.0.3.md`.

**Consequences**:
- v2.0.2 users are stuck on v2.0.2 forever (their broken updater can never finish). They must download v2.0.3 manually one time. This is acceptable: future updates will always be manual, so this is the natural migration moment.
- Even though `CanEnterMain = false` is kept, clicking "前往下载" doesn't actually replace anything — the user will close the app manually, replace exe, relaunch. The gate is mainly a UX nudge ("here is an update, please act on it"), not a hard correctness requirement. If users dismiss / minimize and try to keep using the app: they'll be stuck at the wizard because gate stays on — acceptable trade-off per user's choice.

## Out of Scope (explicit)

- Rewriting the updater to work properly. (Decision: just delete it.)
- Changing the version-check mechanism, release API, or source selection.
- Auto-installing the downloaded exe (user does it manually).
- Differential / patch updates.
- In-app changelog rendering changes (post-update modal is unchanged).

## Technical Notes

- Files inspected: `main.go`, `internal/updater/updater.go`, `internal/envsetup/selfupdate.go`, `internal/envsetup/service_actions.go`, `internal/envsetup/state.go`, `internal/envsetup/release_fallback.go`, `internal/endpoints/endpoints.go`, `internal/app/app_startup.go`, `internal/app/app_workspace_test.go`, `frontend/src/features/startup/components/StartupWizard.vue`, `frontend/src/features/startup/composables/useStartupWizard.ts`, `frontend/src/features/startup/composables/startup-display.ts`, `frontend/src/shared/types/startup.ts`, `frontend/src/shared/i18n/zh-CN.json`.
- Constraint (CLAUDE.md): do not hand-edit `frontend/wailsjs/**` — must regen via build.
- Constraint (CLAUDE.md): only edit `zh-CN.json`, not `en-US.json`.
- Constraint (CLAUDE.md): Wails event/method names are stable public contracts — but `ApplySelfUpdate` will be DELETED; this is a hard break for any external consumer (there are none for this app). The frontend update is part of the same change so no internal break.
