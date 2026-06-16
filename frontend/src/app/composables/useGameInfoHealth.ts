import { computed, ref } from "vue";
import type { GameInfoHealth } from "@/shared/types";

const defaultHealth: GameInfoHealth = {
  status: "unknown",
  needs_repair: false,
  gameinfo_path: "",
  message: "",
  error: "",
};

const health = ref<GameInfoHealth>({ ...defaultHealth });
const loading = ref(false);
const repairing = ref(false);
const lastError = ref("");

async function callBackend<T>(method: string, ...args: unknown[]): Promise<T> {
  const api = window.go?.app?.App as Record<string, (...a: unknown[]) => Promise<unknown>> | undefined;
  const fn = api?.[method];
  if (!fn) throw new Error(`Wails API not loaded: ${method}`);
  return (await fn(...args)) as T;
}

export function useGameInfoHealth() {
  const needsRepair = computed(() => health.value.status === "needs_repair" || health.value.needs_repair);
  const isHealthy = computed(() => health.value.status === "ok" && !health.value.needs_repair);
  const isUnknown = computed(() => health.value.status === "unknown");

  async function refresh(): Promise<GameInfoHealth> {
    loading.value = true;
    lastError.value = "";
    try {
      const next = await callBackend<GameInfoHealth>("GetGameInfoHealth");
      health.value = next || { ...defaultHealth };
      return health.value;
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      lastError.value = message;
      health.value = {
        ...defaultHealth,
        message,
        error: message,
      };
      return health.value;
    } finally {
      loading.value = false;
    }
  }

  async function repair(): Promise<GameInfoHealth> {
    repairing.value = true;
    lastError.value = "";
    try {
      const next = await callBackend<GameInfoHealth>("RepairGameInfo");
      health.value = next || { ...defaultHealth };
      return health.value;
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      lastError.value = message;
      throw error;
    } finally {
      repairing.value = false;
    }
  }

  return {
    health,
    loading,
    repairing,
    lastError,
    needsRepair,
    isHealthy,
    isUnknown,
    refresh,
    repair,
  };
}
