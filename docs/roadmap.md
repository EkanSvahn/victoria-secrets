# Production Readiness Roadmap

This document briefs the next agent (or human) picking up production
hardening of Ephemeral. Phase 1 is merged and live; Phases 2–4 are
designed but not implemented. Phase 5 is parked for later.

> **For the next agent:** read this top-to-bottom before doing any work.
> The "How Erik works" section is non-negotiable and reflects feedback
> from earlier sessions that won't survive memory transfer.

---

## 1. App context

**Ephemeral** is a self-hosted, zero-knowledge secret-sharing app:

- Backend: Go 1.25.9, `net/http`, Redis (ephemeral-mode, no persistence).
- Frontend: Vue 3 + Vite + TypeScript, Web Crypto API (AES-GCM-256),
  Argon2id KDF for password-derived keys, PBKDF2 legacy decrypt path.
- Infra: Docker Compose with a shared edge Caddy that terminates TLS
  for multiple apps. Static Vue files are served by an internal nginx
  container behind the edge Caddy.
- Architecture: hexagonal (domain → ports → adapters) in backend.
  Crypto runs in browser; server stores ciphertext only.

Read these in the repo if you need a deeper picture before working:
- `README.md` — features, API, security posture, limitations
- `docs/threat-model.md` — assets, adversaries, goals, controls
- `docs/production-hardening.md` — env policy, header baseline
- `docs/security-checklist.md` — pre-flight + nice-to-haves
- `docs/deployment-runbook.md` — how prod is deployed
- `backend/internal/adapters/http/handler.go` — main HTTP wiring
- `frontend/src/api/crypto.ts` — encrypt/decrypt entry points

---

## 2. Production environment (as of phase 1 merge)

- Host: GCP Compute Engine VM (`ephemeral-prod`), Ubuntu, Docker Engine
  + Compose plugin.
- Public domain: `secrets.eriksvahn.dev` (TLS via ACME).
- The same Ubuntu host **also runs `app.heima.family`** (a different app
  in a separate repo). Both share **one edge Caddy container**
  (`infra-proxy-1`, image `caddy:2.11-alpine`) which owns ports 80/443.
- The edge Caddy's actual `Caddyfile` on the server is **a divergent
  copy** of `infra/Caddyfile.prod` in this repo — Erik manually added
  the `app.heima.family` site block. A backup exists on the server as
  `Caddyfile.prod.backup-YYYYMMDD`. **Do not assume the repo's
  `Caddyfile.prod` matches production.**
- Internal frontend container (`infra-web-1`) is `nginx:1.27-alpine`,
  serves the built Vue bundle on port 80 inside the Docker network.
- Backend container is the Go binary on `gcr.io/distroless/static-debian12:nonroot`.
- Redis is private to the compose network, ephemeral-mode enforced
  via startup self-check (`appendonly=no`, `save=""`).

---

## 3. How Erik works (non-negotiable)

- **Language:** Swedish in chat, English in code/commits/docs.
- **Git:** Erik runs all git commands himself. Never run `git commit`,
  `git push`, `git checkout`, etc. Provide him **branch name,
  commit message, PR title + body, and a short release note** for
  every phase. He copies and runs.
- **Style of help:** terse and concrete. State what changed and what's
  next. Skip narration. Use file_path:line_number when referencing code.
- **Verification before claiming done:** when you change Go code, run
  `go test ./...` and `go vet ./...` (Go is at `/usr/local/go/bin/go`
  on the WSL host; new Linux instance may have it on PATH). When you
  change frontend code, run `cd frontend && npm test && npm run build`.
  When you change Docker, run `docker build` and ideally a Trivy scan.
  Never claim a phase is done if you couldn't verify locally — call it
  out explicitly.
- **Risky actions:** confirm before destructive git ops, before
  pushing, before changing prod config. Erik's prod is internet-exposed
  and shared with another app.
- **Don't add features beyond scope.** Each phase has a defined set
  of changes. If you find new issues, list them — don't silently fold
  them in.
