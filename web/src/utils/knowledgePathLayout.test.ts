import { describe, expect, it } from 'vitest'
import { buildKnowledgePathUnits } from './knowledgePathLayout'
import type { KnowledgeGraphEdge, KnowledgeGraphNode, KnowledgeGraphUnit } from '@/types/knowledgeGraph'

const units: KnowledgeGraphUnit[] = [
  {
    key: 'course-unit:10', id: 10, title: '测试区域', objective: '验证视图一致性', sort_order: 1,
    status: 'learning', blueprint_unit_id: 20, expansion_status: 'expanded', blueprint_lesson_count: 2,
    applied_lesson_count: 1, generation_scope: 'missing_core', needs_expansion: false, needs_generation: true,
  },
]

const nodes: KnowledgeGraphNode[] = [
  {
    id: 101, node_id: 'lesson:101', node_type: 'lesson', lesson_id: 101, blueprint_lesson_id: 201,
    blueprint_unit_id: 20, unit_key: 'course-unit:10', unit_id: 10, title: '同一正式节点', summary: '正式摘要',
    importance: 'core', generation_scope: 'missing_core', is_core: true, content_role: 'core', depth_level: 2,
    status: 'learning', node_status: 'learning', is_current: true,
  },
  {
    id: 202, node_id: 'blueprint:202', node_type: 'blueprint', lesson_id: null, blueprint_lesson_id: 202,
    blueprint_unit_id: 20, unit_key: 'course-unit:10', unit_id: 10, title: '同一蓝图节点', summary: '蓝图摘要',
    importance: 'recommended', generation_scope: 'missing_core', is_core: false, content_role: 'extension', depth_level: 3,
    status: 'pending', node_status: 'blueprint', is_current: false,
  },
]

const edges: KnowledgeGraphEdge[] = [
  {
    id: 1, edge_id: 'blueprint-relation:1', source: 'lesson:101', target: 'blueprint:202',
    from_lesson_id: 0, to_lesson_id: 0, from_blueprint_lesson_id: 201, to_blueprint_lesson_id: 202,
    relation_type: 'prerequisite',
  },
]

describe('buildKnowledgePathUnits', () => {
  it('keeps every path node identical to the graph nodes consumed by list view', () => {
    const pathNodes = buildKnowledgePathUnits(units, nodes, edges, true)
      .flatMap((unit) => unit.ranks)
      .flatMap((rank) => rank.nodes)

    expect(pathNodes.map((node) => node.node_id).sort()).toEqual(nodes.map((node) => node.node_id).sort())
    for (const listNode of nodes) {
      expect(pathNodes.find((node) => node.node_id === listNode.node_id)).toEqual(listNode)
    }
  })

  it('hides only blueprint nodes without changing formal lesson data', () => {
    const pathNodes = buildKnowledgePathUnits(units, nodes, edges, false)
      .flatMap((unit) => unit.ranks)
      .flatMap((rank) => rank.nodes)

    expect(pathNodes).toEqual([nodes[0]])
  })
})
