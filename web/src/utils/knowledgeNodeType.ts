export const knowledgeNodeTypes = ['foundation', 'core', 'deepening', 'application', 'extension'] as const

export type KnowledgeNodeType = typeof knowledgeNodeTypes[number]

const labels: Record<KnowledgeNodeType, string> = {
  foundation: '基础',
  core: '核心',
  deepening: '深化',
  application: '应用',
  extension: '扩展',
}

export function knowledgeNodeTypeText(type: KnowledgeNodeType | string): string {
  return labels[type as KnowledgeNodeType] ?? '知识节点'
}

export function isKnowledgeNodeType(value: string): value is KnowledgeNodeType {
  return knowledgeNodeTypes.includes(value as KnowledgeNodeType)
}

export function knowledgeDepthText(depth: number): string {
  return ({ 1: '入门层', 2: '基础层', 3: '进阶层', 4: '深入层', 5: '专题层' } as Record<number, string>)[depth] ?? `第 ${depth} 层`
}
