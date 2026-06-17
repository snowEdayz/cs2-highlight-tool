# Changelog Modal Conventions in Open-Source Desktop Apps

- **Query**: How do open-source desktop apps implement the "show changelog after self-update" modal? Focus on content storage format and rendering strategy.
- **Scope**: external (source-pinned evidence from upstream repos)
- **Date**: 2026-06-17

Sources surveyed (with raw-source confirmation):
- VS Code (Electron, MIT) — `microsoft/vscode`
- Standard Notes (Electron, AGPL) — `standardnotes/app`
- Joplin (Electron, MIT) — `laurent22/joplin`
- Logseq (Electron, AGPL) — `logseq/logseq`
- Lapce (Rust/Floem, Apache-2.0) — `lapce/lapce`
- Atom (archived Electron) — `atom/atom`
- Satisfactory Mod Manager (Wails v2, GPL-3.0) — `satisfactorymodding/SatisfactoryModManager`

A web-search for an Obsidian source was not performed: Obsidian's main app is closed-source, and no definitive public implementation file could be located in this session. **Could not locate definitive source for Obsidian or Discord** — both are closed-source, and the available blog posts are second-hand. Omitted from the table to avoid fabrication.

## Summary table

| App | Stack | Storage | Rendering | Persistence key | Cross-ver behaviour |
| --- | --- | --- | --- | --- | --- |
| VS Code | Electron | **Per-major-minor remote markdown** (`https://code.visualstudio.com/raw/v1_109_update.md`) with JSON frontmatter; falls back to per-version release-notes markdown for the full editor | Webview, markdown rendered via internal `renderMarkdownDocument` (no external `marked`); compact widget uses VS Code's `MarkdownString` + `IMarkdownRendererService` | `postUpdateWidget/lastKnownVersion` (stores `{version, commit, timestamp}` JSON, `StorageScope.APPLICATION`) and **also** `releaseNotes/lastVersion` (older path, stores parsed `IVersion`) | Single comparison: stored vs current. Shows widget only on **major/minor change** (`isMajorMinorVersionChange`). Patch updates intentionally silent. **First install: silent** (`from === undefined` returns `false`). |
| Standard Notes | Electron | **Single remote `CHANGELOG.md.json`** (pre-built from conventional-changelog markdown) at `https://raw.githubusercontent.com/.../web/CHANGELOG.md.json`; fetched at runtime, cached in memory | Custom React component renders structured `parsed` sections (`Bug Fixes`, `Features`) as plain `<ul><li>` — no markdown renderer, no sanitizer | Disk service key `StorageKey.LastReadChangelogVersion` stores a single semver string (latest version on read) | When user opens the "What's New" preferences pane, **all** versions are listed; "unread" badge per-version via `compareSemVersions(version, lastReadVersion) > 0`. Calling `markAsRead()` sets lastRead to `versions[0].version` — i.e. user is opted in to viewing intermediate notes, not auto-modaled. |
| Joplin | Electron | **None local** — links to GitHub Releases page (`https://github.com/laurent22/joplin/releases`) in external browser | Browser (no in-app rendering) | None (no per-version "seen" key; the popup is tied to electron-updater `UpdateDownloaded` ipc event, so it only fires once per actual update) | N/A — user only sees latest release page on GitHub |
| Logseq | Electron (ClojureScript) | **None local** — help menu links to `https://docs.logseq.com/#/page/changelog` (hosted docs site) | External browser | None | N/A — pull model. User opens menu manually. |
| Lapce | Rust / Floem | **None local** — fetches GitHub Releases API at runtime for version metadata only; no changelog text shown in-app | N/A (just version badge + download/restart buttons) | None | N/A — does not show release notes; relies on GitHub release page if user wants details |
| Atom (archived) | Electron | **None local** — release notes are shown via the About pane only when an update is available; content is the release-page link | About-pane HTML | `localStorage['about:version-available']` (stores the available, not-yet-applied version) | Cleared when installed version catches up: `semver.lte(availableVersion, atom.getVersion()) → clearUpdateState()`. No modal on launch — only a status-bar tile while an update is pending. |
| Satisfactory Mod Manager | Wails v2 + Svelte | **Remote GitHub Releases API**: backend `GetChangelogs() (map[string]string, error)` loops `releases` and returns `tag → release.Body` (markdown). Emitted to frontend via Wails event `updateAvailable` with `{Version, Changelogs}` payload. | Frontend renders each release body via `marked` + `DOMPurify` (`frontend/src/lib/utils/markdown.ts`); displayed inside a Skeleton-UI modal | None for "post-update show-once" — modal is shown reactively from `updateAvailable` event each time a new update is detected (i.e. only fires once per actual updater detection cycle) | Frontend receives a `map[version]string`; modals can iterate / present a version range, so users see all intermediate notes if the SMM updater accumulated them. |

