# 13 — Test Plan

**Date:** 2026-08-03
**Status:** Draft — **new synthesis**, built from [04-prd.md](04-prd.md)'s FR list + [08](08-validation-rules.md)/[09](09-error-handling.md)'s edge cases; no test plan exists in the source artifact
**Source:** [04-prd.md](04-prd.md), [08-validation-rules.md](08-validation-rules.md), [09-error-handling.md](09-error-handling.md), [10-business-rules.md](10-business-rules.md)
**Dependency:** [04-prd.md](04-prd.md) (FR/AC), [08-validation-rules.md](08-validation-rules.md) (edge case), [09-error-handling.md](09-error-handling.md) (error case)

## Test Scenario Matrix (Given/When/Then)

| # | FR | Scenario | Given | When | Then |
|---|---|---|---|---|---|
| 1 | FR-1 | Happy — trigger scan | No scan/fix lock held for `repo_ref`+`branch` | `POST /scans` with valid `repo_ref`+`branch` | `202 Accepted`, `scan_id` returned, `status=running` |
| 2 | FR-1 | Error — missing field | Request omits `branch` | `POST /scans` | `422 VALIDATION_FAILED` |
| 3 | FR-1 | Edge — lock contention (BR-1) | A scan is already `running` for the same `repo_ref`+`branch` | `POST /scans` again for the same repo+branch | `409 SCAN_LOCKED`, no second job enqueued |
| 4 | FR-2 | Happy — view issues | Scan completed, `LintIssue` rows persisted | `GET /scans/:id/issues` | Paginated, filterable table returned |
| 5 | FR-2 | Error — unknown scan | `scan_id` does not exist | `GET /scans/:id/issues` | `404 NOT_FOUND` |
| 6 | FR-2 | Boundary — zero issues | Scan completed clean (0 issues found) | `GET /scans/:id/issues` | Empty list, not an error |
| 7 | FR-3 | Happy — request plan | `issue_ids` all belong to one completed scan | `POST /plans {issue_ids}` | Plan created, generation begins |
| 8 | FR-3 | Error — empty batch | `issue_ids = []` | `POST /plans` | `422 VALIDATION_FAILED` |
| 9 | FR-3 | Edge — cross-scan batch | `issue_ids` span two different scans | `POST /plans {issue_ids}` | **Decided (default)**: `422 VALIDATION_FAILED` — a plan batch may only reference issues from a single scan |
| 10 | FR-4 | Happy — plan ready | Plan generation succeeded | `GET /plans/:id` | All 7 fields present (root cause, current behavior, recommended fix, risk, breaking change, files impacted, test plan) |
| 11 | FR-4 | Error — AI Layer failure | Claude call errors/times out | `GET /plans/:id` polled after failure | `502 AI_LAYER_UNAVAILABLE`, UI shows error state, not an infinite spinner |
| 12 | FR-4 | Edge — plan cache hit | Same issue+source fingerprint requested twice | Second `POST /plans` for the identical fingerprint | Cached plan returned, no second AI call (mitigates AI-risk NFR-6) |
| 13 | FR-5 | Happy — approve | Plan status is `[TBD enum]` pending-equivalent, caller role = `approver` | `POST /plans/:id/approve` | `status=approved`, approver id + timestamp recorded |
| 14 | FR-5 | Error — insufficient role | Caller role = `viewer` | `POST /plans/:id/approve` | `403 INSUFFICIENT_ROLE` |
| 15 | FR-5 | Boundary — high risk without confirmation (BR-5) | `risk_level=High` or `breaking_change=true` | `POST /plans/:id/approve` without confirmation flag | `422 CONFIRMATION_REQUIRED` |
| 16 | FR-6 | Happy — apply approved fix | `FixPlan.status=approved` | `POST /fixes {plan_id}` | Fix applied on a `lint-fix/*` branch, rescan triggered |
| 17 | FR-6 | Error — apply without approval (BR-3) | `FixPlan.status != approved` | `POST /fixes {plan_id}` | `409 PLAN_NOT_APPROVED`, no git write attempted |
| 18 | FR-6 | Edge — branch restriction (BR-2) | Any actor/role | Fix Service attempts a commit outside `lint-fix/*` | `403 BRANCH_RESTRICTED` — should be unreachable via the documented API contract; this is a defense-in-depth test at the Fix Service level, not just the Gateway |
| 19 | FR-6 | Boundary — autofixable vs AI-patch path | Issue's linter supports native `--fix` vs does not | `POST /fixes` | Both code paths (native fix, AI patch) produce a valid `FixHistory` row |
| 20 | FR-7 | Happy — fix confirmed resolved | Rescan runs after apply | Rescan completes | `result=passed`, issue transitions `fix_applied → resolved` (state machine, [03](03-proposed-workflow.md)) |
| 21 | FR-7 | Error — fix ineffective | Rescan still detects the same fingerprint | Rescan completes | `result=failed`, issue transitions `fix_applied → reopened` |
| 22 | FR-7 | Edge — severity increased post-fix (NFR-6 mitigation) | Rescan shows more/higher-severity issues than pre-fix | Rescan completes | Result flagged for review, not silently marked `passed` — mitigation stated in Risk register ([14](14-risk-analysis.md)) |
| 23 | FR-8 | Happy — view history tabs | Scans/fixes/rollbacks exist | `GET /history/scans`, `/history/fixes`, `/history/rollbacks` | Correct rows per tab |
| 24 | FR-8 | Boundary — empty history | No scans have ever run | `GET /history/scans` | Empty list, not an error |
| 25 | FR-8 | Edge — pagination at scale | Large history (hundreds of rows) | `GET /history/scans?limit=&offset=` | **Decided (default)**: offset/limit query params, default `limit=50`, max `limit=200` |
| 26 | FR-9 | Happy — rollback succeeds | `FixHistory` entry exists, no later conflicting commit | `POST /rollbacks {fix_history_id}` | `git revert` succeeds, confirmation rescan triggered, `RollbackHistory` written |
| 27 | FR-9 | Error — unknown fix history | `fix_history_id` does not exist | `POST /rollbacks` | `404 NOT_FOUND` |
| 28 | FR-9 | Edge — revert conflict (BR-4) | A later commit conflicts with the revert | `POST /rollbacks` | Manual-resolution state surfaced, revert is **not** forced (no reset/force-push) |
| 29 | FR-9 | Boundary — double rollback | Entry already has a completed `RollbackHistory` | `POST /rollbacks` again on the same `fix_history_id` | **Decided (default)**: `409 Conflict`, `Code=ALREADY_ROLLED_BACK` |
| 30 | FR-10 | Happy — role boundary equality | Caller role exactly equals an endpoint's minimum role | Request to that endpoint | Request succeeds (equality, not strictly-greater, must pass) |
| 31 | FR-10 | Error — below minimum role | Caller role below minimum | Request to any RBAC'd endpoint | `403 INSUFFICIENT_ROLE` |
| 32 | FR-11 | Happy — admin reads config | Caller role = `admin` | `GET /config` | Severity mapping, permissions, model config returned |
| 33 | FR-11 | Error — non-admin reads config | Caller role < `admin` | `GET /config` | `403 INSUFFICIENT_ROLE` |

