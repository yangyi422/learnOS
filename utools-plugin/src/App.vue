<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  CaptureFailure,
  captureInbox,
  DEFAULT_TIMEOUT_MS,
  describeCaptureFailure,
  isHttpUrl,
  normalizeBaseUrl,
} from './api'
import { createCaptureKey } from './captureKey'
import { DEFAULT_BASE_URL, getBridge, type LifeosConfig, type PluginEnterAction } from './bridge'

const MAX_LENGTH = 10000
const SUCCESS_HOLD_MS = 420
const bridge = getBridge()

const config = ref<LifeosConfig>(bridge.getConfig())
const showConfig = ref(!isConfigComplete(config.value))
const form = ref({
  baseUrl: config.value.baseUrl || DEFAULT_BASE_URL,
  token: config.value.token,
})
const configError = ref('')

const content = ref('')
const phase = ref<'idle' | 'saving' | 'success' | 'error'>('idle')
const message = ref('')
const errorText = ref('')
const errorAction = ref<'config' | 'retry' | null>(null)
const note = ref('')
const lastEnter = ref('')

const pendingKey = ref('')
const keyContent = ref('')

const textarea = ref<HTMLTextAreaElement | null>(null)
const baseUrlInput = ref<HTMLInputElement | null>(null)
let hideTimer: number | undefined

const trimmed = computed(() => content.value.trim())
const tooLong = computed(() => trimmed.value.length > MAX_LENGTH)
const saving = computed(() => phase.value === 'saving')
const canSubmit = computed(
  () => !saving.value && !tooLong.value && trimmed.value !== '' && isConfigComplete(config.value),
)

const primaryDisabled = computed(() => {
  if (saving.value) return true
  // A failed credential turns the primary action into "open settings".
  if (phase.value === 'error' && errorAction.value === 'config') return false
  return !canSubmit.value
})

const submitLabel = computed(() => {
  if (saving.value) return '保存中…'
  if (phase.value === 'success') return '已保存'
  if (phase.value === 'error' && errorAction.value === 'config') return '去配置'
  if (phase.value === 'error' && errorAction.value === 'retry') return '重试'
  return '存入收集箱'
})

function isConfigComplete(value: LifeosConfig): boolean {
  return Boolean(normalizeBaseUrl(value.baseUrl)) && Boolean(value.token.trim())
}

async function focusComposer(selectAll = false): Promise<void> {
  await nextTick()
  const element = textarea.value
  if (!element) return
  element.focus()
  if (selectAll) {
    element.select()
  } else {
    const end = element.value.length
    element.setSelectionRange(end, end)
  }
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key !== 'Enter') return
  if (event.shiftKey) return
  // Chinese IME: Enter confirms the candidate, it must not submit.
  if (event.isComposing || event.keyCode === 229) return
  event.preventDefault()
  onPrimary()
}

function onPrimary(): void {
  if (phase.value === 'error' && errorAction.value === 'config') {
    openConfig()
    return
  }
  void submit()
}

async function submit(): Promise<void> {
  if (!canSubmit.value || saving.value) return
  const snapshot = trimmed.value
  // One key per submission; retries of the same text reuse it (idempotent).
  if (!pendingKey.value || keyContent.value !== snapshot) {
    pendingKey.value = createCaptureKey()
    keyContent.value = snapshot
  }

  phase.value = 'saving'
  errorText.value = ''
  errorAction.value = null
  message.value = ''

  try {
    await captureInbox({
      baseUrl: config.value.baseUrl,
      token: config.value.token,
      content: snapshot,
      captureKey: pendingKey.value,
      timeoutMs: DEFAULT_TIMEOUT_MS,
    })
    phase.value = 'success'
    message.value = '已进入收集箱'
    note.value = ''
    content.value = ''
    pendingKey.value = ''
    keyContent.value = ''
    bridge.clearDraft()
    scheduleHide()
  } catch (error) {
    const failure =
      error instanceof CaptureFailure
        ? error
        : new CaptureFailure('network', '提交失败。', { retryable: true, detail: String(error) })
    if (failure.kind === 'conflict') {
      // The server already saw this key with other content; let the next try mint a new one.
      pendingKey.value = ''
      keyContent.value = ''
    }
    phase.value = 'error'
    errorText.value = describeCaptureFailure(failure).message
    errorAction.value = describeCaptureFailure(failure).action
  }
}