## Per-app deep dives

### VS Code (most relevant — two distinct, co-existing UIs)

VS Code has **two** post-update UI paths registered side-by-side in `update.contribution.ts`:

1. **"Release Notes" editor** (older, full webview)
   - Triggered by `ProductContribution` in `src/vs/workbench/contrib/update/browser/update.ts` (line ~150-200).
   - Storage key: `releaseNotes/lastVersion` (`StorageScope.APPLICATION`, `StorageTarget.MACHINE`).
   - Persistence: stores the **current** version as a string after detecting a major/minor jump and showing the notes.
   - Trigger: on workbench startup, **gated by `hostService.hadLastFocus()`** — only the most-recently-focused window opens it, avoiding spam when multiple windows reopen.
   - First-install handling: relies on the `tryParseVersion(... '')` returning `undefined`, which then **falls through the `lastVersion && currentVersion && isMajorMinorUpdate(...)` guard** — so no notes on first install.
   - Render: `ReleaseNotesManager.loadReleaseNotes()` fetches `https://code.visualstudio.com/raw/v{1_85}.md` (per-minor remote markdown), passes it through `renderMarkdownDocument` (internal renderer, no `marked` dependency), opens result in a `webviewWorkbenchService` webview.
   - Source: https://github.com/microsoft/vscode/blob/main/src/vs/workbench/contrib/update/browser/update.ts
   - Source: https://github.com/microsoft/vscode/blob/main/src/vs/workbench/contrib/update/browser/releaseNotesEditor.ts

2. **"Post-update widget"** (newer compact hover/dialog)
   - Code: `src/vs/workbench/contrib/update/browser/postUpdateWidget.ts`.
   - Storage key: `postUpdateWidget/lastKnownVersion` (`StorageScope.APPLICATION`, `StorageTarget.MACHINE`).
   - Value stored: JSON of `{ version, commit, timestamp }` (so commit hash, not just version, can disambiguate insider builds).
   - First-install handling (explicit): `detectVersionChange()` returns `false` when `from === undefined`, **after** storing the current value — so the very first run is silent and only subsequent major/minor jumps show the widget.
   - Trigger: `tryShowOnStartup()` runs on workbench start, gated by `hostService.hadLastFocus()` and by the `update.showPostInstallInfo` setting.
   - Remote content URL: built by `getUpdateInfoUrl(version)` → `https://code.visualstudio.com/raw/v1_109_update.md` (note the `_update` suffix that distinguishes this from the long-form release notes).
   - Content format: a markdown file with optional **JSON envelope** or **`---` frontmatter** carrying structured metadata (title, badge, banner image, button list, **and `features[]` list of `{icon, title, description}`** — capped at 5). See `parseUpdateInfoInput` in `updateInfoParser.ts`.
   - Render: VS Code's own `MarkdownString` + `IMarkdownRendererService`; opens links via `IOpenerService`; sticky hover ("dialog") UI.
   - Cross-version: only checks `from` vs `to` — does not accumulate. If a user jumps 1.85 → 1.88, the widget shows content for **1.88 only** (the URL is built from the current version, so intermediate 1.86/1.87 are skipped).
   - Source: https://github.com/microsoft/vscode/blob/main/src/vs/workbench/contrib/update/browser/postUpdateWidget.ts
   - Source: https://github.com/microsoft/vscode/blob/main/src/vs/workbench/contrib/update/common/updateInfoParser.ts
   - Source: https://github.com/microsoft/vscode/blob/main/src/vs/workbench/contrib/update/common/updateUtils.ts

Take-away: VS Code has converged on **JSON-frontmatter-in-markdown** as the content format. Storage is remote (CDN), not embedded. Version comparison is single-step, not accumulating.

### Standard Notes (closest analog to "structured-JSON approach")

- Backend builds a structured **`CHANGELOG.md.json`** at release time by parsing the conventional-changelog `CHANGELOG.md`. Each version entry is:
  ```ts
  type ChangelogVersion = {
    version: string | null
    title: string                  // contains the compare-URL header from conventional-changelog
    date: string | null
    body: string                   // raw markdown body
    parsed: Record<string, string[]>  // sections like "Bug Fixes": [...lines...]
  }
  ```
