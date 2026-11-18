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
        <el-input v-model="query" aria-label="媒体标题" placeholder="搜索标题" :disabled="batchBusy" />
        <el-select v-model="sort" aria-label="排序" :disabled="batchBusy">
          <el-option v-for="option in mediaSortOptions" :key="option.value" :label="option.label" :value="option.value" />
        </el-select>
        <el-checkbox v-if="r18Relevant" v-model="showR18" :disabled="batchBusy" @change="load(1)">显示 R18 与待分类的 Jellyfin 内容</el-checkbox>
        <div class="media-view-toggle" role="group" aria-label="视图">
          <button type="button" :aria-pressed="view === 'gallery'" :disabled="batchBusy" @click="view = 'gallery'">画廊</button>
          <button type="button" :aria-pressed="view === 'list'" :disabled="batchBusy" @click="view = 'list'">列表</button>
        </div>
        <el-button native-type="submit" :disabled="batchBusy">搜索</el-button>
        <el-button v-if="kind === 'movie' && !batchMode" :disabled="loading || !!error || !items.length" @click="batchMode = true">批量删除</el-button>
      </form>

      <section v-if="batchMode" class="media-batch" aria-label="批量选择电影">
        <div class="media-batch__controls">
          <el-checkbox :model-value="allSelected" :indeterminate="selectedIDs.length > 0 && !allSelected" :disabled="loading || batchBusy || !items.length" @change="selectPage(Boolean($event))">全选当前页</el-checkbox>
          <span class="media-muted" role="status">已选择 {{ selectedIDs.length }} 项</span>
          <div class="media-batch__actions">
            <el-button type="danger" plain :disabled="!selectedIDs.length || loading || batchBusy" :loading="deleting" @click="confirmDelete">删除所选（{{ selectedIDs.length }}）</el-button>
            <el-button :disabled="batchBusy" @click="exitBatchMode">退出选择</el-button>
          </div>
        </div>
        <p class="media-muted">仅选择当前页，翻页或更改筛选会清空选择。</p>
        <el-alert v-if="deleteError" type="error" :title="deleteError" :closable="false" />
      </section>

      <section class="media-results" aria-live="polite">
        <el-alert v-if="error" type="error" :title="error" :closable="false" />
        <p v-if="loading" class="media-muted" role="status">正在读取媒体…</p>
        <el-empty v-else-if="!error && !items.length" description="暂无媒体" />
        <div v-else-if="view === 'gallery'" class="media-grid">
          <div v-for="item in sortedItems" :key="item.id" class="media-card-wrap" :class="{ 'is-selected': selectedIDs.includes(item.id) }">
            <el-checkbox v-if="batchMode" class="media-choice" :aria-label="`勾选《${item.Title}》`" :model-value="selectedIDs.includes(item.id)" :disabled="batchBusy" @change="setItemSelected(item.id, Boolean($event))" />
          <button type="button" class="media-card" :aria-label="batchMode ? `选择《${item.Title}》` : undefined" :aria-pressed="batchMode ? selectedIDs.includes(item.id) : undefined" :disabled="batchBusy" @click="openItem(item)">
            <img v-if="item.cover_id" class="media-cover" :src="mediaCoverURL(item.cover_id)" alt="" loading="lazy" />
            <span v-else class="media-cover media-cover--empty">无封面</span>
            <strong>{{ item.Title }}</strong>
            <span class="media-card__meta">{{ mediaSubtitle(item) }}</span>
            <span v-if="item.Description" class="media-card__note">{{ item.Description }}</span>
          </button>
          </div>
        </div>
        <ul v-else class="media-list">
          <li v-for="item in sortedItems" :key="item.id" class="media-list-item" :class="{ 'is-selected': selectedIDs.includes(item.id) }">
            <el-checkbox v-if="batchMode" class="media-choice media-choice--list" :aria-label="`勾选《${item.Title}》`" :model-value="selectedIDs.includes(item.id)" :disabled="batchBusy" @change="setItemSelected(item.id, Boolean($event))" />
            <button type="button" class="media-list__row" :aria-label="batchMode ? `选择《${item.Title}》` : undefined" :aria-pressed="batchMode ? selectedIDs.includes(item.id) : undefined" :disabled="batchBusy" @click="openItem(item)">
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
          :disabled="batchBusy"
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

        <section v-if="canPlayLocally" class="media-play">
          <header class="media-play__head">
            <div>
              <h3>本地播放</h3>
              <p>由 Windows Player Bridge 调用 PotPlayer，直接播放 NAS 上的原始文件，不经过服务器中转。</p>
            </div>
            <el-button type="primary" @click="startLocalPlayback">使用 PotPlayer 播放</el-button>
          </header>
          <p class="media-play__state">{{ playState }}</p>
          <div class="media-play__actions">
            <el-button text @click="copyPlayLink">复制播放链接</el-button>
            <button type="button" class="media-play__toggle" @click="showGuide = !showGuide">
              {{ showGuide ? "收起检查清单" : "没反应？查看检查清单" }}
            </button>
          </div>
          <ol v-if="showGuide" class="media-play__guide">
            <li>已在本机运行 <code>tools\lifeos-player-bridge\scripts\install.ps1</code>（注册 <code>lifeos-player://</code> 协议）</li>
            <li>PotPlayer 已安装，或 config 里指定了 <code>player.executable</code></li>
            <li>NAS 映射盘可访问：资源管理器里能打开对应盘符目录</li>
            <li>Jellyfin 可访问：家庭网络内能打开 <code>http://192.168.31.122:8096</code></li>
            <li><code>path_mappings</code> 的容器路径与本机盘符一一对应</li>
          </ol>
          <div class="media-play__actions">
            <button type="button" class="media-play__toggle" @click="showQuietGuide = !showQuietGuide">
              {{ showQuietGuide ? "收起说明" : "每次都要确认？让它不再弹窗" }}
            </button>
          </div>
          <div v-if="showQuietGuide" class="media-play__quiet">
            <p>「此站点正在尝试打开 player-bridge.exe」是浏览器的安全确认，网页和桥接都无法跳过，只能让浏览器记住这一个来源。</p>
            <p><strong>Chrome / Edge</strong>：用<strong>管理员</strong> PowerShell 执行下面这条（只对当前来源生效，可撤销），然后完全退出浏览器再打开。</p>
            <div class="media-play__cmd">
              <code>{{ quietCommand }}</code>
              <el-button text @click="copyQuietCommand">复制</el-button>
            </div>
            <p class="media-play__log">撤销：把命令末尾的 <code>-Origin ...</code> 换成 <code>-Origin {{ origin }} -Remove</code>。Chrome 88 之后对话框默认不显示「始终允许」勾选框，这条命令会一并打开它。</p>
            <p class="media-play__log"><strong>Firefox</strong>：<code>about:config</code> 里把 <code>network.protocol-handler.warn-external.lifeos-player</code> 设为 <code>false</code>，不需要管理员。</p>
          </div>
          <p class="media-play__log">失败原因看 <code>%LOCALAPPDATA%\LifeOS\PlayerBridge\logs\bridge.log</code>，或在 Bridge 目录运行 <code>player-bridge.exe -verify</code>。</p>
        </section>
      </template>
    </el-dialog>
  </PageShell>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import PageHeader from "../components/PageHeader.vue";
