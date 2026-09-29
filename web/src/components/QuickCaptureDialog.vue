<template>
  <el-dialog v-model="open" title="快速记录" width="min(520px, calc(100vw - 28px))" class="quick-capture-dialog" @opened="focusInput">
    <p class="capture-hint">先记下来，稍后再整理。</p>
    <el-input ref="captureInput" v-model="content" type="textarea" :rows="5" maxlength="10000" show-word-limit placeholder="想法、待办或链接……" @keydown.ctrl.enter.prevent="submit" @keydown.meta.enter.prevent="submit" />
    <template #footer><el-button @click="open = false">取消</el-button><el-button type="primary" :loading="saving" :disabled="!content.trim()" @click="submit">保存到收集箱</el-button></template>
  </el-dialog>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { createInboxItem } from '@/api/inbox'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; created: [] }>()
const open = ref(props.modelValue)
const saving = ref(false)
const content = ref('')
const captureInput = ref<{ focus: () => void } | null>(null)
watch(() => props.modelValue, value => { open.value = value })
watch(open, value => emit('update:modelValue', value))
function focusInput() { void nextTick(() => captureInput.value?.focus()) }
async function submit() {
  if (saving.value || !content.value.trim()) return
  saving.value = true
  try {
    await createInboxItem(content.value)
    content.value = ''
    open.value = false
    emit('created')
    ElMessage.success('已保存到收集箱')
  } catch (reason) {
    ElMessage.error(reason instanceof Error ? reason.message : '保存失败，请重试')
  } finally { saving.value = false }
}
</script>

<style scoped>
.capture-hint { margin: 0 0 12px; color: var(--text-secondary); font-size: 14px; }
</style>
