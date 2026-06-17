# Self-Update Button Opens Direct Asset URL Instead of GitHub Release Page

## Goal

Make the "前往下载" button on the startup wizard open the **direct asset download URL** (country-aware: `mirror_url` for CN, asset `github_url` for non-CN) in the user's default browser, instead of the GitHub **release HTML page**.

Motivation: CN users currently click the button and land on `github.com/.../releases/tag/vX.Y.Z`, which is often slow / unreachable without a VPN. The release API already returns a country-appropriate direct download URL (mirror for CN, GitHub for non-CN) which is what's used in-app for component downloads. Surfacing that same URL to the browser gives CN users a fast, reliable download path. Update notes are already shown in-app via the post-update changelog modal (v2.0.3 feature), so losing the release-page-as-notes-display is acceptable.

## What I already know (from repo inspection)

- `internal/envsetup/selfupdate.go:28` — `checkSelfUpdate` already calls `collectReleaseAssetCandidates("self_update", source, release.SelectAppExeAsset)` and gets a slice `candidates[]releaseAssetCandidate` ordered by country preference. `candidates[0].AssetURL` is exactly what we want.
- `internal/envsetup/release_fallback.go:88-96` — `orderedAssetURLsByCountry`:
  - CN (or empty country code): `mirror_url` first, then `url`/`download_url` (direct GitHub)
  - non-CN: `github_url` only
  Empty URLs are skipped (line 56-64), so `candidates[0].AssetURL` is always non-empty if the function returns success.
- `internal/envsetup/selfupdate.go:76, 98` — both branches (up-to-date / update-available) currently set `state.SelfUpdate.URL = infoManualURL(...)` which returns `info.HTMLURL` first → the GitHub release page.
- `internal/envsetup/service_actions.go:140-165` — `OpenManualDownload("self_update")` opens `state.SelfUpdate.URL` via `runtime.BrowserOpenURL`. No change needed there.
- Fetch-failure branch (`selfupdate.go:33-50`) sets `URL: endpoints.ManualURLFor("self_update", string(source))` — keep as-is (when we have no asset info, falling back to a manual page is sensible).

## Requirements

1. In `checkSelfUpdate`, change the `update available` branch (`selfupdate.go:92-101`) to set `state.SelfUpdate.URL = candidates[0].AssetURL` (the country-ordered top candidate).
2. Leave the `up-to-date` branch using `infoManualURL(...)` — `Available=false` means the button isn't shown, so this URL is effectively unused; no need to touch it. (Could simplify later if desired, but out of scope.)
3. Leave the fetch-failure branch unchanged (keep `endpoints.ManualURLFor` fallback).
4. Frontend: no change. Button still calls `OpenManualDownload("self_update")`; backend swap is transparent.
5. Update `internal/changelog/notes/v2.0.3.md` to note the behavior change (one bilingual line).

## Acceptance Criteria

- [ ] `go build ./...` and `go test ./...` pass.
- [ ] `cd frontend && npm run build` passes.
- [ ] On CN GeoIP, when an update is available, clicking 前往下载 opens a `mirror_url` (or `gh-proxy`/direct GitHub asset URL if mirror is absent) — NOT a `/releases/tag/...` HTML page.
- [ ] On non-CN GeoIP, clicking 前往下载 opens the asset `github_url` (direct `.exe` download from GitHub).
- [ ] When the release-info fetch fails, clicking still opens the existing `ManualURLFor` fallback (no regression).
- [ ] Existing tests still pass (logging_export, app_workspace, etc.).
- [ ] v2.0.3 changelog updated.

## Definition of Done

- Tests pass (Go + frontend build).
- Changelog notes file updated.
- PR opened against `main`.

## Technical Approach

One-line backend change in `checkSelfUpdate`:

```go
// update-available branch (selfupdate.go:92-101)
s.state.SelfUpdate = SelfUpdateState{
    Status:    statusNeedsAction,
    Available: true,
    Current:   current,
    Latest:    latest,
    URL:       candidates[0].AssetURL,  // was: infoManualURL("self_update", source, info)
}
```

Country selection happens upstream in `orderedAssetURLsByCountry`, so no new logic needed here.

## Decision (ADR-lite)

**Context**: Browser opens GitHub release page; CN users can't reach it reliably. The release API already produces a country-appropriate asset URL.

**Decision**: Reuse `candidates[0].AssetURL` (the same URL the in-app downloader would have used pre-removal) as the `SelfUpdate.URL`.

**Consequences**:
- Pro: CN users get fast, mirror-hosted direct download.
- Pro: Symmetry with how component downloads (HLAE/plugin/ffmpeg) are sourced.
- Con: Browser will immediately prompt to download `.exe` — no intermediate release-notes page. Mitigated by the in-app post-update changelog modal.
- Con: If mirror_url is broken at a given moment, the user has no easy fallback in the UI (they'd need to navigate to GitHub manually). Acceptable risk — same fallback shape as the previous in-app updater.

## Out of Scope

- Changing the up-to-date branch URL (button isn't shown so URL is dead).
- Adding a secondary "open release notes page" link in the UI.
- Auto-retry across `candidates[1:]` if `candidates[0]` is unreachable (browser handles unreachable URLs; we don't get a callback).
- Touching `infoManualURL` itself — still used elsewhere (`OpenManualDownload` for component IDs that aren't `self_update`).

## Technical Notes

- Files to edit: `internal/envsetup/selfupdate.go` (one branch), `internal/changelog/notes/v2.0.3.md` (one line).
- No new Wails methods; no frontend changes; no wailsjs regen needed.
- Constraint reminder (CLAUDE.md): do NOT hand-edit `frontend/wailsjs/**`. Confirmed not needed for this change.
- `releaseAssetCandidate.AssetURL` is a struct from `release_fallback.go` — separate from the deleted `SelfUpdateState.AssetURL` (which is gone and stays gone).
