# Wails Bindings

Concrete contracts for Go methods exposed through `internal/app.App` and consumed by `window.go.app.App.*`.

## Scenario: Clip Settings Launch Resolution Contract

### 1. Scope / Trigger

- Trigger: Settings UI exposes CS2 launch resolution choices through `GetClipSettings` / `SaveClipSettings`, and HLAE launch consumes the persisted value.
- Scope: `internal/app` Wails binding, `internal/config` persistence validation, frontend `ClipSettings` type/options, and HLAE command-line generation.
- Boundary: Settings UI selection -> Wails `SaveClipSettings` -> `config.json` -> HLAE `-cmdLine`.

### 2. Signatures

```go
func (a *App) GetClipSettings() (*ClipSettings, error)
func (a *App) SaveClipSettings(input ClipSettings) (*ClipSettings, error)

type ClipSettings struct {
    LaunchResolution string `json:"launch_resolution"`
}
```

Frontend shared type:

```ts
launch_resolution: "16:9" | "4:3" | "4:3_1280x960";
```

### 3. Contracts

- `launch_resolution` allowed values:
  - `16:9`: launch without explicit 4:3 width/height override.
  - `4:3`: launch with `-w 1440 -h 1080`.
  - `4:3_1280x960`: launch with `-w 1280 -h 960`.
- Default is `4:3`.
- Existing `4:3` config values must continue to mean `1440x1080`; do not repurpose this value for lower 4:3 resolutions.
- Frontend option labels are i18n keys under `main.settings.*`; per frontend rules, new labels are added to `zh-CN.json` only.

### 4. Validation & Error Matrix

- Missing, empty, or unsupported `launch_resolution` -> normalize to `config.DefaultLaunchResolution`.
- Supported value with surrounding whitespace -> trim and preserve the supported value.
- HLAE command generation should ignore unsupported values and omit resolution override rather than inventing dimensions.

### 5. Good/Base/Bad Cases

- Good: selecting `4:3_1280x960` persists that exact value and launches with `-w 1280 -h 960`.
- Base: selecting `4:3` preserves backward-compatible `-w 1440 -h 1080` behavior.
- Bad: changing the meaning of `4:3` to `1280x960`, which silently alters existing user configs.

### 6. Tests Required

- `internal/config`: `ApplyDefaults` preserves every supported `launch_resolution` value.
- `internal/app`: `SaveClipSettings` and `GetClipSettings` round-trip each supported launch resolution value.
- `internal/app`: `buildHLAECommandLine` maps `4:3` to `1440x1080`, maps `4:3_1280x960` to `1280x960`, and leaves `16:9` without 4:3 dimensions.
- Frontend: `cd frontend && npm run build` must pass so the shared TypeScript union and settings options stay aligned.

### 7. Wrong vs Correct

#### Wrong

```go
if launchResolution == "4:3" {
    cmdLine += " -w 1280 -h 960"
}
```

This breaks existing persisted `4:3` settings that already mean `1440x1080`.

#### Correct

```go
switch launchResolution {
case config.LaunchResolution4x3:
    cmdLine += " -w 1440 -h 1080"
case config.LaunchResolution4x3Low:
    cmdLine += " -w 1280 -h 960"
}
```

## Scenario: Settings Outputs Storage Management

### 1. Scope / Trigger

- Trigger: Settings needs to inspect and mutate app-managed files under `<dataDir>/outputs`.
- Scope: Wails binding methods in `internal/app`; frontend consumes the JSON response shape through shared TypeScript types.
- Boundary: Filesystem state -> Go binding response -> Vue Settings UI.

### 2. Signatures

```go
func (a *App) GetOutputsStorageStats() (*OutputsStorageStats, error)
func (a *App) OpenOutputsDirectory() error
func (a *App) ClearOutputsDirectory() (*OutputsStorageStats, error)

type OutputsStorageStats struct {
    OutputDir      string `json:"output_dir"`
    VideoCount     int    `json:"video_count"`
    TotalSizeBytes int64  `json:"total_size_bytes"`
}
```

### 3. Contracts

