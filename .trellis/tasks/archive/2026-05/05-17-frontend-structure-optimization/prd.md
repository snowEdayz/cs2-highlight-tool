# 优化前端项目结构 — 模块解耦与文件精简

## Goal

在不改变任何功能的前提下，优化 `frontend/src/` 的代码结构——拆分最大的 .vue 页面（脚本逻辑提取为 composable/子组件）、拆分臃肿的 composable 文件和 shared types 文件，提升可维护性和可读性。

## 现状

| 文件 | 行数 | 问题 |
|------|------|------|
| `ProducePage.vue` | 1,042 | 脚本 708 行，含大量 computed + 函数 + interfaces，远超页面职责 |
| `FiveEImport.vue` | 740 | 脚本 482 行，列表+分页+下载流程混在页面 |
| `WanmeiImport.vue` | 718 | 脚本 507 行，同上 |
| `EditPage.vue` | 827 | 脚本 382 行，页面逻辑可提取 |
| `TopBar.vue` | 686 | 导航 + 历史下拉 + 窗口控制混在一个文件 |
| `useImportDemos.ts` | 389 | 14+ 个函数，数据获取+选择+格式化混杂 |
| `useStartupWizard.ts` | 341 | 显示逻辑与编排逻辑混杂 |
| `shared/types.ts` | 356 | 37 个接口全在一个文件 |

## 拆分方案

### 组 1 — 大 .vue 页面提取

#### 1️⃣ `ProducePage.vue` (1,042 行 → ~400 行)

| 新/改文件 | 职责 |
|-----------|------|
| `ProducePage.vue` (保留缩减) | 模板 + 轻量胶水 |
| `useProducePage.ts` 🆕 | 提取所有 computed、refs、handler |
| `ProduceTakeTable.vue` 🆕 | 取帧计划表格子组件 |

#### 2️⃣ `FiveEImport.vue` (740 行) + `WanmeiImport.vue` (718 行)

| 新/改文件 | 职责 |
|-----------|------|
| `FiveEImport.vue` (保留缩减) | 5E 特有模板 + 胶水 |
| `WanmeiImport.vue` (保留缩减) | 完美特有模板 + 胶水 |
| `useFiveEImport.ts` 🆕 | 5E 特有逻辑（ListFiveERecentMatches/ImportFiveEMatch） |
| `useWanmeiImport.ts` 🆕 | 完美特有逻辑（ListWanmeiRecentMatches/ImportWanmeiMatch） |
| `ImportMatchList.vue` 🆕 | 共用战绩表格子组件（分页 + 列表） |

#### 3️⃣ `EditPage.vue` (827 行 → ~450 行)

| 新/改文件 | 职责 |
|-----------|------|
| `EditPage.vue` (保留缩减) | 模板 + 胶水 |
| `useEditPage.ts` 🆕 | 页面级状态 + 合成流程逻辑 |
| `EditConcatPanel.vue` 🆕 | 拼接参数设置面板 |

#### 4️⃣ `TopBar.vue` (686 行 → ~350 行)

| 新/改文件 | 职责 |
|-----------|------|
| `TopBar.vue` (保留缩减) | 导航 + 路由 + 窗口控制 |
| `ProduceHistoryDropdown.vue` 🆕 | 导出历史下拉面板 |
| `topbar-nav.ts` 🆕 | 导航菜单配置 + 路由映射常量 |

### 组 2 — Composable / TS 拆分

#### 5️⃣ `useImportDemos.ts` (389 行)

| 新/改文件 | 职责 |
|-----------|------|
| `useImportDemos.ts` (保留缩减) | 核心 Demo 列表管理（选择/导航/增删） |
| `useDemoData.ts` 🆕 | 后端调用包装 + 数据格式化 |
| `demo-helpers.ts` 🆕 | 纯工具函数（basename, normalizeSpecMode, normalizeClipOverrides） |

#### 6️⃣ `useStartupWizard.ts` (341 行)

| 新/改文件 | 职责 |
|-----------|------|
| `useStartupWizard.ts` (保留缩减) | 启动流程编排 + 后端调用 |
| `startup-display.ts` 🆕 | 纯展示函数（statusText, statusTagType, componentName 等） |

#### 7️⃣ `shared/types.ts` (356 行 → 拆分为 6 个文件)

| 新/改文件 | 包含类型 |
|-----------|---------|
| `types/index.ts` 🆕 | barrel re-export（保持 `@/shared/types` 路径有效） |
| `types/startup.ts` 🆕 | StartupState, ComponentStatus, ProgressMessage 等 |
| `types/demo.ts` 🆕 | DemoMetadata, DemoClipKill, DemoListEntry 等 |
| `types/import.ts` 🆕 | WanmeiMatchItem, FiveEMatchItem 等 |
| `types/clips.ts` 🆕 | ClipSettings, GeneratePluginJSONRequest 等 |
| `types/edit.ts` 🆕 | EditConcatRequest, ComposeProgressMessage 等 |
| `types.ts` (删除) | — |

## 约束（不改变的事项）

- **不修改** `frontend/wailsjs/**`（自动生成绑定）
- **不修改** `frontend/src/auto-imports.d.ts`、`components.d.ts`
- **不修改** `i18n/` 翻译文件
- **不修改**后端代码（`internal/`）
- **不修改**功能行为（纯拆分，无重构）
- **不修改** `shared/events.ts`（事件名常量）

## Acceptance Criteria

- [ ] `ProducePage.vue` 从 1,042 行缩减到 ~400 行
- [ ] `FiveEImport.vue` 从 740 行缩减到 ~400 行
- [ ] `WanmeiImport.vue` 从 718 行缩减到 ~400 行
- [ ] `EditPage.vue` 从 827 行缩减到 ~450 行
- [ ] `TopBar.vue` 从 686 行缩减到 ~350 行
- [ ] `useImportDemos.ts` 从 389 行缩减，工具函数已提取
- [ ] `useStartupWizard.ts` 从 341 行缩减，展示函数已提取
- [ ] `shared/types.ts` 拆分为 `types/*.ts` + barrel export
- [ ] `npm run build` 通过
- [ ] 所有 `@/shared/types` 导入路径保持有效
- [ ] 无功能行为变化

## Definition of Done

- [ ] 前端编译校验通过：`cd frontend && npm run build`
- [ ] 后端编译通过：`go build ./...`
- [ ] 后端测试通过：`go test ./...`
- [ ] WailsJS 绑定无变化
- [ ] AGENTS.md 同步更新（如文件路径映射有变化）

## Out of Scope

- 不修改 `features/startup/` 的组件结构（StartupWizard 行数少，脚本也少）
- 不修改 `features/ads/`、`features/settings/`（文件小，职责明确）
- 不修改 `app/AppShell.vue`、`app/router.ts`
- 不修改 `main.ts`、`App.vue`
- 不做功能重构或行为变化
- 不改动 `i18n/` 翻译内容

## Technical Notes

- `.vue` 文件拆出 composable 时，确保不会引入循环依赖
- `shared/types/index.ts` barrel export 使用 `export type { ... } from './xxx'` 避免运行时膨胀
- `@/shared/types` 是一个 Vite path alias，对应 `src/shared/types`
  - 拆分后 `src/shared/types/index.ts` 重新导出各子文件类型，`@/shared/types` 仍有效
- 测试文件如引用原路径无需修改（因为 barrel export 保持接口一致）
