import type { ProjectTask, TodayView } from '@/api/projects'

export function localCalendarDate(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

// due_date is a calendar date, never a UTC instant. Accept legacy API values
// with a time suffix without converting the stored calendar day.
export function taskDueDate(value: string | null): string | null {
  return value && /^\d{4}-\d{2}-\d{2}(?:$|T)/.test(value) ? value.slice(0, 10) : null
}

export function groupTodayTasks(view: TodayView | null, date: string) {
  const groups: { key: string; title: string; tasks: ProjectTask[] }[] = [
    { key: 'overdue', title: '已逾期', tasks: [] },
    { key: 'due', title: '今天到期', tasks: [] },
    { key: 'doing', title: '正在进行', tasks: [] },
    { key: 'next', title: '下一步', tasks: [] },
  ]
  const seen = new Set<number>()
  for (const task of [...(view?.due ?? []), ...(view?.doing ?? []), ...(view?.next ?? [])]) {
    if (seen.has(task.id) || task.status === 'done') continue
    seen.add(task.id)
    const due = taskDueDate(task.due_date)
    const group = due && due < date ? 0 : due === date ? 1 : task.status === 'doing' ? 2 : task.status === 'next' ? 3 : -1
    if (group >= 0) groups[group]!.tasks.push(task)
  }
  for (const group of groups) {
    group.tasks.sort((a, b) => (taskDueDate(a.due_date) ?? '9999').localeCompare(taskDueDate(b.due_date) ?? '9999') || a.sort_order - b.sort_order || a.id - b.id)
  }
  return groups
}
