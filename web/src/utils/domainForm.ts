export const DOMAIN_DEPTHS = ['overview', 'foundation', 'systematic'] as const

export type DomainDepth = (typeof DOMAIN_DEPTHS)[number]

export interface DomainFormInput {
  domain_name: string
  learning_goal: string
  target_depth: string
}

export type DomainFormField = keyof DomainFormInput
export type DomainFormErrors = Partial<Record<DomainFormField, string>>

export function normalizeDomainForm(input: DomainFormInput): DomainFormInput & { target_depth: DomainDepth } {
  return {
    domain_name: input.domain_name.trim(),
    learning_goal: input.learning_goal.trim(),
    target_depth: input.target_depth.trim() as DomainDepth,
  }
}

export function validateDomainForm(input: DomainFormInput): DomainFormErrors {
  const normalized = normalizeDomainForm(input)
  const errors: DomainFormErrors = {}
  const domainNameLength = Array.from(normalized.domain_name).length
  const learningGoalLength = Array.from(normalized.learning_goal).length

  if (domainNameLength === 0) errors.domain_name = '请输入领域名称'
  else if (domainNameLength < 2 || domainNameLength > 50) errors.domain_name = '领域名称需为 2–50 个字符'

  if (learningGoalLength === 0) errors.learning_goal = '请输入为什么想学这个领域'
  else if (learningGoalLength < 10) errors.learning_goal = '学习原因至少需要 10 个字符'
  else if (learningGoalLength > 4000) errors.learning_goal = '学习原因不能超过 4000 个字符'

  if (!DOMAIN_DEPTHS.includes(normalized.target_depth as DomainDepth)) errors.target_depth = '请选择有效的期望深度'

  return errors
}