- The JSON is fetched at runtime from `https://raw.githubusercontent.com/standardnotes/app/main/packages/web/CHANGELOG.md.json` — i.e. **also remote**, not embedded.
- `ChangelogService.markAsRead()` writes `LastReadChangelogVersion = changelog.versions[0].version` to disk service.
- `WhatsNew.tsx` highlights each version as "unread" via `compareSemVersions(version.version, lastReadVersion) > 0`.
- Rendering is **plain React `<ul><li>{item}</li></ul>`** of the `parsed[section]` strings; **no markdown renderer at all**. The cost is that the JSON is pre-formatted at build time by Standard Notes' CI (formatChangelogLine strips parens & skips lines that are a single word).
- First-install handling: when `lastReadVersion === undefined`, every entry is shown as "unread" badge — this is intentional in the preferences pane (it's not modal, it's user-initiated).
- This is **not a popup-after-update** — it's a "What's New" tab inside Preferences that the user opens manually. Standard Notes does **not** auto-open a post-update modal; it relies on the new-version badge in the sidebar.
- Sources:
  - https://github.com/standardnotes/app/blob/main/packages/ui-services/src/Changelog/ChangelogService.ts
  - https://github.com/standardnotes/app/blob/main/packages/ui-services/src/Changelog/Changelog.ts
  - https://github.com/standardnotes/app/blob/main/packages/web/src/javascripts/Components/Preferences/Panes/WhatsNew/WhatsNew.tsx
  - https://github.com/standardnotes/app/blob/main/packages/web/src/javascripts/Components/Preferences/Panes/WhatsNew/getSectionItems.tsx

### Joplin

- File: `packages/app-desktop/gui/UpdateNotification/UpdateNotification.tsx` on `dev` branch.
- No in-app changelog modal at all. When electron-updater fires `UpdateDownloaded`, Joplin shows a popup notification with a "See changelog" button that calls `shim.openUrl('https://github.com/laurent22/joplin/releases')` — the OS browser handles rendering.
- "Show once" is implicit: the popup is gated on the `UpdateDownloaded` IPC event, which only fires once per update cycle from electron-updater. No persisted version key.
- Source: https://github.com/laurent22/joplin/blob/dev/packages/app-desktop/gui/UpdateNotification/UpdateNotification.tsx

### Logseq

- The Help menu lists `[(t :help/changelog) "https://docs.logseq.com/#/page/changelog"]` (see `src/main/frontend/components/onboarding.cljs` line 32).
- Container component (`src/main/frontend/components/container.cljs` line ~241) similarly maps "Release Notes" to a hosted docs page.
- No post-update modal. Pull-model only.
- Source: https://github.com/logseq/logseq/blob/master/src/main/frontend/components/container.cljs

### Lapce

- `lapce-app/src/update.rs` only handles GitHub Releases API lookup + asset download + extraction (per-OS).
- `lapce-app/src/app.rs` (line ~3960–4000) spawns a background thread that polls `get_latest_release()` every 60 minutes and stores the result in a Floem `RwSignal<Arc<Option<ReleaseInfo>>>`. The UI surface is small — just a version-available indicator.
- No in-app changelog modal; release notes are on GitHub. No "show once" persistence because nothing is shown.
- Source: https://github.com/lapce/lapce/blob/master/lapce-app/src/update.rs
- Source: https://github.com/lapce/lapce/blob/master/lapce-app/src/app.rs (lines 3960-4000 region)

### Atom (archived)

- `packages/about/lib/main.js` stores `'about:version-available'` in `localStorage`. The status-bar tile shows when this key is set; clicking opens the About pane which contains the release-notes link.
- `clearUpdateState()` is called when `semver.lte(availableVersion, atom.getVersion())` — i.e. after the user updates and relaunches, the pending-update key is cleared.
- No post-update modal — Atom's design treats the About pane as the changelog surface.
- Source: https://github.com/atom/atom/blob/master/packages/about/lib/main.js

### Satisfactory Mod Manager (Wails v2 — most directly relevant to our stack)

- Backend (`backend/autoupdate/source/github/github.go`):
  ```go
  func (g *source) GetChangelogs() (map[string]string, error) {
      releases, _ := g.getReleasesData()  // GET https://api.github.com/repos/{repo}/releases
      changelogs := make(map[string]string)
      for _, release := range releases {
          changelogs[release.TagName] = release.Body  // raw GitHub release-notes markdown
      }
      return changelogs, nil
  }
  ```
