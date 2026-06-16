# Startup State Machine

## Scenario: Self-Update Gates Component Checks

### 1. Scope / Trigger

Trigger: `RunStartupChecks()` changes the startup state consumed by the frontend startup wizard. The flow spans unified Release snapshot loading, self-update state, component status steps, and `can_enter_main`.

### 2. Signatures

Backend entry points stay unchanged:

```go
func (s *Service) RunStartupChecks() StartupState
func (s *Service) ApplySelfUpdate() StartupState
```

Wails-facing bindings stay unchanged:

```go
func (a *App) RunStartupChecks() envsetup.StartupState
func (a *App) ApplySelfUpdate() envsetup.StartupState
```

### 3. Contracts

After source detection and unified Release snapshot fetch, `RunStartupChecks()` must check `SelfUpdate` before starting component tasks.

If a newer app version is available:

```text
self_update.status    = needs_action
self_update.available = true
can_enter_main        = false
steps[].status        = pending
```

The component jobs for `hlae`, `plugin`, `ffmpeg`, and `cs2` must not start in that run. The frontend update button becomes usable from backend state/events once `running=false`; no frontend-local override should be needed.

If the app is up to date, component jobs may run concurrently as before. If self-update checking fails, it remains non-fatal and component checks continue.

### 4. Validation & Error Matrix

| Condition | Expected state | Component tasks |
|---|---|---|
| New app version found | `self_update=needs_action`, `available=true`, `can_enter_main=false` | Not started |
| App is current | `self_update=ready`, `available=false` | Started |
| Self-update release lookup fails | `self_update=failed`, error set | Started |
| Unified Release snapshot fetch fails before tasks | `source_step=failed`, local fallback rules apply | Existing fallback behavior |

### 5. Good/Base/Bad Cases

Good: unified snapshot contains app `v2.0.0` while current app is `1.0.0`; HLAE/plugin/FFmpeg/CS2 stay `pending` and the user updates the app first.

Base: unified snapshot app version equals current app; component checks proceed in the existing concurrent path.

Bad: self-update check runs in parallel with component downloads, causing outdated app builds to download/install components before the user can apply the software update.

### 6. Tests Required

Required backend assertions:

* New app version defers component tasks and leaves steps `pending`.
* `running` is false in the final stored startup state after `RunStartupChecks()` returns.
* `can_enter_main` remains false while `self_update.available=true`.
* Existing self-update failure tests still prove component checks continue.

Run:

```bash
go test ./internal/envsetup
go test ./...
```

### 7. Wrong vs Correct

#### Wrong

```go
go s.checkSelfUpdate(source)
go s.runComponent(componentHLAE, s.ensureHLAEWithFallback)
```

This lets component installation race ahead while the app itself is outdated.

#### Correct

```go
s.checkSelfUpdate(source)
if s.state.SelfUpdate.Available {
    return
}
// Start component jobs only after app version is accepted.
```

The real implementation must read/update state under `Service.mu` and emit state/log events using the existing envsetup helpers.
