import { describe, expect, it } from 'vitest'
import { isKnowledgeNodeType, knowledgeDepthText, knowledgeNodeTypeText, knowledgeNodeTypes } from './knowledgeNodeType'

describe('knowledge node type mapping', () => {
  it('uses one exhaustive enum and label map in every view', () => {
    expect(knowledgeNodeTypes).toEqual(['foundation', 'core', 'deepening', 'application', 'extension'])
    expect(knowledgeNodeTypes.map(knowledgeNodeTypeText)).toEqual(['基础', '核心', '深化', '应用', '扩展'])
  })

  it('does not infer a type from depth or legacy core flags', () => {
    expect(knowledgeNodeTypeText('application')).toBe('应用')
    expect(knowledgeNodeTypeText('unknown')).toBe('知识节点')
    expect(isKnowledgeNodeType('deepening')).toBe(true)
    expect(isKnowledgeNodeType('depth-3')).toBe(false)
  })

  it('explains depth without turning it into a node type', () => {
    expect([1, 2, 3, 4, 5].map(knowledgeDepthText)).toEqual(['入门层', '基础层', '进阶层', '深入层', '专题层'])
  })
})
