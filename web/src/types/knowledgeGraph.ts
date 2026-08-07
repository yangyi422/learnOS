export type ContentRole = 'foundation' | 'core' | 'application' | 'extension'
export type LessonStatus = 'pending' | 'learning' | 'completed'
export type LessonRelationType = 'prerequisite' | 'extends' | 'application' | 'related'

export interface KnowledgeGraphUnit {
  id: number
  title: string
  objective: string
  sort_order: number
  status: 'pending' | 'learning' | 'completed'
}

export interface KnowledgeGraphNode {
  id: number
  unit_id: number
  title: string
  is_core: boolean
  content_role: ContentRole
  depth_level: number
  status: LessonStatus
}

export interface KnowledgeGraphEdge {
  id: number
  from_lesson_id: number
  to_lesson_id: number
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
}

export interface LessonRelations {
  lesson: LessonRelationLesson
  prerequisites: LessonRelationLesson[]
  next_lessons: LessonRelationLesson[]
  extensions: LessonRelationLesson[]
  applications: LessonRelationLesson[]
  related: LessonRelationLesson[]
}
