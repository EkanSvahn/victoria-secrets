<script setup lang="ts">
const appVersion = __APP_VERSION__.startsWith('v') ? __APP_VERSION__ : `v${__APP_VERSION__}`
</script>

<template>
  <section class="card info-card">
    <h1>about</h1>
    <p>
      Ephemeral is a secure, open-source secret sharing app inspired by Cryptgeon, implemented with a different stack and
      hardened operational defaults.
    </p>

    <h2>how it works</h2>
    <p>
      Each secret uses a generated id for retrieval and a client-side encryption key for AES-GCM encryption in the browser.
      The server receives ciphertext and metadata only, and one-time consumption removes the secret after access.
    </p>

    <h2>security model</h2>
    <p>
      Client-side encryption, optional password-derived keys (Argon2id), one-time view defaults, strict input validation,
      and rate limiting are enabled to reduce exposure and abuse risk.
    </p>

    <h2>runtime and storage</h2>
    <p>
      Secrets are stored in Redis with expiry/view constraints. For stricter deployments, ephemeral mode checks can enforce
      non-persistent Redis configuration.
    </p>

    <h2>tech stack</h2>
    <p>Backend: Go + Redis. Frontend: Vue + TypeScript + Web Crypto API.</p>

    <h2>open source</h2>
    <p>
      This project is intended to be forked and audited. Use the <code>/code</code> link in navigation to access the source
      repository.
    </p>

    <p class="version">version {{ appVersion }}</p>
  </section>
</template>

<style scoped>
.info-card {
  padding: 1.25rem;
  gap: 0.72rem;
}

.info-card h1,
.info-card h2 {
  text-transform: lowercase;
}

.info-card h2 {
  margin-top: 0.35rem;
  font-size: 0.98rem;
  color: var(--accent);
  font-family: var(--mono);
}

.info-card p {
  font-size: 0.94rem;
  line-height: 1.5;
}

.version {
  margin-top: 0.6rem;
  font-family: var(--mono);
  color: var(--text-muted);
}
</style>
