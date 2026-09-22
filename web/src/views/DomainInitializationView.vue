<template>
  <section class="page-stack domain-init-page">
    <PageHeader eyebrow="新建学习领域" title="创建学习领域" description="LearnOS 会先建立领域地图，再逐步展开起步区域和第一批课程。">
      <template #actions><el-button text @click="router.push('/')">返回首页</el-button></template>
    </PageHeader>
    <AIRequestError v-if="requestError && requestError.code !== 'DOMAIN_ALREADY_EXISTS'" :message="requestError.message" :retryable="requestError.retryable" :retry="retryAction" />
    <el-card v-if="requestError?.code === 'DOMAIN_ALREADY_EXISTS'" shadow="never" class="duplicate-domain-card">
      <h3>这个学习领域已经存在。</h3>
      <p class="muted">请选择已有学习领域，或返回课程档案创建其他领域。</p>
      <div class="actions">
        <el-button type="primary" @click="enterExistingCourse">进入课程</el-button>
        <el-button @click="router.push('/courses')">返回课程档案</el-button>
      </div>
    </el-card>
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />

    <el-card v-if="!view" shadow="never" class="form-card">
      <el-form label-position="top" @submit.prevent="create">
        <el-form-item label="领域名称" required>
          <el-input
            v-model="form.domain_name"
            maxlength="50"
            show-word-limit
            placeholder="例如：营养学、摄影、统计学"
            :aria-invalid="Boolean(fieldErrors.domain_name)"
            :aria-describedby="fieldErrors.domain_name ? 'domain-name-error' : undefined"
            @blur="validateField('domain_name')"
            @input="validateIfTouched('domain_name')"
          />
          <p v-if="fieldErrors.domain_name" id="domain-name-error" class="field-error" role="alert">{{ fieldErrors.domain_name }}</p>
        </el-form-item>
        <el-form-item label="为什么想学" required>
          <el-input
            v-model="form.learning_goal"
            type="textarea"
            :rows="4"
            maxlength="4000"
            show-word-limit
            placeholder="例如：改善日常饮食，并系统理解营养学基础"
            :aria-invalid="Boolean(fieldErrors.learning_goal)"
            :aria-describedby="fieldErrors.learning_goal ? 'learning-goal-error' : undefined"
            @blur="validateField('learning_goal')"
            @input="validateIfTouched('learning_goal')"
          />
          <p v-if="fieldErrors.learning_goal" id="learning-goal-error" class="field-error" role="alert">{{ fieldErrors.learning_goal }}</p>
        </el-form-item>
        <el-form-item label="期望深度" required>
          <el-radio-group v-model="form.target_depth" :aria-invalid="Boolean(fieldErrors.target_depth)" :aria-describedby="fieldErrors.target_depth ? 'target-depth-error' : undefined" @change="validateField('target_depth')">
            <el-radio-button label="overview">快速了解</el-radio-button>
            <el-radio-button label="foundation">基础入门</el-radio-button>
            <el-radio-button label="systematic">系统学习</el-radio-button>
          </el-radio-group>
          <p v-if="fieldErrors.target_depth" id="target-depth-error" class="field-error" role="alert">{{ fieldErrors.target_depth }}</p>
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" :disabled="loading">
          <span aria-live="polite">{{ loading ? '正在建立领域地图…' : '建立领域地图' }}</span>
        </el-button>
      </el-form>
    </el-card>

    <template v-else>
      <el-card shadow="never"><template #header><div class="card-header-row"><strong>{{ view.skeleton?.course.name || view.draft.domain_name }}</strong><el-tag>{{ statusText(view.draft.status) }}</el-tag></div></template><div class="stage-list"><span :class="stageClass(1)">① 建立领域地图 <b v-if="view.skeleton">✓</b></span><span :class="stageClass(2)">② 展开起步区域 <b v-if="view.starter_blueprint">✓</b></span><span :class="stageClass(3)">③ 准备第一批课程 <b v-if="view.initial_world">✓</b></span></div></el-card>

      <el-card v-if="view.skeleton" shadow="never"><template #header><div class="card-header-row"><span>领域地图</span><el-button size="small" :loading="loading" @click="regenerate">重新生成领域地图</el-button></div></template><p class="muted">{{ view.skeleton.course.description }}</p><div class="unit-grid"><div v-for="unit in view.skeleton.blueprint.units" :key="unit.key" class="unit-item"><strong>{{ unit.title }}</strong><small>{{ unit.description }}</small><el-tag size="small" effect="plain">{{ importanceText(unit.importance) }}</el-tag></div></div><p class="muted">建议从以下区域开始：{{ starterTitles.join('、') }}</p></el-card>

      <el-card v-if="view.skeleton && !view.starter_blueprint" shadow="never"><template #header>选择起步区域</template><el-checkbox-group v-model="selectedUnits"><el-checkbox v-for="unit in view.skeleton.blueprint.units" :key="unit.key" :label="unit.key">{{ unit.title }}</el-checkbox></el-checkbox-group><div class="actions"><el-button type="primary" :loading="loading" :disabled="selectedUnits.length < 1 || selectedUnits.length > 2" @click="expand">继续展开起步区域</el-button></div></el-card>

      <el-card v-if="view.starter_blueprint" shadow="never"><template #header><div class="card-header-row"><span>起步蓝图</span><el-button v-if="!view.initial_world" size="small" :loading="loading" @click="expand">重新展开</el-button></div></template><div class="unit-grid"><div v-for="unit in view.starter_blueprint.expanded_units" :key="unit.key" class="unit-item expanded"><strong>{{ unitTitle(unit.key) }}</strong><ul><li v-for="lesson in unit.lessons" :key="lesson.key">{{ lesson.title }}<small>{{ lesson.summary }}</small></li></ul></div></div><div v-if="!view.initial_world" class="actions"><el-button type="primary" :loading="loading" @click="generateWorld">准备第一批课程</el-button></div></el-card>

      <el-card v-if="view.initial_world" shadow="never"><template #header><div class="card-header-row"><span>第一批课程</span><el-button size="small" :loading="loading" @click="generateWorld">重新生成第一批课程</el-button></div></template><p class="muted">已准备 {{ view.initial_world.initial_lessons.length }} 个正式课程节点草案。创建前仍不会写入正式课程、课程蓝图或个人认知数据。</p><div class="lesson-preview"><article v-for="lesson in view.initial_world.initial_lessons" :key="lesson.blueprint_lesson_key"><el-tag size="small" type="success">{{ lesson.blueprint_lesson_key === view.initial_world.recommended_first_lesson_key ? '建议第一课' : '初始课程节点' }}</el-tag><h3>{{ lesson.title }}</h3><p><strong>核心问题：</strong>{{ lesson.core_question }}</p><p class="muted">{{ lesson.expected_understanding }}</p></article></div><div class="actions"><el-button type="primary" :loading="loading" @click="apply">确认并创建学习领域</el-button><el-button @click="router.push('/')">稍后再说</el-button></div></el-card>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { APIRequestError } from '@/api/http'
