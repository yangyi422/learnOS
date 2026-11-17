<template>
  <PageShell class="media-page" variant="standard">
    <PageHeader :title="title" :description="subtitle">
      <template #breadcrumbs><nav class="library-breadcrumb">
        <RouterLink to="/explore/library">我的资料库</RouterLink><span>/</span>
        <RouterLink to="/explore/library/media">媒体中心</RouterLink><span>/</span>
        <RouterLink to="/explore/library/media/series">电视剧</RouterLink><span>/</span><span>{{ series?.Title ?? "剧集" }}</span>
      </nav></template>
      <template #actions><RouterLink class="library-entry-link" to="/explore/library/media/series">返回电视剧</RouterLink></template>
    </PageHeader>

    <el-skeleton v-if="loading" :rows="6" animated />
    <div v-else-if="error" class="media-series__error">
      <p>{{ error }}</p>
      <el-button @click="reload">重试</el-button>
    </div>

    <template v-else-if="series">
      <section class="media-series">
        <div class="media-series__poster">
          <img v-if="series.cover_id" :src="mediaCoverURL(series.cover_id)" :alt="series.Title" loading="lazy" />
          <span v-else class="media-series__poster-empty">无海报</span>
        </div>
        <div class="media-series__body">
          <p class="media-series__overview">{{ series.Description || "这部剧集还没有简介。" }}</p>
          <p v-if="series.Genres?.length" class="media-series__genres">{{ series.Genres.join(" / ") }}</p>
        </div>
      </section>

      <section v-if="seasons.length" class="media-season-picker" aria-label="季选择">
        <button
          v-for="season in seasons"
          :key="season.season_number"
          type="button"
          class="media-season-chip"
          :class="{ 'is-active': season.season_number === selectedSeason }"
          :aria-pressed="season.season_number === selectedSeason"
          @click="selectSeason(season.season_number)"
        >
          <span>{{ seasonLabel(season.season_number) }}</span>
          <em>{{ season.episodes }} 集</em>
        </button>
      </section>

      <section class="media-episodes" aria-live="polite">
        <p v-if="episodesLoading" class="media-muted" role="status">正在读取分集…</p>
        <el-empty v-else-if="!episodes.length" description="这一季还没有同步分集" />
        <ul v-else class="media-episode-list">
          <li v-for="episode in episodes" :key="episode.id" class="media-episode">
            <div class="media-episode__thumb">
              <img v-if="episode.cover_id" :src="mediaCoverURL(episode.cover_id)" :alt="episode.Title" loading="lazy" />
              <span v-else class="media-episode__thumb-empty">{{ formatEpisodeCode(episode.season_number ?? 0, episode.episode_number ?? 0) }}</span>
            </div>
            <div class="media-episode__body">
              <p class="media-episode__code">
                <span>{{ formatEpisodeCode(episode.season_number ?? 0, episode.episode_number ?? 0) }}</span>
                <span v-if="formatRuntime(episode.runtime_seconds)" class="media-episode__runtime">{{ formatRuntime(episode.runtime_seconds) }}</span>
                <span v-if="episode.premiere_date" class="media-episode__runtime">{{ episode.premiere_date }}</span>
              </p>
              <h3>{{ episode.Title }}</h3>
              <p v-if="episode.Description" class="media-episode__overview">{{ episode.Description }}</p>
            </div>
            <div class="media-episode__actions">
              <el-button v-if="desktop" type="primary" plain @click="play(episode)">播放</el-button>
              <span v-else class="media-muted">桌面端可播放</span>
            </div>
          </li>
        </ul>
        <p v-if="playState" class="media-episode__state">{{ playState }}</p>
        <div v-if="playState" class="media-play__actions">
          <el-button text @click="copyPlayLink">复制播放链接</el-button>
          <span class="media-play__log">失败原因看 <code>%LOCALAPPDATA%\LifeOS\PlayerBridge\logs\bridge.log</code>，或在 Bridge 目录运行 <code>player-bridge.exe -resolve &lt;剧集 id&gt;</code>。</span>
        </div>
      </section>
    </template>
  </PageShell>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import { ElMessage } from "element-plus";
