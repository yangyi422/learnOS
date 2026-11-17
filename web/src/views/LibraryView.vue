<template>
  <PageShell class="library-page" variant="standard">
    <PageHeader
      title="我的资料库"
      description="整理常用资料、个人专题与数字媒体收藏"
    >
      <template #breadcrumbs><nav class="library-breadcrumb">
        <RouterLink to="/explore">探索</RouterLink><span>/</span><span>我的资料库</span>
      </nav></template>
      <template #actions><el-button type="primary" @click="creating = true">新建专题</el-button></template>
    </PageHeader>

    <section class="library-entries" aria-label="内容入口">
      <article class="library-entry-card">
        <header class="library-entry-head">
          <h2>工具</h2>
          <span class="library-entry-badge">{{ tools.length }} 个</span>
        </header>
        <p class="library-entry-desc">规划与手账类小工具，进度随账号保存，Web 与桌面端共用。</p>
        <div class="library-entry-body">
          <a
            class="library-entry-item"
            :href="toolHref"
            :target="toolTarget"
            :rel="toolTarget ? 'noopener' : undefined"
          >
            <strong>万缕千丝 · 挖角手帐</strong>
            <span>四线挖角与培养规划：主力 / 板凳、名声与交涉条件、职业路线与备注。</span>
          </a>
        </div>
        <div class="library-entry-actions">
          <a
            class="library-entry-link is-primary"
            :href="toolHref"
            :target="toolTarget"
            :rel="toolTarget ? 'noopener' : undefined"
          >打开手帐</a>
          <span class="library-muted">Web 与桌面端共用</span>
        </div>
      </article>

      <article class="library-entry-card">
        <header class="library-entry-head">
          <h2>媒体中心</h2>
          <span class="library-entry-badge">{{ mediaBadge }}</span>
        </header>
        <p class="library-entry-desc">统一管理电子书、电影、电视剧与后续游戏收藏。</p>
        <div class="library-entry-body">
          <template v-if="media.total > 0">
            <div class="library-media-stats">
              <span><strong>{{ mediaCountLabel(media.counts.book) }}</strong>电子书</span>
              <span><strong>{{ mediaCountLabel(media.counts.movie) }}</strong>电影</span>
              <span><strong>{{ mediaCountLabel(media.counts.series) }}</strong>电视剧</span>
              <span class="is-reserved"><strong>—</strong>游戏 · 即将接入</span>
            </div>
            <div v-if="media.recent.length" class="library-media-covers">
              <img
                v-for="item in media.recent"
                :key="item.id"
                :src="coverURL(item.id, item.cover_id)"
                :alt="item.Title"
                loading="lazy"
                decoding="async"
              />
            </div>
            <p v-if="recentNames" class="library-media-recent">最近新增：{{ recentNames }}</p>
          </template>
          <div v-else class="library-media-empty">
            <svg class="library-media-empty__icon" viewBox="0 0 24 24" aria-hidden="true">
              <path d="M4 6.5A2.5 2.5 0 0 1 6.5 4h4l1.6 2H17.5A2.5 2.5 0 0 1 20 8.5v8A2.5 2.5 0 0 1 17.5 19h-11A2.5 2.5 0 0 1 4 16.5v-10Z" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round" />
              <path d="M8 13.5h8" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
            </svg>
            <h3>媒体中心尚未同步内容</h3>
            <p>后续将在这里查看书籍、电影、电视剧与游戏收藏。</p>
          </div>
        </div>
        <div class="library-entry-actions">
          <RouterLink class="library-entry-link is-primary" to="/explore/library/media">进入媒体中心</RouterLink>
          <span class="library-muted">Calibre / Jellyfin / 后续 Steam</span>
        </div>
      </article>
    </section>

    <section id="library-collections" class="library-collections-section" aria-labelledby="library-collections-title">
      <header class="library-collections-head">
        <div>
          <h2 id="library-collections-title">专题</h2>
          <p>每个专题是一份长期维护的个人资料库。</p>
        </div>
        <span class="library-collections-count">{{ collections.length }} 个</span>
      </header>
      <el-alert v-if="error" :title="error" type="error" :closable="false">
        <el-button text @click="load">重试</el-button>
      </el-alert>
      <p v-if="loading" role="status">正在读取专题…</p>
      <template v-else>
        <div v-if="collections.length" class="library-home-filter">
          <el-input v-model="keyword" aria-label="查找专题" placeholder="查找专题名称或说明" clearable />
          <el-select v-model="category" clearable placeholder="全部分类" aria-label="专题分类">
            <el-option v-for="c in categories" :key="c" :label="c" :value="c" />
          </el-select>
        </div>
        <div v-if="!collections.length && !error" class="library-empty">
          <svg class="library-empty__icon" viewBox="0 0 24 24" aria-hidden="true">
            <path d="M5 5.5A1.5 1.5 0 0 1 6.5 4H12v15H6.5A1.5 1.5 0 0 1 5 17.5v-12Zm14 0A1.5 1.5 0 0 0 17.5 4H12v15h5.5a1.5 1.5 0 0 0 1.5-1.5v-12Z" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round" />
          </svg>
          <div>
            <h3>先为一个常用主题建立自己的资料库</h3>
            <p>创建专题后，页面、小节与自己的结论都会归档在这里。</p>
          </div>
          <el-button type="primary" @click="creating = true">创建第一份专题</el-button>
        </div>
        <p v-else-if="!filtered.length" class="library-muted">没有符合条件的专题。</p>
        <div class="library-collections">
          <RouterLink
            v-for="c in filtered"
            :key="c.id"
            :to="`/explore/library/${c.id}`"
            class="library-collection-card"
          >
            <img
              v-if="safeCover(c.cover_url)"
              :src="libraryCardImageURL(c.cover_url)"
              alt=""
              loading="lazy"
              decoding="async"
              referrerpolicy="no-referrer"
            />
            <span class="library-muted">{{ c.category || "未分类" }} · {{ templateName(c.template) }}</span>
            <h2>{{ c.name }}</h2>
            <p>{{ c.description || "从一篇页面开始，逐步整理自己的资料。" }}</p>
            <small>{{ c.page_count || 0 }} 篇页面 · {{ date(c.updated_at) }} 更新</small>
          </RouterLink>
        </div>
      </template>
    </section>

    <CollectionDialog :open="creating" @close="creating = false" @saved="load" />
  </PageShell>