import AIRequestError from '@/components/AIRequestError.vue'
import PageHeader from '@/components/PageHeader.vue'
import { listCourses } from '@/api/courses'
import { applyDomainDraft, createDomainDraft, expandStarter, generateInitialWorld, getDomainDraft, regenerateSkeleton } from '@/api/domainInitialization'
import type { Course } from '@/types/course'
import type { DomainInitializationView } from '@/types/domainInitialization'
import { normalizeDomainForm, validateDomainForm, type DomainFormErrors, type DomainFormField } from '@/utils/domainForm'
import { invalidateExplorationRadar } from '@/api/exploration'

const router = useRouter(); const view = ref<DomainInitializationView | null>(null); const loading = ref(false); const error = ref(''); const requestError = ref<APIRequestError | null>(null); const retryAction = ref<(() => void) | undefined>(); const selectedUnits = ref<string[]>([]); const existingCourseID = ref<number | null>(null)
const form = ref({ domain_name: '', learning_goal: '', target_depth: 'systematic' as 'overview' | 'foundation' | 'systematic' })
const fieldErrors = reactive<DomainFormErrors>({})
const touched = reactive<Record<DomainFormField, boolean>>({ domain_name: false, learning_goal: false, target_depth: false })
const starterTitles = computed(() => (view.value?.skeleton?.recommended_starter_unit_keys ?? []).map(unitTitle))
const recoverDraftKey = 'learnos:active-domain-draft'
function unitTitle(key: string) { return view.value?.skeleton?.blueprint.units.find((unit) => unit.key === key)?.title ?? key }
function clearError() { error.value = ''; requestError.value = null; retryAction.value = undefined; existingCourseID.value = null }
function normalizedDomainName(value: string) { return value.trim().toLowerCase() }
async function findExistingCourse() {
  try {
    const courses: Course[] = await listCourses()
    const match = courses.find((course) => normalizedDomainName(course.name) === normalizedDomainName(form.value.domain_name))
    existingCourseID.value = match?.id ?? null
  } catch {
    existingCourseID.value = null
  }
}
function validateField(field: DomainFormField) {
  touched[field] = true
  fieldErrors[field] = validateDomainForm(form.value)[field]
}
function validateIfTouched(field: DomainFormField) {
  if (touched[field]) validateField(field)
}
function validateCreateForm() {
  const errors = validateDomainForm(form.value)
  const fields: DomainFormField[] = ['domain_name', 'learning_goal', 'target_depth']
  for (const field of fields) {
    touched[field] = true
    fieldErrors[field] = errors[field]
  }
  return fields.every((field) => !errors[field])
}
async function run(action: () => Promise<DomainInitializationView>, fallback: string) { if (loading.value) return; loading.value = true; clearError(); try { view.value = await action(); if (view.value.draft.status !== 'applied') window.localStorage.setItem(recoverDraftKey, String(view.value.draft.id)) } catch (reason) { if (reason instanceof APIRequestError) { requestError.value = reason; retryAction.value = () => { void run(action, fallback) }; if (reason.code === 'DOMAIN_ALREADY_EXISTS') await findExistingCourse() } else error.value = reason instanceof Error ? reason.message : fallback } finally { loading.value = false } }
async function create() {
  if (loading.value || !validateCreateForm()) return
  const payload = normalizeDomainForm(form.value)
  await run(() => createDomainDraft(payload), '领域地图生成失败')
}
async function regenerate() { await run(() => regenerateSkeleton(view.value!.draft.id), '领域地图生成失败') }
async function expand() { const keys = selectedUnits.value.length ? selectedUnits.value : (view.value?.skeleton?.recommended_starter_unit_keys ?? []); await run(() => expandStarter(view.value!.draft.id, keys), '起步蓝图生成失败') }
async function generateWorld() { await run(() => generateInitialWorld(view.value!.draft.id), '第一批课程生成失败') }
async function apply() { await run(async () => { const result = await applyDomainDraft(view.value!.draft.id); if (result.draft.applied_course_id) { invalidateExplorationRadar(); window.localStorage.removeItem(recoverDraftKey); ElMessage.success(`${result.draft.domain_name}知识世界已建立`); await router.push(`/courses/${result.draft.applied_course_id}/archive`) }; return result }, '创建学习领域失败') }
function enterExistingCourse() { router.push(existingCourseID.value ? `/courses/${existingCourseID.value}/archive` : '/courses') }
function statusText(status: string) { return { skeleton_confirmed: '领域地图已建立', starter_expanded: '起步区域已展开', world_ready: '第一批课程已准备', applied: '已创建' }[status] ?? status }
function importanceText(value: string) { return { core: '核心', recommended: '建议', optional: '可选' }[value] ?? value }
function stageClass(stage: number) { const status = view.value?.draft.status ?? ''; const done = (stage === 1 && view.value?.skeleton) || (stage === 2 && view.value?.starter_blueprint) || (stage === 3 && view.value?.initial_world); return done ? 'done' : statusStage(status) === stage ? 'active' : '' }
function statusStage(status: string) { return status === 'skeleton_confirmed' ? 2 : status === 'starter_expanded' ? 3 : status === 'world_ready' ? 4 : status === 'applied' ? 5 : 1 }
onMounted(async () => {
  const draftID = Number(window.localStorage.getItem(recoverDraftKey))
  if (!Number.isInteger(draftID) || draftID <= 0) return
  try {
    const recovered = await getDomainDraft(draftID)
    if (recovered.draft.status === 'applied' || recovered.draft.status === 'rejected') window.localStorage.removeItem(recoverDraftKey)
    else {
      view.value = recovered
      selectedUnits.value = recovered.skeleton?.recommended_starter_unit_keys ?? []
    }
  } catch {
    window.localStorage.removeItem(recoverDraftKey)
  }
})
</script>