function scheduleHide(): void {
  window.clearTimeout(hideTimer)
  hideTimer = window.setTimeout(() => {
    bridge.hideWindow()
  }, SUCCESS_HOLD_MS)
}

const runtimeText = computed(() => {
  if (bridge.runtime === 'utools') return 'uTools 桥接'
  if (bridge.runtime === 'utools-page') return 'uTools 页面兜底（preload 隔离）'
  return '浏览器兜底（preload 未生效）'
})

function lastEnterText(): string {
  // 用于排查“输入没有进到输入框”：显示 uTools 实际传来的 code/type 与文本长度。
  const raw = lastEnter.value
  if (!raw) return '还没有收到进入事件'
  try {
    const parsed = JSON.parse(raw) as { code?: string; type?: string; payloadLength?: number; payloadPreview?: string }
    return `${parsed.code || '-'} · ${parsed.type || '-'} · ${parsed.payloadLength ?? 0} 字符${parsed.payloadPreview ? ` · ${parsed.payloadPreview}` : ''}`
  } catch {
    return raw
  }
}

function openConfig(): void {
  lastEnter.value = bridge.getLastEnter?.() ?? ''
  form.value = {
    baseUrl: config.value.baseUrl || DEFAULT_BASE_URL,
    token: config.value.token,
  }
  configError.value = ''
  showConfig.value = true
  void nextTick(() => baseUrlInput.value?.focus())
}

function closeConfig(): void {
  if (!isConfigComplete(config.value)) return
  showConfig.value = false
  configError.value = ''
  void focusComposer()
}

function saveConfig(): void {
  const baseUrl = normalizeBaseUrl(form.value.baseUrl)
  const token = form.value.token.trim()
  if (!baseUrl || !isHttpUrl(baseUrl)) {
    configError.value = '地址需要以 http:// 或 https:// 开头'
    return
  }
  if (!token) {
    configError.value = '粘贴 LifeOS 设置页生成的采集凭证'
    return
  }
  config.value = { baseUrl, token }
  bridge.setConfig(config.value)
  configError.value = ''
  showConfig.value = false
  void focusComposer()
}

function readClipboard(): string {
  try {
    return bridge.readClipboard()
  } catch {
    return ''
  }
}

// uTools 匹配指令（选中文本 / 任意输入）会把文本放在 action.payload 里。它比剪贴板
// 和草稿更接近用户意图，因此优先使用：呼出即已填好，按一次 Enter 即可保存。
const prefilledFromPayload = ref(false)

function applyEnterPayload(action?: PluginEnterAction): boolean {
  const payload = typeof action?.payload === 'string' ? action.payload.trim() : ''
  if (!payload || payload.length > MAX_LENGTH) return false
  content.value = payload
  note.value = action?.code === 'lifeos-capture-text' ? '来自 uTools 输入' : '来自选中文本'
  phase.value = 'idle'
  prefilledFromPayload.value = true
  void focusComposer(true)
  return true
}

bridge.onEnter((action) => {
  applyEnterPayload(action)
})

onMounted(async () => {
  // 页面注册晚于 uTools 派发时，从桥接里取回被缓冲的进入事件。
  if (!prefilledFromPayload.value) {
    const buffered = bridge.consumeEnter?.()
    if (buffered) applyEnterPayload(buffered as PluginEnterAction)
  }
  if (prefilledFromPayload.value) return
  const draft = bridge.getDraft()
  if (draft.trim()) {
    content.value = draft
    note.value = '已恢复上次未保存内容'
    await focusComposer()
    return
  }
  const clip = readClipboard()
  if (clip.trim() && clip.trim().length <= MAX_LENGTH) {
    content.value = clip
    note.value = '来自剪贴板'
    await focusComposer(true)
    return
  }
  if (showConfig.value) {
    await nextTick()
    baseUrlInput.value?.focus()
    return
  }
  await focusComposer()
})

