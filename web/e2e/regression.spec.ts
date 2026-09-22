import { expect, test, type Page, type Route } from '@playwright/test'
import type { CognitiveStateDetail } from '../src/types/cognitive'

const now = '2026-09-20T08:00:00Z'
const secretAPIKey = 'sk-regression-secret-must-not-render'
const testBackup = { name: 'learnos-20260920-080000.db', path: '/data/backups/learnos-20260920-080000.db', size: 4096, created_at: now }

const courses = [
  {
    id: 1, name: '测试营养学', description: '用于浏览器回归测试的课程。', goal: '建立稳定测试基线',
    status: 'learning', progress: 48, current_unit: '饮水判断', current_unit_id: 10, current_lesson_id: 101,
    generation_status: 'in_progress', generation_progress: 50, learning_status: 'in_progress', coverage_progress: 40,
    mastery_status: 'developing', mastery_progress: 24,
    last_studied_at: now, created_at: now, updated_at: now,
  },
  {
    id: 2, name: '逻辑与科学思维', description: '探索目标领域。', goal: '理解证据',
    status: 'learning', progress: 0, current_unit: '推理基础', current_unit_id: 20, current_lesson_id: 201,
    generation_status: 'ready', generation_progress: 100, learning_status: 'not_started', coverage_progress: 0,
    mastery_status: 'unseen', mastery_progress: 0,
    last_studied_at: null, created_at: now, updated_at: now,
  },
]

const currentLesson = {
  course: { id: 1, name: '测试营养学' },
  unit: { id: 10, title: '饮水判断', objective: '结合多个信号判断补水需求。' },
  lesson: { id: 101, title: '口渴是否可靠', core_question: '只要不口渴，是否说明身体不缺水？', status: 'learning', content_role: 'application', depth_level: 2 },
}

const cognitiveDetail = {
  lesson: { id: 101, title: '口渴是否可靠' },
  state: { current_level: 'understand', status: 'stable', understanding_summary: '能结合情境解释口渴信号。', last_evaluated_at: now },
  evidence: [],
  timeline: [],
}

const graph = {
  course: { id: 1, name: '测试营养学' },
  units: [{
    key: 'course-unit:10', id: 10, title: '饮水判断', objective: '结合多个信号判断补水需求。', sort_order: 1,
    status: 'learning', blueprint_unit_id: 20, expansion_status: 'expanded', blueprint_lesson_count: 2,
    applied_lesson_count: 1, generation_scope: 'missing_recommended', needs_expansion: false, needs_generation: true,
  }, {
    key: 'blueprint-unit:21', id: 21, title: '运动场景', objective: '把补水原则用于运动。', sort_order: 2,
    status: 'pending', blueprint_unit_id: 21, expansion_status: 'expanded', blueprint_lesson_count: 1,
    applied_lesson_count: 0, generation_scope: 'missing_core', needs_expansion: false, needs_generation: true,
  }],
  nodes: [
    {
      id: 101, node_id: 'lesson:101', node_type: 'lesson', lesson_id: 101, blueprint_lesson_id: 301,
      blueprint_unit_id: 20, unit_key: 'course-unit:10', unit_id: 10, title: '口渴是否可靠',
      summary: '同一个正式知识节点的摘要。', importance: 'core', generation_scope: 'missing_recommended',
      is_core: true, content_role: 'application', depth_level: 2, status: 'learning', node_status: 'learning', is_current: true,
    },
    {
      id: 102, node_id: 'lesson:102', node_type: 'lesson', lesson_id: 102, blueprint_lesson_id: 303,
      blueprint_unit_id: 21, unit_key: 'blueprint-unit:21', unit_id: 21, title: '耐力运动补水计划',
      summary: '另一个区域的正式节点。', importance: 'core', generation_scope: '',
      is_core: true, content_role: 'application', depth_level: 3, status: 'pending', node_status: 'pending', is_current: false,
    },
    {
      id: 302, node_id: 'blueprint:302', node_type: 'blueprint', lesson_id: null, blueprint_lesson_id: 302,
      blueprint_unit_id: 20, unit_key: 'course-unit:10', unit_id: 10, title: '运动补水边界',
      summary: '尚未生成的蓝图知识节点。', importance: 'recommended', generation_scope: 'missing_recommended',
      is_core: false, content_role: 'extension', depth_level: 3, status: 'pending', node_status: 'blueprint', is_current: false,
    },
  ],
  edges: [{
    id: 1, edge_id: 'blueprint-relation:1', source: 'lesson:101', target: 'blueprint:302',
    from_lesson_id: 0, to_lesson_id: 0, from_blueprint_lesson_id: 301, to_blueprint_lesson_id: 302,
    relation_type: 'prerequisite',
  }],
  stats: { node_count: 2, edge_count: 1, core_node_count: 1, optional_node_count: 1, root_node_count: 1, leaf_node_count: 1, max_depth_level: 3 },
}

const explorationDirection = {
  id: 501, course_id: 1, context_course_id: 1, context_lesson_id: 101,
  source_domain_id: 2, source_course_id: 2, source_lesson_id: 201, target_course_id: 2, target_lesson_id: 201,
  direction_type: 'cross_domain', title: '如何判断证据可靠性', summary: '把饮水判断连接到证据评价。',
  why_worth_exploring: '帮助检查单一信号是否足够。', score: 80, reason_code: 'cross_course_bridge', reason_data: {},
  status: 'active', generated_by: 'rule', provider: '', model: '', prompt_version: '',
  source_domain: { id: 2, name: '逻辑与科学思维' }, source_course: { id: 2, name: '逻辑与科学思维' },
  source_lesson: { id: 201, title: '如何判断证据可靠性', course_id: 2 },
  target_course: { id: 2, name: '逻辑与科学思维' }, target_lesson: { id: 201, title: '如何判断证据可靠性', course_id: 2 },
  created_at: now, updated_at: now,
}

const explorationQuestion = {
  id: 601, course_id: 1, source_direction_id: 501, source_lesson_id: 101, target_course_id: 2, target_lesson_id: 201,
  question: '口渴信号和证据可靠性有什么联系？', context: '连接两个领域', why_this_question: '检验跨领域判断',
  question_type: 'connect', status: 'open', origin: 'exploration_direction',
  priority: 'normal',
  source_course: { id: 1, name: '测试营养学' }, source_lesson: { id: 101, title: '口渴是否可靠', course_id: 1 },
  target_course: { id: 2, name: '逻辑与科学思维' }, target_lesson: { id: 201, title: '如何判断证据可靠性', course_id: 2 },
  created_at: now, updated_at: now,
}

interface MockState {
  domainCreateAttempts: number
  validDomainDrafts: number
  aiConfigResponses: string[]
  answerAttempts: number
  answerWrites: number
  challengeWrites: number
  currentLessonWrites: number
  savedQuestions: number
  restoreRequests: number
  graphReads: number
  relationReads: number
  cognitiveDetailReads: number
}