<style scoped>
.domain-init-page { max-width: 980px; margin: 0 auto; }
.form-card { max-width: 680px; margin: 0 auto; }
.duplicate-domain-card { max-width: 680px; margin: 0 auto 18px; }
.duplicate-domain-card h3 { margin-bottom: 8px; }
.stage-list { display: flex; justify-content: space-between; gap: 12px; color: var(--el-text-color-secondary); }
.stage-list span { padding: 10px 14px; border-radius: 8px; background: var(--el-fill-color-light); }
.stage-list .active { color: var(--el-color-primary); font-weight: 600; }
.stage-list .done { color: var(--el-color-success); }
.unit-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 12px; }
.unit-item { display: grid; gap: 6px; padding: 14px; border: 1px solid var(--el-border-color-lighter); border-radius: 8px; }
.unit-item small, .lesson-preview p, .muted { color: var(--el-text-color-secondary); }
.unit-item ul { margin: 8px 0 0; padding-left: 18px; display: grid; gap: 8px; }
.unit-item li small { display: block; margin-top: 3px; }
.actions { display: flex; gap: 10px; margin-top: 18px; flex-wrap: wrap; }
.lesson-preview { display: grid; gap: 14px; }
.lesson-preview article { padding: 16px; background: var(--el-fill-color-light); border-radius: 8px; }
.field-error { width: 100%; margin: 5px 0 0; color: var(--color-danger); font-size: 12px; line-height: 1.45; }
</style>
