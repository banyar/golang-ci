# Add an English/Burmese language toggle to the dashboard UI

## Context

The user wants the Burmese-language detail content already produced by two
standalone CLI tools — `cmd/golangci-report.sh`'s `-my.md` reports (per-rule
"why this breaks the rule" explanations, e.g. `JQ_REASON_MY`'s `reason($l;$t)`
jq function) and `cmd/lint-fixed-plan.sh`'s `before-fixed/*.md` fix-plan docs
(per-rule Root cause / Fix strategy templates) — shown instead in the real,
already-built web dashboard (`ui/`, a React+TS app with Dashboard/IssueList/
PlanViewer/FixProgress/History pages, already wired to the Go backend API).

Investigation found these are two disconnected systems today: the bash
tools' Burmese text only ever lands in static `.md` files, never in the
database the live API/UI reads from. The live `FixPlan`/`LintIssue` content
(`backend/planner/mock_client.go`) is English-only. Closing that gap means
porting the bash tools' Burmese template logic into Go so the API can serve
it, then adding a language toggle in the UI to switch between the existing
English fields and new Burmese ones. There is no i18n library or global
state container in `ui/` today (confirmed by exploration) — the toggle and
translation layer are net-new, sized to fit this app's existing patterns
rather than pulling in a framework.

**Approach: additive, non-breaking.** All new Burmese content lands in new
`_my`-suffixed fields alongside the existing English fields — nothing
existing is renamed or removed, so the existing English-only behavior is
completely unaffected if the toggle is never touched. Severity-level names
(Critical/High/etc.) stay English in both modes, matching the established
precedent in the bash tools' own Burmese reports (`-my.md` never translates
the severity word itself, only surrounding prose) — only free-text
explanations get translated. Default toggle state is Burmese, since that's
the whole point of the request; English is one click away.

## Backend changes

**1. `backend/storage/models.go`** — add 4 new fields (GORM `AutoMigrate`
picks up new columns automatically on next app start; confirmed no manual
migration needed, and nothing existing is touched):
- `LintIssue.ReasonMy string` (`gorm:"type:text" json:"reason_my"`) — the
  per-rule "why" explanation, parallel to the existing raw `Message` field.
- `FixPlan.RootCauseMy`, `CurrentBehaviorMy`, `RecommendedFixMy string`
  (same `gorm:"type:text"` pattern) — Burmese counterparts to the existing
  English fields of the same name.

**2. New `backend/parser/reason_my.go`** — a Go port of
`cmd/golangci-report.sh`'s `JQ_REASON_MY` jq function (`reason($l;$t)`,
lines 373-397), covering every branch found in that dictionary: `gosec`
sub-codes (G304/G115/G301/G302/G706/G117 + generic fallback), `errcheck`,
`gocyclo`, `funlen`, `misspell`, `nestif`, `ineffassign`, `revive`,
`staticcheck`, `govet`, `bodyclose`, `nilerr`, and the generic else-case —
same coverage, ported verbatim (not reinterpreted) so the UI's explanations
match what the CLI report already says. Exposed as
`reasonMy(linter, text string) string`.

**3. `backend/parser/parser.go`** — wire `reasonMy()` into the
`storage.LintIssue{...}` literal (currently lines 58-69 building `Rule`/
`Message`/etc. from `ri.FromLinter`/`ri.Text`): add
`ReasonMy: reasonMy(ri.FromLinter, ri.Text)`.

**4. `backend/planner/client.go`** — add `RootCauseMy`, `CurrentBehaviorMy`,
`RecommendedFixMy string` to the `PlanResult` struct alongside the existing
English fields.

**5. `backend/planner/mock_client.go`** — extend `GeneratePlan` to also
synthesize Burmese sentences for the 3 new `PlanResult` fields, using the
same input data (`first.Linter`, `first.Rule`, `first.FilePath`,
`first.Line`, `first.Message`) as the existing English templates at lines
43-60 — same deterministic-template approach, just Burmese wording.

**6. `backend/planner/service.go`** — `FulfillPlan`'s `updates` map (lines
166-176) needs the 3 new fields added (`"root_cause_my": result.RootCauseMy`
etc.) so they actually get persisted — this is the one place that isn't
automatic pass-through.

**7. Re-run `swag init`** (same command as before:
`swag init -g backend/cmd/dashboard/main.go -d . -o docs --parseDependency --parseInternal`
from `golangci/`) so `docs/swagger.json` picks up the new struct fields.
`backend/api/scans.go` and `backend/api/plans.go` need **no changes** —
confirmed every handler serializes the full struct directly (no field
filtering), so new fields appear in API responses automatically.

## Frontend changes (`ui/`)

**8. Regenerate the TS API types**: `cd ui && npm run gen:api` (reads
`../docs/swagger.json`, already the existing mechanism) — picks up
`reason_my`, `root_cause_my`, `current_behavior_my`, `recommended_fix_my`
on the generated types in `ui/src/api/schema.ts`, no manual type edits.

**9. New `ui/src/lib/lang.tsx`** — a small `LangContext`/`LangProvider`/
`useLang()` hook (React Context, since there's no state container in this
app today and a full i18n library would be disproportionate for a 2-language
finite string set). Persists to `localStorage` under `"golangci_lang"`,
mirroring the existing `getApiKey`/`setApiKey` pattern in `ui/src/api/client.ts`.
Default value: `"my"`.