async function json(route: Route, data: unknown, status = 200) {
  await route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(data) })
}

async function installAPIMocks(page: Page): Promise<MockState> {
  const state: MockState = {
    domainCreateAttempts: 0,
    validDomainDrafts: 0,
    aiConfigResponses: [],
    answerAttempts: 0,
    answerWrites: 0,
    challengeWrites: 0,
    currentLessonWrites: 0,
    savedQuestions: 0,
    restoreRequests: 0,
    graphReads: 0,
    relationReads: 0,
    cognitiveDetailReads: 0,
  }
  const learningTurns: Record<string, unknown>[] = []
  const answerResponses = new Map<string, Record<string, unknown>>()
  let nextTurnID = 801
  let masteryScore = 0
  let cognitive = JSON.parse(JSON.stringify(cognitiveDetail)) as CognitiveStateDetail
  page.on('response', async (response) => {
    if (response.url().includes('/api/v1/system/ai-config')) {
      state.aiConfigResponses.push(await response.text())
    }
  })
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const path = url.pathname
    const method = request.method()

    if (path === '/api/v1/courses') return json(route, { data: courses })
    if (path === '/api/v1/domains/drafts' && method === 'POST') {
      state.domainCreateAttempts += 1
      const body = request.postDataJSON() as { domain_name?: string; learning_goal?: string; target_depth?: string }
      const valid = (body.domain_name?.trim().length ?? 0) >= 2
        && (body.learning_goal?.trim().length ?? 0) >= 10
        && ['overview', 'foundation', 'systematic'].includes(body.target_depth ?? '')
      if (!valid) return json(route, { error: { code: 'DOMAIN_INPUT_INVALID', message: '请检查领域名称、学习原因和期望深度。', retryable: false } }, 400)
      state.validDomainDrafts += 1
      await new Promise((resolve) => setTimeout(resolve, 120))
      return json(route, { data: {
        draft: { id: 701, domain_name: body.domain_name?.trim(), learning_goal: body.learning_goal?.trim(), target_depth: body.target_depth, status: 'skeleton_confirmed', generated_by: 'rule', provider: '', model: '', skeleton_prompt_version: '', starter_prompt_version: '', world_prompt_version: '', skeleton_json: '{}', starter_blueprint_json: '', initial_world_json: '', created_at: now, updated_at: now },
        skeleton: { course: { name: body.domain_name?.trim(), description: '测试领域地图' }, blueprint: { name: body.domain_name?.trim(), domain: body.domain_name?.trim(), learning_goal: body.learning_goal?.trim(), target_depth: body.target_depth, units: [{ key: 'start', title: '起步区域', description: '测试区域', importance: 'core' }] }, recommended_starter_unit_keys: ['start'] },
      } }, 201)
    }
    if (path === '/api/v1/domains/drafts/701' && method === 'GET') return json(route, { data: {
      draft: { id: 701, domain_name: '系统测试领域', learning_goal: '用完整流程验证领域创建可恢复', target_depth: 'systematic', status: 'skeleton_confirmed', generated_by: 'rule', provider: '', model: '', skeleton_prompt_version: '', starter_prompt_version: '', world_prompt_version: '', skeleton_json: '{}', starter_blueprint_json: '', initial_world_json: '', created_at: now, updated_at: now },
      skeleton: { course: { name: '系统测试领域', description: '测试领域地图' }, blueprint: { name: '系统测试领域', domain: '系统测试领域', learning_goal: '用完整流程验证领域创建可恢复', target_depth: 'systematic', units: [{ key: 'start', title: '起步区域', description: '测试区域', importance: 'core' }] }, recommended_starter_unit_keys: ['start'] },
    } })
    if (path === '/api/v1/domains/drafts/701/expand-starter' && method === 'POST') return json(route, { data: {
      draft: { id: 701, domain_name: '系统测试领域', learning_goal: '用完整流程验证领域创建可恢复', target_depth: 'systematic', status: 'starter_expanded', generated_by: 'mock', provider: 'mock', model: 'domain-test', skeleton_prompt_version: 'v1', starter_prompt_version: 'v1', world_prompt_version: '', skeleton_json: '{}', starter_blueprint_json: '{}', initial_world_json: '', created_at: now, updated_at: now },
      skeleton: { course: { name: '系统测试领域', description: '测试领域地图' }, blueprint: { name: '系统测试领域', domain: '系统测试领域', learning_goal: '完整流程测试', target_depth: 'systematic', units: [{ key: 'start', title: '起步区域', description: '测试区域', importance: 'core' }] }, recommended_starter_unit_keys: ['start'] },
      starter_blueprint: { expanded_units: [{ key: 'start', lessons: [{ key: 'start.first', title: '第一个学习节点', summary: '用于验收完整流程', importance: 'core', content_role: 'foundation', depth_level: 1, assessment_target_level: 'understand' }] }], relations: [] },
    } })
    if (path === '/api/v1/domains/drafts/701/generate-initial-world' && method === 'POST') return json(route, { data: {
      draft: { id: 701, domain_name: '系统测试领域', learning_goal: '用完整流程验证领域创建可恢复', target_depth: 'systematic', status: 'world_ready', generated_by: 'mock', provider: 'mock', model: 'domain-test', skeleton_prompt_version: 'v1', starter_prompt_version: 'v1', world_prompt_version: 'v1', skeleton_json: '{}', starter_blueprint_json: '{}', initial_world_json: '{}', created_at: now, updated_at: now },
      skeleton: { course: { name: '系统测试领域', description: '测试领域地图' }, blueprint: { name: '系统测试领域', domain: '系统测试领域', learning_goal: '完整流程测试', target_depth: 'systematic', units: [{ key: 'start', title: '起步区域', description: '测试区域', importance: 'core' }] }, recommended_starter_unit_keys: ['start'] },
      starter_blueprint: { expanded_units: [{ key: 'start', lessons: [{ key: 'start.first', title: '第一个学习节点', summary: '用于验收完整流程', importance: 'core', content_role: 'foundation', depth_level: 1, assessment_target_level: 'understand' }] }], relations: [] },
      initial_world: { initial_lessons: [{ blueprint_lesson_key: 'start.first', title: '第一个学习节点', core_question: '如何证明这个学习节点已经理解？', expected_understanding: '能用自己的话解释并应用。', content_role: 'foundation', depth_level: 1, assessment_target_level: 'understand', is_core: true }], relations: [], recommended_first_lesson_key: 'start.first' },
    } })
    if (path === '/api/v1/domains/drafts/701/apply' && method === 'POST') return json(route, { data: {
      draft: { id: 701, domain_name: '系统测试领域', learning_goal: '用完整流程验证领域创建可恢复', target_depth: 'systematic', status: 'applied', generated_by: 'mock', provider: 'mock', model: 'domain-test', skeleton_prompt_version: 'v1', starter_prompt_version: 'v1', world_prompt_version: 'v1', skeleton_json: '{}', starter_blueprint_json: '{}', initial_world_json: '{}', applied_course_id: 1, created_at: now, updated_at: now, applied_at: now },
    } })
    if (path === '/api/v1/courses/1/current-lesson' && method === 'GET') return json(route, { data: currentLesson })
    if (path === '/api/v1/courses/1/current-lesson' && method === 'POST') { state.currentLessonWrites += 1; return json(route, { data: currentLesson }) }
    if (path.endsWith('/current-lesson') && method === 'POST') { state.currentLessonWrites += 1; return json(route, { data: currentLesson }) }
    if (path === '/api/v1/courses/1/next-lesson') return json(route, { data: { recommended: null, alternatives: [], reason: '', needs_expansion: false, needs_generation: false, recommended_unit: null } })
    if (path === '/api/v1/courses/1/answers' && method === 'POST') {
      state.answerAttempts += 1
      const body = request.postDataJSON() as { answer?: string; idempotency_key?: string }
      const key = body.idempotency_key ?? ''
      const existing = answerResponses.get(key)
      if (existing) return json(route, { data: existing })
      if (!body.answer?.trim()) return json(route, { error: { code: 'INVALID_ANSWER', message: '请先输入回答。', retryable: false } }, 400)
      await new Promise((resolve) => setTimeout(resolve, 120))
      const turnID = nextTurnID++
      const before = masteryScore
      masteryScore = before === 0 ? 0.72 : (before + 0.82) / 2
      const evidence = [
        { evidence_type: 'concept_explanation', cognitive_level: 'understand' as const, polarity: 'support' as const, description: '说明了口渴只是补水判断信号之一。' },
        { evidence_type: 'boundary_awareness', cognitive_level: 'understand' as const, polarity: 'support' as const, description: '指出高温、运动与年龄会改变信号可靠性。' },
      ]
      const response = {
        turn_id: turnID,
        result: 'mostly_correct',
        feedback: '判断方向正确；下一版可补充体液调节机制。',
        explanation: '回答同时给出了结论、依据和适用边界。',
        correct_parts: ['没有把口渴当作唯一依据', '考虑了环境与身体状态'],
        missing_parts: ['体液平衡与口渴信号滞后的机制'],
        misconceptions: [],
        boundary_conditions: ['高温、运动、老年或疾病状态下需要额外判断'],
        mastery_evidence: ['能够解释单一信号的局限'],
        evidence_used: ['回答中提到“结合环境和身体状态”', '回答中否定了“不口渴就一定不缺水”'],
        confidence: 0.86,
        uncertainty: '回答没有展开体液调节机制，因此对机制理解的判断仍有限。',
        recommended_next_action: '补充口渴信号为何可能滞后，再用新场景完成迁移验证。',
        transfer_challenge_eligible: true,
        mastery_score: 0.72,
        mastery_score_before: before,
        mastery_score_after: masteryScore,
        needs_review: false,
        evaluation_source: 'mock',
        provider: 'mock',
        model: 'deterministic-learning-loop',
        prompt_version: 'lesson-evaluation.v4',
        demonstrated_level: 'understand',
        user_understanding_summary: '已能结合情境解释口渴不是唯一的补水依据。',
        cognitive_evidence: evidence,
        cognitive_state: { current_level: 'understand', status: 'stable' },
      }
      const timestamp = new Date(Date.parse(now) + turnID * 1000).toISOString()
      learningTurns.unshift({
        id: turnID,
        lesson_id: 101,
        turn_kind: 'lesson_answer',
        question: currentLesson.lesson.core_question,
        user_answer: body.answer.trim(),
        ...response,
        state_change: { from_level: before ? 'understand' : 'unseen', to_level: 'understand', from_status: before ? 'stable' : 'unknown', to_status: 'stable', reason: '结构化评价形成了理解与边界证据。' },
        created_at: timestamp,
      })
      cognitive = {
        ...cognitive,
        state: { current_level: 'understand', status: 'stable', understanding_summary: response.user_understanding_summary, last_evaluated_at: timestamp },
        evidence: [
          ...cognitive.evidence,
          ...evidence.map((item, index) => ({ id: turnID * 10 + index, course_id: 1, lesson_id: 101, learning_turn_id: turnID, evidence_index: index, ...item, source: 'mock', created_at: timestamp })),
        ],
        timeline: [...cognitive.timeline, { id: turnID, course_id: 1, lesson_id: 101, learning_turn_id: turnID, from_level: before ? 'understand' : 'unseen', to_level: 'understand', from_status: before ? 'stable' : 'unknown', to_status: 'stable', understanding_summary: response.user_understanding_summary, reason: '结构化评价形成了理解与边界证据。', created_at: timestamp }],
      }
      answerResponses.set(key, response)
      state.answerWrites += 1
      return json(route, { data: response })
    }
    if (path === '/api/v1/courses/1/learning-turns') return json(route, { data: learningTurns })
    if (path === '/api/v1/courses/1/lessons/101/challenges' && method === 'POST') {
      state.challengeWrites += 1
      return json(route, { data: {
        id: 901, course_id: 1, lesson_id: 101, challenge_type: 'transfer', target_misconception_id: null,
        prompt: '一次冬季长途飞行中没有明显口渴，是否仍需要主动补水？',
        scenario_context: '环境干燥、活动量较低，口渴感不明显。',
        evaluation_criteria: ['能迁移“单一信号不足”的判断', '能结合新场景中的环境因素'],
        why_this_is_transfer: '原问题讨论一般补水判断；这里需要把原则迁移到陌生的长途飞行场景。',
        source_concepts: ['口渴信号', '体液平衡'], target_level: 'transfer', status: 'pending',
        provider: 'mock', model: 'deterministic-challenge', prompt_version: 'challenge-generation.v1', created_at: now,
      } })
    }
    if (path === '/api/v1/courses/1/challenges/901/answers' && method === 'POST') {
      const body = request.postDataJSON() as { answer?: string }
      const before = masteryScore
      masteryScore = Math.min(1, before + 0.1)
      const timestamp = new Date(Date.parse(now) + nextTurnID * 1000).toISOString()
      const turnID = nextTurnID++
      const transferEvidence = { evidence_type: 'transfer', cognitive_level: 'transfer' as const, polarity: 'support' as const, description: '在长途飞行的新场景中应用了多信号判断。' }
      learningTurns.unshift({
        id: turnID, lesson_id: 101, turn_kind: 'transfer_challenge', question: '一次冬季长途飞行中没有明显口渴，是否仍需要主动补水？',
        user_answer: body.answer?.trim() ?? '', result: 'correct', feedback: '成功把原有原则迁移到新的场景。', explanation: '使用了环境干燥与信号不明显两个条件。',
        correct_parts: [], missing_parts: [], misconceptions: [], boundary_conditions: [], mastery_evidence: ['新场景迁移成功'], evidence_used: [], confidence: 1,
        uncertainty: '', recommended_next_action: '', transfer_challenge_eligible: true, mastery_score: 1, mastery_score_before: before, mastery_score_after: masteryScore,
        needs_review: false, evaluation_source: 'mock', provider: 'mock', model: 'deterministic-challenge', prompt_version: 'challenge-evaluator.v1',
        demonstrated_level: 'transfer', user_understanding_summary: '能在陌生场景应用多信号判断。', cognitive_evidence: [transferEvidence],
        state_change: { from_level: 'understand', to_level: 'transfer', from_status: 'stable', to_status: 'stable', reason: '迁移挑战通过。' }, created_at: timestamp,
      })
      cognitive = {
        ...cognitive,
        state: { current_level: 'transfer', status: 'stable', understanding_summary: '能在陌生场景应用多信号判断。', last_evaluated_at: timestamp },
        evidence: [...cognitive.evidence, { id: turnID * 10, course_id: 1, lesson_id: 101, learning_turn_id: turnID, evidence_index: 0, ...transferEvidence, source: 'mock', created_at: timestamp }],
        timeline: [...cognitive.timeline, { id: turnID, course_id: 1, lesson_id: 101, learning_turn_id: turnID, from_level: 'understand', to_level: 'transfer', from_status: 'stable', to_status: 'stable', understanding_summary: '能在陌生场景应用多信号判断。', reason: '迁移挑战通过。', created_at: timestamp }],
      }
      return json(route, { data: {
        challenge_id: 901, attempt_id: 902, learning_turn_id: turnID, result: 'correct', demonstrated_level: 'transfer', passed: true,
        feedback: '成功把原有原则迁移到新的场景。', explanation: '使用了环境干燥与信号不明显两个条件。',
        mastery_score_before: before, mastery_score_after: masteryScore, mastery_impact: '迁移成功，累计掌握度增加 10 个百分点。',
        cognitive_evidence: [transferEvidence], cognitive_state: { current_level: 'transfer', status: 'stable' }, misconception_validation: null,
      } })
    }
    if (path === '/api/v1/courses/1/knowledge-graph') { state.graphReads += 1; return json(route, { data: graph }) }
    if (path === '/api/v1/courses/1/cognitive-states') return json(route, { data: { course_id: 1, states: [{ lesson_id: 101, current_level: 'understand', status: 'stable', understanding_summary: cognitiveDetail.state.understanding_summary, evidence_count: 0 }, { lesson_id: 102, current_level: 'unseen', status: 'unknown', understanding_summary: '', evidence_count: 0 }] } })
    if (path === '/api/v1/courses/1/lessons/101/cognitive-state') { state.cognitiveDetailReads += 1; return json(route, { data: cognitive }) }
    if (path === '/api/v1/courses/1/lessons/101/relations') { state.relationReads += 1; return json(route, { data: { lesson: { id: 101, title: '口渴是否可靠' }, prerequisites: [], next_lessons: [], extensions: [], applications: [], related: [] } }) }
    if (path === '/api/v1/courses/1/lessons/102/relations') return json(route, { data: { lesson: { id: 102, title: '耐力运动补水计划' }, prerequisites: [], next_lessons: [], extensions: [], applications: [], related: [] } })
    if (path === '/api/v1/courses/1/lessons/102/cognitive-state') return json(route, { data: { lesson: { id: 102, title: '耐力运动补水计划' }, state: { current_level: 'unseen', status: 'unknown', understanding_summary: '', last_evaluated_at: null }, evidence: [], timeline: [] } })
    if (path === '/api/v1/courses/1/lessons/101/misconceptions') return json(route, { data: [] })
    if (path === '/api/v1/courses/1/misconception-network') return json(route, { data: { patterns: [], misconceptions: [], edges: [] } })
    if (path === '/api/v1/exploration/radar') return json(route, { data: { directions: [explorationDirection] } })
    if (path === '/api/v1/exploration/questions') return json(route, { data: [explorationQuestion] })
    if (path === '/api/v1/exploration/directions/501/questions' && method === 'POST') { state.savedQuestions += 1; return json(route, { data: explorationQuestion }) }
    if (path === '/api/v1/exploration/directions/501/questions/undo' && method === 'POST') { state.savedQuestions = 0; return json(route, { data: { ...explorationDirection, status: 'active' } }) }
    if (/\/api\/v1\/exploration\/questions\/601\/(start|later|resolve|reopen|archive|priority)/.test(path) && method === 'POST') return json(route, { data: explorationQuestion })
    if (path === '/api/v1/courses/1/curriculum/coverage') {
      const blueprintLesson = {
        id: 301, blueprint_id: 30, blueprint_unit_id: 20, key: 'hydration.thirst', title: '口渴是否可靠',
        summary: '同一个正式知识节点的摘要。', importance: 'core', content_role: 'core', depth_level: 2,
        assessment_target_level: 'understand', sort_order: 1, applied_lesson_id: 101,
      }
      return json(route, { data: {
        blueprint: { id: 30, course_id: 1, name: '测试蓝图', domain: '营养学', description: '', learning_goal: '', audience: '', target_depth: 'systematic', version: 'v0', status: 'active', created_by: 'seed', grounding_status: 'provisional' },
        units: [{ unit: { id: 20, blueprint_id: 30, key: 'hydration', title: '饮水判断', description: '', sort_order: 1, importance: 'core', expansion_status: 'expanded' }, lessons: [{ blueprint_lesson: blueprintLesson, state: 'covered', applied_lesson: { id: 101, title: '口渴是否可靠', unit_id: 10, unit_title: '饮水判断' } }], core_total: 1, core_covered: 1, recommended_total: 0, recommended_covered: 0, optional_total: 0, optional_covered: 0 }],
        metrics: { core_total: 1, core_covered: 1, core_missing: 0, recommended_total: 0, recommended_covered: 0, optional_total: 0, optional_covered: 0, core_coverage_percent: 100, overall_coverage_percent: 100 },
        missing_core: [],
      } })
    }
    if (path === '/api/v1/courses/1/curriculum/drafts') return json(route, { data: [] })
    if (path === '/api/v1/system/diagnostics') return json(route, { data: { app_version: '0.1.0', environment: 'test', schema_version: 10, database_status: { status: 'ok' }, database_path: '/tmp/test.db', database_size: 1024, last_backup: testBackup, ai_provider: 'deepseek', ai_model: 'test-model', ai_timeouts_seconds: { evaluation: 45 }, course_count: 2, lesson_count: 2, learning_turn_count: 0, cognitive_state_count: 1, generated_at: now } })
    if (path === '/api/v1/system/ai-config' && method === 'GET') return json(route, { data: { provider: 'deepseek', api_key_configured: true, base_url: 'https://example.test', model: 'test-model', effective_provider: 'deepseek', effective_model: 'test-model', last_successful_call_at: now, updated_at: now } })
    if (path === '/api/v1/system/ai-config' && method === 'PATCH') return json(route, { data: { provider: 'deepseek', api_key_configured: true, base_url: 'https://example.test', model: 'test-model', effective_provider: 'deepseek', effective_model: 'test-model', last_successful_call_at: now, updated_at: now } })
    if (path === '/api/v1/system/ai-config/test' && method === 'POST') return json(route, { data: { status: 'ok', provider: 'deepseek', model: 'test-model', checked_at: now, latency_ms: 18 } })
    if (path === '/api/v1/system/backups' && method === 'GET') return json(route, { data: [testBackup] })
    if (path === '/api/v1/system/backups' && method === 'POST') return json(route, { data: testBackup })
    if (path === '/api/v1/system/backups/restore' && method === 'POST') { state.restoreRequests += 1; return json(route, { data: { backup: testBackup, requested_at: now, restart_pending: true } }, 202) }
    if (path === '/api/v1/system/export' && method === 'POST') return route.fulfill({ status: 200, contentType: 'application/json', headers: { 'Content-Disposition': 'attachment; filename="learnos-export.json"' }, body: JSON.stringify({ export_version: '1', courses }) })
    if (path === '/api/v1/system/consistency') return json(route, { data: { healthy: true, warnings: [], errors: [] } })

    return json(route, { error: `unmocked endpoint: ${method} ${path}` }, 404)
  })
  return state
}

