import { ref } from "vue";
import type { PendingChangelog } from "@/shared/types";

const pending = ref<PendingChangelog | null>(null);
const checked = ref(false);

async function callBackend<T = unknown>(method: string, ...args: unknown[]): Promise<T> {
  const api = window.go?.app?.App as Record<string, (...a: unknown[]) => Promise<unknown>> | undefined;
  const fn = api?.[method];
  if (!fn) throw new Error(`Wails API not loaded: ${method}`);
  return (await fn(...args)) as T;
}

export function useChangelog() {
  async function checkPending(): Promise<void> {
    if (checked.value) return;
    checked.value = true;
    try {
      const result = await callBackend<PendingChangelog>("GetPendingChangelog");
      if (result?.should_show) {
        pending.value = result;
      }
    } catch {
      // 失败时静默：弹窗丢失不影响主流程
    }
  }

  async function ack(): Promise<void> {
    const current = pending.value;
    pending.value = null;
    if (!current?.version) return;
    try {
      await callBackend("AckChangelog", current.version);
    } catch {
      // ack 写盘失败不阻塞用户，下次启动还会再弹一次（属于可接受的优雅退化）
    }
  }

  return {
    pending,
    checkPending,
    ack,
  };
}
