<template>
  <div class="library-image-editor">
    <div class="library-image-tools">
      <el-button :loading="uploading" :disabled="disabled" @click="choose"
        >上传图片</el-button
      >
      <input
        ref="picker"
        type="file"
        accept="image/png,image/jpeg,image/webp"
        hidden
        @change="picked"
      />
      <span class="library-muted"
        >可粘贴截图或拖入图片 · PNG / JPEG / WebP · 单张 10MB</span
      >
    </div>
    <el-input
      ref="editor"
      :model-value="modelValue"
      type="textarea"
      :rows="18"
      maxlength="200000"
      aria-label="Markdown 正文"
      class="library-body-input"
      :disabled="disabled"
      @update:model-value="$emit('update:modelValue', String($event))"
      @paste="paste"
      @dragover="dragOver"
      @drop="drop"
    />
    <div v-if="uploading" class="library-image-status" role="status">
      图片上传中，正文仍可编辑；上传成功后插入引用。
    </div>
    <div v-if="failure" class="library-image-status" role="alert">
      <span>{{ failure }}</span
      ><el-button text :disabled="uploading || disabled" @click="retry"
        >重试上传</el-button
      >
    </div>
    <p class="library-muted">
      上传不等于保存页面。取消编辑后的未引用图片会保留至少 7
      天，仅在手动清理时删除。
    </p>
  </div>
</template>
<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from "vue";
import { uploadLibraryImage } from "@/api/library";
const props = defineProps<{ modelValue: string; disabled?: boolean }>();
const emit = defineEmits<{
  (e: "update:modelValue", value: string): void;
  (e: "uploading", value: boolean): void;
}>();
const editor = ref<{ textarea?: HTMLTextAreaElement }>(),
  picker = ref<HTMLInputElement>();
const uploading = ref(false),
  failure = ref("");
let pending: File | null = null,
  bookmark = 0,
  disposed = false,
  controller: AbortController | undefined;
watch(
  () => props.modelValue,
  (value, previous) => {
    if (!pending) return;
    let prefix = 0,
      suffix = 0;
    while (
      prefix < previous.length &&
      prefix < value.length &&
      previous[prefix] === value[prefix]
    )
      prefix++;
    while (
      suffix < previous.length - prefix &&
      suffix < value.length - prefix &&
      previous[previous.length - 1 - suffix] ===
        value[value.length - 1 - suffix]
    )
      suffix++;
    const oldEnd = previous.length - suffix,
      newEnd = value.length - suffix;
    if (bookmark >= oldEnd) bookmark += newEnd - oldEnd;
    else if (bookmark > prefix) bookmark = newEnd;
    bookmark = Math.max(0, Math.min(bookmark, value.length));
  },
  { flush: "sync" },
);
function caret() {
  return editor.value?.textarea?.selectionStart ?? props.modelValue.length;
}
function choose() {
  if (uploading.value || props.disabled) return;
  bookmark = caret();
  picker.value?.click();
}
function picked(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (file) void begin(file, bookmark);
}
function paste(event: ClipboardEvent) {
  const file = Array.from(event.clipboardData?.items ?? [])
    .find((item) => item.kind === "file" && item.type.startsWith("image/"))
    ?.getAsFile();
  if (!file) return; // ordinary text retains native paste behavior
  event.preventDefault();
  if (!uploading.value && !props.disabled) void begin(file, caret());
}
function dragOver(event: DragEvent) {
  if (event.dataTransfer?.types.includes("Files")) event.preventDefault();
}
function drop(event: DragEvent) {
  if (!event.dataTransfer?.files.length) return;
  event.preventDefault();
  if (uploading.value || props.disabled) return;
  const file = event.dataTransfer.files[0];
  if (file) void begin(file, caret());
}
async function begin(file: File, position: number) {
  if (uploading.value || props.disabled) return;
  pending = file;
  bookmark = position;
  failure.value = "";
  if (!["image/png", "image/jpeg", "image/webp"].includes(file.type)) {
    failure.value = "仅支持 PNG、JPEG、WebP 图片。";
    return;
  }
  if (file.size > 10 * 1024 * 1024) {
    failure.value = "单张图片不能超过 10MB。";
    return;
  }
  uploading.value = true;
  emit("uploading", true);
  controller = new AbortController();
  try {
    const image = await uploadLibraryImage(file, controller.signal);
    if (disposed) return;
    const alt =
      file.name.replace(/[\[\]\\()\r\n]/g, " ").slice(0, 120) || "图片";
    const markdown = `\n![${alt}](${image.url})\n`;
    const pos = Math.min(bookmark, props.modelValue.length);
    pending = null;
    emit(
      "update:modelValue",
      props.modelValue.slice(0, pos) + markdown + props.modelValue.slice(pos),
    );
    await nextTick();
    editor.value?.textarea?.focus();
    editor.value?.textarea?.setSelectionRange(
      pos + markdown.length,
      pos + markdown.length,
    );
  } catch (error) {
    if (!disposed)
      failure.value =
        error instanceof Error ? error.message : "图片上传失败，正文未改变。";
  } finally {
    uploading.value = false;
    if (!disposed) emit("uploading", false);
  }
}
function retry() {
  if (pending) void begin(pending, bookmark);
}
onBeforeUnmount(() => {
  disposed = true;
  controller?.abort();
  emit("uploading", false);
});
</script>
<style scoped>
.library-image-tools,
.library-image-status {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin: 10px 0;
}
.library-image-status {
  font-size: 13px;
  color: var(--text-secondary);
}
.library-image-status[role="alert"] {
  color: var(--color-danger);
}
</style>
