import type { KnowledgeNodeType } from '@/utils/knowledgeNodeType'

export type ContentRole = KnowledgeNodeType
export type LessonStatus = 'pending' | 'learning' | 'completed'
export type LessonRelationType = 'prerequisite' | 'extends' | 'application' | 'related'

export interface KnowledgeGraphUnit {
  key: string
  id: number
  title: string
  objective: string
  sort_order: number
  status: 'pending' | 'learning' | 'completed'
  blueprint_unit_id: number | null
  expansion_status: string
  blueprint_lesson_count: number
  applied_lesson_count: number
  generation_scope: 'missing_core' | 'missing_recommended' | ''
  needs_expansion: boolean
  needs_generation: boolean
}

export interface KnowledgeGraphNode {
  id: number
  node_id: string
  node_type: 'lesson' | 'blueprint' | string
  lesson_id: number | null
  blueprint_lesson_id: number | null
  blueprint_unit_id: number | null
  unit_key: string
  unit_id: number
  title: string
  summary: string
  importance: 'core' | 'recommended' | 'optional' | string
  generation_scope: 'missing_core' | 'missing_recommended' | ''
  is_core: boolean
  content_role: ContentRole
  depth_level: number
  status: LessonStatus | string
  node_status: 'blueprint' | string
  is_current: boolean
  cognitive_level?: string
  cognitive_status?: string
}

export interface KnowledgeGraphEdge {
  id: number
  edge_id: string
  source: string
  target: string
  from_lesson_id: number
  to_lesson_id: number
  from_blueprint_lesson_id: number | null
  to_blueprint_lesson_id: number | null
  relation_type: LessonRelationType
}

export interface KnowledgeGraphStats {
  node_count: number
  edge_count: number
  core_node_count: number
  optional_node_count: number
  root_node_count: number
  leaf_node_count: number
  max_depth_level: number
}

export interface KnowledgeGraph {
  course: {
    id: number
    name: string
  }
  units: KnowledgeGraphUnit[]
  nodes: KnowledgeGraphNode[]
  edges: KnowledgeGraphEdge[]
  stats: KnowledgeGraphStats
}

export interface LessonRelationLesson {
  id: number
  title: string
  node_id?: string
}

export interface LessonRelations {
  lesson: LessonRelationLesson
  prerequisites: LessonRelationLesson[]
  next_lessons: LessonRelationLesson[]
  extensions: LessonRelationLesson[]
  applications: LessonRelationLesson[]
  related: LessonRelationLesson[]
}
