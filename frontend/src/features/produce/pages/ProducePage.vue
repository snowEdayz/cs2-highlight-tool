<template>
  <div class="produce-page">
    <n-card
      :bordered="true"
      class="produce-card"
      content-style="height: 100%; overflow: hidden; padding: 0;"
      content-class="produce-card-content"
    >
      <div class="panel-head">
        <span class="panel-title">{{ t("main.produce.title") }}</span>
      </div>
      <div class="card-body">

      <n-space vertical :size="10">
        <div v-if="!displayDemos.length" class="produce-empty">
          <n-empty :description="t('main.produce.no_selection_or_done')" />
          <n-space>
            <n-button type="primary" secondary @click="openHistoryDrawer">
              {{ t("main.produce.open_history_drawer") }}
            </n-button>
            <n-button
              v-if="hasEditableClips"
              type="success"
              secondary
              @click="goToEdit"
            >
              {{ t("main.produce.goto_edit") }}
            </n-button>
          </n-space>
        </div>

        <n-collapse v-else v-model:expanded-names="expandedNames">
          <n-collapse-item
            v-for="entry in displayDemos"
            :key="entry.key"
            :name="entry.key"
            :title="entry.file_name"
          >
            <template #header-extra>
              <n-tag size="small">{{ displayCountForDemo(entry) }}</n-tag>
            </template>

            <n-empty
              v-if="displayCountForDemo(entry) === 0"
              :description="t('main.produce.no_materials_for_demo')"
              size="small"
            />

            <n-space v-else vertical :size="8">
              <template v-if="plannedRowsForDemo(entry).length > 0">
                <n-collapse
                  :expanded-names="getPlannedRoundExpandedNames(entry)"
                  @update:expanded-names="onPlannedRoundExpandedChange(entry, $event)"
                >
                  <n-collapse-item
                    v-for="group in plannedRoundGroupsForDemo(entry)"
                    :key="`${entry.key}-planned-round-${group.name}`"
                    :name="group.name"
                    :title="plannedRoundTitle(group)"
                  >
                    <n-space vertical :size="8">
                      <div v-for="rowItem in group.rows" :key="rowItem.key" class="material-row">
                        <div class="material-main">
                          <div class="material-head">
                            <n-tag size="small" :bordered="false" :type="viewTagType(rowItem.row.view)">{{ viewLabel(rowItem.row.view) }}</n-tag>
                          </div>

                          <div v-if="rowItem.kills.length" class="take-kill-list">
                            <DeathNoticeLine v-for="kill in rowItem.kills" :key="kill.id" :kill="kill" compact />
                          </div>
                          <div v-else class="result-demo">{{ t("main.produce.kill_info_missing") }}</div>
                        </div>

                        <div class="planned-side">
                          <div class="planned-status">
                            <template v-if="isSpinningState(resolveTakeState(rowItem.row))">
                              <div class="status-spinner status-spinner--orange" />
                              <span class="result-demo">{{ statusText(resolveTakeState(rowItem.row)) }}</span>
                            </template>
                            <n-tag
                              v-else
                              size="small"
                              :type="statusTagType(resolveTakeState(rowItem.row))"
                              :bordered="false"
                            >
                              {{ statusText(resolveTakeState(rowItem.row)) }}
                            </n-tag>
                          </div>

                          <n-button
                            size="tiny"
                            quaternary
                            :disabled="!canOpenClip(rowItem.row)"
                            @click="openProducedClip(rowItem.row)"
                          >
                            {{ t("main.produce.open_clip_folder") }}
                          </n-button>

                          <div v-if="takeFileByRow(rowItem.row)?.error" class="result-error">
                            {{ takeFileByRow(rowItem.row)?.error }}
                          </div>
                        </div>
                      </div>
                    </n-space>
                  </n-collapse-item>
                </n-collapse>
              </template>

              <template v-else>
                <div
                  v-for="group in selectedRoundGroupsForDemo(entry)"
                  :key="`${entry.key}-selected-round-${group.round}`"
                  class="selected-round-group"
                >
                  <div class="selected-round-title">
                    {{ t("main.clips.round_title", { round: group.round, kills: group.items.length }) }}
                  </div>
                  <div v-for="item in group.items" :key="item.kill.id" class="material-row">
                    <div class="material-main">
                      <div class="material-line">
                        <DeathNoticeLine :kill="item.kill" compact />
                      </div>
                      <div class="selected-tags">
                        <n-tag size="small" :bordered="false" type="success">
                          {{ t("main.clips.killer_view") }}
                        </n-tag>
                        <n-tag v-if="item.include_victim" size="small" :bordered="false" type="warning">
                          {{ t("main.clips.victim_view") }}
                        </n-tag>
                      </div>
                    </div>
                  </div>
                </div>
              </template>
            </n-space>
          </n-collapse-item>
        </n-collapse>

        <n-space align="center" wrap>
          <n-button
            v-if="debugEnabled"
            type="default"
            :loading="generatingConfigOnlyLoading"
            :disabled="!hasPendingMaterials || generatingAndLaunching || generatingConfigOnlyLoading || queueState.running"
            @click="generateConfigOnly"
          >
            {{ t("main.produce.generate_all_json") }}
          </n-button>
        </n-space>

        <n-alert v-if="runtimeStateMessage" :type="runtimeStateType" :bordered="false">
          {{ runtimeStateMessage }}
        </n-alert>
        <n-alert v-if="queueState.last_error" type="error" :bordered="false">{{ queueState.last_error }}</n-alert>
        <n-alert v-if="errorMessage" type="error" :bordered="false">{{ errorMessage }}</n-alert>
      </n-space>
      </div>

      <div class="float-action-bar">
        <n-tag v-if="queueState.running" :type="wsState.connected ? 'success' : 'warning'" size="small">
          {{ wsState.connected ? t("main.produce.plugin_connected") : t("main.produce.plugin_disconnected") }}
        </n-tag>
        <n-button
          type="warning"
          :loading="generatingAndLaunching"
          :disabled="!hasPendingMaterials || generatingAndLaunching || generatingConfigOnlyLoading || queueState.running"
          @click="generateAndLaunch"
        >
          {{ t("main.produce.start_produce") }}
        </n-button>
      </div>
    </n-card>

    <PlatformClientCheckModal
      :show="showPlatformCheckModal"
      @confirm="onPlatformCheckConfirmed"
      @cancel="onPlatformCheckCancelled"
    />
  </div>