test('空的新建领域表单不会产生有效数据', async ({ page }) => {
  const state = await installAPIMocks(page)
  await page.goto('/domains/new')
  await page.getByRole('button', { name: '建立领域地图' }).click()

  await expect(page.getByText('请输入领域名称')).toBeVisible()
  await expect(page.getByText('请输入为什么想学这个领域')).toBeVisible()
  const nameInput = page.getByRole('textbox', { name: '领域名称' })
  await expect(nameInput).toHaveValue('')
  await expect(nameInput).toHaveAttribute('aria-invalid', 'true')
  await expect(nameInput).toHaveAttribute('aria-describedby', 'domain-name-error')
  expect(state.domainCreateAttempts).toBe(0)
  expect(state.validDomainDrafts).toBe(0)
})

test('新建领域会去除空白并阻止连续点击重复创建', async ({ page }) => {
  const state = await installAPIMocks(page)
  await page.goto('/domains/new')
  await page.getByRole('textbox', { name: '领域名称' }).fill('  心理学  ')
  await page.getByRole('textbox', { name: '为什么想学' }).fill('  系统理解心理学并应用于日常生活  ')

  const submit = page.getByRole('button', { name: '建立领域地图' })
  await submit.evaluate((button) => {
    const element = button as HTMLButtonElement
    element.click()
    element.click()
  })

  await expect(page.getByRole('button', { name: '正在建立领域地图…' })).toBeDisabled()
  await expect(page.getByText('领域地图已建立')).toBeVisible()
  expect(state.domainCreateAttempts).toBe(1)
  expect(state.validDomainDrafts).toBe(1)
})

