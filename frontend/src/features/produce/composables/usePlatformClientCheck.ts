import { computed, ref } from "vue";
import type { PlatformClientStatus } from "@/shared/types";

const statuses = ref<PlatformClientStatus[]>([]);
const refreshing = ref(false);

export function usePlatformClientCheck() {
  const allClosed = computed(() => statuses.value.every((s) => !s.running));
  const anyRunning = computed(() => statuses.value.some((s) => s.running));

  async function checkAll(): Promise<boolean> {
    try {
      const result = await callBackend<PlatformClientStatus[]>("CheckPlatformClients");
      statuses.value = result;
      return result.every((s) => !s.running);
    } catch {
      return false;
    }
  }

  async function refresh(): Promise<void> {
    refreshing.value = true;
    try {
      await checkAll();
    } finally {
      refreshing.value = false;
    }
  }

  function reset(): void {
    statuses.value = [];
    refreshing.value = false;
  }

  return {
    statuses,
    allClosed,
    anyRunning,
    refreshing,
    checkAll,
    refresh,
    reset,
  };
}

async function callBackend<T>(method: string, ...args: unknown[]): Promise<T> {
  const api = (window as any).go?.app?.App as Record<string, (...a: unknown[]) => Promise<unknown>> | undefined;
  const fn = api?.[method];
  if (!fn) throw new Error(`Wails API not loaded: ${method}`);
  return fn(...args) as Promise<T>;
}
