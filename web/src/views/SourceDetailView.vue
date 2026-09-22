<template>
  <section v-if="view" class="page-stack source-page">
    <div class="section-heading"><div><el-button text @click="router.push('/sources')">← 返回来源</el-button><h2>{{ view.source.title }}</h2><p class="muted">{{ sourceTypeText(view.source.source_type) }} · {{ statusText(view.source.verification_status) }}</p></div></div>
    <el-alert title="可信度评分评价来源质量和适用性，不等于自动证明来源中的每个结论正确。" type="info" :closable="false" />
    <el-card shadow="never"><el-descriptions :column="2" border><el-descriptions-item label="作者 / 组织">{{ view.source.authors || view.source.organization || '—' }}</el-descriptions-item><el-descriptions-item label="年份">{{ view.source.publication_year || '—' }}</el-descriptions-item><el-descriptions-item label="网址">{{ view.source.url || '—' }}</el-descriptions-item><el-descriptions-item label="DOI / ISBN">{{ view.source.doi || view.source.isbn || '—' }}</el-descriptions-item></el-descriptions></el-card>
    <el-card shadow="never"><template #header><div class="card-header"><span>来源可信度</span><el-button size="small" @click="showCredibility = true">创建可信度草案</el-button></div></template><el-empty v-if="!view.credibility.length" description="暂无可信度评估" /><div v-for="item in view.credibility" :key="item.id" class="review-row"><span>总分 {{ item.overall_score }} / 100 · {{ statusText(item.status) }}</span><el-button v-if="item.status === 'draft'" size="small" type="primary" @click="review(item.id)">审核可信度评估</el-button></div></el-card>
    <el-card shadow="never"><template #header><div class="card-header"><span>来源证据</span><el-button size="small" @click="showEvidence = true">添加来源证据</el-button></div></template><el-empty v-if="!view.evidence.length" description="暂无来源证据" /><div v-for="item in view.evidence" :key="item.id" class="evidence-row"><el-tag size="small">{{ statusText(item.verification_status) }}</el-tag><div><strong>{{ item.locator || item.evidence_type }}</strong><p>{{ item.summary || item.quote }}</p></div><el-button size="small" @click="showLink(item.id)">建立证据关联</el-button></div></el-card>
    <el-card shadow="never"><template #header>证据关联</template><el-empty v-if="!links.length" description="暂无关联" /><div v-for="item in links" :key="item.link.id" class="review-row"><span>{{ targetTypeText(item.link.target_type) }} #{{ item.link.target_id }} · {{ relationText(item.link.relation) }} · {{ statusText(item.link.status) }}</span><el-button v-if="item.link.status === 'proposed'" size="small" type="primary" @click="reviewLink(item.link.id)">审核证据关联</el-button></div></el-card>

    <el-dialog v-model="showEvidence" title="添加来源证据"><el-form label-position="top"><el-form-item label="定位"><el-input v-model="evidence.locator" placeholder="例如：第 1 节" /></el-form-item><el-form-item label="摘要或摘录"><el-input v-model="evidence.summary" type="textarea" /></el-form-item></el-form><template #footer><el-button @click="showEvidence=false">取消</el-button><el-button type="primary" @click="saveEvidence">保存来源证据</el-button></template></el-dialog>
    <el-dialog v-model="showCredibility" title="创建可信度草案"><el-form label-position="top"><el-form-item v-for="field in scoreFields" :key="field.key" :label="field.label"><el-input-number v-model="(credibility as any)[field.key]" :min="0" :max="20" /></el-form-item></el-form><template #footer><el-button @click="showCredibility=false">取消</el-button><el-button type="primary" @click="saveCredibility">创建可信度草案</el-button></template></el-dialog>
    <el-dialog v-model="showLinkDialog" title="建立证据关联"><el-form label-position="top"><el-form-item label="关联对象类型"><el-select v-model="link.target_type" style="width:100%"><el-option label="蓝图节点" value="blueprint_lesson" /><el-option label="正式课程节点" value="lesson" /><el-option label="课程蓝图" value="curriculum_blueprint" /></el-select></el-form-item><el-form-item label="关联对象 ID"><el-input-number v-model="link.target_id" :min="1" /></el-form-item><el-form-item label="关系"><el-select v-model="link.relation"><el-option label="支持" value="supports" /><el-option label="限制" value="limits" /><el-option label="补充背景" value="contextualizes" /><el-option label="矛盾" value="contradicts" /></el-select></el-form-item><el-form-item label="证据强度"><el-select v-model="link.strength"><el-option label="弱" value="weak" /><el-option label="中等" value="moderate" /><el-option label="强" value="strong" /></el-select></el-form-item></el-form><template #footer><el-button @click="showLinkDialog=false">取消</el-button><el-button type="primary" @click="saveLink">创建待审核关联</el-button></template></el-dialog>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { addEvidence, createCredibility, createGroundingLink, getSource, listSourceLinks, reviewCredibility, reviewGroundingLink } from '@/api/grounding'
