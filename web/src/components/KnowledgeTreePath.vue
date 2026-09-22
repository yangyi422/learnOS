<template>
  <div ref="pathContent" class="knowledge-tree-path">
    <svg class="knowledge-tree-path__connections" :width="svgSize.width" :height="svgSize.height" :viewBox="`0 0 ${svgSize.width} ${svgSize.height}`" aria-hidden="true">
      <defs>
        <marker id="knowledge-path-arrow" markerWidth="7" markerHeight="7" refX="5" refY="3.5" orient="auto">
          <path d="M0,0 L7,3.5 L0,7 z" fill="#c7d3e0" />
        </marker>
      </defs>
      <path v-for="connection in connections" :key="connection.id" :d="connection.path" marker-end="url(#knowledge-path-arrow)" />
    </svg>

    <section v-for="(section, index) in pathUnits" :key="section.unit.id" :ref="(element) => registerUnit(section.unit.key, element)" class="knowledge-tree-unit" :class="{ 'is-selected': selectedUnitKey === section.unit.key }" :data-unit-key="section.unit.key">
      <header class="knowledge-tree-unit__header">
        <div class="knowledge-tree-unit__identity">
          <span class="knowledge-tree-unit__number">{{ String(index + 1).padStart(2, '0') }}</span>
          <div><h2>{{ section.unit.title }}</h2><p>{{ section.unit.objective || '逐步建立这个知识区域。' }}</p></div>
        </div>
        <div class="knowledge-tree-unit__progress">
          <span>{{ section.unit.applied_lesson_count }} / {{ section.unit.blueprint_lesson_count }} 已展开</span>
          <el-progress :percentage="section.unit.blueprint_lesson_count ? Math.round(section.unit.applied_lesson_count / section.unit.blueprint_lesson_count * 100) : 0" :show-text="false" :stroke-width="4" />
          <button type="button" :aria-expanded="isExpanded(section.unit.key)" @click="$emit('toggle-unit', section.unit.key)">{{ isExpanded(section.unit.key) ? '收起' : `展开 · ${section.ranks.reduce((sum, rank) => sum + rank.nodes.length, 0)} 个节点` }}</button>
        </div>
      </header>

      <div v-if="isExpanded(section.unit.key) && section.ranks.length" class="knowledge-tree-unit__path">
        <div v-for="(rank, rankIndex) in section.ranks" :key="rank.rank" class="knowledge-tree-rank" :class="{ 'knowledge-tree-rank--first': rankIndex === 0 }" :style="{ '--rank-columns': rank.nodes.length }">
          <button
            v-for="node in rank.nodes"
            :key="node.node_id"
            :ref="(element) => registerNode(node.node_id, element)"
            type="button"
            class="knowledge-tree-node"
            :class="nodeClasses(node)"
            :aria-current="node.node_id === selectedNodeId ? 'true' : undefined"
            @click="selectNode(node)"
          >
            <span class="knowledge-tree-node__marker">{{ marker(node) }}</span>
            <span class="knowledge-tree-node__body"><strong>{{ node.title }}</strong><small>{{ node.node_type === 'blueprint' ? '待生成节点' : `${roleText(node)} · ${knowledgeDepthText(node.depth_level)}` }}</small></span>
            <span v-if="node.is_current" class="knowledge-tree-node__current">当前</span>
          </button>
        </div>
      </div>

      <div v-else-if="isExpanded(section.unit.key)" class="knowledge-tree-unit__empty">
        <p>这个区域还没有可展示的知识节点。</p>
      </div>

      <div v-if="isExpanded(section.unit.key) && section.unit.needs_expansion && section.unit.blueprint_unit_id" class="knowledge-tree-unit__action">
        <p>{{ expandingUnitId === section.unit.blueprint_unit_id ? '正在展开知识区域…' : '这个区域尚未展开。' }}</p><button type="button" :disabled="expandingUnitId === section.unit.blueprint_unit_id" :aria-busy="expandingUnitId === section.unit.blueprint_unit_id" @click="$emit('expand', section.unit.blueprint_unit_id!)">{{ expandingUnitId === section.unit.blueprint_unit_id ? '正在展开知识区域…' : '展开这个知识区域' }} <span>→</span></button>
      </div>
      <div v-else-if="isExpanded(section.unit.key) && section.unit.needs_generation && section.unit.blueprint_unit_id" class="knowledge-tree-unit__action">
        <p>{{ generatingUnitId === section.unit.blueprint_unit_id ? '正在准备课程草案…' : `${section.unit.applied_lesson_count} / ${section.unit.blueprint_lesson_count} 个学习节点已生成；下一批最多生成 ${generationCount(section.unit)} 个可审核草案，不会切换当前课程节点。` }}</p><button type="button" :disabled="generatingUnitId === section.unit.blueprint_unit_id" :aria-busy="generatingUnitId === section.unit.blueprint_unit_id" @click="$emit('generate', { id: section.unit.blueprint_unit_id!, scope: section.unit.generation_scope })">{{ generatingUnitId === section.unit.blueprint_unit_id ? '正在准备课程草案…' : (section.unit.generation_scope === 'missing_recommended' ? `继续生成 ${generationCount(section.unit)} 个节点` : `生成第一批 ${generationCount(section.unit)} 个学习节点`) }} <span aria-hidden="true">→</span></button>
      </div>
    </section>

    <el-empty v-if="!pathUnits.some((section) => section.ranks.length)" description="当前筛选条件下没有知识节点" />
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch, type ComponentPublicInstance } from 'vue'
import type { CognitiveStateSummary } from '@/types/cognitive'
import type { KnowledgeGraphEdge, KnowledgeGraphNode, KnowledgeGraphUnit } from '@/types/knowledgeGraph'
import { buildKnowledgePathUnits, type KnowledgePathUnit } from '@/utils/knowledgePathLayout'
import { knowledgeDepthText, knowledgeNodeTypeText } from '@/utils/knowledgeNodeType'

