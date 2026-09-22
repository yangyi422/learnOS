import type { CognitiveStateSummary } from '@/types/cognitive'
import type { KnowledgeGraphEdge, KnowledgeGraphNode } from '@/types/knowledgeGraph'

export type KnowledgeGraphFilter = 'all' | 'current_path' | 'generated' | 'pending' | 'unseen' | 'learning' | 'mastered' | 'application'

export function currentPathNodeIDs(nodes: KnowledgeGraphNode[], edges: KnowledgeGraphEdge[], currentLessonID: number): Set<string> {
  const current = nodes.find((node) => node.lesson_id === currentLessonID)?.node_id
  if (!current) return new Set()
  const result = new Set<string>([current])
  const visit = (target: string) => edges.filter((edge) => edge.relation_type === 'prerequisite' && edge.target === target).forEach((edge) => {
    if (!result.has(edge.source)) { result.add(edge.source); visit(edge.source) }
  })
  visit(current)
  return result
}

export function filterKnowledgeNodes(nodes: KnowledgeGraphNode[], edges: KnowledgeGraphEdge[], states: Map<number, CognitiveStateSummary>, currentLessonID: number, filter: KnowledgeGraphFilter): KnowledgeGraphNode[] {
  if (filter === 'all') return nodes
  const currentPath = filter === 'current_path' ? currentPathNodeIDs(nodes, edges, currentLessonID) : new Set<string>()
  return nodes.filter((node) => {
    const level = node.lesson_id ? states.get(node.lesson_id)?.current_level ?? 'unseen' : 'unseen'
    if (filter === 'current_path') return currentPath.has(node.node_id)
    if (filter === 'generated') return node.node_type === 'lesson'
    if (filter === 'pending') return node.node_type === 'blueprint'
    if (filter === 'unseen') return node.node_type === 'lesson' && level === 'unseen'
    if (filter === 'learning') return node.node_type === 'lesson' && ['exposed', 'recognize', 'understand'].includes(level)
    if (filter === 'mastered') return node.node_type === 'lesson' && ['apply', 'transfer'].includes(level)
    return node.content_role === 'application'
  })
}
