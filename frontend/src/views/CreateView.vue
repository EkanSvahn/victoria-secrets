<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { createSecret, getStatus, toApiErrorMessage } from '../api/client'
import { encryptFile, encryptText } from '../api/crypto'

type StatusPolicy = {
  maxViews: number
  maxTTLMinutes: number
  maxFileBytes: number
  allowedFileMIMEs: string[]
  requirePassword: boolean
}

const status = ref<StatusPolicy>({
  maxViews: 100,
  maxTTLMinutes: 24 * 60,
  maxFileBytes: 4 * 1024 * 1024,
  allowedFileMIMEs: ['application/pdf', 'image/png', 'image/jpeg', 'text/plain', 'application/octet-stream'],
  requirePassword: (import.meta.env.VITE_REQUIRE_PASSWORD ?? 'false').toLowerCase() === 'true'
})

const text = ref('')
const mode = ref<'text' | 'file'>('text')
const file = ref<File | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const lifetimeMode = ref<'views' | 'ttl'>('views')
const views = ref(1)
const ttlMinutes = ref(60)
const password = ref('')
const error = ref('')
const link = ref('')
const loading = ref(false)
const copied = ref(false)
const statusLoaded = ref(false)

const policyMaxFileText = computed(() => formatBytes(status.value.maxFileBytes))
const allowedMimeText = computed(() => (status.value.allowedFileMIMEs.length > 0 ? status.value.allowedFileMIMEs.join(', ') : 'Any'))
const policyLine = computed(
  () => `limits: ${status.value.maxViews} views · ${status.value.maxTTLMinutes} min · ${policyMaxFileText.value} max · mime ${allowedMimeText.value}`
)
const summaryText = computed(() => {
  const lifetime = lifetimeMode.value === 'views' ? `${views.value} view${views.value > 1 ? 's' : ''}` : `${ttlMinutes.value} min TTL`
  const pwd = status.value.requirePassword ? 'password required' : password.value.trim() ? 'password enabled' : 'auto key fragment'
  return `expires: ${lifetime} · ${pwd}`
})

const canSubmit = computed(() => {
  if (lifetimeMode.value === 'views' && views.value < 1) return false
  if (lifetimeMode.value === 'ttl' && ttlMinutes.value < 1) return false
  if (views.value > status.value.maxViews) return false
  if (ttlMinutes.value > status.value.maxTTLMinutes) return false
  if (status.value.requirePassword && password.value.trim().length < 8) return false
  if (mode.value === 'text') return text.value.trim().length > 0
  return file.value !== null
})

onMounted(async () => {
  try {
    const backendStatus = await getStatus()
    status.value = {
      maxViews: backendStatus.max_views,
      maxTTLMinutes: Math.max(1, Math.floor(backendStatus.max_ttl_seconds / 60)),
      maxFileBytes: backendStatus.max_file_bytes,
      allowedFileMIMEs: backendStatus.allowed_file_mime_types,
      requirePassword: backendStatus.require_password
    }
    if (views.value > status.value.maxViews) views.value = status.value.maxViews
    if (ttlMinutes.value > status.value.maxTTLMinutes) ttlMinutes.value = status.value.maxTTLMinutes
    statusLoaded.value = true
  } catch {
    statusLoaded.value = false
  }
})

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  file.value = input.files?.[0] ?? null
}

function openFilePicker() {
  fileInput.value?.click()
}

function resetForm() {
  text.value = ''
  file.value = null
  password.value = ''
  copied.value = false
  error.value = ''
  link.value = ''
  mode.value = 'text'
  views.value = 1
  lifetimeMode.value = 'views'
}

async function copyLink() {
  if (!link.value) return
  await navigator.clipboard.writeText(link.value)
  copied.value = true
  window.setTimeout(() => {
    copied.value = false
  }, 1200)
}

async function onSubmit() {
  if (!canSubmit.value) return
  error.value = ''
  link.value = ''
  copied.value = false
  loading.value = true
  try {
    if (status.value.requirePassword && password.value.trim().length < 8) {
      throw new Error('Password is required and must be at least 8 characters.')
    }
    if (mode.value === 'file') {
      if (!file.value) throw new Error('Please select a file.')
      if (file.value.size > status.value.maxFileBytes) throw new Error(`File too large. Max ${policyMaxFileText.value}.`)
      if (status.value.allowedFileMIMEs.length > 0 && !status.value.allowedFileMIMEs.includes(file.value.type || 'application/octet-stream')) {
        throw new Error('File type is not allowed by server policy.')
      }
    }

    const encrypted =
      mode.value === 'text'
        ? await encryptText(text.value, password.value || undefined)
        : await encryptFile(file.value as File, password.value || undefined)

    const created = await createSecret({
      meta: encrypted.meta,
      ciphertext: encrypted.ciphertext,
      kind: mode.value,
      ...(lifetimeMode.value === 'ttl' ? { ttl_seconds: ttlMinutes.value * 60 } : { views: views.value })
    })

    const base = `${window.location.origin}/s/${created.id}`
    link.value = encrypted.linkSecret ? `${base}#${encrypted.linkSecret}` : base

    if (status.value.requirePassword && encrypted.linkSecret) {
      throw new Error('Password-only mode is enabled; fragment key links are not allowed.')
    }
  } catch (err) {
    error.value = toApiErrorMessage(err, 'Failed to create secret.')
  } finally {
    loading.value = false
  }
}

