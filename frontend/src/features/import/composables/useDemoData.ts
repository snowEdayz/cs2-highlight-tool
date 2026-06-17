import { ref } from "vue";
import { t } from "@/shared/i18n";
import type {
  DemoClipKill,
  DemoClipPlayer,
  DemoClipRound,
  DemoListEntry,
  DemoMaterialSelection,
  DemoMetadata,
  DemoPlayerInfo,
} from "@/shared/types";

export const selectedPlayerByDemo = ref<Record<string, string>>({});
export const materialByDemo = ref<Record<string, DemoMaterialSelection[]>>({});
export const fullRoundPovByDemo = ref<Record<string, DemoFullRoundPOVSelection>>({});
export const fullRoundPlanByDemo = ref<Record<string, import("@/shared/types").FullRoundPOVPlan>>({});

export interface DemoFullRoundPOVSelection {
  enabled: boolean;
  player_steam_id: string;
}

export function useDemoData() {
  return {
    selectedPlayerByDemo,
    materialByDemo,
    fullRoundPovByDemo,
  };
}

export function getClipPlayers(entry: DemoListEntry | null): DemoClipPlayer[] {
  if (!entry?.meta?.clip_players) return [];
  return entry.meta.clip_players;
}

export function getFullRoundPlayers(entry: DemoListEntry | null): DemoPlayerInfo[] {
  if (!entry?.meta?.players) return [];
  return entry.meta.players;
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

export function getFullRoundPlayerSteamID(player: DemoPlayerInfo): string {
  const explicit = String(player.steam_id_text || "").trim();
  if (explicit) return explicit;
  return String(player.steam_id || "").trim();
}

export function getClipRounds(entry: DemoListEntry | null, playerSteamID: string): DemoClipRound[] {
  if (!entry || !playerSteamID) return [];
  const players = getClipPlayers(entry);
  const player = players.find((item) => item.steam_id === playerSteamID);
  return player?.rounds ?? [];
}

export function getFullRoundPOVSelection(entry: DemoListEntry | null): DemoFullRoundPOVSelection {
  if (!entry) return { enabled: false, player_steam_id: "" };
  return fullRoundPovByDemo.value[entry.key] ?? { enabled: false, player_steam_id: "" };
}

export function setFullRoundPOVEnabled(entry: DemoListEntry | null, enabled: boolean) {
  if (!entry) return;
  if (enabled) {
    syncDefaultFullRoundPlayer(entry);
    const playerSteamID = selectedPlayerByDemo.value[entry.key] || "";
    setDemoMaterials(entry, []);
    fullRoundPovByDemo.value = {
      ...fullRoundPovByDemo.value,
      [entry.key]: { enabled: true, player_steam_id: playerSteamID },
    };
    return;
  }
  fullRoundPovByDemo.value = {
    ...fullRoundPovByDemo.value,
    [entry.key]: { enabled: false, player_steam_id: "" },
  };
}

export function syncFullRoundPOVPlayer(entry: DemoListEntry | null, playerSteamID: string) {
  if (!entry) return;
  const current = getFullRoundPOVSelection(entry);
  if (!current.enabled) return;
  fullRoundPovByDemo.value = {
    ...fullRoundPovByDemo.value,
    [entry.key]: { enabled: true, player_steam_id: String(playerSteamID || "").trim() },
  };
  const nextPlanCache = { ...fullRoundPlanByDemo.value };
  delete nextPlanCache[entry.key];
  fullRoundPlanByDemo.value = nextPlanCache;
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

export async function fetchFullRoundPOVPlan(entry: DemoListEntry | null, playerSteamID: string): Promise<void> {
  if (!entry || !playerSteamID) return;
  try {
    const api = (window as any).go?.app?.App as
      | Record<string, (...a: unknown[]) => Promise<unknown>>
      | undefined;
    const fn = api?.PreviewFullRoundPOV;
    if (!fn) return;
    const plan = await fn(entry.file_path, playerSteamID);
    fullRoundPlanByDemo.value = {
      ...fullRoundPlanByDemo.value,
      [entry.key]: plan as import("@/shared/types").FullRoundPOVPlan,
    };
  } catch {
    fullRoundPlanByDemo.value = {
      ...fullRoundPlanByDemo.value,
      [entry.key]: { player_name: "", player_steam_id: "", segments: [] },
    };
  }
}

export function getFullRoundPOVTrackingLabel(entry: DemoListEntry | null): string {
  if (!entry) return "";
  const sel = getFullRoundPOVSelection(entry);
  if (!sel.enabled || !sel.player_steam_id) return "";
  const players = getFullRoundPlayers(entry);
  const player = players.find((p) => getFullRoundPlayerSteamID(p) === sel.player_steam_id);
  return player?.name || sel.player_steam_id;
}

export function syncDefaultFullRoundPlayer(entry: DemoListEntry | null): void {
  if (!entry?.meta?.players?.length) return;
  const players = entry.meta.players;
  const current = selectedPlayerByDemo.value[entry.key];
  const exists = players.some((player) => getFullRoundPlayerSteamID(player) === current);
  if (exists) return;
  selectedPlayerByDemo.value = {
    ...selectedPlayerByDemo.value,
    [entry.key]: getFullRoundPlayerSteamID(players[0]),
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
