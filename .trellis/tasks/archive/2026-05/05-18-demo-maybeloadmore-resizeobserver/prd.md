# 重构 Demo 导入页无限滚动：使用 maybeLoadMore + ResizeObserver 方案

## Goal

重构 `ImportMatchList.vue` 的无限滚动分页加载逻辑，替换当前基于 IntersectionObserver + DOM sentinel 注入（依赖 Naive UI 内部 class）的实现，采用 **`maybeLoadMore()` + `ResizeObserver`** 方案。行为更可预测，不再依赖 Naive UI 内部 CSS class，升级风险更低。

## What I already know

- 当前实现（Session 6 产物）使用 IntersectionObserver + 注入 sentinel `<div>` 到 `.n-scrollbar-content` 内部
- 强依赖 Naive UI 内部 class：`.n-scrollbar-container`、`.n-scrollbar-content`
- 每次 rows 更新后在 `nextTick` 中重新 inject sentinel（Vue 重渲染会移除注入的 DOM）
- 已有 `handleTableScroll` 监听 `on-scroll` 事件检测接近底部
- 已有 `checkShouldAutoLoad` 检查 `scrollHeight <= clientHeight + 1`
- 父层（`useWanmeiImport` / `useFiveEImport`）已有 `loading`、`loadingMore`、`hasMorePages` 兜底

## Requirements

1. **保留**现有的 `on-scroll` "接近底部触发" 逻辑
2. **新增** `maybeLoadMore()` 方法：同时处理 scroll 到底部和内容未撑满两种情况
   - 内容未撑满时：`scrollHeight <= clientHeight + 1` → 触发 load-more
3. 在 `rows` 更新后（`watch` + `nextTick`）调用 `maybeLoadMore()`
4. **新增** `ResizeObserver` 监听容器尺寸变化，变化后调用 `maybeLoadMore()`
5. **新增**本地防抖/互斥标记，避免一帧内重复 emit `load-more`
6. **移除**全部 IntersectionObserver + sentinel DOM 注入代码
7. **不再依赖** `.n-scrollbar-container` / `.n-scrollbar-content` 等 Naive UI 内部 class
8. 生命周期更简单：不需要动态插入/清理 DOM

## Acceptance Criteria

- [ ] 内容不足一屏时自动触发 load-more（`maybeLoadMore` 检测 `scrollHeight <= clientHeight + 1`）
- [ ] 手动滚动到底部时触发 load-more（`handleTableScroll` 检测）
- [ ] 容器 resize 后重新检测是否需要 load-more
- [ ] 防抖/互斥机制有效，一帧内不多次 emit
- [ ] 无 IntersectionObserver 相关代码
- [ ] 无 `.n-scrollbar-container` / `.n-scrollbar-content` 查询
- [ ] 无动态 DOM 插入/清理
- [ ] 父层 `loading`/`loadingMore`/`hasMorePages` 仍作为兜底
- [ ] `go test ./...` 后端测试通过
- [ ] `cd frontend && npm run build` 前端构建通过

## Out of Scope

- 不改变 `useWanmeiImport` / `useFiveEImport` / `WanmeiImport.vue` / `FiveEImport.vue` 的接口
- 不改变后端逻辑

## Technical Notes

- 修改文件仅限：`frontend/src/features/import/components/ImportMatchList.vue`
- 使用 Vue `ResizeObserver` 通过 `ref` 获取容器 DOM 后挂载 `ResizeObserver`
- 防抖使用 `let loadMorePending = false` 标记 + `requestAnimationFrame` 合并
- `maybeLoadMore()` 调用链：
  1. 检查防抖标记
  2. 获取 scroll container（通过 `on-scroll` 事件捕获的 `target` 引用）
  3. 检查 `loading` / `loadingMore` / `hasMorePages`
  4. 执行 scroll 底部检测或内容未撑满检测
  5. emit "load-more"
