import { computed, ref } from "vue";
import type { ProduceHistoryItem } from "@/shared/types";

// Kept for compatibility with existing component type imports.
export interface EditTimelineClip {
  id: string;
  historyItem: ProduceHistoryItem;
  videoPath: string;
  duration: number;
}

// Kept for compatibility with existing component type imports.
export interface EditTimelineTransition {
  id: string;
  type: "fade";
  duration: number;
  afterClipId: string;
}

export interface EditSequenceItem {
  id: string;
  historyItem: ProduceHistoryItem;
  videoPath: string;
  duration: number;
}

export type EditTransitionMode = "none" | "fade";

function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

const sequenceItems = ref<EditSequenceItem[]>([]);
const exporting = ref(false);
const exportError = ref("");
const exportPath = ref("");
const transitionMode = ref<EditTransitionMode>("none");
const transitionDuration = ref(0.3);

const totalDuration = computed(() =>
  sequenceItems.value.reduce((sum, item) => sum + item.duration, 0),
);

function addSequenceItem(item: ProduceHistoryItem, duration: number) {
  sequenceItems.value.push({
    id: generateId(),
    historyItem: item,
    videoPath: item.video_path,
    duration,
  });
}

function moveSequenceItemUp(index: number) {
  if (index <= 0 || index >= sequenceItems.value.length) return;
  const next = sequenceItems.value.slice();
  const [moved] = next.splice(index, 1);
  next.splice(index - 1, 0, moved);
  sequenceItems.value = next;
}

function moveSequenceItemDown(index: number) {
  if (index < 0 || index >= sequenceItems.value.length - 1) return;
  const next = sequenceItems.value.slice();
  const [moved] = next.splice(index, 1);
  next.splice(index + 1, 0, moved);
  sequenceItems.value = next;
}

function removeSequenceItem(index: number) {
  if (index < 0 || index >= sequenceItems.value.length) return;
  const next = sequenceItems.value.slice();
  next.splice(index, 1);
  sequenceItems.value = next;
}

function clearSequence() {
  sequenceItems.value = [];
  exporting.value = false;
  exportError.value = "";
  exportPath.value = "";
}

function setTransitionMode(mode: EditTransitionMode) {
  transitionMode.value = mode;
}

function setTransitionDuration(duration: number) {
  transitionDuration.value = duration;
}

function setExporting(value: boolean) {
  exporting.value = value;
}

function setExportError(value: string) {
  exportError.value = value;
}

function setExportPath(value: string) {
  exportPath.value = value;
}

export function useEditState() {
  return {
    sequenceItems,
    exporting,
    exportError,
    exportPath,
    transitionMode,
    transitionDuration,
    totalDuration,
    addSequenceItem,
    moveSequenceItemUp,
    moveSequenceItemDown,
    removeSequenceItem,
    clearSequence,
    setTransitionMode,
    setTransitionDuration,
    setExporting,
    setExportError,
    setExportPath,
  };
}
