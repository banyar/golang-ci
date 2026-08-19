# 14 — Risk Analysis

**Date:** 2026-08-03
**Status:** Draft — risk register sourced verbatim; "Open Risks" section is new, aggregating every `[TBD]` flagged across files 01–13
**Source:** `golangci/golangci-lint-dashboard.html` §Risks (register) + files 01–13 of this suite (open items)
**Dependency:** ALL prior files (01–13)

## Risk Register

| Category | Risk | Mitigation |
|---|---|---|
| Architecture *(neutral)* | Scan Runner coupled to API process blocks horizontal scaling | Separate worker pool behind a job queue (see `worker/` in [12-component-design.md](12-component-design.md)) |
| Performance *(neutral)* | Large monorepo scan takes minutes, blocks the UI thread of thought | Async scan + polling/SSE; `--new-from-rev` incremental scans by default |
| Concurrency *(warn)* | Two fixes race on the same file/branch | Redis-based per-repo/branch lock; serialized fix queue (Rule BR-1, [10-business-rules.md](10-business-rules.md)) |
| AI wrong fix *(crit)* | LLM proposes a plausible but behavior-changing patch | Mandatory human approval gate (Rule BR-3); automatic rescan diff; block apply if severity count increases ([13-test-plan.md](13-test-plan.md) scenario 22) |
| AI wrong fix *(crit)* | Non-deterministic AI output across identical inputs | Cache plan by issue+source fingerprint; store model + prompt version on `FixPlan` ([13-test-plan.md](13-test-plan.md) scenario 12) |
| Rollback *(warn)* | Revert conflicts with later commits on the same branch | Rollback only via `git revert`, never reset/force-push; surface manual-resolution state instead of forcing (Rule BR-4) |
| Security *(warn)* | Fix Service granted overly broad repo write access | Credentials scoped to `lint-fix/*` branch prefix via branch protection; never `main` (Rule BR-2) |

Severity tiers as encoded in the source (`neutral` / `warn` / `crit`) — both AI-wrong-fix rows sit at the highest tier (`crit`), the only category to do so.

## Open Risks — resolved with default assumptions (2026-08-04)

Every `[TBD]` marker raised while authoring this suite. Per the requestor's decision, all 11 were closed with a sensible default rather than left blocking — each default is written into its source file, not just checked off here. Anything marked "Assumption/Decided (default)" should be revisited if it turns out wrong once the feature has real usage.

| # | Open item | Resolution | Resolved in |
|---|---|---|---|
| 1 | Exact 21-section PRD structure | Standard scaffold accepted as final for this build cycle | [04-prd.md](04-prd.md) |
| 2 | Success metric, deadline/budget, requestor's formal role | Qualitative success criteria adopted; no deadline/budget; requestor acts as de facto BA/PM | [01-business-requirement.md](01-business-requirement.md) |
| 3 | NFR numeric targets | No hard SLA numbers set for an internal tool at this scale; NFR-1..6 stay qualitative | [04-prd.md §10](04-prd.md) |
| 4 | API versioning scheme | `/api/v1/...` prefix, `/api/v2` on breaking change, no formal deprecation window | [06-api-design.md](06-api-design.md) |
| 5 | Enum value lists | `LintIssue.severity`, `FixPlan.status`, `FixPlan.risk_level`, `FixHistory.result` all given concrete value sets | [08-validation-rules.md](08-validation-rules.md) |
| 6 | Missing 403/502 constructors in `utils.go` | `utils.ForbiddenErr` and `utils.BadGatewayErr` added, mirroring the existing `NotFound`/`ConflictErr` pattern — implemented as part of M1 | [09-error-handling.md](09-error-handling.md) |
| 7 | Async boundary for `POST /plans`/`POST /rollbacks` | Both return `202 Accepted`, consistent with `POST /scans`/`POST /fixes` | [11-sequence-diagram.md](11-sequence-diagram.md) |
| 8 | Cross-scan `issue_ids` batch | Rejected with `422 VALIDATION_FAILED` — one plan batch, one scan | [13-test-plan.md](13-test-plan.md) scenario 9 |
| 9 | `/history/*` pagination | `limit`/`offset`, default 50, max 200 | [13-test-plan.md](13-test-plan.md) scenario 25 |
| 10 | Double rollback | `409 Conflict`, `Code=ALREADY_ROLLED_BACK` | [13-test-plan.md](13-test-plan.md) scenario 29 |
| 11 | Replace vs complement existing `golangci-report.sh` reports | Complement — v1 is additive, existing `make lint*` CLI keeps working unchanged | [13-test-plan.md §Regression Checklist](13-test-plan.md), [02-current-workflow.md](02-current-workflow.md) |

Per `plans-structure-analysis.md` Phase 3.2, this resolution (default assumptions, explicitly flagged, not silent) is what lets [16-review-checklist.md](16-review-checklist.md)'s Pre-Implementation Checklist be honestly checked — the substance is now in each source file, not just a checkbox.
