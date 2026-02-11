<script setup lang="ts">
import { computed, ref } from 'vue'
import { createSecret } from '../api/client'
import { encryptFile, encryptText } from '../api/crypto'

const MAX_FILE_SIZE_BYTES = 4 * 1024 * 1024
const REQUIRE_PASSWORD = (import.meta.env.VITE_REQUIRE_PASSWORD ?? 'false').toLowerCase() === 'true'
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

const canSubmit = computed(() => {
  if (lifetimeMode.value === 'views' && views.value < 1) return false
  if (lifetimeMode.value === 'ttl' && ttlMinutes.value < 1) return false
  if (REQUIRE_PASSWORD && password.value.trim().length < 8) return false
  if (mode.value === 'text') return text.value.trim().length > 0
  return file.value !== null
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
    if (REQUIRE_PASSWORD && password.value.trim().length < 8) {
      throw new Error('Password is required and must be at least 8 characters.')
    }
    if (mode.value === 'file') {
      if (!file.value) {
        throw new Error('Please select a file.')
      }
      if (file.value.size > MAX_FILE_SIZE_BYTES) {
        throw new Error('File too large. Max 4 MiB.')
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
    if (REQUIRE_PASSWORD && encrypted.linkSecret) {
      throw new Error('Password-only mode is enabled; fragment key links are not allowed.')
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to create secret'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <section class="card">
    <h1>VaultDrop</h1>
    <p>Create a one-time secret link for text or a file with client-side encryption.</p>

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
      File (max 4 MiB)
      <input type="file" @change="onFileChange" />
      <small v-if="file">{{ file.name }} ({{ file.size }} bytes)</small>
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
          max="100"
        />
        <input
          v-else
          v-model.number="ttlMinutes"
          type="number"
          min="1"
          max="1440"
        />
      </label>

      <label>
        Optional password
        <input
          v-model="password"
          type="password"
          :placeholder="REQUIRE_PASSWORD ? 'Required (min 8 chars)' : 'Leave blank for auto key in URL fragment'"
        />
      </label>
    </div>
    <p v-if="REQUIRE_PASSWORD">
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
</style>
