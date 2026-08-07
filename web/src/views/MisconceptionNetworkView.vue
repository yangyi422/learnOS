<template>
  <section class="learning-page">
    <div class="learning-toolbar">
      <el-button text @click="router.push(`/courses/${courseID}/learn`)">← 返回学习</el-button>
      <el-button text :loading="loading" @click="loadNetwork">刷新网络</el-button>
    </div>
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <div v-loading="loading" class="misconception-network">
      <el-card shadow="never">
        <template #header>底层思维模式</template>
        <el-empty v-if="network && network.patterns.length === 0" description="还没有形成误区网络" />
        <div v-else class="pattern-grid">
          <div v-for="pattern in network?.patterns" :key="pattern.key" class="pattern-card">
            <strong>{{ pattern.name }}</strong>
            <small>{{ pattern.key }}</small>
            <span>活跃 {{ pattern.active_count }} · 已修正 {{ pattern.resolved_count }}</span>
          </div>
        </div>
      </el-card>
      <el-card shadow="never">
        <template #header>具体误区与事件</template>
        <el-empty v-if="network && network.misconceptions.length === 0" description="还没有记录误区" />
        <div v-for="item in network?.misconceptions" :key="item.id" class="misconception-network-item">
          <div class="card-header-row">
            <strong>{{ item.lesson_title }}</strong>
            <el-tag :type="item.status === 'active' ? 'warning' : 'success'">{{ item.status === 'active' ? '活跃' : '已修正' }}</el-tag>
          </div>
          <p><strong>原理解：</strong>{{ item.original_understanding }}</p>
          <p><strong>正确理解：</strong>{{ item.correct_understanding }}</p>
          <p v-if="item.boundary_notes"><strong>边界：</strong>{{ item.boundary_notes }}</p>
          <p class="muted-text">出现 {{ item.occurrence_count }} 次 · 模式：{{ item.pattern_keys.join('、') || '未标注' }}</p>
          <div class="event-list">
            <span v-for="event in item.events" :key="event.id">{{ eventText(event.event_type) }}：{{ event.notes }}</span>
          </div>
          <el-button v-if="item.status === 'active'" size="small" plain @click="router.push(`/courses/${courseID}/learn`)" title="回到当前 Lesson 进行针对性重新测试">去验证修正</el-button>
        </div>
      </el-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getMisconceptionNetwork } from '@/api/misconceptions'
import type { MisconceptionNetwork } from '@/types/misconception'

const route = useRoute()
const router = useRouter()
const courseID = Number(String(route.params.id))
const network = ref<MisconceptionNetwork | null>(null)
const loading = ref(false)
const error = ref('')

async function loadNetwork() {
  loading.value = true
  error.value = ''
  try {
    network.value = await getMisconceptionNetwork(courseID)
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '误区网络读取失败'
  } finally {
    loading.value = false
  }
}

function eventText(type: string): string {
  return { observed: '发现', resolved: '修正', reopened: '重新出现' }[type] ?? type
}

onMounted(loadNetwork)
</script>
