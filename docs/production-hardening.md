# Production Hardening Baseline

## 1) Reverse Proxy and TLS
- Use `infra/Caddyfile.prod` with your real domain.
- Terminate TLS at the proxy and enable HSTS.
- Route `/api/*` to backend and everything else to frontend.

## 2) Required Environment Policy
- `REQUIRE_PASSWORD=true` for organization/internal deployments.
- `ALLOWED_ORIGINS` set to your real HTTPS origin.
- `STRICT_REDIS_EPHEMERAL=true` to fail startup if Redis persistence is enabled.
- Set `MAX_FILE_BYTES` and `ALLOWED_FILE_MIME_TYPES` to your policy limits.
- Keep Redis private (no public ports exposed).

## 3) Header Baseline
- `Strict-Transport-Security`
- `Content-Security-Policy`
- `X-Content-Type-Options`
- `X-Frame-Options`
- `Referrer-Policy`
- `Permissions-Policy`

## 4) Operational Baseline
- Enable CI dependency scanning (npm + Go vulnerability checks).
- Keep dependency updates active via Dependabot.
- Review logs only for metadata: request id, route template, status, duration.
- Never log secret payloads, decryption keys, or URL fragments.