import type { GroundingLinkView, SourceView } from '@/types/grounding'

const route = useRoute()
const router = useRouter()
const sourceID = Number(String(route.params.id))
const view = ref<SourceView | null>(null)
const links = ref<GroundingLinkView[]>([])
const showEvidence = ref(false)
const showCredibility = ref(false)
const showLinkDialog = ref(false)
const evidence = reactive({ locator: '', summary: '' })
const credibility = reactive({ authority_score: 12, methodology_score: 12, directness_score: 12, recency_score: 12, independence_score: 12 })
const link = reactive({ evidence_id: 0, target_type: 'blueprint_lesson', target_id: 1, relation: 'supports', strength: 'moderate', rationale: '' })
const scoreFields = [{ key: 'authority_score', label: '权威性（0–20）' }, { key: 'methodology_score', label: '方法质量（0–20）' }, { key: 'directness_score', label: '直接性（0–20）' }, { key: 'recency_score', label: '时效性（0–20）' }, { key: 'independence_score', label: '独立性（0–20）' }]

async function load() { try { view.value = await getSource(sourceID); links.value = await listSourceLinks(sourceID) } catch (error) { ElMessage.error(error instanceof Error ? error.message : '来源加载失败') } }
async function saveEvidence() { try { await addEvidence(sourceID, evidence); showEvidence.value = false; Object.assign(evidence, { locator: '', summary: '' }); await load(); ElMessage.success('来源证据已添加') } catch (error) { ElMessage.error(error instanceof Error ? error.message : '来源证据添加失败') } }
async function saveCredibility() { try { await createCredibility(sourceID, credibility); showCredibility.value = false; await load(); ElMessage.success('可信度草案已创建') } catch (error) { ElMessage.error(error instanceof Error ? error.message : '评估创建失败') } }
async function review(id: number) { try { await reviewCredibility(sourceID, id); await load(); ElMessage.success('可信度评估已审核') } catch (error) { ElMessage.error(error instanceof Error ? error.message : '审核失败') } }
function showLink(id: number) { link.evidence_id = id; showLinkDialog.value = true }
async function saveLink() { try { await createGroundingLink(link); showLinkDialog.value = false; ElMessage.success('证据关联已创建为待审核状态') } catch (error) { ElMessage.error(error instanceof Error ? error.message : '证据关联创建失败') } }
async function reviewLink(id: number) { try { await reviewGroundingLink(id); await load(); ElMessage.success('证据关联已审核') } catch (error) { ElMessage.error(error instanceof Error ? error.message : '证据关联审核失败') } }
function sourceTypeText(value: string) { return ({ textbook: '教材', course_material: '课程材料', guideline: '指南', research_paper: '研究论文', other: '其他' } as Record<string, string>)[value] ?? value }
function statusText(value: string) { return ({ draft: '草案', proposed: '待审核', reviewed: '已审核', verified: '已核验', unverified: '未核验', rejected: '已拒绝' } as Record<string, string>)[value] ?? value }
function targetTypeText(value: string) { return ({ blueprint_lesson: '蓝图节点', lesson: '正式课程节点', curriculum_blueprint: '课程蓝图' } as Record<string, string>)[value] ?? value }
function relationText(value: string) { return ({ supports: '支持', limits: '限制', contextualizes: '补充背景', contradicts: '矛盾' } as Record<string, string>)[value] ?? value }
onMounted(load)
</script>

<style scoped>
.source-page { max-width: 1100px; margin: 0 auto; }
.muted { color: var(--text-secondary); }
.card-header, .review-row, .evidence-row { display: flex; justify-content: space-between; align-items: center; gap: 16px; }
.review-row, .evidence-row { padding: 10px 0; border-bottom: 1px solid var(--border-subtle); }
.evidence-row { align-items: flex-start; }
.evidence-row p { margin: 6px 0 0; color: var(--text-secondary); }
</style>
