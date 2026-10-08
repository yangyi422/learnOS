<template>
  <section class="page-stack life-page">
    <PageHeader title="生活" description="留下重要经历，也留住当时的想法。"><template #actions><el-button type="primary" @click="create">{{ tab === 'timeline' ? '+ 记录生活事件' : '+ 写下长期目标' }}</el-button></template></PageHeader>
    <div class="life-toolbar"><nav class="life-tabs" aria-label="生活视图"><button type="button" role="tab" :aria-selected="tab === 'timeline'" @click="tab = 'timeline'">生活时间线</button><button type="button" role="tab" :aria-selected="tab === 'goals'" @click="tab = 'goals'">长期目标</button></nav><el-select v-model="domain" clearable placeholder="全部生活领域" aria-label="筛选生活领域" style="width:190px"><el-option v-for="d in lifeDomains" :key="d[0]" :value="d[0]" :label="d[1]" /></el-select><el-checkbox v-if="tab === 'timeline'" v-model="milestones">只看里程碑</el-checkbox></div>
    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon><el-button text @click="load">重试</el-button></el-alert>
    <div v-loading="loading">
      <template v-if="tab === 'timeline'">
        <EmptyState v-if="!loading && !error && !events.length" :title="domain || milestones ? '这里还没有相关经历' : '从一件值得记住的事开始'" description="可以记录今天，也可以补录过去。普通任务不会自动出现在这里。" action-label="记录生活事件" @action="create" />
        <div class="life-timeline"><section v-for="group in months" :key="group.month"><h2 class="life-month">{{ group.month.replace('-', ' 年 ') }} 月</h2><article v-for="event in group.events" :key="event.id" class="life-event-row"><time class="life-event-date" :datetime="event.occurred_on">{{ event.occurred_on.slice(5).replace('-', ' / ') }}</time><button class="life-event-card" :class="{ 'is-milestone': event.milestone }" type="button" @click="openEvent(event.id)"><h3>{{ event.title }}</h3><p v-if="event.description">{{ event.description }}</p><div class="life-metadata"><span v-if="event.milestone">◇ 里程碑</span><span v-if="event.primary_domain">{{ domainLabel(event.primary_domain) }}</span><span v-if="event.secondary_domain">{{ domainLabel(event.secondary_domain) }}</span><span v-if="event.goal_id">关联长期目标</span></div></button></article></section></div>
      </template>
      <template v-else><EmptyState v-if="!loading && !error && !shownGoals.length" title="给长期愿望留一个位置" description="不必设定期限或百分比，记录它为什么重要，以及现在的判断。" action-label="写下长期目标" @action="create" /><div class="life-goals"><button v-for="goal in shownGoals" :key="goal.id" class="life-event-card life-goal-card" type="button" @click="openGoal(goal.id)"><h3>{{ goal.title }}</h3><p v-if="goal.current_note || goal.why">{{ goal.current_note || goal.why }}</p><div class="life-metadata"><el-tag size="small" effect="plain">{{ goalStatusLabel(goal.status) }}</el-tag><span>{{ domainLabel(goal.domain) }}</span></div></button></div></template>
    </div>
    <LifeEventDialog v-model="eventOpen" :event="selectedEvent" @saved="eventSaved" @deleted="load" />
    <LifeGoalDialog v-model="goalOpen" :detail="selectedGoal" @saved="goalSaved" @detail="selectedGoal = $event" @event="openRelatedEvent" />
  </section>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import PageHeader from '@/components/PageHeader.vue'
import EmptyState from '@/components/EmptyState.vue'
import LifeEventDialog from '@/components/life/LifeEventDialog.vue'
import LifeGoalDialog from '@/components/life/LifeGoalDialog.vue'
import { domainLabel, getLifeEvent, getLifeGoal, goalStatusLabel, lifeDomains, listLifeEvents, listLifeGoals, type LifeEvent, type LifeGoal, type GoalDetail } from '@/api/life'
import '@/life.css'
const route = useRoute()
const tab = ref<'timeline' | 'goals'>('timeline'), domain = ref(''), milestones = ref(false), loading = ref(false), error = ref(''), events = ref<LifeEvent[]>([]), goals = ref<LifeGoal[]>([]), eventOpen = ref(false), goalOpen = ref(false), selectedEvent = ref<LifeEvent | null>(null), selectedGoal = ref<GoalDetail | null>(null)
let sequence = 0, detailSequence = 0
const months = computed(() => { const grouped = new Map<string, LifeEvent[]>(); for (const event of events.value) { const key = event.occurred_on.slice(0, 7); grouped.set(key, [...(grouped.get(key) ?? []), event]) }; return [...grouped].map(([month, events]) => ({ month, events })) })
const shownGoals = computed(() => goals.value.filter(g => !domain.value || g.domain === domain.value))
const message = (reason: unknown) => reason instanceof Error ? reason.message : '生活档案读取失败，请重试。'
async function load() { const token = ++sequence; loading.value = true; error.value = ''; try { const [e, g] = await Promise.all([listLifeEvents(domain.value, milestones.value), listLifeGoals()]); if (token === sequence) { events.value = e; goals.value = g } } catch (reason) { if (token === sequence) error.value = message(reason) } finally { if (token === sequence) loading.value = false } }
function create() { if (tab.value === 'timeline') { selectedEvent.value = null; eventOpen.value = true } else { selectedGoal.value = null; goalOpen.value = true } }
async function openEvent(id: number) { const token = ++detailSequence; try { const event = await getLifeEvent(id); if (token === detailSequence) { selectedEvent.value = event; eventOpen.value = true } } catch (reason) { error.value = message(reason) } }
async function openGoal(id: number) { const token = ++detailSequence; try { const detail = await getLifeGoal(id); if (token === detailSequence) { selectedGoal.value = detail; goalOpen.value = true } } catch (reason) { error.value = message(reason) } }
async function eventSaved(event: LifeEvent) { if (selectedEvent.value) { selectedEvent.value = { ...event, source_available: selectedEvent.value.source_available, source_status: selectedEvent.value.source_status }; try { selectedEvent.value = await getLifeEvent(event.id) } catch (reason) { selectedEvent.value = event; error.value = message(reason) } }; await load() }
async function goalSaved(goal: LifeGoal) { if (selectedGoal.value) { selectedGoal.value = { ...selectedGoal.value, goal }; try { selectedGoal.value = await getLifeGoal(goal.id) } catch (reason) { error.value = message(reason) } }; await load() }
function openRelatedEvent(id: number) { goalOpen.value = false; void openEvent(id) }
async function queryDetail() { if (route.query.event && Number(route.query.event) > 0) { tab.value = 'timeline'; await openEvent(Number(route.query.event)) } else if (route.query.goal && Number(route.query.goal) > 0) { tab.value = 'goals'; await openGoal(Number(route.query.goal)) } }
watch([domain, milestones], load)
watch(() => route.fullPath, queryDetail)
onMounted(async () => { await load(); await queryDetail() })
onBeforeUnmount(() => { ++sequence; ++detailSequence })
</script>
