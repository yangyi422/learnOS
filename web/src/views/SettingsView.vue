<template>
  <PageShell class="settings-page" variant="standard">
    <PageHeader eyebrow="系统管理" title="系统设置" description="管理 AI 连接、备份恢复、数据导出和运行状态。">
      <template #actions><el-button :loading="loading" @click="load">刷新系统状态</el-button></template>
    </PageHeader>

    <p class="sr-only" aria-live="polite">{{ liveMessage }}</p>
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />

    <HomeAppearanceDialog v-model="appearanceOpen" />
    <el-card shadow="never" class="appearance-settings" aria-labelledby="appearance-settings-title">
      <div class="appearance-settings__copy">
        <div>
          <span class="appearance-settings__eyebrow">WORKSPACE APPEARANCE</span>
          <h2 id="appearance-settings-title">外观主题</h2>
          <p>主题偏好只保存在当前浏览器，不会写入学习数据。</p>
          <el-button @click="appearanceOpen = true">调整界面色系</el-button>
        </div>
        <fieldset class="appearance-settings__options">
          <legend class="sr-only">选择外观主题</legend>
          <button
            v-for="option in themeOptions"
            :key="option.value"
            type="button"
            :aria-label="option.label"
            :aria-pressed="preference === option.value"
            :class="{ 'is-active': preference === option.value }"
            @click="setThemePreference(option.value)"
          >
            <WorkspaceIcon :name="option.icon" />
            <span><strong>{{ option.label }}</strong><small>{{ option.description }}</small></span>
          </button>
        </fieldset>
      </div>
    </el-card>

    <el-card shadow="never" aria-labelledby="ai-settings-title">
      <template #header><span id="ai-settings-title">AI 配置</span></template>
      <p class="settings-help">领域初始化、课程生成、探索和学习评价会使用这里的配置。API Key 只保存在后端，页面和接口仅显示配置状态。</p>
      <el-alert
        v-if="providerChanged"
        title="修改 AI 服务商可能改变生成内容、响应速度和费用。保存后会立即影响新的 AI 请求。"
        type="warning"
        show-icon
        :closable="false"
        class="settings-inline-alert"
      />
      <el-form :model="aiForm" label-position="top" class="ai-form" @submit.prevent="saveAIConfiguration">
        <div class="settings-form-grid">
          <el-form-item label="AI 服务商">
            <el-select v-model="aiForm.provider" aria-label="选择 AI 服务商">
              <el-option label="DeepSeek（真实 AI）" value="deepseek" />
              <el-option label="本地模拟（不调用外部服务）" value="mock" />
            </el-select>
          </el-form-item>
          <el-form-item label="模型">
            <el-select v-model="aiForm.model" filterable allow-create default-first-option aria-label="选择或输入 AI 模型" :disabled="aiForm.provider === 'mock'">
              <el-option label="DeepSeek Chat" value="deepseek-chat" />
              <el-option label="DeepSeek Reasoner" value="deepseek-reasoner" />
              <el-option v-if="aiForm.model && !knownModels.includes(aiForm.model)" :label="aiForm.model" :value="aiForm.model" />
            </el-select>
          </el-form-item>
        </div>
        <el-form-item v-if="aiForm.provider === 'deepseek'" label="API Key">
          <div class="secret-field">
            <el-input
              v-model="aiForm.api_key"
              type="password"
              show-password
              autocomplete="new-password"
              aria-describedby="api-key-help api-key-status"
              placeholder="输入新 Key；留空则保留已保存 Key"
            />
            <StatusTag id="api-key-status" :status="aiConfig?.api_key_configured ? 'configured' : 'unseen'" :label="aiConfig?.api_key_configured ? '已配置' : '未配置'" />
          </div>
          <p id="api-key-help" class="field-help">密钥不会回传、写入浏览器持久化状态或出现在日志中。</p>
        </el-form-item>
        <el-form-item v-if="aiForm.provider === 'deepseek' && aiConfig?.api_key_configured" label="密钥操作">
          <el-checkbox v-model="aiForm.clear_api_key">清除已保存的 API Key</el-checkbox>
        </el-form-item>
        <el-form-item v-if="aiForm.provider === 'deepseek'" label="服务地址">
          <el-input v-model="aiForm.base_url" aria-label="AI 服务地址" />
        </el-form-item>
        <div class="settings-actions">
          <el-button type="primary" native-type="submit" :loading="aiSaving">保存 AI 配置</el-button>
          <el-button :loading="aiTesting" @click="runAIConnectionTest">测试 AI 连接</el-button>
          <span class="settings-help">保存后立即生效，无需手动重启。</span>
        </div>
      </el-form>

      <div v-if="aiConfig" class="effective-config" aria-label="实际生效的 AI 配置">
        <div><span>实际服务商</span><strong>{{ providerText(aiConfig.effective_provider) }}</strong></div>
        <div><span>实际模型</span><strong>{{ aiConfig.effective_model || '未设置' }}</strong></div>
        <div><span>最近成功调用</span><strong>{{ aiConfig.last_successful_call_at ? formatDate(aiConfig.last_successful_call_at) : '暂无成功调用记录' }}</strong></div>
        <div v-if="connectionResult"><span>最近连接测试</span><strong>{{ formatDate(connectionResult.checked_at) }} · {{ connectionResult.latency_ms }} ms</strong></div>
      </div>
    </el-card>

    <el-card v-if="diagnostics" shadow="never" aria-labelledby="runtime-title">
      <template #header><span id="runtime-title">运行状态</span></template>
      <el-descriptions :column="descriptionColumns" border>
        <el-descriptions-item label="版本">{{ diagnostics.app_version }}</el-descriptions-item>
        <el-descriptions-item label="运行环境">{{ environmentText(diagnostics.environment) }}</el-descriptions-item>
        <el-descriptions-item label="数据库状态">{{ diagnostics.database_status.status === 'ok' ? '正常' : '异常' }}</el-descriptions-item>
        <el-descriptions-item label="数据结构版本">{{ diagnostics.schema_version }}</el-descriptions-item>
        <el-descriptions-item label="AI 服务商">{{ providerText(diagnostics.ai_provider) }}</el-descriptions-item>
        <el-descriptions-item label="AI 模型">{{ diagnostics.ai_model }}</el-descriptions-item>
        <el-descriptions-item label="数据库路径" :span="descriptionColumns">
          <span class="path-value">{{ diagnostics.database_path }}</span>
          <el-button text size="small" aria-label="复制数据库路径" @click="copyDatabasePath">复制路径</el-button>
        </el-descriptions-item>
        <el-descriptions-item label="数据库大小">{{ formatSize(diagnostics.database_size) }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card shadow="never" aria-labelledby="capture-token-title">
      <template #header><span id="capture-token-title">快速采集凭证</span></template>
      <p class="settings-help">
        供 uTools 插件等外部客户端写入收集箱。凭证只能调用「快速采集」接口，不能登录、不能读取或修改其他数据，可随时撤销。
      </p>
      <div class="settings-actions">
        <el-input v-model="tokenName" maxlength="60" placeholder="凭证名称（例如 uTools 采集）" aria-label="凭证名称" style="max-width: 260px" />
        <el-select v-model="tokenDays" aria-label="有效期" style="max-width: 150px">
          <el-option :value="30" label="30 天" />
          <el-option :value="90" label="90 天" />
          <el-option :value="180" label="180 天" />
          <el-option :value="365" label="365 天" />
        </el-select>
        <el-button type="primary" :loading="tokenCreating" @click="createToken">生成凭证</el-button>
      </div>
      <el-alert v-if="createdToken" type="success" :closable="false" show-icon>
        <template #title>凭证已生成，只显示这一次</template>
        <div class="token-secret">
          <code>{{ createdToken }}</code>
          <el-button text size="small" @click="copyToken">复制</el-button>
        </div>
        <p class="settings-help">请立即粘贴到 uTools 插件设置中；关闭或刷新后无法再次查看。</p>
      </el-alert>
      <el-empty v-if="tokens.length === 0" description="还没有采集凭证" :image-size="60" />
      <ul v-else class="token-list">
        <li v-for="item in tokens" :key="item.id">
          <div>
            <strong>{{ item.name }}</strong>
            <code>{{ item.prefix }}…</code>
          </div>
          <div class="token-meta">
            <span>创建 {{ formatDate(item.created_at) }}</span>
            <span>{{ tokenExpiryText(item) }}</span>
            <span>{{ item.last_used_at ? `最近使用 ${formatDate(item.last_used_at)}` : '尚未使用' }}</span>
          </div>
          <el-button v-if="!item.revoked_at" text type="danger" @click="revokeToken(item)">撤销</el-button>
          <span v-else class="settings-help">已撤销</span>
        </li>
      </ul>
    </el-card>

    <el-card shadow="never" aria-labelledby="data-safety-title">
      <template #header><span id="data-safety-title">数据安全</span></template>
      <div class="settings-actions">
        <el-button type="primary" :loading="backupLoading" @click="backup">创建完整备份</el-button>
        <el-button @click="download('json')">导出学习数据为 JSON</el-button>
        <el-button @click="download('markdown')">导出学习数据为 Markdown</el-button>
        <el-button :loading="consistencyLoading" @click="checkConsistency">检查数据一致性</el-button>
      </div>

      <div v-if="latestBackup" class="operation-result" role="status">
        <strong>最近备份已完成</strong>
        <span>{{ latestBackup.name }}</span>
        <span>{{ formatSize(latestBackup.size) }} · {{ formatDate(latestBackup.created_at) }}</span>
        <span>范围：{{ backupScope(latestBackup) }}</span><el-button text @click="downloadBackup(latestBackup.name)">下载备份</el-button><span>保存位置：{{ latestBackup.path }}</span>
      </div>

      <section class="restore-panel" aria-labelledby="restore-title">
        <div>
          <h3 id="restore-title">从备份恢复</h3>
          <p>完整备份包含数据库及资料库图片；旧 .db 仅含数据库。恢复会先备份当前数据，合并校验通过的图片并替换数据库，再安全重启。现有图片不会被清空。</p>
        </div>
        <div class="restore-controls">
          <el-select v-model="selectedBackup" placeholder="选择备份文件" aria-label="选择要恢复的备份文件">
            <el-option v-for="item in backups" :key="item.name" :label="`${item.name} · ${backupScope(item)} · ${formatSize(item.size)}`" :value="item.name" />
          </el-select>
          <el-button type="danger" plain :disabled="!selectedBackup" :loading="restoreLoading" @click="confirmRestore">确认恢复所选备份</el-button>
        </div>
      </section>

      <section v-if="consistency" class="consistency-report" aria-labelledby="consistency-title">
        <h3 id="consistency-title">一致性检查报告</h3>
        <el-alert :type="consistency.healthy ? 'success' : 'error'" :title="consistency.healthy ? '未发现数据一致性问题' : `发现 ${consistency.errors.length} 个错误和 ${consistency.warnings.length} 个警告`" :closable="false" show-icon />
        <div v-for="group in consistencyGroups" :key="group.category" class="consistency-group">
          <h4>{{ group.category }}（{{ group.items.length }}）</h4>
          <article v-for="item in group.items" :key="`${item.code}-${item.entity}`">
            <strong>{{ item.message }}</strong>
            <code>{{ item.code }} · {{ item.entity }}</code>
            <p>建议：{{ item.suggestion }}</p>
          </article>
        </div>
      </section>
    </el-card>

    <el-collapse class="advanced-settings">
      <el-collapse-item name="timeouts" title="高级设置：AI 超时">
        <div v-if="diagnostics" class="timeout-grid">
          <div v-for="(value, key) in diagnostics.ai_timeouts_seconds" :key="key"><span>{{ timeoutText(key) }}</span><strong>{{ value }} 秒</strong></div>
        </div>
        <p class="settings-help">超时由服务器环境配置。AI 失败或超时不会写入半成品学习数据。</p>
      </el-collapse-item>
    </el-collapse>
  </PageShell>
</template>

<script setup lang="ts">
import PageShell from '@/components/PageShell.vue'
import PageHeader from '@/components/PageHeader.vue'
import StatusTag from '@/components/StatusTag.vue'
import WorkspaceIcon from '@/components/WorkspaceIcon.vue'
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { setThemePreference, useTheme, type ThemePreference } from '@/theme'
import HomeAppearanceDialog from '@/components/HomeAppearanceDialog.vue'
import {
  createBackup, exportLearningData, getAIConfiguration, getConsistency, getDiagnostics, listBackups,
  requestBackupRestore, testAIConnection, updateAIConfiguration,
} from '@/api/system'
import type { AIConfiguration, AIConnectionTest, BackupInfo, ConsistencyIssue, ConsistencyReport, Diagnostics } from '@/api/system'
import { createAPIToken, listAPITokens, revokeAPIToken } from '@/api/apiTokens'
import type { APITokenView } from '@/api/apiTokens'

const diagnostics = ref<Diagnostics | null>(null)
const aiConfig = ref<AIConfiguration | null>(null)
const aiForm = ref({ provider: 'mock' as 'mock' | 'deepseek', api_key: '', clear_api_key: false, base_url: 'https://api.deepseek.com', model: 'deepseek-chat' })
const consistency = ref<ConsistencyReport | null>(null)
const backups = ref<BackupInfo[]>([])
const selectedBackup = ref('')
const latestBackup = ref<BackupInfo | null>(null)
const connectionResult = ref<AIConnectionTest | null>(null)
const loading = ref(false)
const backupLoading = ref(false)
const consistencyLoading = ref(false)
const restoreLoading = ref(false)
const aiSaving = ref(false)
const aiTesting = ref(false)
const tokens = ref<APITokenView[]>([])
const tokenName = ref('uTools 采集')
const tokenDays = ref(90)
const tokenCreating = ref(false)
const createdToken = ref('')
const error = ref('')
const liveMessage = ref('')
const knownModels = ['deepseek-chat', 'deepseek-reasoner']
const { preference } = useTheme()
const appearanceOpen = ref(false)
const themeOptions: { value: ThemePreference; label: string; description: string; icon: 'system' | 'sun' | 'moon' }[] = [
  { value: 'system', label: '跟随系统', description: '随操作系统明暗设置变化', icon: 'system' },
  { value: 'light', label: '浅色主题', description: '透明玻璃与柔和环境色', icon: 'sun' },
  { value: 'dark', label: '深色主题', description: '使用深紫画布与玻璃表面', icon: 'moon' },
]
const descriptionColumns = computed(() => window.matchMedia('(max-width: 700px)').matches ? 1 : 2)
const providerChanged = computed(() => Boolean(aiConfig.value && aiForm.value.provider !== aiConfig.value.provider))
const consistencyGroups = computed(() => {
  const groups = new Map<string, ConsistencyIssue[]>()
  for (const item of [...(consistency.value?.errors ?? []), ...(consistency.value?.warnings ?? [])]) {
    const category = item.category || '其他'
    groups.set(category, [...(groups.get(category) ?? []), item])
  }
  return [...groups.entries()].map(([category, items]) => ({ category, items }))
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [nextDiagnostics, nextAIConfig, nextBackups] = await Promise.all([getDiagnostics(), getAIConfiguration(), listBackups()])
    diagnostics.value = nextDiagnostics
    aiConfig.value = nextAIConfig
    backups.value = nextBackups
    latestBackup.value = nextBackups[0] ?? nextDiagnostics.last_backup
    aiForm.value.provider = nextAIConfig.provider
    aiForm.value.base_url = nextAIConfig.base_url
    aiForm.value.model = nextAIConfig.model
    aiForm.value.api_key = ''
    aiForm.value.clear_api_key = false
    liveMessage.value = '系统状态已刷新'
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '系统状态读取失败'
  } finally { loading.value = false }
}

async function saveAIConfiguration() {
  if (aiSaving.value) return
  aiSaving.value = true
  try {
    aiConfig.value = await updateAIConfiguration(aiForm.value)
    aiForm.value.api_key = ''
    aiForm.value.clear_api_key = false
    liveMessage.value = 'AI 配置已保存并立即生效'
    ElMessage.success(liveMessage.value)
    await load()
  } catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : 'AI 配置保存失败') } finally { aiSaving.value = false }
}

