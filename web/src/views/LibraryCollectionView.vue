<template>
  <section class="library-page">
    <nav class="library-breadcrumb">
      <RouterLink to="/explore">探索</RouterLink><span>/</span
      ><RouterLink to="/explore/library">我的资料库</RouterLink
      ><template v-if="collection"
        ><span>/</span
        ><RouterLink :to="collectionURL">{{ collection.name }}</RouterLink
        ><template v-if="current"
          ><span>/</span><span>{{ current.title }}</span></template
        ></template
      >
    </nav>
    <el-alert v-if="error" :title="error" type="error" :closable="false"
      ><el-button v-if="!editing" text @click="load"
        >重新读取</el-button
      ></el-alert
    >
    <p v-if="loading" role="status">正在读取资料…</p>
    <template v-if="collection"
      ><header class="library-topic-header">
        <div>
          <span class="library-muted">{{
            collection.category || "个人专题"
          }}</span>
          <h1>{{ collection.name }}</h1>
          <p>{{ collection.description }}</p>
        </div>
        <div class="library-actions">
          <el-button
            :disabled="busy || uploading || editing || loading"
            @click="collectionDialog = true"
            >编辑专题</el-button
          ><el-button
            :disabled="busy || uploading || editing || loading"
            @click="exportTopic"
            >导出</el-button
          ><el-button
            type="primary"
            :disabled="busy || uploading || editing || loading"
            @click="startNew"
            >新建页面</el-button
          >
        </div>
      </header>
      <div class="library-topic-tools">
        <el-radio-group
          :model-value="collection.display_mode"
          aria-label="专题展示方式"
          :disabled="busy || uploading || editing || loading"
          @change="changeMode"
          ><el-radio-button value="wiki">Wiki 目录</el-radio-button
          ><el-radio-button value="list"
            >卡片清单</el-radio-button
          ></el-radio-group
        ><el-button
          text
          :disabled="busy || uploading || editing || loading"
          @click="newSection"
          >新建分组</el-button
        ><el-button
          text
          :disabled="busy || uploading || editing || loading"
          @click="cleanupImages"
          >清理未使用图片</el-button
        ><el-button
          text
          type="danger"
          :disabled="busy || uploading || editing || loading"
          @click="removeCollection"
          >删除专题</el-button
        >
      </div>
      <div
        class="library-topic-layout"
        :class="{
          'is-list': collection.display_mode === 'list',
          'has-reader': current || editing,
        }"
      >
        <aside class="library-directory" aria-label="专题目录">
          <form class="library-search" @submit.prevent="search">
            <el-input
              v-model="searchInput"
              placeholder="搜索标题、正文或标签"
              aria-label="专题内搜索"
              clearable
            /><el-button native-type="submit" :disabled="busy">搜索</el-button>
          </form>
          <nav class="library-sections" aria-label="分组目录">
            <RouterLink
              :to="filterURL('')"
              :aria-current="sectionFilter === '' ? 'page' : undefined"
              >全部页面</RouterLink
            ><RouterLink
              :to="filterURL('root')"
              :aria-current="sectionFilter === 'root' ? 'page' : undefined"
              >专题根目录</RouterLink
            >
            <div v-for="s in sections" :key="s.id" class="library-section-row">
              <RouterLink
                :to="filterURL(String(s.id))"
                :aria-current="
                  sectionFilter === String(s.id) ? 'page' : undefined
                "
                >{{ s.name }}</RouterLink
              ><el-button
                text
                :disabled="busy || uploading || editing || loading"
                :aria-label="`重命名分组 ${s.name}`"
                @click="renameSection(s)"
                >改名</el-button
              ><el-button
                text
                :disabled="busy || uploading || editing || loading"
                :aria-label="`删除分组 ${s.name}`"
                @click="removeSection(s)"
                >删除</el-button
              >
            </div>
          </nav>
          <p class="library-muted">
            {{ total }} 篇{{ query ? "匹配页面" : "页面" }}
          </p>
          <p v-if="!rows.length && !loading" class="library-muted">
            {{
              query
                ? "没有找到相关资料，试试其他关键词。"
                : "这里还没有页面。可以直接在根目录创建，也可以先建分组。"
            }}
          </p>
          <div class="library-page-list">
            <RouterLink
              v-for="p in rows"
              :key="p.id"
              :to="pageURL(p.id)"
              class="library-page-link"
              :aria-current="current?.id === p.id ? 'page' : undefined"
              ><img
                v-if="
                  collection.display_mode === 'list' && safeCover(p.cover_url)
                "
                :src="p.cover_url"
                alt=""
                referrerpolicy="no-referrer"
                loading="lazy"
              />
              <h3>{{ p.title }}</h3>
              <span class="library-muted"
                >{{ sectionName(p.section_id) }} ·
                {{ date(p.updated_at) }}</span
              >
              <p>{{ summary(p.summary) }}</p>
              <div class="library-tags">
                <span v-for="tag in p.tags" :key="tag">{{ tag }}</span>
              </div></RouterLink
            >
          </div>
          <el-pagination
            v-if="total > 50"
            layout="prev,pager,next"
            :current-page="pageNumber"
            :page-size="50"
            :total="total"
            :pager-count="5"
            @current-change="paginate"
          />
        </aside>
        <article v-if="editing" class="library-reader library-editor">
          <h2>{{ editingID ? "编辑页面" : "新建页面" }}</h2>
          <el-form label-position="top" @submit.prevent="save"
            ><el-form-item label="页面标题"
              ><el-input
                v-model="form.title"
                aria-label="页面标题"
                maxlength="200"
            /></el-form-item>
            <div class="library-form-grid">
              <el-form-item label="所属分组"
                ><el-select v-model="form.section_id" aria-label="页面分组"
                  ><el-option :value="0" label="专题根目录" /><el-option
                    v-for="s in sections"
                    :key="s.id"
                    :label="s.name"
                    :value="s.id" /></el-select></el-form-item
              ><el-form-item label="标签（可选，逗号分隔）"
                ><el-input v-model="tagsInput" aria-label="页面标签"
              /></el-form-item>
            </div>
            <el-form-item label="页面封面（可选）">
              <PageCoverEditor
                v-model="form.cover_url"
                :body="form.body"
                :collection="collection.id"
                :disabled="busy || bodyUploading"
                @uploading="coverUploading = $event"
              />
            </el-form-item>
            <div class="library-editor-tabs">
              <el-radio-group v-model="editorTab" :disabled="uploading"
                ><el-radio-button value="edit">Markdown 正文</el-radio-button
                ><el-radio-button value="preview"
                  >阅读预览</el-radio-button
                ></el-radio-group
              >
            </div>
            <LibraryImageEditor
              v-if="editorTab === 'edit'"
              v-model="form.body"
              :disabled="busy || coverUploading"
              @uploading="bodyUploading = $event"
            /><MarkdownReader
              v-else
              :body="form.body"
              :collection="collection.id"
            />
            <p class="library-muted">
              支持 Markdown 表格、待办清单、本地图片和外部图片
              URL。内部链接可从其他页面复制后粘贴；HTML 脚本与危险链接不会执行。
            </p>
            <h3>外部资料来源</h3>
            <p class="library-muted">
              记录原始资料地址；正文中的结论和实践心得由你自己整理。
            </p>
            <div
              v-for="(source, i) in form.sources"
              :key="i"
              class="library-source-form"
            >
              <el-input
                v-model="source.url"
                :aria-label="`来源 ${i + 1} URL`"
                placeholder="来源 URL"
              /><el-input
                v-model="source.name"
                placeholder="来源名称（可选）"
                maxlength="200"
              /><el-input
                v-model="source.note"
                placeholder="简短备注（可选）"
                maxlength="1000"
              /><el-button
                text
                type="danger"
                :aria-label="`移除来源 ${i + 1}`"
                @click="form.sources.splice(i, 1)"
                >移除</el-button
              >
            </div>
            <el-button
              text
              :disabled="form.sources.length >= 30"
              @click="form.sources.push({ url: '', name: '', note: '' })"
              >添加来源</el-button
            ></el-form
          >
          <footer class="library-editor-footer">
            <el-button :disabled="busy" @click="cancelEdit">取消编辑</el-button
            ><el-button
              type="primary"
              :loading="busy"
              :disabled="uploading"
              @click="save"
              >保存页面</el-button
            >
          </footer>
        </article>
        <article v-else-if="current" class="library-reader">
          <header class="library-reader-header">
            <span class="library-muted"
              >{{ sectionName(current.section_id) }} ·
              {{ date(current.updated_at) }} 更新</span
            >
            <h2>{{ current.title }}</h2>
            <div class="library-actions">
              <el-button :disabled="busy" @click="startEdit">编辑页面</el-button
              ><el-button text @click="copyLink">复制内部链接</el-button
              ><el-button
                text
                type="danger"
                :disabled="busy"
                @click="removePage"
                >删除页面</el-button
              >
            </div>
            <p v-if="copied" class="library-muted">
              内部链接：<code>{{ copied }}</code>
            </p>
            <div class="library-tags">
              <span v-for="tag in current.tags" :key="tag">{{ tag }}</span>
            </div>
          </header>
          <img
            v-if="safeCover(current.cover_url)"
            class="library-reader-cover"
            :src="current.cover_url"
            alt="页面封面"
            referrerpolicy="no-referrer"
          /><MarkdownReader
            :body="current.body"
            :collection="collection.id"
            :revision="current.version"
          />
          <section v-if="current.sources.length" class="library-sources">
            <h3>外部资料来源</h3>
            <div v-for="(s, i) in current.sources" :key="i">
              <ExternalLink :url="s.url" :name="s.name || s.url" />
              <p v-if="s.note">{{ s.note }}</p>
            </div>
          </section>
        </article>
        <div v-else-if="!loading" class="library-reading-hint">
          <h2>
            {{
              collection.display_mode === "wiki"
                ? "从目录打开一篇资料"
                : "自己的常用资料，随时翻阅"
            }}
          </h2>
          <p>选择已有页面开始阅读，或新建页面记录自己的经验。</p>
        </div>
      </div> </template
    ><CollectionDialog
      :open="collectionDialog"
      :collection="collection"
      @close="collectionDialog = false"
      @saved="collectionSaved"
    />
  </section>
