# Error Handling

> How Go errors are created, wrapped, logged, and surfaced in this project.

---

## Overview

The project uses standard Go `error` interface with `fmt.Errorf` + `%w` for error wrapping. Custom error types are not used, and sentinel error variables should be avoided unless callers need stable machine-readable control flow across wrapping layers. Errors flow up from low-level packages through the service layer to the Wails binding layer, which returns them as strings to the frontend.

---

## Patterns

### 1. Error Creation

**Use `fmt.Errorf` with descriptive Chinese messages for user-facing contexts:**

```go
return fmt.Errorf("请选择包含 cs2.exe 的 CS2 安装目录")
return fmt.Errorf("解析配置文件失败: %w", err)
return fmt.Errorf("未找到 %s 文件", ext)
```

**Use English messages for developer-facing or internal-only contexts** (uncommon — Chinese is preferred for this project).

### 2. Error Wrapping

**Always wrap errors from lower layers with context:**

```go
// ✅ Good: add context about what failed
return fmt.Errorf("删除 HLAE 目录失败: %w", err)
return fmt.Errorf("获取 HLAE Release 失败: %w", err)

// ✅ Good: wrapping with contextual action
return fmt.Errorf("创建目录失败 %s: %w", dir, err)
```

**Use `%w` (not `%v` or `%s`) for wrapping** so `errors.Is` / `errors.As` work up the call chain.

### Cancellation Sentinel Exception

Download cancellation is a deliberate control-flow case, not a generic failure. `internal/download.ErrCanceled` is the stable sentinel used by startup download code to stop URL fallback chains and local-version fallback without parsing Chinese error text.

```go
if errors.Is(err, download.ErrCanceled) {
    return err
}
```

Rules for this exception:
- Keep the user-facing message as `"下载已取消"`.
- Wrap cancellation with `%w` if adding context; never detect it with `strings.Contains(err.Error(), ...)`.
- Limit `ErrCanceled` handling to code that must distinguish user cancellation from network/download failures.
- On cancellation, remove partially written download targets before returning.

**Do NOT wrap with fixed string concatenation** — always use `fmt.Errorf`:

```go
// ❌ Bad
errors.New("read failed: " + err.Error())

// ✅ Good
fmt.Errorf("读取配置文件失败: %w", err)
```

### 3. Return nil on Success

All functions return `(result, error)` — standard Go idiom:

```go
func resolveCS2Exe(dirOrEmpty, exeOrEmpty string) (string, error) {
    // ...
    return candidate, nil
}
```

### 4. Error as Unidirectional Flow

Errors flow **up**: low-level packages → service layer → Wails binding → frontend. They are not passed sideways or returned to callers twice.

---

## Error Handling in Wails Bindings

The `internal/app` package exposes methods to the frontend via Wails bindings. Errors are returned as Go `error` values, which Wails serializes to JavaScript Error objects.

**Convention**: Return `nil` on success, `fmt.Errorf` with a descriptive message on failure. The frontend displays these messages to the user.

```go
func (a *App) EnterMainApp() error {
    // ...
    if !canEnter {
        return fmt.Errorf("环境准备尚未完成")
    }
    return nil
}
```

---

## Error Handling in Service Layer (`internal/envsetup`)

### Step-based Error Flow

The startup state machine tracks per-component errors in `ComponentStatus.Error`:

```go
s.failStep(componentID, err, manualURL)
```

Internally: mutates `step.Status = statusFailed` and `step.Error = err.Error()`, emits a log, then calls `s.emitState()`.

### Fatal Errors

Errors that block the entire startup process are stored in `StartupState.FatalError`:

```go
s.setFatalError(err)
// Sets state.FatalError, sets CanEnterMain=false, emits log + state
```

Fatal errors stop the task runner. Non-fatal errors leave the step in `failed` or `needs_action` state but allow other steps to complete.

### Fallback Errors

Some components have fallback modes: if the remote release fetch fails but a local installation exists, the component enters `warning` state instead of `failed`:

```go
if localErr == nil {
    s.updateStep(componentHLAE, func(step *ComponentStatus) {
        step.Status = statusWarning
        step.Error = "最新版本获取失败，当前使用本地版本"
    })
    return nil // not an error — just a warning
}
return err // bubble up as real error
```

---

## Input Validation in `internal/config`

Config validation uses guard clauses in `ApplyDefaults()`:

```go
if cfg.EditFPS < MinEditFPS {
    cfg.EditFPS = MinEditFPS
}
if cfg.EditFPS > MaxEditFPS {
    cfg.EditFPS = MaxEditFPS
}
```

**For enum-constrained fields**: use a helper function + switch:

```go
func isSupportedEditQuality(quality string) bool {
    switch strings.ToLower(strings.TrimSpace(quality)) {
    case "standard", "high", "ultra":
        return true
    default:
        return false
    }
}
```

---

## Error Messages

- **User-facing errors**: Written in Chinese (e.g., `"请选择包含 cs2.exe 的 CS2 安装目录"`)
- **Technical / log-only messages**: Chinese with English variable names inserted via `fmt.Sprintf` or log fields
- **Do NOT include PII or sensitive paths**: URLs and file paths are logged via structured fields, not concatenated into message strings — the logging adapter sanitizes token/header values.

---

## Common Patterns

| Pattern | Where Used | Example |
|---------|-----------|---------|
| `fmt.Errorf("...: %w", err)` | All packages | Wrap low-level errors with context |
| `s.failStep(id, err, manualURL)` | `envsetup` | Mark step failed, log, emit state |
| `s.setFatalError(err)` | `envsetup` | Block entire startup process |
| `s.emitLog(fields, Error: ...)` | All `envsetup` | Log error with structured fields |
| Validation guard clause | `config` | Clamp to range, reject invalid enums |
| `os.IsNotExist(err)` | `config` | Handle missing config file |
| Nested `%w` wrapping | `release`, `download` | Deep error chains for debugging |

---

## What Not to Do

- ❌ **Do not define custom error types** — standard `error` + `fmt.Errorf` is sufficient
- ❌ **Do not parse localized error text for control flow** — use `errors.Is` for the explicit cancellation sentinel
- ❌ **Do not use `errors.New` with dynamic messages** — always use `fmt.Errorf`
- ❌ **Do not log-and-return** — return the error to the caller; the Wails binding layer is the final handler
- ❌ **Do not swallow errors silently** — even recoverable errors should surface via step state updates
- ❌ **Do not expose raw error messages to the frontend from interactive operations** — the Wails binding method is trusted; its error messages are directly displayed
