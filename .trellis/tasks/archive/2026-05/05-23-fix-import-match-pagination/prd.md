# fix infinite match auto-pagination on import pages

## Goal

修复导入页中完美对战平台与 5E 对战平台战绩列表的分页加载行为：首屏先请求第一页；如果此时列表还没有出现纵向滚动条，则自动继续请求后续页，直到列表出现滚动条或没有更多数据；一旦出现滚动条，后续分页仅在用户滚动到当前列表末尾时触发。同时避免通过硬编码最大高度制造多余空白。

## What I already know

* 问题范围位于 `frontend/src/features/import/**`。
* 两个平台共用 `ImportMatchList.vue` 处理表格滚动与 `load-more` 触发。
* 当前 `ImportMatchList.vue` 除了“滚动接近底部”判断，还包含 `isContentNotFilling()` 兜底，会在列表内容没有撑满容器时自动继续请求下一页。
* `WanmeiImport.vue` 与 `FiveEImport.vue` 都把列表区域做成 `flex: 1`，会在大屏下继续拉高可视区域。
* 完美平台后端单页大小固定为 11 条（`internal/wanmei/client.go` 中 `matchPageSize = 11`），因此在高分辨率下更容易出现首屏不溢出的问题。

## Assumptions (temporary)

* 用户接受“首屏自动补页”，但只接受一个有限目标：补到列表出现滚动条为止。
* 不应该通过固定表格最大高度解决问题，因为这会直接影响页面版式并引入空白区域。

## Open Questions

* 暂无阻塞问题，先按现有需求直接修复。

## Requirements (evolving)

* 完美与 5E 战绩列表初次加载后，如果当前内容还未产生纵向滚动条，可以自动继续请求下一页。
* 自动补页必须在“列表出现纵向滚动条”后立即停止。
* 一旦列表已可滚动，后续下一页请求只能由用户实际滚动到末尾附近触发。
* 页面应恢复自适应高度，不通过固定最大高度制造底部空白。
* 该修复应同时作用于完美与 5E，因为二者共用同一列表组件。

## Acceptance Criteria (evolving)

* [ ] 打开完美或 5E 导入页并点击刷新/查询时，至少请求第一页；若首屏未出现滚动条，则自动继续拉页直到出现滚动条或无更多数据。
* [ ] 一旦滚动条出现，不会继续因为数据刷新或尺寸变化而自动连刷多页。
* [ ] 用户将列表滚动到末尾附近时，才会触发滚动场景下的下一页请求。
* [ ] 页面不再因为固定表格高度而在列表下方出现明显空白。
* [ ] `cd frontend && npm run build` 通过。

## Definition of Done (team quality bar)

* 相关前端代码已更新并通过构建校验
* 不修改自动生成文件
* 不破坏完美/5E 现有导入与分页状态管理

## Out of Scope (explicit)

* 不改后端分页接口或返回结构
* 不新增新的分页 UI（例如“加载更多”按钮）
* 不修改其它导入页面或启动页布局

## Technical Notes

* 关键文件：
  * `frontend/src/features/import/components/ImportMatchList.vue`
  * `frontend/src/features/import/pages/WanmeiImport.vue`
  * `frontend/src/features/import/pages/FiveEImport.vue`
  * `frontend/src/features/import/composables/useWanmeiImport.ts`
  * `frontend/src/features/import/composables/useFiveEImport.ts`
* 这次调整后的目标不是完全删除自动补页，而是把它改成“只补到出现滚动条为止”的有限策略。
* 需要区分两种触发来源：
  * 自动补页：仅在当前列表没有纵向滚动条时触发
  * 滚动分页：仅在已有纵向滚动条且用户滚动接近底部时触发