test('创建领域完整流程可从地图进入正式课程', async ({ page }) => {
  await installAPIMocks(page)
  await page.goto('/domains/new')
  await page.getByRole('textbox', { name: '领域名称' }).fill('系统测试领域')
  await page.getByRole('textbox', { name: '为什么想学' }).fill('用完整流程验证领域创建可恢复')
  await page.getByRole('button', { name: '建立领域地图' }).click()

  await expect(page.getByText('领域地图已建立')).toBeVisible()
  await page.reload()
  await expect(page.getByText('领域地图已建立')).toBeVisible()
  await expect(page.getByRole('checkbox', { name: '起步区域' })).toBeChecked()
  await page.getByRole('button', { name: '继续展开起步区域' }).click()
  await expect(page.getByText('第一个学习节点')).toBeVisible()
  await page.getByRole('button', { name: '准备第一批课程' }).click()
  await expect(page.getByText('已准备 1 个正式课程节点草案。')).toBeVisible()
  await page.getByRole('button', { name: '确认并创建学习领域' }).click()

  await expect(page).toHaveURL(/\/courses\/1\/archive$/)
  expect(await page.evaluate(() => localStorage.getItem('learnos:active-domain-draft'))).toBeNull()
})

test('领域接口 400 显示可理解提示而不是 HTTP 状态码', async ({ page }) => {
  await installAPIMocks(page)
  await page.route('**/api/v1/domains/drafts', (route) => json(route, {
    error: { code: 'DOMAIN_INPUT_INVALID', message: '请检查领域名称、学习原因和期望深度。', retryable: false },
  }, 400))
  await page.goto('/domains/new')
  await page.getByRole('textbox', { name: '领域名称' }).fill('心理学')
  await page.getByRole('textbox', { name: '为什么想学' }).fill('系统理解心理学并应用于日常生活')
  await page.getByRole('button', { name: '建立领域地图' }).click()

  await expect(page.getByText('请检查领域名称、学习原因和期望深度。')).toBeVisible()
  await expect(page.getByText(/HTTP 400/)).toHaveCount(0)
})

