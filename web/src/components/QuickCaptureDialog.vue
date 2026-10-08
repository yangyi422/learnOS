<template>
  <el-dialog v-model="open" title="快速记录" width="min(520px, calc(100vw - 28px))" class="quick-capture-dialog" :before-close="close" @opened="capture?.focus()">
    <p class="capture-hint">Enter 保存，Shift+Enter 换行。</p>
    <InboxCaptureForm ref="capture" success-message="已保存到收集箱" @created="created" />
    <template #footer><el-button @click="close(() => open = false)">取消</el-button></template>
  </el-dialog>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue'
import InboxCaptureForm from '@/components/InboxCaptureForm.vue'
const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; created: [] }>()
const open = computed({ get: () => props.modelValue, set: value => emit('update:modelValue', value) })
const capture = ref<InstanceType<typeof InboxCaptureForm> | null>(null)
function close(done: () => void) { if (!capture.value?.isSaving()) done() }
function created() { open.value = false; emit('created') }
</script>
<style scoped>.capture-hint { margin: 0 0 12px; color: var(--text-secondary); font-size: 13px; }</style>
