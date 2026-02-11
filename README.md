# VaultDrop (Inspired-by Secure Secret Sharing)

VaultDrop is a from-scratch Go + Vue secure secret-sharing app inspired by Cryptgeon concepts but implemented with a different architecture and stronger operational defaults.

## Core Properties
- Client-side encryption for secret content
- Supports both text and single-file secret payloads
- One-time retrieval semantics (`consume` deletes atomically)
- TTL-based expiry in Redis
- Password-derived key flow now defaults to Argon2id
- Minimal server-side metadata and strict security headers
- IP-based token-bucket rate limiting on secret endpoints
- Structured request logs (request id, route template, status, latency) without secret payloads
- Optional password-only mode (disables fragment-key links)

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
- `GET /api/metrics`
- `POST /api/v1/secrets`
- `GET /api/v1/secrets/{id}`
- `POST /api/v1/secrets/{id}/consume`

## Security Notes
- The secret key is either in URL fragment (`#...`) or derived from user password.
- URL fragments are not sent to the server by browsers.
- Backend never receives plaintext secret in default flow.
- Frontend currently enforces a 4 MiB max file upload for secure performance bounds.
- Backend enforces strict `meta` schema + payload size caps for `meta` and `ciphertext`.
- Backend enforces file metadata policy (safe filename, max raw file bytes, MIME allowlist).
- In password mode, metadata includes KDF parameters and salt; plaintext key is never sent.
- Legacy PBKDF2 links remain decryptable; new password-protected secrets use Argon2id.
- Startup performs Redis ephemeral-mode self-check (`appendonly=no`, `save=""`) when strict mode is enabled.

Read `docs/threat-model.md`, `docs/security-checklist.md`, and `docs/production-hardening.md` before production deployment.