- **Production has real users (Erik's friends/colleagues).** This is
  not a toy. Bugs leak secrets. Be careful.

---

## 4. Phase 1 — DONE

PR: `feat/phase-1-production-readiness` (already merged to `main`).
Released as the production-readiness baseline.

What was shipped:
- 12-test Vitest suite for `frontend/src/api/crypto.ts` (round-trip
  text/file with and without password, PBKDF2 legacy decrypt,
  AES-GCM tamper rejection, base64url padding edge cases, salt/IV
  uniqueness).
- `GET /api/ready` endpoint (pings Redis with 1s timeout); `/api/health`
  remains pure liveness. Load balancers should target `/api/ready`.
- Server `WriteTimeout` now scales with `RequestTimeout` (was hardcoded
  10s, was clipping large file responses).
- `frontend/Dockerfile`: `npm install` → `npm ci`. Switched base image
  from `caddy:2.8-alpine` to `nginx:1.27-alpine` with `apk upgrade
  --no-cache` at build time. Replaced `Caddyfile` with `nginx.conf`
  (same 5 security headers, SPA fallback). Trivy: 0 HIGH/CRITICAL.
- `infra/docker-compose.prod.yml`: edge Caddy bumped from 2.8 → 2.11
  (Erik manually applied this on the prod server too — see §2).
- `.env.example`: `REQUEST_TIMEOUT_MS=60000` (was 5000, mismatched prod).
- `.dockerignore` for both backend and frontend contexts.
- CI: frontend `npm test` step + new `container-scan` job using Trivy
  against built images, blocks HIGH/CRITICAL.
- Docs updated: README API list, deployment-runbook health/ready,
  security-checklist container scanning marked done.

---

## 5. Outstanding small items (do these alongside Phase 2)

These came up during Phase 1 but were out of scope. Pick them off
opportunistically — they're not phase-blocking.

### 5.1 Pin Trivy action

In `.github/workflows/ci.yml` the `aquasecurity/trivy-action` is
currently on `@master`. Pin to a tagged version (e.g.
`aquasecurity/trivy-action@0.28.0`) or a SHA. Keeps CI deterministic.

### 5.2 Caddyfile divergence

`infra/Caddyfile.prod` in this repo is single-tenant (`vaultdrop.example.com`)
but production has a multi-tenant version with both
`secrets.eriksvahn.dev` and `app.heima.family`. Two reasonable fixes
(Erik to choose):
- Rename to `infra/Caddyfile.prod.example`, gitignore `Caddyfile.prod`,
  document in deployment-runbook that the real one lives on the server.
- Or use Caddy's `import` directive to allow a `Caddyfile.local`
  override that the server pulls from a separate location.

This is one piece of the larger "platform vs apps" separation — see §9.

**Note (2026-05-04):** the analogous problem with
`infra/docker-compose.prod.yml` was solved in PR `fix/externalize-prod-env`:
the compose file uses `${VAR:-default}` interpolation, real values
live in `infra/.env` on the server (gitignored), `infra/.env.example`
in the repo documents the schema. The Caddyfile case is harder because
the divergence is structural (different site blocks) rather than
just values — keep the two fixes above as the options.

### 5.3 Build warnings

`docker build` emits two `SecretsUsedInArgOrEnv` warnings for
`VITE_REQUIRE_PASSWORD`. False positives — it's a boolean policy
flag, not a secret. Either suppress with `# hadolint ignore=` or
accept as noise. Not urgent.

---

## 6. Phase 2 — Test and robustness coverage

Goal: close the test gaps that block confidence in production behavior
under stress and edge conditions.

### 6.1 Backend tests still missing

- **X-Forwarded-For spoofing** when `TRUSTED_PROXY_CIDR` is empty,
  invalid, or the request comes from outside the trusted range. The
  current logic in `backend/internal/adapters/http/clientip.go` is
  defensive but untested. Add tests that verify spoofed XFF headers
  do **not** override `RemoteAddr` when no trust is configured.
- **Per-IP rate-limit isolation.** Verify client A's burst exhaustion
  does not affect client B. Currently `backend/internal/adapters/http/ratelimit.go`
  uses per-IP+route bucket keys, but no test enforces isolation.
- **CORS preflight** with allowed and disallowed origins. Existing
  CORS code in `middleware.go` has logic for `Vary: Origin` and
  preflight responses; add table-driven tests.
- **Defensive path in repository.go:77** (`unexpected payload type`
  from Lua script return). Hard to trigger with real Redis; can be
  exercised with a fake repo that returns a non-string from `Consume`.

### 6.2 E2E tests against real Redis

Currently `integration_test.go` uses an in-process fake repo. Add a
parallel suite that runs against real Redis to validate the Lua
consume script atomicity under contention. Two options:
- `github.com/testcontainers/testcontainers-go` (heavyweight, real
  Redis container).
- `github.com/alicebob/miniredis/v2` (in-process, supports Lua but
  not 100% identical).

Recommend **testcontainers** for the atomicity test specifically (race
between concurrent `consume` calls) and miniredis for unit-level
coverage. Mark testcontainer tests with `//go:build integration` so
they don't slow the default `go test ./...`.

### 6.3 Strengthen `RequirePassword` mode on backend

In `RequirePassword=true` deployments, currently the backend accepts
PBKDF2-encrypted secrets too. For new secrets, force Argon2id; keep
the legacy PBKDF2 decrypt path so existing share links don't break.
Implementation: in `validateKDF` (handler.go ~line 246), if
`limits.RequirePassword` is true and `meta.KDF == "PBKDF2-SHA256"`,
reject with a clear error. Add Argon2id minimum bumps:
- `tt >= 3` (already enforced)
- `tm >= 65536` (already enforced as floor 8192 — tighten in
  RequirePassword mode to 65536)

Add tests for both accept (new ARGON2ID secret) and reject (PBKDF2
secret in RequirePassword mode).

### 6.4 Prometheus format on `/api/metrics`

Current implementation in `backend/internal/metrics/counters.go`
returns JSON. Convert to Prometheus text format
(`# HELP`, `# TYPE`, `metric_name value`) so it's directly scrapable
by Prometheus / Grafana Agent. Maintain JSON output as a fallback if
`Accept: application/json` is requested, or migrate fully — Erik to
choose. Recommend full migration since `METRICS_ENABLED=false` by
default in prod anyway and the current consumers are zero.

### 6.5 Argon2id parameter validation symmetry

Frontend uses fixed Argon2id params (t=3, m=65536, p=1). Backend
allows ranges (m: 8192–262144, t: 1–10, p: 1–8). For `RequirePassword`
deployments, narrow the backend ranges to match what frontend produces,
so an attacker scripting the API can't downgrade to weaker params.

### 6.6 Phase 2 deliverables

For Erik:
- **Branch:** `feat/phase-2-test-and-robustness`
- **Commit:** one logical commit per area is fine (six small commits
  better than one huge one for review). Or one commit if Erik prefers.
- **PR title:** "Phase 2: test coverage and security path hardening"
- **PR body:** summary of what's covered, breaking changes (none
  expected), test plan checklist.
