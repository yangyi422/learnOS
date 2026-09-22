export interface DomainUnit { key: string; title: string; description: string; importance: string }
export interface DomainSkeleton { course: { name: string; description: string }; blueprint: { name: string; domain: string; learning_goal: string; target_depth: string; units: DomainUnit[] }; recommended_starter_unit_keys: string[] }
export interface DomainBlueprintLesson { key: string; title: string; summary: string; importance: string; content_role: string; depth_level: number; assessment_target_level: string }
export interface ExpandedDomainUnit { key: string; lessons: DomainBlueprintLesson[] }
export interface DomainRelation { from_lesson_key: string; to_lesson_key: string; relation_type: string }
export interface DomainStarter { expanded_units: ExpandedDomainUnit[]; relations: DomainRelation[] }
export interface InitialWorldLesson { blueprint_lesson_key: string; title: string; core_question: string; expected_understanding: string; content_role: string; depth_level: number; assessment_target_level: string; is_core: boolean }
export interface InitialWorld { initial_lessons: InitialWorldLesson[]; relations: DomainRelation[]; recommended_first_lesson_key: string }
export interface DomainDraft { id: number; domain_name: string; learning_goal: string; target_depth: string; status: string; generated_by: string; provider: string; model: string; skeleton_prompt_version: string; starter_prompt_version: string; world_prompt_version: string; skeleton_json: string; starter_blueprint_json: string; initial_world_json: string; applied_course_id?: number; created_at: string; updated_at: string; applied_at?: string }
export interface DomainInitializationView { draft: DomainDraft; skeleton?: DomainSkeleton; starter_blueprint?: DomainStarter; initial_world?: InitialWorld }
