# 10 — Business Rules

**Date:** 2026-08-03
**Status:** Draft — consolidated from real content scattered across the Architecture Artifact (Architecture, Security posture, Component Responsibilities, Risks sections) — no single rule list exists there, this file is the first place they're gathered together
**Source:** `golangci/golangci-lint-dashboard.html` §System Architecture, §Component Responsibilities, §Risks
**Dependency:** [04-prd.md](04-prd.md) (FR elaborate), [01-business-requirement.md](01-business-requirement.md) (goal trace)

## Business Rule Catalog

| ID | Rule | Rationale | Exception | Owner |
|---|---|---|---|---|
| BR-1 | Only one scan or fix may run at a time per repo+branch (Redis-based distributed lock) | Prevents two operations racing on the same git worktree/branch | None stated | `[TBD — BA/PM confirm]` |
| BR-2 | Fix Service may commit only to `lint-fix/*`-prefixed branches; direct write to `main`/`master` is never permitted, by any actor or role | Keeps AI-authored or autofix changes out of protected branches; human merge via PR/MR is the only path to `main` | None stated | `[TBD — BA/PM confirm]` |
| BR-3 | A `FixPlan` must have `status = approved` (with approver identity + timestamp recorded) before the Fix Service will accept an apply request — enforced server-side, not just hidden by the UI | Prevents an unreviewed AI-authored patch from ever being applied, even if a client bypasses the UI | None stated | `[TBD — BA/PM confirm]` |
| BR-4 | A rollback may only be performed via `git revert`; `git reset`/force-push are never used | Avoids destructive history rewrite and conflicts with commits made after the one being reverted; a revert conflict surfaces as a manual-resolution state instead of being forced | None stated | `[TBD — BA/PM confirm]` |
| BR-5 | If `FixPlan.risk_level = High` or `FixPlan.breaking_change = true`, an explicit confirmation step is required before apply, in addition to the normal approval gate (BR-3) | Extra friction for the riskiest changes, on top of the baseline approval requirement | None stated | `[TBD — BA/PM confirm]` |

## Rule-to-FR Traceability

| Rule | Traces to FR (see [04-prd.md](04-prd.md)) |
|---|---|
| BR-1 | Implicit precondition for FR-1 (trigger scan) and FR-6 (apply fix) |
| BR-2 | FR-6 (apply an approved fix on a `lint-fix/*` branch) |
| BR-3 | FR-5 (approve/reject gate), FR-6 (apply) |
| BR-4 | FR-9 (rollback) |
| BR-5 | FR-5 (approval gate — extended condition) |

No business rule beyond these 5 is stated or clearly inferable anywhere in the source material — anything else (e.g. a rule limiting how many issues one plan batch may contain, or a cooldown between scans) is `[TBD — BA/PM confirm]`, not invented here.
