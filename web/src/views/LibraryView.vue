<template>
  <PageShell class="library-page" variant="standard">

    <PageHeader
      eyebrow="个人资料"
      title="我的资料库"
      description="整理常用资料，留下自己的结论与实践心得。"
      ><template #breadcrumbs><nav class="library-breadcrumb">
      <RouterLink to="/explore">探索</RouterLink><span>/</span
      ><span>我的资料库</span>
    </nav></template><template #actions
        ><el-button type="primary" @click="creating = true"
          >新建专题</el-button
        ></template
      ></PageHeader
    >
    <section class="library-tools" aria-labelledby="library-tools-title">
      <header class="library-tools-head">
        <div>
          <h2 id="library-tools-title">工具</h2>
          <p>独立的小工具，规划随账号保存（Web 与桌面端共用）。</p>
        </div>
        <span class="library-tools-count">1 个</span>
      </header>
      <div class="library-tool-grid">
        <a
          class="library-tool-card"
          :href="toolHref"
          :target="toolTarget"
          :rel="toolTarget ? 'noopener' : undefined"
        >
          <span class="library-tool-eyebrow">角色规划</span>
          <h3>万缕千丝 · 挖角手帐</h3>
          <p>四线挖角与培养规划：主力 / 板凳、名声与交涉条件、职业路线与备注。</p>
          <span class="library-tool-action">打开手帐</span>
        </a>
      </div>
    </section>
    <el-alert v-if="error" :title="error" type="error" :closable="false"
      ><el-button text @click="load">重试</el-button></el-alert
    >
    <p v-if="loading" role="status">正在读取专题…</p>
    <template v-else
      ><div v-if="collections.length" class="library-home-filter">
        <el-input
          v-model="keyword"
          aria-label="查找专题"
          placeholder="查找专题名称或说明"
          clearable
        /><el-select
          v-model="category"
          clearable
          placeholder="全部分类"
          aria-label="专题分类"
          ><el-option v-for="c in categories" :key="c" :label="c" :value="c"
        /></el-select>
      </div>
      <el-empty
        v-if="!collections.length && !error"
        description="先为一个常用主题建立自己的资料库"
        ><el-button type="primary" @click="creating = true"
          >创建第一份专题</el-button
        ></el-empty
      >
      <p v-else-if="!filtered.length" class="library-muted">
        没有符合条件的专题。
      </p>
      <div class="library-collections">
        <RouterLink
          v-for="c in filtered"
          :key="c.id"
          :to="`/explore/library/${c.id}`"
          class="library-collection-card"
          ><img
            v-if="safeCover(c.cover_url)"
            :src="libraryCardImageURL(c.cover_url)"
            alt=""
            loading="lazy"
            decoding="async"
            referrerpolicy="no-referrer"
          /><span class="library-muted"
            >{{ c.category || "未分类" }} · {{ templateName(c.template) }}</span
          >
          <h2>{{ c.name }}</h2>
          <p>{{ c.description || "从一篇页面开始，逐步整理自己的资料。" }}</p>
          <small
            >{{ c.page_count || 0 }} 篇页面 ·
            {{ date(c.updated_at) }} 更新</small
          ></RouterLink
        >
      </div></template
    ><CollectionDialog
      :open="creating"
      @close="creating = false"
      @saved="load"
    />
  </PageShell>
</template>
<script setup lang="ts">
import PageShell from '@/components/PageShell.vue'
import { computed, onMounted, ref, onBeforeUnmount } from "vue";
import PageHeader from "@/components/PageHeader.vue";
import CollectionDialog from "@/components/library/CollectionDialog.vue";
import { listCollections, type Collection } from "@/api/library";
import { safeExternalLink } from "@/utils/externalLinks";
import { isDesktopShell } from "@/shell";
import { localLibraryImage, libraryCardImageURL } from "@/utils/libraryMarkdown";
import "@/library.css";

// 桌面端没有标签页：在那里 target="_blank" 会被交给 Windows 打开系统浏览器，
// 因此改为同窗口内打开；网页端保持新标签页。
const toolHref = "/library-tools/fefw-planner/planner.html?from=library";
const toolTarget = isDesktopShell() ? undefined : "_blank";
const collections = ref<Collection[]>([]),
  loading = ref(false),
  error = ref(""),
  creating = ref(false),
  keyword = ref(""),
  category = ref("");
let sequence = 0;
const categories = computed(() => [
    ...new Set(collections.value.map((c) => c.category).filter(Boolean)),
  ]),
  filtered = computed(() =>
    collections.value.filter(
      (c) =>
        (!category.value || c.category === category.value) &&
        `${c.name} ${c.description}`.includes(keyword.value.trim()),
    ),
  );
function safeCover(url: string) {
  return localLibraryImage(url) || (/^https?:\/\//i.test(url) && safeExternalLink(url));
}
function templateName(t: string) {
  return (
    { general: "通用资料库", wiki: "Wiki 资料库", recipe: "个人菜谱" }[t] || t
  );
}
function date(value: string) {
  return new Date(value).toLocaleDateString();
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
onMounted(load);
onBeforeUnmount(() => sequence++);
</script>