</template>

<script setup lang="ts">
import { useDebugSettings } from "@/shared/state/useDebugSettings";
import { t } from "@/shared/i18n";
import DeathNoticeLine from "@/features/clips/components/DeathNoticeLine.vue";
import PlatformClientCheckModal from "@/features/produce/components/PlatformClientCheckModal.vue";
import { useProducePage } from "@/features/produce/composables/useProducePage";

const { debugEnabled } = useDebugSettings();

const {
  generatingAndLaunching,
  generatingConfigOnlyLoading,
  expandedNames,
  queueState,
  wsState,
  displayDemos,
  hasPendingMaterials,
  hasEditableClips,
  plannedRowsForDemo,
  plannedRoundGroupsForDemo,
  getPlannedRoundExpandedNames,
  onPlannedRoundExpandedChange,
  plannedRoundTitle,
  displayCountForDemo,
  selectedRoundGroupsForDemo,
  resolveTakeState,
  statusText,
  isSpinningState,
  statusTagType,
  viewLabel,
  viewTagType,
  takeFileByRow,
  canOpenClip,
  openProducedClip,
  runtimeStateType,
  runtimeStateMessage,
  generateAndLaunch,
  generateConfigOnly,
  showPlatformCheckModal,
  onPlatformCheckConfirmed,
  onPlatformCheckCancelled,
  openHistoryDrawer,
  goToEdit,
  errorMessage,
} = useProducePage();
</script>

<style scoped>
.produce-page {
  height: 100%;
  min-height: 0;
}

.produce-card {
  height: 100%;
  background: #181b19;
}

.produce-card :deep(.produce-card-content) {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  position: relative;
}

.panel-head {
  flex-shrink: 0;
  min-height: 34px;
  padding: 6px 10px;
  border-bottom: 1px solid #303732;
  display: flex;
  align-items: center;
}

.card-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px 10px 64px;
}

.float-action-bar {
  position: absolute;
  bottom: 14px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 8px;
  z-index: 10;
  white-space: nowrap;
}

.float-action-bar :deep(.n-button) {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.55);
}

.panel-title {
  font-size: 13px;
  font-weight: 600;
}

.produce-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 16px 0;
}

.material-row {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  border: 1px solid #2f3631;
  border-radius: 8px;
  padding: 8px;
}

.material-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.material-head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.material-line {
  min-width: 0;
}

.selected-tags {
  display: flex;
  gap: 8px;
}

.selected-round-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.selected-round-title {
  font-size: 12px;
  color: #a7b2aa;
}

.take-kill-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.planned-side {
  width: 260px;
  min-width: 220px;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
}

.planned-status {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-spinner {
  width: 13px;
  height: 13px;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.28);
  border-top-color: currentColor;
  animation: spin 0.85s linear infinite;
}

.status-spinner--orange {
  color: #d09f49;
}

.result-demo {
  font-size: 12px;
  color: #8d9890;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.result-error {
  color: #e07f7f;
  font-size: 12px;
}

@media (max-width: 980px) {
  .material-row {
    flex-direction: column;
  }

  .planned-side {
    width: 100%;
    min-width: 0;
    align-items: flex-start;
  }
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