test('移动端导航可以打开、关闭和切换页面', async ({ page }) => {
  await installAPIMocks(page)
  await page.setViewportSize({ width: 375, height: 812 })
  await page.goto('/')

  await page.getByLabel('打开导航').click()
  const drawer = page.locator('.mobile-nav-drawer')
  await expect(drawer).toBeVisible()
  await expect(drawer.getByRole('link', { name: '学习首页' })).toBeFocused()
  expect(await page.evaluate(() => document.body.classList.contains('el-popup-parent--hidden'))).toBe(false)
  await page.keyboard.press('Shift+Tab')
  expect(await page.evaluate(() => Boolean(document.activeElement?.closest('.mobile-nav-drawer')))).toBe(true)
  await page.keyboard.press('Escape')
  await expect(drawer).toBeHidden()
  await expect(page.getByLabel('打开导航')).toBeFocused()

  await page.getByLabel('打开导航').click()
  await expect(drawer).toBeVisible()
  await page.locator('.el-overlay').click({ position: { x: 360, y: 400 } })
  await expect(drawer).toBeHidden()
  await expect(page.getByLabel('打开导航')).toBeFocused()

  await page.getByLabel('打开导航').click()
  await drawer.getByRole('link', { name: '课程档案' }).click()
  await expect(page).toHaveURL(/\/courses$/)
  await expect(drawer).toBeHidden()
  const pageHeading = page.getByRole('heading', { name: '你的知识世界' })
  await expect(pageHeading).toBeVisible()
  await expect(pageHeading).toBeFocused()
  await expect(page.locator('.el-overlay')).toBeHidden()
  expect(await page.evaluate(() => document.body.classList.contains('el-popup-parent--hidden'))).toBe(false)
})

