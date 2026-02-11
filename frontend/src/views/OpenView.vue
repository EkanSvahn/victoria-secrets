<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { consumeSecret } from '../api/client'
import { decryptFile, decryptText } from '../api/crypto'

const route = useRoute()
const secret = ref('')
const fileName = ref('')
const fileUrl = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const consumed = ref(false)
const isFile = ref(false)

const hasResult = computed(() => (isFile.value ? fileUrl.value !== '' : secret.value !== ''))

function revokeFileUrl() {
  if (fileUrl.value) {
    URL.revokeObjectURL(fileUrl.value)
    fileUrl.value = ''
  }
}

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
    if (response.kind === 'file') {
      const file = await decryptFile(response.ciphertext, response.meta, fragment, password.value || undefined)
      revokeFileUrl()
      fileUrl.value = URL.createObjectURL(file)
      fileName.value = file.name
      isFile.value = true
      secret.value = ''
    } else {
      secret.value = await decryptText(response.ciphertext, response.meta, fragment, password.value || undefined)
      isFile.value = false
      fileName.value = ''
      revokeFileUrl()
    }
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

onBeforeUnmount(() => {
  revokeFileUrl()
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

    <label v-if="hasResult && !isFile">
      Decrypted content
      <textarea :value="secret" rows="8" readonly></textarea>
    </label>
    <div v-if="hasResult && isFile">
      <p>File decrypted successfully.</p>
      <a class="download-link" :href="fileUrl" :download="fileName">Download {{ fileName }}</a>
    </div>
  </section>
</template>

<style scoped>
.download-link {
  font-weight: 700;
  color: #0d6e5f;
}
</style>
