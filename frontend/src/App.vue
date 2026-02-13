<script setup lang="ts">
import { onMounted, ref } from 'vue'

const appVersion = __APP_VERSION__.startsWith('v') ? __APP_VERSION__ : `v${__APP_VERSION__}`
const theme = ref<'light' | 'dark'>('dark')
const logoSrc = ref('/logo-light.svg')

onMounted(() => {
  const stored = localStorage.getItem('vaultdrop-theme')
  if (stored === 'light' || stored === 'dark') {
    theme.value = stored
  } else {
    theme.value = 'dark'
  }
  applyTheme(theme.value)
})

function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
  applyTheme(theme.value)
  localStorage.setItem('vaultdrop-theme', theme.value)
}

function applyTheme(nextTheme: 'light' | 'dark') {
  document.documentElement.setAttribute('data-theme', nextTheme)
  logoSrc.value = nextTheme === 'dark' ? '/logo-light.svg' : '/logo-dark.svg'
}
</script>

<template>
  <main class="page">
    <img class="brand-logo" :src="logoSrc" alt="VaultDrop logo" />
    <router-view />
    <button class="theme-fab" type="button" @click="toggleTheme">{{ theme === 'dark' ? 'light' : 'dark' }}</button>
    <div class="version-badge" aria-label="app-version">{{ appVersion }}</div>
  </main>
</template>

<style scoped>
.brand-logo {
  width: min(360px, 72vw);
  height: auto;
  margin-bottom: 1.4rem;
  opacity: 0.96;
}

.theme-fab {
  position: fixed;
  right: 0.72rem;
  bottom: 0.72rem;
  border: none;
  background: var(--badge-bg);
  color: var(--text-muted);
  border-radius: 0;
  padding: 0.22rem 0.5rem;
  font-size: 0.78rem;
  z-index: 10;
}

.theme-fab:hover {
  color: var(--accent);
}

.version-badge {
  position: fixed;
  left: 0.75rem;
  bottom: 0.75rem;
  padding: 0.25rem 0.52rem;
  border-radius: 0;
  background: var(--badge-bg);
  color: var(--badge-text);
  border: none;
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  pointer-events: none;
}
</style>
