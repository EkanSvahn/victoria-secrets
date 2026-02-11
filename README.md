# VaultDrop (Inspired-by Secure Secret Sharing)

VaultDrop is a from-scratch Go + Vue secure secret-sharing app inspired by Cryptgeon concepts but implemented with a different architecture and stronger operational defaults.

## Core Properties
- Client-side encryption for secret content
- Supports both text and single-file secret payloads
- One-time retrieval semantics (`consume` deletes atomically)
- TTL-based expiry in Redis
- Optional password-derived key flow
- Minimal server-side metadata and strict security headers
- IP-based token-bucket rate limiting on secret endpoints

## Stack
- Backend: Go (`net/http`), Redis
- Frontend: Vue 3 + Vite + Web Crypto API
- Infra: Docker Compose (dev/prod templates)

## Quick Start (Local)
1. `cd infra`
2. `docker compose -f docker-compose.dev.yml up --build`
3. Open `http://localhost:5173`

## API (MVP)
- `GET /api/health`
- `POST /api/v1/secrets`
- `GET /api/v1/secrets/{id}`
- `POST /api/v1/secrets/{id}/consume`

## Security Notes
- The secret key is either in URL fragment (`#...`) or derived from user password.
- URL fragments are not sent to the server by browsers.
- Backend never receives plaintext secret in default flow.
- Frontend currently enforces a 4 MiB max file upload for secure performance bounds.

Read `docs/threat-model.md` and `docs/security-checklist.md` before production deployment.
