# 00 — Project Overview: golangci-lint Web Dashboard

**Date:** 2026-08-03
**Status:** System Design Review · Draft · No Implementation (mirrors the Architecture Artifact's own header label)
**Source:** `golangci/dashboard.md`, `golangci/golangci-lint-dashboard.html`, `golangci/plans-structure-analysis.md`
**Dependency:** none — this is the suite's root/entry document

## Objective

Upgrade the CLI-only golangci-lint workflow in this repo into a Web UI dashboard: Scan → interactive Issue List → AI-generated Fix Plan (Claude) → human Approval → Apply (on a `lint-fix/*` branch) → automatic Re-scan → persistent History/Rollback. See [01-business-requirement.md](01-business-requirement.md) for the full problem statement.

## Stakeholders

| Role | Name | Status |
|---|---|---|
| Requestor | Banyar Sithu (banyar.sithu@frontiir.net) | Confirmed |
| BA/PM owner | `[TBD — BA/PM confirm]` | Not yet named |
| Architect / Senior Dev (11, 12 author) | `[TBD — BA/PM confirm]` | Not yet named |
| QA (13 author) | `[TBD — BA/PM confirm]` | Not yet named |

## Status

All 17 files have been authored, in the exact Group A→F order derived in `golangci/plans-structure-analysis.md` Phase 3, with the requestor's review after each checkpoint:

- ✅ Checkpoint A (00–04)
- ✅ Checkpoint B (05–07)
- ✅ Checkpoint C (08–10)
- ✅ Checkpoint D (11–12)
- ✅ Checkpoint E (13–14)
- ✅ Checkpoint F (15–16)

**This is a documentation/design-review milestone, not a launch.** [16-review-checklist.md §Pre-Implementation Checklist](16-review-checklist.md) has 15 unchecked items, and [14-risk-analysis.md §Open Risks needing decision](14-risk-analysis.md) has 11 open items — both need real BA/PM/Architect/Dev/QA sign-off before milestone M1 in [15-implementation-plan.md](15-implementation-plan.md) begins. No code under `golangci/api|scanner|parser|planner|fixer|history|worker|storage|ui|config/` has been written as part of authoring this suite.

## Document Index

| # | File | Purpose | Status |
|---|---|---|---|
| 00 | [00-project-overview.md](00-project-overview.md) | Entry point / index (this file) | ✅ |
| 01 | [01-business-requirement.md](01-business-requirement.md) | Raw business ask, problem statement | ✅ |
| 02 | [02-current-workflow.md](02-current-workflow.md) | As-is CLI workflow (grounded in real scripts) | ✅ |
| 03 | [03-proposed-workflow.md](03-proposed-workflow.md) | To-be user journey, sequence + state diagrams | ✅ |
| 04 | [04-prd.md](04-prd.md) | Enterprise PRD (FR/NFR/AC) | ✅ (21-section structure needs confirmation — see file) |
| 05 | [05-ui-design.md](05-ui-design.md) | Screen wireframes | ✅ |
| 06 | [06-api-design.md](06-api-design.md) | REST endpoint contract | ✅ (versioning scheme unconfirmed — see file) |
| 07 | [07-database-design.md](07-database-design.md) | ER diagram + field tables | ✅ |
| 08 | [08-validation-rules.md](08-validation-rules.md) | Field/business validation spec | ✅ (enum lists TBD — see file) |
| 09 | [09-error-handling.md](09-error-handling.md) | Error taxonomy + HTTP status mapping | ✅ (403/502 constructor gap in `utils.go` — see file) |
| 10 | [10-business-rules.md](10-business-rules.md) | Domain rule catalog | ✅ |
| 11 | [11-sequence-diagram.md](11-sequence-diagram.md) | Fine-grained per-flow sequence diagrams | ✅ (Rollback flow is net-new — see file) |
| 12 | [12-component-design.md](12-component-design.md) | Service breakdown + folder structure | ✅ |
| 13 | [13-test-plan.md](13-test-plan.md) | QA GWT test matrix (33 scenarios) | ✅ |
| 14 | [14-risk-analysis.md](14-risk-analysis.md) | Risk register + 11 open items | ✅ |
| 15 | [15-implementation-plan.md](15-implementation-plan.md) | Task breakdown, milestones, DoD | ✅ |
| 16 | [16-review-checklist.md](16-review-checklist.md) | Pre/Post-Implementation sign-off gate | ✅ |

## How to read this suite

Read in numeric order for a linear pass, or jump by role:

- **BA/PM**: 00 → 01 → 04 → 16
- **Architect/Dev**: 02 → 03 → 06 → 07 → 11 → 12 → 15
- **QA**: 04 (AC overview) → 08 → 09 → 13
- **Everyone, before any code is written**: 16's Pre-Implementation Checklist must be ✅ first — per `plans-structure-analysis.md` Phase 3.2, this is a hard gate, not a suggestion.