- Backend dispatches via Wails events (`backend/autoupdate/autoupdate.go`):
  ```go
  Updater.Updater.UpdateFound.On(func(update updater.PendingUpdate) {
      wailsRuntime.EventsEmit(common.AppContext, "updateAvailable", &PendingUpdate{
          Version:    update.Version.String(),
          Changelogs: update.Changelogs,   // map[version]markdown
      })
  })
  ```
- Frontend renders via `marked + DOMPurify` (`frontend/src/lib/utils/markdown.ts`):
  ```ts
  import DOMPurify from 'dompurify';
  import { marked } from 'marked';
  export const markdown = (md: string): string =>
      DOMPurify.sanitize(marked(md) as string);
  ```
  Used inside `frontend/src/lib/components/Markdown.svelte`, which is then mounted inside the SMM update modal.
- No `lastShownChangelogVersion` persistence in SMM — the modal is gated on the *updater event*, not on app startup. This means SMM doesn't have a "first-install-spam" problem because it only ever shows the modal at the moment an update is detected (pre-install), not after install.
- Sources:
  - https://github.com/satisfactorymodding/SatisfactoryModManager/blob/master/backend/autoupdate/autoupdate.go
  - https://github.com/satisfactorymodding/SatisfactoryModManager/blob/master/backend/autoupdate/source/github/github.go
  - https://github.com/satisfactorymodding/SatisfactoryModManager/blob/master/frontend/src/lib/utils/markdown.ts
  - https://github.com/satisfactorymodding/SatisfactoryModManager/blob/master/frontend/src/lib/components/Markdown.svelte
  - https://github.com/satisfactorymodding/SatisfactoryModManager/blob/master/frontend/src/lib/components/modals/smmUpdate/SMMUpdateReady.svelte

## Patterns I see across projects

1. **Remote-fetched content dominates over embedded.** VS Code, Standard Notes, and SMM all pull changelog content over HTTPS at runtime (VS Code: per-version `.md` on its CDN; Standard Notes: a single `CHANGELOG.md.json` on raw.githubusercontent.com; SMM: GitHub Releases API). The reasoning is that release notes for the latest version are usually **not known when the binary was built** (the release notes are written/finalized at tag time, sometimes hot-patched). None of the surveyed projects embed per-version JSON in the binary.
2. **Format is markdown more often than structured JSON.** Four of seven (VS Code, Joplin, Logseq, SMM) treat the changelog body as opaque markdown. Standard Notes is the outlier — it parses markdown into structured `parsed[section] -> string[]` at build time (so the runtime renderer is dumb React). VS Code's newer widget adds a thin JSON envelope/frontmatter on top of markdown to carry buttons/badges/features, but the body itself is still markdown.
3. **Rendering uses platform-native facilities, not always `marked + dompurify`.** VS Code uses its own internal markdown renderer (with strict trust controls and command-link sanitization). Standard Notes avoids markdown rendering entirely by pre-parsing. SMM uses `marked + DOMPurify` (the closest match to your "Option B" path). Joplin and Logseq punt to the OS browser.
4. **First-install silence is an explicit, defensive design.** VS Code's `detectVersionChange()` returns `false` when no previous version is stored — *after* writing the current one — so a fresh install is always silent. Atom does similar: only sets `about:version-available` when the updater says an update is pending; first install never reaches that code path. **This is the most important behavioural pattern to copy.**
5. **The persisted value is almost always a single string (the last-shown version), not an ack-array.** VS Code: stores `{version, commit, timestamp}` (one entry). Standard Notes: single semver string. Atom: single pending-version string. No project stores a list of "all acknowledged versions" — that overhead is unnecessary when comparisons are monotonic semver.
6. **Cross-version accumulation is "show latest", not "show all intermediate".** Every project that auto-shows on startup (VS Code) shows **only the target/current version's notes**, not a concatenation 1.85→1.86→1.87. Standard Notes is the exception, but it's a user-initiated preferences pane, not a modal. So if your user jumps v1.0 → v1.3 you would typically show v1.3 only, with a "View full changelog" link for intermediate versions.
7. **Modal is triggered post-launch, not during launch.** VS Code uses `lifecycleService.when(LifecyclePhase.Restored)` (after workbench restore) plus `hadLastFocus()` gating. SMM uses a Wails event after update detection. None block app startup. Trigger latency is in the order of "after the main window is fully rendered."

