<template>
  <section class="knowledge-graph-page">
    <PageHeader eyebrow="知识世界" :title="graph?.course.name || '知识结构'" description="在知识地图与路径列表之间切换，理解每个学习节点的位置与进展。">
      <template #actions>
        <el-button text @click="router.push(`/courses/${courseID}/learn`)">返回学习</el-button>
        <el-button text :loading="loading" @click="loadGraph">刷新结构</el-button>
      </template>
    </PageHeader>

    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <AIRequestError v-if="aiError" :message="aiError.message" :retryable="aiError.retryable" :retry="retryAction" />

    <div v-loading="loading">
      <section v-if="graph" class="knowledge-summary">
        <div>
          <p class="eyebrow">知识地图</p>
          <p class="graph-note">正式学习节点与待生成节点共同组成这个知识世界。点击节点只查看详情，明确选择学习后才会切换当前课程节点。</p>
        </div>
        <div class="graph-stats">
          <div><strong>{{ graph.stats.node_count }}</strong><span>知识节点</span></div>
          <div><strong>{{ graph.stats.core_node_count }}</strong><span>核心节点</span></div>
          <div><strong>{{ blueprintNodeCount }}</strong><span>待生成</span></div>
          <div><strong>{{ graph.stats.edge_count }}</strong><span>结构关系</span></div>
        </div>
      </section>

      <section v-if="graph" class="knowledge-map-toolbar">
        <div class="knowledge-map-toolbar__views" role="tablist" aria-label="知识结构视图">
          <button type="button" :class="{ 'is-active': activeView === 'path' }" @click="setView('path')">路径视图</button>
          <button type="button" :class="{ 'is-active': activeView === 'list' }" @click="setView('list')">列表视图</button>
        </div>
        <div class="knowledge-map-toolbar__filters">
          <el-select v-model="selectedUnitKey" size="small" aria-label="定位知识区域" style="width: 150px" @change="scrollToSelectedUnit">
            <el-option label="全部知识区域" value="all" />
            <el-option v-for="unit in graph.units" :key="unit.key" :label="unit.title" :value="unit.key" />
          </el-select>
          <el-select v-model="activeFilter" size="small" aria-label="筛选知识节点" style="width: 150px">
            <el-option label="全部节点" value="all" /><el-option label="当前路径" value="current_path" /><el-option label="已生成" value="generated" /><el-option label="待生成" value="pending" /><el-option label="未接触" value="unseen" /><el-option label="学习中" value="learning" /><el-option label="已掌握" value="mastered" /><el-option label="应用节点" value="application" />
          </el-select>
        </div>
      </section>

      <div v-if="graph && activeView === 'path'" class="knowledge-path-workspace">
        <KnowledgeTreePath
          ref="pathRef"
          :units="graph.units"
          :nodes="pathNodes"
          :edges="graph.edges"
          :current-lesson-id="currentLessonID"
          :cognitive-states="cognitiveStates"
          :selected-node-id="selectedGraphNode?.node_id"
          :selected-unit-key="selectedUnitKey"
          :show-blueprint="showBlueprint"
          :expanded-unit-keys="expandedUnitKeys"
          :expanding-unit-id="expandingUnitID"
          :generating-unit-id="generatingUnitID"
          @select="selectNode"
          @toggle-unit="toggleUnit"
          @expand="expandUnit"
          @generate="({ id, scope }) => generateUnitDraft(id, scope)"
        />
        <NodeDetailPanel
          class="node-detail-panel--desktop"
          :node="selectedGraphNode"
          :current-lesson-id="currentLessonID"
          :relations="relations"
          :blueprint-relations="blueprintRelations"
          :relations-loading="relationsLoading"
          :cognitive-detail="cognitiveDetail"
          :evidence-items="evidenceItems"
          :can-expand="canExpandSelectedUnit"
          :can-generate="canGenerateSelectedUnit"
          :expanding="expandingUnitID === selectedGraphNode?.blueprint_unit_id"
          :generating="generatingUnitID === selectedGraphNode?.blueprint_unit_id"
          @clear="clearSelection"
          @start="startLearning"
          @relation-select="selectRelatedLesson"
          @expand="expandUnit"
          @generate="({ id, scope }) => generateUnitDraft(id, scope)"
        />
        <el-drawer v-model="mobileDetailOpen" class="knowledge-node-drawer" direction="rtl" size="min(92vw, 380px)" :with-header="false">
          <NodeDetailPanel
            :node="selectedGraphNode"
            :current-lesson-id="currentLessonID"
            :relations="relations"
            :blueprint-relations="blueprintRelations"
            :relations-loading="relationsLoading"
            :cognitive-detail="cognitiveDetail"
            :evidence-items="evidenceItems"
            :can-expand="canExpandSelectedUnit"
            :can-generate="canGenerateSelectedUnit"
            :expanding="expandingUnitID === selectedGraphNode?.blueprint_unit_id"
            :generating="generatingUnitID === selectedGraphNode?.blueprint_unit_id"
            @clear="clearSelection"
            @start="startLearning"
            @relation-select="selectRelatedLesson"
            @expand="expandUnit"
            @generate="({ id, scope }) => generateUnitDraft(id, scope)"
          />
        </el-drawer>
      </div>

      <div v-if="graph && activeView === 'list'" class="knowledge-workspace">
        <KnowledgePath :units="graph.units" :nodes="pathNodes" :current-lesson-id="currentLessonID" :cognitive-states="cognitiveStates" :selected-node-id="selectedGraphNode?.node_id" :expanded-unit-keys="expandedUnitKeys" :expanding-unit-id="expandingUnitID" :generating-unit-id="generatingUnitID" @select="selectNode" @toggle-unit="toggleUnit" @expand="expandUnit" @generate="({ id, scope }) => generateUnitDraft(id, scope)" />
        <NodeDetailPanel
          :node="selectedGraphNode"
          :current-lesson-id="currentLessonID"
          :relations="relations"
          :blueprint-relations="blueprintRelations"
          :relations-loading="relationsLoading"
          :cognitive-detail="cognitiveDetail"
          :evidence-items="evidenceItems"
          :can-expand="canExpandSelectedUnit"
          :can-generate="canGenerateSelectedUnit"
          :expanding="expandingUnitID === selectedGraphNode?.blueprint_unit_id"
          :generating="generatingUnitID === selectedGraphNode?.blueprint_unit_id"
          @clear="clearSelection"
          @start="startLearning"
          @relation-select="selectRelatedLesson"
          @expand="expandUnit"
          @generate="({ id, scope }) => generateUnitDraft(id, scope)"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { getKnowledgeGraph, getLessonRelations } from '@/api/knowledgeGraph'
