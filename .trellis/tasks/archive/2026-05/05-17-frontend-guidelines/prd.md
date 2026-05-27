# Frontend Development Guidelines

**Objective**: Populate `.trellis/spec/frontend/` with real frontend coding conventions extracted from the `frontend/src/` codebase.

## Spec Files to Populate

| File | What to document | Status |
|------|------------------|--------|
| `directory-structure.md` | Vue/TS file organization: features/*, composables, pages vs components, shared/ | ✅ Populated |
| `component-guidelines.md` | Naive UI usage, Vue 3 Composition API (`<script setup>`), template conventions | ✅ Populated |
| `state-management.md` | Composables-based reactive state, ref/reactive patterns, module-level singletons | ✅ Populated |
| `i18n-guidelines.md` | t() helper, locale resolution, key naming conventions | ✅ Populated |
| `quality-guidelines.md` | Lint/type-check (`vue-tsc`), build requirements, forbidden patterns | ✅ Populated |

## Method

1. Import patterns from `AGENTS.md` frontend section and existing conventions
2. Read real `frontend/src/` code for patterns
3. Write concise, example-backed spec files matching actual codebase behavior
