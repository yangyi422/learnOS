<template>
  <Handle type="target" :position="Position.Left" />
  <button class="knowledge-flow-node" :class="nodeClasses" type="button" @click="$emit('select', data)">
    <span class="knowledge-flow-node__marker">{{ marker }}</span>
    <span class="knowledge-flow-node__content">
      <strong>{{ data.title }}</strong>
      <small>{{ data.node_type === 'blueprint' ? '待生成节点' : `${roleText} · ${levelText}` }}</small>
    </span>
    <span v-if="data.is_current" class="knowledge-flow-node__badge">当前</span>
  </button>
  <Handle type="source" :position="Position.Right" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import type { KnowledgeGraphNode } from '@/types/knowledgeGraph'
import { knowledgeNodeTypeText } from '@/utils/knowledgeNodeType'

const props = defineProps<{
  data: KnowledgeGraphNode
  selected?: boolean
}>()

defineEmits<{ select: [node: KnowledgeGraphNode] }>()

const nodeClasses = computed(() => ({
  'is-selected': props.selected,
  'is-current': props.data.is_current,
  'is-blueprint': props.data.node_type === 'blueprint',
  'is-learned': props.data.node_type === 'lesson' && props.data.cognitive_level && props.data.cognitive_level !== 'unseen',
}))

const marker = computed(() => props.data.node_type === 'blueprint' ? '◇' : props.data.is_current ? '●' : props.data.cognitive_level === 'transfer' ? '✦' : props.data.cognitive_level && props.data.cognitive_level !== 'unseen' ? '●' : '○')
const roleText = computed(() => knowledgeNodeTypeText(props.data.content_role))
const levelText = computed(() => ({ unseen: '未接触', exposed: '已接触', recognize: '识别', understand: '理解', apply: '应用', transfer: '迁移' } as Record<string, string>)[props.data.cognitive_level ?? 'unseen'] ?? '未接触')
</script>
