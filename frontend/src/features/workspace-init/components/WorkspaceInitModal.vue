<template>
  <n-modal
    :show="true"
    :closable="false"
    :mask-closable="false"
    :close-on-esc="false"
    preset="card"
    :title="t('workspace.init.title')"
    style="width: 480px"
  >
    <n-space vertical :size="14">
      <div class="modal-description">{{ t("workspace.init.description") }}</div>

      <div class="path-row">
        <n-button :disabled="submitting" @click="onPick">
          {{ t("workspace.init.browse_button") }}
        </n-button>
        <n-input
          :value="path"
          readonly
          :placeholder="t('workspace.init.path_placeholder')"
        />
      </div>

      <n-text v-if="errorMessage" type="error" class="error-text">
        {{ errorMessage }}
      </n-text>
    </n-space>

    <template #footer>
      <div class="footer-row">
        <n-button :disabled="submitting" @click="onExit">
          {{ t("workspace.init.exit_button") }}
        </n-button>
        <n-button
          type="primary"
          :disabled="!canSubmit"
          :loading="submitting"
          @click="onConfirm"
        >
          {{ t("workspace.init.confirm_button") }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { useWorkspaceInit } from "@/features/workspace-init/composables/useWorkspaceInit";

const { t, path, errorMessage, submitting, canSubmit, pick, confirm, exitApp } =
  useWorkspaceInit();

async function onPick(): Promise<void> {
  await pick();
}

async function onConfirm(): Promise<void> {
  await confirm();
}

async function onExit(): Promise<void> {
  await exitApp();
}
</script>

<style scoped>
.modal-description {
  font-size: 13px;
  color: #c9d3cb;
  line-height: 1.6;
  white-space: pre-wrap;
}

.path-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.path-row .n-input {
  flex: 1;
  min-width: 0;
}

.error-text {
  font-size: 13px;
  line-height: 1.4;
  word-break: break-word;
}

.footer-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
</style>