import PageHeader from "../components/PageHeader.vue";
import PageShell from "../components/PageShell.vue";
import { loadSeriesDetail, loadSeriesEpisodes, MediaNotFoundError, mediaCoverURL, type MediaItem, type MediaSeason } from "@/api/media";
import { formatEpisodeCode, formatRuntime, seasonLabel } from "@/utils/mediaHub";
import "@/library.css";

const route = useRoute();
const externalId = computed(() => String(route.params.externalID ?? ""));
const loading = ref(true);
const episodesLoading = ref(false);
const error = ref("");
const series = ref<MediaItem | null>(null);
const seasons = ref<MediaSeason[]>([]);
const episodes = ref<MediaItem[]>([]);
const selectedSeason = ref<number | null>(null);
const desktop = ref(true);
const playState = ref("");
let playItem: MediaItem | null = null;

const title = computed(() => series.value?.Title ?? "剧集");
const facts = computed(() => {
  const value = series.value;
  if (!value) return [];
  const total = seasons.value.reduce((sum, season) => sum + season.episodes, 0);
  return [
    value.Year ? String(value.Year) : "",
    "电视剧",
    value.Rating ? value.Rating.toFixed(1) + " 分" : "",
    seasons.value.length ? `${seasons.value.length} 季 · ${total} 集` : "尚未同步分集",
    value.Source,
  ].filter(Boolean);
});
const subtitle = computed(() => facts.value.join(" · ") || "从 Jellyfin 同步的剧集信息");

async function loadEpisodes() {
  if (selectedSeason.value === null) {
    episodes.value = [];
    return;
  }
  episodesLoading.value = true;
  try {
    episodes.value = await loadSeriesEpisodes(externalId.value, selectedSeason.value);
  } catch (e) {
    episodes.value = [];
    error.value = (e as Error).message;
  } finally {
    episodesLoading.value = false;
  }
}

async function reload() {
  loading.value = true;
  error.value = "";
  try {
    const detail = await loadSeriesDetail(externalId.value);
    series.value = detail.series;
    seasons.value = detail.seasons;
    selectedSeason.value = detail.seasons.length ? detail.seasons[0]!.season_number : null;
    await loadEpisodes();
  } catch (e) {
    series.value = null;
    error.value = e instanceof MediaNotFoundError ? "这部剧集不在资料库中，可能已从 Jellyfin 移除。" : (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function selectSeason(season: number) {
  if (season === selectedSeason.value) return;
  selectedSeason.value = season;
  playState.value = "";
  void loadEpisodes();
}

function play(episode: MediaItem) {
  if (!episode.external_id) {
    ElMessage.warning("这一集缺少 Jellyfin 标识，无法播放");
    return;
  }
  playItem = episode;
  // 协议没有回调：只能表示“已提交启动请求”，是否真的播放以 PotPlayer 是否打开为准。
  playState.value = `已向 Windows 提交《${episode.Title}》的启动请求（浏览器可能弹出确认）；播放是否成功请以 PotPlayer 是否打开为准。`;
  window.location.href = `lifeos-player://play?itemId=${encodeURIComponent(episode.external_id)}`;
}

async function copyPlayLink() {
  if (!playItem?.external_id) return;
  try {
    await navigator.clipboard.writeText(`lifeos-player://play?itemId=${playItem.external_id}`);
    ElMessage.success("播放链接已复制");
  } catch {
    ElMessage.warning("复制失败，请手动复制");
  }
}

function syncViewport() {
  desktop.value = window.matchMedia("(pointer: fine)").matches && window.innerWidth > 820;
}

onMounted(() => {
  syncViewport();
  window.addEventListener("resize", syncViewport);
  void reload();
});
onBeforeUnmount(() => window.removeEventListener("resize", syncViewport));
</script>