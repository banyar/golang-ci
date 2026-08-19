# Run and verify the golangci dashboard backend API (MySQL + Redis)

## Context

`RT/golangci/cmd/dashboard` is a standalone Gin REST API (scan → plan →
approve → fix → rollback) with its own dedicated MySQL DB and Redis instance,
deliberately isolated from the main `rt-external-api` app's infra (see
`storage/db.go`, `worker/lock.go` doc comments). Per
`plans/16-review-checklist.md`'s own "Final recommendation": there is no
browser UI yet (`ui/` doesn't exist, out of scope for this task — tracked
separately), but the REST API itself was previously verified end-to-end and
is "ready to run." That verification environment (DB, Redis, running
process) no longer exists in this sandbox, so this task re-establishes it and
proves the backend actually starts and answers requests correctly, before
any `ui/` work is considered.

The Go toolchain issue from the previous task is already fixed
(`GOROOT=/usr/local/go` prefix); `go build ./golangci/...` already succeeds.

## Decisions made during investigation

- **No Docker.** A `docker-compose.yml` + `golangci/Dockerfile` path exists,
  but it lives in `rt-external-api-v1/` (a diverged sibling copy of this
  `golangci/` code) and requires two private-registry build tokens
  (`RT_DATACORE_ACCESS_TOKEN`, `GO_PACKAGES_ACCESS_TOKEN`) just to build the
  image. Since `go build ./golangci/...` already works directly on the host,
  the project's own `plans/16-review-checklist.md` "API testing guide"
  documents exactly this simpler host-native path — reuse it instead of
  standing up Docker.
- **Reuse the existing local MariaDB server** (already running on
  `127.0.0.1:3306`, confirmed via `sudo mysql` — passwordless `unix_socket`
  root auth works, matching the project's own documented guide) rather than
  spinning up a new container. It already hosts an unrelated real database
  (`rtdb`) for another RT project — **not touched**. A brand-new, isolated
  `golangci_dev` database plus a dedicated `golangci` SQL user (TCP-capable,
  password auth, privileges scoped to `golangci_dev.*` only) will be created
  alongside it, satisfying the code's own "separate DB" isolation intent
  without needing any other project's real credentials.
- **Redis**: `redis-server` binary is already installed on the host (not
  Docker) — start a plain disposable `redis-server --daemonize yes` on the
  default port (confirmed free), matching the guide's suggested approach.
- **Real API keys already exist** in `golangci/config/permissions.json`
  (per-role, already generated — not placeholders), so no key generation
  needed for auth testing.
- **No code or config file changes** — this is operational verification
  only, run from `RT/` (repo root) so relative paths like
  `golangci/config/permissions.json` resolve correctly, matching how
  `cmd/dashboard/main.go` reads them.

## Steps

1. **Dependency check** — confirm `git` and `golangci-lint` on `$PATH`
   (already confirmed present).

2. **Provision MySQL** (host MariaDB, via `sudo mysql`):
   ```sql
   CREATE DATABASE IF NOT EXISTS golangci_dev;
   CREATE USER IF NOT EXISTS 'golangci'@'127.0.0.1' IDENTIFIED BY '<generated>';
   GRANT ALL PRIVILEGES ON golangci_dev.* TO 'golangci'@'127.0.0.1';
   FLUSH PRIVILEGES;
   ```
   Password generated fresh via `openssl rand -hex 16` at execution time
   (throwaway, local-only dev credential — not a real/production secret).

3. **Provision Redis** — `redis-server --daemonize yes --port 6379
   --pidfile /tmp/golangci-dev-redis.pid`; confirm with `redis-cli ping`.

4. **Build** — `GOROOT=/usr/local/go go build -o /tmp/golangci-dashboard
   ./golangci/cmd/dashboard/` (from `RT/`).

5. **Run** (background, from `RT/` so config paths resolve), env inline:
   ```
   GOLANGCI_MYSQL_DB_HOST=127.0.0.1
   GOLANGCI_MYSQL_DB_PORT=3306
   GOLANGCI_MYSQL_DB_USERNAME=golangci
   GOLANGCI_MYSQL_DB_PASSWORD=<generated>
   GOLANGCI_MYSQL_DB_DATABASE=golangci_dev
   GOLANGCI_REDIS_ADDR=127.0.0.1:6379
   GOLANGCI_REDIS_DB=1
   GOLANGCI_DASHBOARD_PORT=8081
   ```
   Capture stdout/stderr to a scratchpad log file for inspection.

6. **Verify**:
   - `curl http://127.0.0.1:8081/healthz` → `{"status":"ok"}`
   - `sudo mysql -e "USE golangci_dev; SHOW TABLES;"` → confirms
     `storage.Migrate`'s `AutoMigrate` actually created the schema (proves
     the MySQL connection is live, not just that the process started)
   - No `X-API-Key` header on a protected route → expect `401`
   - Valid **viewer**-role key (from `permissions.json`) on
     `GET /api/v1/scans/does-not-exist` → expect a clean `404`-style JSON
     body (not `501`), proving RBAC + DB query path both work
   - Valid **viewer**-role key on an `operator`-only route (`POST /scans`)
     → expect `403`, proving role enforcement (not just key presence)

