# Ephemeral

Ephemeral is a secure secret-sharing application for one-time or time-limited sharing of notes and files.

It is designed for self-hosted use, with client-side encryption in the browser and server-side storage of ciphertext only.
The project is inspired by the broader encrypted secret-sharing pattern (including tools like PrivNote and Cryptgeon), but
focuses on a Go + Vue stack and clear operational hardening defaults.

## Core Properties
- Client-side encryption for secret content
- Supports both text and single-file secret payloads
- One-time retrieval semantics (`consume` deletes atomically)
- Views-based expiry by default (`views=1`), or TTL-based expiry
- Password-derived key flow now defaults to Argon2id
- Minimal server-side metadata and strict security headers
- IP-based token-bucket rate limiting on secret endpoints
- Structured request logs (request id, route template, status, latency) without secret payloads
- Optional password-only mode (disables fragment-key links)

## What Ephemeral Does
- Encrypts secrets in the browser before upload
- Stores encrypted payloads in Redis with expiry / view limits
- Generates share links for one-time or time-limited retrieval
- Supports optional password-based decryption flow
- Provides policy and status endpoints for deployment visibility

## Security Posture
Ephemeral is intended to be a highly secure way to share secrets when it is configured and operated correctly.

Security comes from the combination of:
- Client-side encryption (server stores ciphertext, not plaintext)
- One-time / limited-view and TTL-based destruction
- Password-derived keys (Argon2id for new password-protected secrets)
- Strict server-side validation and payload limits
- Rate limiting and security headers
- TLS termination and reverse-proxy hardening
- Optional Redis ephemeral-mode enforcement checks

This reduces exposure significantly, but does not eliminate all risk.

## Stack
- Backend: Go (`net/http`), Redis
- Frontend: Vue 3 + Vite + Web Crypto API
- Infra: Docker Compose (dev/prod templates)

## Quick Start (Local)
1. `cd infra`
2. `docker compose -f docker-compose.dev.yml up --build`
3. Open `http://localhost:5173`

## Production Baseline
1. Set domain in `infra/Caddyfile.prod` (replace `vaultdrop.example.com`).
2. Use `docker compose -f infra/docker-compose.prod.yml up --build -d`.
3. Proxy terminates TLS, applies HSTS/security headers, and routes:
   - `/api/*` -> backend
   - all other paths -> frontend

## API (MVP)
- `GET /api/health`
- `GET /api/status`
- `GET /api/metrics`
- `POST /api/v1/secrets`
- `GET /api/v1/secrets/{id}`
- `POST /api/v1/secrets/{id}/consume`

`POST /api/v1/secrets` accepts either:
- `views` (default behavior is one-time view if omitted)
- `ttl_seconds` (time-based expiry)

## Security Notes
- The secret key is either in URL fragment (`#...`) or derived from user password.
- URL fragments are not sent to the server by browsers.
- Backend never receives plaintext secret in default flow.
- The example dev/prod compose configuration sets a 50 MiB file upload limit with matching body/ciphertext/request timeout caps.
- Backend enforces strict `meta` schema + payload size caps for `meta` and `ciphertext`.
- Backend enforces file metadata policy (safe filename, max raw file bytes, MIME allowlist).
- In password mode, metadata includes KDF parameters and salt; plaintext key is never sent.
- Legacy PBKDF2 links remain decryptable; new password-protected secrets use Argon2id.
- Startup performs Redis ephemeral-mode self-check (`appendonly=no`, `save=""`) when strict mode is enabled.

## Security Limitations (Important)
Ephemeral does not protect against:
- Compromised sender or recipient devices/browsers
- Weak or reused passwords
- Secrets copied into insecure systems after decryption
- Unsafe sharing of links/passwords through insecure channels
- Deployment misconfiguration (DNS, TLS, headers, logging, policy settings)

Use it as a hardened transport-and-sharing mechanism, not as a substitute for endpoint security or operational hygiene.

## File Upload Sizing Notes
- The current architecture is optimized for secrets and small/medium files (browser encryption + Redis-backed ciphertext storage).
- Larger limits increase memory and timeout pressure in the browser, backend, and Redis.
- Increase `MAX_FILE_BYTES` together with `MAX_BODY_BYTES`, `MAX_CIPHERTEXT_BYTES`, and `REQUEST_TIMEOUT_MS`.
- For very large files (for example large database backups), prefer a separate chunked/object-storage flow and use Ephemeral to share keys/passwords.

Read `docs/threat-model.md`, `docs/security-checklist.md`, `docs/production-hardening.md`, and `docs/deployment-runbook.md` before production deployment.