import PageShell from "../components/PageShell.vue";
import { deleteMovies, listMedia, mediaCoverURL, type MediaItem } from "@/api/media";
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
const router = useRouter();
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
const sort = ref<MediaSort>("recent");
const showR18 = ref(false);
const view = ref<"gallery" | "list">("gallery");
const page = ref(1);
const mediaTotal = ref(0);
const error = ref("");
const loading = ref(false);
const showGuide = ref(false);
const showQuietGuide = ref(false);
const desktop = ref(true);
const playRequested = ref(false);
const batchMode = ref(false);
const selectedIDs = ref<number[]>([]);
const confirming = ref(false);
const deleting = ref(false);
const deleteError = ref("");
const batchBusy = computed(() => confirming.value || deleting.value);
const allSelected = computed(() => items.value.length > 0 && items.value.every(item => selectedIDs.value.includes(item.id)));
let request = 0;

const PLAY_SCHEME = "lifeos-player://play";

// 弹窗只能在浏览器侧关掉，所以把「针对当前来源」的那条命令生成出来，避免用户自己拼。
const origin = computed(() => (typeof window === "undefined" ? "" : window.location.origin));
const quietCommand = computed(
  () => `tools\\lifeos-player-bridge\\scripts\\allow-browser-launch.ps1 -Origin ${origin.value}`,
);

// 剧集需要季/集列表，所以卡片改为进入详情页；电影保持原来的详情弹窗。
function openItem(item: MediaItem) {
  if (batchMode.value) {
    setItemSelected(item.id, !selectedIDs.value.includes(item.id));
    return;
  }
  if (item.Type === "series" && item.external_id) {
    void router.push(`/explore/library/media/series/${item.external_id}`);
    return;
  }
  selected.value = item;
}

function setItemSelected(id: number, checked: boolean) {
  if (batchBusy.value || !items.value.some(item => item.id === id)) return;
  selectedIDs.value = checked ? [...new Set([...selectedIDs.value, id])] : selectedIDs.value.filter(value => value !== id);
  deleteError.value = "";
}

function selectPage(checked: boolean) {
  if (batchBusy.value) return;
  selectedIDs.value = checked ? items.value.map(item => item.id) : [];
  deleteError.value = "";
}

function exitBatchMode() {
  batchMode.value = false;
  selectedIDs.value = [];
  deleteError.value = "";
}