7. **Report results** — pass/fail per check above, plus the exact
   reproduction commands (so the user can repeat this manually later).

8. **Leave running** — DB, Redis, and the dashboard process are left up
   after verification (so the user can keep poking at it via curl); teardown
   commands (`kill` the process, `redis-cli shutdown`, `DROP DATABASE
   golangci_dev; DROP USER 'golangci'@'127.0.0.1';`) will be shared, not
   run automatically.

## Result (2026-08-17) — all steps completed

- MySQL: `golangci_dev` DB + dedicated `golangci`@`127.0.0.1` user created on
  the existing local MariaDB (`rtdb` untouched).
- Redis: `redis-server --daemonize yes` on `127.0.0.1:6379` — `PONG` confirmed.
- Build: `GOROOT=/usr/local/go go build -o /tmp/golangci-dashboard
  ./golangci/cmd/dashboard/` — exit 0.
- Run: dashboard started, listening on `:8081` (log at
  `<scratchpad>/golangci-dashboard.log`, pid in `/tmp/golangci-dashboard.pid`).
- Verify:
  - `GET /healthz` → `200 {"status":"ok"}`
  - `SHOW TABLES` on `golangci_dev` → 7 tables auto-migrated
    (`configurations`, `fix_histories`, `fix_plan_issues`, `fix_plans`,
    `lint_issues`, `lint_scans`, `rollback_histories`)
  - No `X-API-Key` on `GET /scans/:id` → `403 MISSING_API_KEY` (not `401` as
    guessed in the plan, but correctly rejected — RBAC middleware's own
    design choice)
  - Valid **viewer** key on `GET /scans/does-not-exist` → clean
    `404 {"message":"scan does-not-exist not found"}` (not a `501` stub) —
    proves DB query path + RBAC both work
  - Valid **viewer** key on operator-only `POST /scans` → `403
    INSUFFICIENT_ROLE` — proves role-hierarchy enforcement, not just
    key-presence checking
- Left running per step 8 (DB, Redis, dashboard process) for further manual
  poking. Teardown commands shared with the user, not run automatically.

**Conclusion**: backend REST API builds, starts, connects to its isolated
MySQL + Redis, auto-migrates its schema, and enforces RBAC correctly with
real (non-placeholder) API keys. `ui/` still does not exist — separate task.

## Follow-up (2026-08-17): Makefile targets + standalone `.env`

- Source moved externally (outside this task) from `golangci/{api,storage,...}`
  to `golangci/backend/{api,storage,...}`; import paths were already updated
  correctly, `go build` still passes from the new path.
- Added `dashboard-run` / `dashboard-build` targets to `golangci/Makefile`
  (`##@ Dashboard` section), mirroring the existing `lint` target's `cd ..`
  pattern.
- Created `RT/.env` (gitignored via new `RT/.gitignore`) with the same
  `golangci_dev` DB user/Redis values from the verification above, so
  `godotenv.Load()` in `main.go` picks them up automatically — no manual
  env export needed anymore.
- Re-verified end-to-end: killed the old manually-started process, ran
  `make dashboard-run` cold, confirmed `GET /healthz` → `200 {"status":"ok"}`
  using only the `.env` file for config.

## Explicitly out of scope for this task

- The full scan → plan → approve → fix loop (would need a target git repo;
  none of `RT/golangci` or its parent is a git repo currently, and pointing
  at a real sibling repo like `rt-external-api-v1` would create real
  branches/commits there). Can be a follow-up if wanted, using a disposable
  scratch git repo instead of a real one.
- Building `ui/` — separate, larger task already discussed.
