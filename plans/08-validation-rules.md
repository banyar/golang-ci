# 08 — Validation Rules

**Date:** 2026-08-03
**Status:** Draft — **net-new**; the Architecture Artifact has no explicit validation-rules content
**Source:** Inferred from [07-database-design.md](07-database-design.md) field types + this repo's config-driven validation convention (`frontiir/config/ticket_create_validataion_required_fields.json`)
**Dependency:** [06-api-design.md](06-api-design.md), [07-database-design.md](07-database-design.md)

> **Net-new content notice**: the HTML Architecture Artifact contains no required/optional field list, no format constraints, and no enum value lists anywhere. Everything below is inferred from the ER diagram's field types (`string`/`int`/`bool`/`json`, PK/FK markers) and this repo's own config-driven convention — not sourced from the artifact. Enum value lists in particular need explicit `[TBD — BA/QA confirm]`.

## Field Validation Table (per write endpoint)

| Endpoint | Field | Required | Rule | Status |
|---|---|---|---|---|
| `POST /scans` | `repo_ref` | Yes | non-empty string | Inferred (PK-adjacent FK in `LintScan`) |
| `POST /scans` | `branch` | Yes | non-empty string | Inferred |
| `POST /plans` | `issue_ids` | Yes | non-empty array, all IDs must reference existing `LintIssue` rows in the same scan | Inferred (junction table implies batch) |
| `POST /plans/:id/approve` | approver identity | Yes (implicit, from auth context) | must resolve to a role ≥ `approver` | Inferred from RBAC table in [06](06-api-design.md) |
| `POST /fixes` | `plan_id` | Yes | must reference a `FixPlan` with `status = approved` | Inferred — see [10-business-rules.md](10-business-rules.md) Rule BR-3 (this is the business-rule side of the same constraint; API-layer validation just rejects early if approval is missing) |
| `POST /rollbacks` | `fix_history_id` | Yes | must reference an existing `FixHistory` row | Inferred |

## Cross-field / business validation

- `FixPlan.breaking_change = true` or `FixPlan.risk_level = High` → apply request must include an explicit confirmation flag (mirrors [10-business-rules.md](10-business-rules.md) Rule BR-5) — this is enforced at the API layer as a 422/409, not just a UI dialog.
- No cross-field rule beyond the above is stated or inferable from the source artifact.

## Enum value lists — **Decided (default, 2026-08-04)**

The ER diagram (07) uses plain `string` for these fields but they are clearly enum-shaped; exact allowed values were not given anywhere in the source material, so the values below are adopted defaults, not derivations:

| Field | Decided values | Basis |
|---|---|---|
| `LintScan.status` | `running`, `success`, `failed` | Workflow sequence diagram's own labels in [03](03-proposed-workflow.md) |
| `LintIssue.status` | `open`, `planned`, `fix_applied`, `resolved`, `reopened`, `ignored` | Taken directly from the state machine in [03](03-proposed-workflow.md) |
| `LintIssue.severity` | `critical`, `high`, `medium`, `low`, `info` | Standard 5-tier severity scale; UI mockup's "Critical"/"Minor" examples are free text within this set |
| `FixPlan.status` | `generating`, `pending`, `approved`, `rejected`, `applied`, `failed` | `approved` required by Rule BR-3 ([10](10-business-rules.md)); `generating`/`failed` added during M3 implementation — `06-api-design.md`'s "poll plan generation status" needs a distinguishable in-flight/failed sub-state that the original 4-value enum didn't have (see [07-database-design.md §M3 schema additions](07-database-design.md)) |
| `FixPlan.risk_level` | `low`, `medium`, `high` | Matches UI mockup's "Medium" example; kept as a distinct scale from the Risk register's `neutral/warn/crit` (those describe risk *categories* in [14](14-risk-analysis.md), not per-plan risk) |
| `FixHistory.result` | `passed`, `failed` | UI mockup ("Passed"/"Failed") |

## Config-driven vs code-driven

Following this repo's existing pattern (`frontiir/config/ticket_create_validataion_required_fields.json` — a flat `required_fields`/`optional_fields` JSON), simple per-endpoint required-field lists should live in a config file (e.g. `golangci/config/*_required_fields.json`), while cross-field/business rules (approval-gate check, branch-prefix restriction) stay in code — matching this repo's convention of "config for what varies, code for what's structural."
