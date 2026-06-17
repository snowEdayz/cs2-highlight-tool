<template>
  <div class="timeline-outer">
    <div ref="timelineScrollRef" class="timeline-scroll">
      <div class="timeline-inner" :style="timelineInnerStyle">
        <div class="timeline-ruler">
          <div
            v-for="mark in rulerMarks"
            :key="mark.px"
            class="ruler-mark"
            :style="{ left: mark.px + 'px' }"
            :class="{ 'ruler-major': mark.major }"
          >
            <span v-if="mark.major" class="ruler-label">{{ mark.label }}</span>
          </div>
        </div>

        <div
          ref="timelineTrackRef"
          class="timeline-track"
          :class="{ 'timeline-track--drag-over': showDropIndicator }"
          @dragenter.prevent
          @dragover.prevent="onDragOver"
          @dragleave="onDragLeave"
          @drop="onDrop"
        >
          <div
            v-for="(clip, index) in clips"
            :key="clip.id"
            class="timeline-clip"
            :class="clipColorClass(clip)"
            :style="{
              left: clipLeftPx(index) + 'px',
              width: clipWidthPx(clip) + 'px',
            }"
          >
            <span class="clip-label" :title="clipLabel(clip)">{{
              clipLabel(clip)
            }}</span>
            <button
              class="clip-remove"
              title="Remove"
              @click="$emit('removeClip', clip.id)"
            >
              &times;
            </button>
          </div>

          <div
            v-for="trans in transitions"
            :key="trans.id"
            class="timeline-transition"
            :style="{
              left: transitionLeftPx(trans) + 'px',
              width: trans.duration * PX_PER_SECOND + 'px',
            }"
            :title="`Fade ${trans.duration}s`"
          >
            <span class="trans-label">Fade</span>
          </div>

          <div
            v-if="showDropIndicator"
            class="drop-indicator"
            :style="{ left: dropIndicatorPx + 'px' }"
          />

          <div v-if="!clips.length" class="track-empty-hint">
            {{ t("main.edit.clip_empty_hint") }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { t } from "@/shared/i18n";
import type {
  EditTimelineClip,
  EditTimelineTransition,
} from "@/features/edit/composables/useEditState";
import type { ProduceHistoryItem } from "@/shared/types";

const props = defineProps<{
  clips: EditTimelineClip[];
  transitions: EditTimelineTransition[];
}>();

const emit = defineEmits<{
  addClip: [item: any, insertIndex: number];
  addTransition: [type: "fade", duration: number, afterClipId: string];
  removeClip: [clipId: string];
}>();

const PX_PER_SECOND = 80;

const timelineScrollRef = ref<HTMLElement | null>(null);
const timelineTrackRef = ref<HTMLElement | null>(null);
const showDropIndicator = ref(false);
const dropIndicatorPx = ref(0);

const totalTimelinePx = computed(() => {
  if (!props.clips.length) return 400;
  const clipPx = props.clips.reduce((sum, clip) => sum + clip.duration, 0) * PX_PER_SECOND;
  const transOverlap = props.transitions.reduce((sum, trans) => sum + trans.duration, 0) * PX_PER_SECOND;
  return Math.max(clipPx - transOverlap + 40, 400);
});

const timelineInnerStyle = computed(() => ({
  width: totalTimelinePx.value + "px",
  minWidth: "100%",
}));

const rulerMarks = computed(() => {
  const marks: Array<{ px: number; label: string; major: boolean }> = [];
  const totalSec = Math.ceil(totalTimelinePx.value / PX_PER_SECOND) + 1;
  for (let second = 0; second <= totalSec; second++) {
    marks.push({
      px: second * PX_PER_SECOND,
      label: second + "s",
      major: second % 5 === 0,
    });
  }
  return marks;
});

function clipLeftPx(index: number): number {
  let position = 0;
  for (let i = 0; i < index; i++) {
    position += props.clips[i].duration * PX_PER_SECOND;
  }
  for (const transition of props.transitions) {
    const transitionIndex = props.clips.findIndex((clip) => clip.id === transition.afterClipId);
    if (transitionIndex >= 0 && transitionIndex < index) {
      position -= transition.duration * PX_PER_SECOND;
    }
  }
  return position;
}

function clipWidthPx(clip: EditTimelineClip): number {
  return Math.max(clip.duration * PX_PER_SECOND, 40);
}

function transitionLeftPx(trans: EditTimelineTransition): number {
  const clipIndex = props.clips.findIndex((clip) => clip.id === trans.afterClipId);
  if (clipIndex < 0) return 0;
  const endOfClip = clipLeftPx(clipIndex) + clipWidthPx(props.clips[clipIndex]);
  return endOfClip - trans.duration * PX_PER_SECOND;
}

function historyItemView(item: ProduceHistoryItem): "killer" | "victim" | "full_round_pov" {
  const view = String(item.view || "").toLowerCase();
  if (view === "victim") return "victim";
  if (view === "full_round_pov") return "full_round_pov";
  if (String(item.source_id || "").toLowerCase().startsWith("full_round_pov:")) {
    return "full_round_pov";
  }
  return "killer";
}

function clipLabel(clip: EditTimelineClip): string {
  const item = clip.historyItem;
  const view = historyItemView(item);
  const viewLabel = view === "victim"
    ? t("main.clips.victim_view")
    : view === "full_round_pov"
      ? t("main.clips.full_round_pov_tag")
      : t("main.clips.killer_view");
  const kills = item.kills?.length
    ? `R${item.kills[0].round || "?"}`
    : `${item.kill_ids?.length || 0}k`;
  return `${viewLabel} ${kills} ${clip.duration.toFixed(1)}s`;
}

function clipColorClass(clip: EditTimelineClip): string {
  const view = historyItemView(clip.historyItem);
  if (view === "victim") return "timeline-clip--victim";
  if (view === "full_round_pov") return "timeline-clip--pov";
  return "timeline-clip--killer";
}

function pxToInsertIndex(px: number): number {
  for (let i = 0; i < props.clips.length; i++) {
    const midpoint = clipLeftPx(i) + clipWidthPx(props.clips[i]) / 2;
    if (px < midpoint) return i;
  }
  return props.clips.length;
}

function getDragPx(event: DragEvent): number {
  const track = timelineTrackRef.value;
  const scroll = timelineScrollRef.value;
  if (!track || !scroll) {
    return 0;
  }
  const rect = track.getBoundingClientRect();
  const raw = event.clientX - rect.left + scroll.scrollLeft;
  return Math.max(0, Math.min(raw, totalTimelinePx.value));
}

function onDragOver(event: DragEvent) {
  event.preventDefault();
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = "copy";
  }
  dropIndicatorPx.value = getDragPx(event);
  showDropIndicator.value = true;
}

