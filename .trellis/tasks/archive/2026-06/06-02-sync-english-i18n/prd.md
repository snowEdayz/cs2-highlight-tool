# Sync English i18n from Chinese updates

## Goal

Update `frontend/src/shared/i18n/en-US.json` so it matches the user's current uncommitted changes in `zh-CN.json`.

## What I already know

- The user manually modified `frontend/src/shared/i18n/zh-CN.json`.
- The dirty worktree also contains generated Wails files under `frontend/wailsjs/**`; those must remain untouched.
- Direct user request overrides the project default that AI normally does not edit `en-US.json`.

## Requirements

- Preserve all existing user changes in `zh-CN.json`.
- Update only the corresponding English translations in `en-US.json`.
- Do not edit generated files under `frontend/wailsjs/**`.

## Acceptance Criteria

- [ ] 5E player ID strings in `en-US.json` match the changed Chinese meaning.
- [ ] `workspace.*` keys present in `zh-CN.json` are present in `en-US.json`.
- [ ] Both locale JSON files parse successfully.
- [ ] Frontend build passes.

## Out of Scope

- Backend changes.
- Wails binding regeneration.
- Broader copywriting changes beyond the Chinese JSON delta.

## Technical Notes

- Relevant guidelines read: `AGENTS.md`, `frontend/AGENTS.md`, `.trellis/spec/frontend/i18n-guidelines.md`, `.trellis/spec/frontend/quality-guidelines.md`.