test('同一知识节点在路径视图和列表视图中数据一致', async ({ page }) => {
  const state = await installAPIMocks(page)
  await page.setViewportSize({ width: 1440, height: 900 })
  await page.goto('/courses/1/map')

  await page.locator('.knowledge-tree-node', { hasText: '口渴是否可靠' }).click()
  await expect(page.locator('.knowledge-tree-node', { hasText: '口渴是否可靠' })).toContainText('应用')
  const detail = page.locator('.node-detail-panel--desktop')
  await expect(detail).toContainText('同一个正式知识节点的摘要。')
  await expect(detail).toContainText('我的理解')
  const pathDetail = await detail.innerText()

  await page.getByRole('button', { name: '列表视图' }).click()
  await page.locator('.knowledge-node', { hasText: '口渴是否可靠' }).click()
  await expect(page.locator('.knowledge-node', { hasText: '口渴是否可靠' })).toContainText('应用')
  const listDetail = await page.locator('.knowledge-workspace .node-detail-panel').innerText()
  expect(listDetail).toBe(pathDetail)
  expect(state.graphReads).toBe(1)
  expect(state.relationReads).toBe(1)
  expect(state.cognitiveDetailReads).toBe(1)
})

test('知识区域默认只展开当前区域，查看节点不会切换 Lesson', async ({ page }) => {
  const state = await installAPIMocks(page)
  await page.goto('/courses/1/map')

  await expect(page.locator('.knowledge-tree-unit', { hasText: '饮水判断' }).locator('.knowledge-tree-node')).toHaveCount(2)
  const otherRegion = page.locator('.knowledge-tree-unit', { hasText: '运动场景' })
  await expect(otherRegion.locator('.knowledge-tree-node')).toHaveCount(0)
  await otherRegion.getByRole('button', { name: /展开/ }).click()
  await otherRegion.locator('.knowledge-tree-node', { hasText: '耐力运动补水计划' }).click()
  expect(state.currentLessonWrites).toBe(0)
  await expect(page.locator('.node-detail-panel--desktop').getByRole('button', { name: '切换到此节点' })).toBeVisible()
})

test('探索推荐显示候选节点真实来源而不是当前筛选领域', async ({ page }) => {
  await installAPIMocks(page)
  await page.goto('/exploration/questions')

  const card = page.locator('.discovery-card', { hasText: explorationDirection.title })
  await expect(card).toContainText('来自')
  await expect(card).toContainText(explorationDirection.source_course.name)
  await expect(card).toContainText(explorationDirection.source_lesson.title)
  await expect(card).not.toContainText(courses[0].name)
})

test('探索推荐可以保存到问题池并撤销', async ({ page }) => {
  const state = await installAPIMocks(page)
  await page.goto('/exploration/questions')
  const card = page.locator('.discovery-card', { hasText: explorationDirection.title })
  await card.getByRole('button', { name: '保存到问题池' }).evaluate((button) => {
    const element = button as HTMLButtonElement
    element.click()
    element.click()
  })
  await expect(card.getByRole('button', { name: '已保存' })).toBeVisible()
  expect(state.savedQuestions).toBe(1)
  await card.getByRole('button', { name: '撤销' }).click()
  await expect(card.getByRole('button', { name: '保存到问题池' })).toBeVisible()
  expect(state.savedQuestions).toBe(0)
})

test('问题池长列表按 20 条分段渲染', async ({ page }) => {
  await installAPIMocks(page)
  const manyQuestions = Array.from({ length: 45 }, (_, index) => ({
    ...explorationQuestion,
    id: 700 + index,
    question: `待探索问题 ${index + 1}`,
  }))
  await page.route('**/api/v1/exploration/questions**', (route) => json(route, { data: manyQuestions }))
  await page.goto('/exploration/questions')
  await page.getByRole('tab', { name: '问题池' }).click()

  await expect(page.locator('.exploration-item')).toHaveCount(20)
  await page.getByRole('button', { name: '再显示 20 个问题' }).click()
  await expect(page.locator('.exploration-item')).toHaveCount(40)
  await page.getByRole('button', { name: '再显示 5 个问题' }).click()
  await expect(page.locator('.exploration-item')).toHaveCount(45)
})

test('误区空状态解释数据来源并提供回答入口', async ({ page }) => {
  await installAPIMocks(page)
  await page.goto('/courses/1/misconceptions')
  await expect(page.getByText(/一次 AI 评价只会记录为推测/)).toBeVisible()
  await expect(page.getByRole('button', { name: '去完成一次回答' }).first()).toBeVisible()
})

test('误区推测可由用户确认和纠正', async ({ page }) => {
  await installAPIMocks(page)
  const reviewStatus = new Map<number, string>([[701, 'ai_inferred'], [702, 'ai_inferred']])
  const misconception = (id: number, title: string) => ({
    id, lesson_id: 101, original_understanding: title, correct_understanding: '需要结合多个证据', boundary_notes: '具体情境会影响判断',
    status: 'active', lesson_title: '口渴是否可靠', pattern_keys: ['single_cause'], occurrence_count: 1,
    review_status: reviewStatus.get(id), user_note: '', stable_pattern_evidence: false,
    events: [{ id, event_type: 'observed', notes: '来自一次回答的 AI 推测', created_at: now, user_answer: title }],
  })
  await page.route('**/api/v1/courses/1/misconception-network', (route) => json(route, { data: {
    patterns: [], misconceptions: [misconception(701, '只看单一信号'), misconception(702, '忽略情境边界')], edges: [],
  } }))
  await page.route('**/api/v1/courses/1/misconceptions/*/review', async (route) => {
    const id = Number(new URL(route.request().url()).pathname.split('/').at(-2))
    const body = route.request().postDataJSON() as { action: string }
    reviewStatus.set(id, body.action === 'confirm' ? 'user_confirmed' : body.action === 'correct' ? 'user_corrected' : 'ignored')
    return json(route, { data: misconception(id, id === 701 ? '只看单一信号' : '忽略情境边界') })
  })
  await page.goto('/courses/1/misconceptions')

  const confirmed = page.locator('.misconception-network-item', { hasText: '只看单一信号' })
  await confirmed.getByRole('button', { name: '确认推测' }).click()
  await expect(confirmed.getByText('用户确认', { exact: true })).toBeVisible()

  const corrected = page.locator('.misconception-network-item', { hasText: '忽略情境边界' })
  page.once('dialog', (dialog) => dialog.accept('应该补充环境和个体差异'))
  await corrected.getByRole('button', { name: '纠正推测' }).click()
  await expect(corrected.getByText('用户已纠正', { exact: true })).toBeVisible()
})

