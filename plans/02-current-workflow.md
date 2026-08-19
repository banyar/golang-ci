# 02 — Current Workflow (As-Is)

**Date:** 2026-08-03
**Status:** Draft — factual, grounded in the actual scripts (not inferred)
**Source:** `golangci/Makefile`, `golangci/cmd/golangci-report.sh`, `golangci/cmd/lint-fixed-plan.sh`, `golangci/cmd/lint-fixed-plan-result.sh`, `golangci/cmd/lint-fixed-plan.md`
**Dependency:** [01-business-requirement.md](01-business-requirement.md)

## Current Steps

| # | Step | Command | Detail |
|---|---|---|---|
| 1 | Scan | `make lint` (from `golangci/`) | Runs `golangci/cmd/golangci-report.sh` from repo root. For each config in `golangci/lint-configs.txt`, runs `golangci-lint run --out-format json` and generates, per config, under `golangci/linter-report/<config-name>/{TS}.*`: `.json` (raw), `-en.md`, `-my.md`, `.html` (self-contained, dark-mode aware), `.sarif` (GitLab/GitHub/VSCode). A combined `summary-{TS}.md` is written to the report root. |
| 2 | Review | *(manual)* | Developer opens the generated `.html` or `.md` report directly — no interactive filtering/sorting beyond what a static file offers. |
| 3 | Plan | `make lint-fixed-plan N=<issue_number>` | Runs `lint-fixed-plan.sh <N>`. Finds the latest JSON report, extracts issue `N`, and generates a **rule-based** (not AI-generated) plan file at `plans/YYYY-MM-DD-lint-fix-<N>.md` using a hardcoded `case`/`switch` on linter name (`errcheck`, `gosec` sub-rules, `gocyclo`, `nestif`, `ineffassign`, `misspell`, `revive`, `funlen`, `bodyclose`, `staticcheck`, `govet`, fallback). Plan content follows a fixed 9-section template (`lint-fixed-plan.md`): Issue summary, Original code context, Root cause analysis, Fix strategy, Before→After preview, Possible side effects, Impact analysis, Recommended tests, Acceptance criteria. |
| 4 | Apply | *(manual)* | Developer edits the source file in their own editor, following the generated plan. No automated patch application exists today. |
| 5 | Verify | `make lint-fixed-plan-result N=<issue_number>` | Runs `lint-fixed-plan-result.sh <N>`. Requires `jq`, `python3`, `golangci-lint`, `gofmt`. Always backs up the source file first; re-runs lint/build validation; **rolls back automatically** on validation failure. Writes a result report to `golangci/after-fixed/YYYY-MM-DD-lint-fixed-<N>.md`. |
| 6 | *(no step)* | — | No batch/multi-issue handling — one `N` at a time. No manual rollback command beyond the automatic one inside step 5. No persistent, browsable scan/fix/rollback history. |

## Actors Involved

| Actor | Involvement |
|---|---|
| Developer (CLI operator) | Runs all `make` targets, reads reports, edits code manually. Sole actor in the current flow. |
| BA / PM / QA | No touchpoint — the current flow has no approval step, no audit trail, no non-technical-readable UI. |

## Pain Points

| Pain point | Evidence |
|---|---|
| CLI/file-only review — no interactive table, no filter/sort/multi-select | Reports are static `.md`/`.html` files (`golangci-report.sh` output) |
| Fix plan is a static rule template, not contextual AI reasoning | `lint-fixed-plan.sh`'s hardcoded `case` statement per linter |
| One issue at a time (`N=<num>`) | `lint-fixed-plan.sh` / `lint-fixed-plan-result.sh` usage contract |
| No persistent scan history by default | `golangci-report.sh`'s `KEEP_HISTORY` defaults to `0` — "delete all old" runs |
| No approval gate before a fix is "applied" (script edits are still manual) | No confirmation/approval concept anywhere in the three scripts |
| No browsable rollback history (only automatic rollback-on-failure inside one script run) | `lint-fixed-plan-result.sh`'s backup/rollback is internal to a single invocation, not a queryable history |

## Cost of Current Process

`[TBD — BA/PM confirm]` — no time/effort measurement of the current CLI flow exists in any source file; an actual cost baseline (e.g. average minutes per issue from scan to verified fix) needs to be measured or estimated by BA/Dev before it can be compared against the proposed workflow in [03-proposed-workflow.md](03-proposed-workflow.md).

## Open scope question surfaced by this comparison

The current pipeline already produces English **and** Myanmar Markdown reports plus SARIF (for GitHub/GitLab code scanning) — none of which are mentioned in the original dashboard ask (`dashboard.md`) or the Architecture Artifact. `[TBD — BA/PM confirm]`: should the new Web UI **replace** `golangci-report.sh`'s file-based reports entirely, or run alongside them (e.g. keep SARIF export for CI integration)? Architecture Artifact's own "Future Enhancements" section separately proposes a *new* SARIF export from the Parser Service — worth reconciling with the SARIF export this script already has today.
