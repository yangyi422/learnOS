import { describe, expect, it } from 'vitest'
import { normalizeDomainForm, validateDomainForm } from './domainForm'

describe('domain form validation', () => {
  it('rejects blank and whitespace-only required fields', () => {
    expect(validateDomainForm({ domain_name: '  ', learning_goal: '\n  ', target_depth: 'systematic' })).toEqual({
      domain_name: '请输入领域名称',
      learning_goal: '请输入为什么想学这个领域',
    })
  })

  it('validates field lengths and target depth', () => {
    expect(validateDomainForm({ domain_name: '学', learning_goal: '太短', target_depth: 'unknown' })).toEqual({
      domain_name: '领域名称需为 2–50 个字符',
      learning_goal: '学习原因至少需要 10 个字符',
      target_depth: '请选择有效的期望深度',
    })
    expect(validateDomainForm({ domain_name: '学'.repeat(51), learning_goal: '建立完整而可靠的基础理解', target_depth: 'systematic' }).domain_name).toBe('领域名称需为 2–50 个字符')
  })

  it('trims a valid payload before submission', () => {
    const input = { domain_name: '  心理学  ', learning_goal: '  系统理解心理学并应用于日常生活  ', target_depth: 'systematic' }
    expect(validateDomainForm(input)).toEqual({})
    expect(normalizeDomainForm(input)).toEqual({
      domain_name: '心理学',
      learning_goal: '系统理解心理学并应用于日常生活',
      target_depth: 'systematic',
    })
  })
})