function formatBytes(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${Math.floor(bytes / (1024 * 1024))} MiB`
  if (bytes >= 1024) return `${Math.floor(bytes / 1024)} KiB`
  return `${bytes} B`
}
</script>

<template>
  <section class="card create-card">
    <p class="intro">End-to-end encrypted in your browser. The server stores ciphertext only, and each shared link decrypts once before auto-destruction.</p>

    <template v-if="!link">
      <div class="mode-switch" role="group" aria-label="secret mode">
        <button type="button" class="toggle" :class="{ active: mode === 'text' }" @click="mode = 'text'">note</button>
        <button type="button" class="toggle" :class="{ active: mode === 'file' }" @click="mode = 'file'">file</button>
      </div>

      <label v-if="mode === 'text'" class="field">
        Secret
        <textarea v-model="text" rows="10" placeholder="Paste your secret text..."></textarea>
      </label>

      <label v-else class="field">
        File
        <input ref="fileInput" class="file-input-hidden" type="file" @change="onFileChange" />
        <button type="button" class="file-picker" @click="openFilePicker">click to add file</button>
        <small v-if="file">{{ file.name }} ({{ formatBytes(file.size) }})</small>
        <small v-else class="file-hint">No file selected</small>
      </label>

      <div class="action-row">
        <p class="summary">{{ summaryText }}</p>
        <button :disabled="!canSubmit || loading" @click="onSubmit">{{ loading ? 'creating...' : 'create' }}</button>
      </div>

      <p v-if="error" class="error">{{ error }}</p>

      <details class="advanced">
        <summary>Advanced</summary>

        <div class="advanced-grid">
          <label>
            Expiration mode
            <select v-model="lifetimeMode">
              <option value="views">Views</option>
              <option value="ttl">Time (minutes)</option>
            </select>
          </label>

          <label>
            {{ lifetimeMode === 'views' ? 'Views' : 'TTL (minutes)' }}
            <input
              v-if="lifetimeMode === 'views'"
              v-model.number="views"
              type="number"
              min="1"
              :max="status.maxViews"
            />
            <input
              v-else
              v-model.number="ttlMinutes"
              type="number"
              min="1"
              :max="status.maxTTLMinutes"
            />
          </label>

          <label>
            Password
            <input
              v-model="password"
              type="password"
              :placeholder="status.requirePassword ? 'Required (min 8 chars)' : 'Optional'"
            />
          </label>
        </div>

        <p class="policy-line">
          {{ policyLine }}
        </p>
        <p v-if="status.requirePassword" class="policy-line warn">policy: password-only mode enabled.</p>
        <p v-if="!statusLoaded" class="policy-line warn">status unavailable: using safe fallback limits.</p>
      </details>
    </template>

    <template v-else>
      <h2>Share Link</h2>
      <p class="intro">Send this once. The secret is consumed on open.</p>

      <div class="share-row">
        <input class="share-link-input" :value="link" readonly @focus="$event.target.select()" />
        <button type="button" class="copy" @click="copyLink">{{ copied ? 'Copied' : 'Copy' }}</button>
      </div>

      <button type="button" class="ghost" @click="resetForm">New Secret</button>
      <p v-if="error" class="error">{{ error }}</p>
    </template>
  </section>
</template>

<style scoped>
.intro {
  max-width: 64ch;
  font-size: 1.06rem;
  font-weight: 500;
  color: var(--text);
  line-height: 1.45;
}

.create-card {
  padding: 1.25rem;
}

.mode-switch {
  display: grid;
  grid-template-columns: 1fr 1fr;
  border: none;
  border-radius: 0;
  width: 170px;
  background: color-mix(in oklab, var(--panel) 84%, #1f1f1f);
}

.toggle {
  border: 0;
  border-right: 0;
  border-radius: 0;
  background: transparent;
  color: var(--text-muted);
  min-height: 34px;
  text-align: center;
  font-family: var(--mono);
  letter-spacing: 0.02em;
  text-transform: lowercase;
  padding: 0.3rem 0.5rem;
}

.toggle:last-child {
  border-right: 0;
}

.toggle.active {
  background: #1d1d1d;
  color: var(--text);
  box-shadow: inset 0 0 0 1px var(--accent);
}

.field textarea {
  min-height: 255px;
  border: 1px solid color-mix(in oklab, var(--line-strong) 86%, #8b8b8b);
  background: #151515;
  padding: 0.72rem;
  box-shadow: 0 8px 22px rgba(0, 0, 0, 0.18);
}

.file-input-hidden {
  display: none;
}

.file-picker {
  width: fit-content;
  min-width: 180px;
  margin: 0.35rem auto 0;
}

.file-hint {
  color: var(--text-muted);
  margin-top: 0.45rem;
  text-align: center;
}

.field small {
  margin-top: 0.45rem;
  text-align: center;
}

.action-row {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: end;
  gap: 1rem;
}

.summary {
  color: var(--text-muted);
  font-family: var(--mono);
  font-size: 0.9rem;
  line-height: 1.35;
}

.advanced {
  border-top: none;
  padding-top: 0.84rem;
  background: color-mix(in oklab, var(--panel) 84%, #1f1f1f);
  padding: 0.88rem;
}

.advanced summary {
  cursor: pointer;
  font-family: var(--mono);
  color: var(--text-muted);
  font-size: 0.86rem;
}

.advanced-grid {
  margin-top: 0.72rem;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.65rem;
}

.policy-line {
  margin-top: 0.35rem;
  font-size: 0.84rem;
}

.warn {
  color: var(--danger);
}

.share-row {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 0.5rem;
}

.share-link-input {
  border: 1px solid var(--line-strong);
  background: #151515;
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.15);
}

.copy {
  min-width: 90px;
}

.ghost {
  background: transparent;
  color: var(--text);
}

@media (max-width: 820px) {
  .advanced-grid {
    grid-template-columns: 1fr;
  }

  .share-row {
    grid-template-columns: 1fr;
  }

  .action-row {
    grid-template-columns: 1fr;
    align-items: start;
  }
}
</style>
