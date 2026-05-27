<template>
  <div class="settings-page">
    <n-card :bordered="true" class="settings-card" content-style="height: 100%; overflow: auto;">
      <template #header>
        <div class="header-row">
          <n-button quaternary size="small" @click="goBack">
            {{ t("main.settings.back") }}
          </n-button>
          <span class="panel-title">{{ t("main.settings.title") }}</span>
          <span />
        </div>
      </template>
      <SettingsPanel :active="true" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
import { t } from "@/shared/i18n";
import SettingsPanel from "@/features/settings/components/SettingsPanel.vue";

const router = useRouter();

function goBack() {
  if (window.history.length > 1) {
    router.back();
    return;
  }
  router.push("/clips");
}
</script>

<style scoped>
.settings-page {
  height: 100%;
  min-height: 0;
}

.settings-card {
  height: 100%;
  background: #181b19;
}

.header-row {
  align-items: center;
  display: grid;
  gap: 8px;
  grid-template-columns: auto 1fr auto;
}

.panel-title {
  font-size: 14px;
  font-weight: 600;
  text-align: center;
}
</style>