## Mapping back to our 3 candidate approaches

- **A — Per-version structured JSON, embedded via `go:embed`, rendered with Naive UI components**
  - Closest industry analog: **Standard Notes' `CHANGELOG.md.json`** (same shape, just embedded instead of remote). No surveyed project embeds, but Standard Notes proves the structured format works without a markdown renderer.
  - Strength: zero new frontend deps, fully airgap-safe, type-safe content, easiest to render with Naive UI (`NCard`, `NList`, `NTag`, `NCollapse`).
  - Weakness: every release requires regenerating + committing a new JSON file; can't hot-patch a typo without a new app build (unlike VS Code's remote CDN).
- **B — Per-version Markdown, embedded via `go:embed`, rendered with `marked` + sanitizer**
  - Closest industry analog: **SatisfactoryModManager** (Wails + `marked` + DOMPurify), except SMM fetches markdown from GitHub Releases at runtime instead of embedding.
  - Strength: easiest content authoring (markdown is what release notes are usually written in anyway); minor formatting flexibility (links, bold, lists).
  - Weakness: adds `marked` + `DOMPurify` to the frontend bundle (~30 KB gzipped combined); needs to be careful with link handling (open in OS browser via `BrowserOpenURL`, not in webview).
- **C — Single CHANGELOG.md (Keep a Changelog style) parsed at runtime**
  - Closest industry analog: **Standard Notes' build-time JSON generation step** is the same idea but moved to build time. No surveyed project parses Keep-a-Changelog at runtime in the desktop binary itself.
  - Strength: one canonical file; same content reused for GitHub Releases, README, in-app modal.
  - Weakness: parser must be robust to formatting quirks (Keep-a-Changelog allows lots of variation); harder to extract structured fields like "highlights" without conventions; runtime parse cost (small but non-zero).

Hybrid options worth mentioning:
- **A + remote refresh** (Standard Notes flavor): ship a minimal embedded JSON for offline first-run, then optionally fetch the latest version's JSON from your release CDN to display newer entries. Best of both worlds for an internet-connected tool.
- **C → A at build time**: keep authoring in a single CHANGELOG.md and run a Go build-step that converts it to per-version JSON before `go:embed` (mirrors Standard Notes' build pipeline). You author once, ship structured.
- **B with embedded markdown but VS Code's frontmatter idea**: per-version `.md` files with a small JSON header (`title`, `highlights[]`, `badge`) so you can render a polished hero in Naive UI **above** the rendered markdown body. Lets you escape "wall of bullet points" feel without losing markdown freedom.

## Recommendation for this project

Given (1) this is a small Wails app the user is shipping to gamers who may be offline / on slow CN networks, (2) the existing project already has a unified release-snapshot fetched at startup (so a "fetch changelog from CDN" path is essentially free to add later), and (3) Naive UI already provides `NCard`/`NList`/`NTag`/`NCollapse`/`NScrollbar` — **Approach A (per-version JSON embedded via `go:embed`)** is the closest match to the project's stack and constraints. It mirrors Standard Notes' structural model without adding `marked` + `DOMPurify` to the frontend bundle, and it never depends on network reachability at the moment we want to celebrate an update.

Caveats:
- If release notes are authored by a non-engineer or change frequently after release, consider the **C → A build-step hybrid** so the canonical source stays markdown. A 30-line Go pre-build converter is enough.
- For persistence, copy VS Code's exact `detectVersionChange()` semantics: store `last_shown_changelog_version` (single semver string) in `config.json`, return early when it's empty (= first install), only show when stored < current AND major/minor change differs. This avoids both welcome-spam and patch-noise.
- For cross-version jumps (v1.0 → v1.3), show only the current version's notes by default but include a "View previous releases" link to GitHub releases — that's the dominant pattern across VS Code, Joplin, and SMM.

## Caveats / Not found

- **Obsidian**: closed-source; could not locate a definitive implementation file. Excluded.
- **Discord**: closed-source; only second-hand blog posts describe their changelog UX. Excluded.
- **Tauri example apps**: none of the official Tauri repos hosts a canonical "post-update changelog modal" example; community examples are scattered. SatisfactoryModManager (Wails) is a closer analog for our stack anyway, so we focused there.
- **GitHub code search** was used until rate-limited mid-session; remaining file discovery relied on `contents` API and raw URLs from known paths. All file paths shown above were retrieved (not guessed) from upstream repos as of the search date (2026-06-17).
