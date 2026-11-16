<template>
  <PageShell variant="standard">
    <PageHeader title="媒体中心">
      <template #breadcrumbs><RouterLink to="/explore/library">我的资料库</RouterLink></template>
    </PageHeader>
    <form class="media-filters" @submit.prevent="load(1)">
      <el-input v-model="query" aria-label="媒体标题" placeholder="搜索标题" />
      <el-select v-model="kind" aria-label="媒体类型" @change="load(1)">
        <el-option v-for="option in types" :key="option.value" v-bind="option" />
      </el-select>
      <el-select v-model="source" aria-label="媒体来源" @change="load(1)">
        <el-option label="全部来源" value="all" />
        <el-option label="Calibre" value="calibre" />
        <el-option label="Jellyfin" value="jellyfin" />
      </el-select>
      <el-button native-type="submit">搜索</el-button>
    </form>
    <el-alert v-if="error" type="error" :title="error" :closable="false" />
    <p v-if="loading" role="status">正在读取媒体…</p>
    <el-empty v-else-if="!error && items.length === 0" description="暂无媒体" />
    <div v-else class="media-grid">
      <button v-for="item in items" :key="item.id" class="media-card" @click="selected = item">
        <img v-if="item.cover_id" class="media-cover" :src="coverURL(item.cover_id)" alt="" loading="lazy" />
        <span v-else class="media-cover media-cover--empty">无封面</span>
        <strong>{{ item.Title }}</strong>
        <span>{{ typeName(item.Type) }} · {{ item.Source }}</span>
        <span v-if="item.Authors?.length || item.Year">{{ [...(item.Authors ?? []), item.Year || ''].filter(Boolean).join(' · ') }}</span>
      </button>
    </div>
    <el-pagination v-if="total > 24" :current-page="page" :page-size="24" :total="total"
      layout="prev, pager, next" @current-change="load" />
    <el-dialog :model-value="selected !== null" :title="selected?.Title" width="min(560px, 90vw)" @close="selected = null">
      <template v-if="selected">
        <img v-if="selected.cover_id" class="media-cover media-cover--detail" :src="coverURL(selected.cover_id)" alt="" />
        <p>{{ typeName(selected.Type) }} · {{ selected.Source }} {{ selected.Year || '' }}</p>
        <p v-if="selected.Authors?.length">{{ selected.Authors.join('、') }}</p>
        <p v-if="selected.Description" class="media-description">{{ selected.Description }}</p>
        <p v-if="selected.Tags?.length">{{ selected.Tags.join('、') }}</p>
        <p v-if="selected.favorite">已收藏</p>
        <p v-if="selected.notes" class="media-description">{{ selected.notes }}</p>
      </template>
    </el-dialog>
  </PageShell>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import PageShell from '../components/PageShell.vue'

interface MediaItem {
  cover_id?: string
  id: number
  Type: string
  Title: string
  Source: string
  Authors?: string[]
  Tags?: string[]
  Description?: string
  Year?: number
  favorite?: boolean
  notes?: string
}

const types = [
  { label: '全部类型', value: 'all' }, { label: '电子书', value: 'book' },
  { label: '电影', value: 'movie' }, { label: '电视剧', value: 'series' },
]
const typeName = (value: string) => types.find((type) => type.value === value)?.label ?? value
const coverURL = (id: string) => `/api/v1/library/images/${encodeURIComponent(id)}?variant=card-v1`
const items = ref<MediaItem[]>([])
const selected = ref<MediaItem | null>(null)
const query = ref('')
const kind = ref('all')
const source = ref('all')
const page = ref(1)
const total = ref(0)
const error = ref('')
const loading = ref(false)
let request = 0

async function load(next: number) {
  const current = ++request
  error.value = ''
  loading.value = true
  try {
    const params = new URLSearchParams({ q: query.value, type: kind.value === "all" ? "" : kind.value, source: source.value === "all" ? "" : source.value, page: String(next) })
    const response = await fetch('/api/v1/library/media?' + params)
    if (!response.ok) throw new Error()
    const { data } = await response.json()
    if (current !== request) return
    items.value = data.items
    total.value = data.total
    page.value = next
  } catch {
    if (current === request) error.value = '媒体读取失败，请重试。'
  } finally {
    if (current === request) loading.value = false
  }
}
onMounted(() => load(1))
</script>

<style scoped>
.media-filters, .media-grid { display: grid; gap: 12px; margin-block: 16px; }
.media-filters { grid-template-columns: repeat(auto-fit, minmax(min(100%, 140px), 1fr)); }
.media-grid { grid-template-columns: repeat(auto-fit, minmax(min(100%, 220px), 1fr)); }
.media-card {
  display: grid; gap: 8px; text-align: start; padding: 16px;
  color: inherit; overflow-wrap: anywhere; cursor: pointer; font: inherit;
  border: 1px solid var(--material-surface-border); border-radius: var(--radius-surface);
  background: var(--material-surface-bg);
}
.media-card:focus-visible { outline: 2px solid var(--color-accent); }
.media-cover { width: 100%; aspect-ratio: 2 / 3; object-fit: contain; border-radius: var(--radius-field); background: var(--color-surface); }
.media-cover--empty { display: grid; place-items: center; }
.media-cover--detail { max-width: 160px; }
.media-description { white-space: pre-wrap; overflow-wrap: anywhere; }
</style>
