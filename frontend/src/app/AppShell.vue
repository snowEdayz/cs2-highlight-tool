<template>
  <n-config-provider :theme="darkTheme" :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-dialog-provider>
        <n-notification-provider>
          <div class="app-shell">
            <TopBar />
            <div class="app-content" :class="{ 'main-mode': state.mode === 'main' }">
              <StartupWizard
                v-if="state.mode !== 'main'"
                :state="state"
                :progress-map="progressMap"
              />
              <MainApp v-else :ads="state.ads" />
            </div>
          </div>
        </n-notification-provider>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { onMounted, reactive } from "vue";
import { darkTheme, type GlobalThemeOverrides } from "naive-ui";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import MainApp from "@/app/components/MainApp.vue";
import TopBar from "@/app/components/TopBar.vue";
import StartupWizard from "@/features/startup/components/StartupWizard.vue";
import { useI18n } from "@/shared/i18n";
import type { ProgressMessage, StartupState } from "@/shared/types";

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: "#2f9462",
    primaryColorHover: "#34a56e",
    primaryColorPressed: "#268353",
    primaryColorSuppl: "#2f9462",
    bodyColor: "#111312",
    cardColor: "#181b19",
    modalColor: "#181b19",
    popoverColor: "#1f2421",
    borderColor: "#303732",
    dividerColor: "#303732",
    textColorBase: "#edf1ee",
    textColor1: "#edf1ee",
    textColor2: "#c9d3cb",
    textColor3: "#8d9890",
    successColor: "#2f9462",
    successColorHover: "#34a56e",
    successColorPressed: "#268353",
    warningColor: "#d09f49",
    errorColor: "#a94f4f",
    fontFamily:
      '"Barlow Semi Condensed","Noto Sans SC","PingFang SC","Microsoft YaHei",sans-serif',
    borderRadius: "10px",
  },
  Card: {
    borderColor: "#303732",
    color: "#181b19",
  },
  Alert: {
    borderRadius: "10px",
  },
  Tag: {
    borderRadius: "999px",
  },
};

const { t } = useI18n();

const state = reactive<StartupState>({
  mode: "startup",
  phase: "detecting_source",
  running: false,
  source_step: {
    status: "pending",
    source: "custom",
    country_code: "",
    message: t("app.source_waiting"),
    error: "",
  },
  fatal_error: "",
  entry_notice: "",
  ads: [],
  self_update: {
    status: "pending",
    available: false,
    current: "0.0.0",
    latest: "",
    url: "",
    asset_url: "",
    error: "",
  },
  steps: [],
  can_enter_main: false,
  config: {},
});

const progressMap = reactive<Record<string, ProgressMessage>>({});

function isActiveStatus(status: string | undefined): boolean {
  return ["checking", "downloading", "installing"].includes(status || "");
}

function statusForProgressKey(key: string): string {
  if (key === "self_update") {
    return state.self_update.status || "";
  }
  const step = state.steps.find((item) => item.id === key);
  return step?.status || "";
}

function applyState(next: StartupState) {
  const wasRunning = state.running;
  Object.assign(state, next);

  if (!wasRunning && state.running) {
    for (const key of Object.keys(progressMap)) {
      delete progressMap[key];
    }
  }

  for (const [key, progress] of Object.entries(progressMap)) {
    if (!isActiveStatus(statusForProgressKey(key)) && !progress.active) {
      delete progressMap[key];
    }
  }
}

async function callBackend(method: string, ...args: unknown[]) {
  const api = window.go?.app?.App as Record<string, (...args: unknown[]) => Promise<unknown>> | undefined;
  const fn = api?.[method];
  if (!fn) {
    throw new Error(`Wails API not loaded: ${method}`);
  }
  return fn(...args);
}

onMounted(async () => {
  EventsOn("startup_state_changed", (next: StartupState) => {
    applyState(next);
  });

  EventsOn("download_progress", (next: ProgressMessage) => {
    if (!next.component_id) return;
    const safePercent = Number.isFinite(next.percent)
      ? Math.max(0, Math.min(100, next.percent))
      : 0;
    progressMap[next.component_id] = {
      ...next,
      percent: safePercent,
    };
  });

  try {
    applyState((await callBackend("GetStartupState")) as StartupState);
    await callBackend("RunStartupChecks");
  } catch {
    // startup errors are surfaced through backend state/log events
  }
});
</script>

<style scoped>
.app-shell {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

.app-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.app-content.main-mode {
  overflow: hidden;
}
</style>