function onDragLeave(event: DragEvent) {
  const track = timelineTrackRef.value;
  const related = event.relatedTarget as Node | null;
  if (track && related && track.contains(related)) {
    return;
  }
  showDropIndicator.value = false;
}

function onDrop(event: DragEvent) {
  event.preventDefault();
  showDropIndicator.value = false;

  const raw = event.dataTransfer?.getData("application/json") || "";
  if (!raw) return;

  let data: any;
  try {
    data = JSON.parse(raw);
  } catch {
    return;
  }

  const dropPx = getDragPx(event);

  if (data.type === "clip") {
    const insertIndex = pxToInsertIndex(dropPx);
    emit("addClip", data.item, insertIndex);
    return;
  }

  if (data.type === "transition") {
    const clipIdx = pxToInsertIndex(dropPx);
    if (clipIdx > 0 && clipIdx <= props.clips.length) {
      const afterClip = props.clips[clipIdx - 1];
      emit("addTransition", data.transitionType, data.duration, afterClip.id);
    }
  }
}

function scrollToEnd() {
  const scroll = timelineScrollRef.value;
  if (!scroll) return;
  scroll.scrollLeft = scroll.scrollWidth;
}

defineExpose({
  scrollToEnd,
});
</script>

<style scoped>
.timeline-outer {
  background: rgba(17, 19, 18, 0.6);
  border: 1px solid #303732;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.timeline-scroll {
  flex: 1;
  min-height: 0;
  overflow-x: auto;
  overflow-y: hidden;
}

.timeline-inner {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.timeline-ruler {
  border-bottom: 1px solid #303732;
  height: 24px;
  position: relative;
  user-select: none;
}

.ruler-mark {
  border-left: 1px solid #303732;
  height: 100%;
  position: absolute;
  top: 0;
  width: 0;
}

.ruler-mark::before {
  background: #4a584c;
  content: "";
  display: block;
  height: 6px;
  left: -1px;
  position: absolute;
  top: 0;
  width: 1px;
}

.ruler-major::before {
  height: 10px;
}

.ruler-label {
  color: #8d9890;
  font-size: 10px;
  left: 3px;
  position: absolute;
  top: 10px;
}

.timeline-track {
  background: rgba(24, 27, 25, 0.6);
  flex: 1;
  min-height: 80px;
  position: relative;
}

.timeline-track--drag-over {
  background: rgba(24, 27, 25, 0.8);
}

.track-empty-hint {
  align-items: center;
  color: #5a6860;
  display: flex;
  font-size: 13px;
  height: 100%;
  justify-content: center;
  left: 0;
  position: absolute;
  top: 0;
  width: 100%;
}

.timeline-clip {
  align-items: center;
  border-radius: 6px;
  display: flex;
  gap: 6px;
  height: 44px;
  justify-content: space-between;
  margin-top: 8px;
  padding: 0 8px;
  position: absolute;
  overflow: hidden;
  user-select: none;
}

.timeline-clip--killer {
  background: rgba(47, 148, 98, 0.3);
  border: 1px solid rgba(47, 148, 98, 0.5);
}

.timeline-clip--victim {
  background: rgba(230, 162, 60, 0.25);
  border: 1px solid rgba(230, 162, 60, 0.45);
}

.timeline-clip--pov {
  background: rgba(34, 128, 166, 0.28);
  border: 1px solid rgba(61, 174, 212, 0.5);
}

.clip-label {
  color: #edf1ee;
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.clip-remove {
  align-items: center;
  background: rgba(0, 0, 0, 0.3);
  border: none;
  border-radius: 3px;
  color: #aeb8b0;
  cursor: pointer;
  display: flex;
  flex-shrink: 0;
  font-size: 14px;
  height: 18px;
  justify-content: center;
  line-height: 1;
  padding: 0;
  width: 18px;
}

.clip-remove:hover {
  background: rgba(230, 80, 80, 0.7);
  color: #fff;
}

.timeline-transition {
  align-items: center;
  background: repeating-linear-gradient(
    -45deg,
    rgba(47, 148, 98, 0.35),
    rgba(47, 148, 98, 0.35) 3px,
    rgba(47, 148, 98, 0.15) 3px,
    rgba(47, 148, 98, 0.15) 6px
  );
  border: 1px solid rgba(47, 148, 98, 0.5);
  border-radius: 4px;
  display: flex;
  height: 44px;
  justify-content: center;
  margin-top: 8px;
  position: absolute;
  z-index: 2;
}

.trans-label {
  color: #c5cec6;
  font-size: 9px;
  font-weight: 600;
  text-transform: uppercase;
}

.drop-indicator {
  background: #2f9462;
  height: 100%;
  position: absolute;
  top: 0;
  width: 2px;
  z-index: 10;
}
</style>
