# fix: gate startup flow on self update

## Goal

软件启动进入环境准备页后，必须先完成软件自身版本检查。只要检测到软件自身有新版本，后续 HLAE、插件、FFmpeg、CS2 路径等组件安装/更新/检测流程都不应开始；用户应优先点击“立即更新”完成软件更新。只有当前软件版本确认已是最新版本后，才继续执行组件环境准备流程。

## What I Already Know

* 用户观察到：启动后检测到软件自身需要更新时，界面显示“立即更新”按钮，但如果其他组件还未下载，软件会优先下载组件。
* 用户期望：自更新检查优先；发现新版本时阻断后续组件流程，让用户先更新软件。
* 当前后端 `runTasksDefault` 同时启动 `checkSelfUpdate` 与组件检查 goroutine，导致组件下载和自更新检查并行发生。
* 当前前端 `busy` 包含 `state.running`，自更新按钮在启动检查运行期间会被禁用。
* 当前 `CanEnterMain` 已要求 `!SelfUpdate.Available`，因此“有软件更新时不能进入主页面”这个约束已经存在。

## Assumptions

* 自更新检查失败不是“发现新版本”，因此仍沿用现有语义：记录失败/警告后继续环境检查，避免网络异常时完全卡住启动。
* 不新增 Wails 方法、事件名、组件 ID、状态枚举或阶段枚举；优先通过现有 `SelfUpdateState`、`startup_state_changed`、`download_progress` 和组件状态表达流程。
* 组件步骤在自更新待处理时保持 `pending`，不显示下载/安装进度。

## Requirements

* `RunStartupChecks` 在统一 Release 快照可用后，先把 `self_update` 置为 `checking` 并同步执行软件版本检查。
* 如果自更新检查结果为 `available=true` 且 `status=needs_action`：
  * 立即结束本轮启动检查任务。
  * 不启动 HLAE、插件、FFmpeg、CS2 的组件检查/下载/安装任务。
  * 组件步骤保持 `pending`（除非本轮开始前已有安全保留的状态；实现时应避免展示“已开始下载/安装”的假象）。
  * `state.running` 最终回到 `false`，使“立即更新”按钮可点击。
  * `can_enter_main=false`，`phase` 保持能表达仍在环境准备中的状态（建议继续使用 `running_tasks`，不新增阶段）。
* 如果自更新检查结果为已是最新版本：
  * 继续执行现有组件检查流程。
  * 组件任务可继续并发执行，以保留现有启动性能。
* 如果自更新检查失败：
  * 维持现有容错策略，继续组件环境检查。
  * 日志应继续明确说明“软件更新检查失败，将继续环境检查”。
* `ApplySelfUpdate` 行为保持不变：点击后下载更新、进入 installing、触发重启替换。
* 前端只需要消费后端 state；如后端已保证发现更新时 `running=false`，前端不需要绕过全局 busy 规则。

## Acceptance Criteria

* [ ] 当统一 Release 响应包含高于当前版本的软件更新时，`RunStartupChecks()` 返回的 `self_update.available=true`、`self_update.status=needs_action`、`running=false`、`can_enter_main=false`。
* [ ] 同一场景下 HLAE、插件、FFmpeg、CS2 组件检查函数未被调用，步骤未进入 downloading/installing。
* [ ] 同一场景下前端“立即更新”按钮显示且不因 `state.running` 被禁用。
* [ ] 当软件已是最新版本时，启动流程继续执行现有组件检查；所有组件 ready 后仍可进入主页面。
* [ ] 当软件更新检查失败时，流程继续执行组件检查，保留现有离线/失败容错语义。
* [ ] 新增或更新后端测试覆盖“发现软件更新时阻断组件任务”。
* [ ] 如涉及前端按钮策略调整，新增或更新前端 display/composable 相关测试；否则至少通过构建验证。

## Definition of Done

* `go test ./...` 通过。
* 若修改前端文件，`cd frontend && npm run build` 通过。
* 涉及启动状态机的测试重点覆盖 `internal/envsetup`。
* 不修改自动生成文件：`frontend/wailsjs/**`、`frontend/src/auto-imports.d.ts`、`frontend/src/components.d.ts`。
* 不新增与现有契约冲突的新状态名、阶段名或组件 ID。

## Out of Scope

* 不改变自更新下载/替换机制。
* 不新增“跳过软件更新继续安装组件”的入口，除非用户后续明确要求。
* 不改变统一更新源、GeoIP、组件下载回退策略。
* 不调整主页面进入条件之外的业务流程。

## Technical Notes

* 代码调查记录见 `research/startup-self-update-flow.md`。
* 主要改动预计集中在 `internal/envsetup/service_startup.go` 和相关测试。
* 若实现时发现前端按钮仍因状态竞态不可用，再局部检查 `frontend/src/features/startup/composables/useStartupWizard.ts` 与 `StartupWizard.vue`。
