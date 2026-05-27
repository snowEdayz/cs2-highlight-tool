<template>
  <div class="import-page" :class="{ 'detail-expanded': detailExpanded }">
    <div
      ref="upperRef"
      class="import-upper"
    >
      <div
        class="import-list-panel"
        :style="listPanelStyle"
      >
        <ImportDemoList
          :demo-list="demoList"
          :selected-index="selectedIndex"
          :format-duration="formatDuration"
          @select="toggleSelected"
          @remove="removeDemoAt"
        />
      </div>

      <div
        class="import-splitter"
        :class="{ dragging: isResizing }"
        @mousedown="startResize"
      />

      <div
        class="import-action-panel"
        :style="actionPanelStyle"
      >
        <router-view @demos-selected="onDemosSelected" />
      </div>
    </div>

    <div v-if="selectedEntry" class="import-detail-panel">
      <ImportDetailPanel
        :detail-collapsed="detailCollapsed"
        :selected-entry="selectedEntry"
        :selected-demo="selectedDemo"
        :can-select-prev="canSelectPrev"
        :can-select-next="canSelectNext"
        :format-duration="formatDuration"
        @select-prev="selectPrevDemo"
        @select-next="selectNextDemo"
        @toggle-collapse="toggleDetailCollapsed"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from "vue";
import ImportDemoList from "@/features/import/components/ImportDemoList.vue";
import ImportDetailPanel from "@/features/import/components/ImportDetailPanel.vue";
import { useImportDemos } from "@/features/import/composables/useImportDemos";

const {
  demoList,
  selectedIndex,
  detailCollapsed,
  selectedEntry,
  selectedDemo,
  canSelectPrev,
  canSelectNext,
  onDemosSelected,
  removeDemoAt,
  toggleSelected,
  formatDuration,
  selectPrevDemo,
  selectNextDemo,
  toggleDetailCollapsed,
} = useImportDemos();

const upperRef = ref<HTMLElement | null>(null);
const isResizing = ref(false);
const actionPanelRatio = ref(0.7);
const splitterWidthPX = 12;

const listPanelStyle = computed(() => ({
  flexBasis: `calc(${(1 - actionPanelRatio.value) * 100}% - ${(1 - actionPanelRatio.value) * splitterWidthPX}px)`,
}));

const actionPanelStyle = computed(() => ({
  flexBasis: `calc(${actionPanelRatio.value * 100}% - ${actionPanelRatio.value * splitterWidthPX}px)`,
}));

const detailExpanded = computed(() => !detailCollapsed.value && !!selectedEntry.value);

function clampActionRatio(next: number): number {
  const containerWidth = upperRef.value?.clientWidth ?? 0;
  const availableWidth = containerWidth - splitterWidthPX;
  if (containerWidth <= 0) return Math.max(0.3, Math.min(0.7, next));
  if (availableWidth <= 0) return 0.5;
  const minPanelWidth = 260;
  const minRatio = Math.max(0.2, minPanelWidth / availableWidth);
  const maxRatio = Math.min(0.8, 1 - minPanelWidth / availableWidth);
  if (minRatio > maxRatio) return 0.5;
  return Math.max(minRatio, Math.min(maxRatio, next));
}

function updateResize(clientX: number) {
  const rect = upperRef.value?.getBoundingClientRect();
  if (!rect) return;
  const splitterHalf = splitterWidthPX / 2;
  const minX = rect.left + splitterHalf;
  const maxX = rect.right - splitterHalf;
  const clampedX = Math.max(minX, Math.min(clientX, maxX));
  const availableWidth = rect.width - splitterWidthPX;
  if (availableWidth <= 0) return;
  const rightWidth = rect.right - clampedX - splitterHalf;
  const nextRatio = rightWidth / availableWidth;
  actionPanelRatio.value = clampActionRatio(nextRatio);
}

function stopResize() {
  if (!isResizing.value) return;
  isResizing.value = false;
  window.removeEventListener("mousemove", handleResizeMove);
  window.removeEventListener("mouseup", stopResize);
  document.body.style.userSelect = "";
  document.body.style.cursor = "";
}

function handleResizeMove(event: MouseEvent) {
  updateResize(event.clientX);
}

function startResize(event: MouseEvent) {
  if (event.button !== 0) return;
  event.preventDefault();
  isResizing.value = true;
  document.body.style.userSelect = "none";
  document.body.style.cursor = "col-resize";
  updateResize(event.clientX);
  window.addEventListener("mousemove", handleResizeMove);
  window.addEventListener("mouseup", stopResize);
}

onBeforeUnmount(() => {
  stopResize();
});
</script>

<style scoped>
.import-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 10px;
  min-height: 0;
}

.import-upper {
  display: flex;
  flex: 1;
  min-height: 0;
}

.import-list-panel {
  flex: 0 0 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.import-action-panel {
  flex: 0 0 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.import-splitter {
  position: relative;
  flex: 0 0 12px;
  cursor: col-resize;
}

.import-splitter::before {
  content: "";
  position: absolute;
  left: 50%;
  top: 8px;
  bottom: 8px;
  width: 2px;
  border-radius: 999px;
  background: #303732;
  transform: translateX(-50%);
  transition: background-color 0.2s ease;
}

.import-splitter:hover::before,
.import-splitter.dragging::before {
  background: #2f9462;
}

.import-detail-panel {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

/* When the detail is expanded, hide the upper area and let the detail panel
   fill the entire view. The flex chain below guarantees the inner scroll
   container actually scrolls. */
.import-page.detail-expanded .import-upper {
  display: none;
}

.import-page.detail-expanded {
  overflow: hidden;
}

.import-page.detail-expanded .import-detail-panel {
  flex: 1 1 0;
  min-height: 0;
  position: relative;
}

.import-page.detail-expanded :deep(.detail-panel) {
  position: absolute;
  inset: 0;
}
</style>
