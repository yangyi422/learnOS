import { describe, expect, it } from 'vitest'
import type { CognitiveStateSummary } from '@/types/cognitive'
import type { KnowledgeGraphEdge, KnowledgeGraphNode } from '@/types/knowledgeGraph'
import { currentPathNodeIDs, filterKnowledgeNodes } from './knowledgeGraphFilters'

const node = (id: number, role = 'core', type = 'lesson'): KnowledgeGraphNode => ({
  id, node_id: `${type}:${id}`, node_type: type, lesson_id: type === 'lesson' ? id : null,
  blueprint_lesson_id: type === 'blueprint' ? id : null, blueprint_unit_id: 1, unit_key: 'course-unit:1', unit_id: 1,
  title: `N${id}`, summary: '', importance: 'core', generation_scope: '', is_core: true,
  content_role: role as KnowledgeGraphNode['content_role'], depth_level: 2, status: type === 'blueprint' ? 'blueprint' : 'pending', node_status: type === 'blueprint' ? 'blueprint' : 'pending', is_current: id === 3,
})
const nodes = [node(1, 'foundation'), node(2), node(3, 'application'), node(4, 'core', 'blueprint')]
const edges: KnowledgeGraphEdge[] = [
  { id: 1, edge_id: '1', source: 'lesson:1', target: 'lesson:2', from_lesson_id: 1, to_lesson_id: 2, from_blueprint_lesson_id: null, to_blueprint_lesson_id: null, relation_type: 'prerequisite' },
  { id: 2, edge_id: '2', source: 'lesson:2', target: 'lesson:3', from_lesson_id: 2, to_lesson_id: 3, from_blueprint_lesson_id: null, to_blueprint_lesson_id: null, relation_type: 'prerequisite' },
]
const states = new Map<number, CognitiveStateSummary>([
  [1, { lesson_id: 1, current_level: 'transfer', status: 'stable', understanding_summary: '', evidence_count: 2 }],
  [2, { lesson_id: 2, current_level: 'understand', status: 'developing', understanding_summary: '', evidence_count: 1 }],
  [3, { lesson_id: 3, current_level: 'unseen', status: 'unknown', understanding_summary: '', evidence_count: 0 }],
])

describe('knowledge graph filters', () => {
  it('derives the current path only from prerequisite IDs', () => {
    expect([...currentPathNodeIDs(nodes, edges, 3)]).toEqual(['lesson:3', 'lesson:2', 'lesson:1'])
  })

  it('keeps pending, learning, mastered and application meanings separate', () => {
    expect(filterKnowledgeNodes(nodes, edges, states, 3, 'pending').map((item) => item.id)).toEqual([4])
    expect(filterKnowledgeNodes(nodes, edges, states, 3, 'learning').map((item) => item.id)).toEqual([2])
    expect(filterKnowledgeNodes(nodes, edges, states, 3, 'mastered').map((item) => item.id)).toEqual([1])
    expect(filterKnowledgeNodes(nodes, edges, states, 3, 'application').map((item) => item.id)).toEqual([3])
  })
})
