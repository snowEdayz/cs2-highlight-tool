# 优化后端项目结构 — 模块解耦与文件精简

## Goal

在不改变任何功能的前提下，优化 `internal/app` 包的代码结构——将 3 个最大的源文件拆分为职责单一的多个文件，提升可维护性和可读性。

## 现状

| 文件 | 行数 | 函数数 | 问题 |
|------|------|--------|------|
| `internal/app/produce_session.go` | 1,557 | 54 | 状态管理 + 取帧文件历史 + 合并队列 + 媒体合成 + 游戏配置管理 + 临时文件清理，6 个独立职责混在一个文件 |
| `internal/app/app_clips.go` | 1,112 | 29 | 剪辑设置 CRUD + 插件 JSON 生成 + HLAE 启动，3 个独立职责混在一起 |
| `internal/app/app_edit.go` | 935 | 30 | 核心拼接逻辑 + 进度追踪器 + FFmpeg 命令运行，3 个独立职责混在一起 |

## 拆分方案

所有新文件放在 `internal/app/` 包内，使用 `produce_*`、`clip_*`、`edit_*` 前缀。

### 1️⃣ `produce_session.go` → 5 个文件

| 新文件 | 职责 | 关键内容 |
|--------|------|----------|
| `produce_session.go` (保留收缩) | 核心状态定义 + session 生命周期 | `produceSessionState`, `produceSessionRuntime`, `startProduceSessionWorker`, `stopProduceSessionWorker`, `runProduceSessionWorker`, `canStopProduceSession`, `isSessionWorkDrained` |
| `produce_takefile.go` | 取帧文件与历史记录管理 | `ProduceTakeFile`, `ProduceTakeFileSnapshot`, `ProduceHistoryItem`, `ProduceHistorySnapshot`, `GetProduceTakeFiles`, `ExportProduceHistoryVideos`, `OpenProducedClipInFolder`, `resetProduceTakeFiles`, `updateTakeFileEntry`, `addProduceHistoryEntry`, `addEditedHistoryEntry` |
| `produce_merge.go` | 合并队列 + 音视频合成 | `pendingCompletedTake`, `mergeTask`, `enqueueCompletedTakes`, `dispatchMergeTasks`, `mergeWorker`, `handleMergeTask`, `mergeTakeVideoAudio`, `waitForTakeFilesReady`, `probeTakeFilesReadable` |
| `produce_gameconfig.go` | 游戏配置备份与恢复 | `gameInfoSessionState`, `pluginDLLSessionState`, `prepareGameInfoForProduce`, `preparePluginDLLForProduce`, `forceRestoreGameInfoForProduce`, `forceRestorePluginDLLForProduce`, `forceRestoreProduceEnvironmentForProduce` |
| `produce_cleanup.go` | 临时文件清理工具 | `cleanupProduceTemporaryFiles`, `removeFileIfExists`, `removeMuxTmpFiles`, `removeEmptyDirUpward`, `requestCloseCS2Process` |

### 2️⃣ `app_clips.go` → 3 个文件

| 新文件 | 职责 | 关键内容 |
|--------|------|----------|
| `clip_settings.go` | 剪辑设置与动作设置 CRUD | `ClipSettings`, `ClipActionSettings`, `GetClipSettings`, `SaveClipSettings`, `GetClipActionSettings`, `SaveClipActionSettings`, `PickRecordOutputDir` |
| `plugin_generate.go` | 插件 JSON 生成逻辑 | `GeneratePluginJSONBatch`, `GeneratePluginJSONBatchAndLaunchHLAE`, `GeneratePluginJSON`, `generatePluginJSONInternal`, `filterItemsByHistory`, `normalizeSelectedItems`, `normalizeClipSettings`, `buildProduceHistoryKey`, `registerProduceKillSnapshot` |
| `hlae_launch.go` | HLAE 启动与命令行构建 | `launchJobContext`, `launchHLAEGame`, `buildHLAECommandLine`, `resolveCS2ExeForLaunch` |

### 3️⃣ `app_edit.go` → 3 个文件

| 新文件 | 职责 | 关键内容 |
|--------|------|----------|
| `edit_concat.go` (保留收缩) | 核心拼接 + 转场逻辑 | `ConcatEditClips`, `concatSimple`, `concatWithTransitions`, `resolveEditClips`, `normalizeEditTransitions`, `applyFadeTransition`, `concatHardCutPair`, `editComposeStageCount` |
| `edit_progress.go` | 合成进度追踪器 | `composeProgressTracker`, `newComposeProgressTracker`, `stageStart`, `stageProgress`, `stageDone`, `fail`, `complete`, `emit`, `clampProgressPercent` |
| `edit_ffmpeg.go` | FFmpeg 命令运行 + 编码参数 | `runFFmpegCommandWithProgress`, `parseFFmpegProgressSeconds`, `resolveEditEncodeSettings`, `buildEditRetryProfiles`, `ProbeClipDuration`, `probeDurationByFFprobe`, `resolveFFprobeExe`, `resolveEditOutputPaths`, `withFFmpegProgressArgs` |

## 约束（不改变的事项）

- **不修改**任何 Wails 暴露方法签名（`func (a *App) MethodName` 的签名不变）
- **不修改** `App` 结构体字段定义
- **不修改**与前端交互的事件名/状态枚举
- **不修改**类型字段的 JSON tag 或序列化行为
- **不修改**测试文件——仅移动函数位置，不改变函数体代码
- **不修改** `cli` 命令/其他包的导入路径

## Acceptance Criteria

- [ ] `produce_session.go` 从 1,557 行缩减到 ~250 行，仅保留会话生命周期逻辑
- [ ] `app_clips.go` 从 1,112 行缩减到 0 行（完全拆分为 3 个新文件）
- [ ] `app_edit.go` 从 935 行缩减到 ~400 行，仅保留拼接逻辑
- [ ] 新增的 8 个文件每个职责单一，文件名体现用途
- [ ] `go build ./...` 通过
- [ ] `go test ./internal/app/...` 全部通过
- [ ] 所有 Wails 暴露方法签名不变
- [ ] 前端编译（`cd frontend && npm run build`）通过

## Definition of Done

- [ ] 后端编译通过：`go build ./...`
- [ ] 后端测试通过：`go test ./...`
- [ ] 前端编译校验通过：`cd frontend && npm run build`
- [ ] 前端 Wails 绑定不变（自动生成的 `frontend/wailsjs/**` 无变化）
- [ ] AGENTS.md 同步更新（如果文件路径映射有变化）

## Out of Scope

- 不动 `internal/envsetup`, `internal/release`, `internal/producews`, `internal/clipsjson` 等其他包
- 不动测试文件的结构（测试代码保持原样，即使它们引用已移动的函数）
- 不做功能重构或行为变化
- 不修改 `config.json` 持久化字段
- 不改动 Gofmt 之外的代码风格

## Technical Notes

- 所有新文件属于同一个 Go 包 `app`，因此跨文件引用无需导入
- `App` 结构体在 `app.go` 定义，所有方法使用 `(a *App)` receiver，方法可以分布在不同文件中
- `var` 定义的全局测试替身（如 `ffmpegCommand`, `launchHLAECommand`）随引用函数移动
- `const` 常量（如 `produceMergeWorkers`）随使用场景移动
- `sync.Mutex` 保护的 `produceStateMu` 位于 `App` 结构体，跨文件共享
