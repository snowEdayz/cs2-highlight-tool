import { ref } from "vue";
import { EventsOn } from "../../../../wailsjs/runtime/runtime";
import type { ProduceHistorySnapshot } from "@/shared/types";

const historySnapshot = ref<ProduceHistorySnapshot>({
  items: [],
  updated_at_ms: 0,
});

let initialized = false;

async function callBackend<T>(method: string, ...args: unknown[]): Promise<T> {
  const api = (window as any).go?.app?.App as Record<string, (...a: unknown[]) => Promise<unknown>> | undefined;
  const fn = api?.[method];
  if (!fn) {
    throw new Error(`Wails API not loaded: ${method}`);
  }
  return fn(...args) as Promise<T>;
}

export async function ensureProduceHistoryInitialized(): Promise<void> {
  if (initialized) {
    return;
  }
  initialized = true;

  try {
    historySnapshot.value = await callBackend<ProduceHistorySnapshot>("GetProduceHistorySnapshot");
  } catch {
    // ignore and wait for events
  }

  EventsOn("produce_history_changed", (next: ProduceHistorySnapshot) => {
    historySnapshot.value = next || { items: [], updated_at_ms: 0 };
  });
}

export function useProduceHistory() {
  return {
    historySnapshot,
  };
}
