# State Management

> Reactive state patterns in the frontend.

---

## Overview

This project does **not use Pinia** or any external state management library. All state is managed via Vue 3's built-in `ref()` / `reactive()` / `computed()` within composables. The pattern is deliberately simple — module-level singleton refs shared across a composable's consumers.

---

## The Module-Level Singleton Pattern

State is declared at the **module level** of a composable file, outside the `export function`:

```ts
// ⚠️ BAD — state inside the function, recreated every call
export function useCounter() {
  const count = ref(0);
  return { count, increment: () => count.value++ };
}
```

```ts
// ✅ GOOD — module-level singleton, shared across all consumers
const count = ref(0);

export function useCounter() {
  const increment = () => count.value++;
  return { count, increment };
}
```

The project uses this pattern consistently:

```ts
// features/import/composables/useImportDemos.ts
const demoList = ref<DemoListEntry[]>([]);
const selectedIndex = ref<number | null>(null);
const detailCollapsed = ref(true);
const selectedPlayerByDemo = ref<Record<string, string>>({});
const materialByDemo = ref<Record<string, DemoMaterialSelection[]>>({});
const autoAddVictimView = ref(true);
let keyCounter = 0;
```

```ts
// features/edit/composables/useEditState.ts
const sequenceItems = ref<EditSequenceItem[]>([]);
const exporting = ref(false);
const exportError = ref("");
const exportPath = ref("");
const transitionMode = ref<EditTransitionMode>("none");
const transitionDuration = ref(0.3);
```

And in `shared/state/`:

```ts
// shared/state/useDebugSettings.ts
const brandClickCount = ref(0);
const debugEnabled = ref(false);
const keepProduceIntermediates = ref(false);
```

---

## State Categories

### 1. Feature-local state (default)

Declared in the composable file at module level — consumed by all components within the feature.

```ts
// These are shared across all components that import useImportDemos()
const demoList = ref<DemoListEntry[]>([]);
const selectedIndex = ref<number | null>(null);
```

### 2. Shared cross-feature state

Placed in `shared/state/` for consumption across features:

```ts
// shared/state/useDebugSettings.ts — consumed by startup, produce, and settings
export function useDebugSettings() {
  return { debugEnabled, keepProduceIntermediates, activateDebugByBrandClick };
}
```

### 3. Route guard / transient state

When state is truly page- or component-local and should not be shared, declare it **inside the component** using `<script setup>`:

```ts
// Inside a page component — only used here
const generatingAndLaunching = ref(false);
const generatingConfigOnlyLoading = ref(false);
const errorMessage = ref("");
```

But in many cases the project still uses module-level state even for what appears to be page-local state — this is an accepted pattern as long as there's no cross-feature name collision.

---

## Computed Properties

`computed` is used for derived state — never store derived data in a `ref`:

```ts
const selectedEntry = computed<DemoListEntry | null>(() => {
  const idx = selectedIndex.value;
  if (idx == null || idx < 0 || idx >= demoList.value.length) return null;
  return demoList.value[idx];
});

const selectedDemo = computed<DemoMetadata | null>(() => selectedEntry.value?.meta ?? null);

const canSelectPrev = computed(() => selectedIndex.value != null && selectedIndex.value > 0);
const canSelectNext = computed(() =>
  selectedIndex.value != null && selectedIndex.value < demoList.value.length - 1,
);
```

---

## TypeScript Integration

All shared types are defined in `shared/types.ts`, closely mirroring the Go backend structs:

```ts
export interface StartupState {
  mode: string;
  phase: string;
  running: boolean;
  source_step: SourceStepState;
  // ...
}
```

**Key conventions**:
- `number` for numeric values (even tick counts that are conceptually ints)
- `string` for enum-like values (no TypeScript enums — use union types like `"standard" | "high" | "ultra"`)
- `boolean` for flags
- `Record<string, T>` for string-keyed maps
- `undefined` for optional fields (via `?` — `field?: string`)

---

## watchers

`watch` is used for reactive side effects:

```ts
watch(
  () => displayDemos.value.map((entry) => entry.key),
  (keys) => {
    // Auto-expand collapsible sections when data changes
    expandedNames.value = [...keys];
  },
  { immediate: true },
);
```

---

## Mutations

State is mutated by **reassigning `.value`** on refs:

```ts
demoList.value.push(...newEntries);
demoList.value[idx] = nextEntry;

// Or for object-refs:
selectedPlayerByDemo.value = {
  ...selectedPlayerByDemo.value,
  [entry.key]: players[0].steam_id,
};
```

No action/dispatch pattern — direct mutation is the convention.

---

## What Not to Do

- ❌ **Do not install Pinia or Vuex** — the module-level composable pattern is sufficient
- ❌ **Do not declare state inside `setup()` or `export function`** — use module-level refs for shared state
- ❌ **Do not store derived data in a `ref`** — use `computed`
- ❌ **Do not create TypeScript enums for UI state** — use string union types
- ❌ **Do not use `reactive()` for primitive values** — always use `ref()` for primitives