async function runAIConnectionTest() {
  if (aiTesting.value) return
  aiTesting.value = true
  liveMessage.value = '正在测试 AI 连接'
  try {
    connectionResult.value = await testAIConnection()
    liveMessage.value = `AI 连接成功，响应时间 ${connectionResult.value.latency_ms} 毫秒`
    ElMessage.success(liveMessage.value)
  } catch (reason) {
    liveMessage.value = reason instanceof Error ? reason.message : 'AI 连接测试失败'
    ElMessage.error(liveMessage.value)
  } finally { aiTesting.value = false }
}

async function backup() {
  if (backupLoading.value) return
  backupLoading.value = true
  try {
    latestBackup.value = await createBackup()
    backups.value = await listBackups()
    liveMessage.value = `备份已创建：${latestBackup.value.name}`
    ElMessage.success(liveMessage.value)
  } catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : '备份失败') } finally { backupLoading.value = false }
}

function downloadBackup(name: string) { const link = document.createElement('a'); link.href = `/api/v1/system/backups/${encodeURIComponent(name)}/download`; link.download = name; link.click() }
function backupScope(item: BackupInfo) { return item.scope === 'database_and_library_images' || item.name.endsWith('.zip') ? '数据库 + 资料库图片' : '仅数据库（不含图片）' }
async function confirmRestore() {
  if (!selectedBackup.value || restoreLoading.value) return
  const expected = `恢复 ${selectedBackup.value}`
  try {
    const result = await ElMessageBox.prompt(`恢复会替换当前数据库，并在恢复前自动完整备份当前数据。完整 ZIP 会恢复图片；旧 .db 不含图片，现有图片目录会保留。请输入“${expected}”继续。`, '确认恢复数据库', {
      confirmButtonText: '确认恢复并重启', cancelButtonText: '取消', inputPlaceholder: expected,
      inputValidator: (value) => value === expected || `请输入完整确认文字：${expected}`,
      type: 'warning',
    })
    restoreLoading.value = true
    await requestBackupRestore(selectedBackup.value, result.value)
    liveMessage.value = '恢复请求已确认，系统正在安全重启'
    ElMessage.success(liveMessage.value)
  } catch (reason) {
    if (reason !== 'cancel' && reason !== 'close') ElMessage.error(reason instanceof Error ? reason.message : '备份恢复请求失败')
  } finally { restoreLoading.value = false }
}

