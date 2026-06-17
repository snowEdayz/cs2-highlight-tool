# brainstorm: full-round player POV recording

## Goal

Support a recording mode for users who want to record a selected player's full first-person round POV, rather than only short kill-window clips. The recording should start at each round's start tick, follow the selected player, stop when that player dies or the round ends, and output one video per round.

## What I already know

* Current users can select kill clips and optionally record victim POV clips.
* The requested feature is for recording a specified player through normal match rounds, not just near kill ticks.
* Full-round POV recording should not include freeze time; each round's recording start should be the freeze-time end / live-round start tick.
* The desired recording flow is one round per output file to preserve completed rounds if a later round fails.
* If victim POV recording is enabled, victim POV remains clip-based and should run after all full-round killer/player POV videos finish.
* The full-round POV recording plan should be independent from the existing kill-clip plan. It should be modeled as `demo + tracked player + per-round segments`, not as fake or expanded kill clip selections.
* MVP scope: each demo can track only one player for full-round POV recording.
* UI entry point proposed by the user: add a switch on the right side of the "selected materials" title area in the clip selection page.
* When the switch is enabled, previously selected killer clips for the current demo should be removed automatically.
* In full-round POV mode, the selected demo/material list should show a distinct tag indicating that the current demo tracks a player for full-round POV recording.
* While the switch is enabled, double-clicking kill clips should add victim POV clip selections only, not killer POV selections.

## Assumptions (temporary)

* The initial technical risk is demo parsing accuracy: we must prove we can derive round start ticks, tracked player death ticks, and round end ticks reliably from the demo library.
* The existing demoinfocs-golang dependency should be reused if it exposes the required events and player identity mapping.
* Full implementation will likely touch both backend and frontend contracts, so AGENTS stable contract docs may need updates when new fields/methods are finalized.

## Open Questions

* None for the current MVP. Reopen if implementation uncovers a blocking ambiguity.

## Requirements (evolving)

* Parse a demo and produce a textual simulation of full-round POV recording for a chosen player:
  * round number
  * round start tick and freeze-time end tick
  * tracked player death tick if present
  * round end tick
  * simulated recording segment start/end, where segment start excludes freeze time by using freeze-time end when available
* Do not implement production UI or plugin generation changes until demo tick extraction is validated.
* Keep full-round POV planning/generation code separated from kill-clip planning/generation code where practical, while reusing low-level shared helpers only when they are already generic.
* Allow exactly one full-round tracked player per demo in the MVP.
* For rounds where the tracked player survives, end the full-round POV segment at `RoundEnd`; do not extend to `RoundEndOfficial` or the next round transition.
* In full-round POV mode, the right-side player selector must use full demo player data (`meta.players`) so players without kill clips can still be tracked.
* In full-round POV mode, after selecting a full-demo player, the right-side kill list should still show that player's kill clips when they exist.
* In full-round POV mode, if the selected full-demo player has no kill clips, the right-side kill list must show an empty state explaining that the player has no kill information; this is not an error.
* In full-round POV mode, double-clicking a kill clip adds only victim POV recording for that kill; it must not add a killer POV clip because the killer/full-round POV is represented by the separate full-round plan.
* Enabling the full-round POV switch should immediately create/enable the full-round plan for the current demo and currently selected player. Changing the selected player while the switch is enabled updates that demo's full-round plan.
* When enabling full-round POV mode for a demo, clear all existing kill-clip material selections for that demo, including previously selected victim-view clips. Users can then re-add victim POV clips explicitly while full-round POV mode is enabled.

## Acceptance Criteria (evolving)