import { getCurrentLesson, setCurrentLesson } from '@/api/learning'
import { createCurriculumDraft, expandBlueprintUnit } from '@/api/curriculum'
import AIRequestError from '@/components/AIRequestError.vue'
import KnowledgePath from '@/components/KnowledgePath.vue'
import KnowledgeTreePath from '@/components/KnowledgeTreePath.vue'
import NodeDetailPanel from '@/components/NodeDetailPanel.vue'
import PageHeader from '@/components/PageHeader.vue'
import { APIRequestError, isRequestAborted } from '@/api/http'
import type { KnowledgeGraph, KnowledgeGraphNode, LessonRelationLesson, LessonRelations } from '@/types/knowledgeGraph'
import { getCourseCognitiveStates, getLessonCognitiveState } from '@/api/cognitive'
import type { CognitiveStateDetail, CognitiveStateSummary, CognitiveLevel } from '@/types/cognitive'
import { filterKnowledgeNodes, type KnowledgeGraphFilter } from '@/utils/knowledgeGraphFilters'
import { invalidateExplorationRadar } from '@/api/exploration'

const route = useRoute()
const router = useRouter()
const courseID = Number(String(route.params.id))
const graph = ref<KnowledgeGraph | null>(null)
const selectedGraphNode = ref<KnowledgeGraphNode | null>(null)
const relations = ref<LessonRelations | null>(null)
const cognitiveDetail = ref<CognitiveStateDetail | null>(null)
const cognitiveStates = ref<Map<number, CognitiveStateSummary>>(new Map())
const loading = ref(false)
const relationsLoading = ref(false)
const error = ref('')
const aiError = ref<APIRequestError | null>(null)
const retryAction = ref<(() => void) | undefined>()
const currentLessonID = ref(0)
const expandingUnitID = ref(0)
const generatingUnitID = ref(0)
const activeView = ref<'path' | 'list'>(readStoredView())
const selectedUnitKey = ref('all')
const showBlueprint = ref(true)
const activeFilter = ref<KnowledgeGraphFilter>('all')
const expandedUnitKeys = ref<string[]>([])
const pathRef = ref<InstanceType<typeof KnowledgeTreePath> | null>(null)
const mobileDetailOpen = ref(false)
const pathEntryHandled = ref(false)
const scrollStorageKey = `knowledge-map-scroll:${courseID}`
let selectionRequestID = 0
let graphRequestController: AbortController | null = null
let detailRequestController: AbortController | null = null
let generationPollTimer: number | null = null
const relationCache = new Map<number, LessonRelations>()
const cognitiveDetailCache = new Map<number, CognitiveStateDetail>()

