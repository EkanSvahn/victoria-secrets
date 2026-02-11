# Threat Model (MVP)

## Assets
- Secret plaintext (highest sensitivity)
- Secret encryption key / password
- Secret metadata (lower sensitivity but still confidential)
- Service availability

## Trust Boundaries
- Browser performs encryption/decryption.
- Backend stores only encrypted payload + metadata.
- Redis is private network only and never publicly exposed.

## Adversaries
- Passive network observer
- Internet attacker probing public endpoints
- Misconfiguration causing data leakage
- Insider with server/database access

## Security Goals
- Server must not be able to decrypt content by default flow.
- One-time retrieval is atomic and race-safe across instances.
- Secrets expire automatically and are deleted.
- Abuse is constrained with request limits and payload caps.

## Non-Goals (MVP)
- Multi-tenant RBAC
- Full legal/compliance controls
- Guaranteed perfect forward secrecy across user mistakes

## Key Controls
- AES-GCM encryption in client
- URL fragment key handling when no custom password is used
- Optional password with Argon2id-derived key (legacy PBKDF2 compatibility for old links)
- Optional password-only deployment mode to disable fragment key links
- Redis Lua consume script to GET+DEL atomically
- Strict HTTP security headers
- No sensitive payload logging
