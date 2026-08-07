<template>
  <section class="knowledge-graph-page">
    <div class="learning-toolbar">
      <el-button text @click="router.push(`/courses/${courseID}/learn`)">← 返回学习</el-button>
      <el-button text @click="loadGraph">刷新结构</el-button>
    </div>

    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />

    <div v-loading="loading">
      <el-card v-if="graph" shadow="never" class="graph-header-card">
        <div class="graph-heading">
          <div>
            <span class="eyebrow">KNOWLEDGE WORLD · STATIC STRUCTURE</span>
            <h2>{{ graph.course.name }} · 知识结构</h2>
            <p>这里展示课程知识之间的结构关系，不代表掌握度、解锁条件或唯一学习路线。</p>
          </div>
          <el-button type="primary" @click="router.push(`/courses/${courseID}/learn`)">回到学习</el-button>
        </div>
        <div class="graph-stats">
          <div><strong>{{ graph.stats.node_count }}</strong><span>知识节点</span></div>
          <div><strong>{{ graph.stats.core_node_count }}</strong><span>核心节点</span></div>
          <div><strong>{{ graph.stats.optional_node_count }}</strong><span>扩展节点</span></div>
          <div><strong>{{ graph.stats.edge_count }}</strong><span>结构关系</span></div>
        </div>
      </el-card>

      <div v-if="graph" class="graph-layout">
        <div class="unit-list">
          <el-card v-for="unit in graph.units" :key="unit.id" shadow="never" class="unit-card">
            <template #header>
              <div class="card-header-row">
                <div>
                  <strong>{{ unit.title }}</strong>
                  <small v-if="unit.objective" class="muted-text">{{ unit.objective }}</small>
                </div>
                <el-tag size="small" effect="plain">{{ unitNodes(unit.id).length }} 个节点</el-tag>
              </div>
            </template>
            <div class="node-list">
              <button
                v-for="node in unitNodes(unit.id)"
                :key="node.id"
                type="button"
                class="node-card"
                :class="{ 'node-card--active': selectedNode?.id === node.id }"
                @click="selectNode(node)"
              >
                <span class="node-card__title">{{ node.title }}</span>
                <span class="node-card__meta">
                  <el-tag size="small" :type="node.is_core ? 'primary' : 'warning'">
                    {{ node.is_core ? '核心' : '扩展' }}
                  </el-tag>
                  <el-tag size="small" effect="plain">{{ roleText(node.content_role) }}</el-tag>
                  <span>Depth {{ node.depth_level }}</span>
                  <el-tag size="small" effect="plain" type="success">{{ levelText(nodeState(node.id).current_level) }}</el-tag>
                  <el-tag v-if="nodeState(node.id).current_level === 'transfer'" size="small" type="success">迁移已验证</el-tag>
                  <span>{{ statusText(nodeState(node.id).status) }}</span>
                </span>
              </button>
            </div>
          </el-card>
        </div>

        <el-card shadow="never" class="relation-card">
          <template #header>
            <div class="card-header-row">
              <span>节点关系</span>
              <el-tag v-if="selectedNode" effect="plain">静态结构</el-tag>
            </div>
          </template>
          <el-empty v-if="!selectedNode" description="点击一个节点查看它从哪里来、会通向哪里" />
          <div v-else v-loading="relationsLoading">
            <h3>{{ selectedNode.title }}</h3>
            <p class="muted-text">{{ roleText(selectedNode.content_role) }} · Depth {{ selectedNode.depth_level }}</p>
            <RelationGroup title="前置知识" :items="relations?.prerequisites ?? []" />
            <RelationGroup title="后续知识" :items="relations?.next_lessons ?? []" />
            <RelationGroup title="深化内容" :items="relations?.extensions ?? []" />
            <RelationGroup title="应用场景" :items="relations?.applications ?? []" />
            <RelationGroup title="相关知识" :items="relations?.related ?? []" />
            <div v-if="cognitiveDetail" class="cognitive-overlay">
              <h4>我的认知</h4>
              <div class="cognitive-overlay__status">
                <el-tag type="success">{{ levelText(cognitiveDetail.state.current_level) }}</el-tag>
                <el-tag effect="plain">{{ statusText(cognitiveDetail.state.status) }}</el-tag>
                <el-tag v-if="cognitiveDetail.state.current_level === 'transfer'" type="success">迁移已验证</el-tag>
              </div>
              <p v-if="cognitiveDetail.state.understanding_summary">{{ cognitiveDetail.state.understanding_summary }}</p>
              <p v-else class="muted-text">尚无 Phase 5 认知证据，当前为未接触。</p>
              <RelationGroup title="掌握证据" :items="evidenceItems" />
              <div class="relation-group">
                <h4>理解变化</h4>
                <ul v-if="cognitiveDetail.timeline.length > 0">
                  <li v-for="event in cognitiveDetail.timeline" :key="event.id">
                    {{ levelText(event.from_level) }} → {{ levelText(event.to_level) }} · {{ statusText(event.to_status) }}
                  </li>
                </ul>
                <p v-else class="muted-text">暂无变化记录</p>
              </div>
            </div>
          </div>
        </el-card>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getKnowledgeGraph, getLessonRelations } from '@/api/knowledgeGraph'
