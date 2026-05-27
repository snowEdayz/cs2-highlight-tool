<template>
  <n-modal
    :show="show"
    :mask-closable="false"
    :closable="false"
    preset="card"
    :title="t('main.produce.platform_client.modal_title')"
    style="width: 480px"
  >
    <n-space vertical :size="12">
      <div class="modal-hint">{{ t("main.produce.platform_client.modal_hint") }}</div>

      <div class="client-list">
        <div v-for="status in statuses" :key="status.exe_name" class="client-row">
          <span class="client-name">{{ status.display_name }}</span>
          <n-tag
            :type="status.running ? 'warning' : 'success'"
            size="small"
            :bordered="false"
          >
            {{ status.running ? t("main.produce.platform_client.running") : t("main.produce.platform_client.not_running") }}
          </n-tag>
        </div>
      </div>
    </n-space>

    <template #footer>
      <n-space justify="end">
        <n-button @click="emit('cancel')">
          {{ t("main.produce.platform_client.cancel_btn") }}
        </n-button>
        <n-button :loading="refreshing" @click="refresh">
          {{ t("main.produce.platform_client.refresh_btn") }}
        </n-button>
        <n-button type="warning" :disabled="!allClosed" @click="emit('confirm')">
          {{ t("main.produce.platform_client.confirm_btn") }}
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { t } from "@/shared/i18n";
import { usePlatformClientCheck } from "../composables/usePlatformClientCheck";

defineProps<{
  show: boolean;
}>();

const emit = defineEmits<{
  (e: "confirm"): void;
  (e: "cancel"): void;
}>();

const { statuses, allClosed, refreshing, refresh } = usePlatformClientCheck();
</script>

<style scoped>
.modal-hint {
  font-size: 13px;
  color: #8d9890;
  line-height: 1.5;
}

.client-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.client-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 10px;
  border: 1px solid #2f3631;
  border-radius: 6px;
}

.client-name {
  font-size: 13px;
  color: #c8d4cc;
}
</style>
