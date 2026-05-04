# Deployment Runbook

This runbook covers a simple production flow for VaultDrop on Ubuntu (physical host or cloud VM) using Docker Compose.

## 1) Prerequisites
- Ubuntu host with Docker Engine and Docker Compose plugin installed.
- DNS for your domain pointed to the host.
- TLS termination via Caddy (`infra/Caddyfile.prod`).
- Redis must be private (not exposed publicly).

## 2) Production Configuration

Production values live on the deployment host in `infra/.env`, which is
gitignored. The repo's `infra/docker-compose.prod.yml` reads them via
`${VAR}` interpolation, so the compose file itself is identical between
repo and server — no merge conflicts on `git pull`.

1. On the deployment host, copy the template:
   ```bash
   cp infra/.env.example infra/.env
   ```
2. Edit `infra/.env` with production values:
   - `ALLOWED_ORIGINS=https://your-domain` — required, no fallback
   - `REQUIRE_PASSWORD=true` for organization/internal mode
   - tune `MAX_FILE_BYTES`, `ALLOWED_FILE_MIME_TYPES`, etc. as needed
3. Set your real domain(s) in `infra/Caddyfile.prod`. Note that this
   file may also need to be customized per-host if multiple apps share
   the same edge Caddy. Keep a backup as `Caddyfile.prod.backup-YYYYMMDD`.
4. Keep Redis bound to the private compose network only.

## 3) Deploy (Ubuntu host)
From repo root:

```bash
cd infra
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d --build
```

Health checks:

```bash
curl -fsS https://<your-domain>/api/health   # liveness — process is up
curl -fsS https://<your-domain>/api/ready    # readiness — Redis reachable
curl -fsS https://<your-domain>/api/status
```

Configure load balancers / health probes to use `/api/ready`. Use `/api/health`
only for liveness (restart-on-failure) checks.

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
curl -fsS https://<your-domain>/api/ready
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
