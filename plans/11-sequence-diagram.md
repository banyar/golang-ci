# 11 — Sequence Diagrams (per-flow, fine-grained)

**Date:** 2026-08-03
**Status:** Draft — diagrams 1–3 decompose the single combined diagram already in [03](03-proposed-workflow.md) into per-flow views; diagram 4 (Rollback) is **net-new**, no rollback sequence exists in the source artifact
**Source:** `golangci/golangci-lint-dashboard.html` §Workflow (decomposed), §Component Responsibilities (Rollback Service description, for diagram 4)
**Dependency:** [03-proposed-workflow.md](03-proposed-workflow.md) (journey), [06-api-design.md](06-api-design.md) (endpoints), [07-database-design.md](07-database-design.md) (DB writes)

[03-proposed-workflow.md](03-proposed-workflow.md) has the single end-to-end diagram; this file splits it by flow and adds the explicit sync/async boundary each flow crosses, per Phase 2's intent for this file (finer-grained than 03, ahead of [12-component-design.md](12-component-design.md)).

## Flow 1 — Scan

**Sync/async boundary**: `POST /scans` returns `202 Accepted` immediately; the actual `golangci-lint` run happens on a worker, off the request path. Client polls `GET /scans/:id` for completion.

```mermaid
%%{init: {'theme':'neutral'}}%%
sequenceDiagram
  actor U as User
  participant UI as Web UI
  participant GW as API Gateway
  participant SC as Scan Service
  participant RUN as Runner (golangci-lint)
  participant PA as Parser Service

  U->>UI: Click "Scan Lint"
  UI->>GW: POST /scans
  GW->>SC: enqueue scan job
  SC-->>UI: 202 Accepted (scan_id, status=running)
  SC->>RUN: checkout worktree + run golangci-lint
  RUN-->>PA: raw JSON output
  PA->>PA: normalize + map severity
  PA-->>SC: LintIssue rows persisted
  SC-->>UI: status=success (poll)
```

## Flow 2 — Plan generation

**Sync/async boundary — Decided (default, 2026-08-04)**: `POST /plans` returns `202 Accepted` immediately, same as Flow 1 — an AI call has non-trivial, variable latency, so it gets the same async treatment for consistency. The UI polls `GET /plans/:id`.

```mermaid
%%{init: {'theme':'neutral'}}%%
sequenceDiagram
  actor U as User
  participant UI as Web UI
  participant GW as API Gateway
  participant PL as Plan Service
  participant AI as Claude (AI Layer)

  U->>UI: Select rows, click "View Plan"
  UI->>GW: POST /plans {issue_ids}
  GW->>PL: build prompt (rule + source + convention)
  PL->>AI: request structured fix plan
  AI-->>PL: root cause, fix, risk, test plan
  PL-->>UI: plan preview
```

## Flow 3 — Approve & Apply

**Sync/async boundary**: approval (`POST /plans/:id/approve`) is synchronous — it only flips a status field. Apply (`POST /fixes`) triggers work on a worker (git operations + re-scan), analogous to Flow 1's async boundary.

```mermaid
%%{init: {'theme':'neutral'}}%%
sequenceDiagram
  actor U as User
  participant UI as Web UI
  participant GW as API Gateway
  participant PL as Plan Service
  participant FX as Fix Service
  participant SC as Scan Service
  participant HI as History Service

  U->>UI: Review, click "Approve & Apply Fix"
  UI->>GW: POST /plans/:id/approve
  GW->>PL: mark plan approved (actor recorded)
  UI->>GW: POST /fixes {plan_id}
  GW->>FX: apply (autofix or AI patch) on lint-fix/* branch
  FX->>SC: trigger re-scan on fix branch
  SC-->>FX: post-fix issue delta
  FX->>HI: write FixHistory + AuditLog
  FX-->>UI: Passed / Failed + diff view
```

## Flow 4 — Rollback (`[net-new]`)

No rollback sequence diagram exists anywhere in the source artifact. This flow is derived from the Rollback Service's stated responsibility ([12-component-design.md](12-component-design.md): "Reverts a `FixHistory` entry via `git revert`, triggers confirmation rescan") and Business Rule BR-4 ([10-business-rules.md](10-business-rules.md): revert-only, never reset/force-push).

```mermaid
%%{init: {'theme':'neutral'}}%%
sequenceDiagram
  actor U as User
  participant UI as Web UI
  participant GW as API Gateway
  participant RB as Rollback Service
  participant SC as Scan Service
  participant HI as History Service

  U->>UI: Click "Rollback" on a FixHistory entry
  UI->>GW: POST /rollbacks {fix_history_id}
  GW->>RB: git revert (never reset/force-push)
  alt revert succeeds
    RB->>SC: trigger confirmation re-scan
    SC-->>RB: post-rollback issue state
    RB->>HI: write RollbackHistory + AuditLog
    RB-->>UI: rollback done
  else revert conflicts with a later commit
    RB->>HI: write RollbackHistory (result=conflict) + AuditLog
    RB-->>UI: manual-resolution required (no forced revert)
  end
```

**Decided (default, 2026-08-04)**: `POST /rollbacks` returns `202 Accepted`, same as the other three flows — it involves a git operation plus a confirmation rescan, which is not instant.
