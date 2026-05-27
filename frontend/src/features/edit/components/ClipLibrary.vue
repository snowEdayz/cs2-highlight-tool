<template>
  <div class="clip-library">
    <div class="library-header">
      <span class="library-title">{{ t("main.edit.clip_library") }}</span>
      <n-tag v-if="historyGroups.length" size="small" :bordered="false">
        {{ totalProduceClipCount }}
      </n-tag>
    </div>

    <n-empty
      v-if="!historyGroups.length"
      :description="t('main.edit.clip_library_empty')"
      size="small"
    />

    <div v-else class="library-scroll">
      <n-collapse
        :expanded-names="getDemoExpandedNames()"
        @update:expanded-names="handleDemoExpanded"
      >
        <n-collapse-item
          v-for="group in historyGroups"
          :key="group.demo_path"
          :name="group.demo_path"
          :title="basename(group.demo_path)"
        >
          <template #header-extra>
            <n-tag size="tiny" :bordered="false">{{ group.items.length }}</n-tag>
          </template>

          <n-collapse
            :expanded-names="getRoundExpanded(group.demo_path)"
            @update:expanded-names="handleRoundExpanded(group.demo_path, $event)"
          >
            <n-collapse-item
              v-for="roundGroup in roundGroupsForDemo(group)"
              :key="`${group.demo_path}-${roundGroup.name}`"
              :name="roundGroup.name"
              :title="roundTitle(roundGroup)"
            >
              <div
                v-for="item in roundGroup.items"
                :key="historyRowKey(item)"
                class="library-item"
                draggable="true"
                @dragstart="onDragStart($event, item)"
              >
                <div class="item-head">
                  <n-tag
                    size="tiny"
                    :bordered="false"
                    :type="viewTagType(item.view)"
                  >
                    {{ viewLabel(item.view) }}
                  </n-tag>
                  <span class="item-time">{{ formatTime(item.completed_at_ms) }}</span>
                </div>
                <div v-if="item.kills?.length" class="item-kills">
                  <div v-for="kill in item.kills" :key="kill.id" class="item-kill-row">
                    <DeathNoticeLine :kill="kill" compact />
                  </div>
                </div>
                <div v-else class="item-meta">
                  {{ t("topbar.history_kill_count", { count: item.kill_ids?.length || 0 }) }}
                </div>
              </div>
            </n-collapse-item>
          </n-collapse>
        </n-collapse-item>
      </n-collapse>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useProduceHistory } from "@/features/produce/composables/useProduceHistory";
import DeathNoticeLine from "@/features/clips/components/DeathNoticeLine.vue";
import { t } from "@/shared/i18n";
import type { DemoClipKill, ProduceHistoryItem } from "@/shared/types";

const { historySnapshot } = useProduceHistory();

const demoExpanded = ref<string[]>([]);
const demoExpandedInitialized = ref(false);
const roundExpandedByDemo = ref<Record<string, string[]>>({});

interface HistoryDemoGroup {
  demo_path: string;
  items: ProduceHistoryItem[];
}

interface HistoryRoundGroup {
  name: string;
  round: number;
  kill_count: number;
  items: ProduceHistoryItem[];
}

const produceHistoryItems = computed(() =>
  [...(historySnapshot.value.items || [])]
    .filter((item) => (item.history_type || "produce_clip") !== "edited_video")
    .sort((a, b) => (b.completed_at_ms || 0) - (a.completed_at_ms || 0)),
);

const totalProduceClipCount = computed(() => produceHistoryItems.value.length);

const historyGroups = computed<HistoryDemoGroup[]>(() => {
  const order: string[] = [];
  const byDemo = new Map<string, ProduceHistoryItem[]>();
  for (const item of produceHistoryItems.value) {
    const demoPath = item.demo_path || "";
    if (!demoPath) continue;
    if (!byDemo.has(demoPath)) {
      byDemo.set(demoPath, []);
      order.push(demoPath);
    }
    byDemo.get(demoPath)!.push(item);
  }
  return order.map((demoPath) => ({
    demo_path: demoPath,
    items: byDemo.get(demoPath) || [],
  }));
});

watch(
  () => historyGroups.value.map((group) => group.demo_path),
  (demoKeys) => {
    if (!demoExpandedInitialized.value) {
      demoExpanded.value = [...demoKeys];
      demoExpandedInitialized.value = true;
      return;
    }
    const allowed = new Set(demoKeys);
    demoExpanded.value = demoExpanded.value.filter((name) => allowed.has(name));
  },
  { immediate: true },
);

const roundGroupsByDemo = computed(() => {
  const next = new Map<string, HistoryRoundGroup[]>();
  for (const group of historyGroups.value) {
    const groupedByRound = new Map<string, HistoryRoundGroup>();
    for (const item of group.items) {
      const split = splitKillsByRound(
        (item.kills || []).filter((kill): kill is DemoClipKill => !!kill?.id),
      );
      if (!split.length) {
        split.push({ round: 0, kills: [] });
      }
      for (const part of split) {
        const name = part.round > 0 ? `round-${part.round}` : "round-unknown";
        if (!groupedByRound.has(name)) {
          groupedByRound.set(name, {
            name,
            round: part.round,
            kill_count: 0,
            items: [],
          });
        }
        groupedByRound.get(name)!.items.push(item);
        groupedByRound.get(name)!.kill_count += part.kills.length;
      }
    }
    const sortedGroups = Array.from(groupedByRound.values()).sort((a, b) => {
      if (a.round <= 0 && b.round <= 0) return 0;
      if (a.round <= 0) return 1;
      if (b.round <= 0) return -1;
      return a.round - b.round;
    });
    next.set(group.demo_path, sortedGroups);
  }
  return next;
});

