# 06 — API Design

**Date:** 2026-08-03
**Status:** Draft — endpoint table sourced verbatim; versioning note is net-new (flagged)
**Source:** `golangci/golangci-lint-dashboard.html` §API Design
**Dependency:** [03-proposed-workflow.md](03-proposed-workflow.md) (steps→endpoints), [07-database-design.md](07-database-design.md) (entity exposure), [05-ui-design.md](05-ui-design.md) (UI needs)

## Endpoint Summary

Scan/Fix are long-running operations — most `POST` endpoints return `202 Accepted` immediately; the client polls or uses Server-Sent Events to follow progress.

| Method | Path | Purpose | Min. role |
|---|---|---|---|
| POST | `/scans` | Enqueue a scan for a repo + branch | `operator` |
| GET | `/scans/:id` | Scan status + summary counts | `viewer` |
| GET | `/scans/:id/issues` | Paginated, filterable issue list | `viewer` |
| GET | `/issues/:id` | Single issue detail + source snippet | `viewer` |
| POST | `/plans` | Request an AI Fix Plan for selected `issue_ids` | `viewer` |
| GET | `/plans/:id` | Poll plan generation status / read plan | `viewer` |
| POST | `/plans/:id/approve` | Human approval gate (records approver) | `approver` |
| POST | `/fixes` | Apply an approved plan on a `lint-fix/*` branch | `operator` |
| GET | `/fixes/:id` | Apply status + post-fix diff + rescan result | `viewer` |
| POST | `/rollbacks` | Revert a `FixHistory` entry (`git revert`) | `operator` |
| GET | `/history/scans` | Scan History audit view | `viewer` |
| GET | `/history/fixes` | Applied Fixes audit view | `viewer` |
| GET | `/history/rollbacks` | Rollback History audit view | `viewer` |
| GET | `/config` | Severity mapping, permission, model config | `admin` |

## Auth / Role notes

The role column (`viewer / approver / operator / admin`) is enforced by the API Gateway, not by the UI — see [12-component-design.md §Security controls](12-component-design.md) for exactly where each control lives. This mirrors this org's existing `api_key_permissions.json` ACL shape (per `AGENTS.md` §Security and gotchas).

## Versioning note

`[Net-new — not present in the source artifact]` — the HTML Architecture Artifact has no `/v1` prefix, no `API-Version` header, and no deprecation policy anywhere in its API Design section. **Decided (default, 2026-08-04)** — all 14 endpoints above are prefixed accordingly, e.g. `POST /api/v1/scans`:

| Item | Decision |
|---|---|
| Path prefix | `/api/v1/...` for all 14 endpoints above |
| Breaking change policy | New major version (`/api/v2`) on any breaking response-shape change; additive fields do not bump version |
| Deprecation window | No formal window for v1 — internal tooling, single consumer (the bundled UI); revisit if a second consumer appears |

This gap is also called out in [02-current-workflow.md §Open scope question](02-current-workflow.md#open-scope-question-surfaced-by-this-comparison) — no versioning exists in the current CLI tooling either, so there was no existing convention to inherit from within this sub-project; the decision above is a default, not a derivation, and should be revisited if it doesn't fit.