</template>
<script setup lang="ts">
import {
  computed,
  onMounted,
  onBeforeUnmount,
  reactive,
  ref,
  watch,
} from "vue";
import {
  useRoute,
  useRouter,
  onBeforeRouteLeave,
  onBeforeRouteUpdate,
} from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  getCollection,
  listPages,
  getPage,
  savePage,
  deletePage,
  saveSection,
  deleteSection,
  deleteCollection,
  saveCollection,
  exportCollection,
  cleanupLibraryImages,
  pageTemplate,
  type Collection,
  type Section,
  type LibraryPage,
  type PageRow,
  type PageInput,
  type CollectionInput,
} from "@/api/library";
import { localLibraryImage } from "@/utils/libraryMarkdown";
import { safeExternalLink } from "@/utils/externalLinks";
import CollectionDialog from "@/components/library/CollectionDialog.vue";
import PageCoverEditor from "@/components/library/PageCoverEditor.vue";
import LibraryImageEditor from "@/components/library/LibraryImageEditor.vue";
import MarkdownReader from "@/components/library/MarkdownReader.vue";
import ExternalLink from "@/components/ExternalLink.vue";
import "@/library.css";
const route = useRoute(),
  router = useRouter(),
  collection = ref<Collection>(),
  sections = ref<Section[]>([]),
  rows = ref<PageRow[]>([]),
  total = ref(0),
  current = ref<LibraryPage>(),
  loading = ref(false),
  busy = ref(false),
  bodyUploading = ref(false),
  coverUploading = ref(false),
  uploading = computed(() => bodyUploading.value || coverUploading.value),
  error = ref(""),
  collectionDialog = ref(false),
  editing = ref(false),
  editingID = ref<number>(),
  editorTab = ref("edit"),
  tagsInput = ref(""),
  searchInput = ref(""),
  copied = ref(""),
  initial = ref("");
