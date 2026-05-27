import { computed, type Ref } from "vue";
import type { StartupAd } from "@/shared/types";

export function useMainTopBannerAds(ads: Ref<StartupAd[]>) {
  const sortedAds = computed(() =>
    (ads.value || [])
      .filter((ad) => ad.placement === "main_steps_top_banner")
      .slice(),
  );

  async function callBackend(method: string, ...args: unknown[]) {
    const api = window.go?.app?.App as Record<string, (...a: unknown[]) => Promise<unknown>> | undefined;
    const fn = api?.[method];
    if (!fn) throw new Error(`Wails API not loaded: ${method}`);
    return fn(...args);
  }

  async function openAd(clickURL: string) {
    await callBackend("OpenExternalURL", clickURL);
  }

  return {
    sortedAds,
    openAd,
  };
}
