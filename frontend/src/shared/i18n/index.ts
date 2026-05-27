import { computed, ref } from "vue";
import enUS from "./en-US.json";
import zhCN from "./zh-CN.json";

export type Locale = "zh-CN" | "en-US";

const storageKey = "cs2-highlight-tool.locale";
const supportedLocales: Locale[] = ["zh-CN", "en-US"];

type MessageValue = string | Record<string, unknown>;
type MessageTree = Record<string, MessageValue>;

const messages: Record<Locale, MessageTree | undefined> = {
  "zh-CN": zhCN as MessageTree,
  "en-US": enUS as MessageTree,
};

const localeRef = ref<Locale>(resolveInitialLocale());

function normalizeLocale(value: string | null | undefined): Locale | undefined {
  if (!value) return undefined;
  if (value === "zh-CN" || value === "en-US") return value;
  return undefined;
}

function detectLocaleByNavigator(): Locale {
  if (typeof navigator === "undefined") return "zh-CN";
  const browserLocale = (navigator.language || "").toLowerCase();
  if (browserLocale.startsWith("zh")) return "zh-CN";
  return "en-US";
}

function resolveInitialLocale(): Locale {
  if (typeof window === "undefined") return "zh-CN";
  const stored = normalizeLocale(window.localStorage.getItem(storageKey));
  if (stored) return stored;
  return detectLocaleByNavigator();
}

function isMessageTree(value: unknown): value is MessageTree {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function resolveMessage(tree: MessageTree | undefined, key: string): string | undefined {
  if (!tree) return undefined;
  const segments = key.split(".");
  let current: unknown = tree;
  for (const segment of segments) {
    if (!isMessageTree(current)) return undefined;
    current = current[segment];
  }
  return typeof current === "string" ? current : undefined;
}

function formatMessage(template: string, params?: Record<string, string | number>): string {
  if (!params) return template;
  return template.replace(/\{(\w+)\}/g, (_, token: string) => {
    const value = params[token];
    return value === undefined ? `{${token}}` : String(value);
  });
}

export function t(key: string, params?: Record<string, string | number>): string {
  const activeLocale = localeRef.value;
  const template =
    resolveMessage(messages[activeLocale], key) ??
    resolveMessage(messages["zh-CN"], key) ??
    key;
  return formatMessage(template, params);
}

export function setLocale(next: Locale): void {
  if (!supportedLocales.includes(next)) return;
  localeRef.value = next;
  if (typeof window !== "undefined") {
    window.localStorage.setItem(storageKey, next);
  }
}

export function useI18n() {
  return {
    locale: computed(() => localeRef.value),
    locales: supportedLocales,
    setLocale,
    t,
  };
}
