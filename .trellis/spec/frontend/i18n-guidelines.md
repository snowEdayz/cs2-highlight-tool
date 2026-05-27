# i18n Guidelines

> Internationalization conventions in the frontend.

---

## Overview

The project uses a custom lightweight i18n system (no vue-i18n). All translations live in `shared/i18n/` as JSON files and are accessed via the `t()` helper.

---

## Architecture

```
shared/i18n/
├── index.ts           # t() helper, useI18n(), locale management
├── zh-CN.json         # Chinese translations (the one to edit)
└── en-US.json         # English translations (user-maintained, do not edit)
```

The system:
1. Detects the locale from `localStorage` or browser `navigator.language`
2. Falls back to `zh-CN` if no preference is stored
3. Falls back to `zh-CN` keys when a `en-US` key is missing
4. Falls back to the raw key string if neither locale has it

---

## Locale Management

```ts
type Locale = "zh-CN" | "en-US";
```

Locale is persisted to `localStorage` under key `cs2-highlight-tool.locale`:

```ts
export function setLocale(next: Locale): void {
  localeRef.value = next;
  window.localStorage.setItem(storageKey, next);
}
```

---

## The `t()` Helper

```ts
export function t(key: string, params?: Record<string, string | number>): string;
```

The `t()` function is available in all composables and components. Import it directly:

```ts
import { t } from "@/shared/i18n";
```

### Usage in Templates

```vue
<n-empty :description="t('main.produce.no_selection_or_done')" />
<n-button>{{ t("main.produce.start_produce") }}</n-button>
```

### Usage with Parameters

```ts
t("startup.version.component_meta", {
  local: versionWithPrefix(task.component.local_version),
  remote: versionWithPrefix(task.component.remote_version),
});

t("main.import.duration_fmt", { minutes: m, seconds: s });
```

Template parameters use `{paramName}` syntax in the JSON:

```json
// zh-CN.json
{
  "main": {
    "import": {
      "duration_fmt": "{minutes}分{seconds}秒"
    },
    "produce": {
      "batch_summary": "成功: {success}, 失败: {failed}"
    }
  }
}
```

---

## Key Naming Convention

Translation keys follow a **dot-separated path** hierarchical structure:

| Prefix | Domain | Example |
|--------|--------|---------|
| `startup.` | Startup wizard | `startup.status.ready`, `startup.components.hlae`, `startup.actions.pick_cs2` |
| `main.` | Main app area | `main.import.duration_fmt`, `main.produce.title`, `main.clips.round_title` |

Deep nesting mirrors the JSON structure:

```json
{
  "startup": {
    "status": {
      "pending": "待处理",
      "checking": "检查中",
      "downloading": "下载中",
      "ready": "已完成",
      "failed": "失败"
    },
    "components": {
      "hlae": "HLAE",
      "plugin": "插件",
      "ffmpeg": "FFmpeg",
      "cs2": "CS2"
    }
  },
  "main": {
    "produce": {
      "title": "录制与生产",
      "start_produce": "开始生产",
      "no_selection_or_done": "暂未选择素材或已完成所有录制"
    }
  }
}
```

---

## The `useI18n()` Composable

For components that need locale switching, the `useI18n()` composable is available:

```ts
export function useI18n() {
  return {
    locale: computed(() => localeRef.value),   // current locale
    locales: supportedLocales,                   // ["zh-CN", "en-US"]
    setLocale,                                   // setter
    t,                                           // the translation function
  };
}
```

---

## Editing Rules

- ✅ **Only modify `zh-CN.json`** — the source of truth
- ❌ **Do NOT edit `en-US.json`** — the user maintains this separately (per AGENTS.md)
- When adding a new feature, add all keys to `zh-CN.json` first
- Use `t()` for all user-facing strings — no hardcoded text in templates or composables
- Write keys in `snake_case` or `camelCase` consistently within each domain
- Parametrize variable content (numbers, names, paths) via `{param}` — do not concatenate strings manually

---

## What Not to Do

- ❌ **Do not install `vue-i18n`** — the custom system is sufficient
- ❌ **Do not hardcode user-facing strings** — always use `t()`
- ❌ **Do not edit `en-US.json`** — user handles that
- ❌ **Do not use `$t()` or `this.$t()`** — Vue 3 Composition API uses the imported `t()` function
