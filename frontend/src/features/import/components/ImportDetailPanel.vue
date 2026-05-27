<template>
  <section class="detail-panel">
    <header class="detail-header">
      <span class="panel-title">{{ t("main.import.detail_title") }}</span>
      <div class="detail-header-actions">
        <n-button
          quaternary
          size="small"
          circle
          class="icon-only-btn"
          :title="t('main.import.detail_prev')"
          :disabled="!canSelectPrev"
          @click="$emit('select-prev')"
        >
          <template #icon>
            <n-icon size="14">
              <ChevronLeftIcon />
            </n-icon>
          </template>
        </n-button>
        <n-button
          quaternary
          size="small"
          circle
          class="icon-only-btn"
          :title="t('main.import.detail_next')"
          :disabled="!canSelectNext"
          @click="$emit('select-next')"
        >
          <template #icon>
            <n-icon size="14">
              <ChevronRightIcon />
            </n-icon>
          </template>
        </n-button>
        <n-button
          quaternary
          size="small"
          circle
          class="icon-only-btn"
          :title="detailCollapsed ? t('main.import.detail_expand') : t('main.import.detail_collapse')"
          @click="$emit('toggle-collapse')"
        >
          <template #icon>
            <n-icon size="14">
              <ChevronDownIcon v-if="detailCollapsed" />
              <ChevronUpIcon v-else />
            </n-icon>
          </template>
        </n-button>
      </div>
    </header>

    <div v-if="!detailCollapsed" class="detail-scroll">
      <template v-if="selectedEntry">
        <div v-if="selectedEntry.loading" class="detail-loading">
          <n-spin size="small" />
          <n-text depth="3">{{ t('main.import.detail_loading') }}</n-text>
        </div>
        <div v-else-if="selectedEntry.error" class="detail-error">
          <n-text type="error">{{ selectedEntry.error }}</n-text>
        </div>
        <template v-else-if="selectedDemo">
          <n-descriptions bordered :column="2" size="small">
            <n-descriptions-item :label="t('main.import.detail_file_path')" :span="2">
              <n-text depth="3" style="word-break: break-all">{{ selectedDemo.file_path }}</n-text>
            </n-descriptions-item>
            <n-descriptions-item :label="t('main.import.detail_map')">{{ selectedDemo.map_name || "-" }}</n-descriptions-item>
            <n-descriptions-item :label="t('main.import.detail_server')">{{ selectedDemo.server_name || "-" }}</n-descriptions-item>
            <n-descriptions-item :label="t('main.import.detail_score')" :span="2">
              {{ (selectedDemo.clan_name_ct || "CT") }} {{ selectedDemo.score_ct }} : {{ selectedDemo.score_t }} {{ (selectedDemo.clan_name_t || "T") }}
            </n-descriptions-item>
            <n-descriptions-item :label="t('main.import.detail_duration')">{{ formatDuration(selectedDemo.duration) }}</n-descriptions-item>
            <n-descriptions-item :label="t('main.import.detail_tick_rate')">{{ selectedDemo.tick_rate > 0 ? selectedDemo.tick_rate.toFixed(1) : "-" }}</n-descriptions-item>
            <n-descriptions-item :label="t('main.import.detail_total_rounds')">{{ selectedDemo.total_rounds }}</n-descriptions-item>
            <n-descriptions-item :label="t('main.import.detail_overtime')">{{ selectedDemo.overtime_count }}</n-descriptions-item>
          </n-descriptions>
          <n-h4 style="margin: 12px 0 8px 0">{{ t('main.import.detail_players') }}</n-h4>
          <n-data-table
            :columns="playerColumns"
            :data="selectedDemo.players"
            :bordered="true"
            size="small"
            :pagination="false"
          />
        </template>
      </template>
      <n-empty v-else :description="t('main.import.detail_empty')" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, h } from "vue";
import type { DataTableColumn } from "naive-ui";
import { t } from "@/shared/i18n";
import type { DemoListEntry, DemoMetadata } from "@/shared/types";

defineEmits<{
  (e: "select-prev"): void;
  (e: "select-next"): void;
  (e: "toggle-collapse"): void;
}>();

const props = defineProps<{
  detailCollapsed: boolean;
  selectedEntry: DemoListEntry | null;
  selectedDemo: DemoMetadata | null;
  canSelectPrev: boolean;
  canSelectNext: boolean;
  formatDuration: (seconds: number) => string;
}>();

const ChevronDownIcon = {
  render: () =>
    h(
      "svg",
      { xmlns: "http://www.w3.org/2000/svg", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", "stroke-width": "2", "stroke-linecap": "round", "stroke-linejoin": "round" },
      [h("polyline", { points: "6 9 12 15 18 9" })],
    ),
};

const ChevronLeftIcon = {
  render: () =>
    h(
      "svg",
      { xmlns: "http://www.w3.org/2000/svg", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", "stroke-width": "2", "stroke-linecap": "round", "stroke-linejoin": "round" },
      [h("polyline", { points: "15 18 9 12 15 6" })],
    ),
};

const ChevronRightIcon = {
  render: () =>
    h(
      "svg",
      { xmlns: "http://www.w3.org/2000/svg", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", "stroke-width": "2", "stroke-linecap": "round", "stroke-linejoin": "round" },
      [h("polyline", { points: "9 18 15 12 9 6" })],
    ),
};

const ChevronUpIcon = {
  render: () =>
    h(
      "svg",
      { xmlns: "http://www.w3.org/2000/svg", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", "stroke-width": "2", "stroke-linecap": "round", "stroke-linejoin": "round" },
      [h("polyline", { points: "18 15 12 9 6 15" })],
    ),
};

const playerColumns = computed<DataTableColumn[]>(() => [
  { title: () => t("main.import.pcol_name"), key: "name", ellipsis: true },
  { title: () => t("main.import.pcol_kills"), key: "kills", width: 55, sorter: "default" as const },
  { title: () => t("main.import.pcol_deaths"), key: "deaths", width: 55, sorter: "default" as const },
  { title: () => t("main.import.pcol_assists"), key: "assists", width: 55, sorter: "default" as const },
]);
</script>

<style scoped>
.detail-panel {
  background: #181b19;
  border: 1px solid #303732;
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  position: sticky;
  top: 0;
  z-index: 10;
  border-bottom: 1px solid #303732;
  background: #181b19;
  padding: 12px 16px;
}

.detail-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.icon-only-btn {
  width: 26px;
  height: 26px;
}

.detail-scroll {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 12px 24px 20px;
  box-sizing: border-box;
}

.detail-scroll :deep(.n-empty) {
  padding: 24px 0;
}

.detail-loading {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 24px 0;
}

.detail-error {
  padding: 24px 0;
}

.panel-title {
  font-size: 14px;
  font-weight: 600;
}
</style>
