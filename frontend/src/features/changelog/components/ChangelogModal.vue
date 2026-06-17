<template>
  <n-modal
    :show="visible"
    :mask-closable="false"
    :close-on-esc="true"
    preset="card"
    :title="modalTitle"
    class="changelog-modal"
    :style="{ width: '640px' }"
    @update:show="handleClose"
  >
    <div class="changelog-body" v-html="renderedHtml" />
    <template #footer>
      <div class="changelog-footer">
        <n-button type="primary" @click="handleClose(false)">
          {{ t("changelog.close") }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { marked } from "marked";
import DOMPurify from "dompurify";
import { t, useI18n } from "@/shared/i18n";
import type { PendingChangelog } from "@/shared/types";

const props = defineProps<{
  pending: PendingChangelog | null;
}>();

const emit = defineEmits<{
  (e: "close"): void;
}>();

const { locale } = useI18n();

const visible = computed(() => !!props.pending);

const modalTitle = computed(() => {
  const version = props.pending?.version ?? "";
  return t("changelog.title", { version });
});

const activeBody = computed(() => {
  const current = props.pending;
  if (!current) return "";
  if (locale.value === "en-US") {
    return current.body_en || current.body_zh;
  }
  return current.body_zh || current.body_en;
});

const renderedHtml = computed(() => {
  const body = activeBody.value;
  if (!body) return "";
  const html = marked.parse(body, { async: false }) as string;
  return DOMPurify.sanitize(html);
});

function handleClose(next: boolean) {
  if (next) return;
  emit("close");
}
</script>

<style scoped>
.changelog-modal :deep(.n-card__content) {
  max-height: 60vh;
  overflow-y: auto;
}

.changelog-body :deep(h1),
.changelog-body :deep(h2),
.changelog-body :deep(h3) {
  margin-top: 16px;
  margin-bottom: 8px;
  color: #edf1ee;
}

.changelog-body :deep(h3) {
  font-size: 15px;
  color: #c9d3cb;
}

.changelog-body :deep(ul) {
  padding-left: 20px;
  margin: 8px 0;
}

.changelog-body :deep(li) {
  margin: 4px 0;
  color: #c9d3cb;
  line-height: 1.6;
}

.changelog-body :deep(p) {
  margin: 8px 0;
  color: #c9d3cb;
  line-height: 1.6;
}

.changelog-body :deep(code) {
  background: #1f2421;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 13px;
}

.changelog-body :deep(a) {
  color: #34a56e;
}

.changelog-footer {
  display: flex;
  justify-content: flex-end;
}
</style>