bridge.onOut(() => {
  if (phase.value === 'success' || !trimmed.value) {
    bridge.clearDraft()
    return
  }
  bridge.saveDraft(content.value)
})

// Typing again right after a save starts a new note: keep the window open.
watch(content, (value) => {
  if (phase.value !== 'success' || !value) return
  window.clearTimeout(hideTimer)
  phase.value = 'idle'
  message.value = ''
})

onBeforeUnmount(() => {
  window.clearTimeout(hideTimer)
})
</script>

<template>
  <main class="capture">
    <header class="capture__bar">
      <span class="capture__mark" aria-hidden="true">
        <svg width="15" height="15" viewBox="0 0 16 16" fill="none">
          <path
            d="M2.6 4.1h10.8M2.6 8h7.1M2.6 11.9h10.8"
            stroke="currentColor"
            stroke-width="1.7"
            stroke-linecap="round"
          />
          <circle cx="12.9" cy="8" r="1.45" fill="currentColor" />
        </svg>
      </span>
      <span class="capture__title">LifeOS 收集</span>
      <span v-if="note && !showConfig" class="capture__note">{{ note }}</span>
      <button
        v-if="!showConfig || isConfigComplete(config)"
        type="button"
        class="capture__link"
        @click="showConfig ? closeConfig() : openConfig()"
      >
        {{ showConfig ? '返回' : '连接设置' }}
      </button>
    </header>

    <form v-if="showConfig" class="config" novalidate @submit.prevent="saveConfig">
      <label class="config__field">
        <span class="config__label">API 地址</span>
        <input
          ref="baseUrlInput"
          v-model="form.baseUrl"
          class="config__input"
          type="text"
          inputmode="url"
          autocomplete="off"
          spellcheck="false"
          :placeholder="DEFAULT_BASE_URL"
        />
      </label>
      <div class="config__row">
        <label class="config__field config__field--token">
          <span class="config__label">采集凭证</span>
          <input
            v-model="form.token"
            class="config__input"
            type="password"
            autocomplete="off"
            spellcheck="false"
            placeholder="los_pat_…"
          />
        </label>
        <button class="config__save" type="submit">连接</button>
      </div>
      <p v-if="configError" class="config__error" role="alert">{{ configError }}</p>
      <p v-else class="config__hint">凭证在 LifeOS 设置页的「快速采集凭证」生成，只显示一次</p>
      <p class="config__hint config__hint--debug">运行模式：{{ runtimeText }}</p>
      <p class="config__hint config__hint--debug">最近一次进入：{{ lastEnterText() }}</p>
    </form>

    <form v-else class="composer" novalidate @submit.prevent="onPrimary">
      <textarea
        ref="textarea"
        v-model="content"
        class="composer__input"
        :maxlength="MAX_LENGTH"
        :readonly="saving"
        rows="3"
        aria-label="要存入收集箱的内容"
        placeholder="想法、待办或链接"
        spellcheck="false"
        @keydown="onKeydown"
      ></textarea>
      <div class="composer__foot">
        <p v-if="errorText" class="composer__error" role="alert">{{ errorText }}</p>
        <p v-else-if="phase === 'success'" class="composer__success" role="status">{{ message }}</p>
        <p v-else class="composer__hint">
          <kbd>Enter</kbd> 保存
          <span class="composer__sep">·</span>
          <kbd>Shift</kbd><span class="composer__plus">+</span><kbd>Enter</kbd> 换行
        </p>
        <button class="composer__submit" type="submit" :disabled="primaryDisabled">
          {{ submitLabel }}
        </button>
      </div>
    </form>
  </main>
</template>

<style scoped>
.capture {
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 620px;
  height: 100%;
  margin: 0 auto;
  padding: 16px 20px 18px;
  gap: 10px;
}

.capture__bar {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 22px;
  flex: none;
}

.capture__mark {
  display: grid;
  place-items: center;
  width: 22px;
  height: 22px;
  border-radius: 7px;
  background: var(--color-accent-soft);
  color: var(--color-accent);
}

.capture__title {
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.01em;
  color: var(--color-text);
}

