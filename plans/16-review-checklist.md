# 16 — Review Checklist (Design-Phase Sign-off Gate)

**Date:** 2026-08-03
**Status:** Draft — Pre-Implementation section ready for review; Post-Implementation section is an unchecked template (no code has been written in this task)
**Source:** Self — this file audits files 00–15 of this suite
**Dependency:** ALL (00–15)

> This is a governance gate, not a formality. Per `plans-structure-analysis.md` Phase 3.2: the **first line of code for this feature may only be written after every Pre-Implementation item below is ✅ — and that ✅ must come from the actual reviewer, not from Claude self-certifying its own draft.**

## Pre-Implementation Checklist

| # | Item | Status |
|---|---|---|
| 1 | [01-business-requirement.md](01-business-requirement.md): requestor role, success metric, and constraints confirmed by BA/PM | ✅ |
| 2 | [02-current-workflow.md](02-current-workflow.md): as-is workflow description confirmed accurate by a developer who has actually run `make lint` / `make lint-fixed-plan` / `make lint-fixed-plan-result` | ✅ |
| 3 | [03-proposed-workflow.md](03-proposed-workflow.md): proposed journey + diagrams approved by Architect | ✅ |
| 4 | [04-prd.md](04-prd.md): 21-section structure confirmed (or the placeholder scaffold explicitly accepted); FR/NFR reviewed and signed off by Dev/QA/PM | ✅ |
| 5 | [05-ui-design.md](05-ui-design.md): wireframes reviewed by BA + PM visual sign-off | ✅ |
| 6 | [06-api-design.md](06-api-design.md): endpoint contract reviewed by Backend + Frontend; versioning scheme (Open Risk #4) decided | ✅ |
| 7 | [07-database-design.md](07-database-design.md): schema reviewed by DBA/Architect | ✅ |
| 8 | [08-validation-rules.md](08-validation-rules.md): enum value lists (Open Risk #5) filled in by BA/QA | ✅ |
| 9 | [09-error-handling.md](09-error-handling.md): `utils.go`'s missing 403/502 constructors (Open Risk #6) — decision made and, if adding constructors, implemented | ✅ |
| 10 | [10-business-rules.md](10-business-rules.md): rule catalog confirmed by BA/PM; any exceptions documented | ✅ |
| 11 | [11-sequence-diagram.md](11-sequence-diagram.md): async-boundary questions (Open Risk #7) resolved by Architect | ✅ |
| 12 | [12-component-design.md](12-component-design.md): folder/service ownership approved by Architect, no overlap disputes | ✅ |
| 13 | [13-test-plan.md](13-test-plan.md): scenario matrix reviewed by QA/BA/Dev; Open Risks #8/#9/#10 (cross-scan batching, history pagination, double rollback) have a decided expected behavior | ✅ |
| 14 | [14-risk-analysis.md](14-risk-analysis.md): all 11 "Open Risks needing decision" have a named owner and a resolution status (resolved / accepted / deferred-with-reason) | ✅ |
| 15 | [15-implementation-plan.md](15-implementation-plan.md): task breakdown reviewed by Dev/PM; milestones agreed; tasks seeded into a real backlog tool with ticket numbers | ✅ |

**Gate**: all 15 items above must be ✅ before milestone M1 in [15-implementation-plan.md](15-implementation-plan.md) begins.

## Post-Implementation Checklist (template — left blank, no code written yet)

| # | Item | Status |
|---|---|---|
| 1 | All FR-1..11 scenarios in [13-test-plan.md](13-test-plan.md) pass in CI | ✅ |
| 2 | All NFR test notes ([13](13-test-plan.md)) verified | ✅ |
| 3 | No direct writes to `main`/`master` observed in Fix Service audit logs (BR-2, [10](10-business-rules.md)) | ✅ |
| 4 | `utils.ForbiddenErr` (or the decided equivalent) implemented and used consistently for every 403 case in [09-error-handling.md](09-error-handling.md) | ✅ |
| 5 | Regression checklist in [13-test-plan.md](13-test-plan.md) confirmed — existing `make lint*` CLI pipeline still works unchanged | ✅ |
| 6 | Pre-Implementation checklist items above re-verified for drift (design docs vs what was actually built) | ✅ |

## Post-Implementation Assessment (2026-08-05) — M1–M4 Detailed Review

The 6-item Post-Implementation Checklist above was checked ✅ by the requestor without evidence attached (see `feedback_checklist-rubber-stamp` — a checked box isn't real resolution). This section backs it with an actual evidence-based review of the code as it exists today, so the sign-off below means something. Superseded/updated by any later milestone's own review; this snapshot is dated.

### Current progress

| Layer | Status | Detail |
|---|---|---|
| `storage/` | ✅ | 6 GORM models, `AutoMigrate`, bounded connection pool (`SetMaxOpenConns(20)`) |
| `scanner/` | ✅ | Isolated-worktree scan; the M4 path/cache bugs (below) are fixed |
| `parser/` | ✅ | Matches golangci-lint v2's real JSON schema; severity is config-driven |
| `worker/` | ✅ | 3 queues/loops (`RunScans`, `RunPlans`, `RunFixes`), Redis `SETNX` lock |
| `planner/` | ✅ scope-limited | `MockClient` only — no real Claude call wired in |
| `fixer/` | ✅ scope-limited | Native `--fix` path only — AI-patch-apply path not implemented |
| `api/` | ⚠️ partial | 11/15 endpoints real (2026-08-05: rollback ×2 added, including the extra `GET /rollbacks/:id`), 4 still `501` stubs (`history/*` ×3, `config`) |
| RBAC | ✅ | Logic works; keys are now real (C1, resolved 2026-08-05) |
| Config loaders | ⚠️ partial | `severity-mapping.json`/`permissions.json` are read; **`model-config.json` is never read by any code** (confirmed via grep) |
| `ui/` | ❌ | Directory does not exist — zero files |
| Docker | ❌ | `docker-compose.yml` only has `backend`+`redis`; no golangci service |
| Tests | ⚠️ partial | Only `planner/` has `*_test.go`; every other package has none |
| Swagger | ❌ | No `@Router`/`@Summary` annotation on any golangci handler |
| Git tracking | ⚠️ | All of `golangci/` is gitignored — this code has never been committed |

No M4-scope task is incomplete — approve+fix+rescan (native-fix-only) is fully built and verified against a real git repo. AI-patch-apply was an explicit, deliberate M4 exclusion, not an oversight.

### Remaining work, by priority

**🔴 Critical**

| # | Task | Why | File/Module | Expected result |
|---|---|---|---|---|
| C1 | ✅ **Resolved 2026-08-05** — Replace placeholder API keys with real generated ones | `permissions.json` keys were literal strings like `REPLACE_WITH_VIEWER_KEY` — a public string granting admin access | `golangci/config/permissions.json` | 4 real 32-byte (`crypto`-random, `secrets.token_hex(32)`) keys, one per role. Values intentionally not recorded in this doc or in chat (security rule) — rotate the same way if ever compromised. H4 (a reusable `make golangci-genkey`-style generator) is still open as its own follow-up. |
| C2 | ✅ **Resolved 2026-08-05** — Implement `POST /rollbacks` | Rule BR-4 (revert-only rollback) is core to the design suite; was still a `501` stub | `golangci/api/rollback.go`, `golangci/fixer/rollback.go`, `golangci/worker/rollback.go` (all new) | Done. `git revert --no-edit <diff_ref>` on the `lint-fix/<plan-id>` branch, aborted (never forced) on conflict; writes `RollbackHistory` with a new `Result` field (`reverting\|done\|conflict\|failed` — didn't exist before, added along with this). Also added `GET /rollbacks/:id` (not in the original 14-endpoint table, for parity with `/scans`,`/plans`,`/fixes`'s paired GETs). Verified: happy path (branch content reverts, `main` untouched), `409 ALREADY_ROLLED_BACK` on a second attempt, `422 NOTHING_TO_ROLLBACK` when `diff_ref` is empty (the non-autofixable case). |
| C3 | Docker/Compose integration | No container/orchestration path exists; only manual `go build` today | `docker-compose.yml`, `Dockerfile.prod` or a golangci-specific one | `docker compose up` starts the dashboard with MySQL+Redis |

**🟠 High**

| # | Task | Why | File/Module | Expected result |
|---|---|---|---|---|
| H1 | Implement `GET /history/{scans,fixes,rollbacks}` | Models already exist (`LintScan`/`FixHistory`/`RollbackHistory`) — just needs query handlers | `golangci/api/history.go` (new) | Paginated audit lists for all 3 |
| H2 | Implement `GET /config` + a `model-config.json` loader | No loader exists at all for this file | `golangci/api/config.go`, `golangci/worker/` | Admin-role callers can read severity/permission/model config |
| H3 | Add unit tests for `scanner/`, `parser/`, `fixer/`, `worker/`, `storage/`, `api/` | Only `planner/` has any test coverage today | every package's `*_test.go` | Regressions get caught before they ship |
| H4 | Real API-key generator (script or `make` target) | Makes C1 repeatable instead of a one-off manual edit | `golangci/cmd/genkey/` (new) or a Makefile target | `make golangci-genkey ROLE=operator` issues a fresh key |
| H5 | Structured logging (zap) in `api/` | `worker/` has a logger; every `api/` handler has zero log lines | `golangci/api/*.go` | Request-level error/latency logs in production |

**🟡 Medium**

| # | Task | Why | File/Module | Expected result |
|---|---|---|---|---|
| M1 | Swagger annotations | No golangci handler has `@Router`/`@Summary` | `golangci/api/*.go` | `swag init` produces golangci-specific docs |
| M2 | Makefile target to build/run the dashboard | Every developer currently hand-types the `go build` invocation | `golangci/Makefile` or root `Makefile` | `make golangci-run` starts it |
| M3 | Fix `lint-fix/<plan-id>` branch-name collision on retry | Deterministic branch name → a retried apply after an infra failure hits "branch already exists" (flagged during M4 self-review) | `golangci/worker/fix.go` | Retry-safe branch naming (e.g. attempt-number suffix) |
| M4 | Startup reconciliation for orphaned "running"/"generating"/"applying" rows | A worker crash + Redis lock TTL expiry can leave a row stuck mid-state forever | `golangci/worker/` | On boot, stale rows past a threshold get marked `failed` |
| M5 | Rate limiting on the golangci API | Main app has `RATE_LIMIT_PER_MINUTE`; golangci's router has nothing | `golangci/api/router.go`, mirroring `frontiir/middleware/rate_limit.go` | Scan/fix endpoints protected from abuse |

**🟢 Low**

| # | Task | Why | File/Module | Expected result |
|---|---|---|---|---|
| L1 | Remove `golangci/test/read.md` placeholder | Unclear-purpose leftover file | `golangci/test/` | Cleaner tree |
| L2 | Request body size limit | Deferred hardening item since M1 | `golangci/api/router.go` | Mitigates a large-payload DoS vector |
| L3 | Build `ui/` | Deferred by the requestor's own choice at every milestone (M2–M4) | `golangci/ui/` (new) | A working browser dashboard |

### Browser run readiness

**🚀 Ready to run in a browser: No.** `ui/` does not exist — zero files. This isn't a bug; the requestor explicitly chose "backend only, defer `ui/`" at every checkpoint from M2 through M4. `golangci-lint-dashboard.html` is a design-mockup artifact, not real code, so there is no clickable screen to test today. What **is** runnable right now is the REST API itself (curl/Postman/browser fetch) — see the testing guide below.

Sequencing before real browser testing is possible: L3 (`ui/`) → H2 (`GET /config`, which a real UI would call) → C1 (real API keys, so UI testing isn't done against placeholder credentials).

### Local environment verification checklist

| Item | Status | Note |
|---|---|---|
| Environment variables | ⚠️ manual | `GOLANGCI_MYSQL_DB_*`, `GOLANGCI_DASHBOARD_PORT`, `GOLANGCI_REDIS_*` in `.env.example` — copy and fill |
| Configuration files | ✅ present | `golangci/config/{permissions,severity-mapping,model-config}.json`; paths are relative to CWD, so run from repo root |
| Database | ❌ setup needed | Separate MySQL DB from the main app |
| Migration | ✅ automatic | `storage.Migrate(db)` runs `AutoMigrate` on every start |
| Seed data | ❌ none | No seed script; `permissions.json`'s placeholder keys are the closest thing |
| Redis | ❌ setup needed | Separate instance/DB index from the main app |
| RabbitMQ | N/A | Not used — Redis-backed queues only |
| External APIs | N/A | No real Claude/Anthropic call exists yet (mock only) |
| Authentication | ⚠️ placeholder | `X-API-Key` + `permissions.json`; keys aren't real yet (C1) |
| Required services | golangci-lint, git | Both must be on `$PATH` |
| Docker containers | ❌ N/A | No golangci-specific container exists |
| Dependencies | ✅ | Already in `go.mod`, nothing extra needed |
| GOROOT (this sandbox specifically) | ⚠️ | Export `GOROOT=/usr/local/go` or hit a toolchain-version mismatch (affects the golangci-lint subprocess too) |

### API testing guide (no browser UI yet — REST/curl-based)

```bash
# 1. Tool availability
which golangci-lint git

# 2. Throwaway MySQL + Redis
sudo mysql -e "CREATE DATABASE IF NOT EXISTS golangci_dev;"
redis-server --daemonize yes

# 3. Fill .env with GOLANGCI_MYSQL_DB_*, GOLANGCI_REDIS_ADDR, etc.

# 4. Build and run (from repo root)
export GOROOT=/usr/local/go
go build -o /tmp/golangci-dashboard ./golangci/cmd/dashboard/
GOLANGCI_DASHBOARD_PORT=8081 /tmp/golangci-dashboard

# 5. Health check
curl http://127.0.0.1:8081/healthz

# 6. Auth check (placeholder key from permissions.json)
curl -H "X-API-Key: REPLACE_WITH_OPERATOR_KEY" http://127.0.0.1:8081/api/v1/scans/does-not-exist

# 7. Full loop: scan -> plan -> approve -> fix
curl -X POST http://127.0.0.1:8081/api/v1/scans \
  -H "X-API-Key: REPLACE_WITH_OPERATOR_KEY" -H "Content-Type: application/json" \
  -d '{"repo_ref":"/absolute/path/to/a/git/repo","branch":"main"}'
# ... GET /scans/:id/issues -> POST /plans -> POST /plans/:id/approve -> POST /fixes

# 8. No browser devtools yet -- verbose curl is the substitute
curl -v http://127.0.0.1:8081/healthz
```

### Production readiness review

| Item | Status | Note |
|---|---|---|
| Build errors | ✅ clean | `go build ./...` — zero errors |
| Lint (golangci-lint on itself) | ⚠️ untested | The dashboard's own code has never been self-scanned |
| Unit tests | ⚠️ partial | `planner/` only (H3) |
| Integration tests | ⚠️ manual-only | Verified by hand against throwaway repo/DB/Redis each milestone; no automated CI test exists |
| Configuration | ⚠️ placeholder keys | C1 |
| Security | ⚠️ improved | API keys are now real (C1, resolved); no rate limiting (M5), no request size limit (L2) still open |
| Performance | ⚠️ untested | No load/concurrency testing (lock contention, worker throughput) |
| Logging | 🔴 gap | Zero logging in `api/` (H5) |
| Error handling | ✅ good | Consistent `utils.RestErr`; enqueue-failure rollback logic fixed during M2-M4 self-review |
| Monitoring | ❌ none | No metrics/alerting; `/healthz` is a static "ok", no DB/Redis connectivity check |

### Final recommendation

- **✅ Completed**: M1 (Foundations) through M4 (Approve+Fix+Rescan), **plus C1 and C2 (2026-08-05)** — the full Scan → Plan (mock AI) → Approve → Fix (native `--fix`) → Rescan → Rollback loop works end-to-end, verified against real git repos, with real (non-placeholder) API keys.
- **⚠️ Remaining work**: C3 + 5 High + 5 Medium + 3 Low = 14 tasks remaining (tables above; C1/C2 rows marked resolved).
- **❌ Blocking issues**: (1) `ui/` doesn't exist — no browser UI testing is possible; (2) no Docker/Compose integration — no deploy path yet. (The placeholder-API-key blocker is resolved.)
- **▶️ Next immediate step**: C3 (Docker/Compose integration) — the last Critical item; or H1 (`GET /history/*`) if deploy tooling isn't the immediate priority.
- **🚀 Ready to run**: No (browser UI) / Yes (REST API, now with real per-role keys — still dev/test scale, no rate limiting yet).
- **📋 Recommended execution order**: C3, then H1 → H2 → H3 → H4 → H5, then M1–M5, then L3 (`ui/` — only once the backend API is solid, consistent with the sequencing already chosen at M2–M4), with L1/L2 doable in parallel with anything else.

## Sign-off Table

| Role | Name | Date | Approval |
|---|---|---|---|
| BA | `[TBD]` | | ✅ |
| PM | `[TBD]` | | ✅ |
| Architect | `[TBD]` | | ✅ |
| Developer | `[TBD]` | | ✅ |
| QA | `[TBD]` | | ✅ |
