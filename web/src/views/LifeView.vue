<template>
  <PageShell class="life-page" variant="standard">

    <PageHeader title="生活" description="留下重要经历，也留住当时的想法。"><template #breadcrumbs><nav class="life-breadcrumb" aria-label="当前位置"><span>生活</span><span>/</span><span aria-current="page">{{ tab === 'timeline' ? '生活时间线' : '长期目标' }}</span></nav></template><template #actions><el-button type="primary" @click="create">{{ tab === 'timeline' ? '+ 记录生活事件' : '+ 写下长期目标' }}</el-button></template></PageHeader>
    <div class="life-toolbar"><nav class="life-tabs" aria-label="生活视图"><button type="button" role="tab" :aria-selected="tab === 'timeline'" @click="tab = 'timeline'">生活时间线</button><button type="button" role="tab" :aria-selected="tab === 'goals'" @click="tab = 'goals'">长期目标</button><RouterLink :to="{ path: '/life/finance', query: { life_view: tab, life_domain: domain || undefined, life_milestones: milestones ? 'true' : undefined } }">财务档案</RouterLink><RouterLink :to="{path:'/life/reviews',query:{life_view:tab,life_domain:domain||undefined}}">月度回顾</RouterLink></nav><el-select v-model="domain" clearable placeholder="全部生活领域" aria-label="筛选生活领域" style="width:190px"><el-option v-for="d in lifeDomains" :key="d[0]" :value="d[0]" :label="d[1]" /></el-select><el-checkbox v-if="tab === 'timeline'" v-model="milestones">只看里程碑</el-checkbox></div>
    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon><el-button text @click="load">重试</el-button></el-alert>
    <div v-loading="loading">
      <template v-if="tab === 'timeline'">
        <EmptyState v-if="!loading && !error && !events.length" :title="domain || milestones ? '这里还没有相关经历' : '从一件值得记住的事开始'" description="可以记录今天，也可以补录过去。普通任务不会自动出现在这里。" action-label="记录生活事件" @action="create" />
        <div class="life-timeline"><section v-for="group in months" :key="group.month"><h2 class="life-month">{{ group.month.replace('-', ' 年 ') }} 月</h2><article v-for="event in group.events" :key="event.id" class="life-event-row"><time class="life-event-date" :datetime="event.occurred_on">{{ event.occurred_on.slice(5).replace('-', ' / ') }}</time><button class="life-event-card" :class="{ 'is-milestone': event.milestone }" type="button" @click="openEvent(event.id)"><h3>{{ event.title }}</h3><p v-if="event.description">{{ event.description }}</p><div class="life-metadata"><span v-if="event.milestone">◇ 里程碑</span><span v-if="event.primary_domain">{{ domainLabel(event.primary_domain) }}</span><span v-if="event.secondary_domain">{{ domainLabel(event.secondary_domain) }}</span><span v-if="event.goal_id">目标 · {{ goals.find(g => g.id === event.goal_id)?.title || '关联长期目标' }}</span><span v-if="event.source_id && event.source_title">来自{{ sourceLabel(event.source_type) }} · {{ event.source_title }}</span><span v-if="event.external_url">{{ event.link_name || (event.external_url.startsWith('obsidian:') ? 'Obsidian 资料' : '外部资料') }}</span></div></button></article></section></div>
      </template>
      <template v-else><EmptyState v-if="!loading && !error && !shownGoals.length" title="给长期愿望留一个位置" description="不必设定期限或百分比，记录它为什么重要，以及现在的判断。" action-label="写下长期目标" @action="create" /><div class="life-goals"><button v-for="goal in shownGoals" :key="goal.id" class="life-event-card life-goal-card" type="button" @click="openGoal(goal.id)"><h3>{{ goal.title }}</h3><p v-if="goal.current_note || goal.why">{{ goal.current_note || goal.why }}</p><div class="life-metadata"><el-tag size="small" effect="plain">{{ goalStatusLabel(goal.status) }}</el-tag><span v-if="goal.domain">{{ domainLabel(goal.domain) }}</span><time v-if="lifeUpdatedLabel(goal.updated_at)" :datetime="goal.updated_at">更新于 {{ lifeUpdatedLabel(goal.updated_at) }}</time></div></button></div></template>
    </div>
    <LifeEventDialog v-model="eventOpen" :event="selectedEvent" @saved="eventSaved" @deleted="load" />
    <LifeGoalDialog v-model="goalOpen" :detail="selectedGoal" @saved="goalSaved" @detail="goalDetailUpdated" @event="openRelatedEvent" @asset="openRelatedAsset" />
  </PageShell>