const props = defineProps<{
  units: KnowledgeGraphUnit[]
  nodes: KnowledgeGraphNode[]
  edges: KnowledgeGraphEdge[]
  currentLessonId: number
  cognitiveStates: Map<number, CognitiveStateSummary>
  selectedNodeId?: string | null
  selectedUnitKey?: string
  showBlueprint: boolean
  expandingUnitId: number
  generatingUnitId: number
  expandedUnitKeys: string[]
}>()

const emit = defineEmits<{
  select: [node: KnowledgeGraphNode]
  expand: [unitID: number]
  generate: [payload: { id: number; scope: 'missing_core' | 'missing_recommended' | '' }]
  'toggle-unit': [unitKey: string]
}>()

const pathContent = ref<HTMLElement | null>(null)
const pathUnits = ref<KnowledgePathUnit[]>([])
const nodeElements = new Map<string, HTMLElement>()
const unitElements = new Map<string, HTMLElement>()
const connections = ref<Array<{ id: string; path: string }>>([])
const svgSize = ref({ width: 1, height: 1 })
let observer: IntersectionObserver | null = null
let resizeObserver: ResizeObserver | null = null
const intersections = new Map<string, IntersectionObserverEntry>()

function rebuild() {
  pathUnits.value = buildKnowledgePathUnits(props.units, props.nodes, props.edges, props.showBlueprint)
  void nextTick(() => {
    connectObserver()
    drawConnections()
  })
}

function registerNode(nodeID: string, element: Element | ComponentPublicInstance | null) {
  const htmlElement = element instanceof HTMLElement ? element : null
  if (htmlElement) nodeElements.set(nodeID, htmlElement)
  else nodeElements.delete(nodeID)
  if (htmlElement && observer) observer.observe(htmlElement)
}

function registerUnit(unitKey: string, element: Element | ComponentPublicInstance | null) {
  const htmlElement = element instanceof HTMLElement ? element : null
  if (htmlElement) unitElements.set(unitKey, htmlElement)
  else unitElements.delete(unitKey)
}

function connectObserver() {
  observer?.disconnect()
  observer = new IntersectionObserver(handleIntersections, { rootMargin: '-40% 0px -40% 0px', threshold: [0, 0.5, 1] })
  nodeElements.forEach((element) => observer?.observe(element))
}

function handleIntersections(entries: IntersectionObserverEntry[]) {
  entries.forEach((entry) => {
    const nodeID = [...nodeElements.entries()].find(([, element]) => element === entry.target)?.[0]
    if (nodeID) intersections.set(nodeID, entry)
  })
  const viewportCenter = window.innerHeight / 2
  const candidate = [...intersections.entries()]
    .filter(([, entry]) => entry.isIntersecting)
    .sort(([, left], [, right]) => Math.abs((left.boundingClientRect.top + left.boundingClientRect.bottom) / 2 - viewportCenter) - Math.abs((right.boundingClientRect.top + right.boundingClientRect.bottom) / 2 - viewportCenter))[0]
  if (!candidate || candidate[0] === props.selectedNodeId) return
  const node = props.nodes.find((item) => item.node_id === candidate[0])
  if (node) emit('select', node)
}

