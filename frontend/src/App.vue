<template>
  <n-config-provider :theme="darkTheme">
    <n-message-provider>
      <div class="app">
        <n-space vertical size="large">
          <n-card ref="headerCard">
            <div class="header-top">
              <n-h2 class="header-title">{{ t("app.title") }}</n-h2>
              <div class="header-icons">
                <button class="icon-link" type="button" @click="openExternal(`mailto:${author.email}`)" :aria-label="t('aria.email')">
                  <svg viewBox="0 0 24 24" class="icon" aria-hidden="true">
                    <path
                      fill="currentColor"
                      d="M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 4-8 5-8-5V6l8 5 8-5v2z"
                    />
                  </svg>
                </button>
                <button class="icon-link" type="button" @click="openExternal(author.github)" :aria-label="t('aria.github')">
                  <svg viewBox="0 0 24 24" class="icon" aria-hidden="true">
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.58 2 12.26c0 4.5 2.87 8.31 6.84 9.66.5.09.68-.22.68-.48 0-.24-.01-.87-.01-1.7-2.78.62-3.37-1.38-3.37-1.38-.45-1.19-1.11-1.5-1.11-1.5-.9-.64.07-.63.07-.63 1 .07 1.53 1.06 1.53 1.06.89 1.57 2.34 1.12 2.91.86.09-.67.35-1.12.63-1.38-2.22-.26-4.56-1.15-4.56-5.12 0-1.13.39-2.05 1.03-2.77-.1-.26-.45-1.31.1-2.73 0 0 .84-.28 2.75 1.06.8-.23 1.66-.34 2.52-.34s1.72.12 2.52.34c1.9-1.34 2.75-1.06 2.75-1.06.55 1.42.2 2.47.1 2.73.64.72 1.03 1.64 1.03 2.77 0 3.98-2.34 4.85-4.57 5.1.36.33.68.97.68 1.96 0 1.41-.01 2.55-.01 2.9 0 .26.18.58.69.48 3.97-1.35 6.83-5.16 6.83-9.66C22 6.58 17.52 2 12 2z"
                    />
                  </svg>
                </button>
                <button class="icon-link" type="button" @click="openExternal(author.blog)" :aria-label="t('aria.blog')">
                  <svg viewBox="0 0 24 24" class="icon" aria-hidden="true">
                    <path
                      fill="currentColor"
                      d="M11 18h2v-2h-2v2zm1-16C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm0-14c-2.21 0-4 1.79-4 4h2a2 2 0 1 1 2 2c-1.1 0-2 .9-2 2v1h2v-1c0-.55.45-1 1-1 1.66 0 3-1.34 3-3 0-2.21-1.79-4-4-4z"
                    />
                  </svg>
                </button>
              </div>
            </div>
            <div class="header-bottom">
              <n-text depth="3" class="version-text">{{ appVersion }}</n-text>
              <n-text depth="3" class="stats-text">{{ usageStatsText }}</n-text>
              <n-select v-model:value="locale" size="small" class="lang-select" :options="languageOptions" />
            </div>
          </n-card>

          <div :class="['status-bar', { fixed: statusFixed }]">
            <n-alert :type="statusType" :bordered="false">
              {{ statusText }}
            </n-alert>
          </div>
          <div v-if="statusFixed" class="status-spacer"></div>

          <template v-if="!isSetupComplete">
            <n-card :title="t('setup.title')">
              <n-space vertical>
                <n-text>{{ t("setup.description") }}</n-text>
                <n-form label-placement="top" size="small">
                  <n-grid :cols="2" :x-gap="16" :y-gap="12" responsive="screen">
                    <n-form-item-gi :label="t('setup.cs2_label')" :class="{ 'needs-attention': needsCS2Highlight }">
                      <n-input v-model:value="form.cs2_exe" :placeholder="t('setup.cs2_placeholder')">
                        <template #suffix>
                          <n-button size="tiny" @click="pickCs2Exe" :disabled="isPreparingEnv" :class="{ 'needs-attention': needsCS2Highlight }">
                            {{ t("common.select") }}
                          </n-button>
                        </template>
                      </n-input>
                    </n-form-item-gi>
                    <n-form-item-gi :label="t('setup.hlae_label')">
                      <n-input v-model:value="form.hlae_exe" :placeholder="t('setup.hlae_placeholder')" disabled />
                    </n-form-item-gi>
                    <n-form-item-gi :label="t('setup.ffmpeg_label')">
                      <n-input v-model:value="form.ffmpeg_dir" :placeholder="t('setup.ffmpeg_placeholder')" disabled />
                    </n-form-item-gi>
                  </n-grid>
                </n-form>
                <n-space class="actions">
                  <n-button type="primary" @click="confirmSetup" :disabled="isPreparingEnv">
                    {{ t("setup.confirm") }}
                  </n-button>
                </n-space>
              </n-space>
            </n-card>

            <n-card :title="t('common.logs')">
              <n-log ref="logRef" :log="logContent" :rows="12" />
            </n-card>
          </template>

          <n-grid v-else :cols="2" :x-gap="16" :y-gap="16" responsive="screen">
            <n-gi>
              <n-space vertical size="large">
                <n-card :title="t('config.title')">
                  <n-form label-placement="top" size="small">
                    <n-grid :cols="2" :x-gap="16" :y-gap="12" responsive="screen">
                      <n-form-item-gi :label="t('config.output_dir')">
                        <n-input v-model:value="form.output_dir" :placeholder="t('config.output_placeholder')">
                          <template #suffix>
                            <n-button size="tiny" @click="pickOutputDir">{{ t("common.select") }}</n-button>
                          </template>
                        </n-input>
                      </n-form-item-gi>
                      <n-form-item-gi :label="t('config.record_fps')">
                        <n-input-number v-model:value="form.record_fps" :min="1" />
                      </n-form-item-gi>
                      <n-form-item-gi :label="t('config.tickrate')">
                        <n-input-number v-model:value="form.tickrate" :min="1" />
                      </n-form-item-gi>
                      <n-form-item-gi :label="t('config.video_preset')">
                        <n-select v-model:value="form.video_preset" :options="presetOptions" />
                      </n-form-item-gi>
                      <n-form-item-gi :label="t('config.launch_resolution')">
                        <n-select v-model:value="form.launch_resolution" :options="resolutionOptions" />
                      </n-form-item-gi>
                      <n-form-item-gi :label="t('config.transition_duration')">
                        <n-input-number v-model:value="form.transition_duration" :min="0" :step="0.1" />
                      </n-form-item-gi>
                      <n-form-item-gi :label="t('config.transition_type')" :span="2">
                        <n-select v-model:value="form.transition_type" :options="transitionOptions" />
                      </n-form-item-gi>
                      <n-form-item-gi :label="t('config.killer_pre_seconds')">
                        <n-input-number v-model:value="form.killer_pre_seconds" :min="1" :step="1" />
                      </n-form-item-gi>
                      <n-form-item-gi :label="t('config.killer_post_seconds')">
                        <n-input-number v-model:value="form.killer_post_seconds" :min="1" :step="1" />
                      </n-form-item-gi>
                      <n-form-item-gi :label="t('config.record_victim_view')" :span="2">
                        <n-switch v-model:value="form.record_victim_view"/>
                      </n-form-item-gi>
                      <n-form-item-gi v-if="form.record_victim_view" :label="t('config.victim_pre_seconds')">
                        <n-input-number v-model:value="form.victim_pre_seconds" :min="1" :step="1" />
                      </n-form-item-gi>
                      <n-form-item-gi v-if="form.record_victim_view" :label="t('config.victim_post_seconds')">
                        <n-input-number v-model:value="form.victim_post_seconds" :min="1" :step="1" />
                      </n-form-item-gi>
                    </n-grid>
                  </n-form>
                  <n-space class="actions">
                    <n-button type="primary" @click="saveConfig">{{ t("config.save") }}</n-button>
                  </n-space>
                </n-card>

                <n-card :title="t('preview.title')">
                  <div v-if="lastOutputPath" class="video-preview">
                    <video v-if="lastOutputUrl" :key="lastOutputUrl" :src="lastOutputUrl" controls preload="metadata"></video>
                    <n-text v-else depth="3">{{ t("preview.unavailable") }}</n-text>
                    <n-space align="center">
                      <n-button size="small" @click="openVideoExternal">{{ t("preview.open_external") }}</n-button>
                    </n-space>
                    <n-text depth="3" class="video-path">{{ lastOutputPath }}</n-text>
                  </div>
                  <n-text v-else depth="3">{{ t("preview.no_video") }}</n-text>
                </n-card>
              </n-space>
            </n-gi>

            <n-gi>
              <n-card :title="t('demo.title')">
                <n-space vertical>
                  <n-input
                    v-model:value="perfectMatchId"
                    :placeholder="t('demo.perfect_id_placeholder')"
                    :disabled="isParsing || isDownloadingDemo || !isEnvReady"
                  >
                    <template #suffix>
                      <n-button size="tiny" @click="downloadPerfectWorldDemo" :disabled="isParsing || isDownloadingDemo || !isEnvReady">
                        {{ t("demo.download_parse") }}
                      </n-button>
                    </template>
                  </n-input>
                  <n-input v-model:value="demoPath" :placeholder="t('demo.select_placeholder')" :disabled="!isEnvReady">
                    <template #suffix>
                      <n-button size="tiny" @click="pickDemo" :disabled="!isEnvReady">{{ t("demo.select_button") }}</n-button>
                    </template>
                  </n-input>
                  <n-space>
                    <n-select
                      v-model:value="selectedPlayerSteamId"
                      :placeholder="t('demo.select_player')"
                      :options="playerOptions"
                      @update:value="refreshRounds"
                      style="min-width: 260px"
                      :disabled="isParsing || !isEnvReady"
                    />
                    <n-button size="small" @click="toggleExpandAll" :disabled="!rounds.length || !isEnvReady">
                      {{ expandAllLabel }}
                    </n-button>
                  </n-space>

                  <div class="rounds">
                    <n-card v-for="round in rounds" :key="round.round" size="small" class="round-card">
                      <template #header>
                        <n-space align="center">
                          <n-checkbox
                            :checked="selectedRounds.has(round.round)"
                            @update:checked="(val) => toggleRound(round.round, val)"
                            :disabled="isParsing || !isEnvReady"
                          />
                          <n-text>{{ t("demo.round_label", { round: round.round, kills: round.kills.length }) }}</n-text>
                        </n-space>
                      </template>
                      <n-collapse v-model:expanded-names="expandedRounds">
                        <n-collapse-item :title="t('demo.view_kill_details')" :name="round.round">
                          <DeathNoticeList :kills="round.kills" />
                        </n-collapse-item>
                      </n-collapse>
                    </n-card>
                  </div>

                  <n-space class="actions">
                    <n-button type="primary" @click="runWorkflow(true)" :disabled="isParsing || !isEnvReady">{{ t("demo.make") }}</n-button>
                  </n-space>
                </n-space>
              </n-card>
            </n-gi>

            <n-gi :span="2">
              <n-card :title="t('common.logs')">
                <n-log ref="logRef" :log="logContent" :rows="12" />
              </n-card>
            </n-gi>
          </n-grid>
        </n-space>
        <div class="scroll-fab">
          <n-button
            type="primary"
            size="small"
            @click="handleScrollJump"
            :aria-label="isAtTop ? t('scroll.to_bottom') : t('scroll.to_top')"
          >
            {{ scrollFabIcon }}
          </n-button>
        </div>
      </div>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup>
