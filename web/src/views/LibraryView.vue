<template>
  <section class="library-page">
    <nav class="library-breadcrumb">
      <RouterLink to="/explore">探索</RouterLink><span>/</span
      ><span>我的资料库</span>
    </nav>
    <PageHeader
      eyebrow="个人资料"
      title="我的资料库"
      description="整理常用资料，留下自己的结论与实践心得。"
      ><template #actions
        ><el-button type="primary" @click="creating = true"
          >新建专题</el-button
        ></template
      ></PageHeader
    >
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
  </section>
</template>
<script setup lang="ts">
import { computed, onMounted, ref, onBeforeUnmount } from "vue";
import PageHeader from "@/components/PageHeader.vue";
import CollectionDialog from "@/components/library/CollectionDialog.vue";
import { listCollections, type Collection } from "@/api/library";
import { safeExternalLink } from "@/utils/externalLinks";
import { localLibraryImage, libraryCardImageURL } from "@/utils/libraryMarkdown";
import "@/library.css";
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
