import { ref } from "vue";
import { t } from "@/shared/i18n";
import type {
  DemoClipKill,
  DemoClipPlayer,
  DemoClipRound,
  DemoListEntry,
  DemoMaterialSelection,
  DemoMetadata,
} from "@/shared/types";

export const selectedPlayerByDemo = ref<Record<string, string>>({});
export const materialByDemo = ref<Record<string, DemoMaterialSelection[]>>({});

export function useDemoData() {
  return {
    selectedPlayerByDemo,
    materialByDemo,
  };
}

export function getClipPlayers(entry: DemoListEntry | null): DemoClipPlayer[] {
  if (!entry?.meta?.clip_players) return [];
  return entry.meta.clip_players;
}

export function getSelectedPlayerSteamID(entry: DemoListEntry | null): string {
  if (!entry) return "";
  syncDefaultPlayer(entry);
  return selectedPlayerByDemo.value[entry.key] ?? "";
}

export function setSelectedPlayerSteamID(entry: DemoListEntry | null, steamID: string) {
  if (!entry) return;
  selectedPlayerByDemo.value = {
    ...selectedPlayerByDemo.value,
    [entry.key]: steamID,
  };
}

export function getClipRounds(entry: DemoListEntry | null, playerSteamID: string): DemoClipRound[] {
  if (!entry || !playerSteamID) return [];
  const players = getClipPlayers(entry);
  const player = players.find((item) => item.steam_id === playerSteamID);
  return player?.rounds ?? [];
}

export function getDemoMaterials(entry: DemoListEntry | null): DemoMaterialSelection[] {
  if (!entry) return [];
  const current = materialByDemo.value[entry.key];
  if (current) return current;
  materialByDemo.value = {
    ...materialByDemo.value,
    [entry.key]: [],
  };
  return materialByDemo.value[entry.key] || [];
}

export function setDemoMaterials(entry: DemoListEntry | null, next: DemoMaterialSelection[]) {
  if (!entry) return;
  const sorted = next
    .slice()
    .sort((a, b) =>
      a.kill.tick === b.kill.tick ? a.kill.id.localeCompare(b.kill.id) : a.kill.tick - b.kill.tick,
    );
  materialByDemo.value = {
    ...materialByDemo.value,
    [entry.key]: sorted,
  };
}

export function syncDefaultPlayer(entry: DemoListEntry | null): void {
  if (!entry?.meta?.clip_players?.length) return;
  const players = entry.meta.clip_players;
  const current = selectedPlayerByDemo.value[entry.key];
  const exists = players.some((player) => player.steam_id === current);
  if (exists) return;
  selectedPlayerByDemo.value = {
    ...selectedPlayerByDemo.value,
    [entry.key]: players[0].steam_id,
  };
}

export function formatDuration(seconds: number): string {
  if (!seconds || seconds <= 0) return "-";
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return t("main.import.duration_fmt", { minutes: m, seconds: s });
}

export async function callBackend<T>(method: string, ...args: unknown[]): Promise<T> {
  const api = (window as any).go?.app?.App as
    | Record<string, (...a: unknown[]) => Promise<unknown>>
    | undefined;
  const fn = api?.[method];
  if (!fn) throw new Error(`Wails API not loaded: ${method}`);
  return fn(...args) as Promise<T>;
}
