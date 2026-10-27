<template>
  <article class="plan-task" :data-task-id="task.id">
    <button class="plan-task__complete" type="button" :disabled="busy" :aria-label="'完成任务：' + task.title" @click="$emit('complete', task)"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true"><path d="m5 12 4 4L19 6" /></svg></button>
    <div class="plan-task__copy">
      <button type="button" class="plan-task__title" @click="$emit('open', task)">{{ task.title }}</button>
      <p class="plan-task__meta"><span>{{ task.project_title }}</span><span v-if="context === 'all'">{{ contextLabel(task.effective_context) }}</span><span v-if="isFocus" class="plan-task__focus">今日焦点</span></p>
      <p v-if="task.due_date" class="plan-task__date" :class="{ 'is-overdue': task.due_date < date }">{{ task.due_date < date ? '已逾期' : task.due_date === date ? '今天截止' : '截止' }} {{ task.due_date }}</p>
      <p v-if="task.plan_date && task.plan_date !== date" class="plan-task__meta">{{ task.plan_date < date ? '原计划' : '计划' }} {{ task.plan_date }}</p>
    </div>
    <div class="plan-task__actions">
      <button v-if="!planned" type="button" class="plan-task__today" :disabled="busy" @click="$emit('today', task)">加入今天</button>
      <el-dropdown trigger="click" @command="(command: string) => command === 'remove' ? $emit('remove', task) : $emit('schedule', task)">
        <button type="button" class="plan-task__more" :disabled="busy" :aria-label="'任务安排操作：' + task.title"><svg viewBox="0 0 24 24" aria-hidden="true" fill="currentColor"><circle cx="5" cy="12" r="1.5"/><circle cx="12" cy="12" r="1.5"/><circle cx="19" cy="12" r="1.5"/></svg></button>
        <template #dropdown><el-dropdown-menu><el-dropdown-item command="schedule">重新安排日期</el-dropdown-item><el-dropdown-item v-if="planned" command="remove">移出今天</el-dropdown-item></el-dropdown-menu></template>
      </el-dropdown>
    </div>
  </article>
</template>
<script setup lang="ts">
import type { PlanningTask } from '@/api/workspace'
import type { WorkspaceContext } from '@/workspace-context'
defineProps<{ task: PlanningTask; date: string; context: WorkspaceContext; planned?: boolean; busy: boolean; isFocus?: boolean }>()
defineEmits<{ complete: [task: PlanningTask]; open: [task: PlanningTask]; today: [task: PlanningTask]; schedule: [task: PlanningTask]; remove: [task: PlanningTask] }>()
function contextLabel(value: string) { return value === 'work' ? '工作' : value === 'life' ? '生活' : '未分类' }
</script>
<style scoped>
.plan-task { display:flex; align-items:flex-start; gap:10px; padding:14px 0; border-bottom:1px solid var(--home-edge); }
.plan-task__complete,.plan-task__more { flex:none; display:grid; place-items:center; min-width:32px; min-height:32px; padding:6px; border:1px solid var(--home-edge); border-radius:9px; background:transparent; color:var(--home-secondary); cursor:pointer; }
.plan-task__complete svg,.plan-task__more svg { width:18px; height:18px; }
.plan-task__complete:hover { color:var(--home-link); background:var(--home-tint); }
.plan-task__copy { flex:1; min-width:0; }
.plan-task__title { padding:3px 0; border:0; background:transparent; color:var(--home-ink); font:inherit; font-size:14px; line-height:1.6; text-align:left; overflow-wrap:anywhere; cursor:pointer; }
.plan-task__title:hover { color:var(--home-link); }
.plan-task__meta,.plan-task__date { display:flex; flex-wrap:wrap; gap:4px 8px; margin:4px 0 0; font-size:12px; line-height:1.6; color:var(--home-muted); overflow-wrap:anywhere; }
.plan-task__focus { border:1px solid var(--home-edge); border-radius:999px; padding:0 7px; color:var(--home-link); font-size:11px; line-height:1.7; }
.plan-task__date.is-overdue { color:var(--color-danger); }
.plan-task__actions { display:flex; flex-direction:column; align-items:flex-end; gap:4px; flex:none; }
.plan-task__more { border-color:transparent; }
.plan-task__today { min-height:32px; border:0; background:transparent; color:var(--home-link); font:inherit; font-size:12px; cursor:pointer; }
button:focus-visible { outline:2px solid var(--home-link); outline-offset:3px; }
button:disabled { opacity:.55; cursor:wait; }
@media(max-width:700px) { .plan-task__complete,.plan-task__more,.plan-task__today { min-width:44px; min-height:44px; } }
</style>
