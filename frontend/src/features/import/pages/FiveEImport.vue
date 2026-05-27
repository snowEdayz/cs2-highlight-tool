<template>
  <div class="sub-page">
    <n-card
      :bordered="true"
      class="sub-card"
      content-style="height: 100%; display: flex; flex-direction: column; padding: 0;"
      content-class="fivee-sub-card-content"
    >
      <div class="panel-head">
        <div class="panel-head-row">
          <n-space align="center" :size="8" class="panel-head-left">
            <n-button
              quaternary
              circle
              size="small"
              @click="$router.push({ name: 'import-actions' })"
            >
              <template #icon>
                <n-icon size="16"><BackIcon /></n-icon>
              </template>
            </n-button>
            <span class="panel-title">{{ t("main.import.sub_5e_title") }}</span>
          </n-space>
          <div class="header-actions">
            <n-text depth="3" class="queue-text">
              {{ t("main.import.fivee_import_queue", { running: runningImportCount, queued: queuedImportCount }) }}
            </n-text>
            <n-button
              size="small"
              :loading="loading"
              :disabled="!canQuery"
              @click="refreshMatches"
            >
              {{ t("main.import.fivee_refresh") }}
            </n-button>
          </div>
        </div>
      </div>

      <div class="fivee-content">
        <div class="fivee-input-box">
          <div class="input-block">
            <n-space vertical :size="8">
              <n-text depth="3">{{ t("main.import.fivee_player_hint") }}</n-text>
              <n-space align="center" :size="8" wrap>
                <n-input
                  v-model:value="playerNameInput"
                  size="small"
                  class="player-input"
                  :placeholder="t('main.import.fivee_player_placeholder')"
                  @keyup.enter="refreshMatches"
                />
                <n-button
                  type="primary"
                  size="small"
                  :loading="loading"
                  :disabled="!canQuery"
                  @click="refreshMatches"
                >
                  {{ t("main.import.fivee_query_action") }}
                </n-button>
              </n-space>
            </n-space>
          </div>

          <div class="input-divider" />

          <div class="input-block">
            <n-space vertical :size="6">
              <n-text depth="3">{{ t("main.import.fivee_manual_hint") }}</n-text>
              <div class="manual-input-row">
                <n-input
                  v-model:value="manualMatchInput"
                  size="small"
                  class="manual-input"
                  type="textarea"
                  :autosize="{ minRows: 1, maxRows: 4 }"
                  :placeholder="t('main.import.fivee_manual_placeholder')"
                />
                <n-button
                  size="small"
                  class="manual-submit-btn"
                  type="primary"
                  :disabled="!canSubmitManualMatches"
                  @click="submitManualMatches"
                >
                  {{ t("main.import.fivee_manual_import_action") }}
                </n-button>
              </div>
            </n-space>
          </div>
        </div>

        <ImportMatchList
          :columns="columns"
          :rows="rows"
          :loading="loading"
          :loading-more="loadingMore"
          :can-load-more="hasMorePages"
          @load-more="loadMoreMatches"
        />
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { computed, h } from "vue";
import { NButton, NSpin, NTag } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { t } from "@/shared/i18n";
import ImportMatchList from "@/features/import/components/ImportMatchList.vue";
import { useFiveEImport } from "@/features/import/composables/useFiveEImport";

const emit = defineEmits<{
  (e: "demos-selected", paths: string[]): void;
}>();

function onDemosSelected(paths: string[]) {
  emit("demos-selected", paths);
}

const {
  loading,
  playerNameInput,
  manualMatchInput,
  hasMorePages,
  loadingMore,
  rows,
  runningImportCount,
  queuedImportCount,
  canSubmitManualMatches,
  canQuery,
  refreshMatches,
  loadMoreMatches,
  formatFiveERating,
  resolveTaskState,
  resolveDownloadPercent,
  isTaskPending,
  enqueueMatchImport,
  submitManualMatches,
} = useFiveEImport(onDemosSelected);

const columns = computed<DataTableColumn[]>(() => [
  {
    title: () => t("main.import.col_map"),
    key: "map_name",
    width: "20%",
    ellipsis: true,
  },
  {
    title: () => t("main.import.fivee_col_time"),
    key: "end_time",
    width: "20%",
    ellipsis: true,
  },
  {
    title: () => t("main.import.col_score"),
    key: "score",
    width: "10%",
    render: (row: any) => `${row.score1} : ${row.score2}`,
  },
  {
    title: () => t("main.import.fivee_col_kda"),
    key: "kda",
    width: "12%",
    render: (row: any) => `${row.kill}/${row.death}/${row.assist}`,
  },
  {
    title: () => t("main.import.fivee_col_rating"),
    key: "rating",
    width: "10%",
    render: (row: any) => formatFiveERating(row.rating),
  },
  {
    title: () => t("main.import.fivee_col_action"),
    key: "actions",
    width: "28%",
    fixed: "right",
    align: "center",
    render: (row: any) => {
      const matchID = row.download_match_id || row.match_id;
      const taskState = resolveTaskState(matchID);
      if (taskState === "running") {
        const percent = resolveDownloadPercent(matchID);
        return h("span", { class: "fivee-import-loading" }, [
          h(NSpin, {
            size: 14,
            stroke: "#2f9462",
          }),
          typeof percent === "number"
            ? h("span", { class: "fivee-import-percent" }, `${percent}%`)
            : null,
        ]);
      }
      if (taskState === "queued") {
        return h(
          NTag,
          {
            size: "small",
            bordered: false,
          },
          {
            default: () => t("main.import.fivee_import_queued"),
          },
        );
      }
      return h(
        NButton,
        {
          size: "tiny",
          type: "primary",
          disabled: !matchID || isTaskPending(matchID),
          onClick: () => {
            enqueueMatchImport(matchID);
          },
        },
        { default: () => t("main.import.fivee_import_action") },
      );
    },
  },
]);

const BackIcon = {
  render: () =>
    h(
      "svg",
      { xmlns: "http://www.w3.org/2000/svg", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", "stroke-width": "2", "stroke-linecap": "round", "stroke-linejoin": "round" },
      [h("polyline", { points: "15 18 9 12 15 6" })],
    ),
};
</script>

<style scoped>
.sub-page {
  height: 100%;
}

.sub-card {
  height: 100%;
  background: #181b19;
  display: flex;
  flex-direction: column;
}

.sub-card :deep(.fivee-sub-card-content) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.panel-head {
  flex-shrink: 0;
  min-height: 34px;
  padding: 6px 10px;
  border-bottom: 1px solid #303732;
}

.panel-head-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 10px;
}

.panel-head-left {
  min-width: 0;
}

.header-actions {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.queue-text {
  font-size: 12px;
  white-space: nowrap;
}

.panel-title {
  font-size: 13px;
  font-weight: 600;
}

.fivee-content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 10px 10px;
  overflow: hidden;
}

.fivee-input-box {
  border: 1px solid #2f3631;
  border-radius: 8px;
  padding: 8px;
  background: rgba(17, 19, 18, 0.45);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.input-block {
  min-width: 0;
}

.input-divider {
  height: 1px;
  background: #2f3631;
  opacity: 0.9;
}

.manual-input-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.manual-input {
  flex: 1 1 300px;
  min-width: 220px;
}

.manual-submit-btn {
  flex: 0 0 auto;
  min-width: 96px;
}

.player-input {
  min-width: 260px;
}

.fivee-import-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 22px;
}

.fivee-import-percent {
  font-size: 12px;
  color: #97a49c;
  line-height: 1;
}
</style>