- **Release note:** focused on user-visible/operator-visible items
  (Prometheus format, Argon2id-only mode in `REQUIRE_PASSWORD`).

---

## 7. Phase 3 — Operational maturity

Goal: make production operations defensible. Phase 2 makes the app
robust; Phase 3 makes it observable and disclosed.

### 7.1 SECURITY.md + .well-known/security.txt

- Root-level `SECURITY.md` with: supported versions, how to report a
  vulnerability (email to `e.svahn@proton.me` or a dedicated alias),
  PGP fingerprint if Erik has one, scope and out-of-scope (e.g.
  client-side weakness from compromised browser), expected response
  time.
- `frontend/public/.well-known/security.txt` per RFC 9116. Vite copies
  `public/` to the build output, and our nginx config serves from
  `/usr/share/nginx/html/`, so this should just work — but verify
  with `curl https://secrets.eriksvahn.dev/.well-known/security.txt`
  after deploy.

### 7.2 Alerting guide

A new doc `docs/alerting.md` with concrete Prometheus / Grafana Agent
alert rules:
- `4xx`/`5xx` rate spike (unusual 400/500 over baseline)
- `rate_limited` counter spike (potential abuse)
- Redis ping failure (depends on Phase 6.4 Prometheus migration)
- Container restart loop
- TLS certificate expiry (covered by edge Caddy, but external monitor
  recommended)

This doc is a **template**, not a deployed alerting stack — Erik wires
it into whatever monitoring he uses (likely GCP Cloud Monitoring after
Phase 5).

### 7.3 Threat model update

`docs/threat-model.md` needs additional adversaries currently absent:
- **Malicious admin / insider with Redis access.** Already covered by
  E2EE, but make it explicit.
- **Browser extension exfiltration** (e.g. malicious extension reads
  decrypted plaintext from the page). Out-of-scope, but document.
- **Log poisoning** via crafted request paths/headers. Mitigated by
  structured logging with route templates (not raw paths) — document.
- **Supply-chain via container base images.** Mitigated by CI Trivy
  scanning + `apk upgrade` at build — document the control.

### 7.4 Healthz vs readyz separation

Already done in Phase 1 (`/api/health` and `/api/ready`). In Phase 3,
add:
- `/api/health` returns `{"status": "ok", "version": "<git-sha>"}` so
  operators can see what's deployed without trusting the UI badge.
- `/api/ready` already exists; consider adding a `details` field with
  per-dependency status (`{"redis": "ok"}`) for richer probes.

### 7.5 Bot-protection / abuse-prevention

Currently only IP-based rate limiting. For a public deployment, decide:
- **Captcha on `POST /api/v1/secrets`** for unauthenticated users
  (hCaptcha or Cloudflare Turnstile). Adds friction but blocks
  scripted abuse.
- **Proof-of-work** (e.g. mCaptcha) — friction-light alternative.
- **Or: skip and rely on IP rate-limit.** Erik's call. Recommend
  hCaptcha for public, none for internal/known-team deployment. Add
  a config flag `CAPTCHA_PROVIDER=` so it's opt-in.

### 7.6 Phase 3 deliverables

- **Branch:** `feat/phase-3-operational-maturity`
- **PR title:** "Phase 3: security disclosure, alerting, and ops docs"
- **Release note:** mention `SECURITY.md`, `security.txt`, alerting
  template, threat model update.

