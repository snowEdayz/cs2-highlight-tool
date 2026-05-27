# 制作页面：游戏客户端检测重设计 + 悬浮开始按钮

## Goal

当前制作页的"开始制作"在用户选择很多片段时需要滚动到底部才能点击，体验差。
同时，现有"检测游戏客户端是否运行"功能包含一个从软件端发送关闭信号的按钮，但该信号实际上无法关闭客户端，功能无效。
本任务做两件事：
1. 移除"从软件关闭客户端"功能，改为展示客户端运行状态 + 刷新按钮，引导用户手动关闭后刷新确认；开始按钮仅在所有客户端关闭后才可点击。
2. 将"开始制作"按钮改为悬浮在页面底部中央，无需滚动随时可见。

## What I already know

- `PlatformClientCheckModal.vue` — 当前实现为弹窗，包含"关闭"和"关闭所有"按钮，调用后端 `RequestClosePlatformClient`，并有发送中 spinner、超时错误等状态
- `usePlatformClientCheck.ts` — composable 含 `checkAll`、`requestClose`、`closeAll`、`closingMap`、`closeErrorMap`；`checkAll` 返回 `boolean`（是否全部关闭）
- `useProducePage.ts:541` — `generateAndLaunch` 先调 `platformCheck.checkAll()`，若未全关则显示弹窗
- `ProducePage.vue` — 开始按钮在 `.card-body` 内（overflow: auto），需要滚动到底部才能点击
- 后端检测方法：`CheckPlatformClients`；关闭方法：`RequestClosePlatformClient`（实际无法生效）
- 两个被检测的客户端：暂不确认 exe 名，从后端接口返回

## Assumptions (temporary)

- 移除 `RequestClosePlatformClient` 相关调用（前端侧），后端方法可保留或废弃
- 刷新操作 = 重新调用 `CheckPlatformClients`，更新状态
- 检测时机保持现有逻辑（点击开始制作时触发），而非页面加载时自动检测

## Open Questions

（已全部解决）

## Requirements

**客户端检测流程：**
- 点击"开始制作"时触发 `CheckPlatformClients`，若有客户端运行则弹出简化弹窗
- 弹窗内容：客户端状态列表（运行中/已关闭标签）、手动关闭提示文字、"刷新"按钮
- "刷新"按钮重新调用 `CheckPlatformClients` 更新状态，带 loading 状态防止重复点击
- 弹窗底部保留"取消"按钮 + "确认"按钮（仅 `allClosed` 时可点击）
- 移除所有"从软件关闭"相关逻辑：`requestClose`、`closeAll`、`closingMap`、`closeErrorMap`、关闭中 spinner、错误提示

**悬浮开始按钮：**
- 将"开始制作"按钮及"插件已连接/未连接"状态标签移到 `.card-body` 外，用绝对定位悬浮在卡片底部中央
- `.card-body` 加 `padding-bottom` 防止内容被按钮遮挡
- 按钮禁用条件与当前保持一致：`!hasPendingMaterials || generatingAndLaunching || generatingConfigOnlyLoading || queueState.running`
- Debug 用的"仅生成 JSON"按钮保留在内容区域内（非核心功能）

## Acceptance Criteria

- [ ] 检测到客户端运行时弹出简化弹窗，无"关闭"/"关闭所有"按钮
- [ ] 弹窗内有"刷新"按钮，点击后重新检测并更新状态，刷新中按钮 loading
- [ ] 所有客户端关闭后"确认"按钮可点击
- [ ] 用户点取消后弹窗关闭，不开始制作
- [ ] 所有客户端未运行时点击"开始制作"直接进入制作流程（无弹窗）
- [ ] "开始制作"按钮悬浮在卡片底部中央，滚动内容时按钮不消失
- [ ] 制作运行中时，"插件已连接/未连接"标签紧靠悬浮按钮旁显示
- [ ] `npm run build` 通过，无 TypeScript 错误

## Definition of Done

- `cd frontend && npm run build` 通过（type-check + build）
- 前端无 lint/type 错误
- 无手动编辑 wailsjs/ 目录

## Out of Scope (explicit)

- 后端 `RequestClosePlatformClient` 方法本身不做修改（前端不再调用即可）
- 不改变检测所依赖的后端接口（`CheckPlatformClients`）
- 不改变客户端列表的来源（仍由后端决定检测哪些进程）

## Technical Notes

- 悬浮按钮实现思路：将按钮移到 `.card-body` 外，放在 `produce-card` 内用 `position: absolute; bottom: 16px; left: 50%; transform: translateX(-50%)` 实现悬浮，同时给 `.card-body` 加 `padding-bottom` 避免内容被遮挡
- `usePlatformClientCheck.ts` 模块级共享 state（`statuses`、`closingMap`、`closeErrorMap`）需清理掉不再使用的字段
- 当前 `PlatformClientCheckModal.vue` 可以整体重写（逻辑简单很多）或原地精简
