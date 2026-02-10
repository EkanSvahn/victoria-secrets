<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { consumeSecret } from '../api/client'
import { decryptText } from '../api/crypto'

const route = useRoute()
const secret = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const consumed = ref(false)

async function openSecret() {
  const id = String(route.params.id || '')
  if (!id) {
    error.value = 'Invalid secret id'
    return
  }

  loading.value = true
  error.value = ''
  try {
    const response = await consumeSecret(id)
    const fragment = window.location.hash.replace('#', '')
    secret.value = await decryptText(response.ciphertext, response.meta, fragment, password.value || undefined)
    consumed.value = true
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to open secret'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  if (window.location.hash) {
    void openSecret()
  }
})
</script>

<template>
  <section class="card">
    <h1>Open Secret</h1>
    <p>This operation consumes the secret and cannot be retried.</p>

    <label>
      Password (required only if sender used one)
      <input v-model="password" type="password" placeholder="Optional" />
    </label>

    <button :disabled="loading || consumed" @click="openSecret">{{ loading ? 'Opening...' : 'Open Secret' }}</button>

    <p v-if="error" class="error">{{ error }}</p>

    <label v-if="secret">
      Decrypted content
      <textarea :value="secret" rows="8" readonly></textarea>
    </label>
  </section>
</template>
