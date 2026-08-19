# 12 — Component Design

**Date:** 2026-08-03
**Status:** Draft — sourced verbatim from the Architecture Design Artifact's Component Responsibilities section
**Source:** `golangci/golangci-lint-dashboard.html` §Component Responsibilities
**Dependency:** [11-sequence-diagram.md](11-sequence-diagram.md), [07-database-design.md](07-database-design.md), [06-api-design.md](06-api-design.md)

Six services, each single-responsibility; only the Fix Service has git write access.

## Service Breakdown

| Service | Responsibility | Depends on |
|---|---|---|
| Scan Service | Checks out an isolated worktree, runs `golangci-lint run --out-format json`, tracks scan status | Job queue, Runner |
| Parser Service | Normalizes raw JSON → `LintIssue` rows; applies severity-mapping config; computes fingerprint | Configuration |
| Plan Service | Builds the AI prompt (rule + source context + repo conventions), calls Claude, caches plan by fingerprint | AI Layer, `LintIssue` |
| Fix Service | Applies native `--fix` for autofixable linters; applies AI-authored patch for the rest; commits only to `lint-fix/*` | Git, Scan Service (rescan) |
| History Service | Append-only audit read model over scans, fixes, rollbacks | Storage |
| Rollback Service | Reverts a `FixHistory` entry via `git revert` (never reset/force-push), triggers confirmation rescan | Git, Scan Service |

## Security controls, mapped to component

| Control | Where enforced | Behavior |
|---|---|---|
| Confirmation | UI + API Gateway | Explicit confirm required when `risk = High` or `breaking_change = true` (see [10-business-rules.md](10-business-rules.md) BR-5, [08-validation-rules.md](08-validation-rules.md)) |
| Approval | Plan Service | `FixPlan.status` must be `approved` (approver id + timestamp) before Fix Service accepts the request — enforced server-side, not hidden by UI alone (see [10-business-rules.md](10-business-rules.md) BR-3) |
| Permission (RBAC) | API Gateway | `viewer / approver / operator / admin` roles, same ACL shape as this org's existing `api_key_permissions.json` convention |
| Audit log | History Service | Every approve/apply/rollback writes an immutable entry (actor, before/after, timestamp) — separate from mutable status fields |

## Recommended Folder Structure

```
golangci/
├── api/          ← Gin handlers, request validation, RBAC middleware
├── scanner/      ← golangci-lint process orchestration, worktree lifecycle
├── parser/       ← JSON → LintIssue normalization, severity mapping
├── planner/      ← AI Layer client, prompt templates, plan cache
├── fixer/        ← autofix + AI patch application, git branch/commit ops
├── history/      ← scan/fix/rollback audit read models
├── worker/       ← job queue consumer — separate scalable process (see §Scalability, [01-business-requirement.md](01-business-requirement.md))
├── storage/      ← GORM models + migrations
├── ui/           ← SPA — dashboard, issue list, plan viewer, history
└── config/       ← severity-mapping.json, permissions.json, model-config.json
```

`worker/` was not in the user's original folder list in `dashboard.md` — it was added as a senior-level recommendation in the Architecture Artifact, specifically to keep scan/fix jobs out of the API process for horizontal scaling (per [01-business-requirement.md §Constraints](01-business-requirement.md) and the Scalability rationale in [00](00-project-overview.md)'s source material). This suite's own `golangci/plans/` directory sits alongside these, outside all ten — it is documentation, not a runtime package.
