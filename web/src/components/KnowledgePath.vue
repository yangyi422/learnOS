<template>
  <div class="knowledge-path">
    <section v-for="unit in units" :key="unit.id" class="knowledge-unit">
      <header class="knowledge-unit__header">
        <div><h2>{{ unit.title }}</h2><p>{{ unit.objective || '逐步建立这个知识区域。' }}</p></div>
        <span>{{ unit.applied_lesson_count }} / {{ unit.blueprint_lesson_count }} 已展开</span>
        <button type="button" :aria-expanded="isExpanded(unit.key)" @click="$emit('toggle-unit', unit.key)">{{ isExpanded(unit.key) ? '收起' : `展开 · ${unitNodes(unit.key).length} 个节点` }}</button>
      </header>
      <el-progress :percentage="unit.blueprint_lesson_count ? Math.round(unit.applied_lesson_count / unit.blueprint_lesson_count * 100) : 0" :show-text="false" :stroke-width="4" />
      <div v-if="isExpanded(unit.key) && unitNodes(unit.key).length" class="knowledge-path__nodes">
        <button v-for="node in unitNodes(unit.key)" :key="node.node_id" class="knowledge-node" :class="{ 'knowledge-node--selected': selectedNodeId === node.node_id, 'knowledge-node--blueprint': node.node_type === 'blueprint' }" type="button" @click="$emit('select', node)">
          <span class="knowledge-node__marker" :class="markerClass(node)">{{ marker(node) }}</span>
          <span class="knowledge-node__body"><strong>{{ node.title }}</strong><small>{{ node.node_type === 'blueprint' ? '待生成节点' : knowledgeNodeTypeText(node.content_role) }} · {{ knowledgeDepthText(node.depth_level) }}</small></span>
          <span class="knowledge-node__state">{{ stateText(node) }}</span>
          <span v-if="currentLessonId === node.lesson_id" class="knowledge-node__current">当前</span>
        </button>
      </div>
      <div v-if="isExpanded(unit.key) && unit.needs_expansion && unit.blueprint_unit_id" class="knowledge-unit__empty">
        <p>{{ expandingUnitId === unit.blueprint_unit_id ? '正在展开知识区域…' : '这个区域尚未展开。' }}</p><button type="button" :disabled="expandingUnitId === unit.blueprint_unit_id" :aria-busy="expandingUnitId === unit.blueprint_unit_id" @click="$emit('expand', unit.blueprint_unit_id!)">{{ expandingUnitId === unit.blueprint_unit_id ? '正在展开知识区域…' : '展开这个知识区域' }} <span>→</span></button>
      </div>
      <div v-else-if="isExpanded(unit.key) && unit.needs_generation && unit.blueprint_unit_id" class="knowledge-unit__empty">
        <p>{{ generatingUnitId === unit.blueprint_unit_id ? '正在准备课程草案…' : `下一批最多生成 ${generationCount(unit)} 个可审核草案，不会切换当前课程节点。` }}</p><button type="button" :disabled="generatingUnitId === unit.blueprint_unit_id" :aria-busy="generatingUnitId === unit.blueprint_unit_id" @click="$emit('generate', { id: unit.blueprint_unit_id!, scope: unit.generation_scope })">{{ generatingUnitId === unit.blueprint_unit_id ? '正在准备课程草案…' : `生成 ${generationCount(unit)} 个学习节点` }} <span aria-hidden="true">→</span></button>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import type { CognitiveStateSummary } from '@/types/cognitive'
import type { KnowledgeGraphNode, KnowledgeGraphUnit } from '@/types/knowledgeGraph'
import { knowledgeDepthText, knowledgeNodeTypeText } from '@/utils/knowledgeNodeType'

const props = defineProps<{ units: KnowledgeGraphUnit[]; nodes: KnowledgeGraphNode[]; currentLessonId: number; cognitiveStates: Map<number, CognitiveStateSummary>; selectedNodeId?: string | null; expandedUnitKeys: string[]; expandingUnitId: number; generatingUnitId: number }>()
defineEmits<{ select: [node: KnowledgeGraphNode]; expand: [unitID: number]; generate: [payload: { id: number; scope: 'missing_core' | 'missing_recommended' | '' }]; 'toggle-unit': [unitKey: string] }>()

function unitNodes(unitKey: string) { return props.nodes.filter((node) => node.unit_key === unitKey) }
function isExpanded(unitKey: string) { return props.expandedUnitKeys.includes(unitKey) }
function generationCount(unit: KnowledgeGraphUnit) { return Math.min(5, Math.max(0, unit.blueprint_lesson_count - unit.applied_lesson_count)) }
function nodeState(node: KnowledgeGraphNode) { return node.lesson_id ? props.cognitiveStates.get(node.lesson_id) : undefined }
function marker(node: KnowledgeGraphNode) { return node.node_type === 'blueprint' ? '◇' : nodeState(node)?.current_level === 'transfer' ? '✦' : nodeState(node)?.current_level && nodeState(node)?.current_level !== 'unseen' ? '●' : '○' }
function markerClass(node: KnowledgeGraphNode) { return node.node_type === 'blueprint' ? 'is-blueprint' : nodeState(node)?.current_level && nodeState(node)?.current_level !== 'unseen' ? 'is-learned' : 'is-unseen' }
function stateText(node: KnowledgeGraphNode) { if (node.node_type === 'blueprint') return '尚未生成'; const state = nodeState(node); return state ? ({ unseen: '未接触', exposed: '已接触', recognize: '识别', understand: '理解', apply: '应用', transfer: '迁移' }[state.current_level] ?? state.current_level) : '未接触' }

</script>