function selectNode(node: KnowledgeGraphNode) {
  emit('select', node)
  const element = nodeElements.get(node.node_id)
  if (element) element.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

function scrollToNode(nodeID: string, smooth = true) {
  const element = nodeElements.get(nodeID)
  if (!element) {
    const nodeInLayout = pathUnits.value.some((section) => section.ranks.some((rank) => rank.nodes.some((node) => node.node_id === nodeID)))
    if (pathUnits.value.length && !nodeInLayout) return false
    void nextTick(() => scrollToNode(nodeID, smooth))
    return false
  }
  element.scrollIntoView({ behavior: smooth ? 'smooth' : 'auto', block: 'center' })
  const node = props.nodes.find((item) => item.node_id === nodeID)
  if (node && node.node_id !== props.selectedNodeId) emit('select', node)
  return true
}

function scrollToUnit(unitKey: string, smooth = true) {
  const element = unitElements.get(unitKey)
  if (element) element.scrollIntoView({ behavior: smooth ? 'smooth' : 'auto', block: 'start' })
  else if (props.units.some((unit) => unit.key === unitKey)) void nextTick(() => scrollToUnit(unitKey, smooth))
}

function drawConnections() {
  const root = pathContent.value
  if (!root) return
  const rootRect = root.getBoundingClientRect()
  svgSize.value = { width: Math.max(root.clientWidth, 1), height: Math.max(root.scrollHeight, 1) }
  connections.value = props.edges.filter((edge) => edge.relation_type === 'prerequisite' && nodeElements.has(edge.source) && nodeElements.has(edge.target)).map((edge) => {
    const source = nodeElements.get(edge.source)!.getBoundingClientRect()
    const target = nodeElements.get(edge.target)!.getBoundingClientRect()
    const x1 = source.left + source.width / 2 - rootRect.left
    const y1 = source.bottom - rootRect.top
    const x2 = target.left + target.width / 2 - rootRect.left
    const y2 = target.top - rootRect.top
    const midY = y1 + Math.max((y2 - y1) / 2, 18)
    return { id: edge.edge_id, path: `M ${x1} ${y1} C ${x1} ${midY}, ${x2} ${midY}, ${x2} ${y2}` }
  })
}

function nodeState(node: KnowledgeGraphNode) {
  return node.lesson_id ? props.cognitiveStates.get(node.lesson_id) : undefined
}

function marker(node: KnowledgeGraphNode) {
  if (node.node_type === 'blueprint') return '◇'
  const level = nodeState(node)?.current_level
  if (node.is_current) return '●'
  if (level === 'transfer') return '✦'
  return level && level !== 'unseen' ? '●' : '○'
}

function nodeClasses(node: KnowledgeGraphNode) {
  const state = nodeState(node)
  return {
    'is-selected': props.selectedNodeId === node.node_id,
    'is-current': node.is_current,
    'is-blueprint': node.node_type === 'blueprint',
    'is-learned': node.node_type === 'lesson' && state?.current_level && state.current_level !== 'unseen',
  }
}

function roleText(node: KnowledgeGraphNode) {
  return knowledgeNodeTypeText(node.content_role)
}

function isExpanded(unitKey: string) { return props.expandedUnitKeys.includes(unitKey) }
function generationCount(unit: KnowledgeGraphUnit) { return Math.min(5, Math.max(0, unit.blueprint_lesson_count - unit.applied_lesson_count)) }

watch(() => [props.units, props.nodes, props.edges, props.showBlueprint], rebuild, { deep: true })
watch(() => props.nodes.length, () => void nextTick(drawConnections))

onMounted(() => {
  rebuild()
  resizeObserver = new ResizeObserver(drawConnections)
  if (pathContent.value) resizeObserver.observe(pathContent.value)
  window.addEventListener('resize', drawConnections)
})

onBeforeUnmount(() => {
  observer?.disconnect()
  resizeObserver?.disconnect()
  window.removeEventListener('resize', drawConnections)
})

defineExpose({ scrollToNode, scrollToUnit })
</script>
