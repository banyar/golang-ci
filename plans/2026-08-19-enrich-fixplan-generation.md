# Enrich FixPlan generation + PlanViewer to match lint-fix-plan.md's depth

## Context

`plans/16-review-checklist.md` and the user's own testing surfaced that the
dashboard's DB-backed plan (`FixPlan`, generated via
`planner.MockClient.GeneratePlan`, shown in `PlanViewer.tsx`) is far
thinner than `before-fixed/2026-08-18-lint-fix-plan-46.md` — a 9-section
document produced by a completely separate, disconnected tool
(`make lint-fixed-plan N=<n>`, spec in `cmd/lint-fixed-plan.md`).

Confirmed via evidence (gap analysis already delivered in chat):
- `grep -rn "before-fixed" golangci/backend/ golangci/ui/src/` → **zero
  hits** — no code path connects the two systems today.
- Real `fix_plans` DB rows contain one-line template output, e.g.
  `"gosec (G304) at frontiir/helpers/ticket_helper.go:26 — Potential file
  inclusion via variable"` — literally `MockClient.synthesizeText()`'s
  `fmt.Sprintf` output, nowhere near the `.md`'s code blocks/tables/caller
  analysis.
- The two systems use incompatible IDs (`FixPlan.ID` random hex vs the
  bash tool's "issue number N" = a JSON-report array index) — they cannot
  be merged by lookup; enriching must happen in the **generation path**
  itself.

**User's explicit decision**: don't build a viewer for the static `.md`
file — make the real, DB-backed, Scan→Issue→Plan workflow produce plans
this rich for *any* issue, matching the `.md`'s section depth:
Issue summary, Original code context (±10 lines, flagged marker), Root
cause, Fix strategy (code), Before→After (code), Possible side effects,
Impact analysis (callers), Recommended tests (commands), Acceptance
criteria.

## Key constraint found during investigation

`planner.IssueContext` (what `AIClient.GeneratePlan` receives) currently
carries only `FilePath, Line, Column, Linter, Rule, Message` — **no repo
path**. `FulfillPlan` (`planner/service.go:118-181`)
builds `PlanRequest` purely from already-loaded `LintIssue` rows; it never
loads the parent `LintScan` for `RepoRef`. Without a repo path, nothing
that reads real source (code context, caller search) is possible. This is
the actual reason today's mock output is a single sentence — not a
rendering gap, a **missing input** gap.

## Design

### 1. Thread `RepoRef`/`Branch` through to the AI client

- `planner.IssueContext`: add `RepoRef string` (the fields are already
  batch-validated same-scan in `RequestPlan`, so one repo path covers the
  whole request).
- `FulfillPlan`: load the `LintScan` via `plan.Issues[0].ScanID` (one extra
  `First(&scan, "id = ?", ...)` call) and populate `RepoRef` on each
  `IssueContext`.

### 2. New `planner.PlanResult` fields (mirroring the `.md`'s sections 2, 4, 5, 6, 7, 8, 9 — section 1/3 already covered by `Issues`/`RootCause`)

```go
CodeContext             string       // ±10 lines, flagged line marked "▶" (same convention as the .md)
FixStrategyCode         string       // concrete code-block suggestion (not prose)
BeforeSnippet           string       // the flagged line(s) as-is
AfterSnippet            string       // FixStrategyCode, framed as the proposed replacement
SideEffects             []string
ImpactAnalysis          ImpactInfo   // {AffectedFile, AffectedPackage, AffectedSymbol string; Callers []string}
RecommendedTestCommands []string     // e.g. ["go test ./api/...", "make test-unit TEST_FUNC=ALL"]
AcceptanceCriteria      []string
```
No new Burmese variants — matching the `.md` itself, only the narrative
Root cause/Current behavior/Recommended fix prose is bilingual there; code
blocks, file paths, commands, and checklists are English/technical in the
source document too.

### 3. `storage.FixPlan` — new columns (AutoMigrate handles it, same as every prior schema change this session)

`CodeContext, FixStrategyCode, BeforeSnippet, AfterSnippet string`
(`type:text`); `SideEffects, ImpactAnalysis, RecommendedTestCommands,
AcceptanceCriteria json.RawMessage` (`type:json`) — same pattern as the
existing `FilesImpacted`/`TestPlan` columns.

### 4. `MockClient.GeneratePlan` — new helper files in `planner/` (keeps `mock_client.go` as orchestrator, not a 400-line file)

- **`planner/codecontext.go`**: `os.ReadFile(filepath.Join(repoRef,
  filePath))`, slice ±10 lines around `Line`, prefix the flagged line with
  `▶` — directly reusing the `.md`'s own marker convention. Missing/
  unreadable file → honest fallback text (never fabricate code), same
  spirit as `scanner.go`'s error handling.
- **`planner/impact.go`**: `go/parser` + `go/ast` to find the function
  declaration enclosing `Line` in the target file (per `cmd/lint-fixed-plan.md`
  Rule 7's own "AST ဖြင့် analyze" spec) → then `grep -rn '\b<funcName>\b'
  repoRef --include=*.go`, excluding the defining file, for the caller
  list. No enclosing function found (e.g. package-level var) or grep
  finds nothing → `"No callers found"` (matches the `.md`'s own honest
  wording), not an error.
- **`planner/rule_templates.go`**: per-linter/rule deterministic templates
  for `FixStrategyCode`/`SideEffects`/`AcceptanceCriteria`, covering the
  same linters `cmd/lint-fixed-plan.md`'s own Root Cause Analysis table
  documents (errcheck, gosec, revive, gocyclo, nestif, ineffassign,
  misspell) plus a generic fallback for any other linter — mirrors the
  precedent already established in `cmd/golangci-report.sh`'s
  `JQ_REASON`/`JQ_REASON_MY` embedded per-linter explanation tables (same
  enumerated-rule-based approach, just in Go instead of jq).
- `RecommendedTestCommands`: derived, not templated — `go test
  ./<dir-of-flagged-file>/...` + fixed `make test-unit TEST_FUNC=ALL`.

### 5. API layer — no handler changes needed

`api/plans.go`'s existing `@Success 200 {object} storage.FixPlan` swagger
annotation and `c.JSON(http.StatusOK, plan)` calls already serialize the
whole struct — new columns appear automatically. Re-run `make
swagger-init` once fields are added, then `npm run gen:api` in `ui/` to
refresh `src/api/schema.ts` (established two-step pattern from the
Swagger task).

### 6. `PlanViewer.tsx` — render the new sections properly (not string-dumped)

Extend the existing `<dl className="plan-detail">` layout with real
structure per section, reusing `renderRaw`'s defensive-parsing spirit for
the `json.RawMessage` fields:
- **Issue summary**: new small table above the existing detail list,
  sourced from `plan.issues[0]` (already loaded via `Preload("Issues")` —
  no new API call) — Linter/File/Line/Column/Message.
- **Original code context**: `<pre><code>` block, monospace, flagged line
  (`▶` prefix) styled distinctly (new CSS rule, following existing
  `.plan-detail`/`.risk-*` convention in `index.css`).
- **Fix strategy / Before / After**: three `<pre><code>` blocks.
- **Possible side effects** / **Acceptance criteria**: `<ul>`/checklist
  list, not raw JSON.
- **Impact analysis**: affected file/package/symbol line + caller `<ul>`.
- **Recommended tests**: `<pre><code>` bash block (commands joined by
  newline), separate from the existing Given/When/Then `test_plan` table
  (both are kept — the `.md`'s "Recommended tests" commands and its own
  separate GWT-style acceptance flow map to two different existing/new
  fields, not a merge).
- Existing approve/reject/apply-fix actions, status handling, and the
  generating-poll behavior are **untouched**.

## Explicitly preserved (per user's Rule 5)

Scan → Select Issue → View Plan → Apply Fix → Re-run workflow, `FixPlan`
status enum, RBAC, existing routes — none of this changes. This is
additive: new columns, new generation-time logic, richer display of the
same `FixPlan` record already flowing through the same pipeline.

## Result (2026-08-19) — implemented, reviewed, fixed, verified

- All 8 new `FixPlan` fields implemented and flowing end-to-end through a
  real Scan → Plan → Approve → Fix → Rollback cycle against a disposable
  test repo (real `▶`-marked code context, real AST-based caller list,
  real derived test commands, real per-linter fix template).
- `/frontiir-review` found 2 MEDIUM issues, both fixed and re-verified:
  1. `grepCallers`/`analyzeImpact` had no timeout, unlike every other
     `exec.Command` in this codebase against a scanned repo — now
     `exec.CommandContext` + a 15s `context.WithTimeout`, `ctx` threaded
     from `GeneratePlan` down.
  2. `buildCodeContext`/`flaggedLine` joined `repoRef`+`filePath` with no
     check the result stayed inside `repoRef` — added `resolveInRepo()`
     (the same `filepath.Clean`+`HasPrefix` pattern this same code's own
     gosec-G304 template teaches), used in both.
- Caught and fixed 2 real bugs during manual verification (not part of
  the original review, found by actually running the flow):
  - `grepCallers` excluded the *entire defining file* from caller results
    instead of just the declaration's own line — so callers in the same
    file (the common case) were invisible. Fixed to exclude only the
    exact `file:line` of the `func Name(...)` header.
  - `RecommendedTestCommands` produced `go test ././...` for root-level
    files (`filepath.Dir(".")` = `"."`). Fixed with a `goTestCommand()`
    helper that special-cases `"."`.
- `go build ./...`, `go vet ./...`, `npm run build` (ui/) all clean.

## Verification

1. `go build ./...` from `golangci/` — confirms new fields/files compile.
2. `make swagger-init` then, in `ui/`, `npm run gen:api` — confirm
   `schema.ts` picks up the 8 new `FixPlan` fields with no manual client
   type edits needed.
3. `npm run build` in `ui/` — TypeScript compiles against the refreshed
   schema.
4. End-to-end against the disposable `demo-repo` (already used for the
   earlier admin walkthrough, still on disk) or the real
   `rt-external-api-with-auto-remote-resolved` repo (scanner's sibling-
   worktree fix already verified working there): Scan → select an issue →
   Create Plan → poll until `pending` → confirm via `GET /plans/:id` (or
   the UI) that `code_context` shows real ±10 lines with `▶` on the
   flagged line, `impact_analysis.callers` reflects an actual `grep`
   result (or the honest "no callers" fallback), `recommended_test_commands`
   points at the real affected package directory.
5. Re-run the existing approve → apply-fix → rollback flow once on an
   enriched plan to confirm nothing in the existing workflow broke.
6. `/frontiir-review` on the diff before declaring done (touches Go
   backend logic + DB schema + TS/React — matches this session's own
   standing rule for runtime-behavior changes).
