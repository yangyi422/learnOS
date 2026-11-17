<template>
  <PageShell class="media-page" variant="standard">
    <PageHeader :title="label" :description="subtitle">
      <template #breadcrumbs><nav class="library-breadcrumb">
        <RouterLink to="/explore/library">我的资料库</RouterLink><span>/</span>
        <RouterLink to="/explore/library/media">媒体中心</RouterLink><span>/</span><span>{{ label }}</span>
      </nav></template>
      <template #actions><RouterLink class="library-entry-link" to="/explore/library/media">返回媒体中心</RouterLink></template>
    </PageHeader>

    <div v-if="reserved" class="media-reserved">
      <svg class="media-reserved__icon" viewBox="0 0 24 24" aria-hidden="true">
        <path d="M7.5 8h9a3.5 3.5 0 0 1 3.5 3.5v3a3.5 3.5 0 0 1-3.5 3.5c-1 0-1.9-.43-2.53-1.16L13.1 15.9a1.5 1.5 0 0 0-2.2 0l-.87.94A3.4 3.4 0 0 1 7.5 18 3.5 3.5 0 0 1 4 14.5v-3A3.5 3.5 0 0 1 7.5 8Z" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round" />
        <path d="M8 11.2v2.6M6.7 12.5h2.6M15.2 11.9h.01M16.9 13.4h.01" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
      </svg>
      <h3>游戏库即将接入</h3>
      <p>未来支持 Steam 游戏收藏、封面与游玩时间</p>
      <p class="media-reserved__note">本阶段只预留入口，暂不接入 Steam 数据</p>
    </div>

    <template v-else>
      <form class="media-toolbar" @submit.prevent="load(1)">
        <el-input v-model="query" aria-label="媒体标题" placeholder="搜索标题" />
        <el-select v-model="source" aria-label="媒体来源" @change="load(1)">
          <el-option label="全部来源" value="all" />
          <el-option label="Calibre" value="calibre" />
          <el-option label="Jellyfin" value="jellyfin" />
        </el-select>
        <el-select v-model="sort" aria-label="排序">
          <el-option v-for="option in mediaSortOptions" :key="option.value" :label="option.label" :value="option.value" />
        </el-select>
        <el-checkbox v-if="r18Relevant" v-model="showR18" @change="load(1)">显示 R18 与待分类的 Jellyfin 内容</el-checkbox>
        <div class="media-view-toggle" role="group" aria-label="视图">
          <button type="button" :aria-pressed="view === 'gallery'" @click="view = 'gallery'">画廊</button>
          <button type="button" :aria-pressed="view === 'list'" @click="view = 'list'">列表</button>
        </div>
        <el-button native-type="submit">搜索</el-button>
      </form>

      <section class="media-results" aria-live="polite">
        <el-alert v-if="error" type="error" :title="error" :closable="false" />
        <p v-if="loading" class="media-muted" role="status">正在读取媒体…</p>
        <el-empty v-else-if="!error && !items.length" description="暂无媒体" />
        <div v-else-if="view === 'gallery'" class="media-grid">
          <button v-for="item in sortedItems" :key="item.id" type="button" class="media-card" @click="selected = item">
            <img v-if="item.cover_id" class="media-cover" :src="mediaCoverURL(item.cover_id)" alt="" loading="lazy" />
            <span v-else class="media-cover media-cover--empty">无封面</span>
            <strong>{{ item.Title }}</strong>
            <span class="media-card__meta">{{ mediaSubtitle(item) }}</span>
            <span v-if="item.Description" class="media-card__note">{{ item.Description }}</span>
          </button>
        </div>
        <ul v-else class="media-list">
          <li v-for="item in sortedItems" :key="item.id">
            <button type="button" class="media-list__row" @click="selected = item">
              <img v-if="item.cover_id" class="media-list__cover" :src="mediaCoverURL(item.cover_id)" alt="" loading="lazy" />
              <span v-else class="media-list__cover media-cover--empty">无封面</span>
              <span class="media-list__text"><strong>{{ item.Title }}</strong><span class="media-card__meta">{{ mediaSubtitle(item) }}</span></span>
            </button>
          </li>
        </ul>
        <el-pagination
          v-if="mediaTotal > 24"
          :current-page="page"
          :page-size="24"
          :total="mediaTotal"
          layout="prev, pager, next"
          @current-change="load"
        />
      </section>
    </template>

    <el-dialog :model-value="selected !== null" :title="selected?.Title" width="min(560px, 90vw)" @close="selected = null">
      <template v-if="selected">
        <img v-if="selected.cover_id" class="media-cover media-cover--detail" :src="mediaCoverURL(selected.cover_id)" alt="" />
        <p>{{ mediaSubtitle(selected) }}</p>
        <p v-if="selected.Authors?.length">{{ selected.Authors.join("、") }}</p>
        <p v-if="selected.Description" class="media-description">{{ selected.Description }}</p>
        <p v-if="selected.Tags?.length">{{ selected.Tags.join("、") }}</p>
        <p v-if="selected.favorite">已收藏</p>
        <p v-if="selected.notes" class="media-description">{{ selected.notes }}</p>
      </template>
    </el-dialog>
  </PageShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import PageHeader from "../components/PageHeader.vue";