## Regression Checklist

The dashboard is additive — none of the following, which already work today ([02-current-workflow.md](02-current-workflow.md)), may regress:

- `make lint` still generates JSON/EN-MD/MY-MD/HTML/SARIF reports under `golangci/linter-report/`.
- `make lint-fixed-plan N=<num>` still generates a rule-based plan file (unrelated code path from the new AI-based Plan Service).
- `make lint-fixed-plan-result N=<num>` still backs up, validates, and auto-rolls-back on failure.

## NFR Test Notes

| NFR | Test approach |
|---|---|
| NFR-1 (async) | Load-test `POST /scans` against a large repo; assert response time stays flat regardless of repo size (work happens off the request path) |
| NFR-2 (concurrency) | Concurrent `POST /scans` for the same repo+branch — assert exactly one proceeds, the other gets `409 SCAN_LOCKED` (scenario 3) |
| NFR-3 (security) | Unit test at the Fix Service level (not just API contract) that any commit target outside `lint-fix/*` is rejected (scenario 18) |
| NFR-4 (auditability) | Assert exactly one immutable `AuditLog` row per approve/apply/rollback action — no duplicates, no silent skips |
| NFR-5 (scalability) | Infra-level test, not unit-testable in this suite — worker pool must scale independently of API/UI; `[TBD — Architect/DevOps confirm]` how this gets verified pre-launch |
| NFR-6 (AI-risk) | Scenario 12 (plan cache) and scenario 22 (severity-increase guard) |
