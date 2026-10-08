<template>
  <el-button text :loading="loading" @click.stop="open">收录到生活档案</el-button>
  <LifeEventDialog v-model="dialogOpen" :source="source" @saved="source = null" />
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import LifeEventDialog from './LifeEventDialog.vue'
import { getLifeSource, type LifeSource } from '@/api/life'
import '@/life.css'
const props = defineProps<{ sourceType: 'project' | 'course'; sourceId: number }>()
const source = ref<LifeSource | null>(null), dialogOpen = ref(false), loading = ref(false)
const router = useRouter()
async function open() { if (loading.value) return; loading.value = true; try { const info = await getLifeSource(props.sourceType, props.sourceId); if (info.event_id) { await router.push({ path: '/life', query: { event: info.event_id } }); return }; if (!info.eligible) { ElMessage.info('请先正式完成项目或结业课程，再收录成果。'); return }; source.value = info; dialogOpen.value = true } catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : '来源读取失败') } finally { loading.value = false } }
</script>
