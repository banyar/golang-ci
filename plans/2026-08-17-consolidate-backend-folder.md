# Consolidate the API backend into `golangci/backend/`

## Context

`RT/golangci/` currently mixes several unrelated kinds of content at the same
top level: the actual API backend (Go packages: `api/`, `cmd/dashboard/`,
`worker/`, `storage/`, `planner/`, `scanner/`, `fixer/`, `parser/`, `config/`),
lint-report tooling (bash scripts under `cmd/`), design docs (`plans/`),
generated output (`linter-report/`), and historical markdown artifacts
(`before-fixed/`, `after-fixed/`, `test/`, loose `.md`/`.html` files). The user
wants the backend code grouped into a single folder so it's clearly separated
from everything else.

Chosen approach (per user's answers): new folder `golangci/backend/`. Move the
9 backend packages into it as-is — no internal reorganization — preserving
`cmd/dashboard/main.go`'s nesting exactly (`backend/cmd/dashboard/main.go`),
so `golangci/cmd/` (left in place) keeps only the 3 lint-tooling bash scripts
and their docs, with no name collision against `golangci/backend/cmd/`.

This is a pure move + path-reference fixup — no behavior change.

## What moves into `golangci/backend/`

```
golangci/api/          -> golangci/backend/api/
golangci/cmd/dashboard/ -> golangci/backend/cmd/dashboard/
golangci/worker/       -> golangci/backend/worker/
golangci/storage/      -> golangci/backend/storage/
golangci/planner/      -> golangci/backend/planner/
golangci/scanner/      -> golangci/backend/scanner/
golangci/fixer/        -> golangci/backend/fixer/
golangci/parser/       -> golangci/backend/parser/
golangci/config/       -> golangci/backend/config/
```

## What stays exactly where it is (not part of the backend)

`golangci/cmd/golangci-report.sh`, `golangci/cmd/lint-fixed-plan.sh`,
`golangci/cmd/lint-fixed-plan-result.sh` (+ their `.md` docs) — lint-report
tooling, unaffected. Also unaffected: `plans/`, `linter-report/`, `test/`,
`before-fixed/`, `after-fixed/`, `Makefile`, `.env`/`.env.example`,
`lint-configs.txt`, `Dockerfile` (edited in place, not moved), and the loose
top-level `.md`/`.html` files. `RT/rt-external-api-v1/golangci/` is a separate
sibling checkout — out of scope, not touched.

## Files to edit

**1. Import paths — 12 `.go` files** (confirmed by grep, this is the full
list): `cmd/dashboard/main.go`, `api/rollback.go`, `api/fixes.go`,
`api/plans.go`, `api/router.go`, `api/scans.go`, `worker/fix.go`,
`worker/rollback.go`, `worker/queue.go`, `planner/service.go`,
`planner/service_test.go`, `parser/parser.go`. Change
`rt-external-api/golangci/<pkg>` → `rt-external-api/golangci/backend/<pkg>`
for every subpackage import (`api`, `worker`, `storage`, `planner`, `scanner`,
`fixer`, `parser`). Every other `.go` file in these packages has no
cross-package import and needs no edit.

**2. Hardcoded config-path literals in `backend/cmd/dashboard/main.go`**
(lines 40, 41, 54, 57 currently) — change
`"golangci/config/permissions.json"` → `"golangci/backend/config/permissions.json"`
and `"golangci/config/severity-mapping.json"` → `"golangci/backend/config/severity-mapping.json"`
(both the path argument and the matching log-message string).

**3. `golangci/Dockerfile`** — update the two load-bearing paths:
- Build line: `golangci/cmd/dashboard/main.go` → `golangci/backend/cmd/dashboard/main.go`
- COPY line: `${builder_dir}/golangci/config/` and `./golangci/config/` →
  `${builder_dir}/golangci/backend/config/` and `./golangci/backend/config/`
Also update the two descriptive comments referencing old paths (line 1,
line ~44) so they stay accurate.

**4. Cosmetic comment-only references** (no runtime effect, updated for
accuracy): `worker/severity.go:13`, `api/middleware.go:26`,
`parser/parser.go:34`, and the `"_comment"` field inside
`config/permissions.json`.

## Confirmed to need NO changes

- `Makefile` — only references the 3 bash scripts under `golangci/cmd/`, which aren't moving.
- `cmd/golangci-report.sh` — scans `./golangci/...` recursively; unaffected by nesting depth.
- `golangci/.gitignore` — has no path entries naming these subfolders.
- `RT/go.mod` / `go.sum` — no `replace` directive or subpath-specific entries.
- `test/` — markdown only, no Go code.
- No CI config exists anywhere under `RT/` for this module.

## Verification

1. `find golangci/backend -type d` — confirm all 9 packages present under the new folder, nothing left behind at the old top-level paths.
2. `cd RT && go build ./...` — confirms every import path rewrite is correct and the binary still compiles.
3. `cd RT && go vet ./golangci/backend/...`
4. `cd RT && go test ./golangci/backend/planner/...` — the one package with existing tests; confirm still passing.
5. `cd RT/golangci && make lint` — re-run to confirm the tool still runs end-to-end against the new paths (issue count may shift slightly since reported file paths change, but the run itself must succeed, not error out on missing paths).
6. `grep -rn "golangci/config\|golangci/cmd/dashboard" golangci/backend golangci/Dockerfile` — confirm no stale un-updated literal path remains outside the intentionally-untouched files (`plans/`, `linter-report/`, historical `.md`s).
7. Flag to the user as a manual follow-up (not run here): a real `docker build` of the updated Dockerfile, since it needs private-registry credentials (`RT_DATACORE_ACCESS_TOKEN`, `GO_PACKAGES_ACCESS_TOKEN`) not available in this environment.

## Result (2026-08-17)

All steps completed and verified:
- All 9 packages present under `golangci/backend/`; nothing left at old paths.
- `go build ./...`, `go vet ./golangci/backend/...` clean.
- `go test ./golangci/backend/planner/...` passes.
- `make lint` runs end-to-end against the new paths: **88 issues**, identical
  to the last pre-move run (`linter-report/summary-2026-08-17_20-52-26.md`) —
  confirms the move introduced no new lint issues and dropped none.
- `gofmt -l backend/` clean — import-path rewrites didn't disturb formatting.
- No stale `golangci/config`, `golangci/cmd/dashboard`, or bare
  `golangci/<pkg>` references remain anywhere (checked `backend/`,
  `Dockerfile`, `cmd/GolangCiREADME.md`).
- Not run: real `docker build` (needs `RT_DATACORE_ACCESS_TOKEN` /
  `GO_PACKAGES_ACCESS_TOKEN`, unavailable in this environment) — do this
  manually before deploying from the updated Dockerfile.

Environment note (unrelated to this change, not fixed): this shell's
`GOROOT` env var points at a stale cached `go1.24.7` toolchain download
while `/usr/local/go/bin/go` is 1.26.2, breaking `go build`/`go vet`/`go test`
unless overridden per-command with `GOROOT=/usr/local/go`. Pre-existing
environment issue, out of scope for this task.
</content>
