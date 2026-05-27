# 修复 Demo 导入页无限滚动在数据不足时的分页加载问题

## Goal

修复完美世界 / 5E 对战平台的战绩列表分页加载问题：当初始加载的数据行数不足以填满可视区域、未触发滚动条时，用户无法加载后续页面。

## What I already know

- 完美世界 (`WanmeiImport.vue`) 和 5E (`FiveEImport.vue`) 都使用 `ImportMatchList.vue` 组件来展示战绩列表
- `ImportMatchList.vue` 通过监听 `n-data-table` 的 `scroll` 事件，在滚到底部时发射 `load-more` 事件
- 核心逻辑 (`handleTableScroll`)：当 `scrollHeight <= clientHeight` 时直接 return，不会触发 `load-more`
- 两个 imports 页面都使用 `router-view` 路由展示在 `ImportPage.vue` 的右侧面板（`import-action-panel`），该面板宽度可拖拽调整
- 后端分页：`ListWanmeiRecentMatches(page)` / `ListFiveERecentMatches(playerName, page)`，每页返回固定数量（由后端决定）
- 错误状态：`lastError` 展示，有 `lastError` 不会自动清除

## Constraints

- 必须保持现有 `ImportMatchList.vue` 的接口签名不变（`columns`/`rows`/`loading` props, `load-more` emit）
- 两个 import composable (`useWanmeiImport.ts` / `useFiveEImport.ts`) 已有 `hasMorePages`、`currentPage`、`loadingMore`、`loadMoreMatches()` 等分页状态/方法 — 不需要改动
- `ImportMatchList.vue` 的 `handleTableScroll` — `scrollHeight <= clientHeight + 1` 检测到无需滚动时提前 return，是这个 bug 的直接原因
- 现有前端规格要求：Vue 3 Composition API, Naive UI, scoped styles

## Open Questions

（已收敛，无未决问题）

## Decision (ADR-lite)

**Context**: `ImportMatchList.vue` 的无限滚动依赖 `n-data-table` 的 `scroll` 事件和 `scrollHeight <= clientHeight` 检测，当表格数据少、无滚动条时该检测导致 return，用户无法加载下一页。

**Decision**: 采用方案 A（IntersectionObserver 哨兵）

**Technical approach**:
1. 保留现有的 `on-scroll` 事件监听（用于有滚动条时滚到底部自动加载），移除 `scrollHeight <= clientHeight` 的 guard
2. 在 `onMounted` 中，通过 `n-data-table` 的 ref 获取内部 scroll 容器（`.n-scrollbar-container`），动态注入一个 1px 高的哨兵 `<div>` 作为空元素3. 使用 IntersectionObserver 监听哨兵，`root: scrollContainer`，当哨兵进入可视区时 emit `load-more`
4. 该哨兵在无滚动条时始终在可视区内 → 自动触发分页加载直到填满或数据耗尽；有滚动条时用户滚到底部哨兵进入可视区 → 触发
5. `onBeforeUnmount` 清理 observer 并移除哨兵元素

**Consequences**:
- 不改变 `ImportMatchList.vue` 的 props/emit 接口
- 不依赖后端行为
- 哨兵元素通过 DOM API 注入，每次 `n-data-table` 重渲染后需要重新注入（通过 `watch(rows)` + `nextTick` 重新检查并注入）

## Requirements

- **核心修复**：当初始加载的数据行数不足以触发滚动条时，用户仍能加载后续页面
- **保留原有机制**：滚动到底部自动加载行为不受影响（scroll 事件监听保留）
- **无数据/最后一页不触发**：不再触发加载更多
- **加载状态正确**：loading/loadingMore 状态正确处理，不重复请求
- **不改变接口**：`ImportMatchList.vue` 的 `columns`/`rows`/`loading` props 与 `load-more` emit 签名不变
- **TypeScript 编译通过**：`cd frontend && npm run build` 通过

## Acceptance Criteria

- [ ] 初始加载数据不足填满表格时，自动触发加载下一页，直到填满或数据耗尽
- [ ] 滚动到底部自动加载原有行为不受影响
- [ ] 无数据或加载到最后一页时不再触发加载（由父组件的 `hasMorePages` 控制）
- [ ] 加载中状态（loading/loadingMore）正确处理，不重复请求
- [ ] observer/sentinel 在组件卸载时正确清理（无内存泄漏）
- [ ] TypeScript 编译无错误，build 通过

## Definition of Done

- `cd frontend && npm run build` 通过
- 代码修改限于 `ImportMatchList.vue`（只改这一个文件）
- 在典型场景下验证两种平台（完美、5E）的导入功能正常

## Out of Scope

- 不变更后端分页接口
- 不修改自动生成文件（`frontend/wailsjs/**`）
- 不做全页重写
- 不修改父组件（`useWanmeiImport.ts` / `useFiveEImport.ts` / `WanmeiImport.vue` / `FiveEImport.vue`）

## Technical Notes

### 问题重现

1. 用户进入完美世界/5E 导入页
2. 后端返回初始 10-20 条记录
3. 如果用户窗口高度足够大（或数据行数少），表格内容高度 ≤ 容器高度
4. 没有滚动条 → `scroll` 事件不会触发 → `load-more` 永不发射
5. 用户没有任何方式加载下一页

### 现有代码关键路径

- `frontend/src/features/import/components/ImportMatchList.vue` — 直接问题文件
- `frontend/src/features/import/composables/useWanmeiImport.ts` — `loadMoreMatches()`, `hasMorePages`
- `frontend/src/features/import/composables/useFiveEImport.ts` — `loadMoreMatches()`, `hasMorePages`
- `frontend/src/features/import/pages/WanmeiImport.vue` — 模板中 `@load-more="loadMoreMatches"`
- `frontend/src/features/import/pages/FiveEImport.vue` — 模板中 `@load-more="loadMoreMatches"`

### `ImportMatchList.vue handleTableScroll` 当前代码

```typescript
function handleTableScroll(event: Event) {
  const target = event.target as HTMLElement | null;
  if (!target) return;
  if (target.scrollHeight <= target.clientHeight + 1) return;  // ← BUG: 无滚动条时永不触发
  if (!isNearScrollBottom(target)) return;
  emit("load-more");
}
```
