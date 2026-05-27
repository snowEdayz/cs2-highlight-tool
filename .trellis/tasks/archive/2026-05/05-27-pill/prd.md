# PRD: 制作页面悬浮按钮去掉 pill 容器改为纯悬浮样式

## 问题
`ProducePage.vue` 底部悬浮操作区 `.float-action-bar` 是一个 pill 形状容器
（`border-radius: 24px` + `border: 1px solid #303732`），把按钮包起来，视觉上多余且不好看。

## 目标
去掉 pill 容器的包裹感，让「开始制作」按钮直接悬浮在底部中心，
靠按钮自身圆角 + `box-shadow` 提供层次感，无边框容器。

## 具体变更（仅 CSS，不改逻辑）

`ProducePage.vue` `.float-action-bar` 样式：
- 移除 `border: 1px solid #303732`
- 移除 `border-radius: 24px`
- 移除 `background` / `backdrop-filter`（容器不再需要）
- `padding` 改为 0（容器无背景后不需要内边距）
- 保留 `position: absolute; bottom: 14px; left: 50%; transform: translateX(-50%); z-index: 10; display: flex; align-items: center; gap: 8px;`

在按钮上（通过 `.float-action-bar .n-button` 或为按钮包一个 div）增加：
- `box-shadow: 0 4px 16px rgba(0, 0, 0, 0.5)` 提投影
- 保持按钮自身圆角（Naive UI 默认）

## 验收
- 底部不出现圆形/胶囊边框
- 按钮视觉上有浮起感（投影）
- tag（已连接/已断开）和按钮并排时保持对齐
