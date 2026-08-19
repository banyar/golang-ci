# 15 — Implementation Plan

**Date:** 2026-08-03
**Status:** Draft — task breakdown/sequencing only. **No code, no ticket numbers** (a real backlog tool is needed to assign those)
**Source:** [12-component-design.md](12-component-design.md) (folder/component boundary), [13-test-plan.md](13-test-plan.md) (Definition of Done), [14-risk-analysis.md](14-risk-analysis.md) (risk-mitigating tasks)
**Dependency:** [12-component-design.md](12-component-design.md) (task boundary), [13-test-plan.md](13-test-plan.md) (DoD), [14-risk-analysis.md](14-risk-analysis.md) (risk-mitigating task)

## Task Breakdown (by component)

| Component | Tasks |
|---|---|
| `storage/` | GORM models for `LintScan`, `LintIssue`, `FixPlan`, `FixHistory`, `RollbackHistory`, `Configuration` ([07](07-database-design.md)); migrations |
| `config/` | `severity-mapping.json`, `permissions.json`, `model-config.json`; per-endpoint required-field JSON following the `ticket_create_validataion_required_fields.json` pattern ([08](08-validation-rules.md)) |
| `scanner/` | Isolated worktree checkout; `golangci-lint run --out-format json` invocation; Redis-based per-repo/branch lock acquire/release (BR-1, [10](10-business-rules.md)); scan status tracking |
| `parser/` | Raw JSON → `LintIssue` normalization; severity-mapping lookup; fingerprint computation (`hash(file_path + rule + normalized_message)`, [07](07-database-design.md)) |
| `worker/` | Job queue consumer (Redis) — dispatches scan and fix jobs off the API request path (NFR-1, NFR-5) |
| `planner/` | Prompt template (rule + source context + repo conventions); Claude client call; plan cache keyed by fingerprint (mitigates NFR-6 non-determinism risk); approval-status transition (BR-3) |
| `fixer/` | Native `--fix` invocation for autofixable linters; AI-authored patch application otherwise; git branch/commit ops hard-scoped to `lint-fix/*` (BR-2); triggers Scan Service rescan |
| `history/` | Append-only audit read models (scan/fix/rollback); immutable `AuditLog` write path, called from `planner/`, `fixer/`, and the rollback flow (NFR-4) |
| `api/` | Gin handlers for all 14 endpoints ([06](06-api-design.md)); RBAC middleware (`viewer/approver/operator/admin`); request validation ([08](08-validation-rules.md)); error responses via `utils.RestErr` ([09](09-error-handling.md)) — including the new `utils.ForbiddenErr` needed to close Open Risk #6 |
| `ui/` | 5 screens: Dashboard, Issue List, Plan Viewer, Fix Progress, History ([05](05-ui-design.md)); state management per the 4-state ownership table |

## Milestones & Sequencing

| Milestone | Scope | Depends on | DoD (test scenarios from [13-test-plan.md](13-test-plan.md)) |
|---|---|---|---|
| M1 — Foundations | `storage/`, initial `config/`, `api/` skeleton + RBAC middleware scaffold | none | Migrations run cleanly; RBAC middleware rejects unauthenticated requests |
| M2 — Scan loop | `scanner/`, `parser/`, `worker/`, `api/` scan endpoints, `ui/` Dashboard + Issue List | M1 | Scenarios 1–6 (FR-1, FR-2) |
| M3 — Plan loop | `planner/`, `api/` plan endpoints, `ui/` Plan Viewer | M2 | Scenarios 7–12 (FR-3, FR-4) |
| M4 — Approve + Fix + Rescan | `api/` approve/fix endpoints, `fixer/`, `ui/` Fix Progress | M3 | Scenarios 13–22 (FR-5, FR-6, FR-7); BR-2, BR-3, BR-5 enforced |
| M5 — History + Rollback | `history/`, `api/` history + rollback endpoints, `ui/` History | M4 | Scenarios 23–29 (FR-8, FR-9); BR-4 enforced |
| M6 — Hardening | Full RBAC coverage, error-code additions ([09](09-error-handling.md) gaps), audit-log completeness, resolve as many [14-risk-analysis.md §Open Risks](14-risk-analysis.md) as possible | M5 | Scenarios 30–33 (FR-10, FR-11); NFR test notes ([13](13-test-plan.md)) |

## Definition of Done (applies to every task above)

- Relevant GWT scenario(s) from [13-test-plan.md](13-test-plan.md) pass.
- No direct writes to `main`/`master` observed anywhere (BR-2) — verified via audit log, not just code review.
- Corresponding item in [16-review-checklist.md](16-review-checklist.md)'s Pre-Implementation Checklist is checked before the task's code is merged.

## Explicitly out of scope for this file

Effort estimates, sprint assignment, and ticket numbers are **not** included — `[TBD — Dev/PM]`: this task breakdown is the seed to be loaded into a real backlog tool (Jira/Redmine), not the backlog itself. Per `plans-structure-analysis.md` Phase 3.2, this is also the file whose completion (plus [16](16-review-checklist.md)'s Pre-Implementation Checklist being ✅) is the explicit gate before any of these tasks' code may actually be written.