* [x] Given a demo file and target player, output every round's start tick, tracked player death tick if present, round end tick, and simulated recording range.
* [x] Distinguish tracked-player death from other players' deaths.
* [x] Handle rounds where the tracked player survives by ending at round end.
* [x] Identify any unreliable or missing tick data before feature implementation begins.
* [x] Confirm that full-round POV segments can start after freeze time rather than at raw `RoundStart`.
* [ ] In clip selection, enabling full-round POV mode for a demo clears existing material selections for that demo and creates a full-round plan for the currently selected player.
* [ ] In full-round POV mode, the player selector lists all parsed demo players, including players without kill clips.
* [ ] In full-round POV mode, a selected player with no kill clips shows a normal empty state explaining that the player has no kill information.
* [ ] In full-round POV mode, double-clicking a kill row adds only victim POV for that kill.
* [ ] In produce generation, full-round POV takes are generated before victim POV clip takes.
* [ ] Each full-round POV take records one round from `RoundFreezetimeEnd` to target death or `RoundEnd`.
* [ ] In full-round POV mode, rounds where the tracked player has 0 kills are skipped (not recorded as a take).
* [ ] In full-round POV mode, when the tracked player has zero kills in all rounds, frontend shows an empty-state warning and the start-produce action is disabled for that demo.

## Definition of Done (team quality bar)

* Tests added/updated where behavior changes.
* `go test ./...` passes for backend changes.
* `cd frontend && npm run build` passes for frontend changes.
* AGENTS/spec docs updated if public contracts, events, state fields, or UI/backend interface signatures change.
* Rollout/rollback considered if recording-mode behavior is risky.

## Out of Scope (for the first validation step)

* Final UI implementation.
* Final Wails API design.
* Final plugin JSON generation changes.
* Batch victim POV recording changes beyond understanding current behavior.
* Tracking multiple full-round POV players in the same demo.

## Decision (ADR-lite)

**Context**: Full-round POV recording has different semantics from kill clips: it is selected at the demo/player level, creates one take per live round, and ends at target death or round end instead of a configurable kill window.

**Decision**: Model full-round POV as a separate recording plan (`demo + tracked player + round segments`) rather than converting it into fake kill selections. Keep victim POV clip recording on the existing kill-clip flow and schedule it after full-round POV takes.

**Consequences**: The code remains easier to maintain and extend. The first MVP is intentionally limited to one tracked player per demo; future multi-player support can add multiple full-round plans without changing kill clip semantics.

### Segment Boundary Decision

**Context**: A live-round POV segment should capture normal round gameplay only, excluding freeze time and post-round transition content.

**Decision**: Start each segment at `RoundFreezetimeEnd` and end at the tracked player's in-round death tick, or at `RoundEnd` if the player survives.

**Consequences**: Segments are deterministic and avoid recording freeze time, victory animations, scoreboard transition, or other non-gameplay tail content. `RoundEndOfficial` remains metadata/fallback only, not the normal segment cutoff.

### Player Source Decision

**Context**: The current clip selection page uses `clip_players`, which contains only players with selectable kill clips. Full-round POV recording must also support players who have no kills, while still letting users add victim POV clips from the tracked player's kills when those kills exist.

**Decision**: In normal kill-clip mode, keep using `clip_players`. In full-round POV mode, use `meta.players` for the player selector, then map the selected SteamID back to `clip_players` only for displaying that player's available kill clips.

**Consequences**: Zero-kill players can be selected from the player list, but the backend skips rounds where the tracked player has 0 kills, so a zero-kill player produces an empty `plan.Segments`. The frontend shows an empty-state warning ("该玩家整局没有击杀，无可录回合") and disables the start-produce action for that demo. A player with kills still shows their kill list on the right, and those kill rows add victim POV only while full-round POV mode is enabled.

### Switch Behavior Decision

**Context**: The switch should have direct meaning and should not require a second "add" action after the user chooses a player.

**Decision**: Turning the switch on enables the current demo's full-round POV plan for the currently selected player. While the switch remains on, selecting a different player updates that plan.

**Consequences**: The UI remains simple: the switch controls whether this demo is tracked for full-round POV, and the player selector controls who is tracked.

### Selection Cleanup Decision

