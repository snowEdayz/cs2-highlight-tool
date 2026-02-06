import { createApp } from "vue";
import { createI18n } from "vue-i18n";
import naive from "naive-ui";
import App from "./App.vue";
import en from "./locales/en.json";
import zh from "./locales/zh.json";

const savedLocale = localStorage.getItem("locale");
const browserLocale = navigator.language?.toLowerCase().startsWith("zh") ? "zh" : "en";
const defaultLocale = savedLocale || browserLocale || "zh";

const i18n = createI18n({
  legacy: false,
  locale: defaultLocale,
  fallbackLocale: "zh",
  messages: { en, zh },
});

const app = createApp(App);

app.use(i18n);
app.use(naive);

app.mount("#app");