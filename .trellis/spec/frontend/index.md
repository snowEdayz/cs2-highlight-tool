# Frontend Development Guidelines

> Best practices for Vue 3 + TypeScript frontend development in this project.

---

## Overview

This directory contains frontend coding conventions extracted from the real `frontend/src/` codebase. These guidelines help AI agents write Vue/TS code that matches the team's established patterns.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Feature-first layout, pages/components/composables convention | ✅ Populated |
| [Component Guidelines](./component-guidelines.md) | Naive UI usage, `<script setup>`, template/scoped styles patterns | ✅ Populated |
| [State Management](./state-management.md) | Module-level composable singletons, no Pinia, ref/computed patterns | ✅ Populated |
| [i18n Guidelines](./i18n-guidelines.md) | `t()` helper, dot-path keys, zh-CN.json editing rules | ✅ Populated |
| [Quality Guidelines](./quality-guidelines.md) | Build/type-check, auto-imports, forbidden patterns, code review checklist | ✅ Populated |

---

## How to Use These Guidelines

1. **Before writing frontend code**: Read the relevant guideline for your component type
2. **When adding a new feature**: Follow the `features/<name>/pages/ + components/ + composables/` layout
3. **When calling backend methods**: Use the `callBackend()` helper pattern
4. **When adding UI strings**: Add keys to `zh-CN.json` and use `t()` in templates

## For AI Agents

When dispatched as `trellis-implement` or `trellis-check`, these spec files are auto-injected into your prompt via `implement.jsonl` / `check.jsonl`. Read them before writing code.