- `output_dir`: absolute managed outputs directory, resolved through `a.fixedRecordOutputDir()`.
- `video_count`: recursive count of generated video files with supported video extensions.
- `total_size_bytes`: recursive sum of all regular file bytes under `output_dir`, including non-video intermediate files.
- `GetOutputsStorageStats` must create `output_dir` if missing before scanning.
- `OpenOutputsDirectory` must create `output_dir` if missing before opening it.
- `ClearOutputsDirectory` deletes every direct child under `output_dir` and preserves `output_dir` itself.

### 4. Validation & Error Matrix

- Missing outputs directory -> create it and return zero stats.
- Directory creation fails -> return `创建输出目录失败: %w`.
- Directory scan/read fails -> return `统计输出目录失败: %w` or `读取输出目录失败: %w`.
- Child deletion fails -> return `清理输出目录失败: %w`.
- OS folder opener fails -> return `打开输出目录失败: %w`.

### 5. Good/Base/Bad Cases

- Good: nested `.mp4/.mov/.mkv/.avi` files are counted as videos, and all files contribute to `total_size_bytes`.
- Base: empty or missing `outputs` returns `video_count=0` and `total_size_bytes=0`.
- Bad: cleanup must not remove `<dataDir>/outputs` itself and must not target any sibling directory.

### 6. Tests Required

- Filesystem test with nested video and non-video files:
  - Assert `output_dir` equals `<dataDir>/outputs`.
  - Assert video count is recursive and case-insensitive.
  - Assert total size includes all files.
- Cleanup test:
  - Seed files and nested folders under outputs.
  - Call `ClearOutputsDirectory`.
  - Assert returned stats are zero, outputs directory remains, and direct children are gone.

### 7. Wrong vs Correct

#### Wrong

```go
os.RemoveAll(a.dataPath("outputs"))
```

This removes the managed directory itself and makes later UI/open-folder behavior depend on implicit recreation.

#### Correct

```go
entries, err := os.ReadDir(outputDir)
for _, entry := range entries {
    if err := os.RemoveAll(filepath.Join(outputDir, entry.Name())); err != nil {
        return nil, fmt.Errorf("清理输出目录失败: %w", err)
    }
}
```

This clears all managed children while preserving the stable directory path.

## Scenario: Settings Demo Storage Management

### 1. Scope / Trigger

- Trigger: Settings needs to inspect and mutate app-managed Demo files under `<dataDir>/demo`.
- Scope: Wails binding methods in `internal/app`; frontend consumes the JSON response shape through shared TypeScript types.
- Boundary: Filesystem state -> Go binding response -> Vue Settings UI.

### 2. Signatures

```go
func (a *App) GetDemoStorageStats() (*DemoStorageStats, error)
func (a *App) OpenDemoDirectory() error
func (a *App) ClearDemoDirectory() (*DemoStorageStats, error)

type DemoStorageStats struct {
    DemoDir        string `json:"demo_dir"`
    DemoCount      int    `json:"demo_count"`
    TotalSizeBytes int64  `json:"total_size_bytes"`
}
```

### 3. Contracts

- `demo_dir`: absolute managed Demo directory, resolved through `a.dataPath("demo")`.
- `demo_count`: recursive count of `.dem` files under `demo_dir`, case-insensitive.
- `total_size_bytes`: recursive sum of all regular file bytes under `demo_dir`, including non-Demo metadata or temporary files.
- `GetDemoStorageStats` must create `demo_dir` if missing before scanning.
- `OpenDemoDirectory` must create `demo_dir` if missing before opening it.
- `ClearDemoDirectory` deletes every direct child under `demo_dir` and preserves `demo_dir` itself.
- Clearing `demo_dir` is intentionally broad: it removes `raw`, `wanmei`, `5e`, and any other direct child owned by the managed Demo storage root.

### 4. Validation & Error Matrix

- Missing Demo directory -> create it and return zero stats.
- Directory creation fails -> return `创建Dem 目录失败: %w`.
- Directory scan/read fails -> return `统计Dem 目录失败: %w` or `读取Dem 目录失败: %w`.
- Child deletion fails -> return `清理Dem 目录失败: %w`.
- OS folder opener fails -> return `打开Dem 目录失败: %w`.

### 5. Good/Base/Bad Cases

- Good: nested `.dem/.DEM` files in `raw`, `wanmei`, and `5e` are counted as demos, and all files contribute to `total_size_bytes`.
- Base: empty or missing `demo` returns `demo_count=0` and `total_size_bytes=0`.
- Bad: cleanup must not remove `<dataDir>/demo` itself and must not target `<dataDir>/projects`, `<dataDir>/outputs`, or sibling directories.

