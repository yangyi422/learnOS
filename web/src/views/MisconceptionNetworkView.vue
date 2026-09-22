<template>
  <section class="learning-page">
    <PageHeader eyebrow="学习模式" title="误区网络" description="查看影响学习判断的底层思维模式与具体误区。">
      <template #actions><el-button text @click="router.push(`/courses/${courseID}/learn`)">返回学习</el-button><el-button text :loading="loading" @click="loadNetwork">刷新网络</el-button></template>
    </PageHeader>
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <div v-loading="loading" class="misconception-network">
      <el-card shadow="never">
        <template #header>底层思维模式</template>
        <el-empty v-if="network && network.patterns.length === 0" description="尚无经过重复证据或用户确认的稳定模式。一次 AI 评价只会记录为推测。"><el-button type="primary" plain @click="router.push(`/courses/${courseID}/learn`)">去完成一次回答</el-button></el-empty>
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
        <el-empty v-if="network && network.misconceptions.length === 0" description="完成回答后，AI 可能提出误区线索；你可以确认、纠正或忽略。"><el-button type="primary" plain @click="router.push(`/courses/${courseID}/learn`)">去完成一次回答</el-button></el-empty>
        <div v-for="item in network?.misconceptions" :key="item.id" class="misconception-network-item">
          <div class="card-header-row">
            <strong>{{ item.lesson_title }}</strong>
            <div><el-tag :type="reviewTagType(item.review_status)">{{ reviewText(item.review_status, item.occurrence_count) }}</el-tag> <el-tag v-if="item.stable_pattern_evidence" type="warning" effect="plain">重复证据</el-tag> <el-tag v-if="item.status === 'resolved'" type="success">已修正</el-tag></div>
          </div>
          <p><strong>原理解：</strong>{{ item.original_understanding }}</p>
          <p><strong>正确理解：</strong>{{ item.correct_understanding }}</p>
          <p v-if="item.boundary_notes"><strong>边界：</strong>{{ item.boundary_notes }}</p>
          <p class="muted-text">AI 观察 {{ item.occurrence_count }} 次 · 模式线索：{{ item.pattern_keys.join('、') || '未标注' }}</p>
          <p v-if="!item.stable_pattern_evidence" class="muted-text">当前证据不足以认定为稳定的底层思维模式。</p>
          <div class="event-list">
            <div v-for="event in item.events" :key="event.id" class="misconception-event">
              <span>{{ eventText(event.event_type) }} · {{ formatTime(event.created_at) }}：{{ event.notes }}</span>
              <blockquote v-if="event.user_answer">依据回答：{{ event.user_answer }}</blockquote>
            </div>
          </div>
          <div v-if="item.review_status === 'ai_inferred'" class="exploration-actions">
            <el-button size="small" type="primary" plain @click="review(item.id, 'confirm')">确认推测</el-button>
            <el-button size="small" @click="correct(item.id)">纠正推测</el-button>
            <el-button size="small" text @click="review(item.id, 'ignore')">忽略</el-button>
          </div>
          <el-button v-if="item.status === 'active'" size="small" plain @click="router.push(`/courses/${courseID}/learn`)" title="回到当前课程节点进行针对性重新测试">去验证修正</el-button>
        </div>
      </el-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import PageHeader from '@/components/PageHeader.vue'
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getMisconceptionNetwork, reviewMisconception } from '@/api/misconceptions'
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
  return { observed: 'AI 推测', resolved: '挑战验证已修正', reopened: '再次出现', user_confirmed: '用户确认', user_corrected: '用户纠正', ignored: '用户忽略' }[type] ?? type
}

function reviewText(status: string, occurrences: number): string {
  if (status === 'user_confirmed') return '用户确认'
  if (status === 'user_corrected') return '用户已纠正'
  if (status === 'ignored') return '已忽略'
  return occurrences >= 2 ? 'AI 推测 · 重复出现' : 'AI 推测'
}

function reviewTagType(status: string): 'success' | 'warning' | 'info' {
  return status === 'user_confirmed' ? 'success' : status === 'ai_inferred' ? 'warning' : 'info'
}

async function review(id: number, action: 'confirm' | 'correct' | 'ignore', note = '') {
  try {
    await reviewMisconception(courseID, id, action, note)
    await loadNetwork()
  } catch (reason) { error.value = reason instanceof Error ? reason.message : '误区审阅失败' }
}

function correct(id: number) {
  const note = window.prompt('请写下你认为更准确的理解或需要纠正之处')?.trim()
  if (note) void review(id, 'correct', note)
}

function formatTime(value: string): string { return new Date(value).toLocaleString('zh-CN') }

onMounted(loadNetwork)
</script>
