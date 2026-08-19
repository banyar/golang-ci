# Fix the 123 lint issues found by `make lint`

## Context

`make lint` (fixed in the prior task) now runs cleanly end-to-end against the
real `golangci/` dashboard code plus `frontiir/utils/` (copied from
`rt-external-api-v1` to satisfy that code's own import requirements), using
the external `rt-external-api-with-auto-remote-resolved/.golangci.yml` config.
It reports 123 real issues across 10 linters/formatters. The user wants these
fixed.

The project already ships a semi-automated per-issue tool
(`golangci/cmd/lint-fixed-plan.sh` + `lint-fixed-plan-result.sh`, driven by
`make lint-fixed-plan N=<n>` / `make lint-fixed-plan-result N=<n>`), but it's
hardcoded to `--config golangci.yml` and to searching callers under
`frontiir/` — both assumptions built for the original `rt-external-api`
layout, not our current external-config setup. Using it as-is, one issue at a
time, for 123 issues would also be far slower than needed for the ~58% of
issues that are pure mechanical formatting. This plan handles the bulk
directly and reserves that per-issue tool (or equivalent manual care) for the
handful of genuinely risky issues.

## Breakdown by tier (123 issues total)

**Tier 1 — auto-fix, zero behavior risk (43 issues): `gofumpt`(13) + `gci`(10)
+ `golines`(19) + `goimports`(1).** All four are formatters already enabled in
the external config (`formatters.enable`). `golangci-lint run --fix` applies
them mechanically (whitespace/import-grouping/line-wrapping only).

**Tier 2 — mechanical, low risk, `golangci/`'s own code only (28 issues):**
- `revive` missing doc-comments / package-comments (11): `planner/mock_client.go`,
  `planner/service.go`, `storage/db.go`, `storage/models.go` (x6), `api/fixes.go`
- `errorlint` (1): `planner/service.go:131` — `%v` → `%w` on an already-wrapped error
- `gosec` G112 (1): `cmd/dashboard/main.go:93` — add `ReadHeaderTimeout` to `http.Server`
- `mnd` (15 of 19 — excludes the 4 inside `frontiir/utils.go`, see below):
  named constants for role levels (`api/middleware.go`), a list-limit
  (`api/scans.go`), DB pool settings (`storage/db.go`), UUID byte length
  (`storage/models.go`), worker `BLPop` timeouts (`worker/*.go`), goroutine
  count and shutdown timeouts (`cmd/dashboard/main.go`)

**Tier 3 — needs a small judgment call, still low risk (5 issues):**
- `errcheck` (5): `os.RemoveAll` inside `defer` cleanup in `fixer.go`(x2),
  `rollback.go`(x1), `scanner.go`(x2). The **same files already use** a
  `//nolint:errcheck // best-effort cleanup` convention on adjacent
  `exec.Command(...).Run()` defers — reuse that exact existing convention
  rather than inventing a new pattern.

**Tier 4 — real risk, needs care (28 issues):**
- `funlen` (5): `api/fixes.go:CreateFix`, `api/rollback.go:CreateRollback`,
  `cmd/dashboard/main.go:main`, `worker/fix.go:processFix`,
  `worker/rollback.go:processRollback` — each needs splitting into helpers
  without changing behavior. Only `planner/` has unit tests today, so these
  five get build + lint-recheck + manual code review as the verification gate
  (no test suite regression to lean on).
- `gosec` G304 (3) and G204 (18) — **checked each call site directly, and
  none of these are runtime-injectable:**
  - `api/middleware.go:31` (`LoadPermissions`) and `worker/severity.go:16`
    (`LoadSeverityMap`) both take `path string`, but `cmd/dashboard/main.go`
    calls both with hardcoded literals (`"golangci/config/permissions.json"`,
    `"golangci/config/severity-mapping.json"`) — never a variable from
    outside the process.
  - `scanner.go:79`'s `jsonPath` is built from `os.MkdirTemp` — program-
    generated, not attacker-controlled.
  - `fixer.go`/`rollback.go`/`scanner.go`'s 18 `exec.CommandContext("git", ...)`
    /`exec.CommandContext("golangci-lint", ...)` calls: `fixer.go`'s own
    doc-comment states the invariant explicitly — *"`fixBranch` must already
    be caller-constructed as a `lint-fix/*` name — `Apply` does not accept or
    validate an arbitrary branch target, which is how Rule BR-2
    (`plans/10-business-rules.md`) is enforced (by construction, not a
    runtime check)."* Adding runtime regex-validation here would contradict
    that documented, deliberate design decision, not fix a real bug.
  - **Recommended fix: targeted `//nolint:gosec // G3xx: <reason>` comments**
    reusing the *exact* convention this codebase already uses in the same
    files (e.g. `fixer.go`'s existing
    `//nolint:errcheck // best-effort cleanup`), not new validation logic.
    This conflicts with a checklist line in the project's own
    `lint-fixed-plan-result.sh` output ("no `//nolint` directive") — that
    tool's checklist is guidance for its own semi-automated single-issue
    fixes, not a rule to invent speculative validation for inputs that were
    directly verified to be compile-time constants or caller-contract-
    enforced. Flagging this explicitly since it's a real judgment call.

**Deferred / needs a decision, not yet tiered (21 issues, all in
`frontiir/utils/utils.go`):** 16 `revive` + 4 `mnd` + 1 `gosec` G115. This file
was copied verbatim from `rt-external-api-v1` in the prior task specifically
so its behavior matches that project exactly. Editing it here would fork it
from the canonical copy.

## Decisions (confirmed with you)

1. **`frontiir/utils/utils.go`** — excluded from lint scope entirely (treated
   as vendored, matching the `rt-external-api-v1` original exactly). Mechanism:
   change `run_linter()`'s scan pattern from `./...` to `./golangci/...` —
   this only changes which packages get *reported on*; `frontiir/utils` still
   gets type-checked as a normal dependency (so a real bug in it would still
   break the build), it just stops being a lint-issue target. This is a
   script change only — the external reference `.golangci.yml` is never
   touched, matching the "never modify/copy the reference config" rule from
   the earlier task. Net effect: 123 issues → 100 in scope (123 − 21 in that
   file − the 2 Tier-1 formatting issues also in that file... already counted
   inside the 21).
2. **Scope for today: all four tiers** (100 issues after excluding
   `frontiir/utils.go`).
3. **gosec G304(3)/G204(18) = 21 issues** — fixed via targeted
   `//nolint:gosec // <reason>` comments (reusing the exact
   `//nolint:errcheck // best-effort cleanup` convention already in these
   files), **not** new validation code — confirmed real call sites are either
   compile-time-literal paths or governed by the documented BR-2
   caller-contract invariant in `fixer.go`, so speculative validation would
   be dead code at best and a contradiction of the documented design at worst.

## Verification plan

- After Tier 1: `go build ./...`, `go test ./...` (planner pkg has the only
  tests), `make lint` re-run — confirm exactly the 43 formatting issues are
  gone and issue count/content otherwise unchanged.
- After Tier 2: same, plus manual diff review per file (doc comments and
  constants only — no control-flow change).
- After Tier 3 (if in scope): confirm the reused `//nolint:errcheck` pattern
  matches the existing one exactly; re-lint.
- After Tier 4 (if in scope): per-function before/after diff review, full
  re-build, re-lint, and (for `funlen` targets) manual trace of the split
  functions to confirm no behavior change since no tests exist for those
  packages.
- Final: `make lint` exit code and issue count reflect exactly what was
  fixed vs. deferred; update `golangci/plans/2026-08-17-make-lint-env-config.md`
  addendum or a new dated plan file with the final tally.

