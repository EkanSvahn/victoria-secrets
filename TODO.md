# TODO

## Current Priorities
- [ ] Add backend integration tests for critical secure flows:
  - [ ] create with default one-time behavior (`views=1`) and verify second consume is `404`
  - [ ] file secret create request rejects when file metadata exceeds size limit
  - [ ] file secret create request rejects when MIME type is not allowed
  - [ ] password-only mode rejects create request without KDF metadata
- [ ] Frontend UX polish for policy-driven limits and API errors
- [ ] Prepare `v0.1.0` release baseline and deployment runbook

## Security Hardening Backlog
- [x] Argon2id preferred KDF with PBKDF2 compatibility
- [x] Strict CSP/security headers and production reverse-proxy baseline
- [x] Server-side metadata schema validation
- [x] Optional password-only mode (`REQUIRE_PASSWORD`)
- [x] CI vulnerability scanning and dependency update flow
- [x] Metrics endpoint (`/api/metrics`) for create/consume/not_found/rate_limited
- [x] Redis ephemeral self-check for safer runtime defaults

## Release Checklist (`v0.1.0`)
- [ ] All backend and frontend CI checks green on `main`
- [ ] Local manual smoke tests complete (create/open text + file)
- [ ] Security docs reviewed (`docs/security-checklist.md`, `docs/threat-model.md`, `docs/production-hardening.md`)
- [ ] Tag and release notes drafted