function roundGroupsForDemo(group: HistoryDemoGroup): HistoryRoundGroup[] {
  return roundGroupsByDemo.value.get(group.demo_path) || [];
}

function getDemoExpandedNames(): string[] {
  const allowed = new Set(historyGroups.value.map((group) => group.demo_path));
  return demoExpanded.value.filter((name) => allowed.has(name));
}

function getRoundExpanded(demoPath: string): string[] {
  const groups = roundGroupsByDemo.value.get(demoPath) || [];
  const defaults = groups.map((g) => g.name);
  if (!Object.prototype.hasOwnProperty.call(roundExpandedByDemo.value, demoPath)) {
    return defaults;
  }
  const current = roundExpandedByDemo.value[demoPath] || [];
  const allowed = new Set(defaults);
  return current.filter((name) => allowed.has(name));
}

function handleDemoExpanded(names: Array<string | number> | string | number | null) {
  demoExpanded.value = normalizeNames(names);
}

function handleRoundExpanded(
  demoPath: string,
  names: Array<string | number> | string | number | null,
) {
  roundExpandedByDemo.value = {
    ...roundExpandedByDemo.value,
    [demoPath]: normalizeNames(names),
  };
}

function normalizeNames(names: Array<string | number> | string | number | null): string[] {
  if (Array.isArray(names)) return names.map((n) => String(n));
  if (names == null) return [];
  return [String(names)];
}

function onDragStart(event: DragEvent, item: ProduceHistoryItem) {
  event.dataTransfer!.effectAllowed = "copy";
  event.dataTransfer!.setData(
    "application/json",
    JSON.stringify({ type: "clip", item }),
  );
}

function splitKillsByRound(
  kills: DemoClipKill[],
): Array<{ round: number; kills: DemoClipKill[] }> {
  const grouped = new Map<number, DemoClipKill[]>();
  for (const kill of kills) {
    const r = Number(kill.round || 0);
    const nr = r > 0 ? r : 0;
    if (!grouped.has(nr)) grouped.set(nr, []);
    grouped.get(nr)!.push(kill);
  }
  return Array.from(grouped.entries())
    .sort((a, b) => {
      if (a[0] <= 0 && b[0] <= 0) return 0;
      if (a[0] <= 0) return 1;
      if (b[0] <= 0) return -1;
      return a[0] - b[0];
    })
    .map(([round, roundKills]) => ({ round, kills: roundKills }));
}

function historyRowKey(item: ProduceHistoryItem): string {
  return `${item.history_type || "produce_clip"}#${item.demo_path}#${item.view}#${item.spec_mode}#${item.completed_at_ms}#${item.video_path}#${(item.kill_ids || []).join("|")}`;
}

function roundTitle(group: HistoryRoundGroup): string {
  if (group.round > 0) {
    return t("main.clips.round_title", {
      round: group.round,
      kills: group.kill_count,
    });
  }
  return t("main.produce.round_unknown_title", { kills: group.kill_count });
}

function viewLabel(view: string): string {
  return view === "victim"
    ? t("main.clips.victim_view")
    : t("main.clips.killer_view");
}

function viewTagType(view: string): "success" | "warning" {
  return view === "victim" ? "warning" : "success";
}

function formatTime(tsMs: number): string {
  if (!tsMs) return "-";
  const d = new Date(tsMs);
  if (Number.isNaN(d.getTime())) return "-";
  return (
    String(d.getHours()).padStart(2, "0") +
    ":" +
    String(d.getMinutes()).padStart(2, "0") +
    ":" +
    String(d.getSeconds()).padStart(2, "0")
  );
}

function basename(path: string): string {
  if (!path) return "";
  return path.replaceAll("\\", "/").split("/").pop() || path;
}
</script>

<style scoped>
.clip-library {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.library-header {
  align-items: center;
  display: flex;
  gap: 8px;
  padding: 10px 12px 8px;
  border-bottom: 1px solid #303732;
  flex-shrink: 0;
}

.library-title {
  color: #edf1ee;
  font-size: 13px;
  font-weight: 600;
}

.library-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 6px 4px;
}

.library-item {
  background: rgba(26, 30, 27, 0.5);
  border: 1px solid #2f3631;
  border-radius: 6px;
  cursor: grab;
  padding: 6px 8px;
  margin-bottom: 4px;
  transition: background 0.15s, border-color 0.15s;
}

.library-item:hover {
  background: rgba(47, 54, 49, 0.6);
  border-color: #2f9462;
}

.library-item:active {
  cursor: grabbing;
}

.item-head {
  align-items: center;
  display: flex;
  gap: 6px;
  margin-bottom: 4px;
}

.item-time {
  color: #8d9890;
  font-size: 11px;
}

.item-kills {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
}

.item-kill-row {
  width: 100%;
}

.item-meta {
  color: #8d9890;
  font-size: 11px;
}
</style>
