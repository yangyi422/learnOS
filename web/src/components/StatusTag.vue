<template>
  <el-tag :type="tagType" :effect="effect" :size="size" class="status-tag" :class="`status-tag--${tone}`">
    {{ label || statusLabel }}
  </el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  status: string
  label?: string
  size?: 'large' | 'default' | 'small'
  effect?: 'dark' | 'light' | 'plain'
}>(), { size: 'small', effect: 'light' })

const labels: Record<string, string> = {
  initializing: '初始化中', learning: '学习中', paused: '已暂停', completed: '已完成',
  unseen: '未接触', exposed: '已接触', recognize: '识别', understand: '理解', apply: '应用', transfer: '迁移',
  unknown: '未知', developing: '发展中', stable: '稳定', needs_review: '待复习',
  active: '活跃', saved: '已保存', dismissed: '已略过', opened: '已打开',
  generating: '生成中', draft: '草案', applied: '已应用', rejected: '已拒绝', configured: '已配置',
}

const statusLabel = computed(() => labels[props.status] ?? props.status)
const tagType = computed<'primary' | 'success' | 'warning' | 'danger' | 'info'>(() => {
  if (['stable', 'completed', 'applied', 'saved', 'configured', 'apply', 'transfer'].includes(props.status)) return 'success'
  if (['initializing', 'generating', 'developing', 'needs_review', 'active', 'draft'].includes(props.status)) return 'warning'
  if (['dismissed'].includes(props.status)) return 'info'
  if (['error', 'conflicted', 'failed', 'rejected'].includes(props.status)) return 'danger'
  if (['unseen', 'unknown', 'paused'].includes(props.status)) return 'info'
  return 'primary'
})
const tone = computed(() => {
  if (['stable', 'completed', 'applied', 'saved', 'configured', 'apply', 'transfer'].includes(props.status)) return 'success'
  if (['initializing', 'generating', 'developing', 'needs_review', 'draft'].includes(props.status)) return 'warning'
  if (['exploration', 'active', 'cross_domain', 'adjacent', 'unfamiliar'].includes(props.status)) return 'exploration'
  if (['unseen', 'unknown', 'paused', 'dismissed'].includes(props.status)) return 'neutral'
  if (['error', 'conflicted', 'failed', 'rejected'].includes(props.status)) return 'danger'
  return 'primary'
})
</script>