import RelationGroup from '@/components/RelationGroup.vue'
import type { KnowledgeGraph, KnowledgeGraphNode, LessonRelationLesson, LessonRelations } from '@/types/knowledgeGraph'
import { getCourseCognitiveStates, getLessonCognitiveState } from '@/api/cognitive'
import type { CognitiveStateDetail, CognitiveStateSummary, CognitiveLevel, CognitiveStatus } from '@/types/cognitive'

const route = useRoute()
const router = useRouter()
const courseID = Number(String(route.params.id))
const graph = ref<KnowledgeGraph | null>(null)
const selectedNode = ref<KnowledgeGraphNode | null>(null)
const relations = ref<LessonRelations | null>(null)
const cognitiveDetail = ref<CognitiveStateDetail | null>(null)
const cognitiveStates = ref<Map<number, CognitiveStateSummary>>(new Map())
const loading = ref(false)
const relationsLoading = ref(false)
const cognitiveLoading = ref(false)
const error = ref('')

async function loadGraph() {
  loading.value = true
  error.value = ''
  try {
    const loadedGraph = await getKnowledgeGraph(courseID)
    graph.value = loadedGraph
    try {
      const stateResponse = await getCourseCognitiveStates(courseID)
      cognitiveStates.value = new Map(stateResponse.states.map((state) => [state.lesson_id, state]))
    } catch {
      cognitiveStates.value = new Map()
    }
    const nextNode = selectedNode.value
      ? loadedGraph.nodes.find((node) => node.id === selectedNode.value?.id)
      : loadedGraph.nodes[0]
    if (nextNode) {
      await selectNode(nextNode)
    }
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '知识结构读取失败'
  } finally {
    loading.value = false
  }
}

async function selectNode(node: KnowledgeGraphNode) {
  selectedNode.value = node
  relationsLoading.value = true
  cognitiveLoading.value = true
  try {
    const [nextRelations, nextCognitive] = await Promise.all([
      getLessonRelations(courseID, node.id),
      getLessonCognitiveState(courseID, node.id),
    ])
    relations.value = nextRelations
    cognitiveDetail.value = nextCognitive
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '节点关系读取失败'
  } finally {
    relationsLoading.value = false
    cognitiveLoading.value = false
  }
}

function unitNodes(unitID: number): KnowledgeGraphNode[] {
  return graph.value?.nodes.filter((node) => node.unit_id === unitID) ?? []
}

function roleText(role: KnowledgeGraphNode['content_role']): string {
  return { foundation: '基础', core: '核心', application: '应用', extension: '扩展' }[role]
}

function nodeState(lessonID: number): CognitiveStateSummary {
  return cognitiveStates.value.get(lessonID) ?? {
    lesson_id: lessonID,
    current_level: 'unseen',
    status: 'unknown',
    understanding_summary: '',
    evidence_count: 0,
  }
}

function levelText(level: CognitiveLevel | string): string {
  return {
    unseen: '未接触', exposed: '已接触', recognize: '识别', understand: '理解', apply: '应用', transfer: '迁移',
  }[level] ?? '未知'
}

function statusText(status: CognitiveStatus | string): string {
  return { unknown: '未知', developing: '发展中', stable: '稳定', needs_review: '待复习' }[status] ?? '未知'
}

const evidenceItems = computed<LessonRelationLesson[]>(() => cognitiveDetail.value?.evidence.map((item) => ({
  id: item.id,
  title: `${levelText(item.cognitive_level)} · ${item.description}`,
})) ?? [])

onMounted(loadGraph)
</script>