test('探索空间选择器在桌面和移动端显示当前领域名称', async ({ page }) => {
  await installAPIMocks(page)
  for (const width of [375, 768, 1440]) {
    await page.setViewportSize({ width, height: 900 })
    await page.goto('/exploration/questions')
    await expect(page.getByText('当前探索领域', { exact: true })).toBeVisible()
    await expect(page.getByRole('combobox', { name: '当前探索领域' })).toBeVisible()
    const selected = page.locator('.exploration-course-select .el-select__placeholder')
    await expect(selected).toContainText(courses[0].name)
    expect((await page.locator('.exploration-course-select').boundingBox())?.width).toBeGreaterThan(160)
  }
})

test('切换领域后旧推荐响应不会覆盖新领域结果', async ({ page }) => {
  await installAPIMocks(page)
  const freshDirection = {
    ...explorationDirection,
    id: 502,
    course_id: 2,
    context_course_id: 2,
    context_lesson_id: 201,
    source_domain_id: 1,
    source_course_id: 1,
    source_lesson_id: 101,
    target_course_id: 1,
    target_lesson_id: 101,
    title: '补水判断中的证据边界',
    source_domain: { id: 1, name: '测试营养学' },
    source_course: { id: 1, name: '测试营养学' },
    source_lesson: { id: 101, title: '口渴是否可靠', course_id: 1 },
    target_course: { id: 1, name: '测试营养学' },
    target_lesson: { id: 101, title: '口渴是否可靠', course_id: 1 },
  }
  await page.route('**/api/v1/exploration/radar**', async (route) => {
    const selectedCourseID = Number(new URL(route.request().url()).searchParams.get('course_id'))
    await new Promise((resolve) => setTimeout(resolve, selectedCourseID === 1 ? 250 : 150))
    return json(route, { data: { directions: [selectedCourseID === 2 ? freshDirection : explorationDirection] } })
  })

  await page.goto('/exploration/questions')
  await page.locator('.exploration-course-select .el-select__wrapper').click()
  await page.getByRole('option', { name: courses[1].name }).click()

  await expect(page.getByRole('status')).toHaveText('切换中…')
  await expect(page.locator('.discovery-card')).toHaveCount(0)
  await expect(page.locator('.discovery-card', { hasText: freshDirection.title })).toBeVisible()
  await page.waitForTimeout(350)
  await expect(page.locator('.discovery-card', { hasText: explorationDirection.title })).toHaveCount(0)
})

test('课程卡片分别展示生成、覆盖和掌握进度', async ({ page }) => {
  await installAPIMocks(page)
  await page.goto('/courses')

  const card = page.locator('.domain-tile', { hasText: courses[0].name })
  await expect(card).toContainText('课程生成度')
  await expect(card).toContainText('学习覆盖度')
  await expect(card).toContainText('理解掌握度')
})

test('刷新后可恢复并轮询正在生成的课程草案', async ({ page }) => {
  await installAPIMocks(page)
  let draftReads = 0
  const draft = {
    id: 990, course_id: 1, blueprint_id: 30, blueprint_unit_id: 20, pending_key: 'course:1:unit:20',
    title: '扩充知识区域：饮水判断', summary: '草案内容', status: 'generating', generated_by: 'ai',
    provider: 'mock', model: 'mock', prompt_version: 'test', change_set: '{}', created_at: now, updated_at: now, applied_at: null,
  }
  await page.route('**/api/v1/courses/1/curriculum/drafts', (route) => {
    draftReads += 1
    return json(route, { data: [{
      draft: { ...draft, status: draftReads === 1 ? 'generating' : 'draft' },
      change_set: { new_units: [], new_lessons: draftReads === 1 ? [] : [{ temp_key: 'lesson-1', blueprint_lesson_key: 'hydration.more', title: '新学习节点', summary: '已生成', unit_temp_key: 'unit-1' }], new_relations: [], blueprint_mappings: [] },
    }] })
  })
  await page.goto('/courses/1/archive?tab=drafts')

  await expect(page.getByText('生成中', { exact: true })).toBeVisible()
  await expect(page.getByText('草案', { exact: true })).toBeVisible({ timeout: 4000 })
  expect(draftReads).toBeGreaterThanOrEqual(2)
})

test('首页当前课程与学习页当前 Lesson 一致', async ({ page }) => {
  await installAPIMocks(page)
  await page.goto('/')
  await expect(page.locator('#current-focus-title')).toHaveText(currentLesson.lesson.title)
  await page.getByRole('button', { name: /继续学习/ }).click()
  await expect(page).toHaveURL(/\/courses\/1\/learn$/)
  await expect(page.locator('.learning-context-header h1')).toHaveText(currentLesson.lesson.title)
})

test('学习回答禁止空提交，并自动保存、恢复草稿和提示离开', async ({ page }) => {
  const state = await installAPIMocks(page)
  await page.goto('/courses/1/learn')

  const editor = page.locator('#lesson-answer')
  const submit = page.getByRole('button', { name: /提交回答/ })
  await expect(submit).toBeDisabled()
  await editor.fill('   ')
  await expect(submit).toBeDisabled()
  expect(state.answerAttempts).toBe(0)

  const draft = '不能只看口渴，还应结合环境、活动量和身体状态。'
  await editor.fill(draft)
  await expect(page.getByText(/草稿已自动保存于/)).toBeVisible()

  page.once('dialog', async (dialog) => {
    expect(dialog.message()).toContain('尚未提交的回答')
    await dialog.dismiss()
  })
  await page.getByRole('button', { name: '学习首页' }).click()
  await expect(page).toHaveURL(/\/courses\/1\/learn$/)
  await expect(editor).toHaveValue(draft)

  page.once('dialog', (dialog) => dialog.accept())
  await page.reload()
  await expect(editor).toHaveValue(draft)
  await expect(page.getByText(/已恢复 .* 的本机草稿/)).toBeVisible()
})