async function download(format: 'json' | 'markdown') {
  try {
    const result = await exportLearningData(format)
    const url = URL.createObjectURL(result.blob)
    const link = document.createElement('a')
    link.href = url
    link.download = result.filename
    link.click()
    URL.revokeObjectURL(url)
    liveMessage.value = `学习数据已导出：${result.filename}`
  } catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : '学习数据导出失败') }
}
async function checkConsistency() {
  consistencyLoading.value = true
  try {
    consistency.value = await getConsistency()
    liveMessage.value = consistency.value.healthy ? '一致性检查完成，未发现问题' : `一致性检查完成，发现 ${consistency.value.errors.length} 个错误`
  } catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : '一致性检查失败') } finally { consistencyLoading.value = false }
}
async function copyDatabasePath() {
  if (!diagnostics.value) return
  try { await navigator.clipboard.writeText(diagnostics.value.database_path); ElMessage.success('数据库路径已复制') }
  catch { ElMessage.error('无法自动复制，请手动选择路径') }
}
function formatSize(size: number) { return size >= 1024 * 1024 ? `${(size / 1024 / 1024).toFixed(2)} MB` : `${(size / 1024).toFixed(1)} KB` }
function formatDate(value: string) { return new Date(value).toLocaleString('zh-CN', { hour12: false }) }
function providerText(value: string) { return value === 'deepseek' ? 'DeepSeek' : value === 'mock' ? '本地模拟' : value }
function environmentText(value: string) { return value === 'production' ? '生产环境' : value === 'development' ? '开发环境' : value }
function timeoutText(value: string) { return ({ evaluation: '回答评价', challenge_generation: '迁移挑战生成', curriculum_draft: '课程草案生成', source_credibility: '来源可信度评价', domain_skeleton: '领域地图生成', domain_starter_blueprint: '起步蓝图生成', domain_initial_world: '初始课程生成' } as Record<string, string>)[value] ?? value }
onMounted(() => { void load(); void loadTokens() })