import { computed, h, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { createDiscreteApi, darkTheme } from "naive-ui";
import { useI18n } from "vue-i18n";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { BrowserOpenURL, Quit } from "../wailsjs/runtime/runtime";
import wailsConfig from "../../wails.json";
import * as AppApi from "../wailsjs/go/main/App";
import DeathNoticeList from "./components/DeathNoticeList.vue";

const { t, locale } = useI18n();

const demoPath = ref("");
const perfectMatchId = ref("");
const demoInfo = ref(null);
const selectedPlayerSteamId = ref(null);
const selectedRounds = ref(new Set());
const logs = ref([]);
const logRef = ref(null);
const author = {
  email: "hk_snow@yeah.net",
  github: "https://github.com/hkslover/cs2-highlight-tool",
  blog: "https://snowblog.xyz/posts/cs2-highlight-tool-faqs/",
};
const appVersion = `v${wailsConfig.version || "0.0.0"}`;
const runCount = ref(null);
const makeCount = ref(null);
const statusKey = ref("status.ready");
const statusText = computed(() => t(statusKey.value));
const statusType = computed(() => (statusKey.value === "status.ready" ? "success" : "info"));
const isParsing = ref(false);
const isDownloadingDemo = ref(false);
const isCheckingEnv = ref(false);
const isPreparingEnv = ref(false);
const isEnvReady = ref(false);
const needsCS2Path = ref(false);
const isSetupComplete = ref(false);
const lastOutputPath = ref("");
const lastOutputUrl = ref("");
const expandedRounds = ref([]);
const headerCard = ref(null);
const statusFixed = ref(false);
const isAtTop = ref(true);
const { message, dialog } = createDiscreteApi(["message", "dialog"], {
  configProviderProps: {
    theme: darkTheme,
  },
});
let configReady = false;
let savingConfig = false;

const form = reactive({
  cs2_exe: "",
  hlae_exe: "",
  hlae_version: "",
  ffmpeg_dir: "",
  cfg_dir: "",
  output_dir: "",
  record_fps: 60,
  tickrate: 64,
  video_preset: "n1",
  transition_duration: 1,
  transition_type: "fade",
  launch_resolution: "16:9",
  record_victim_view: false,
  killer_pre_seconds: 5,
  killer_post_seconds: 5,
  victim_pre_seconds: 1,
  victim_post_seconds: 1,
});

const presetOptions = [
  { label: "n1 - NVENC (hevc_nvenc)", value: "n1" },
  { label: "c1 - CPU (libx264)", value: "c1" },
];

const transitionOptions = [
  { label: "fade", value: "fade" },
  { label: "wipeleft", value: "wipeleft" },
  { label: "slideright", value: "slideright" },
  { label: "circleopen", value: "circleopen" },
];

const resolutionOptions = computed(() => [
  { label: t("config.resolution_16_9"), value: "16:9" },
  { label: t("config.resolution_4_3"), value: "4:3" },
]);

const languageOptions = computed(() => [
  { label: t("language.zh"), value: "zh" },
  { label: t("language.en"), value: "en" },
]);

const usageStatsText = computed(() => {
  const runValue = runCount.value ?? "-";
  const makeValue = makeCount.value ?? "-";
  return t("app.usage_stats", { run: runValue, make: makeValue });
});

const players = computed(() => demoInfo.value?.players || []);
const playerOptions = computed(() =>
  players.value.map((p) => ({
    label: t("demo.player_option", { name: p.name, kills: p.total_kills }),
    value: p.steam_id,
  }))
);

const rounds = computed(() => {
  const player = players.value.find((p) => p.steam_id === selectedPlayerSteamId.value);
  return player?.rounds || [];
});

const needsCS2Highlight = computed(() => !isSetupComplete.value && needsCS2Path.value && !form.cs2_exe);

let lastLogMessage = "";

function normalizeLogMessage(message) {
  if (!message) return "";
  let text = String(message).trim();
  if (text.startsWith("===") && text.endsWith("===")) {
    text = text.replace(/^=+\s*/, "").replace(/\s*=+$/, "");
  }
  if (text.startsWith("✓ ")) {
    text = t("log.success_prefix", { message: text.replace(/^✓\s*/, "") });
  } else if (text.startsWith("✗ ")) {
    text = t("log.fail_prefix", { message: text.replace(/^✗\s*/, "") });
  }
  return text;
}

function logLine(message, level = "info") {
  const text = normalizeLogMessage(message);
  if (!text) return;
  if (text === lastLogMessage) return;
  lastLogMessage = text;
  logs.value.push({
    message: text,
    level,
    time: new Date().toLocaleTimeString(),
  });
  if (logs.value.length > 500) {
    logs.value.shift();
  }
}

async function fetchStats() {
  try {
    const data = await callBackend("GetUsageStats");
    if (typeof data?.run === "number") runCount.value = data.run;
    if (typeof data?.make === "number") makeCount.value = data.make;
  } catch (_) {
    return;
  }
}

async function incrementRunCount() {
  try {
    const data = await callBackend("IncrementRunCount");
    if (typeof data?.counts === "number") {
      runCount.value = data.counts;
    }
  } catch (_) {
    return;
  }
}

async function incrementMakeCount() {
  try {
    const data = await callBackend("IncrementMakeCount");
    if (typeof data?.counts === "number") {
      makeCount.value = data.counts;
    }
  } catch (_) {
    return;
  }
}

const logContent = computed(() =>
  logs.value.map((line) => `[${line.time}] ${line.message}`).join("\n")
);

watch(
  logContent,
  async () => {
    await nextTick();
    logRef.value?.scrollTo?.({ position: "bottom", silent: true });
  },
  { flush: "post" }
);

watch(locale, (value) => {
  if (value) {
    localStorage.setItem("locale", value);
  }
});

function setStatus(key) {
  statusKey.value = key || "status.ready";
}

function openExternal(url) {
  if (!url) return;
  BrowserOpenURL(url);
}

function formatError(err) {
  if (!err) return t("common.unknown_error");
  if (typeof err === "string") return err;
  if (err?.message) return err.message;
  try {
    return JSON.stringify(err);
  } catch (_) {
    return String(err);
  }
}

function callBackend(method, ...args) {
  if (typeof AppApi[method] === "function") {
    return AppApi[method](...args);
  }
  return Promise.reject(new Error(t("common.wails_api_not_loaded", { method })));
}

function fillConfig(cfg) {
  if (!cfg) return;
  form.cs2_exe = cfg.cs2_exe || "";
  form.hlae_exe = cfg.hlae_exe || "";
  form.hlae_version = cfg.hlae_version || "";
  form.ffmpeg_dir = cfg.ffmpeg_dir || "";
  form.cfg_dir = cfg.cfg_dir || "";
  form.output_dir = cfg.output_dir || "";
  form.record_fps = cfg.record_fps || 60;
  form.tickrate = cfg.tickrate || 64;
  form.video_preset = cfg.video_preset || "n1";
  form.transition_duration = cfg.transition_duration || 1;
  form.transition_type = cfg.transition_type || "fade";
  form.launch_resolution = cfg.launch_resolution || "16:9";
  form.record_victim_view = cfg.record_victim_view || false;
  form.killer_pre_seconds = cfg.killer_pre_seconds || 5;
  form.killer_post_seconds = cfg.killer_post_seconds || 5;
  form.victim_pre_seconds = cfg.victim_pre_seconds || 1;
  form.victim_post_seconds = cfg.victim_post_seconds || 1;
}

async function loadConfig() {
  const cfg = await callBackend("GetConfig");
  fillConfig(cfg);
}

async function pickCs2Exe() {
  try {
    const path = await callBackend("PickCS2Exe");
    if (!path) return;
    form.cs2_exe = path;
  } catch (err) {
    logLine(formatError(err), "error");
  }
}

async function pickHlaeExe() {
  try {
    const path = await callBackend("PickHLAEExe");
    if (!path) return;
    form.hlae_exe = path;
  } catch (err) {
    logLine(formatError(err), "error");
  }
}

async function pickDemo() {
  if (!isEnvReady.value) {
    message.warning(t("warning.env_not_ready"));
    return;
  }
  try {
    const path = await callBackend("PickDemo");
    if (!path) return;
    demoPath.value = path;
    await tryParseDemo();
  } catch (err) {
    logLine(formatError(err), "error");
  }
}

async function downloadPerfectWorldDemo() {
  if (!isEnvReady.value) {
    message.warning(t("warning.env_not_ready"));
    return;
  }
  const matchId = String(perfectMatchId.value || "").trim();
  if (!matchId) {
    message.warning(t("warning.enter_match_id"));
    return;
  }
  try {
    isDownloadingDemo.value = true;
    setStatus("status.downloading_demo");
    logLine(t("info.downloading_demo_start"));
    const path = await callBackend("DownloadPerfectWorldDemo", matchId);
    demoPath.value = path;
    await tryParseDemo();
  } catch (err) {
    const msg = formatError(err);
    message.error(msg);
    logLine(msg, "error");
    setStatus("status.download_failed");
  } finally {
    isDownloadingDemo.value = false;
    setTimeout(() => setStatus("status.ready"), 800);
  }
}

async function pickOutputDir() {
  try {
    const path = await callBackend("PickOutputDir");
    if (!path) return;
    form.output_dir = path;
  } catch (err) {
    logLine(formatError(err), "error");
  }
}

async function saveConfig(silent = false) {
  if (!configReady || savingConfig) return;
  savingConfig = true;
  const isSilent = silent === true;
  try {
    const saved = await callBackend("SaveConfig", { ...form });
    fillConfig(saved);
    if (!isSilent) {
      message.success(t("info.config_saved"));
      logLine(t("info.config_saved"), "success");
      setStatus("status.config_saved");
    }
  } catch (err) {
    const msg = formatError(err);
    message.error(msg);
    logLine(msg, "error");
  } finally {
    savingConfig = false;
  }
}

async function updatePreviewUrl(path) {
  if (!path) {
    lastOutputUrl.value = "";
    return;
  }
  try {
    const url = await callBackend("GetVideoPreviewURL", path);
    lastOutputUrl.value = url;
  } catch (err) {
    lastOutputUrl.value = "";
    logLine(t("info.preview_unavailable", { error: formatError(err) }), "warning");
  }
}

async function openVideoExternal() {
  if (!lastOutputPath.value) return;
  try {
    await callBackend("OpenVideoExternal", lastOutputPath.value);
  } catch (err) {
    message.error(formatError(err));
  }
}

async function tryParseDemo() {
  if (!isEnvReady.value) {
    logLine(t("warning.env_not_ready"), "warning");
    return;
  }
  if (!demoPath.value) {
    logLine(t("warning.select_demo"), "warning");
    return;
  }
  try {
    isParsing.value = true;
    setStatus("status.parsing_demo");
    const info = await callBackend("ParseDemo", demoPath.value);
    demoInfo.value = info;
    selectedPlayerSteamId.value = info.players?.[0]?.steam_id ?? null;
    selectedRounds.value = new Set();
    expandedRounds.value = [];
    logLine(t("info.demo_parsed"), "success");
    setStatus("status.parse_done");
  } catch (err) {
    const msg = formatError(err);
    message.error(msg);
    logLine(msg, "error");
    setStatus("status.parse_failed");
  } finally {
    isParsing.value = false;
    setTimeout(() => setStatus("status.ready"), 800);
  }
}

function refreshRounds() {
  selectedRounds.value = new Set();
}

function toggleRound(round, checked) {
  const next = new Set(selectedRounds.value);
  if (checked) {
    next.add(round);
  } else {
    next.delete(round);
  }
  selectedRounds.value = next;
}

const allExpanded = computed(() => {
  return rounds.value.length > 0 && expandedRounds.value.length === rounds.value.length;
});

const expandAllLabel = computed(() => (allExpanded.value ? t("demo.collapse_all") : t("demo.expand_all")));

const scrollFabIcon = computed(() => (isAtTop.value ? "↓" : "↑"));

function updateScrollState() {
  const top = window.scrollY || document.documentElement.scrollTop || 0;
  isAtTop.value = top <= 2;
}

function handleScrollJump() {
  if (isAtTop.value) {
    window.scrollTo({ top: document.documentElement.scrollHeight, behavior: "smooth" });
  } else {
    window.scrollTo({ top: 0, behavior: "smooth" });
  }
}

function toggleExpandAll() {
  if (allExpanded.value) {
    expandedRounds.value = [];
  } else {
    expandedRounds.value = rounds.value.map((r) => r.round);
  }
}

function buildMakeInfoText() {
  let text = t("info.make_tip_base");
  if (form.record_victim_view) {
    text += `\n${t("info.make_tip_victim")}`;
  }
  return text;
}

function showMakeInfoDialog() {
  return new Promise((resolve) => {
    dialog.info({
      title: t("info.make_title"),
      content: () => h("div", { style: "white-space: pre-line" }, buildMakeInfoText()),
      positiveText: t("common.ok"),
      onPositiveClick: () => resolve(true),
    });
  });
}

async function runWorkflow(autoMode) {
  if (!isEnvReady.value) {
    message.warning(t("warning.env_not_ready"));
    return;
  }
  await saveConfig(true);
  const player = players.value.find((p) => p.steam_id === selectedPlayerSteamId.value);
  if (!player) {
    logLine(t("warning.select_player"), "warning");
    return;
  }
  if (!demoPath.value) {
    logLine(t("warning.select_demo"), "warning");
    return;
  }
  const selected = Array.from(selectedRounds.value);
  if (!selected.length) {
    logLine(t("warning.select_rounds"), "warning");
    return;
  }

  await showMakeInfoDialog();
  try {
    setStatus(autoMode ? "status.generating_cfg_and_record" : "status.generating_cfg");
    const res = await callBackend("RunWorkflow", {
      demo_path: demoPath.value,
      player_steam_id: player.steam_id,
      selected_rounds: selected,
      auto_mode: autoMode,
      debug_mode: false,
    });
    if (res?.cfg_path) logLine(t("info.cfg_generated", { path: res.cfg_path }), "success");
    if (res?.output_path) {
      logLine(t("info.output_video", { path: res.output_path }), "success");
      lastOutputPath.value = res.output_path;
      await updatePreviewUrl(res.output_path);
      await incrementMakeCount();
    }
    setStatus("status.task_done");
  } catch (err) {
    const msg = formatError(err);
    message.error(msg);
    logLine(msg, "error");
    setStatus("status.task_failed");
  } finally {
    setTimeout(() => setStatus("status.ready"), 1200);
  }
}

async function checkEnvironment() {
  try {
    if (isCheckingEnv.value) return;
    isCheckingEnv.value = true;
    isEnvReady.value = false;
    needsCS2Path.value = false;
    setStatus("status.checking_env");
    await callBackend("CheckEnvironment");
    await loadConfig();
    logLine(t("info.env_checked"), "success");
    message.success(t("info.env_checked"));
    isEnvReady.value = true;
    isSetupComplete.value = true;
  } catch (err) {
    const msg = formatError(err);
    message.error(msg);
    logLine(msg, "error");
    if (msg.includes("程序路径包含中文")) {
      const path = msg.replace(/^.*?:\s*/, "");
      dialog.warning({
        title: t("warning.path_has_cjk_title"),
        content: () => h("div", { style: "white-space: pre-line" }, t("warning.path_has_cjk_desc", { path })),
        positiveText: t("common.ok"),
      });
    }
    if (msg.includes("CS2 未找到")) {
      needsCS2Path.value = true;
      setStatus("status.need_cs2");
      message.warning(t("warning.set_cs2"));
      isSetupComplete.value = false;
    }
  } finally {
    isCheckingEnv.value = false;
    if (!needsCS2Path.value) {
      setStatus("status.ready");
    }
  }
}

async function prepareEnvironment() {
  try {
    if (isPreparingEnv.value) return;
    isPreparingEnv.value = true;
    setStatus("status.preparing_env");
    const cfg = await callBackend("PrepareEnvironment", false);
    fillConfig(cfg);
    configReady = true;
    if (form.cs2_exe) {
      await checkEnvironment();
    } else {
      needsCS2Path.value = true;
      setStatus("status.need_cs2");
    }
  } catch (err) {
    const msg = formatError(err);
    message.error(msg);
    logLine(msg, "error");
    if (msg.includes("程序路径包含中文")) {
      const path = msg.replace(/^.*?:\s*/, "");
      dialog.warning({
        title: t("warning.path_has_cjk_title"),
        content: () => h("div", { style: "white-space: pre-line" }, t("warning.path_has_cjk_desc", { path })),
        positiveText: t("common.ok"),
        onPositiveClick: () => {
          Quit();
        },
      });
      return;
    }
    setStatus("status.prepare_failed");
  } finally {
    isPreparingEnv.value = false;
  }
}

async function confirmSetup() {
  if (!form.cs2_exe) {
    message.warning(t("warning.set_cs2"));
    return;
  }
  await saveConfig();
  await checkEnvironment();
}

onMounted(async () => {
  if (headerCard.value?.$el) {
    const observer = new IntersectionObserver(
      (entries) => {
        const entry = entries[0];
        statusFixed.value = entry ? !entry.isIntersecting : false;
      },
      { threshold: 1 }
    );
    observer.observe(headerCard.value.$el);
  }

  await incrementRunCount();
  await fetchStats();

  EventsOn("log", (msg) => {
    if (msg?.message) {
      logLine(msg.message, msg.level || "info");
      const text = msg.message;
      if (text.includes("正在下载 HLAE")) {
        setStatus("status.downloading_hlae");
      } else if (text.includes("正在解压 HLAE")) {
        setStatus("status.extracting_hlae");
      } else if (text.includes("HLAE 更新完成") || text.includes("HLAE 已是最新版本")) {
        setStatus("status.hlae_ready");
      } else if (text.includes("正在下载 FFmpeg")) {
        setStatus("status.downloading_ffmpeg");
      } else if (text.includes("正在解压 FFmpeg")) {
        setStatus("status.extracting_ffmpeg");
      } else if (text.includes("FFmpeg 已准备就绪")) {
        setStatus("status.ffmpeg_ready");
      } else if (text.includes("下载进度")) {
        setStatus("status.downloading");
      } else if (text.includes("生成配置")) {
        setStatus("status.generating_cfg");
      } else if (text.includes("启动录制")) {
        setStatus("status.launching_record");
      } else if (text.includes("等待录制完成")) {
        setStatus("status.recording");
      } else if (text.includes("视频合成")) {
        setStatus("status.merging_video");
      } else if (text.includes("✓ 全部完成")) {
        setStatus("status.task_done");
      }
    }
  });

  await prepareEnvironment();
  await checkForUpdates();

  updateScrollState();
  window.addEventListener("scroll", updateScrollState, { passive: true });
});

onBeforeUnmount(() => {
  window.removeEventListener("scroll", updateScrollState);
});

async function checkForUpdates() {
  try {
    const info = await callBackend("GetUpdateInfo");
    if (!info?.available) return;
    const content = t("update.message", { current: info.current, latest: info.latest });
    dialog.info({
      title: t("update.title"),
      content: () => h("div", { style: "white-space: pre-line" }, content),
      positiveText: t("update.download"),
      negativeText: t("update.cancel"),
      onPositiveClick: () => {
        if (info.url) {
          BrowserOpenURL(info.url);
        }
      },
    });
  } catch (_) {
    return;
  }
}

</script>

<style scoped>
.status-bar {
  position: sticky;
  top: 0;
  z-index: 10;
  padding: 4px 0;
  background: #0f1115;
}

.status-bar.fixed {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  padding: 8px 16px;
  background: rgba(20, 20, 20, 0.85);
  backdrop-filter: blur(6px);
}

.status-spacer {
  height: 52px;
}

.author-row {
  flex-wrap: wrap;
}

.header-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.header-title {
  margin: 0;
}

.header-icons {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.header-bottom {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  justify-content: flex-end;
}

.stats-text {
  margin-left: 6px;
}

.icon-link {
  display: inline-flex;
  align-items: center;
  color: #cfd6dd;
  background: transparent;
  border: none;
  padding: 0;
  cursor: pointer;
}

.icon-link:hover {
  color: #ffffff;
}

.icon {
  width: 18px;
  height: 18px;
}

.version-text {
  margin-left: 0;
}

.lang-select {
  margin-left: 6px;
  width: 120px;
}

.lang-select :deep(.n-base-selection) {
  width: 120px;
}

.scroll-fab {
  position: fixed;
  right: 16px;
  bottom: 16px;
  z-index: 120;
}

.video-preview {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.video-preview video {
  width: 100%;
  max-height: 320px;
  background: #0a0a0a;
  border-radius: 6px;
}

.video-path {
  word-break: break-all;
}

.needs-attention :deep(.n-input__border) {
  border-color: #e04f5f !important;
  box-shadow: 0 0 0 1px rgba(224, 79, 95, 0.35) inset;
}

.needs-attention :deep(.n-button) {
  border-color: #e04f5f !important;
  color: #e04f5f !important;
}

.needs-attention :deep(.n-button:hover) {
  border-color: #ff6b7a !important;
  color: #ff6b7a !important;
}
</style>
