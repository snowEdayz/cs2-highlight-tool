import { ref } from "vue";
import type { DemoClipKill, GeneratePluginJSONBatchResult } from "@/shared/types";

const batchResult = ref<GeneratePluginJSONBatchResult | null>(null);
const launchViewEnabled = ref(false);
const errorMessage = ref("");
const successMessage = ref("");
const killSnapshotByDemo = ref<Record<string, DemoClipKill[]>>({});

function resetProducePageState() {
  batchResult.value = null;
  launchViewEnabled.value = false;
  errorMessage.value = "";
  successMessage.value = "";
  killSnapshotByDemo.value = {};
}

export function useProducePageState() {
  return {
    batchResult,
    launchViewEnabled,
    errorMessage,
    successMessage,
    killSnapshotByDemo,
    resetProducePageState,
  };
}
