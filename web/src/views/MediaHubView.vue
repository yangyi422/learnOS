<template>
  <PageShell class="media-page" variant="standard">
    <PageHeader title="媒体中心" description="统一管理电子书、电影、电视剧与后续游戏内容">
      <template #breadcrumbs><nav class="library-breadcrumb">
        <RouterLink to="/explore/library">我的资料库</RouterLink><span>/</span><span>媒体中心</span>
      </nav></template>
    </PageHeader>

    <div class="media-entry-grid">
      <RouterLink
        v-for="entry in entries"
        :key="entry.kind"
        :to="entry.route"
        class="media-entry-card"
        :class="{ 'is-reserved': entry.reserved }"
      >
        <div class="media-entry-preview" :class="{ 'is-empty': !entry.covers.length }">
          <div v-if="entry.covers.length" class="media-entry-mosaic">
            <img
              v-for="(cover, index) in entry.covers"
              :key="cover + index"
              :src="mediaCoverURL(cover)"
              alt=""
              loading="lazy"
              decoding="async"
            />
          </div>
          <span v-else class="media-entry-placeholder">
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path :d="entry.icon" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round" stroke-linecap="round" />
            </svg>
            <em>{{ entry.reserved ? "待接入" : "暂无封面" }}</em>
          </span>
        </div>
        <div class="media-entry-body">
          <div class="media-entry-title">
            <h2>{{ entry.label }}</h2>
            <span class="media-entry-status" :class="{ 'is-reserved': entry.reserved }">{{ entry.status }}</span>
          </div>
          <p class="media-entry-source">{{ entry.source }}</p>
          <p v-if="!entry.reserved" class="media-entry-count">{{ entry.count }}</p>
        </div>
      </RouterLink>
    </div>
  </PageShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import PageHeader from "../components/PageHeader.vue";
import PageShell from "../components/PageShell.vue";
import { loadMediaOverview, mediaCoverURL, type MediaOverview } from "@/api/media";
import {
  mediaCategoryRoutes,
  mediaCountLabel,
  mediaKindBadges,
  mediaSources,
  mediaStatusLabel,
  type MediaCategoryKind,
} from "@/utils/mediaHub";
import "@/library.css";

const ICONS: Record<MediaCategoryKind, string> = {
  book: "M6 5.5h4.5v13H6a1.5 1.5 0 0 1-1.5-1.5v-10A1.5 1.5 0 0 1 6 5.5Zm12 0h-4.5v13H18a1.5 1.5 0 0 0 1.5-1.5v-10A1.5 1.5 0 0 0 18 5.5Z",
  movie: "M4 7.5h16v9H4v-9Zm4 0 1.6 9m4.4-9 1.6 9",
  series: "M4 8.5h16v9H4v-9Zm5-3h6v3",
  game: "M7.5 8h9a3.5 3.5 0 0 1 3.5 3.5v3a3.5 3.5 0 0 1-3.5 3.5c-1 0-1.9-.43-2.53-1.16L13.1 15.9a1.5 1.5 0 0 0-2.2 0l-.87.94A3.4 3.4 0 0 1 7.5 18 3.5 3.5 0 0 1 4 14.5v-3A3.5 3.5 0 0 1 7.5 8Zm.5 3.2v2.6M6.7 12.5h2.6M15.2 11.9h.01M16.9 13.4h.01",
};

const overview = ref<MediaOverview | null>(null);
const entries = computed(() =>
  mediaKindBadges.map(badge => {
    const kind = badge.value as MediaCategoryKind;
    const reserved = kind === "game";
    const items = reserved ? [] : overview.value?.previews[kind as "book" | "movie" | "series"] ?? [];
    const covers = items.map(item => item.cover_id).filter((id): id is string => Boolean(id)).slice(0, 6);
    const count = reserved ? undefined : overview.value?.counts[kind as "book" | "movie" | "series"];
    return {
      kind,
      label: badge.label,
      route: mediaCategoryRoutes[kind],
      icon: ICONS[kind],
      source: mediaSources[kind],
      status: mediaStatusLabel(kind),
      reserved,
      covers,
      count: `${mediaCountLabel(count)} 件`,
    };
  }),
);

onMounted(async () => {
  try {
    overview.value = await loadMediaOverview();
  } catch {
    overview.value = { total: 0, counts: {}, recent: [], previews: {}, partial: true };
  }
});
</script>