/** 采集凭证：明文只在创建响应里出现一次，随后只保留前缀。 */
async function loadTokens() {
  try { tokens.value = await listAPITokens() } catch (reason) { error.value = reason instanceof Error ? reason.message : '凭证列表读取失败' }
}
async function createToken() {
  tokenCreating.value = true
  try {
    const created = await createAPIToken({ name: tokenName.value.trim() || 'uTools 采集', expires_in_days: tokenDays.value })
    createdToken.value = created.token
    await loadTokens()
    ElMessage.success('凭证已生成，请复制保存')
  } catch (reason) {
    ElMessage.error(reason instanceof Error ? reason.message : '凭证生成失败')
  } finally { tokenCreating.value = false }
}
async function copyToken() {
  try { await navigator.clipboard.writeText(createdToken.value); ElMessage.success('已复制') } catch { ElMessage.error('复制失败，请手动选择文本') }
}
function tokenExpiryText(item: APITokenView) {
  if (item.revoked_at) return '已撤销'
  if (item.expired) return '已过期'
  return `有效至 ${formatDate(item.expires_at)}`
}
async function revokeToken(item: APITokenView) {
  try {
    await ElMessageBox.confirm(`撤销「${item.name}」后，使用它的客户端会立即失效。`, '撤销凭证', { type: 'warning', confirmButtonText: '撤销', cancelButtonText: '取消' })
  } catch { return }
  try {
    await revokeAPIToken(item.id)
    await loadTokens()
    ElMessage.success('凭证已撤销')
  } catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : '撤销失败') }
}
</script>

