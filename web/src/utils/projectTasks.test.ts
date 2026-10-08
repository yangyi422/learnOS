import { describe, expect, it } from 'vitest'
import { groupTodayTasks, localCalendarDate, taskDueDate } from './projectTasks'
import type { ProjectTask } from '@/api/projects'

const task = (id: number, status: ProjectTask['status'], due_date: string | null): ProjectTask => ({ id, project_id: id % 2 + 1, title: String(id), description: '', status, priority: 'normal', sort_order: id, due_date, created_at: '', updated_at: '', completed_at: null })
describe('Today calendar grouping', () => {
  it('prioritizes overdue and due dates, deduplicates tasks, and excludes completed tasks', () => {
    const overdue = task(1, 'doing', '2026-10-07')
    const due = task(2, 'doing', '2026-10-08')
    const doing = task(3, 'doing', null)
    const next = task(4, 'next', null)
    const groups = groupTodayTasks({ date: '2026-10-08', doing: [overdue, due, doing], due: [overdue, due, task(5, 'done', '2026-10-07')], next: [next, due] }, '2026-10-08')
    expect(groups.map(group => group.tasks.map(task => task.id))).toEqual([[1], [2], [3], [4]])
  })
  it('uses local day components and treats stored dates as calendar dates', () => {
    expect(localCalendarDate(new Date(2026, 9, 8, 0, 5))).toBe('2026-10-08')
    expect(taskDueDate('2026-10-08')).toBe('2026-10-08')
    expect(taskDueDate('2026-10-08T00:00:00Z')).toBe('2026-10-08')
    expect(groupTodayTasks({ date: '', due: [task(1, 'next', '2026-10-08')], doing: [], next: [] }, '2026-10-08')[1]?.tasks).toHaveLength(1)
  })
})
