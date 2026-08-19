# 05 — UI Design

**Date:** 2026-08-03
**Status:** Draft — sourced verbatim from the Architecture Design Artifact's UI Design section
**Source:** `golangci/golangci-lint-dashboard.html` §UI Design
**Dependency:** [03-proposed-workflow.md](03-proposed-workflow.md) (journey→screen), [04-prd.md](04-prd.md) (FR→field/action)

## Screen Flow

**Dashboard → Issue List → Plan Viewer → Fix Progress → History**

Each screen's state is tied to a server-side status field, not local-only UI state — a browser refresh or a re-opened tab rehydrates the current state immediately (see §State Management below).

## Screen 1 — Dashboard

Summary tiles + primary action.

| Tile | Example value |
|---|---|
| Total issues | 238 |
| Critical | 14 (critical/red style) |
| Fixed this week | 31 (ok/green style) |
| Last scan | 2 min ago |

Actions: **▶ Scan Lint** (primary button) · **View recent scans** (ghost/secondary button).

## Screen 2 — Issue List

Filterable, selectable table.

Toolbar filters (chips): `Severity: All`, `Linter: All`, `File contains…`
Bulk action bar (example state): "2 selected — View Plan"

Columns: ☐ · File · Line:Col · Linter · Severity · Message · Suggested fix · Status

| ☐ | File | Line:Col | Linter | Severity | Message | Suggested fix | Status |
|---|---|---|---|---|---|---|---|
| ☑ | services/ticket_service.go | 142:9 | errcheck | Critical | Error return value not checked | Needs AI plan | Open |
| ☑ | controllers/ticket_controller.go | 88:1 | gofumpt | Minor | File is not gofumpt-ed | Auto-fixable | Open |
| ☐ | helpers/field_mapper.go | 54:12 | gocritic | Minor | ifElseChain: rewrite as switch | Needs AI plan | Open |

## Screen 3 — Plan Viewer

AI Fix Plan preview, per selected batch.

- **Root cause**: Return value of `tx.Commit()` is discarded; a failed commit currently looks identical to a successful one.
- **Current behavior**: On commit failure, the handler still returns `200 OK` to the caller.
- **Recommended fix**: Capture the error, wrap with `fmt.Errorf`, return `*utils.RestErr` to the controller.
- **Risk**: Medium (warn style)
- **Breaking change**: No (neutral style)
- **Files impacted**: `ticket_service.go` (chip)
- **Test plan** (embedded GWT table — full matrix lives in [13-test-plan.md](13-test-plan.md), this is the per-plan preview only):

| Given | When | Then |
|---|---|---|
| DB commit fails | ticket create is called | handler returns a 5xx RestErr, not 200 |
| DB commit succeeds | ticket create is called | behavior unchanged (regression check) |

Actions: **Reject** (ghost button) · **Approve & Apply Fix** (primary button).

## Screen 4 — Fix Progress

Apply → rescan → result, shown as a 4-step stepper:

1. Queued — done (✓)
2. Applying — done (✓)
3. Re-scanning — active
4. Result — pending

Status line example: `Branch lint-fix/scan-4821/errcheck-batch · 1 file changed · diff review pending re-scan.`

## Screen 5 — History

Audit tabs: **Scan history** (active) · **Applied fixes** · **Rollback history**

Columns: Scan ID · Repo / branch · Triggered by · Issues found · Result

| Scan ID | Repo / branch | Triggered by | Issues found | Result |
|---|---|---|---|---|
| #4821 | rt-external-api-v1 / ticket-create-sla | banyar.sithu | 238 | Success |
| #4809 | rt-external-api-v1 / master | scheduled | 251 | Success |

## State Management

State is server-backed, not local-UI-owned — the UI polls/renders, it does not own state transitions.

| State | Owned by | Transitions |
|---|---|---|
| Issue selection | UI (`sessionStorage Set<issue_id>`) | Survives filter/pagination changes; cleared on scan change |
| Plan loading | `FixPlan.status` | `idle → requesting → ready \| error`, polled via `GET /plans/:id` |
| Fix progress | `FixHistory.result` | `queued → applying → rescanning → passed \| failed` |
| Rollback | `RollbackHistory` | `confirm → in_progress → done \| error` |

No additional per-field state breakdown exists in the source artifact beyond these 4 states — anything more granular is `[TBD — BA/PM confirm]` if needed.
