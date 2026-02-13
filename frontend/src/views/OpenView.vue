<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useRoute } from 'vue-router'
import { consumeSecret, toApiErrorMessage } from '../api/client'
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
const copied = ref(false)

const hasResult = computed(() => (isFile.value ? fileUrl.value !== '' : secret.value !== ''))
const hasFragment = computed(() => window.location.hash.replace('#', '').trim().length > 0)

function revokeFileUrl() {
  if (!fileUrl.value) return
  URL.revokeObjectURL(fileUrl.value)
  fileUrl.value = ''
}

async function copySecret() {
  if (!secret.value) return
  await navigator.clipboard.writeText(secret.value)
  copied.value = true
  window.setTimeout(() => {
    copied.value = false
  }, 1200)
}

async function openSecret() {
  const id = String(route.params.id || '')
  if (!id) {
    error.value = 'Invalid secret id.'
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
    error.value = toApiErrorMessage(err, 'Failed to open secret.')
    if (!hasFragment.value && password.value.trim() === '') {
      error.value += ' Add the full link fragment or enter password.'
    }
  } finally {
    loading.value = false
  }
}

onBeforeUnmount(() => {
  revokeFileUrl()
})
</script>

<template>
  <section v-if="!hasResult" class="card open-card">
    <h1>Open Secret</h1>

    <label v-if="!hasFragment">
      Password
      <input v-model="password" type="password" placeholder="Password required for this link" />
    </label>

    <button :disabled="loading || consumed" @click="openSecret">{{ loading ? 'opening...' : 'show secret' }}</button>
    <p v-if="error" class="error">{{ error }}</p>
  </section>

  <section v-else class="card open-card">
    <h1>Secret</h1>

    <template v-if="!isFile">
      <textarea :value="secret" rows="10" readonly></textarea>
      <button type="button" class="copy" @click="copySecret">{{ copied ? 'copied' : 'copy secret' }}</button>
    </template>

    <template v-else>
      <a class="download-link" :href="fileUrl" :download="fileName">download {{ fileName }}</a>
    </template>
  </section>
</template>

<style scoped>
.open-card {
  padding: 1.25rem;
}

.copy {
  width: fit-content;
}

.download-link {
  color: var(--accent);
  font-family: var(--mono);
}
</style>