import PageShell from "../components/PageShell.vue";
import { listMedia, mediaCoverURL, type MediaItem } from "@/api/media";
import {
  isReservedMediaKind,
  mediaCategoryByRoute,
  mediaKindName,
  mediaSortOptions,
  mediaSubtitle,
  sortMediaItems,
  type MediaCategoryKind,
  type MediaSort,
} from "@/utils/mediaHub";
import "@/library.css";

const route = useRoute();
const kind = computed<MediaCategoryKind | undefined>(() => mediaCategoryByRoute[String(route.params.kind ?? "")]);
const label = computed(() => (kind.value ? mediaKindName(kind.value) : "媒体分类"));
const reserved = computed(() => (kind.value ? isReservedMediaKind(kind.value) : false));
const r18Relevant = computed(() => kind.value === "movie" || kind.value === "series");
const subtitle = computed(() => {
  if (!kind.value) return "该分类不存在";
  if (reserved.value) return "Steam · Windows · 即将接入";
  const source = kind.value === "book" ? "Calibre · OPDS" : "Jellyfin";
  return `${source} · 共 ${mediaTotal.value} 件`;
});

const items = ref<MediaItem[]>([]);
const selected = ref<MediaItem | null>(null);
const query = ref("");
const source = ref("all");
const sort = ref<MediaSort>("recent");
const showR18 = ref(false);
const view = ref<"gallery" | "list">("gallery");
const page = ref(1);
const mediaTotal = ref(0);
const error = ref("");
const loading = ref(false);
let request = 0;

const sortedItems = computed(() => sortMediaItems(items.value, sort.value));

async function load(next: number) {
  if (!kind.value || reserved.value) {
    request += 1;
    items.value = [];
    mediaTotal.value = 0;
    loading.value = false;
    return;
  }
  const current = ++request;
  error.value = "";
  loading.value = true;
  items.value = [];
  try {
    const result = await listMedia({
      q: query.value,
      type: kind.value,
      source: source.value === "all" ? "" : source.value,
      page: next,
      showR18: showR18.value,
    });
    if (current !== request) return;
    items.value = result.items;
    mediaTotal.value = result.total;
    page.value = next;
  } catch {
    if (current === request) error.value = "媒体读取失败，请重试。";
  } finally {
    if (current === request) loading.value = false;
  }
}

watch(() => route.params.kind, () => {
  selected.value = null;
  query.value = "";
  source.value = "all";
  sort.value = "recent";
  void load(1);
});
onMounted(() => void load(1));
</script>