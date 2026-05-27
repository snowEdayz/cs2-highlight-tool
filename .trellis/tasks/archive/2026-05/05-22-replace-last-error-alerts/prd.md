# replace persistent frontend lastError displays with alerts

## Goal

检查并修正前端中用 `lastError` 常驻展示错误信息的位置。导入相关错误不应作为页面内长期文本残留，应改为使用 Naive UI 的通知式反馈，让错误在用户触发失败时即时出现，随后自动消失或由 Naive UI 控制展示生命周期。

## What I already know

* 用户明确要求创建任务，并检查前端代码中使用 `lastError` 展示错误信息的位置。
* 根级与 `frontend/AGENTS.md` 已读取；本次范围属于 `frontend/**`，完成后需要执行 `cd frontend && npm run build`。
* `rg` 检索到页面上直接渲染 `lastError` 的位置：
  * `frontend/src/features/import/pages/ImportActions.vue`
  * `frontend/src/features/import/pages/WanmeiImport.vue`
  * `frontend/src/features/import/pages/FiveEImport.vue`
* `useWanmeiImport.ts` 已经使用 `useMessage()` 做状态提示，但仍把加载、分页、手动导入、下载导入错误写入 `lastError`。
* `useFiveEImport.ts` 目前没有使用 `useMessage()`，错误主要写入 `lastError`。
* 其他 `n-alert` 用法如 `StartupWizard.vue` 的 fatal error、`ProducePage.vue` 的运行态错误不属于 `lastError` 常驻展示问题，暂不纳入。

## Assumptions

* “native UI” 指当前项目使用的 Naive UI。
* 用户希望去掉页面内长期驻留的 `lastError` 文本，而不是禁止所有 `n-alert` 组件。
* 对导入页这种可恢复、由用户操作触发的错误，推荐使用 Naive UI `message.error()`；它符合现有 `useMessage()` 使用模式，比页面内 alert 区块更不占布局。

## Requirements

* 移除导入页中 `lastError` 的页面常驻渲染。
* 文件导入失败、完美/5E 列表加载失败、加载更多失败、手动输入为空/重复、单场导入失败等错误应通过 Naive UI 通知展示。
* 不改变 Wails 暴露方法、事件名、状态枚举或后端契约。
* 保留现有队列、分页、导入成功回调和下载进度行为。
* 避免修改自动生成文件和 `en-US.json`。

## Acceptance Criteria

* [ ] `rg -n "lastError" frontend/src` 不再出现页面模板常驻渲染 `lastError` 的用法。
* [ ] 导入入口、完美导入页、5E 导入页的错误路径通过 Naive UI `message.error()` 展示。
* [ ] 前端构建通过：`cd frontend && npm run build`。
* [ ] 不引入新的后端接口、事件或状态契约变更。

## Definition of Done

* Frontend Required Check 通过。
* 代码遵守 `frontend/AGENTS.md` 和 `.trellis/spec/frontend` 约定。
* 如无新长期约定，不更新 AGENTS 或 spec。

## Out of Scope

* 不调整启动向导 fatal error、produce 队列错误、settings 保存提示等非 `lastError` 展示。
* 不重构导入页布局或导入队列实现。
* 不修改后端。

## Technical Notes

* 相关规范：
  * `AGENTS.md`
  * `frontend/AGENTS.md`
  * `.trellis/spec/frontend/index.md`
* 相关代码：
  * `frontend/src/features/import/pages/ImportActions.vue`
  * `frontend/src/features/import/pages/WanmeiImport.vue`
  * `frontend/src/features/import/pages/FiveEImport.vue`
  * `frontend/src/features/import/composables/useWanmeiImport.ts`
  * `frontend/src/features/import/composables/useFiveEImport.ts`