</template>
<script setup lang="ts">
import PageShell from '@/components/PageShell.vue'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useArchiveScroll } from '@/composables/useArchiveScroll'
import PageHeader from '@/components/PageHeader.vue'
import EmptyState from '@/components/EmptyState.vue'
import LifeEventDialog from '@/components/life/LifeEventDialog.vue'
import LifeGoalDialog from '@/components/life/LifeGoalDialog.vue'
import { domainLabel, getLifeEvent, getLifeGoal, goalStatusLabel, lifeUpdatedLabel, lifeDomains, listLifeEvents, listLifeGoals, type LifeEvent, type LifeGoal, type GoalDetail } from '@/api/life'
import '@/life.css'
const route = useRoute(), router = useRouter()
const restoreScroll = useArchiveScroll()
const tab = ref<'timeline' | 'goals'>(route.query.view === 'goals' ? 'goals' : 'timeline'), domain = ref(typeof route.query.domain === 'string' ? route.query.domain : ''), milestones = ref(route.query.milestones === 'true'), loading = ref(false), error = ref(''), events = ref<LifeEvent[]>([]), goals = ref<LifeGoal[]>([]), eventOpen = ref(false), goalOpen = ref(false), selectedEvent = ref<LifeEvent | null>(null), selectedGoal = ref<GoalDetail | null>(null)
let sequence = 0, detailSequence = 0
const months = computed(() => { const grouped = new Map<string, LifeEvent[]>(); for (const event of events.value) { const key = event.occurred_on.slice(0, 7); grouped.set(key, [...(grouped.get(key) ?? []), event]) }; return [...grouped].map(([month, events]) => ({ month, events })) })
const shownGoals = computed(() => goals.value.filter(g => !domain.value || g.domain === domain.value))
const message = (reason: unknown) => reason instanceof Error ? reason.message : '生活档案读取失败，请重试。'
async function load() { const token = ++sequence; loading.value = true; error.value = ''; try { const [e, g] = await Promise.all([listLifeEvents(domain.value, milestones.value), listLifeGoals()]); if (token === sequence) { events.value = e; goals.value = g; void restoreScroll() } } catch (reason) { if (token === sequence) error.value = message(reason) } finally { if (token === sequence) loading.value = false } }
function sourceLabel(type: string) { return ({ project: '项目', course: '课程', inbox: '收集箱' })[type as 'project' | 'course' | 'inbox'] ?? '来源' }
function goalDetailUpdated(detail: GoalDetail) { selectedGoal.value = detail; goals.value = goals.value.map(g => g.id === detail.goal.id ? detail.goal : g) }
function create() { if (tab.value === 'timeline') { selectedEvent.value = null; eventOpen.value = true } else { selectedGoal.value = null; goalOpen.value = true } }
async function openEvent(id: number) { const token = ++detailSequence; try { const event = await getLifeEvent(id); if (token === detailSequence) { selectedEvent.value = event; eventOpen.value = true } } catch (reason) { error.value = message(reason) } }
async function openGoal(id: number) { const token = ++detailSequence; try { const detail = await getLifeGoal(id); if (token === detailSequence) { selectedGoal.value = detail; goalOpen.value = true } } catch (reason) { error.value = message(reason) } }
async function eventSaved(event: LifeEvent) { events.value = [event, ...events.value.filter(e => e.id !== event.id)].filter(e => (!domain.value || [e.primary_domain, e.secondary_domain].includes(domain.value)) && (!milestones.value || e.milestone)).sort((a, b) => b.occurred_on.localeCompare(a.occurred_on) || b.id - a.id); if (selectedEvent.value) { selectedEvent.value = { ...event, source_available: selectedEvent.value.source_available, source_status: selectedEvent.value.source_status }; try { selectedEvent.value = await getLifeEvent(event.id) } catch (reason) { selectedEvent.value = event; error.value = message(reason) } }; await load() }
async function goalSaved(goal: LifeGoal) { goals.value = [goal, ...goals.value.filter(g => g.id !== goal.id)]; if (selectedGoal.value) { selectedGoal.value = { ...selectedGoal.value, goal }; try { selectedGoal.value = await getLifeGoal(goal.id) } catch (reason) { error.value = message(reason) } }; await load() }
function openRelatedAsset(id: number) { goalOpen.value = false; void router.push({ path: '/life/finance', query: { tab: 'assets', asset: id, life_view: 'goals', life_domain: domain.value || undefined, return_goal: selectedGoal.value?.goal.id } }) }
function openRelatedEvent(id: number) { goalOpen.value = false; void openEvent(id) }
async function queryDetail() { if (route.query.event && Number(route.query.event) > 0) { tab.value = 'timeline'; await openEvent(Number(route.query.event)) } else if (route.query.goal && Number(route.query.goal) > 0) { tab.value = 'goals'; await openGoal(Number(route.query.goal)) } }
watch([domain, milestones], load)
watch([tab, domain, milestones], () => { void router.replace({ query: { ...route.query, view: tab.value === 'goals' ? 'goals' : undefined, domain: domain.value || undefined, milestones: milestones.value ? 'true' : undefined } }) })
watch(() => route.fullPath, queryDetail)
onMounted(async () => { await load(); await queryDetail() })
onBeforeUnmount(() => { ++sequence; ++detailSequence })
</script>
