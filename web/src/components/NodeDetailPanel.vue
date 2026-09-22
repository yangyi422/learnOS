<template>
  <aside class="node-detail-panel">
    <div class="node-detail-panel__header">
      <span class="eyebrow">节点详情</span>
      <button v-if="node" type="button" @click="$emit('clear')">清除选择</button>
    </div>

    <el-empty v-if="!node" description="选择一个节点查看它从哪里来、会通向哪里" />
    <div v-else v-loading="relationsLoading">
      <div class="node-detail-panel__eyebrow" :class="{ 'is-blueprint': node.node_type === 'blueprint' }">
        {{ node.node_type === 'blueprint' ? '待生成节点' : node.is_current ? '当前 LESSON' : '已生成 LESSON' }}
      </div>
      <h2>{{ node.title }}</h2>
      <p class="muted-text">{{ roleText }} · {{ knowledgeDepthText(node.depth_level) }}（知识由浅入深的第 {{ node.depth_level }} 层）</p>
      <p v-if="node.summary" class="node-detail-panel__summary">{{ node.summary }}</p>

      <template v-if="node.node_type === 'blueprint'">
        <p class="node-detail-panel__helper">这个节点还没有生成正式课程内容。先展开知识区域，再生成一批可审核的学习节点。</p>
        <el-button v-if="canExpand && node.blueprint_unit_id" type="primary" :loading="expanding" :disabled="expanding" @click="$emit('expand', node.blueprint_unit_id)">{{ expanding ? '正在展开知识区域…' : '展开知识区域' }}</el-button>
        <el-button v-if="canGenerate && node.blueprint_unit_id" text :loading="generating" :disabled="generating" @click="$emit('generate', { id: node.blueprint_unit_id, scope: generationScope })">{{ generating ? '正在准备课程草案…' : '继续生成节点' }}</el-button>
        <RelationGroup title="前置知识" :items="blueprintRelations?.prerequisites ?? []" clickable @select="$emit('relation-select', $event)" />
        <RelationGroup title="后续知识" :items="blueprintRelations?.next_lessons ?? []" clickable @select="$emit('relation-select', $event)" />
        <RelationGroup title="深化内容" :items="blueprintRelations?.extensions ?? []" clickable @select="$emit('relation-select', $event)" />
        <RelationGroup title="应用场景" :items="blueprintRelations?.applications ?? []" clickable @select="$emit('relation-select', $event)" />
        <RelationGroup title="相关知识" :items="blueprintRelations?.related ?? []" clickable @select="$emit('relation-select', $event)" />
      </template>
      <template v-else>
        <el-button type="primary" @click="$emit('start', node)">{{ currentLessonId === node.lesson_id ? '继续学习' : '切换到此节点' }} <span aria-hidden="true">→</span></el-button>
        <RelationGroup title="前置知识" :items="relations?.prerequisites ?? []" clickable @select="$emit('relation-select', $event)" />
        <RelationGroup title="后续知识" :items="relations?.next_lessons ?? []" clickable @select="$emit('relation-select', $event)" />
        <RelationGroup title="深化内容" :items="relations?.extensions ?? []" clickable @select="$emit('relation-select', $event)" />
        <RelationGroup title="应用场景" :items="relations?.applications ?? []" clickable @select="$emit('relation-select', $event)" />
        <RelationGroup title="相关知识" :items="relations?.related ?? []" clickable @select="$emit('relation-select', $event)" />
        <div v-if="cognitiveDetail" class="node-detail-panel__cognitive">
          <span class="eyebrow">我的理解</span>
          <div class="cognitive-overlay__status"><el-tag type="success">{{ levelText(cognitiveDetail.state.current_level) }}</el-tag><el-tag effect="plain">{{ statusText(cognitiveDetail.state.status) }}</el-tag></div>
          <p>{{ cognitiveDetail.state.understanding_summary || '尚无认知证据，当前为未接触。' }}</p>
          <RelationGroup title="掌握证据" :items="evidenceItems" />
        </div>
      </template>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import RelationGroup from '@/components/RelationGroup.vue'
import type { CognitiveLevel, CognitiveStateDetail, CognitiveStatus } from '@/types/cognitive'
import type { KnowledgeGraphNode, LessonRelationLesson, LessonRelations } from '@/types/knowledgeGraph'
import { knowledgeDepthText, knowledgeNodeTypeText } from '@/utils/knowledgeNodeType'

const props = withDefaults(defineProps<{
  node: KnowledgeGraphNode | null
  currentLessonId: number
  relations: LessonRelations | null
  blueprintRelations?: LessonRelations | null
  relationsLoading: boolean
  cognitiveDetail: CognitiveStateDetail | null
  evidenceItems: LessonRelationLesson[]
  canExpand?: boolean
  canGenerate?: boolean
  expanding?: boolean
  generating?: boolean
}>(), { blueprintRelations: null, canExpand: false, canGenerate: false, expanding: false, generating: false })

defineEmits<{
  clear: []
  start: [node: KnowledgeGraphNode]
  'relation-select': [item: LessonRelationLesson]
  expand: [unitID: number]
  generate: [payload: { id: number; scope: 'missing_core' | 'missing_recommended' | '' }]
}>()

const roleText = computed(() => knowledgeNodeTypeText(props.node?.content_role ?? ''))
const generationScope = computed(() => props.node?.generation_scope || 'missing_core')

function levelText(level: CognitiveLevel | string): string {
  return { unseen: '未接触', exposed: '已接触', recognize: '识别', understand: '理解', apply: '应用', transfer: '迁移' }[level] ?? '未知'
}

function statusText(status: CognitiveStatus | string): string {
  return { unknown: '未知', developing: '发展中', stable: '稳定', needs_review: '待复习' }[status] ?? '未知'
}
</script>