test('回答—评价—修正—迁移—掌握证据形成可追溯闭环', async ({ page }) => {
  const state = await installAPIMocks(page)
  await page.goto('/courses/1/learn')

  const editor = page.locator('#lesson-answer')
  await editor.fill('不口渴不代表一定不缺水，还要结合高温、运动、年龄和身体状态判断。')
  const submit = page.getByRole('button', { name: /提交回答/ })
  await submit.evaluate((button) => {
    const element = button as HTMLButtonElement
    element.click()
    element.click()
  })

  await expect(page.getByRole('heading', { name: '本次学习反馈' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '评价使用的证据' })).toBeVisible()
  await expect(page.getByText('本次结论可信度：86%')).toBeVisible()
  await expect(page.getByText(/机制理解的判断仍有限/)).toBeVisible()
  await expect(page.getByText(/补充口渴信号为何可能滞后/)).toBeVisible()
  expect(state.answerAttempts).toBe(1)
  expect(state.answerWrites).toBe(1)
  await expect(editor).toHaveValue('')
  expect(await page.evaluate(() => localStorage.getItem('learnos:answer-draft:1:101'))).toBeNull()

  await expect(page.getByRole('heading', { name: /当前学习状态为什么是“初步理解”/ })).toBeVisible()
  await expect(page.getByText('说明了口渴只是补水判断信号之一。')).toBeVisible()
  await page.getByText(/查看状态判断时间线/).click()
  await expect(page.getByText('结构化评价形成了理解与边界证据。')).toBeVisible()

  await page.getByRole('button', { name: '根据缺口修正回答' }).click()
  await expect(editor).toBeFocused()
  await editor.fill('修正版：口渴可能滞后于体液变化，因此还要结合尿色、出汗、环境和个体状态。')
  await page.keyboard.press(process.platform === 'darwin' ? 'Meta+Enter' : 'Control+Enter')
  await expect.poll(() => state.answerWrites).toBe(2)

  await page.getByRole('button', { name: '生成迁移挑战' }).click()
  await expect(page.getByText(/原问题讨论一般补水判断/)).toBeVisible()
  await expect(page.getByText('迁移挑战（独立记录）')).toBeVisible()
  await page.getByPlaceholder('回答这个新挑战……').fill('仍需要结合机舱干燥、飞行时长和个人状态主动补水，不能只依赖口渴。')
  await page.getByRole('button', { name: '提交验证' }).click()

  await expect(page.getByText('验证通过')).toBeVisible()
  await expect(page.locator('.challenge-result').getByText(/累计掌握度增加 10 个百分点/)).toBeVisible()
  await expect(page.getByRole('heading', { name: /当前学习状态为什么是“掌握”/ })).toBeVisible()
  expect(state.challengeWrites).toBe(1)

  await page.getByRole('button', { name: /展开全部/ }).click()
  await expect(page.getByText('回答版本 1')).toBeVisible()
  await expect(page.getByText('回答版本 2')).toBeVisible()
  await expect(page.getByText('迁移挑战通过', { exact: true })).toBeVisible()
  await expect(page.getByText('迁移挑战通过。')).toBeVisible()
})

test('迁移挑战禁用时显示明确解锁条件', async ({ page }) => {
  await installAPIMocks(page)
  await page.route('**/api/v1/courses/1/lessons/101/cognitive-state', (route) => json(route, { data: {
    ...cognitiveDetail,
    state: { current_level: 'unseen', status: 'unknown', understanding_summary: '', last_evaluated_at: null },
  } }))
  await page.goto('/courses/1/learn')

  await expect(page.getByText('解锁条件：核心问题达到“初步理解”。当前状态为“未接触”。')).toBeVisible()
  await expect(page.getByRole('button', { name: '生成迁移挑战' })).toBeDisabled()
})

test('API Key 不出现在响应、日志或保存后的页面明文中', async ({ page }) => {
  const consoleMessages: string[] = []
  page.on('console', (message) => consoleMessages.push(message.text()))
  const state = await installAPIMocks(page)
  await page.goto('/settings')

  const input = page.getByRole('textbox', { name: 'API Key' })
  await input.fill(secretAPIKey)
  await page.getByRole('button', { name: '保存 AI 配置' }).click()
  await expect(input).toHaveValue('')
  await expect(page.getByText('已配置', { exact: true })).toBeVisible()

  expect(await page.locator('body').innerText()).not.toContain(secretAPIKey)
  expect(state.aiConfigResponses.join('\n')).not.toContain(secretAPIKey)
  expect(consoleMessages.join('\n')).not.toContain(secretAPIKey)
})

test('导航使用语义链接并标记当前页面，键盘可完成切换', async ({ page }) => {
  await installAPIMocks(page)
  await page.setViewportSize({ width: 1440, height: 900 })
  await page.goto('/')

  const navigation = page.getByRole('navigation', { name: '主要导航' })
  const homeLink = navigation.getByRole('link', { name: '学习首页' })
  await expect(homeLink).toHaveAttribute('aria-current', 'page')
  const settingsLink = navigation.getByRole('link', { name: '系统设置' })
  await settingsLink.focus()
  await page.keyboard.press('Enter')
  await expect(page).toHaveURL(/\/settings$/)
  await expect(settingsLink).toHaveAttribute('aria-current', 'page')
})

test('系统设置展示生效模型、连接测试、高级超时与安全恢复确认', async ({ page }) => {
  const state = await installAPIMocks(page)
  await page.goto('/settings')

  await expect(page.getByLabel('实际生效的 AI 配置')).toContainText('test-model')
  await expect(page.getByText('最近成功调用')).toBeVisible()
  await page.getByRole('button', { name: '测试 AI 连接' }).click()
  await expect(page.locator('.el-message__content')).toHaveText('AI 连接成功，响应时间 18 毫秒')
  await expect(page.locator('[aria-live="polite"]')).toContainText('AI 连接成功，响应时间 18 毫秒')

  await page.getByRole('button', { name: /高级设置：AI 超时/ }).click()
  await expect(page.getByText('回答评价')).toBeVisible()
  await expect(page.getByText('45 秒')).toBeVisible()

  await page.getByRole('button', { name: '创建数据库备份' }).click()
  await expect(page.locator('.sr-only[aria-live="polite"]')).toHaveText(`备份已创建：${testBackup.name}`)
  await expect(page.getByText(`保存位置：${testBackup.path}`)).toBeVisible()
  await page.getByRole('button', { name: '检查数据一致性' }).click()
  await expect(page.getByText('未发现数据一致性问题')).toBeVisible()
  const downloadPromise = page.waitForEvent('download')
  await page.getByRole('button', { name: '导出学习数据为 JSON' }).click()
  expect((await downloadPromise).suggestedFilename()).toBe('learnos-export.json')

  await page.locator('.restore-controls .el-select__wrapper').click()
  await page.getByRole('option', { name: new RegExp(testBackup.name) }).click()
  await page.getByRole('button', { name: '确认恢复所选备份' }).click()
  const dialog = page.getByRole('dialog', { name: '确认恢复数据库' })
  await dialog.getByPlaceholder(`恢复 ${testBackup.name}`).fill(`恢复 ${testBackup.name}`)
  await dialog.getByRole('button', { name: '确认恢复并重启' }).click()
  await expect(page.locator('.el-message__content').filter({ hasText: '恢复请求已确认，系统正在安全重启' })).toBeVisible()
  expect(state.restoreRequests).toBe(1)
})

const responsiveRoutes = [
  '/', '/courses', '/courses/1/archive', '/courses/1/learn', '/courses/1/map',
  '/courses/1/misconceptions', '/exploration/questions', '/domains/new', '/settings',
]

for (const width of [375, 768, 1024, 1440]) {
  test(`${width}px 下核心页面没有横向溢出`, async ({ page }) => {
    await installAPIMocks(page)
    await page.setViewportSize({ width, height: 900 })
    for (const route of responsiveRoutes) {
      await page.goto(route)
      await expect(page.locator('#app')).toBeVisible()
      await page.waitForTimeout(50)
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
      expect(overflow, `${route} has ${overflow}px horizontal overflow at ${width}px`).toBeLessThanOrEqual(1)
    }
  })
}
