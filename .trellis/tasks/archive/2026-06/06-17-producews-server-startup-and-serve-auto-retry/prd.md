# producews server startup and serve auto retry

## Goal

让 `producews` WebSocket server 在启动失败或运行中异常退出时，能自动重试若干次重新建立监听 —— 与**注入到 CS2 进程内的 HLAE plugin DLL**（即代码里的 "game websocket" 客户端）断线重连的行为对称，避免用户因为端口短暂占用或 `Serve()` 意外退出而被迫重启整个 app。

**架构提醒**：连到 `127.0.0.1:4574` 的不是 HLAE 进程本身，而是 HLAE 在 CS2 启动时通过 `csgo/plugin` 搜索路径（见 `internal/producegame`）注入到 CS2 进程里的 plugin DLL。该 plugin DLL 内置周期性重连逻辑，所以 server 端只要在合理时间内恢复 listen，plugin 自然能重新接上，**不需要 server 主动通知任何外部进程**。

## What I already know

### 当前行为（已确认）
- `internal/producews/service.go:171-205` `Service.Start()`:
  - `net.Listen("tcp", "127.0.0.1:4574")` 失败 → 记录 `wsState.LastError`、emit 状态、直接返回 err，**无重试、无退避**。
  - 监听成功后 `go func() { _ = s.server.Serve(listener) }()`，`Serve` 的返回值被 `_ =` 丢弃；半路异常退出时 `wsState.started` 仍为 true 但实际已死，**无自愈、无状态更新**。
- `internal/app/app.go:127-134` 调用 `produceW.Start()` 仅一次，失败只 `wruntime.LogError`，不重试、不阻塞 app 启动。
- 没有任何 `RestartProduceWS` 类型的 Wails 方法暴露在 `internal/app.App`（CLAUDE.md 列出的稳定方法集）。
- 前端 `frontend/src/features/produce/composables/useProducePage.ts:367,383` 订阅 `produce_ws_state_changed` 但 `ProducePage.vue` 只渲染 `queueState.last_error`，**`wsState.LastError` 当前未在 UI 显示**。

### 现有可对称的 retry 模式
- `internal/envsetup/service_actions.go:14` `RetryStartupComponent` 是"用户手动触发组件重试"的范例：emit log → 改 phase → 跑 component action → 重 emit state。可作为 manual retry 入口的设计参考。

### 仓库约束（CLAUDE.md）
- Wails events 稳定列表中 `produce_ws_state_changed` 已存在，可继续复用承载重试中状态。
- 新增的 Wails 公共方法是稳定契约 —— 要慎重命名。
- `Service` 已有 `mu sync.Mutex` 保护内部状态；锁内禁止 blocking I/O；状态变更后必须 `emitState()`。
- 日志：当前 `app.go` 用 `wruntime.LogError` 直接打；`producews` 包内部本身没引入 `internal/logging`。考虑保持现状（不强行接 slog），最小改动。

## Assumptions (temporary)

- 端口 4574 与 HLAE plugin DLL 硬编码对齐，**不更换端口**（自动 fallback 到其他端口会让 plugin 连不上）。
- 用户最常见触发场景：上一次 app 进程未完全释放端口（秒级竞争窗口）；以及 `Serve()` 在某些边缘条件下异常退出。
- HLAE plugin DLL（CS2 进程内）有自己的周期性重连逻辑 —— server 端在合理时间内恢复 listen 即可被 plugin 自动接回，无需主动 push。
- 自愈不能掩盖永久性占用 —— 必须有上限和退避，到达上限后**停下来**让用户看到错误并手动操作。

## Open Questions

- （无 —— 已对齐，等用户最终确认）

## Requirements (evolving)

- 自动重试覆盖两种失败：（a）初始 `net.Listen` 失败；（b）`Serve()` 运行中异常退出。
- 默认参数：最多 **5 次**重试，退避 `500ms → 1s → 2s → 4s → 8s`（总计约 15.5s 窗口）。
- 重试耗尽后**停下**，不再自动重试；`wsState.LastError` 写明已耗尽的原因，要求用户重启 app。
- `Stop()` 被调用时立刻终止 supervisor 循环，不再触发后续重试。
- **不**新增 Wails 公共方法（无 `RetryProduceWS`），不动 `internal/app.App` 已暴露的稳定契约。
- 前端 `ProducePage.vue` 新增一行 `<n-alert v-if="wsState.last_error">` 渲染 `wsState.LastError`，对称已有的 `queueState.last_error` 渲染。
- 后端 `wsState.LastError` 文案在耗尽时明确写"端口 4574 可能被占用，重试 5 次失败，请检查占用进程后重启 app"（在 producews 包内中文文案，与 `failQueueLocked` 现有 "game websocket disconnected" 风格一致 —— 现有为英文，新文案为中文以贴近 UI 用户语境；最终由 trellis-implement 在阶段确认）。
- retry 计数在每次"成功 Listen 且 Serve 开始运行"后重置为 0（每个失败 burst 独立 5 次预算）。

## Decision (ADR-lite)

**Context**: WebSocket server 失败时如何恢复 —— 在自动化程度、用户控制、改动范围之间选择。
**Decision**: 采用 **Option C**：纯自动重试（5 次有界），耗尽后让用户重启 app。不引入手动重试按钮，不新增 Wails 方法。
**Consequences**:
- 优点：改动最小；不破坏稳定 Wails 契约；端口短期占用（最常见场景）可自愈。
- 取舍：端口长期被占用的边缘场景，用户需要重启 app 才能恢复 —— 接受这一成本以换取最小改动面。

## Acceptance Criteria

