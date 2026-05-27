# 剪辑页可用片段列表按 DEM+回合折叠分组，每组提供一键全部加入序列

## Goal

`EditPage.vue` 左侧"可用片段"面板目前是一个扁平列表，当用户录制了多个 DEM 的片段时，
所有片段混在一起，顺序不清晰。改造为：按 DEM 文件折叠分组，每个 DEM 内再按回合折叠，
每个 DEM 组提供独立的"全部加入序列"按钮，让用户可以按 DEM 顺序逐个导入。

## Requirements

* 左侧"可用片段"列表改为 Collapse，按 `demo_path` 分组
* 每个 DEM 折叠项：
  * 标题显示 DEM 文件名（basename）
  * header-extra 显示该 DEM 的片段数量标签 + "全部加入序列"按钮（click.stop 阻止折叠触发）
  * 内部按回合再折叠（复用 ClipLibrary.vue 中已有的 splitKillsByRound + roundGroupsForDemo 逻辑）
  * 每个回合折叠内显示各 clip 条目，保留原有的单条"添加到序列"按钮
* 顶部面板头的全局"一键导入全部"按钮保留，依旧对所有 DEM 生效
* 新 DEM 组自动展开（watch produceClipsByDemo）
* 每个 DEM 组有独立的 loading 状态（addingAllByDemo）
* 每个回合组默认展开
* 全局 addingAll 为 true 时，各 DEM 的"全部加入序列"按钮禁用
* 全局"一键导入全部"改为**按 DEM 顺序依次加入**：先 orderByView 处理第一个 DEM 全部片段，再处理第二个 DEM，以此类推（不再跨 DEM 混合排序）

## Acceptance Criteria

* [ ] 可用片段面板显示 DEM 折叠列表，每个 DEM 有独立的"全部加入序列"按钮
* [ ] 点击 DEM 组的"全部加入序列"按钮，仅将该 DEM 的所有片段按 orderByView+sortByTick 加入序列
* [ ] 全局"一键导入全部"保留且行为不变
* [ ] 多个 DEM 时，各组独立展示、互不干扰
* [ ] ts build 无错误

## Definition of Done

* Frontend build (`npm run build`) 通过，无 TS 错误

## Technical Approach

**只修改 `EditPage.vue`**（不改 ClipLibrary.vue / useEditPage.ts / useEditState.ts）。

在 EditPage.vue 内：
1. 新增 `produceClipsByDemo` computed（按 demo_path 分组 produceClipItems）
2. 新增 `sourceRoundGroupsByDemo` computed（复用 splitKillsByRound，按回合分组）
3. 新增 refs: `sourceDemoExpanded`, `sourceDemoExpandedInitialized`, `sourceRoundExpandedByDemo`, `addingAllByDemo`
4. 新增函数：`getSourceDemoExpanded`, `handleSourceDemoExpanded`, `getSourceRoundExpanded`,
   `handleSourceRoundExpanded`, `sourceRoundTitle`, `addAllFromDemo`, `isAddingAllForDemo`, `basename`（inline）
5. 模板：将原 `.source-list` 的扁平 v-for 改为 n-collapse + n-collapse-item（DEM） + n-collapse（round）

**样式**：`.source-list` 改为普通 overflow-y 容器，collapse 内嵌 `.source-item` 沿用原有样式。

## Out of Scope

* ClipLibrary.vue 不改动（该组件在 EditPage 中未使用，保持原样）
* 不改变序列面板（右侧）
* 不新增后端 API

## Technical Notes

* 相关文件：`frontend/src/features/edit/pages/EditPage.vue`
* `addFromHistory` / `addAllFromHistory` / `orderByView` / `sortByTick` 逻辑复用，不改
* `addAllFromDemo` 与 `addAllFromHistory` 逻辑相同，仅限定到单个 DEM 的 items
* ClipLibrary.vue 中已有完整的 DEM+round 折叠实现可参考
* Naive UI 组件自动导入，不需要手动 import