---

## 8. Phase 4 — Polish and cleanup

Goal: lower the cognitive load and small inconsistencies that grow
into bigger issues.

### 8.1 Argon2id mobile tuning

Current params (m=65536 KiB, t=3) take ~3–4 seconds in pure JS in
Node. On older mobile browsers this can be too slow or run into
memory pressure. Two paths:
- **Adaptive:** measure browser performance once on page load (or
  cached) and scale params down if slow. Risk: weaker security on
  low-end devices.
- **Document the choice:** keep params fixed, document in README that
  Argon2id at these settings is intentional and may take a few
  seconds on mobile.

Recommend **document, don't tune.** Argon2id security depends on the
parameters; weakening them defeats the purpose. The 3-4 second wait
is the price.

### 8.2 Documentation drift

- `frontend/package.json` says `"version": "0.1.4"` but
  `docs/deployment-runbook.md` references `v0.1.0` as the example
  rollback target. Reconcile to current version or remove example
  versioning.
- Cross-check README, deployment-runbook, threat-model all reference
  the correct app name (`Ephemeral`, sometimes called `VaultDrop` in
  comments — pick one).

### 8.3 Small code smells

- `backend/internal/adapters/http/middleware.go:130` — `writeJSON`
  swallows `Encode` error with `_ =`. Probably fine for our flows but
  should at least log if it fails.
- `backend/internal/adapters/http/ratelimit.go:60` — `cleanupExpired`
  fires every 128th call. Under low traffic, dead buckets accumulate
  for hours. Consider a periodic goroutine instead.
- `frontend/src/api/crypto.ts:3` — `bytesToBinary` could use
  `String.fromCharCode.apply` or a more modern approach. Minor.

### 8.4 CONTRIBUTING.md and editor config

- `CONTRIBUTING.md` at root: how to run tests, how to format, branch
  naming convention, commit message style (Conventional Commits with
  scope).
- `.editorconfig` for tab/space consistency across Go (tabs) and
  TypeScript (2 spaces).

### 8.5 Phase 4 deliverables

- **Branch:** `chore/phase-4-polish`
- **PR title:** "Phase 4: documentation cleanup and small code polish"
- **Release note:** brief, mostly internal.

---

## 9. Phase 5 — GCP migration (parked, decide later)

Currently Erik runs everything on a single Ubuntu VM with Docker
Compose. After Phase 4 is merged, the question is: should we go
cloud-native?

### 9.1 The proposal

Replace the current stack with:
- **Cloud Run** for backend and frontend (containers run serverless).
- **Memorystore Redis** (managed Redis instance, private VPC).
- **Secret Manager** for `REDIS_URL`, `ALLOWED_ORIGINS`, etc.
- **Cloud Armor** as WAF / DDoS protection.
- **Cloud Load Balancer** with managed TLS certificates (replaces
  edge Caddy entirely).
- **Cloud Monitoring + Cloud Logging** (replaces the alerting guide
  from Phase 3).

### 9.2 Trade-offs

Pros:
- No OS patching responsibility (current Ubuntu VM needs unattended-upgrades
  + monitoring).
- Auto-scaling — no capacity planning.
- Built-in observability replaces hand-rolled alerting.
- Managed Redis with proper backup/HA story.
- Removes the "shared edge Caddy" coupling between Ephemeral and
  `app.heima.family` automatically.

Cons:
- ~2–3× monthly cost vs current single-VM setup.
- New infrastructure-as-code (Terraform recommended) to write.
- Cold starts on Cloud Run (mitigatable with min-instances=1).

### 9.3 If we do Phase 5

- Phase 5.1: provision GCP resources via Terraform.
- Phase 5.2: migrate backend to Cloud Run, point at Memorystore.
- Phase 5.3: migrate frontend to Cloud Run, swap edge Caddy for Cloud
  Load Balancer with managed TLS.
- Phase 5.4: cut over DNS, decommission Ubuntu VM.

This is a multi-week project. Don't start it without explicit
approval from Erik and a clear rollback plan.

---

## 10. Memory and continuity

If this brief is read by a fresh Claude Code agent on a new instance,
note that file-based memory (`~/.claude/projects/.../memory/`) does
not transfer between hosts. Re-establish memory by:
- Reading this document fully on first session.
- Saving a `user_role.md` memory: Erik Svahn, working on Ephemeral
  for production, runs all git himself, prefers Swedish in chat.
- Saving a `feedback_no_destructive_git.md`: never run destructive
  git ops, Erik runs git.
- Saving a `project_prod_environment.md`: GCP VM, two domains share
  edge Caddy, see `docs/roadmap.md` for full context.

---

*Last updated by the agent that completed Phase 1.*