### 6. Tests Required

- Filesystem test with nested Demo and non-Demo files:
  - Assert `demo_dir` equals `<dataDir>/demo`.
  - Assert Demo count is recursive and case-insensitive.
  - Assert total size includes all files.
- Cleanup test:
  - Seed files and nested folders under demo.
  - Call `ClearDemoDirectory`.
  - Assert returned stats are zero, demo directory remains, and direct children are gone.

### 7. Wrong vs Correct

#### Wrong

```go
os.RemoveAll(a.dataPath("demo"))
```

This removes the managed root and makes later UI/open-folder behavior depend on implicit recreation while widening the destructive operation.

#### Correct

```go
entries, err := os.ReadDir(demoDir)
for _, entry := range entries {
    if err := os.RemoveAll(filepath.Join(demoDir, entry.Name())); err != nil {
        return nil, fmt.Errorf("清理Dem 目录失败: %w", err)
    }
}
```

This clears all managed Demo children while preserving the stable directory path.

## Scenario: 5E Recent Match Query Input Normalization

### 1. Scope / Trigger

- Trigger: 5E import needs to accept both a raw 5E domain ID and the profile share text/link copied from the 5E client.
- Scope: `internal/app.App.ListFiveERecentMatches` Wails binding normalizes the input before config persistence and `internal/fivee` API calls.
- Boundary: UI text input -> Wails binding -> config cache -> 5E match-list HTTP query.

### 2. Signatures

```go
func (a *App) GetFiveEPlayerName() string
func (a *App) ListFiveERecentMatches(playerName string, page int) (*fivee.FiveEMatchListResult, error)
func NormalizePlayerDomainInput(raw string) string
```

### 3. Contracts

- `playerName`: accepts either a raw 5E domain ID such as `12139xi22eza`, a URL/query string containing `domain=12139xi22eza`, or a full client share text containing that URL.
- `ListFiveERecentMatches` must call `fivee.NormalizePlayerDomainInput` before saving `fivee_player_name` and before issuing the remote request.
- `fivee_player_name`: stores the normalized domain ID when extraction succeeds, not the full pasted share text.
- Empty or whitespace-only input remains empty, saves empty, skips remote calls, and returns an empty match list.
- Non-link input with no `domain=` parameter remains trim-normalized and is passed through unchanged for backward compatibility.

### 4. Validation & Error Matrix

- Whitespace-only input -> no error; return `{player_name:"", matches:[]}` and do not call the 5E API.
- Share text/link with non-empty `domain` -> use extracted domain ID for persistence and query.
- Input without `domain` -> use trimmed input as-is.
- 5E API HTTP/JSON/business error -> return the existing wrapped Chinese error from `internal/fivee`.

### 5. Good/Base/Bad Cases

- Good: `【5E对战平台：...】https://csgo.5eplay.com/app/share_loding_type7?domain=12139xi22eza&tab=77` queries with `domain=12139xi22eza` and caches `12139xi22eza`.
- Base: `12139xi22eza` queries and caches `12139xi22eza`.
- Bad: caching the whole share text causes subsequent app startup auto-refresh to send an invalid `domain` query.

### 6. Tests Required

- App-layer regression test:
  - Call `ListFiveERecentMatches` with a full 5E share text.
  - Assert outbound HTTP query `domain` equals the extracted ID.
  - Assert returned `player_name` and cached `fivee_player_name` equal the extracted ID.
- Leaf helper test:
  - Assert raw ID, full share text, query-string-only, and empty input normalize as expected.
- Existing empty-input test must continue to assert no remote call.

### 7. Wrong vs Correct

#### Wrong

```go
playerName = strings.TrimSpace(playerName)
cfg.FiveEPlayerName = playerName
matches, err := fivee.ListRecentMatches(playerName, page)
```

This stores and queries the entire pasted share text.

#### Correct

```go
playerName = fivee.NormalizePlayerDomainInput(playerName)
cfg.FiveEPlayerName = playerName
matches, err := fivee.ListRecentMatches(playerName, page)
```

This keeps all callers on the same normalized 5E domain ID contract.