function readStoredView(): 'path' | 'list' {
  return window.localStorage.getItem('knowledge-map-view') === 'list' ? 'list' : 'path'
}

function setView(view: 'path' | 'list') {
  activeView.value = view
  window.localStorage.setItem('knowledge-map-view', view)
  if (view === 'path') void initializePathPosition()
}

async function loadGraph() {
  graphRequestController?.abort()
  const controller = new AbortController()
  graphRequestController = controller
  loading.value = true
  error.value = ''
  aiError.value = null
  try {
    const [loadedGraph, currentLesson, stateResponse] = await Promise.all([
      getKnowledgeGraph(courseID, controller.signal),
      getCurrentLesson(courseID, controller.signal).catch((reason) => {
        if (isRequestAborted(reason)) throw reason
        return null
      }),
      getCourseCognitiveStates(courseID, controller.signal).catch((reason) => {
        if (isRequestAborted(reason)) throw reason
        return null
      }),
    ])
    if (controller.signal.aborted) return
    graph.value = loadedGraph
    currentLessonID.value = currentLesson?.lesson.id ?? 0
    if (expandedUnitKeys.value.length === 0) {
      const currentUnit = loadedGraph.nodes.find((node) => node.lesson_id === currentLessonID.value)?.unit_key
      expandedUnitKeys.value = currentUnit ? [currentUnit] : loadedGraph.units[0] ? [loadedGraph.units[0].key] : []
    }
    cognitiveStates.value = new Map((stateResponse?.states ?? []).map((state) => [state.lesson_id, state]))
    const previouslySelectedID = selectedGraphNode.value?.node_id
    const nextNode = previouslySelectedID ? loadedGraph.nodes.find((node) => node.node_id === previouslySelectedID) : undefined
    if (nextNode) await selectNode(nextNode)
    else clearSelection()
    if (activeView.value === 'path') void initializePathPosition()
  } catch (reason) {
    if (isRequestAborted(reason)) return
    error.value = reason instanceof Error ? reason.message : '知识结构读取失败'
  } finally {
    if (graphRequestController === controller) loading.value = false
  }
}

async function startLearning(node: KnowledgeGraphNode) {
  if (node.node_type === 'blueprint' || !node.lesson_id) return
  if (currentLessonID.value !== node.lesson_id && nodeState(node.lesson_id).current_level === 'unseen') {
    const relationData = await getLessonRelations(courseID, node.lesson_id).catch(() => null)
    if (relationData?.prerequisites.length) {
      const unfinished = relationData.prerequisites.filter((item) => nodeState(item.id).current_level === 'unseen').map((item) => item.title)
      if (unfinished.length) {
        try {
          await ElMessageBox.confirm(`这个节点存在尚未完成的前置知识：${unfinished.join('、')}。仍然开始学习吗？`, '前置知识提示', { confirmButtonText: '仍然开始学习', cancelButtonText: '取消', type: 'warning' })
        } catch {
          return
        }
      }
    }
  }
  try {
    await setCurrentLesson(courseID, node.lesson_id)
    invalidateExplorationRadar(courseID)
    currentLessonID.value = node.lesson_id
    if (graph.value) graph.value.nodes.forEach((item) => { item.is_current = item.node_id === node.node_id })
    router.push(`/courses/${courseID}/learn`)
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '切换当前学习节点失败'
  }
}

async function expandUnit(unitID: number) {
  if (expandingUnitID.value === unitID || generatingUnitID.value !== 0) return
  expandingUnitID.value = unitID
  aiError.value = null
  try {
    await expandBlueprintUnit(courseID, unitID)
    ElMessage.success('知识区域已展开，可以生成第一批正式学习节点。')
    await loadGraph()
  } catch (reason) {
    if (reason instanceof APIRequestError) {
      if (reason.code === 'UNIT_EXPANSION_IN_PROGRESS') {
        ElMessage.info('知识区域正在生成，请稍候。')
        scheduleGraphRefresh()
      } else if (reason.code === 'UNIT_ALREADY_EXPANDED') {
        await loadGraph()
      } else {
        aiError.value = reason
        retryAction.value = () => { void expandUnit(unitID) }
      }
    } else {
      error.value = reason instanceof Error ? reason.message : '知识区域展开失败'
    }
  } finally {
    expandingUnitID.value = 0
  }
}