**10. New `ui/src/lib/strings.ts`** — a flat `{ en: {...}, my: {...} }`
dictionary for hardcoded UI-chrome strings currently scattered across
`PlanViewer.tsx` (dt labels: "Root cause", "Current behavior", etc.,
button text), `IssueList.tsx` (table headers, filter labels), `Dashboard.tsx`,
`History.tsx` (tab labels), and `FixProgress.tsx`/`StatusStepper.tsx`
(status-enum-to-text mappings like "applying"→"re-scanning pending").

**11. New `ui/src/components/LangToggle.tsx`** — a two-state control using
`useLang()`. Wired into `ui/src/App.tsx`'s persistent `.app-header` (next to
the existing `<ApiKeyBar />`, which the exploration confirmed is already a
header-resident, always-visible component — the natural, established slot
for another global control) — `App.tsx` also gets wrapped in `<LangProvider>`.

**12. Page updates** — each swaps its rendered strings/fields based on
`useLang()`:
- `IssueList.tsx`: table header labels via `strings.ts`; add a secondary
  line under the existing `message` cell showing `issue.reason_my` when
  `lang === "my"` (falls back silently to nothing if empty, e.g. for
  historical rows predating this change) — keeps the raw linter `message`
  visible always, matching the `-my.md` report's two-column
  (raw issue + why-explanation) design rather than replacing it.
- `PlanViewer.tsx`: dt labels via `strings.ts`; dd content switches between
  `plan.root_cause`/`plan.root_cause_my` (and the other 2 pairs) based on
  `lang`, falling back to the English field if the Burmese one is empty
  (older records, or the mock client's Burmese generation failing open).
- `Dashboard.tsx`, `History.tsx`, `FixProgress.tsx`: swap hardcoded English
  chrome strings and status-enum label sets via `strings.ts`. `StatusStepper`
  itself needs no change (it already just renders whatever `label` strings
  its caller passes) — only `FixProgress.tsx`'s `stepsFor()` needs a
  language-aware label set.
- `SeverityBadge.tsx`: **no change** — severity words stay English in both
  modes (see Context).

## Verification

1. `cd golangci && GOROOT=/usr/local/go go build ./... && go vet ./... && go test ./...`
   — confirms the new struct fields, `reason_my.go`, and the wiring through
   `parser.go`/`mock_client.go`/`service.go` all compile and the existing
   `planner` tests still pass.
2. `grep -c "reason_my\|root_cause_my\|current_behavior_my\|recommended_fix_my" golangci/docs/swagger.json`
   — confirm the regenerated swagger doc includes the new fields (proves
   `swag init` picked them up).
3. `cd ui && npm run gen:api && npx tsc -b --noEmit` — confirms the
   regenerated schema types include the `_my` fields and the whole `ui/`
   TypeScript codebase still type-checks after the page edits.
4. `cd ui && npm run build` — full Vite production build succeeds.
5. **Not verifiable in this environment**: an actual live render of the
   toggle against real data — this sandbox has no running MySQL/Redis (the
   backend fails at the DB-connect step, confirmed in the earlier module-split
   task), so end-to-end AutoMigrate + a real API round-trip can't be exercised
   here. Flag to the user as a manual follow-up: run `make lint`'s prerequisites
   aside, start the backend against a real DB and click through Dashboard →
   IssueList → PlanViewer with the toggle in both positions.

## Result (2026-08-19)

All steps completed and verified:
- Backend: 4 new `_my` fields added (`storage/models.go`), `parser/reason_my.go`
  ported from `JQ_REASON_MY` (map-based, gosec sub-codes handled separately
  to keep `reasonMy` short), wired into `parser.Parse`; `mock_client.go`
  generates bilingual text via a `synthesizeText` helper; `service.go`'s
  `FulfillPlan` persists the 3 new plan fields. `swag init` re-run, confirmed
  4 new field names present in `docs/swagger.json`.
- Frontend: `ui/src/lib/lang.tsx` (Context + localStorage, default "my"),
  `ui/src/lib/strings.ts` (EN/MY dictionary), `ui/src/components/LangToggle.tsx`,
  wired into `App.tsx`'s header. All 5 pages (`Dashboard`, `IssueList`,
  `PlanViewer`, `History`, `FixProgress`) swapped to use the dictionary and,
  where applicable, the new bilingual fields (`pick()` helper with
  English fallback). `SeverityBadge` deliberately unchanged per plan.
  `npm run gen:api` regenerated `schema.ts` with the new fields.
- Verified: `go build/vet/test` clean; `npx tsc -b --noEmit` clean (one
  real type issue found and fixed: `linterLabel` needed to accept
  `string | undefined` to match the actual API type); `npm run build`
  (full Vite production build) succeeds.
- `make lint`: found and fixed 2 new issues this turn's code actually
  introduced (`reasonMy` exceeding funlen's statement limit -- refactored
  into a map + a separate `reasonGosecMy` helper; `mock_client.go`'s
  `GeneratePlan` exceeding funlen's line limit -- extracted `synthesizeText`)
  plus a golines struct-tag-realignment cascade in `models.go` caused by
  the longer new field names. Final count: **93 issues, down from the
  95-issue baseline** (pre-existing `router.go` golines debt fixed as an
  incidental side effect of the same realignment run) -- confirms this
  turn's code added zero net new lint debt.
- Not verifiable here: live render against real data (no MySQL/Redis in
  this sandbox) -- manual follow-up as noted above still stands.
</content>