const form = reactive<Omit<PageInput, "section_id"> & { section_id: number }>({
  title: "",
  body: "",
  section_id: 0,
  tags: [],
  sources: [],
  cover_url: "",
  version: 0,
});
const collectionID = computed(() => Number(route.params.collection)),
  query = computed(() => String(route.query.q || "")),
  sectionFilter = computed(() => String(route.query.section || "")),
  pageNumber = computed(() => Math.max(1, Number(route.query.p) || 1)),
  collectionURL = computed(() => ({
    path: `/explore/library/${collectionID.value}`,
    query: route.query,
  }));
const input = () => ({
  title: form.title,
  body: form.body,
  cover_url: form.cover_url,
  version: form.version,
  section_id: form.section_id || null,
  tags: tagsInput.value
    .split(/[,，]/)
    .map((s) => s.trim())
    .filter(Boolean),
  sources: form.sources.map((s) => ({ ...s })),
});
const dirty = computed(
  () => editing.value && JSON.stringify(input()) !== initial.value,
);
let sequence = 0;
function pageURL(id: number) {
  return {
    path: `/explore/library/${collectionID.value}/pages/${id}`,
    query: route.query,
  };
}
function filterURL(section: string) {
  return {
    path: `/explore/library/${collectionID.value}`,
    query: { q: query.value || undefined, section: section || undefined },
  };
}
function safeCover(url: string) {
  return (
    localLibraryImage(url) ||
    (/^https?:\/\//i.test(url) && safeExternalLink(url))
  );
}
function sectionName(id: number | null) {
  return sections.value.find((s) => s.id === id)?.name || "专题根目录";
}
function summary(text: string) {
  return text
    .replace(/[#*`>|\[\]]/g, "")
    .trim()
    .slice(0, 160);
}
function date(value: string) {
  return new Date(value).toLocaleDateString();
}
async function load() {
  const seq = ++sequence,
    col = collectionID.value;
  loading.value = true;
  if (collection.value?.id !== col) collection.value = undefined;
  error.value = "";
  copied.value = "";
  searchInput.value = query.value;
  try {
    const [detail, list] = await Promise.all([
      getCollection(col),
      listPages(col, query.value, sectionFilter.value, pageNumber.value),
    ]);
    if (seq !== sequence) return;
    collection.value = detail.collection;
    sections.value = detail.sections;
    rows.value = list.items;
    total.value = list.total;
    current.value = undefined;
    if (route.params.page) {
      const row = await getPage(col, Number(route.params.page));
      if (seq === sequence) current.value = row;
    }
  } catch (e) {
    if (seq === sequence) {
      current.value = undefined;
      error.value = (e as Error).message;
    }
  } finally {
    if (seq === sequence) loading.value = false;
  }
}
async function canLeave() {
  if (uploading.value) {
    ElMessage.warning("图片仍在上传，请稍候。");
    return false;
  }
  if (busy.value) return false;
  if (!dirty.value) return true;
  try {
    await ElMessageBox.confirm(
      "页面内容尚未保存，确定放弃修改？",
      "未保存的内容",
      { confirmButtonText: "放弃修改", cancelButtonText: "继续编辑" },
    );
    return true;
  } catch {
    return false;
  }
}
onBeforeRouteLeave(canLeave);
onBeforeRouteUpdate(async () => {
  if (!(await canLeave())) return false;
  editing.value = false;
});
watch(
  () => route.fullPath,
  () => {
    if (route.path.startsWith("/explore/library/")) load();
  },
);
onMounted(load);
onBeforeUnmount(() => sequence++);
function beforeUnload(event: BeforeUnloadEvent) {
  if (dirty.value || uploading.value) {
    event.preventDefault();
    event.returnValue = "";
  }
}
onMounted(() => window.addEventListener("beforeunload", beforeUnload));
onBeforeUnmount(() => window.removeEventListener("beforeunload", beforeUnload));
function populate(row: PageInput) {
  Object.assign(form, {
    title: row.title,
    body: row.body,
    cover_url: row.cover_url,
    version: row.version,
    section_id: row.section_id || 0,
    sources: row.sources.map((s) => ({ ...s })),
  });
  tagsInput.value = row.tags.join(", ");
  initial.value = JSON.stringify(input());
  editorTab.value = "edit";
  error.value = "";
  editing.value = true;
}
function startNew() {
  if (!collection.value) return;
  editingID.value = undefined;
  populate({
    title: "",
    body: pageTemplate(collection.value.template),
    tags: [],
    sources: [],
    section_id: /^\d+$/.test(sectionFilter.value)
      ? Number(sectionFilter.value)
      : null,
    cover_url: "",
    version: 0,
  });
}
function startEdit() {
  if (!current.value) return;
  editingID.value = current.value.id;
  populate(current.value);
}
async function cancelEdit() {
  if (await canLeave()) {
    editing.value = false;
    error.value = "";
  }
}
async function save() {
  if (busy.value || uploading.value) return;
  const payload = input();
  if (!payload.title.trim()) {
    error.value = "请填写页面标题。";
    return;
  }
  if (
    payload.tags.length > 20 ||
    payload.tags.some((t) => [...t].length > 40)
  ) {
    error.value = "最多 20 个标签，每个最多 40 字。";
    return;
  }
  if (payload.sources.some((s) => !safeExternalLink(s.url))) {
    error.value =
      "请检查来源地址，支持 HTTP、HTTPS 或有效的 Obsidian 打开链接。";
    return;
  }
  if (payload.cover_url && !safeCover(payload.cover_url)) {
    error.value = "请填写有效的图片 URL。";
    return;
  }
  busy.value = true;
  error.value = "";
  try {
    const row = await savePage(collectionID.value, editingID.value, payload);
    initial.value = JSON.stringify(input());
    editing.value = false;
    current.value = row;
    ElMessage.success("页面已保存");
    busy.value = false;
    await router.replace(pageURL(row.id));
    await load();
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}
async function search() {
  await router.push({
    path: `/explore/library/${collectionID.value}`,
    query: {
      q: searchInput.value.trim() || undefined,
      section: sectionFilter.value || undefined,
    },
  });
}
async function paginate(p: number) {
  await router.push({
    path: route.path,
    query: { ...route.query, p: String(p) },
  });
}
async function cleanupImages() {
  if (
    !(await confirm(
      "清理所有专题中上传超过 7 天、未被任何已保存页面或封面引用的图片？未保存的草稿不计入引用；请先保存草稿并确认已备份。不会删除其他页面仍在使用的图片。",
      "清理未使用图片",
    ))
  )
    return;
  await action(async () => {
    const count = await cleanupLibraryImages();
    ElMessage.success(`已清理 ${count} 张未使用图片`);
  });
}
async function confirm(text: string, title = "确认操作") {
  try {
    await ElMessageBox.confirm(text, title, {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning",
    });
    return true;
  } catch {
    return false;
  }
}
async function action(fn: () => Promise<unknown>) {
  if (busy.value || uploading.value) return;
  busy.value = true;
  error.value = "";
  try {
    await fn();
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}
async function newSection() {
  try {
    const result = await ElMessageBox.prompt(
      "分组只支持一层，页面也可以直接放在专题根目录。",
      "新建分组",
      {
        inputPlaceholder: "分组名称",
        confirmButtonText: "创建",
        cancelButtonText: "取消",
        inputValidator: (v) => !!v?.trim() || "请填写名称",
      },
    );
    await action(async () => {
      await saveSection(collectionID.value, undefined, result.value);
      await load();
    });
  } catch {
    /* cancelled */
  }
}
async function renameSection(s: Section) {
  try {
    const result = await ElMessageBox.prompt(
      "修改分组名称不会改变页面链接。",
      "重命名分组",
      {
        inputValue: s.name,
        confirmButtonText: "保存",
        cancelButtonText: "取消",
        inputValidator: (v) => !!v?.trim() || "请填写名称",
      },
    );
    await action(async () => {
      await saveSection(collectionID.value, s.id, result.value);
      await load();
    });
  } catch {
    /* cancelled */
  }
}
async function removeSection(s: Section) {
  if (!(await confirm(`删除分组「${s.name}」？页面不会被删除。`))) return;
  await action(async () => {
    try {
      await deleteSection(collectionID.value, s.id, false);
    } catch (e) {
      if ((e as { code?: string }).code !== "SECTION_NOT_EMPTY") throw e;
      if (
        !(await confirm(
          "该分组仍有页面。将全部页面移回专题根目录，然后删除分组？",
          "保留页面并移回根目录",
        ))
      )
        return;
      await deleteSection(collectionID.value, s.id, true);
    }
    if (sectionFilter.value === String(s.id)) {
      busy.value = false;
      await router.replace(filterURL("root"));
    }
    await load();
  });
}
async function removePage() {
  if (
    !current.value ||
    !(await confirm(
      `删除「${current.value.title}」？正文及来源将被删除，指向它的内部链接会显示不可用。`,
      "删除页面",
    ))
  )
    return;
  await action(async () => {
    await deletePage(
      collectionID.value,
      current.value!.id,
      current.value!.version,
    );
    busy.value = false;
    await router.replace(collectionURL.value);
    await load();
  });
}
async function removeCollection() {
  if (
    !collection.value ||
    !(await confirm(
      `删除专题「${collection.value.name}」及其中所有页面、正文、来源和分组？此操作不能撤销，建议先导出。`,
      "删除整个专题",
    ))
  )
    return;
  await action(async () => {
    await deleteCollection(collectionID.value, collection.value!.version);
    busy.value = false;
    await router.push("/explore/library");
  });
}
function collectionInput(c: Collection): CollectionInput {
  return {
    name: c.name,
    description: c.description,
    category: c.category,
    template: c.template,
    display_mode: c.display_mode,
    cover_url: c.cover_url,
    version: c.version,
  };
}
async function changeMode(value: unknown) {
  if (!collection.value) return;
  await action(async () => {
    collection.value = await saveCollection(collectionID.value, {
      ...collectionInput(collection.value!),
      display_mode: value as "wiki" | "list",
    });
  });
}
async function collectionSaved(row: Collection) {
  collection.value = row;
}
async function exportTopic() {
  await action(() => exportCollection(collectionID.value));
}
async function copyLink() {
  if (!current.value) return;
  copied.value = `/explore/library/${collectionID.value}/pages/${current.value.id}`;
  try {
    await navigator.clipboard.writeText(
      `[${current.value.title.replace(/[\[\]]/g, "")}](${copied.value})`,
    );
    ElMessage.success("Markdown 内部链接已复制");
  } catch {
    ElMessage.info("可选择下方内部链接复制，粘贴到 Markdown 链接中。");
  }
}
</script>
