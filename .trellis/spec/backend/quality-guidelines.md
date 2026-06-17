# Quality Guidelines

> Code review standards, testing requirements, lint/format conventions, and forbidden patterns.

---

## Testing Requirements

### Required Test Coverage

| Package | Minimum | Notes |
|---------|---------|-------|
| `internal/config` | ✓ | Defaults, validation, range clamping |
| `internal/envsetup` | ✓ | Component lifecycle, logging, state transitions, fallback logic |
| `internal/release` | ✓ | Version comparison, asset selection, API parsing |
| `internal/endpoints` | ✓ | URL generation, source whitelist |
| `internal/app` | ✗ (currently minimal) | Integration-style tests preferred |
| `internal/download` | ✗ (currently minimal) | Integration with temp dirs |
| `internal/logging` | ✗ (interface-only package) | Test via envsetup log tests |

### Test Patterns Used in This Project

**1. Table-driven tests** — preferred for functions with multiple input/output cases:

```go
func TestCompareVersions(t *testing.T) {
    tests := []struct {
        a, b string
        want int
    }{
        {"1.0.0", "1.0.0", 0},
        {"2.0.0", "1.0.0", 1},
        {"1.0.0", "2.0.0", -1},
        {"1.0.0-beta", "1.0.0", -1},
        {"invalid", "1.0.0", 0},
    }
    for _, tt := range tests {
        got := CompareVersions(tt.a, tt.b)
        if got != tt.want {
            t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
        }
    }
}
```

**2. Temp directory tests** — for filesystem operations:

```go
func TestWriteHLAEFfmpegIni(t *testing.T) {
    hlaeDir := t.TempDir()
    ffmpegExe := `C:\tools\ffmpeg\bin\ffmpeg.exe`

    if err := writeHLAEFfmpegIni(hlaeDir, ffmpegExe); err != nil {
        t.Fatalf("writeHLAEFfmpegIni failed: %v", err)
    }
    // ... assertions
}
```

**3. Test helpers** — extracted for reuse across test files:

```go
func createZipBytesForTest(t *testing.T, entries map[string][]byte) []byte {
    t.Helper()
    // ...
}
```

**4. HTTP test server** — for API-dependent tests:

```go
func TestFetchUnifiedLatest(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // ...
    }))
    defer server.Close()
    // ...
}
```

**5. `testing.T` for `t.Cleanup`, `t.TempDir`, `t.Setenv`** — use Go 1.24+ test capabilities.

**6. Injectable function vars** — for packages that wrap external tools (FFmpeg exec, HTTP, file ops), define package-level `var` as the function to call, then substitute in tests:

```go
// In the package under test:
var FFmpegCommand = exec.Command          // production default
var FFmpegCommandContext = exec.CommandContext

// In _test.go:
func TestMerge(t *testing.T) {
    orig := producemerge.FFmpegCommand
    producemerge.FFmpegCommand = func(name string, args ...string) *exec.Cmd {
        return exec.Command("echo", "fake")
    }
    t.Cleanup(func() { producemerge.FFmpegCommand = orig })
    // ...
}
```

This pattern is used in `fivee`, `wanmei`, and `producemerge`. It applies to **leaf/utility packages only** — never to `internal/app` or `internal/envsetup` (those use struct-field injection instead).

### Assertion Style

- **Use standard `testing` package** — no external test assertion library
- Use `t.Fatal`/`t.Fatalf` when the test cannot continue (setup failure, precondition missing)
- Use `t.Error`/`t.Errorf` for assertion failures
- Use `t.Helper()` on shared test helpers

```go
if err != nil {
    t.Fatalf("write failed: %v", err)   // can't continue
}
if !strings.Contains(content, expected) {
    t.Errorf("missing %q in output, got: %q", expected, content)  // can continue
}
```

---

## Concurrency

### Pattern: Call `cancel()` inside the lock when a goroutine re-acquires it

When a long-running goroutine (supervisor / watcher / state-machine loop) re-acquires the same `s.mu` that `Stop()` holds, the `cancel()` (or `close(stopCh)`) call MUST happen **inside** `Stop()`'s critical section, not after `Unlock()`. Otherwise the goroutine can pass its `ctx.Done()` check **between** `Stop()`'s unlock and cancel, miss the signal, then continue work — typically leaving an orphaned listener / server / connection that `Stop()` never closes, deadlocking `WaitGroup.Wait()`.

**Wrong** — race window between `Unlock` and `cancel`:

