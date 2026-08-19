# 04 — Product Requirements Document (PRD)

**Date:** 2026-08-03
**Status:** Draft — consolidates [01](01-business-requirement.md)+[02](02-current-workflow.md)+[03](03-proposed-workflow.md); needs Dev/QA/PM sign-off
**Source:** 01, 02, 03 (this repo's own prior suite files) + `golangci/golangci-lint-dashboard.html`
**Dependency:** [01-business-requirement.md](01-business-requirement.md), [02-current-workflow.md](02-current-workflow.md), [03-proposed-workflow.md](03-proposed-workflow.md)

> **Decided (default, 2026-08-04)**: `golangci/plans-structure-analysis.md` row 04 references *"User သတ်မှတ်ထားသော 21 Section structure"* (a 21-section structure the user had already specified) — that exact list was never found in `plans-structure-analysis.md`, `dashboard.md`, or the HTML artifact. The standard enterprise-PRD scaffold below is accepted as final for this build cycle. Reopen if the original list turns up.

## 1. Document Control

| Field | Value |
|---|---|
| Feature | golangci-lint Web Dashboard |
| Author(s) | Claude (draft), Banyar Sithu (owner — de facto BA/PM for this build cycle, per [01-business-requirement.md §Requestor](01-business-requirement.md)) |
| Reviewers | Banyar Sithu (self-review, single-person project at this stage) |
| Status | Draft — System Design Review, no implementation started |

## 2. Background & Problem Statement

See [01-business-requirement.md §Background, §Problem Statement](01-business-requirement.md). Summary: lint triage/fix is CLI-only today, uses a rule-based (not AI) plan template, handles one issue at a time, and has no persistent history or approval gate.

## 3. Business Goal & Objectives

See [01-business-requirement.md §Business Goal](01-business-requirement.md). Summary objective: move the full Scan → Plan → Approve → Fix → Rescan → History loop into a browser-based, AI-assisted, human-approved workflow.

## 4. Scope (in)

- Web UI for: Dashboard, Issue List, Plan Viewer, Fix Progress, History (5 screens — [05-ui-design.md](05-ui-design.md)).
- REST API surface for scans, issues, plans, fixes, rollbacks, history, config (14 endpoints — [06-api-design.md](06-api-design.md)).
- AI-generated (Claude) Fix Plans, replacing the current rule-based bash template.
- Human approval gate before any fix is applied.
- Persistent, browsable Scan/Fix/Rollback history.

## 5. Out of scope / Non-goals

Per the HTML artifact's own "Future Enhancements" section ([14-risk-analysis.md](14-risk-analysis.md) cross-refs the risk side of this): SARIF export beyond what already exists in `golangci-report.sh`, PR/MR bot mode, pluggable non-Go linters (`eslint`/`ruff`), AI cost guardrails beyond a basic cap, and trend analytics dashboards are all explicitly v2+, not v1.

## 6. Stakeholders & Roles

| Stakeholder | Interest |
|---|---|
| Requestor (Banyar Sithu) | Primary developer requesting the tooling upgrade |
| BA/PM | Banyar Sithu (de facto, per §1) |
| QA | Consumes [13-test-plan.md](13-test-plan.md) |
| Architect/Senior Dev | Authors [11](11-sequence-diagram.md)/[12](12-component-design.md) |

## 7. Personas / User types

Maps 1:1 to the RBAC roles already defined in the Architecture Artifact — see [03-proposed-workflow.md §Actors & Roles](03-proposed-workflow.md#actors--roles) and [06-api-design.md](06-api-design.md) for the full role-to-endpoint table. No additional personas beyond `viewer / approver / operator / admin` are defined anywhere in the source material.

## 8. User journey summary

See [03-proposed-workflow.md §User Journey](03-proposed-workflow.md) for the full 9-step journey and sequence diagram. Not duplicated here.

## 9. Functional Requirements

| ID | Requirement | Source |
|---|---|---|
| FR-1 | User can trigger a lint scan for a repo+branch from the Web UI | `POST /scans` |
| FR-2 | User can view scan status and a filterable, checkbox-selectable issue list | Issue List screen |
| FR-3 | User can select one or more issues and request an AI-generated Fix Plan | `POST /plans` |
| FR-4 | Fix Plan must show: root cause, current behavior, recommended fix, risk, breaking-change flag, files impacted, test plan | Plan Viewer screen |
| FR-5 | User (with `approver` role) can approve or reject a Fix Plan; the approving actor is recorded | `POST /plans/:id/approve` |
| FR-6 | An approved plan can be applied only on a `lint-fix/*` branch — never directly to `main`/`master` | `POST /fixes` |
| FR-7 | After a fix is applied, the system automatically re-scans and reports Passed/Failed with a diff | Fix Progress screen |
| FR-8 | User can browse Scan History, Applied Fixes, and Rollback History as separate audit views | History screen, `GET /history/*` |
| FR-9 | User (with `operator` role) can roll back an applied fix via `git revert` (never reset/force-push) | `POST /rollbacks` |
| FR-10 | Every API endpoint enforces RBAC per the `viewer/approver/operator/admin` role table | API Gateway |
| FR-11 | Admin can configure severity mapping and AI model choice | `GET /config` |

## 10. Non-Functional Requirements

| ID | Requirement | Rationale |
|---|---|---|
| NFR-1 (Async) | Scan/Fix requests must return `202 Accepted` immediately; execution happens out of the request path (worker), status is polled/subscribed | Scans can take many seconds on large repos |
| NFR-2 (Concurrency) | At most one scan/fix per repo+branch at a time, enforced by a Redis-based distributed lock | Prevent two fixes racing on the same file/branch |
| NFR-3 (Security) | Fix Service may only write to `lint-fix/*` branches; no direct `main`/`master` write access | Branch-protection posture, confirmed constraint |
| NFR-4 (Auditability) | Every approve/apply/rollback action writes an immutable audit entry, separate from mutable status fields | Governance/traceability |
| NFR-5 (Scalability) | Worker pool scales independently of the API/UI layer; `Configuration` is scoped to `repo_ref + subpath` for monorepo support | Repo count/size growth shouldn't require re-architecture |
| NFR-6 (AI-risk mitigation) | Mandatory human approval gate; automatic rescan-diff check; block apply if severity count increases after a fix | AI-authored patches are plausible but can be behavior-changing — see [14-risk-analysis.md](14-risk-analysis.md) |
| — | Exact numeric SLA targets (max scan latency, max concurrent scans, uptime target) | **Decided (default)**: none formalized — internal tool at current scale; revisit if usage grows enough to warrant a real SLA |

## 11. API overview

See [06-api-design.md](06-api-design.md) for the full 14-endpoint table. Not duplicated here.

## 12. Data overview

See [07-database-design.md](07-database-design.md) for the full ER diagram and field tables. Not duplicated here.

## 13. UI/UX overview

See [05-ui-design.md](05-ui-design.md) for all 5 screens. Not duplicated here.

## 14. Business rules

See [10-business-rules.md](10-business-rules.md) for the full rule catalog (per-repo/branch lock, branch-prefix restriction, approval-gate rule, revert-only rollback, high-risk confirmation). Not duplicated here.

## 15. Validation rules

See [08-validation-rules.md](08-validation-rules.md). Flagged there as **net-new** — the source material has no explicit field-validation content.

## 16. Error handling overview

See [09-error-handling.md](09-error-handling.md). Flagged there as **net-new**, grounded in this repo's existing `utils.RestErr` convention.

## 17. Security & Compliance

RBAC (`viewer/approver/operator/admin`), approval gate, branch-scoped git credentials, and audit logging are covered in [12-component-design.md §Security controls](12-component-design.md). No additional compliance requirement (e.g. data residency, PII handling) is stated anywhere in the source material. **Decided (default)**: none applies — this is an internal developer tool operating on this repo's own source code, not customer/PII data.

## 18. Dependencies & Assumptions

| Item | Type |
|---|---|
| Reuses existing GORM + MySQL stack, `git.frontiir.net/sa-dev/rtdatacore` pattern | Confirmed dependency |
| Assumes Claude (Messages API) as the AI Layer — no alternative model evaluated in source material | Assumption |
| Assumes `golangci-lint` remains the only linter for v1 (multi-linter support is Future Enhancement) | Assumption |

## 19. Risks (summary)

See [14-risk-analysis.md](14-risk-analysis.md) for the full 7-row risk register. Not duplicated here.

## 20. Acceptance Criteria (overview)

Full Given/When/Then matrix lives in [13-test-plan.md](13-test-plan.md) (per this suite's own no-duplication rule — [plans-structure-analysis.md §1.2](../plans-structure-analysis.md)). At PRD level, each FR above is considered accepted only when its corresponding row(s) in 13 pass.

## 21. Open Questions / Sign-off

All items formerly listed here were resolved with default assumptions on 2026-08-04 — see [14-risk-analysis.md §Open Risks](14-risk-analysis.md) for the consolidated resolution table across all 11 items raised in this suite. Sign-off recorded in [16-review-checklist.md](16-review-checklist.md).