</template>

<script setup lang="ts">
import PageShell from "@/components/PageShell.vue";
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import PageHeader from "@/components/PageHeader.vue";
import CollectionDialog from "@/components/library/CollectionDialog.vue";
import { listCollections, type Collection } from "@/api/library";
import { loadMediaOverview, mediaCoverURL, type MediaOverview } from "@/api/media";
import { mediaCountLabel, recentMediaNames } from "@/utils/mediaHub";
import { safeExternalLink } from "@/utils/externalLinks";
import { isDesktopShell } from "@/shell";
import { localLibraryImage, libraryCardImageURL } from "@/utils/libraryMarkdown";
import "@/library.css";

// 桌面端没有标签页：在那里 target="_blank" 会被交给 Windows 打开系统浏览器，
// 因此改为同窗口内打开；网页端保持新标签页。
const toolHref = "/library-tools/fefw-planner/planner.html?from=library";
const toolTarget = isDesktopShell() ? undefined : "_blank";
const tools = [{ key: "fefw-planner" }];
const emptyMedia: MediaOverview = { total: 0, counts: {}, recent: [], previews: {}, partial: false };
const collections = ref<Collection[]>([]),
  loading = ref(false),
  error = ref(""),
  creating = ref(false),
  media = ref<MediaOverview>(emptyMedia),
  keyword = ref(""),
  category = ref("");
let sequence = 0, mediaSequence = 0;

const categories = computed(() => [...new Set(collections.value.map((c) => c.category).filter(Boolean))]),
  filtered = computed(() =>
    collections.value.filter(
      (c) =>
        (!category.value || c.category === category.value) &&
        `${c.name} ${c.description}`.includes(keyword.value.trim()),
    ),
  );
const recentNames = computed(() => recentMediaNames(media.value.recent));
const mediaBadge = computed(() => {
  if (media.value.partial && media.value.total === 0) return "统计暂不可用";
  return `${media.value.total} 件`;
});

function safeCover(url: string) {
  return localLibraryImage(url) || (/^https?:\/\//i.test(url) && safeExternalLink(url));
}
function templateName(t: string) {
  return ({ general: "通用资料库", wiki: "Wiki 资料库", recipe: "个人菜谱" } as Record<string, string>)[t] || t;
}
function date(value: string) {
  return new Date(value).toLocaleDateString();
}
function coverURL(id: number, coverId?: string) {
  return mediaCoverURL(coverId || String(id));
}
async function load() {
  const seq = ++sequence;
  loading.value = true;
  error.value = "";
  try {
    const rows = await listCollections();
    if (seq === sequence) collections.value = rows;
  } catch (e) {
    if (seq === sequence) error.value = (e as Error).message;
  } finally {
    if (seq === sequence) loading.value = false;
  }
}
// 概览失败不阻塞页面：卡片降级为“统计暂不可用 + 完整空状态”。
async function loadMedia() {
  const seq = ++mediaSequence;
  try {
    const overview = await loadMediaOverview();
    if (seq === mediaSequence) media.value = overview;
  } catch {
    if (seq === mediaSequence) media.value = { ...emptyMedia, partial: true };
  }
}
onMounted(() => {
  void load();
  void loadMedia();
});
onBeforeUnmount(() => {
  sequence++;
  mediaSequence++;
});
</script>