async function generateUnitDraft(unitID: number, scope: 'missing_core' | 'missing_recommended' | '') {
  if (generatingUnitID.value === unitID || expandingUnitID.value !== 0) return
  generatingUnitID.value = unitID
  aiError.value = null
  try {
    await createCurriculumDraft(courseID, scope || 'missing_core', 5, unitID)
    ElMessage.success('草案已生成，请审核后加入知识世界。')
    router.push(`/courses/${courseID}/archive?tab=drafts&return_to=map`)
  } catch (reason) {
    if (reason instanceof APIRequestError) {
      if (reason.code === 'CURRICULUM_DRAFT_ALREADY_PENDING') {
        ElMessage.info('你已经有一份待审核的课程草案。')
        await router.push(`/courses/${courseID}/archive?tab=drafts&return_to=map`)
      } else if (reason.code === 'CURRICULUM_DRAFT_GENERATION_IN_PROGRESS') {
        ElMessage.info('课程草案正在生成，请稍候。')
        router.push(`/courses/${courseID}/archive?tab=drafts&return_to=map`)
      } else {
        aiError.value = reason
        retryAction.value = () => { void generateUnitDraft(unitID, scope) }
      }
    } else {
      error.value = reason instanceof Error ? reason.message : '学习节点草案生成失败'
    }
  } finally {
    generatingUnitID.value = 0
  }
}

async function selectNode(node: KnowledgeGraphNode) {
  const requestID = ++selectionRequestID
  detailRequestController?.abort()
  detailRequestController = null
  selectedGraphNode.value = node
  if (activeView.value === 'path' && window.matchMedia('(max-width: 820px)').matches) mobileDetailOpen.value = true
  relations.value = null
  cognitiveDetail.value = null
  relationsLoading.value = false
  if (node.node_type === 'blueprint' || !node.lesson_id) return
  const cachedRelations = relationCache.get(node.lesson_id)
  const cachedCognitive = cognitiveDetailCache.get(node.lesson_id)
  if (cachedRelations && cachedCognitive) {
    relations.value = cachedRelations
    cognitiveDetail.value = cachedCognitive
    return
  }
  const controller = new AbortController()
  detailRequestController = controller
  relationsLoading.value = true
  try {
    const [nextRelations, nextCognitive] = await Promise.all([
      cachedRelations ?? getLessonRelations(courseID, node.lesson_id, controller.signal),
      cachedCognitive ?? getLessonCognitiveState(courseID, node.lesson_id, controller.signal),
    ])
    if (requestID !== selectionRequestID) return
    relationCache.set(node.lesson_id, nextRelations)
    cognitiveDetailCache.set(node.lesson_id, nextCognitive)
    relations.value = nextRelations
    cognitiveDetail.value = nextCognitive
  } catch (reason) {
    if (isRequestAborted(reason)) return
    if (requestID === selectionRequestID) error.value = reason instanceof Error ? reason.message : '节点关系读取失败'
  } finally {
    if (requestID === selectionRequestID) relationsLoading.value = false
  }
}

function selectRelatedLesson(item: LessonRelationLesson) {
  const target = graph.value?.nodes.find((node) => item.node_id ? node.node_id === item.node_id : node.lesson_id === item.id)
  if (!target) return
  if (activeView.value === 'path') pathRef.value?.scrollToNode(target.node_id)
  else void selectNode(target)
}

function clearSelection() {
  detailRequestController?.abort()
  detailRequestController = null
  selectionRequestID += 1
  selectedGraphNode.value = null
  relations.value = null
  cognitiveDetail.value = null
  relationsLoading.value = false
  mobileDetailOpen.value = false
}

function scheduleGraphRefresh() {
  if (generationPollTimer !== null) window.clearTimeout(generationPollTimer)
  generationPollTimer = window.setTimeout(async () => {
    generationPollTimer = null
    await loadGraph()
    if (graph.value?.units.some((unit) => unit.expansion_status === 'expanding')) scheduleGraphRefresh()
  }, 1500)
}

function nodeState(lessonID: number): CognitiveStateSummary {
  return cognitiveStates.value.get(lessonID) ?? { lesson_id: lessonID, current_level: 'unseen', status: 'unknown', understanding_summary: '', evidence_count: 0 }
}

function scrollToSelectedUnit() {
  if (selectedUnitKey.value !== 'all') {
    if (!expandedUnitKeys.value.includes(selectedUnitKey.value)) expandedUnitKeys.value = [...expandedUnitKeys.value, selectedUnitKey.value]
    nextTick(() => pathRef.value?.scrollToUnit(selectedUnitKey.value))
  }
}

