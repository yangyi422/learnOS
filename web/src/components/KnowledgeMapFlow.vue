<template>
  <div class="knowledge-map-flow">
    <VueFlow
      :nodes="nodes"
      :edges="edges"
      :node-types="nodeTypes"
      :default-viewport="{ x: 24, y: 24, zoom: 0.8 }"
      :min-zoom="0.25"
      :max-zoom="1.8"
      :elements-selectable="true"
      :nodes-draggable="true"
      :nodes-connectable="false"
      fit-view-on-init
      :fit-view-on-init-options="{ padding: 0.18 }"
      @node-click="handleNodeClick"
    >
      <Background pattern-color="#cbd5e1" :gap="24" :size="1" />
      <Controls position="bottom-right" :show-interactive="false" />
    </VueFlow>
  </div>
</template>

<script setup lang="ts">
import { VueFlow, useVueFlow, type Edge, type Node, type NodeMouseEvent, type NodeTypesObject } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { KnowledgeFlowNode, KnowledgeUnitLabel } from './knowledgeFlowNodeRegistry'
import type { KnowledgeGraphNode } from '@/types/knowledgeGraph'

defineProps<{
  nodes: Node[]
  edges: Edge[]
}>()

const emit = defineEmits<{ select: [node: KnowledgeGraphNode] }>()
const nodeTypes = { knowledge: KnowledgeFlowNode, 'unit-label': KnowledgeUnitLabel } as unknown as NodeTypesObject
const { fitView } = useVueFlow()

function handleNodeClick({ node }: NodeMouseEvent) {
  const graphNode = node.data?.graphNode as KnowledgeGraphNode | undefined
  if (graphNode) emit('select', graphNode)
}

defineExpose({ fitView })
</script>

<style>
@import '@vue-flow/core/dist/style.css';
@import '@vue-flow/core/dist/theme-default.css';
</style>
