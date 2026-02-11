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
const lifetimeMode = ref<'views' | 'ttl'>('views')
const views = ref(1)
const ttlMinutes = ref(60)
const password = ref('')
const error = ref('')
const link = ref('')
const loading = ref(false)
const statusLoaded = ref(false)

const policyMaxFileText = computed(() => formatBytes(status.value.maxFileBytes))
const allowedMimeText = computed(() =>
  status.value.allowedFileMIMEs.length > 0 ? status.value.allowedFileMIMEs.join(', ') : 'Any'
)
const passwordHelpText = computed(() =>
  status.value.requirePassword ? 'Required by policy (min 8 chars).' : 'Optional. Leave blank to use URL fragment key.'
)

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
    // Keep safe local defaults if backend status is temporarily unavailable.
    statusLoaded.value = false
  }
})

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const picked = input.files?.[0] ?? null
  file.value = picked
}

async function onSubmit() {
  if (!canSubmit.value) return
  error.value = ''
  link.value = ''
  loading.value = true
  try {
    if (status.value.requirePassword && password.value.trim().length < 8) {
      throw new Error('Password is required and must be at least 8 characters.')
    }
    if (mode.value === 'file') {
      if (!file.value) {
        throw new Error('Please select a file.')
      }
      if (file.value.size > status.value.maxFileBytes) {
        throw new Error(`File too large. Max ${policyMaxFileText.value}.`)
      }
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
      ...(lifetimeMode.value === 'ttl'
        ? { ttl_seconds: ttlMinutes.value * 60 }
        : { views: views.value })
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
  <section class="card">
    <h1>VaultDrop</h1>
    <p>Create a one-time secret link for text or a file with client-side encryption.</p>
    <p class="policy">
      Policy:
      max views {{ status.maxViews }},
      max TTL {{ status.maxTTLMinutes }} minutes,
      max file {{ policyMaxFileText }},
      allowed MIME {{ allowedMimeText }},
      password {{ status.requirePassword ? 'required' : 'optional' }}.
    </p>
    <p v-if="!statusLoaded" class="policy warn">Using fallback limits because `/api/status` is unavailable.</p>

    <div class="row mode-row">
      <label class="mode-option">
        <input v-model="mode" type="radio" value="text" />
        Text
      </label>
      <label class="mode-option">
        <input v-model="mode" type="radio" value="file" />
        File
      </label>
    </div>

    <label v-if="mode === 'text'">
      Secret text
      <textarea v-model="text" rows="8" placeholder="Paste secret text..."></textarea>
    </label>
    <label v-else>
      File (max {{ policyMaxFileText }})
      <input type="file" @change="onFileChange" />
      <small v-if="file">{{ file.name }} ({{ file.size }} bytes)</small>
      <small>Allowed MIME types: {{ allowedMimeText }}</small>
    </label>

    <div class="row">
      <label>
        Expiration mode
        <select v-model="lifetimeMode">
          <option value="views">Views (default)</option>
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
        Optional password
        <input
          v-model="password"
          type="password"
          :placeholder="status.requirePassword ? 'Required (min 8 chars)' : 'Leave blank for auto key in URL fragment'"
        />
        <small>{{ passwordHelpText }}</small>
      </label>
    </div>
    <p v-if="status.requirePassword">
      Organization policy: password-only mode is enabled. Link fragments are disabled.
    </p>

    <button :disabled="!canSubmit || loading" @click="onSubmit">{{ loading ? 'Creating...' : 'Create Link' }}</button>

    <p v-if="error" class="error">{{ error }}</p>
    <label v-if="link">
      Share link
      <input :value="link" readonly @focus="$event.target.select()" />
    </label>
  </section>
</template>

<style scoped>
.mode-row {
  grid-template-columns: 1fr 1fr;
  align-items: center;
}

.mode-option {
  display: flex;
  flex-direction: row;
  gap: 0.5rem;
  align-items: center;
}

.policy {
  margin: 0;
  font-size: 0.92rem;
  color: #374b63;
}

.warn {
  color: #8a5b08;
}
</style>