.capture__note {
  flex: 1;
  min-width: 0;
  padding-left: 4px;
  font-size: 12px;
  color: var(--color-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.capture__link {
  margin-left: auto;
  padding: 3px 6px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  font-size: 12px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: color 120ms ease, background-color 120ms ease;
}

.capture__link:hover {
  color: var(--color-accent);
  background: var(--color-accent-soft);
}

/* Composer: the text is the interface; chrome stays quiet. */
.composer {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  padding: 12px 14px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-surface);
  background: var(--color-surface);
  box-shadow: var(--shadow-composer);
  transition: border-color 140ms ease, box-shadow 140ms ease;
}

.composer:focus-within {
  border-color: var(--color-accent-ring);
  box-shadow: var(--shadow-composer), 0 0 0 3px var(--color-accent-soft);
}

.composer__input {
  flex: 1;
  min-height: 0;
  width: 100%;
  padding: 0;
  border: 0;
  background: transparent;
  font-family: inherit;
  font-size: 16px;
  line-height: 1.55;
  color: var(--color-text);
  resize: none;
}

.composer__input::placeholder {
  color: var(--color-text-muted);
}

.composer__input:focus {
  outline: none;
}

.composer__input[readonly] {
  color: var(--color-text-secondary);
}

.composer__input::-webkit-scrollbar {
  width: 8px;
}

.composer__input::-webkit-scrollbar-thumb {
  border: 2px solid var(--color-surface);
  border-radius: 999px;
  background: rgba(35, 48, 84, 0.18);
}

.composer__foot {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 30px;
  flex: none;
}

.composer__hint,
.composer__error,
.composer__success {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 12px;
  line-height: 1.35;
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.composer__error {
  color: var(--color-danger);
}

.composer__success {
  color: var(--color-success);
  font-weight: 500;
}

.composer__hint kbd {
  display: inline-block;
  min-width: 18px;
  padding: 1px 5px;
  border: 1px solid var(--color-border);
  border-radius: 5px;
  background: var(--color-surface-quiet);
  font-family: inherit;
  font-size: 11px;
  line-height: 1.4;
  text-align: center;
  color: var(--color-text-secondary);
}

.composer__sep {
  padding: 0 4px;
  color: var(--color-text-muted);
}

.composer__plus {
  padding: 0 1px;
  color: var(--color-text-muted);
}

.composer__submit,
.config__save {
  flex: none;
  padding: 7px 16px;
  border: 0;
  border-radius: var(--radius-pill);
  background: var(--color-accent);
  box-shadow: var(--shadow-button);
  font-size: 13px;
  font-weight: 600;
  color: #ffffff;
  cursor: pointer;
  transition: background-color 140ms ease, transform 140ms ease, box-shadow 140ms ease;
}

.composer__submit:hover:not(:disabled),
.config__save:hover {
  background: var(--color-accent-hover);
}

.composer__submit:active:not(:disabled),
.config__save:active {
  transform: translateY(1px);
  box-shadow: none;
}

.composer__submit:disabled {
  background: rgba(35, 48, 84, 0.14);
  box-shadow: none;
  color: var(--color-text-muted);
  cursor: default;
}

/* Config: same card, two fields, no admin-form density. */
.config {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
  min-height: 0;
  padding: 11px 14px 9px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-surface);
  background: var(--color-surface);
  box-shadow: var(--shadow-composer);
}

.config__field {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.config__row {
  display: flex;
  align-items: flex-end;
  gap: 10px;
}

.config__field--token {
  flex: 1;
}

.config__label {
  font-size: 11.5px;
  color: var(--color-text-muted);
}

.config__input {
  width: 100%;
  height: 32px;
  padding: 0 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-field);
  background: var(--color-surface);
  font-family: inherit;
  font-size: 13px;
  color: var(--color-text);
  transition: border-color 140ms ease, box-shadow 140ms ease;
}

.config__input::placeholder {
  color: var(--color-text-muted);
}

.config__input:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px var(--color-accent-soft);
}

.config__save {
  margin-bottom: 0;
}

.config__hint,
.config__error {
  margin: 0;
  font-size: 11px;
  line-height: 1.3;
  color: var(--color-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.config__error {
  color: var(--color-danger);
}
</style>
