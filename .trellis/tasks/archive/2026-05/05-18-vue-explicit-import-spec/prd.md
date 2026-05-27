# 记录 Vue API 显式导入原则到前端编码规范

## Goal

将本次发现的编码准则——Vue API 必须在 `<script setup>` 中显式导入，不得依赖 `unplugin-auto-import` 全局声明——持久化记录到 `.trellis/spec/frontend/` 规范文件中，使得未来 AI 会话能自动读取并遵守此原则。

## 背景

- **问题**: `TopBar.vue` 使用了 `nextTick()` 但没有通过 `import { nextTick } from "vue"` 显式导入，依赖 `unplugin-auto-import` 的全局声明。Windows 下 TypeScript strict 构建无法解析全局声明，导致 `TS2304: Cannot find name 'nextTick'`。
- **已处理**: 修复了代码 + 记录了 `frontend/AGENTS.md`
- **缺少**: `.trellis/spec/frontend/` 中尚无此准则

## 要求

在 `.trellis/spec/frontend/component-guidelines.md` 中添加一条编码准则，包含：
- 原则说明
- 为什么（Windows 构建失败）
- 正确 vs 错误的代码示例
- 相关的 TS 错误号（TS2304）

## Acceptance Criteria

- [ ] `.trellis/spec/frontend/component-guidelines.md` 新增一条关于 Vue API 显式导入的准则
- [ ] 包含正确示例（`import { ref, nextTick } from "vue"`）
- [ ] 包含错误示例（依赖全局声明）
- [ ] 解释 Windows 构建失败的根因

## Out of Scope

- 不修改任何代码文件
- 不修改 `frontend/AGENTS.md`（已记录）
