<template>
  <section class="page-stack curriculum-page coverage-page">
    <PageHeader eyebrow="知识盘点" title="课程档案与覆盖度" description="盘点课程蓝图已经覆盖什么，以及下一步应该补齐哪里。">
      <template #actions><el-button text @click="router.push(`/courses/${courseID}/learn`)">返回学习</el-button><el-button type="primary" @click="router.push(`/courses/${courseID}/map`)">查看知识结构 →</el-button></template>
    </PageHeader>

    <AIRequestError v-if="draftError" :message="draftError.message" :retryable="draftError.retryable" :retry="retryDraft" />
    <CoverageSummary v-if="coverage" :metrics="coverage.metrics" />

    <nav class="coverage-tabs" aria-label="课程档案视图">
      <button type="button" :class="{ 'is-active': activeTab === 'coverage' }" @click="activeTab = 'coverage'">课程覆盖 <span v-if="coverage">{{ coverage.units.length }}</span></button>
      <button type="button" :class="{ 'is-active': activeTab === 'drafts' }" @click="activeTab = 'drafts'">课程扩充草案 <span>{{ drafts.length }}</span></button>
    </nav>

    <section v-if="activeTab === 'coverage'" class="coverage-content">
      <SectionHeader title="知识区域" description="每个区域代表课程蓝图中的一组知识，节点状态只表示课程内容是否已经映射。" />
      <div v-if="coverage" class="coverage-unit-list"><CoverageUnitSection v-for="unit in coverage.units" :key="unit.unit.id" :unit="unit" /></div>
      <EmptyState v-else title="正在读取课程蓝图" description="课程覆盖信息加载中。" />
    </section>

    <section v-else class="coverage-content">
      <SectionHeader title="课程扩充草案" description="AI 只能生成待审核草案；确认应用后才会新增正式课程节点。">
        <template #actions><el-button type="primary" :loading="generating" :disabled="!coverage?.metrics.core_missing" @click="generateDraft">生成核心缺口草案 →</el-button></template>
      </SectionHeader>
      <EmptyState v-if="!drafts.length" title="暂无课程扩充草案" description="当课程蓝图出现核心缺口时，可以在这里生成待审核草案。" />
      <div v-else class="curriculum-draft-list"><CurriculumDraftRow v-for="item in drafts" :key="item.draft.id" :item="item" @reject="rejectDraft" @apply="applyDraft" /></div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { applyCurriculumDraft, createCurriculumDraft, getCurriculumCoverage, listCurriculumDrafts, rejectCurriculumDraft } from '@/api/curriculum'
import { APIRequestError } from '@/api/http'
import AIRequestError from '@/components/AIRequestError.vue'
import PageHeader from '@/components/PageHeader.vue'
import SectionHeader from '@/components/SectionHeader.vue'
import CoverageSummary from '@/components/CoverageSummary.vue'
import CoverageUnitSection from '@/components/CoverageUnitSection.vue'
import CurriculumDraftRow from '@/components/CurriculumDraftRow.vue'
import EmptyState from '@/components/EmptyState.vue'
import type { CurriculumCoverageView, CurriculumDraftView } from '@/types/curriculum'
import { invalidateExplorationRadar } from '@/api/exploration'

const route = useRoute()
const router = useRouter()
const courseID = Number(String(route.params.id))
const returnToMap = route.query.return_to === 'map'
const activeTab = ref(route.query.tab === 'drafts' ? 'drafts' : 'coverage')
const coverage = ref<CurriculumCoverageView | null>(null)
const drafts = ref<CurriculumDraftView[]>([])
const generating = ref(false)
const draftError = ref<APIRequestError | null>(null)
const retryDraft = () => { void generateDraft() }
let generationPollTimer: number | null = null

function scheduleGenerationPoll() {
  if (generationPollTimer !== null) window.clearTimeout(generationPollTimer)
  if (!drafts.value.some((item) => item.draft.status === 'generating')) return
  generationPollTimer = window.setTimeout(async () => {
    generationPollTimer = null
    await refreshDrafts()
  }, 1500)
}

async function refreshDrafts() {
  try {
    drafts.value = await listCurriculumDrafts(courseID)
    scheduleGenerationPoll()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '加载课程草案失败')
  }
}

async function load() {
  try {
    const [nextCoverage, nextDrafts] = await Promise.all([getCurriculumCoverage(courseID), listCurriculumDrafts(courseID)])
    coverage.value = nextCoverage
    drafts.value = nextDrafts
    scheduleGenerationPoll()
  } catch (error) { ElMessage.error(error instanceof Error ? error.message : '加载课程覆盖度失败') }
}
async function generateDraft() { if (generating.value) return; generating.value = true; draftError.value = null; try { drafts.value.unshift(await createCurriculumDraft(courseID)); activeTab.value = 'drafts'; ElMessage.success('草案已生成，请审核后确认应用') } catch (error) { if (error instanceof APIRequestError) { draftError.value = error; if (error.code === 'CURRICULUM_DRAFT_GENERATION_IN_PROGRESS') { activeTab.value = 'drafts'; await refreshDrafts() } } else ElMessage.error(error instanceof Error ? error.message : '生成草案失败') } finally { generating.value = false } }
async function applyDraft(id: number) { try { await ElMessageBox.confirm('应用草案只会新增课程节点，不会改变当前课程节点、学习历史或个人认知状态。继续？', '确认课程扩充'); await applyCurriculumDraft(courseID, id); invalidateExplorationRadar(); ElMessage.success('草案已应用'); if (returnToMap) await router.push(`/courses/${courseID}/map`); else await load() } catch (error) { if (error !== 'cancel') ElMessage.error(error instanceof Error ? error.message : '应用失败') } }
async function rejectDraft(id: number) { try { await rejectCurriculumDraft(courseID, id); ElMessage.success('草案已拒绝'); await load() } catch (error) { ElMessage.error(error instanceof Error ? error.message : '拒绝失败') } }
onMounted(load)
onBeforeUnmount(() => { if (generationPollTimer !== null) window.clearTimeout(generationPollTimer) })
</script>

<style scoped>
.curriculum-page { max-width: 1180px; margin: 0 auto; }
</style>
