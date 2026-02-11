<script setup lang="ts">
const appVersion = __APP_VERSION__.startsWith('v') ? __APP_VERSION__ : `v${__APP_VERSION__}`
</script>

<template>
  <main class="page">
    <router-view />
    <div class="version-badge" aria-label="app-version">{{ appVersion }}</div>
  </main>
</template>

<style scoped>
.version-badge {
  position: fixed;
  left: 0.75rem;
  bottom: 0.75rem;
  padding: 0.28rem 0.55rem;
  border-radius: 999px;
  background: rgba(17, 33, 52, 0.8);
  color: #f4f8ff;
  font-size: 0.76rem;
  font-weight: 700;
  letter-spacing: 0.02em;
  pointer-events: none;
}
</style>