**Context**: Existing material selections are kill-clip records that can include killer POV and victim POV state. Keeping them when switching modes would make the UI ambiguous because full-round POV already replaces killer POV for the tracked player.

**Decision**: When the switch is turned on for a demo, clear all existing kill-clip material selections for that demo. In full-round POV mode, new double-click selections are victim POV only.

**Consequences**: Mode switching is predictable and avoids mixed old/new semantics. Users who still want victim POV clips can add them intentionally after enabling full-round POV mode.

## Technical Notes

* Repository stack: Wails v2, Go, Vue 3, TypeScript, Naive UI.
* Relevant backend area: `internal/demo` for demo parsing and likely produce/plugin generation code for recording commands.
* Relevant frontend area: `frontend/src/features/clips` for clip selection UI.
* Root, `internal`, and `frontend` AGENTS rules have been read for this task.
* Existing `internal/demo.ParseMetadata` uses `demoinfocs-golang/v5` and currently listens to `RoundStart` plus `Kill`; it records kill tick, killer/victim SteamID, slot, entity ID, side, weapon, headshot, and wallbang.
* Existing metadata does not yet expose round start tick, freeze-time end tick, round end tick, or a per-player death timeline.
* `demoinfocs-golang/v5@v5.2.0` exposes `events.RoundStart`, `events.RoundFreezetimeEnd`, `events.RoundEnd`, `events.RoundEndOfficial`, and `events.Kill`; current tick can be read via `parser.GameState().IngameTick()`.
* For the requested "normal match round POV" behavior, `RoundFreezetimeEnd` is the default recording start tick; `RoundStart` is only a fallback when freeze-time end is missing.
* Existing clip generation already separates killer passes before victim passes in `internal/clipsjson.Build`, which aligns with the requested "full player POV first, victim clips afterwards" flow.
* Existing frontend clip selection state stores selected player per demo and material selections per demo key; full-round POV selection should likely be demo-level/player-level state rather than being represented as fake kill selections.
* Local shell `go` is not on PATH, but Go is available at `/opt/homebrew/bin/go` (`go1.24.0 darwin/arm64`).
* Validation demo received: `9210250352818846092.dem`.
* Research report: `research/demo-round-tick-validation.md`.
* Probe result: for target `monesy` (`steam=76561199605406701`), all 20 rounds produced usable `record_start -> record_end` ranges, and `target_slot=12` was stable at each round freeze end.
* Recommended tick rule: use `RoundFreezetimeEnd` as recording start, fallback to `RoundStart`; use target death tick only if it is inside `[record_start, RoundEnd]`, otherwise end at `RoundEnd`.
* Edge case confirmed: `RoundEndOfficial` may be missing on the final round, so it cannot be the primary cutoff.
* Existing clip generation (`internal/clipsjson`) already schedules killer passes before victim passes. Full-round POV should not be forced through `buildKillerSegments`; add a separate full-round segment/plan path and reuse only generic low-level action/bootstrap helpers where practical.
* Existing frontend selected materials are stored in `materialByDemo`; full-round POV needs separate demo-level state so clearing/re-adding victim selections does not corrupt the full-round plan.

## Technical Approach

* Extend demo parsing with full-round POV data: per player/per round enough metadata to build `record_start`, `record_end`, `target_slot`, and ending reason.
* Add frontend state for full-round POV plans keyed by demo, separate from kill-clip material selections.
* In `ClipsPage`, add the switch in the selected-materials header area and change the right-side selector/list behavior based on mode:
  * normal mode uses `clip_players`
  * full-round POV mode uses `players` for selection and `clip_players` only to display available kills for victim POV selection
* Extend produce request types to include optional full-round POV plan data alongside existing selected kill items.
* Add a separate backend generation path for full-round POV segments, then append existing victim clip segments after full-round segments.
* Keep generated take metadata distinguishable via `view` or an equivalent field so history/status can show full-round POV takes separately from victim clip takes.
