<template>
  <form class="inbox-capture" @submit.prevent="submit">
    <div class="capture-input">
      <el-input ref="input" v-model="content" type="textarea" :autosize="{ minRows: 1, maxRows: 5 }" :disabled="saving" maxlength="10000" aria-label="快速记录到收集箱" placeholder="想法、待办或链接……" @keydown="keydown" @input="error = ''; captureKey = ''" />
      <p v-if="error" class="capture-error" role="alert">{{ error }}</p>
    </div>
    <el-button type="primary" native-type="submit" :loading="saving" :disabled="!content.trim()" aria-label="记录到收集箱">{{ buttonText || '保存到收集箱' }}</el-button>
  </form>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { createInboxItem, type InboxItem } from '@/api/inbox'
const props = defineProps<{ buttonText?: string; successMessage?: string }>()
const emit = defineEmits<{ created: [item: InboxItem] }>()
const captureKey = ref('')
const content = ref(''); const saving = ref(false); const error = ref('')
const input = ref<{ focus: () => void } | null>(null)
function keydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' || event.shiftKey || event.isComposing || event.keyCode === 229) return
  event.preventDefault(); void submit()
}
async function submit() {
  if (saving.value || !content.value.trim()) return
  saving.value = true; error.value = ''
  captureKey.value ||= crypto.randomUUID?.() || `capture-${Date.now()}-${Math.random().toString(36).slice(2)}`
  try { const item = await createInboxItem(content.value, captureKey.value); content.value = ''; captureKey.value = ''; emit('created', item); ElMessage.success(props.successMessage || '已记录') }
  catch (reason) { error.value = reason instanceof Error ? reason.message : '保存失败，请重试。' }
  finally { saving.value = false }
}
defineExpose({ focus: () => input.value?.focus(), isSaving: () => saving.value })
</script>
<style scoped>
.inbox-capture { display: flex; gap: 10px; align-items: flex-start; margin-bottom: 18px; }
.capture-input { flex: 1; min-width: 0; }
.inbox-capture :deep(.el-textarea__inner) { min-height: 42px !important; resize: none; border-radius: var(--radius-control); background: var(--bg-surface); color: var(--text-primary); }
.el-button { min-height: 42px; }
.capture-error { margin: 7px 0 0; color: var(--color-danger); font-size: 13px; }
@media(max-width:560px) { .inbox-capture { flex-wrap: wrap; } .capture-input { flex-basis: 100%; } .el-button { margin-left: auto; } }
</style>
