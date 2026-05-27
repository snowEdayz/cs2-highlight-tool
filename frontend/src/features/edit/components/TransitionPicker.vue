<template>
  <div class="transition-picker">
    <span class="picker-label">{{ t("main.edit.transition_picker") }}</span>
    <div
      v-for="opt in transitionOptions"
      :key="opt.label"
      class="transition-item"
      draggable="true"
      @dragstart="onDragStart($event, opt)"
    >
      <span class="transition-name">{{ opt.name }}</span>
      <span class="transition-dur">{{ opt.label }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { t } from "@/shared/i18n";

interface TransitionOption {
  name: string;
  label: string;
  type: "fade";
  duration: number;
}

const transitionOptions: TransitionOption[] = [
  { name: t("main.edit.fade"), label: "0.3s", type: "fade", duration: 0.3 },
  { name: t("main.edit.fade"), label: "0.5s", type: "fade", duration: 0.5 },
  { name: t("main.edit.fade"), label: "1.0s", type: "fade", duration: 1.0 },
];

function onDragStart(event: DragEvent, opt: TransitionOption) {
  event.dataTransfer!.effectAllowed = "copy";
  event.dataTransfer!.setData(
    "application/json",
    JSON.stringify({
      type: "transition",
      transitionType: opt.type,
      duration: opt.duration,
    })
  );
}
</script>

<style scoped>
.transition-picker {
  align-items: center;
  display: flex;
  gap: 8px;
  padding: 8px 12px;
  background: rgba(26, 30, 27, 0.6);
  border-bottom: 1px solid #303732;
  user-select: none;
}

.picker-label {
  color: #8d9890;
  font-size: 12px;
  margin-right: 4px;
}

.transition-item {
  align-items: center;
  background: rgba(47, 54, 49, 0.8);
  border: 1px solid #3a443d;
  border-radius: 6px;
  color: #c5cec6;
  cursor: grab;
  display: flex;
  font-size: 11px;
  gap: 6px;
  padding: 4px 10px;
  transition: background 0.15s, border-color 0.15s;
}

.transition-item:hover {
  background: rgba(47, 54, 49, 1);
  border-color: #2f9462;
}

.transition-item:active {
  cursor: grabbing;
}

.transition-name {
  font-weight: 600;
}

.transition-dur {
  color: #8d9890;
}
</style>
