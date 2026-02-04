<template>
  <n-config-provider :theme="darkTheme">
    <n-message-provider>
      <div class="app">
        <n-space vertical size="large">
          <n-card ref="headerCard">
            <n-space vertical>
              <n-h2>CS2 击杀集锦制作工具</n-h2>
              <n-space align="center" size="small" class="author-row">
                <button class="icon-link" type="button" @click="openExternal(`mailto:${author.email}`)" aria-label="邮箱">
                  <svg viewBox="0 0 24 24" class="icon" aria-hidden="true">
                    <path
                      fill="currentColor"
                      d="M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 4-8 5-8-5V6l8 5 8-5v2z"
                    />
                  </svg>
                </button>
                <button class="icon-link" type="button" @click="openExternal(author.github)" aria-label="GitHub">
                  <svg viewBox="0 0 24 24" class="icon" aria-hidden="true">
                    <path
                      fill="currentColor"
                      d="M12 2C6.48 2 2 6.58 2 12.26c0 4.5 2.87 8.31 6.84 9.66.5.09.68-.22.68-.48 0-.24-.01-.87-.01-1.7-2.78.62-3.37-1.38-3.37-1.38-.45-1.19-1.11-1.5-1.11-1.5-.9-.64.07-.63.07-.63 1 .07 1.53 1.06 1.53 1.06.89 1.57 2.34 1.12 2.91.86.09-.67.35-1.12.63-1.38-2.22-.26-4.56-1.15-4.56-5.12 0-1.13.39-2.05 1.03-2.77-.1-.26-.45-1.31.1-2.73 0 0 .84-.28 2.75 1.06.8-.23 1.66-.34 2.52-.34s1.72.12 2.52.34c1.9-1.34 2.75-1.06 2.75-1.06.55 1.42.2 2.47.1 2.73.64.72 1.03 1.64 1.03 2.77 0 3.98-2.34 4.85-4.57 5.1.36.33.68.97.68 1.96 0 1.41-.01 2.55-.01 2.9 0 .26.18.58.69.48 3.97-1.35 6.83-5.16 6.83-9.66C22 6.58 17.52 2 12 2z"
                    />
                  </svg>
                </button>
                <n-text depth="3" class="version-text">{{ appVersion }}</n-text>
              </n-space>
            </n-space>
          </n-card>

          <div :class="['status-bar', { fixed: statusFixed }]">
            <n-alert :type="statusType" :bordered="false">
              {{ statusText }}
            </n-alert>
          </div>
          <div v-if="statusFixed" class="status-spacer"></div>

          <template v-if="!isSetupComplete">
            <n-card title="环境准备">
              <n-space vertical>
                <n-text>正在检测并下载 FFmpeg和HLAE，请稍候。</n-text>
                <n-form label-placement="top" size="small">
                  <n-grid :cols="2" :x-gap="16" :y-gap="12" responsive="screen">
                    <n-form-item-gi label="CS2.exe" :class="{ 'needs-attention': needsCS2Highlight }">
                      <n-input v-model:value="form.cs2_exe" placeholder="请选择 CS2.exe 路径">
                        <template #suffix>
                          <n-button size="tiny" @click="pickCs2Exe" :disabled="isPreparingEnv" :class="{ 'needs-attention': needsCS2Highlight }">选择</n-button>
                        </template>
                      </n-input>
                    </n-form-item-gi>
                    <n-form-item-gi label="HLAE.exe">
                      <n-input v-model:value="form.hlae_exe" placeholder="HLAE.exe 路径" disabled />
                    </n-form-item-gi>
                    <n-form-item-gi label="FFmpeg 目录">
                      <n-input v-model:value="form.ffmpeg_dir" placeholder="FFmpeg 目录" disabled />
                    </n-form-item-gi>
                  </n-grid>
                </n-form>
                <n-space class="actions">
                  <n-button type="primary" @click="confirmSetup" :disabled="isPreparingEnv">完成设置</n-button>
                </n-space>
              </n-space>
            </n-card>

            <n-card title="日志">
              <n-log ref="logRef" :log="logContent" :rows="12" />
            </n-card>
          </template>

          <n-grid v-else :cols="2" :x-gap="16" :y-gap="16" responsive="screen">
            <n-gi>
              <n-space vertical size="large">
                <n-card title="配置">
                  <n-form label-placement="top" size="small">
                    <n-grid :cols="2" :x-gap="16" :y-gap="12" responsive="screen">
                      <n-form-item-gi label="输出目录">
                        <n-input v-model:value="form.output_dir" placeholder="outputs 目录">
                          <template #suffix>
                            <n-button size="tiny" @click="pickOutputDir">选择</n-button>
                          </template>
                        </n-input>
                      </n-form-item-gi>
                      <n-form-item-gi label="录制 FPS">
                        <n-input-number v-model:value="form.record_fps" :min="1" />
                      </n-form-item-gi>
                      <n-form-item-gi label="Tickrate">
                        <n-input-number v-model:value="form.tickrate" :min="1" />
                      </n-form-item-gi>
                      <n-form-item-gi label="视频预设">
                        <n-select v-model:value="form.video_preset" :options="presetOptions" />
                      </n-form-item-gi>
                      <n-form-item-gi label="转场时长 (秒)">
                        <n-input-number v-model:value="form.transition_duration" :min="0" :step="0.1" />
                      </n-form-item-gi>
                      <n-form-item-gi label="转场类型">
                        <n-select v-model:value="form.transition_type" :options="transitionOptions" />
                      </n-form-item-gi>
                    </n-grid>
                  </n-form>
                  <n-space class="actions">
                    <n-button type="primary" @click="saveConfig">保存配置</n-button>
                  </n-space>
                </n-card>

                <n-card title="生成视频预览">
                  <div v-if="lastOutputPath" class="video-preview">
                    <video v-if="lastOutputUrl" :key="lastOutputUrl" :src="lastOutputUrl" controls preload="metadata"></video>
                    <n-text v-else depth="3">预览不可用，请使用外部播放器打开</n-text>
                    <n-space align="center">
                      <n-button size="small" @click="openVideoExternal">打开外部播放器</n-button>
                    </n-space>
                    <n-text depth="3" class="video-path">{{ lastOutputPath }}</n-text>
                  </div>
                  <n-text v-else depth="3">本次启动尚未生成视频</n-text>
                </n-card>
              </n-space>
            </n-gi>

            <n-gi>
              <n-card title="Demo 解析">
                <n-space vertical>
                  <n-input v-model:value="demoPath" placeholder="请选择 demo 文件 (.dem)" :disabled="!isEnvReady">
                    <template #suffix>
                      <n-button size="tiny" @click="pickDemo" :disabled="!isEnvReady">选择 Demo</n-button>
                    </template>
                  </n-input>
                  <n-space>
                    <n-select
                      v-model:value="selectedPlayerSteamId"
                      placeholder="选择玩家"
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
                          <n-text>回合 {{ round.round }} ({{ round.kills.length }} 杀)</n-text>
                        </n-space>
                      </template>
                      <n-collapse v-model:expanded-names="expandedRounds">
                        <n-collapse-item title="查看击杀详情" :name="round.round">
                          <DeathNoticeList :kills="round.kills" />
                        </n-collapse-item>
                      </n-collapse>
                    </n-card>
                  </div>

                  <n-space class="actions">
                    <n-button type="primary" @click="runWorkflow(true)" :disabled="isParsing || !isEnvReady">制作</n-button>
                  </n-space>
                </n-space>
              </n-card>
            </n-gi>

            <n-gi :span="2">
              <n-card title="日志">
                <n-log ref="logRef" :log="logContent" :rows="12" />
              </n-card>
            </n-gi>
          </n-grid>
        </n-space>
      </div>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup>
