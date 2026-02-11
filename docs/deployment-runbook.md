# Deployment Runbook

This runbook covers a simple production flow for VaultDrop on Ubuntu (physical host or cloud VM) using Docker Compose.

## 1) Prerequisites
- Ubuntu host with Docker Engine and Docker Compose plugin installed.
- DNS for your domain pointed to the host.
- TLS termination via Caddy (`infra/Caddyfile.prod`).
- Redis must be private (not exposed publicly).

## 2) Production Configuration
1. Set your real domain in `infra/Caddyfile.prod`.
2. Copy `.env.example` to `.env` and set production values:
   - `ALLOWED_ORIGINS` to your real HTTPS frontend URL
   - `REQUIRE_PASSWORD=true` for organization/internal policy mode
   - `STRICT_REDIS_EPHEMERAL=true`
   - tune `MAX_FILE_BYTES`, `ALLOWED_FILE_MIME_TYPES`, `MAX_TTL_SECONDS`, `MAX_VIEWS`
3. Keep Redis bound to private network only.

## 3) Deploy (Ubuntu host)
From repo root:

```bash
cd infra
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d --build
```

Health checks:

```bash
curl -fsS https://<your-domain>/api/health
curl -fsS https://<your-domain>/api/status
```

## 4) Deploy (Cloud VM)
Use the same commands as section 3. Ensure:
- inbound firewall allows only `80` and `443`
- SSH access is restricted to trusted IPs
- host updates are applied regularly

## 5) Rollback
If a release causes issues:
1. Identify last known good git tag (example `v0.1.0`).
2. Checkout that tag on the host.
3. Redeploy containers.

```bash
git fetch --tags
git checkout v0.1.0
cd infra
docker compose -f docker-compose.prod.yml up -d --build
```

Post-rollback validation:

```bash
curl -fsS https://<your-domain>/api/health
curl -fsS https://<your-domain>/api/status
```

## 6) Release Procedure (`vX.Y.Z`)
1. Ensure `main` CI is green.
2. Bump `frontend/package.json` version (for local/dev version parity).
3. Merge release prep PR to `main`.
4. Create GitHub tag and release from `main` (`vX.Y.Z`).
5. Deploy from that tag.
6. Verify version badge and API health/status in production.

Notes:
- CI injects `VITE_APP_VERSION` from git ref. Tag builds show tag version in the UI badge.
- Local builds use `frontend/package.json` version as fallback.
