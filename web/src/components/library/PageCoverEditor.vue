<template>
  <div class="library-cover-editor">
    <div class="library-cover-actions">
      <el-button
        :disabled="disabled"
        :loading="uploading"
        @click="picker?.click()"
        >上传封面</el-button
      >
      <el-button
        :disabled="disabled || uploading"
        @click="selecting = !selecting"
        >使用正文图片</el-button
      >
      <el-button
        v-if="modelValue"
        text
        :disabled="disabled || uploading"
        @click="clear"
        >清除封面</el-button
      >
      <input
        ref="picker"
        type="file"
        accept="image/png,image/jpeg,image/webp"
        hidden
        aria-label="选择封面文件"
        @change="picked"
      />
    </div>
    <div v-if="selecting" class="library-cover-options">
      <p v-if="!bodyImages.length" class="library-muted">
        正文暂无本地图片，可先在正文中上传或粘贴图片。
      </p>
      <button
        v-for="image in bodyImages"
        :key="image.url"
        type="button"
        :disabled="disabled || uploading"
        :aria-label="`使用正文图片：${image.name}`"
        :aria-pressed="modelValue === image.url"
        @click="select(image.url)"
      >
        <img :src="image.url" alt="" loading="lazy" /><span>{{
          image.name
        }}</span>
      </button>
    </div>
    <el-input
      :model-value="modelValue"
      :disabled="disabled || uploading"
      aria-label="页面封面地址"
      placeholder="本地图片地址或 https://…（可选）"
      @update:model-value="$emit('update:modelValue', String($event).trim())"
    />
    <p v-if="failure" class="library-cover-error" role="alert">{{ failure }}</p>
    <div v-if="validCover" class="library-cover-preview">
      <img
        v-show="!previewFailed"
        :src="modelValue"
        alt="页面封面预览"
        referrerpolicy="no-referrer"
        @error="previewFailed = true"
        @load="previewFailed = false"
      />
      <p v-if="previewFailed" class="library-muted">
        封面暂时无法加载，请检查地址或登录状态。
      </p>
    </div>
    <p class="library-muted">
      PNG / JPEG / WebP，单张 10MB。保存页面后生效；移除或更换封面不会删除原图。
    </p>
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { uploadLibraryImage } from "@/api/library";
import { localLibraryImage, markdownHTML } from "@/utils/libraryMarkdown";
import { safeExternalLink } from "@/utils/externalLinks";
const props = defineProps<{
  modelValue: string;
  body: string;
  collection: number;
  disabled?: boolean;
}>();
const emit = defineEmits<{
  (e: "update:modelValue", value: string): void;
  (e: "uploading", value: boolean): void;
}>();
const picker = ref<HTMLInputElement>(),
  selecting = ref(false),
  uploading = ref(false),
  failure = ref(""),
  previewFailed = ref(false);
let disposed = false,
  controller: AbortController | undefined;
const validCover = computed(
  () =>
    localLibraryImage(props.modelValue) ||
    (/^https?:\/\//i.test(props.modelValue) &&
      safeExternalLink(props.modelValue)),
);
const bodyImages = computed(() => {
  // Reuse the safe Markdown renderer: reference-style images work and code
  // examples / unrelated URLs never become selectable image references.
  const div = document.createElement("div");
  div.innerHTML = markdownHTML(props.body, props.collection);
  const images = new Map<string, { url: string; name: string }>();
  div.querySelectorAll("img").forEach((img) => {
    const url = img.getAttribute("src") || "";
    if (localLibraryImage(url) && !images.has(url))
      images.set(url, {
        url,
        name: img.getAttribute("alt") || `图片 ${images.size + 1}`,
      });
  });
  return [...images.values()];
});
watch(
  () => props.modelValue,
  () => {
    previewFailed.value = false;
  },
);
function select(url: string) {
  emit("update:modelValue", url);
  selecting.value = false;
  failure.value = "";
}
function clear() {
  emit("update:modelValue", "");
  failure.value = "";
}
async function picked(event: Event) {
  const input = event.target as HTMLInputElement,
    file = input.files?.[0];
  input.value = "";
  if (!file || uploading.value || props.disabled) return;
  failure.value = "";
  if (
    file.type &&
    !["image/png", "image/jpeg", "image/webp"].includes(file.type)
  ) {
    failure.value = "仅支持 PNG、JPEG、WebP 封面。";
    return;
  }
  if (file.size > 10 * 1024 * 1024) {
    failure.value = "封面不能超过 10MB。";
    return;
  }
  uploading.value = true;
  selecting.value = false;
  emit("uploading", true);
  controller = new AbortController();
  try {
    const image = await uploadLibraryImage(file, controller.signal);
    if (!disposed) emit("update:modelValue", image.url);
  } catch (error) {
    if (!disposed)
      failure.value = `${error instanceof Error ? error.message : "封面上传失败。"} 原有封面已保留，可重新选择文件重试。`;
  } finally {
    uploading.value = false;
    if (!disposed) emit("uploading", false);
  }
}
onBeforeUnmount(() => {
  disposed = true;
  controller?.abort();
  emit("uploading", false);
});
</script>
<style scoped>
.library-cover-editor {
  width: 100%;
}
.library-cover-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 10px;
}
.library-cover-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}
.library-cover-options {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 10px;
}
.library-cover-options button {
  display: grid;
  gap: 6px;
  width: 120px;
  padding: 8px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-control);
  background: var(--bg-surface);
  color: var(--text-primary);
  cursor: pointer;
  overflow-wrap: anywhere;
}
.library-cover-options button[aria-pressed="true"] {
  border-color: var(--color-primary);
}
.library-cover-options img {
  width: 100%;
  height: 72px;
  object-fit: contain;
}
.library-cover-preview {
  margin-top: 12px;
}
.library-cover-preview img {
  display: block;
  width: auto;
  height: auto;
  max-width: 100%;
  max-height: 240px;
  object-fit: contain;
}
.library-cover-error {
  color: var(--color-danger);
  font-size: 13px;
}
</style>
