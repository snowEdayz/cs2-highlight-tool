import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useMessage } from "naive-ui";
import { t } from "@/shared/i18n";
import { useEditState, type EditTransitionMode } from "@/features/edit/composables/useEditState";
import { EventsOn } from "../../../../wailsjs/runtime/runtime";
import type { ComposeProgressMessage } from "@/shared/types";

interface EditConcatClipPayload {
  video_path: string;
  duration: number;
}

interface EditConcatTransitionPayload {
  type: "fade";
  duration: number;
  after_index: number;
}

interface EditConcatRequestPayload {
  clips: EditConcatClipPayload[];
  transitions: EditConcatTransitionPayload[];
}

export function useEditPage() {
  const message = useMessage();
  const {
    sequenceItems,
    exporting,
    exportError,
    exportPath,
    transitionMode,
    transitionDuration,
    totalDuration,
    setTransitionMode,
    setTransitionDuration,
    setExporting,
    setExportError,
    setExportPath,
  } = useEditState();

  const composeProgress = ref<ComposeProgressMessage>({
    active: false,
    percent: 0,
    current_step: "",
    elapsed_ms: 0,
    error: "",
  });
  const offEventHandlers: Array<() => void> = [];

  const composePercent = computed(() => {
    const value = Number(composeProgress.value.percent || 0);
    return Math.max(0, Math.min(100, value));
  });

  const composeProgressLabel = computed(() => {
    const step = String(composeProgress.value.current_step || "").trim();
    if (step) return step;
    return t("main.edit.exporting");
  });

  const transitionDurationOptions = [
    { label: "0.3s", value: 0.3 },
    { label: "0.5s", value: 0.5 },
    { label: "1.0s", value: 1.0 },
  ];

  function handleTransitionModeChange(value: string | number) {
    const mode: EditTransitionMode = String(value) === "fade" ? "fade" : "none";
    setTransitionMode(mode);
  }

  function handleTransitionDurationChange(value: string | number | null) {
    const next = Number(value);
    if (!Number.isFinite(next) || next <= 0) return;
    setTransitionDuration(next);
  }

  function buildTransitions(): EditConcatTransitionPayload[] {
    if (transitionMode.value !== "fade") return [];
    if (sequenceItems.value.length <= 1) return [];
    const duration = Number(transitionDuration.value);
    const result: EditConcatTransitionPayload[] = [];
    for (let index = 0; index < sequenceItems.value.length - 1; index++) {
      result.push({
        type: "fade",
        duration,
        after_index: index,
      });
    }
    return result;
  }

  function applyComposeProgress(next: ComposeProgressMessage) {
    const percent = Number.isFinite(next.percent)
      ? Math.max(0, Math.min(100, next.percent))
      : 0;
    composeProgress.value = {
      active: !!next.active,
      percent,
      current_step: String(next.current_step || ""),
      elapsed_ms: Number.isFinite(next.elapsed_ms) ? Number(next.elapsed_ms) : 0,
      error: String(next.error || ""),
    };
  }

  async function exportSequence() {
    if (!sequenceItems.value.length || exporting.value) return;

    setExporting(true);
    setExportError("");
    setExportPath("");
    composeProgress.value = {
      active: true,
      percent: 0,
      current_step: "",
      elapsed_ms: 0,
      error: "",
    };

    try {
      const request: EditConcatRequestPayload = {
        clips: sequenceItems.value.map((item) => ({
          video_path: item.videoPath,
          duration: item.duration,
        })),
        transitions: buildTransitions(),
      };

      const result = await callBackend<string>("ConcatEditClips", request);
      setExportPath(result);
      composeProgress.value = {
        ...composeProgress.value,
        active: false,
        percent: 100,
      };
      message.success(t("main.edit.export_success", { path: basename(result) }));
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setExportError(t("main.edit.export_failed", { error: msg }));
      composeProgress.value = {
        ...composeProgress.value,
        active: false,
        error: msg,
      };
    } finally {
      setExporting(false);
    }
  }

  function clearExportError() {
    setExportError("");
  }

  async function openExportedClipFolder() {
    const target = String(exportPath.value || "").trim();
    if (!target) return;
    try {
      await callBackend<void>("OpenProducedClipInFolder", target);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      message.error(msg);
    }
  }

  onMounted(() => {
    offEventHandlers.push(
      EventsOn("compose_progress", (next: ComposeProgressMessage) => {
        applyComposeProgress(next);
      }),
    );
  });

  onBeforeUnmount(() => {
    for (const off of offEventHandlers) {
      off();
    }
    offEventHandlers.length = 0;
  });

  return {
    // Re-export from useEditState
    sequenceItems,
    exporting,
    exportError,
    exportPath,
    transitionMode,
    transitionDuration,
    totalDuration,
    // Local state
    composeProgress,
    composePercent,
    composeProgressLabel,
    transitionDurationOptions,
    // Actions
    handleTransitionModeChange,
    handleTransitionDurationChange,
    exportSequence,
    clearExportError,
    openExportedClipFolder,
    setExportPath,
    setExportError,
  };
}

function basename(path: string): string {
  if (!path) return "";
  return path.replaceAll("\\", "/").split("/").pop() || path;
}

async function callBackend<T>(method: string, ...args: unknown[]): Promise<T> {
  const api = (window as any).go?.app?.App as
    | Record<string, (...a: unknown[]) => Promise<unknown>>
    | undefined;
  const fn = api?.[method];
  if (!fn) throw new Error(`Backend not available: ${method}`);
  return fn(...args) as Promise<T>;
}
