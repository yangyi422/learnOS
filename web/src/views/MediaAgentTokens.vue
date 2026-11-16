<template>
  <el-card shadow="never" aria-labelledby="media-token-title">
    <template #header><span id="media-token-title">Agent 访问令牌</span></template>
    <p>供 LifeOS Media Agent 同步媒体使用，权限为 <code>media:sync</code>，有效期为 90 天。</p>
    <form class="media-token-actions" @submit.prevent="create">
      <el-input v-model="name" maxlength="60" aria-label="Agent 令牌名称" />
      <el-button type="primary" native-type="submit" :loading="creating">创建令牌</el-button>
    </form>
    <el-alert v-if="created" type="success" :closable="false" title="令牌只显示这一次">
      <p>请妥善保存并录入 Media Agent。隐藏、离开或刷新页面后无法再次查看。</p>
      <code class="media-token-secret">{{ created.token }}</code>
      <el-button text @click="copy">复制令牌</el-button>
      <el-button text @click="created = null">隐藏令牌</el-button>
    </el-alert>
    <el-empty v-if="!tokens.length" description="还没有 Agent 访问令牌" :image-size="60" />
    <ul v-else class="media-token-list">
      <li v-for="token in tokens" :key="token.id">
        <strong>{{ token.name }}</strong>
        <code>{{ token.prefix }}…</code>
        <span v-if="token.revoked_at">已撤销</span>
        <template v-else>
          <span>{{ token.expired ? '已过期' : '有效至 ' + new Date(token.expires_at).toLocaleDateString() }}</span>
          <el-button text type="danger" @click="revoke(token)">撤销</el-button>
        </template>
      </li>
    </ul>
  </el-card>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createAPIToken, revokeAPIToken, type APITokenView } from '@/api/apiTokens'

defineProps<{ tokens: APITokenView[] }>()
const emit = defineEmits<{ changed: [] }>()
const defaultName = 'LifeOS Media Agent'
const name = ref(defaultName)
const creating = ref(false)
const created = ref<{ id: number; token: string } | null>(null)

async function create() {
  if (creating.value) return
  creating.value = true
  created.value = null
  try {
    const token = await createAPIToken({ name: name.value.trim() || defaultName, expires_in_days: 90, scope: 'media:sync' })
    created.value = { id: token.id, token: token.token }
    emit('changed')
  } catch {
    ElMessage.error('令牌创建失败')
  } finally {
    creating.value = false
  }
}

async function copy() {
  if (!created.value) return
  try {
    await navigator.clipboard.writeText(created.value.token)
    ElMessage.success('已复制令牌')
  } catch {
    ElMessage.error('复制失败，请手动选择令牌文本')
  }
}

async function revoke(token: APITokenView) {
  try {
    await ElMessageBox.confirm('撤销后，使用该令牌的 Agent 将无法同步。', '撤销 Agent 令牌', {
      type: 'warning', confirmButtonText: '撤销', cancelButtonText: '取消',
    })
  } catch { return }
  try {
    await revokeAPIToken(token.id)
    if (created.value?.id === token.id) created.value = null
    emit('changed')
  } catch {
    ElMessage.error('令牌撤销失败')
  }
}
</script>

<style scoped>
.media-token-actions, .media-token-list li { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; }
.media-token-actions { margin-block: 12px; }
.media-token-actions .el-input { max-width: 260px; }
.media-token-list { list-style: none; padding: 0; }
.media-token-secret, .media-token-list strong { overflow-wrap: anywhere; }
</style>
