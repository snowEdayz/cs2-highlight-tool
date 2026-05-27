<template>
  <section v-if="sortedAds.length > 0" class="ads-wrap">
    <n-carousel
      class="ads-carousel"
      :autoplay="sortedAds.length > 1"
      :interval="5000"
      :show-dots="true"
      :draggable="true"
      :touchable="true"
      :show-arrow="false"
      :loop="sortedAds.length > 1"
    >
      <n-carousel-item v-for="ad in sortedAds" :key="ad.id">
        <button type="button" class="sponsor-card" @click="onClick(ad.click_url)">
          <img
            class="sponsor-card__image"
            :src="ad.image_url"
            :alt="ad.image_alt || ad.title || 'sponsored card image'"
          />
          <div class="sponsor-card__content">
            <div v-if="ad.sponsor.trim().length > 0" class="sponsor-card__sponsor">{{ ad.sponsor }}</div>
            <div class="sponsor-card__title">{{ ad.title }}</div>
            <div class="sponsor-card__rich" v-html="ad.rich_html" />
          </div>
        </button>
      </n-carousel-item>
    </n-carousel>
  </section>
</template>

<script setup lang="ts">
import { toRef } from "vue";
import { useMainTopBannerAds } from "@/features/ads/composables/useMainTopBannerAds";
import type { StartupAd } from "@/shared/types";

const props = defineProps<{
  ads: StartupAd[];
}>();

const { sortedAds, openAd } = useMainTopBannerAds(toRef(props, "ads"));

async function onClick(clickURL: string) {
  try {
    await openAd(clickURL);
  } catch {
    // keep ad click silent on failure
  }
}
</script>

<style scoped>
.ads-wrap {
  margin-bottom: 6px;
}

.ads-carousel {
  border-radius: 10px;
  overflow: hidden;
}

.sponsor-card {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 82px;
  border: 1px solid #303732;
  border-radius: 10px;
  text-align: left;
  background: #181b19;
  color: inherit;
  padding: 0;
  cursor: pointer;
}

.sponsor-card__image {
  width: 80%;
  height: 60px;
  border-radius: 8px;
  margin: 5px;
  background: #141816;
  display: block;
  object-fit: contain;
}

.sponsor-card__content {
  flex: 1;
  min-width: 0;
  padding: 8px 10px 8px 0;
}

.sponsor-card__sponsor {
  font-size: 11px;
  color: #aeb8b1;
  margin-bottom: 2px;
}

.sponsor-card__title {
  font-size: 13px;
  font-weight: 600;
  color: #edf1ee;
  margin-bottom: 4px;
  line-height: 1.25;
}

.sponsor-card__rich {
  font-size: 11px;
  line-height: 1.35;
  color: #c9d3cb;
}

.sponsor-card__rich :deep(p),
.sponsor-card__rich :deep(ul),
.sponsor-card__rich :deep(ol),
.sponsor-card__rich :deep(pre) {
  margin: 0;
}

.sponsor-card__rich :deep(a) {
  color: #85d3a7;
  text-decoration: underline;
}

.sponsor-card__rich :deep(strong),
.sponsor-card__rich :deep(b) {
  font-weight: 600;
}
</style>
