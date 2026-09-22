<template>
  <section class="page-stack source-page">
    <div class="section-heading"><div><span class="eyebrow">来源登记</span><h2>知识来源</h2><p class="muted">来源质量与证据关联需要人工审核；可信度不等于真值。</p></div><el-button type="primary" @click="showCreate = true">新增来源</el-button></div>
    <el-input v-model="query" clearable placeholder="搜索标题、作者或组织" @keyup.enter="load" />
    <el-empty v-if="!loading && !sources.length" description="暂无来源。可以先添加一条参考资料进行验收。" />
    <div v-loading="loading" class="source-grid">
      <el-card v-for="source in sources" :key="source.id" shadow="never" class="source-card" @click="router.push(`/sources/${source.id}`)">
        <div class="card-header"><strong>{{ source.title }}</strong><el-tag size="small">{{ source.source_type }}</el-tag></div>
        <p>{{ source.authors || source.organization || '未填写作者 / 组织' }}</p><p class="muted">{{ source.publication_year || '年份未知' }} · {{ source.verification_status }} · {{ source.access_status }}</p>
      </el-card>
    </div>
    <el-dialog v-model="showCreate" title="新增知识来源" width="520px">
      <el-form label-position="top"><el-form-item label="标题"><el-input v-model="form.title" /></el-form-item><el-form-item label="来源类型"><el-select v-model="form.source_type" style="width:100%"><el-option label="教材" value="textbook"/><el-option label="课程材料" value="course_material"/><el-option label="指南" value="guideline"/><el-option label="研究论文" value="research_paper"/><el-option label="其他" value="other"/></el-select></el-form-item><el-form-item label="作者 / 组织"><el-input v-model="form.organization" /></el-form-item><el-form-item label="摘要说明"><el-input v-model="form.description" type="textarea" /></el-form-item></el-form>
      <template #footer><el-button @click="showCreate = false">取消</el-button><el-button type="primary" @click="save">保存知识来源</el-button></template>
    </el-dialog>
  </section>
</template>
<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { createSource, listSources } from '@/api/grounding'
import type { KnowledgeSource } from '@/types/grounding'
const router = useRouter(); const sources = ref<KnowledgeSource[]>([]); const loading = ref(false); const query = ref(''); const showCreate = ref(false); const form = reactive({ title: '', source_type: 'course_material', organization: '', description: '' })
async function load() { loading.value = true; try { sources.value = await listSources(query.value) } catch (error) { ElMessage.error(error instanceof Error ? error.message : '来源加载失败') } finally { loading.value = false } }
async function save() { try { await createSource(form); showCreate.value = false; Object.assign(form, { title: '', organization: '', description: '' }); await load(); ElMessage.success('来源已创建') } catch (error) { ElMessage.error(error instanceof Error ? error.message : '来源创建失败') } }
onMounted(load)
</script>
<style scoped>.source-page { max-width: 1100px; margin: 0 auto; }.source-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(280px,1fr)); gap:16px; margin-top:18px; }.source-card { cursor:pointer; }.source-card:hover { border-color:var(--el-color-primary); }.muted { color:var(--el-text-color-secondary); }</style>