import { computed, nextTick, onMounted, reactive, ref, watch } from "vue";
import { createDiscreteApi, darkTheme } from "naive-ui";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { BrowserOpenURL } from "../wailsjs/runtime/runtime";
import wailsConfig from "../../wails.json";
import * as AppApi from "../wailsjs/go/main/App";
import DeathNoticeList from "./components/DeathNoticeList.vue";

const demoPath = ref("");
const demoInfo = ref(null);
const selectedPlayerSteamId = ref(null);
const selectedRounds = ref(new Set());
const logs = ref([]);
const logRef = ref(null);
const author = {
  email: "hk_snow@yeah.net",
  github: "https://github.com/hkslover/cs2-highlight-tool",
};
const appVersion = `v${wailsConfig.version || "0.0.0"}`;
const statusText = ref("就绪");
const statusType = computed(() => (statusText.value === "就绪" ? "success" : "info"));
const isParsing = ref(false);
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
const { message } = createDiscreteApi(["message"], {
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

const players = computed(() => demoInfo.value?.players || []);
const playerOptions = computed(() =>
  players.value.map((p) => ({
    label: `${p.name} (${p.total_kills} kills)`,
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
    text = `成功: ${text.replace(/^✓\s*/, "")}`;
  } else if (text.startsWith("✗ ")) {
    text = `失败: ${text.replace(/^✗\s*/, "")}`;
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

function setStatus(text) {
  statusText.value = text || "就绪";
}

function openExternal(url) {
  if (!url) return;
  BrowserOpenURL(url);
}

function formatError(err) {
  if (!err) return "未知错误";
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
  return Promise.reject(new Error(`Wails API 未加载: ${method}`));
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
    message.warning("请先完成环境检查并设置 CS2.exe");
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
      message.success("配置已保存");
      logLine("配置已保存", "success");
      setStatus("配置已保存");
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
    logLine(`视频预览不可用: ${formatError(err)}`, "warning");
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
    logLine("请先完成环境检查并设置 CS2.exe", "warning");
    return;
  }
  if (!demoPath.value) {
    logLine("请先选择 demo 文件", "warning");
    return;
  }
  try {
    isParsing.value = true;
    setStatus("正在解析 Demo...");
    const info = await callBackend("ParseDemo", demoPath.value);
    demoInfo.value = info;
    selectedPlayerSteamId.value = info.players?.[0]?.steam_id ?? null;
    selectedRounds.value = new Set();
    expandedRounds.value = [];
    logLine("Demo 解析完成", "success");
    setStatus("解析完成");
  } catch (err) {
    const msg = formatError(err);
    message.error(msg);
    logLine(msg, "error");
    setStatus("解析失败");
  } finally {
    isParsing.value = false;
    setTimeout(() => setStatus("就绪"), 800);
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

const expandAllLabel = computed(() => (allExpanded.value ? "全部关闭" : "全部展开"));

function toggleExpandAll() {
  if (allExpanded.value) {
    expandedRounds.value = [];
  } else {
    expandedRounds.value = rounds.value.map((r) => r.round);
  }
}

async function runWorkflow(autoMode) {
  if (!isEnvReady.value) {
    message.warning("请先完成环境检查并设置 CS2.exe");
    return;
  }
  await saveConfig(true);
  const player = players.value.find((p) => p.steam_id === selectedPlayerSteamId.value);
  if (!player) {
    logLine("请先选择玩家", "warning");
    return;
  }
  if (!demoPath.value) {
    logLine("请先选择 demo 文件", "warning");
    return;
  }
  const selected = Array.from(selectedRounds.value);
  if (!selected.length) {
    logLine("请选择回合", "warning");
    return;
  }

  try {
    setStatus(autoMode ? "正在生成 CFG 并启动录制..." : "正在生成 CFG...");
    const res = await callBackend("RunWorkflow", {
      demo_path: demoPath.value,
      player_steam_id: player.steam_id,
      selected_rounds: selected,
      auto_mode: autoMode,
      debug_mode: false,
    });
    if (res?.cfg_path) logLine(`CFG 已生成: ${res.cfg_path}`, "success");
    if (res?.output_path) {
      logLine(`输出视频: ${res.output_path}`, "success");
      lastOutputPath.value = res.output_path;
      await updatePreviewUrl(res.output_path);
    }
    setStatus("任务完成");
  } catch (err) {
    const msg = formatError(err);
    message.error(msg);
    logLine(msg, "error");
    setStatus("任务失败");
  } finally {
    setTimeout(() => setStatus("就绪"), 1200);
  }
}

async function checkEnvironment() {
  try {
    if (isCheckingEnv.value) return;
    isCheckingEnv.value = true;
    isEnvReady.value = false;
    needsCS2Path.value = false;
    setStatus("正在检查环境...");
    await callBackend("CheckEnvironment");
    await loadConfig();
    logLine("环境检查完成", "success");
    message.success("环境检查完成");
    isEnvReady.value = true;
    isSetupComplete.value = true;
  } catch (err) {
    const msg = formatError(err);
    message.error(msg);
    logLine(msg, "error");
    if (msg.includes("CS2 未找到")) {
      needsCS2Path.value = true;
      setStatus("请先设置 CS2.exe");
      message.warning("请先设置 CS2.exe");
      isSetupComplete.value = false;
    }
  } finally {
    isCheckingEnv.value = false;
    if (!needsCS2Path.value) {
      setStatus("就绪");
    }
  }
}

async function prepareEnvironment() {
  try {
    if (isPreparingEnv.value) return;
    isPreparingEnv.value = true;
    setStatus("正在准备环境...");
    const cfg = await callBackend("PrepareEnvironment", false);
    fillConfig(cfg);
    configReady = true;
    if (form.cs2_exe) {
      await checkEnvironment();
    } else {
      needsCS2Path.value = true;
      setStatus("请先设置 CS2.exe");
    }
  } catch (err) {
    const msg = formatError(err);
    message.error(msg);
    logLine(msg, "error");
    setStatus("环境准备失败");
  } finally {
    isPreparingEnv.value = false;
  }
}

async function confirmSetup() {
  if (!form.cs2_exe) {
    message.warning("请先设置 CS2.exe");
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

  EventsOn("log", (msg) => {
    if (msg?.message) {
      logLine(msg.message, msg.level || "info");
      const text = msg.message;
      if (text.includes("正在下载 HLAE")) {
        setStatus("正在下载 HLAE...");
      } else if (text.includes("正在解压 HLAE")) {
        setStatus("正在解压 HLAE...");
      } else if (text.includes("HLAE 更新完成") || text.includes("HLAE 已是最新版本")) {
        setStatus("HLAE 已准备就绪");
      } else if (text.includes("正在下载 FFmpeg")) {
        setStatus("正在下载 FFmpeg...");
      } else if (text.includes("正在解压 FFmpeg")) {
        setStatus("正在解压 FFmpeg...");
      } else if (text.includes("FFmpeg 已准备就绪")) {
        setStatus("FFmpeg 已准备就绪");
      } else if (text.includes("下载进度")) {
        setStatus("正在下载...");
      } else if (text.includes("生成配置")) {
        setStatus("正在生成 CFG...");
      } else if (text.includes("启动录制")) {
        setStatus("正在启动录制...");
      } else if (text.includes("等待录制完成")) {
        setStatus("正在录制...");
      } else if (text.includes("视频合成")) {
        setStatus("正在合成视频...");
      } else if (text.includes("✓ 全部完成")) {
        setStatus("任务完成");
      }
    }
  });

  await prepareEnvironment();
});

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
  margin-left: 4px;
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
