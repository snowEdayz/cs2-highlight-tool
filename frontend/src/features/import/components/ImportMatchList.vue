<template>
  <div class="table-section">
    <n-data-table
      ref="nTableRef"
      :columns="columns as any"
      :data="rows as any"
      :loading="loading"
      :on-scroll="handleTableScroll"
      :bordered="false"
      :flex-height="true"
      size="small"
      class="import-table"
    >
      <template #empty>
        <div class="import-table-empty" />
      </template>
    </n-data-table>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import type { DataTableColumn } from "naive-ui";

const props = defineProps<{
  columns: DataTableColumn[];
  rows: unknown[];
  loading: boolean;
  loadingMore: boolean;
  canLoadMore: boolean;
}>();

const emit = defineEmits<{
  (e: "load-more"): void;
}>();

const SCROLL_BOTTOM_THRESHOLD = 24;
const nTableRef = ref<any>(null);

let scrollContainerEl: HTMLElement | null = null;
let loadMorePending = false;
let resizeObserver: ResizeObserver | null = null;

function hasVerticalOverflow(container: HTMLElement): boolean {
  return container.scrollHeight > container.clientHeight + 1;
}

/** True when the user has scrolled close to the bottom of the container. */
function isNearScrollBottom(container: HTMLElement): boolean {
  return (
    container.scrollTop + container.clientHeight >=
    container.scrollHeight - SCROLL_BOTTOM_THRESHOLD
  );
}

/**
 * Emit a single `load-more` event after the user scrolls close to the end.
 *
 * Uses a local mutex (`loadMorePending`) to prevent multiple emissions
 * within the same animation frame – the parent's `loading` / `loadingMore` /
 * `hasMorePages` guards serve as the ultimate backstop.
 */
function emitLoadMore(): void {
  if (!scrollContainerEl) return;
  if (loadMorePending) return;
  if (props.loading) return;
  if (props.loadingMore) return;
  if (!props.canLoadMore) return;

  loadMorePending = true;
  emit("load-more");

  // Reset the mutex at the start of the next frame.
  requestAnimationFrame(() => {
    loadMorePending = false;
  });
}

/**
 * Auto-fill: keep requesting more pages until a vertical scrollbar appears
 * or there is no more data. Only runs when there is no overflow yet.
 */
function maybeAutoFill(): void {
  if (!scrollContainerEl) return;
  if (props.rows.length === 0) return;
  if (hasVerticalOverflow(scrollContainerEl)) return;
  emitLoadMore();
}

function maybeLoadMoreFromScroll(): void {
  if (!scrollContainerEl) return;
  if (!hasVerticalOverflow(scrollContainerEl)) return;
  if (!isNearScrollBottom(scrollContainerEl)) return;
  emitLoadMore();
}

function attachResizeObserver(): void {
  if (!scrollContainerEl || resizeObserver) return;
  resizeObserver = new ResizeObserver(() => {
    maybeAutoFill();
  });
  resizeObserver.observe(scrollContainerEl);
}

function findScrollContainer(): HTMLElement | null {
  const root = nTableRef.value?.$el;
  if (!root) return null;

  let best: HTMLElement | null = null;
  let bestClientHeight = 0;
  const candidates = root.querySelectorAll("div");
  // eslint-disable-next-line @typescript-eslint/prefer-for-of
  for (let i = 0; i < candidates.length; i++) {
    const el = candidates[i] as HTMLElement;
    const style = getComputedStyle(el);
    if (
      (style.overflowY === "auto" || style.overflowY === "scroll") &&
      el !== document.body &&
      el !== document.documentElement &&
      el.clientHeight > bestClientHeight
    ) {
      best = el;
      bestClientHeight = el.clientHeight;
    }
  }
  return best;
}

function ensureScrollContainer(): void {
  if (scrollContainerEl && scrollContainerEl.clientHeight > 0) return;
  const found = findScrollContainer();
  if (!found || found.clientHeight === 0) return;
  if (found === scrollContainerEl) return;
  if (resizeObserver) {
    resizeObserver.disconnect();
    resizeObserver = null;
  }
  scrollContainerEl = found;
  attachResizeObserver();
}

function handleTableScroll(event: Event): void {
  const target = event.target as HTMLElement | null;
  if (!target) return;

  if (scrollContainerEl !== target) {
    if (resizeObserver) {
      resizeObserver.disconnect();
      resizeObserver = null;
    }
    scrollContainerEl = target;
    attachResizeObserver();
  }

  maybeLoadMoreFromScroll();
}

watch(
  () => props.rows.length,
  () => {
    void nextTick(() => {
      ensureScrollContainer();
      maybeAutoFill();
    });
  },
  { flush: "post" },
);

watch(
  () => [props.canLoadMore, props.loading, props.loadingMore] as const,
  () => {
    void nextTick(() => {
      ensureScrollContainer();
      maybeAutoFill();
    });
  },
  { flush: "post" },
);

onMounted(() => {
  void nextTick(() => {
    ensureScrollContainer();
    maybeAutoFill();
  });
});

onBeforeUnmount(() => {
  if (resizeObserver) {
    resizeObserver.disconnect();
    resizeObserver = null;
  }
});
</script>

<style scoped>
.table-section {
  flex: 1 1 0;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.import-table {
  flex: 1 1 0;
  min-height: 0;
}

.import-table :deep(.n-data-table-empty) {
  padding: 0;
}

.import-table-empty {
  height: 0;
}
</style>
