import type { KnowledgeGraphEdge, KnowledgeGraphNode, KnowledgeGraphUnit } from '@/types/knowledgeGraph'

export interface KnowledgePathRank {
  rank: number
  nodes: KnowledgeGraphNode[]
}

export interface KnowledgePathUnit {
  unit: KnowledgeGraphUnit
  ranks: KnowledgePathRank[]
}

/**
 * Produces stable topological ranks for the DOM path view. The graph API
 * already validates prerequisite DAGs; the defensive cycle fallback keeps the
 * page renderable if a malformed draft ever reaches the client.
 */
export function buildKnowledgePathUnits(
  units: KnowledgeGraphUnit[],
  sourceNodes: KnowledgeGraphNode[],
  sourceEdges: KnowledgeGraphEdge[],
  showBlueprint: boolean,
): KnowledgePathUnit[] {
  const nodes = sourceNodes.filter((node) => showBlueprint || node.node_type !== 'blueprint')
  const nodeByID = new Map(nodes.map((node) => [node.node_id, node]))
  const prerequisiteEdges = sourceEdges.filter((edge) => edge.relation_type === 'prerequisite' && nodeByID.has(edge.source) && nodeByID.has(edge.target))
  const outgoing = new Map<string, string[]>()
  const incoming = new Map<string, number>()
  const rankByID = new Map<string, number>()

  nodes.forEach((node) => {
    outgoing.set(node.node_id, [])
    incoming.set(node.node_id, 0)
    rankByID.set(node.node_id, 0)
  })
  prerequisiteEdges.forEach((edge) => {
    outgoing.get(edge.source)?.push(edge.target)
    incoming.set(edge.target, (incoming.get(edge.target) ?? 0) + 1)
  })

  const queue = nodes.filter((node) => incoming.get(node.node_id) === 0).map((node) => node.node_id)
  let processed = 0
  while (queue.length) {
    const nodeID = queue.shift()!
    processed += 1
    const currentRank = rankByID.get(nodeID) ?? 0
    for (const targetID of outgoing.get(nodeID) ?? []) {
      rankByID.set(targetID, Math.max(rankByID.get(targetID) ?? 0, currentRank + 1))
      const nextIncoming = (incoming.get(targetID) ?? 0) - 1
      incoming.set(targetID, nextIncoming)
      if (nextIncoming === 0) queue.push(targetID)
    }
  }

  if (processed < nodes.length) {
    nodes.forEach((node) => {
      if ((incoming.get(node.node_id) ?? 0) > 0) rankByID.set(node.node_id, Math.max(rankByID.get(node.node_id) ?? 0, node.depth_level - 1, 0))
    })
  }

  return units.map((unit) => {
    const unitNodes = nodes.filter((node) => node.unit_key === unit.key)
    const ranks = new Map<number, KnowledgeGraphNode[]>()
    unitNodes.forEach((node) => {
      const rank = rankByID.get(node.node_id) ?? 0
      const rankNodes = ranks.get(rank) ?? []
      rankNodes.push(node)
      ranks.set(rank, rankNodes)
    })
    return {
      unit,
      ranks: [...ranks.entries()]
        .sort(([left], [right]) => left - right)
        .map(([rank, rankNodes]) => ({ rank, nodes: rankNodes })),
    }
  })
}

