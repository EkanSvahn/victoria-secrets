<script setup lang="ts">
const appVersion = __APP_VERSION__.startsWith('v') ? __APP_VERSION__ : `v${__APP_VERSION__}`
</script>

<template>
  <section class="card info-card">
    <h1>about</h1>
    <p>
      Ephemeral is a secure, open-source secret sharing app inspired by tools like PrivNote and Cryptgeon, implemented
      with a different stack and a strong focus on clear security defaults.
    </p>

    <h2>how it works</h2>
    <p>
      Each secret uses a generated id for retrieval and a client-side encryption key for AES-GCM encryption in the browser.
      The server receives ciphertext and metadata only, and one-time consumption removes the secret after access.
    </p>

    <h2>security model</h2>
    <p>
      The security model focuses on client-side encryption, optional password-derived keys (Argon2id), one-time view
      defaults, strict input validation, rate limiting, and defensive HTTP headers. The goal is to reduce exposure and
      abuse risk without adding unnecessary complexity.
    </p>

    <h2>runtime and storage</h2>
    <p>
      Secrets are stored in Redis with expiry/view constraints. For stricter deployments, Redis self-checks can enforce
      non-persistent settings so data is not written to disk.
    </p>

    <h2>what this does not protect against</h2>
    <p>
      This does not protect against a compromised client device or browser, weak or reused passwords, secrets copied into
      insecure systems after decryption, or links/passwords shared through unsafe channels. Security still depends on user
      handling and endpoint trust.
    </p>

    <h2>tech stack</h2>
    <p>Backend: Go + Redis. Frontend: Vue + TypeScript + Web Crypto API.</p>

    <h2>open source</h2>
    <p>
      This project is intended to be audited, self-hosted, and forked. Use the <code>/code</code> link in navigation to
      access the source repository.
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
