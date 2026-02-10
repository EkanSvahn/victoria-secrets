<script setup lang="ts">
import { computed, ref } from 'vue'
import { createSecret } from '../api/client'
import { encryptText } from '../api/crypto'

const text = ref('')
const ttlMinutes = ref(60)
const password = ref('')
const error = ref('')
const link = ref('')
const loading = ref(false)

const canSubmit = computed(() => text.value.trim().length > 0 && ttlMinutes.value >= 1)

async function onSubmit() {
  if (!canSubmit.value) return
  error.value = ''
  link.value = ''
  loading.value = true
  try {
    const encrypted = await encryptText(text.value, password.value || undefined)
    const created = await createSecret({
      meta: encrypted.meta,
      ciphertext: encrypted.ciphertext,
      kind: 'text',
      ttl_seconds: ttlMinutes.value * 60
    })

    const base = `${window.location.origin}/s/${created.id}`
    link.value = encrypted.linkSecret ? `${base}#${encrypted.linkSecret}` : base
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
    <p>Create a one-time secret link with client-side encryption.</p>

    <label>
      Secret text
      <textarea v-model="text" rows="8" placeholder="Paste secret text..."></textarea>
    </label>

    <div class="row">
      <label>
        TTL (minutes)
        <input v-model.number="ttlMinutes" type="number" min="1" max="1440" />
      </label>

      <label>
        Optional password
        <input v-model="password" type="password" placeholder="Leave blank for auto key in URL fragment" />
      </label>
    </div>

    <button :disabled="!canSubmit || loading" @click="onSubmit">{{ loading ? 'Creating...' : 'Create Link' }}</button>

    <p v-if="error" class="error">{{ error }}</p>
    <label v-if="link">
      Share link
      <input :value="link" readonly @focus="$event.target.select()" />
    </label>
  </section>
</template>
