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
const isFileDragOver = ref(false)
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
  setSelectedFile(input.files?.[0] ?? null)
}

function openFilePicker() {
  fileInput.value?.click()
}

function setSelectedFile(next: File | null) {
  file.value = next
}

function onFileDragOver(event: DragEvent) {
  if (mode.value !== 'file') return
  event.preventDefault()
  isFileDragOver.value = true
}

function onFileDragEnter(event: DragEvent) {
  if (mode.value !== 'file') return
  event.preventDefault()
  isFileDragOver.value = true
}

function onFileDragLeave(event: DragEvent) {
  if (mode.value !== 'file') return
  event.preventDefault()
  const nextTarget = event.relatedTarget as Node | null
  const currentTarget = event.currentTarget as Node | null
  if (currentTarget && nextTarget && currentTarget.contains(nextTarget)) return
  isFileDragOver.value = false
}

function onFileDrop(event: DragEvent) {
  if (mode.value !== 'file') return
  event.preventDefault()
  isFileDragOver.value = false
  const dropped = event.dataTransfer?.files?.[0] ?? null
  if (dropped) setSelectedFile(dropped)
}

function resetForm() {
  text.value = ''
  file.value = null
  password.value = ''
  copied.value = false
  error.value = ''
  link.value = ''
  mode.value = 'text'
  isFileDragOver.value = false
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

      <label class="field">
        {{ mode === 'text' ? 'Secret' : 'File' }}
        <input ref="fileInput" class="file-input-hidden" type="file" @change="onFileChange" />
        <div
          class="content-panel"
          :class="{ 'is-file': mode === 'file', 'is-drag-over': mode === 'file' && isFileDragOver }"
          @dragenter="onFileDragEnter"
          @dragover="onFileDragOver"
          @dragleave="onFileDragLeave"
          @drop="onFileDrop"
        >
          <Transition name="panel-fade" mode="out-in">
            <textarea
              v-if="mode === 'text'"
              key="text"
              v-model="text"
              rows="10"
              placeholder="Paste your secret text..."
            ></textarea>

            <div v-else key="file" class="file-panel">
              <button type="button" class="file-picker" @click="openFilePicker">
                {{ isFileDragOver ? 'drop file to attach' : 'click to add file' }}
              </button>
              <small
                v-if="file"
                class="file-selection"
                :title="`${file.name} (${formatBytes(file.size)})`"
              >
                {{ file.name }} ({{ formatBytes(file.size) }})
              </small>
              <small v-else class="file-hint">{{ isFileDragOver ? 'Release to attach file' : 'No file selected' }}</small>
            </div>
          </Transition>
        </div>
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
            <div class="mini-switch" role="group" aria-label="expiration mode">
              <button
                type="button"
                class="mini-toggle"
                :class="{ active: lifetimeMode === 'views' }"
                @click="lifetimeMode = 'views'"
              >
                views
              </button>
              <button
                type="button"
                class="mini-toggle"
                :class="{ active: lifetimeMode === 'ttl' }"
                @click="lifetimeMode = 'ttl'"
              >
                time
              </button>
            </div>
          </label>

          <label>
            {{ lifetimeMode === 'views' ? 'Views' : 'TTL (minutes)' }}
            <div class="unit-field">
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
              <span class="unit-suffix">{{ lifetimeMode === 'views' ? 'views' : 'min' }}</span>
            </div>
          </label>

          <label class="password-advanced-field">
            Password
            <div class="unit-field">
              <input
                v-model="password"
                type="password"
                :placeholder="status.requirePassword ? 'Enter password' : 'Leave empty for link key'"
              />
              <span class="unit-suffix" :class="{ 'unit-suffix-accent': status.requirePassword }">
                {{ status.requirePassword ? 'required' : 'optional' }}
              </span>
            </div>
            <small class="field-meta">
              {{
                status.requirePassword
                  ? 'Minimum 8 characters. Link fragment key is disabled by policy.'
                  : 'Adds password-based decryption. Otherwise a one-time key is stored in the URL fragment.'
              }}
            </small>
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
      <p class="state-line">share link · opens once · auto-destroys after access</p>

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

.create-card button {
  min-height: 38px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
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
  min-height: 38px;
  text-align: center;
  font-family: var(--mono);
  letter-spacing: 0.02em;
  text-transform: lowercase;
  padding: 0.3rem 0.5rem;
  transition:
    color 100ms ease,
    background-color 100ms ease,
    box-shadow 100ms ease;
}

.toggle:last-child {
  border-right: 0;
}

.toggle.active {
  background: #1d1d1d;
  color: var(--text);
  box-shadow: inset 0 0 0 1px var(--accent);
}

.file-input-hidden {
  display: none;
}

.content-panel {
  min-height: 255px;
  border: 1px solid color-mix(in oklab, var(--line-strong) 86%, #8b8b8b);
  background: #151515;
  box-shadow: 0 8px 22px rgba(0, 0, 0, 0.18);
  transition:
    border-color 120ms ease,
    box-shadow 120ms ease,
    background-color 120ms ease;
}

.content-panel textarea {
  width: 100%;
  min-height: 255px;
  border: none;
  background: transparent;
  padding: 0.72rem;
  box-shadow: none;
  display: block;
}

.content-panel.is-file {
  display: grid;
  place-items: center;
  padding: 0.9rem;
}

.content-panel.is-drag-over {
  border-color: var(--accent);
  box-shadow:
    0 8px 22px rgba(0, 0, 0, 0.18),
    inset 0 0 0 1px var(--accent);
  background: #121712;
}

.file-panel {
  width: 100%;
  min-height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.file-picker {
  width: fit-content;
  min-width: 180px;
  margin: 0;
}

.file-hint {
  color: var(--text-muted);
  margin-top: 0.65rem;
  text-align: center;
}

.file-selection {
  margin-top: 0.65rem;
  text-align: center;
  max-width: 100%;
  width: min(100%, 520px);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.panel-fade-enter-active,
.panel-fade-leave-active {
  transition: opacity 110ms ease;
}

.panel-fade-enter-from,
.panel-fade-leave-to {
  opacity: 0;
}

.field small {
  margin-top: 0.45rem;
  text-align: center;
}

.field-meta {
  margin-top: 0.35rem;
  color: var(--text-muted);
  font-family: var(--mono);
  font-size: 0.75rem;
  line-height: 1.3;
  text-align: left;
}

.action-row {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: end;
  gap: 1rem;
}

.action-row button:disabled {
  opacity: 1;
  background: #101010;
  color: #727272;
  box-shadow: inset 0 0 0 1px var(--line);
}

.summary {
  color: var(--text-muted);
  font-family: var(--mono);
  font-size: 0.9rem;
  line-height: 1.35;
}

.state-line {
  color: var(--text);
  font-family: var(--mono);
  font-size: 0.9rem;
  line-height: 1.35;
}

.advanced {
  border-top: none;
  background: color-mix(in oklab, var(--panel) 84%, #1f1f1f);
  padding: 0.95rem;
}

.advanced summary {
  cursor: pointer;
  font-family: var(--mono);
  color: var(--text-muted);
  font-size: 0.86rem;
  line-height: 1.2;
}

.advanced-grid {
  margin-top: 0.8rem;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
  align-items: start;
}

.advanced-grid label {
  gap: 0.4rem;
  font-size: 0.86rem;
}

.password-advanced-field {
  grid-column: span 2;
}

.mini-switch {
  display: grid;
  grid-template-columns: 1fr 1fr;
  background: #101010;
  border: 1px solid var(--line);
  min-height: 38px;
}

.mini-toggle {
  border: none;
  background: transparent;
  color: var(--text-muted);
  font-family: var(--mono);
  font-size: 0.84rem;
  padding: 0.35rem 0.45rem;
  min-height: 36px;
  transition:
    color 100ms ease,
    background-color 100ms ease,
    box-shadow 100ms ease;
}

.mini-toggle.active {
  color: var(--text);
  box-shadow: inset 0 0 0 1px var(--accent);
  background: #151515;
}

.unit-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: stretch;
  min-height: 38px;
  border: 1px solid var(--line);
  background: #101010;
}

.unit-field input {
  border: none;
  background: transparent;
  min-width: 0;
  padding: 0.42rem 0.5rem;
}

.unit-field input:focus {
  outline: none;
}

.unit-field:focus-within {
  box-shadow: inset 0 0 0 1px var(--accent);
}

.unit-suffix {
  display: inline-flex;
  align-items: center;
  padding: 0 0.55rem;
  border-left: 1px solid var(--line);
  color: var(--text-muted);
  font-family: var(--mono);
  font-size: 0.8rem;
  white-space: nowrap;
  min-width: 84px;
  justify-content: center;
}

.unit-suffix-accent {
  color: var(--accent);
}

.policy-line {
  margin-top: 0.55rem;
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

  .password-advanced-field {
    grid-column: auto;
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