async function confirmDelete() {
  if (batchBusy.value || loading.value || !selectedIDs.value.length) return;
  const ids = [...selectedIDs.value];
  const context = request;
  confirming.value = true;
  try {
    await ElMessageBox.confirm(
      `将从 LifeOS 删除所选 ${ids.length} 条电影记录及其收藏、笔记，无法撤销。不会删除 NAS 文件或 Jellyfin 媒体；下次同步可能重新导入。`,
      `删除 ${ids.length} 条电影记录？`,
      { confirmButtonText: "确认删除", cancelButtonText: "取消", type: "warning", confirmButtonClass: "el-button--danger" },
    );
  } catch {
    return;
  } finally {
    confirming.value = false;
  }
  if (context !== request) return;
  deleting.value = true;
  deleteError.value = "";
  try {
    const deleted = await deleteMovies(ids);
    if (context !== request) return;
    selectedIDs.value = [];
    ElMessage.success(`已删除 ${deleted} 条电影记录`);
    const lastPage = Math.max(1, Math.ceil((mediaTotal.value - deleted) / 24));
    await load(Math.min(page.value, lastPage));
  } catch (cause) {
    if (context === request) deleteError.value = cause instanceof Error ? cause.message : "媒体删除失败，请重试。";
  } finally {
    deleting.value = false;
  }
}

async function copyQuietCommand() {
  try {
    await navigator.clipboard.writeText(quietCommand.value);
    ElMessage.success("命令已复制，请在管理员 PowerShell 中执行");
  } catch {
    ElMessage.warning("复制失败，请手动选择命令文本");
  }
}

// 自定义协议没有回调：只能确认“已提交启动请求”，不能声称播放成功。
const playState = computed(() =>
  playRequested.value
    ? "已向 Windows 提交启动请求（浏览器可能弹出确认）。协议没有回调，所以这里只表示已提交，播放是否成功请以 PotPlayer 是否打开为准。"
    : "点击后由 Windows 确认是否启动 PotPlayer；本机需已安装 Player Bridge。",
);

const canPlayLocally = computed(
  () => desktop.value && kind.value === "movie" && Boolean(selected.value?.external_id),
);

const playURL = computed(() =>
  selected.value?.external_id ? `${PLAY_SCHEME}?itemId=${encodeURIComponent(selected.value.external_id)}` : "",
);

function startLocalPlayback() {
  if (!playURL.value) return;
  playRequested.value = true;
  window.location.href = playURL.value;
}

async function copyPlayLink() {
  if (!playURL.value) return;
  try {
    await navigator.clipboard.writeText(playURL.value);
    ElMessage.success("播放链接已复制");
  } catch {
    ElMessage.warning("复制失败，请手动复制： " + playURL.value);
  }
}

function syncViewport() {
  desktop.value = window.matchMedia("(pointer: fine)").matches && window.innerWidth > 820;
}

const sortedItems = computed(() => sortMediaItems(items.value, sort.value));

async function load(next: number) {
  selectedIDs.value = [];
  deleteError.value = "";
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
      page: next,
      showR18: showR18.value,
    });
    if (current !== request) return;
    const lastPage = Math.max(1, Math.ceil(result.total / 24));
    if (next > lastPage) {
      await load(lastPage);
      return;
    }
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
  exitBatchMode();
  selected.value = null;
  playRequested.value = false;
  showGuide.value = false;
  query.value = "";
  sort.value = "recent";
  void load(1);
});
onMounted(() => {
  syncViewport();
  window.addEventListener("resize", syncViewport);
  void load(1);
});
onBeforeUnmount(() => {
  request += 1;
  window.removeEventListener("resize", syncViewport);
});
</script>

<style scoped>
.media-batch { display: grid; gap: 10px; margin-bottom: 16px; padding: 14px 16px; border-radius: var(--radius-surface); background: var(--material-reading-fill); border: 1px solid var(--material-surface-edge); }
.media-batch__controls { display: flex; flex-wrap: wrap; align-items: center; gap: 12px 20px; }
.media-batch__controls > .el-checkbox { margin-right: 0; min-height: 44px; }
.media-batch__actions { display: flex; flex-wrap: wrap; gap: 8px; margin-left: auto; }
.media-batch__actions > .el-button { margin-left: 0; min-height: 44px; }
.media-card-wrap { position: relative; display: grid; min-width: 0; }
.media-card-wrap > .media-card { width: 100%; }
.media-choice { position: absolute; z-index: 1; top: 18px; left: 18px; display: flex; justify-content: center; width: 44px; height: 44px; margin: 0; border-radius: var(--radius-field); background: var(--material-reading-fill); }
.is-selected > .media-card, .is-selected > .media-list__row { outline: 2px solid var(--color-accent); outline-offset: -2px; }
.media-list-item { display: flex; align-items: center; gap: 8px; min-width: 0; }
.media-choice--list { position: static; flex: 0 0 44px; background: transparent; }
.media-list__row { flex: 1; min-width: 0; }
.media-list__row:focus-visible { outline: 2px solid var(--color-focus); outline-offset: 2px; }
.media-card:disabled, .media-list__row:disabled, .media-view-toggle button:disabled { cursor: default; }
@media (max-width: 700px) {
  .media-toolbar > .el-button, .media-view-toggle > button { min-height: 44px; }
  .media-batch__actions { margin-left: 0; width: 100%; }
}
</style>