<style scoped>

.appearance-settings { margin-bottom: 18px; }
.appearance-settings__copy { display: flex; align-items: center; justify-content: space-between; gap: 24px; }
.appearance-settings__eyebrow { color: var(--text-tertiary); font-size: var(--font-meta); font-weight: 500; letter-spacing: normal; }
.appearance-settings h2 { margin: 5px 0 4px; color: var(--text-primary); font-size: 18px; }
.appearance-settings p { margin: 0; color: var(--text-secondary); font-size: 13px; }
.appearance-settings__options { display: grid; grid-template-columns: repeat(3, minmax(145px, 1fr)); gap: 8px; min-width: min(100%, 510px); margin: 0; padding: 0; border: 0; }
.appearance-settings__options button { display: flex; min-height: 62px; align-items: center; gap: 10px; padding: 10px 12px; border: 1px solid var(--border-subtle); border-radius: 9px; background: var(--bg-surface); color: var(--text-secondary); text-align: left; cursor: pointer; transition: border-color 140ms ease, background-color 140ms ease, color 140ms ease, box-shadow 140ms ease; }
.appearance-settings__options button:hover { border-color: var(--card-border-hover); color: var(--text-primary); }
.appearance-settings__options button.is-active { border-color: var(--color-primary); background: var(--color-primary-soft); color: var(--color-primary); box-shadow: 0 0 0 2px var(--focus-ring); }
.appearance-settings__options svg { width: 18px; height: 18px; flex: none; }
.appearance-settings__options strong, .appearance-settings__options small { display: block; }
.appearance-settings__options strong { font-size: 13px; font-weight: 650; }
.appearance-settings__options small { margin-top: 3px; color: var(--text-tertiary); font-size: var(--font-meta); line-height: 1.5; }
.settings-help, .field-help { color: var(--text-secondary); line-height: 1.65; }
.settings-inline-alert { margin: 16px 0; }
.ai-form { max-width: 820px; }
.settings-form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
.settings-form-grid .el-select { width: 100%; }
.secret-field { display: flex; width: 100%; align-items: center; gap: 10px; }
.field-help { width: 100%; margin: 6px 0 0; font-size: 12px; }
.settings-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; }
.effective-config { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; margin-top: 22px; }
.effective-config div, .operation-result { display: grid; gap: 4px; padding: 13px 15px; border: 1px solid var(--border-subtle); border-radius: var(--radius-control); background: var(--bg-subtle); }
.effective-config span, .operation-result span { color: var(--text-secondary); font-size: 12px; }
.path-value { overflow-wrap: anywhere; }
.operation-result { margin-top: 18px; }
.restore-panel { display: grid; grid-template-columns: minmax(0, 1fr) minmax(300px, .9fr); gap: 24px; margin-top: 22px; padding-top: 22px; border-top: 1px solid var(--border-subtle); }
.restore-panel h3, .consistency-report h3 { margin-bottom: 6px; font-size: 16px; }
.restore-panel p { margin: 0; color: var(--text-secondary); line-height: 1.65; }
.restore-controls { display: grid; align-content: start; gap: 10px; }
.restore-controls .el-select { width: 100%; }
.consistency-report { margin-top: 24px; }
.consistency-group { margin-top: 18px; }
.consistency-group h4 { margin: 0 0 8px; }
.consistency-group article { display: grid; gap: 5px; padding: 12px 14px; border-left: 3px solid var(--color-danger); background: var(--color-danger-soft); }
.consistency-group article + article { margin-top: 8px; }
.consistency-group code, .consistency-group p { color: var(--text-secondary); font-size: 12px; }
.consistency-group p { margin: 0; }
.advanced-settings { border: 1px solid var(--border-default); border-radius: var(--radius-card); padding: 0 18px; background: var(--bg-surface); }
.timeout-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(190px, 1fr)); gap: 12px; }
.timeout-grid div { padding: 14px; background: var(--bg-subtle); border-radius: 8px; display: flex; justify-content: space-between; gap: 10px; }
@media (max-width: 700px) {
  .appearance-settings__copy { align-items: stretch; flex-direction: column; }
  .appearance-settings__options { min-width: 0; }
  .settings-form-grid, .effective-config, .restore-panel { grid-template-columns: minmax(0, 1fr); }
  .secret-field { align-items: flex-start; flex-direction: column; }
}

.token-secret { display: flex; align-items: center; gap: 12px; margin-top: 6px; }
.token-secret code { font-size: 13px; word-break: break-all; }
.token-list { list-style: none; margin: 0; padding: 0; display: grid; gap: 8px; }
.token-list li { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; gap: 8px 16px;
  padding: 12px 16px; border: 1px solid var(--border-default); border-radius: 12px; background: var(--bg-surface); }
.token-list li > div { min-width: 0; }
.token-list strong { font-weight: 600; margin-right: 8px; }
.token-list code { font-size: 12px; color: var(--text-secondary); }
.token-meta { grid-column: 1 / 2; display: flex; flex-wrap: wrap; gap: 4px 16px; color: var(--text-tertiary); font-size: 12px; }
</style>
