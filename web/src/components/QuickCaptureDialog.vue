<template>
  <el-dialog v-model="open" title="快速记录" width="min(520px, calc(100vw - 28px))" class="quick-capture-dialog" :before-close="close" @opened="capture?.focus()">
    <p class="capture-hint">Enter 保存，Shift+Enter 换行。</p>
    <InboxCaptureForm ref="capture" success-message="已保存到收集箱" @created="created" />
    <template #footer><el-button @click="close(() => open = false)">取消</el-button></template>
  </el-dialog>
</template>
<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import type { InboxItem } from '@/api/inbox'
import InboxCaptureForm from '@/components/InboxCaptureForm.vue'
const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; created: [item: InboxItem] }>()
const open = computed({ get: () => props.modelValue, set: value => emit('update:modelValue', value) })
const capture = ref<InstanceType<typeof InboxCaptureForm> | null>(null)
async function canDiscard() {
  if (capture.value?.isSaving()) return false
  if (capture.value?.hasDraft()) {
    try { await ElMessageBox.confirm('这条记录尚未保存，确定放弃吗？', '放弃未保存的记录？', { confirmButtonText: '放弃记录', cancelButtonText: '继续编辑', type: 'warning' }) }
    catch { return false }
  }
  capture.value?.discard()
  return true
}
async function close(done: () => void) { if (await canDiscard()) done() }
function created(item: InboxItem) { open.value = false; emit('created', item) }
onBeforeRouteLeave(async () => !open.value || await canDiscard())
function beforeUnload(event: BeforeUnloadEvent) {
  if (open.value && (capture.value?.hasDraft() || capture.value?.isSaving())) {
    event.preventDefault()
    event.returnValue = ''
  }
}
onMounted(() => window.addEventListener('beforeunload', beforeUnload))
onBeforeUnmount(() => window.removeEventListener('beforeunload', beforeUnload))
</script>
<style scoped>.capture-hint { margin: 0 0 12px; color: var(--text-secondary); font-size: 13px; }</style>