```go
func (s *Service) Stop() error {
    s.mu.Lock()
    server := s.server
    s.mu.Unlock()        // <-- supervisor may take s.mu here
    s.cancel()           // <-- and observe ctx not yet Done
    server.Close()
    s.wg.Wait()          // deadlock: orphaned supervisor never sees cancellation
    return nil
}
```

**Correct** — cancellation visible to any future `s.mu` acquirer:

```go
func (s *Service) Stop() error {
    s.mu.Lock()
    s.cancel()           // goroutine taking s.mu after this point sees ctx.Err() != nil
    server := s.server
    s.mu.Unlock()
    server.Close()
    s.wg.Wait()
    return nil
}
```

**Why this is safe under "no blocking I/O under lock"**: `context.CancelFunc` and `close(ch)` are non-blocking primitives — they only flip a state flag and wake any selectors. They are explicitly exempt from the "no blocking I/O" rule.

**Where this applies**: any package that holds a long-running goroutine whose first action after a `select { case <-time.After: case <-ctx.Done: }` (or similar wake-up) is to acquire `s.mu`. Used in `internal/producews` (supervisor goroutine wrapping `Listen + Serve` with bounded retry).

**Companion checklist item**: see Code Review Checklist below.

---

## Code Style & Formatting

### Go

- **`gofmt` / `go vet` must pass** — the project uses standard Go formatting
- **No lint tools** (golangci-lint, staticcheck) are configured — rely on `go vet`
- **Use `go test ./...`** for verification — all packages must pass

### TypeScript / Vue

- Standard TypeScript via `npm run build` with `vue-tsc` type checking
- **Vue 3 Composition API** (not Options API) — all components use `<script setup lang="ts">`
- **Auto-imports**: unplugin-auto-import handles Vue APIs (`ref`, `computed`, `onMounted`); do not import them manually

---

## Forbidden Patterns

### Go

| Pattern | Why | Instead |
|---------|-----|---------|
| `errors.New` with dynamic string | Cannot wrap, harder to chain | `fmt.Errorf("...: %w", err)` |
| `log.Println` or `fmt.Print*` for startup logging | Bypasses structured logging system | `s.emitLogWithFields(...)` in envsetup context |
| Global mutable state / package-level vars for data | Test pollution, race conditions | Dependency injection via `Service` struct fields |
| Injectable function vars in `app`/`envsetup` | Those packages use struct fields | Use the injectable var pattern only in leaf packages (see Testing Pattern #6 above) |
| Hand-editing `frontend/wailsjs/**` | Will be overwritten by `wails build` | Regenerate via Wails |
| Hand-editing `frontend/src/auto-imports.d.ts` or `components.d.ts` | Auto-generated | Regenerate via dev server |

### Frontend

| Pattern | Why | Instead |
|---------|-----|---------|
| Vue 2 Options API | Mixin-based, deprecated | `defineComponent` with `<script setup>` |
| Manual Vue API imports (`import { ref } from 'vue'`) | Inconsistent with auto-import setup | Use unplugin-auto-import |
| Hand-editing `wailsjs` files | Will be overwritten | Change Go binding signatures, regenerate |
| Using `@/` prefix for `frontend/src` imports | Correct — keep using this | N/A |

---

## PR / Commit Standards

- **Use conventional commit prefixes**: `feat:`, `fix:`, `chore:`, `docs:`, `test:`
- **Keep commits focused on one concern** — do not mix unrelated changes
- **Run `go test ./...`** (backend) and **`cd frontend && npm run build`** (frontend) before commit
- When changing both backend and frontend, run **both** checks

---

## Code Review Checklist

- [ ] All new code has corresponding tests (or clear reason why not)
- [ ] Tests use `testing.T` standard library only — no external assertion libs
- [ ] `go test ./...` passes
- [ ] `cd frontend && npm run build` passes (if frontend changed)
- [ ] No hand-edited auto-generated files (`wailsjs/`, `auto-imports.d.ts`, `components.d.ts`)
- [ ] Error messages are descriptive and in Chinese (user-facing) where appropriate
- [ ] Log entries use structured fields, not inline `fmt.Sprintf`
- [ ] State mutators acquire `mu.Lock()` / `mu.Unlock()` — no data races
- [ ] `Stop()` / shutdown methods cancel ctx **inside** `mu` if any owned goroutine re-acquires the same lock (see Concurrency section above)
- [ ] No dead code, no commented-out blocks
- [ ] New or changed stable contracts (Wails methods, event names, state enums) are documented in `AGENTS.md`

---

## Running Checks

```bash
# Backend
go test ./...

# Specific packages (for faster iteration)
go test ./internal/envsetup ./internal/release

# Frontend
cd frontend && npm run build

# Development mode
wails dev
```
