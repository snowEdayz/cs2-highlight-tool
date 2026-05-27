<template>
  <n-card
    :bordered="true"
    class="import-list-card"
    content-style="padding: 0;"
    content-class="import-list-card-content"
  >
    <div class="panel-head">
      <span class="panel-title">{{ t("main.import.list_title") }}</span>
    </div>
    <div class="list-body">
      <template v-if="demoList.length">
        <n-data-table
          :columns="columns"
          :data="demoList"
          :bordered="false"
          :single-line="false"
          :row-class-name="rowClassName"
          :row-props="rowProps"
          size="small"
          class="demo-table"
        />
      </template>
      <n-empty
        v-else
        class="list-empty"
        :description="t('main.import.list_empty')"
      />
    </div>
  </n-card>
</template>

<script setup lang="ts">
import { computed, h } from "vue";
import { NButton, NSpin } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { t } from "@/shared/i18n";
import type { DemoListEntry } from "@/shared/types";

const props = defineProps<{
  demoList: DemoListEntry[];
  selectedIndex: number | null;
  formatDuration: (seconds: number) => string;
}>();

const emit = defineEmits<{
  (e: "select", index: number): void;
  (e: "remove", index: number): void;
}>();

function rowClassName(_row: DemoListEntry, index: number) {
  return index === props.selectedIndex ? "demo-row-active" : "";
}

function rowProps(_row: DemoListEntry, index: number) {
  return {
    style: "cursor: pointer",
    onClick: () => {
      emit("select", index);
    },
  };
}

const columns = computed<DataTableColumn<DemoListEntry>[]>(() => [
  {
    title: () => t("main.import.col_map"),
    key: "map_name",
    width: 140,
    render: (row) => row.meta?.map_name || (row.error ? "!" : "-"),
  },
  {
    title: () => t("main.import.col_score"),
    key: "score",
    width: 80,
    render: (row) => (row.meta ? `${row.meta.score_ct} : ${row.meta.score_t}` : "-"),
  },
  {
    title: () => t("main.import.col_duration"),
    key: "duration",
    width: 100,
    render: (row) => (row.meta ? props.formatDuration(row.meta.duration) : "-"),
  },
  {
    title: "",
    key: "actions",
    width: 60,
    render: (row, index) => {
      if (row.loading) {
        return h(NSpin, { size: 14, stroke: "#2f9462" });
      }
      return h(
        NButton,
        {
          size: "tiny",
          quaternary: true,
          type: "error",
          onClick: (e: MouseEvent) => {
            e.stopPropagation();
            emit("remove", index);
          },
        },
        { default: () => t("main.import.remove") },
      );
    },
  },
]);
</script>

<style scoped>
.import-list-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #181b19;
}

.import-list-card :deep(.import-list-card-content) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  position: relative;
  display: flex;
  flex-direction: column;
}

.panel-head {
  display: flex;
  align-items: center;
  min-height: 34px;
  padding: 6px 10px;
  border-bottom: 1px solid #303732;
}

.list-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.demo-table {
  width: 100%;
}

.list-empty {
  padding: 32px 0;
}

.panel-title {
  font-size: 13px;
  font-weight: 600;
}

:deep(.demo-row-active td) {
  background: rgba(47, 148, 98, 0.12) !important;
}
</style>