- [ ] **AC1**：端口被短暂占用（首次 Listen 失败，2s 内释放）时，supervisor 自愈，`wsState.Connected` 在 plugin DLL 重连入后变 true。
- [ ] **AC2**：端口永久被占用时，5 次重试耗尽后 supervisor 退出，`wsState.LastError` 包含明确文案（"端口可能被占用，重试 5 次失败，请重启 app"）。
- [ ] **AC3**：`Serve()` 异常退出后 supervisor 重新 Listen+Serve，HLAE 重连可成功。
- [ ] **AC4**：`Stop()` 调用后 supervisor 立即退出（不被退避 sleep 卡住），不再触发后续重试。
- [ ] **AC5**：`ProducePage.vue` 在 `wsState.last_error` 非空时显示 `<n-alert type="error">`，与现有 `queueState.last_error` 渲染对称。
- [ ] **AC6**：既有 4 个使用 `127.0.0.1:0` 的测试 100% 通过，无需修改测试逻辑。
- [ ] **AC7**：`Service.mu` 锁内不出现 `Listen` / `Serve` / `Sleep` 调用（既有约束保持）。

## Definition of Done

- 新增 4 个单元测试：
  - `TestService_RetriesListenUntilSuccess` —— 占用端口 → 1s 后释放 → supervisor 第 2 次重试成功。
  - `TestService_RetryListenExhausts` —— 端口永久占用 → 5 次后 `wsState.LastError` 写明耗尽。
  - `TestService_ServeExitTriggersRestart` —— 主动关闭 listener 触发 Serve 退出 → supervisor 重新 Listen+Serve → 模拟 plugin 客户端 dialer 重连成功。
  - `TestService_StopAbortsRetry` —— 占用端口让 supervisor 进入 backoff sleep → 调用 `Stop()` → 立刻返回（< 100ms），不等剩余 sleep。
- `go test ./internal/producews` 全绿；`go test ./...` 不退化。
- `cd frontend && npm run build` 通过（UI 改动只在 `ProducePage.vue` 加一行 `<n-alert>`）。
- 行为变更主要 self-contained 在 `producews/service.go`；`internal/app.App` 公共方法签名零变化。

## Out of Scope (explicit, MVP)

- 不实现端口 fallback（不换端口）。
- 不引入 retry 次数/间隔的 config.json 可配置项 —— 用合理默认值硬编码。
- 不引入 `internal/logging` slog 适配器到 `producews` 包（保持现有 emit 风格）。
- 不改 HLAE 客户端侧任何东西。

## Technical Approach (confirmed)

**`Start()` 契约（最小调整，保持向后兼容）**：

```go
func (s *Service) Start() error {
    // 1. 同步首次 Listen（保留旧行为）
    listener, err := net.Listen("tcp", s.addr)
    if err != nil {
        // 首次失败：返回 err 给调用方（行为不变）
        // 同时启动后台 supervisor，从 attempt=1 开始重试
        s.spawnSupervisor(1)
        return err
    }
    // 2. 首次成功：把 Serve() 也放进 supervisor 里跑，便于检测 Serve 异常退出
    s.spawnSupervisor(0, withInitialListener(listener))
    return nil
}
```

**Supervisor 主循环**（goroutine 内运行）：

```text
runSupervisor(startAttempt, optInitialListener):
  attempt := startAttempt
  listener := optInitialListener
  loop:
    if listener == nil:
      listener, err := net.Listen(addr)
      if err != nil:
        attempt++
        if attempt > maxAttempts:
          emit "retry exhausted, restart app required"; return
        emit "listen failed, retrying (n/N)"
        sleep backoff(attempt) (中途响应 stopCh)
        continue
    emit "listening"; attempt = 0
    err := server.Serve(listener)
    listener = nil
    if stopRequested: return
    // Serve 异常退出 → 继续循环重连
    attempt = 1
    emit "serve exited, restarting"
    sleep backoff(attempt) (中途响应 stopCh)
```

**关键约束**：
- 用 `context.Context`（或 `stopCh chan struct{}`）通知 supervisor 退出，`Stop()` 关 ctx → 退出 sleep / loop。
- 退避 sleep 用 `select { case <-time.After(d): case <-ctx.Done(): return }` 包裹，避免 Stop 时被 8s sleep 卡住。
- 不在 `s.mu` 锁内做 sleep / Listen / Serve；仅在快速读写状态时持锁。
- `attempt = 0` 重置发生在每次"Listen 成功 + Serve 开始运行"之后，保证每个失败 burst 独立 5 次预算。

**Backoff 序列**（硬编码常量）：
```go
var backoffSequence = []time.Duration{
    500 * time.Millisecond,
    1 * time.Second,
    2 * time.Second,
    4 * time.Second,
    8 * time.Second,
}
const maxRetryAttempts = 5  // = len(backoffSequence)
```

**这条路径同时覆盖**：
1. 初始 Listen 失败（端口竞争窗口） —— Start() 返回 err，supervisor 异步重试
2. Serve 半路异常退出（listener 被外部关掉等） —— supervisor 检测后重新 Listen+Serve
3. `Stop()` 被显式调用 —— ctx 取消，supervisor 立刻退出，不再重试

## Research References

（本任务暂不需要外部 research，问题边界清楚。）

## Technical Notes

- 关键文件：
  - `internal/producews/service.go`（核心改动）
  - `internal/producews/service_test.go`（新增测试）
  - `internal/app/app.go:127-134`（可能调整：失败时 emit 用户可见状态）
  - 前端 `frontend/src/features/produce/pages/ProducePage.vue`（可选：显示 `wsState.last_error`）
- 现有 Wails event 复用：`produce_ws_state_changed`
- 可能新增 Wails method（待定）：`RetryProduceWS()` 用于到达上限后用户手动触发再次重试