function toggleUnit(unitKey: string) {
  expandedUnitKeys.value = expandedUnitKeys.value.includes(unitKey) ? expandedUnitKeys.value.filter((key) => key !== unitKey) : [...expandedUnitKeys.value, unitKey]
}

async function initializePathPosition() {
  if (pathEntryHandled.value || !graph.value || activeView.value !== 'path') return
  pathEntryHandled.value = true
  await nextTick()
  window.requestAnimationFrame(() => {
    const explicitFocus = String(route.query.focus ?? '') === 'current'
    const savedPosition = explicitFocus ? null : window.sessionStorage.getItem(scrollStorageKey)
    if (savedPosition !== null) {
      const top = Number(savedPosition)
      if (Number.isFinite(top)) window.scrollTo({ top, behavior: 'auto' })
      return
    }
    if (currentLessonID.value) pathRef.value?.scrollToNode(`lesson:${currentLessonID.value}`, false)
  })
}

const blueprintNodeCount = computed(() => graph.value?.nodes.filter((node) => node.node_type === 'blueprint').length ?? 0)
const evidenceItems = computed<LessonRelationLesson[]>(() => cognitiveDetail.value?.evidence.map((item) => ({ id: item.id, title: `${levelText(item.cognitive_level)} · ${item.description}` })) ?? [])
const pathNodes = computed(() => graph.value ? filterKnowledgeNodes(graph.value.nodes, graph.value.edges, cognitiveStates.value, currentLessonID.value, activeFilter.value) : [])
const selectedUnit = computed(() => {
  const blueprintUnitID = selectedGraphNode.value?.blueprint_unit_id
  return graph.value?.units.find((unit) => unit.blueprint_unit_id === blueprintUnitID) ?? null
})
const blueprintRelations = computed<LessonRelations | null>(() => {
  const current = selectedGraphNode.value
  if (!current || current.node_type !== 'blueprint' || !graph.value) return null
  const result: LessonRelations = {
    lesson: { id: current.blueprint_lesson_id ?? current.id, title: current.title, node_id: current.node_id },
    prerequisites: [],
    next_lessons: [],
    extensions: [],
    applications: [],
    related: [],
  }
  const seen = new Map<keyof Omit<LessonRelations, 'lesson'>, Set<string>>([
    ['prerequisites', new Set()], ['next_lessons', new Set()], ['extensions', new Set()], ['applications', new Set()], ['related', new Set()],
  ])
  const add = (group: keyof Omit<LessonRelations, 'lesson'>, node: KnowledgeGraphNode) => {
    const item = { id: node.lesson_id ?? node.blueprint_lesson_id ?? node.id, title: node.title, node_id: node.node_id }
    if (seen.get(group)?.has(item.node_id)) return
    seen.get(group)?.add(item.node_id)
    result[group].push(item)
  }
  graph.value.edges.filter((edge) => edge.source === current.node_id || edge.target === current.node_id).forEach((edge) => {
    const neighborID = edge.source === current.node_id ? edge.target : edge.source
    const neighbor = graph.value?.nodes.find((node) => node.node_id === neighborID)
    if (!neighbor) return
    if (edge.relation_type === 'prerequisite') add(edge.target === current.node_id ? 'prerequisites' : 'next_lessons', neighbor)
    else add(edge.relation_type === 'extends' ? 'extensions' : edge.relation_type === 'application' ? 'applications' : 'related', neighbor)
  })
  return result
})
const canExpandSelectedUnit = computed(() => selectedGraphNode.value?.node_type === 'blueprint' && Boolean(selectedUnit.value?.needs_expansion))
const canGenerateSelectedUnit = computed(() => selectedGraphNode.value?.node_type === 'blueprint' && Boolean(selectedUnit.value?.needs_generation))

function levelText(level: CognitiveLevel | string): string {
  return { unseen: '未接触', exposed: '已接触', recognize: '识别', understand: '理解', apply: '应用', transfer: '迁移' }[level] ?? '未知'
}

watch(activeView, (view) => {
  if (view === 'path') void initializePathPosition()
})
onMounted(loadGraph)
onBeforeUnmount(() => {
  graphRequestController?.abort()
  detailRequestController?.abort()
  if (generationPollTimer !== null) window.clearTimeout(generationPollTimer)
  window.sessionStorage.setItem(scrollStorageKey, String(window.scrollY))
})
</script>
