import { ref } from "vue";

const DEBUG_ACTIVATION_CLICKS = 10;

const brandClickCount = ref(0);
const debugEnabled = ref(false);
const keepProduceIntermediates = ref(false);

function activateDebugByBrandClick(): boolean {
  if (debugEnabled.value) {
    return true;
  }
  brandClickCount.value += 1;
  if (brandClickCount.value >= DEBUG_ACTIVATION_CLICKS) {
    debugEnabled.value = true;
  }
  return debugEnabled.value;
}

export function useDebugSettings() {
  return {
    debugEnabled,
    keepProduceIntermediates,
    activateDebugByBrandClick,
  };
}
