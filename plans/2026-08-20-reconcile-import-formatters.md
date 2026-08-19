# Reconcile the 5 conflicting import formatters in .golangci.yml

## Context
Earlier verification of the UI fix flow found that files whose only issue is a bare `goimports`
finding (no accompanying gofmt/gofumpt diff) never actually resolve via `golangci-lint --fix`,
even after the GOROOT/`.golangci.yml`-commit fixes — reproduced deterministically in an isolated
worktree outside the app entirely, so it's a `.golangci.yml` config issue, not an app bug.

`formatters.enable` lists `gci`, `gofmt`, `gofumpt`, `goimports`, `golines` with **no
`formatters.settings` block at all**. `gci` and `goimports` both control import
grouping/ordering, but with different default conventions when unconfigured — each one's fix
pass can undo what the other just did, so `--fix` never converges to a state both agree is
clean (confirmed: rerunning `--fix` on a real file moved `gorm.io/gorm` to a new position, then
the very next rescan flagged the identical `goimports` issue again, unchanged).

This isn't a guess at intent — `rule-my.md` (this project's own linter reference doc) documents
the two formatters' actual designed division of labor:
- §6 `gci`: enforce 3-group import order — stdlib → external → **internal/local**, blank-line
  separated (its own worked example uses a `git.frontiir.net/...`-style internal path as the
  3rd group)
- §7 `goimports`: manage missing/unused imports **and** checks the same grouping ("gci
  grouping" — the doc names the overlap explicitly)

So the two are *meant* to agree on one convention, with the local-package group being the
distinguishing third bucket — but neither `gci.sections` nor `goimports.local-prefixes` is
actually configured to tell them what "local" means for this module (`module golangci`, so
local imports are `golangci/...`), so both fall back to incompatible defaults instead.

## Approach
Add the missing `formatters.settings` block to `.golangci.yml`, giving both tools the same
3-group convention `rule-my.md` already documents, with `golangci` as the local prefix:
```yaml
formatters:
  enable:
    - gci
    - gofmt
    - gofumpt
    - goimports
    - golines
  settings:
    gci:
      sections:
        - standard
        - default
        - localmodule
    goimports:
      local-prefixes:
        - golangci
```
`localmodule` (gci's built-in section type) auto-detects "this module's own packages" from
`go.mod`'s `module` line — no need to hardcode the `golangci/` string twice. `gofmt`/`gofumpt`/
`golines` don't touch import grouping at all, so they're unaffected and stay as-is.

## Steps
1. Add the `settings` block above to `.golangci.yml`.
2. **Verify in isolation** (no app/dashboard involved): create a detached worktree at HEAD, run
   `GOROOT=/usr/local/go golangci-lint run --fix --config .golangci.yml ./...`, then run the
   plain `run` (no `--fix`) again immediately after — confirm zero `goimports`/`gci` issues
   remain on files whose only problem was grouping (e.g. `backend/api/middleware.go`,
   `backend/api/router.go`, `backend/worker/fix.go` — the exact files that failed to resolve
   before). This isolates the fix to the config change, independent of the dashboard/GOROOT fix
   already made.
3. **Verify through the actual UI-driving API** (repeat the same scan → plan → approve → fix →
   poll cycle used earlier) against one of those previously-`"failed"` issues (e.g. the
   `backend/api/middleware.go:7` `goimports` issue) — confirm `result: "passed"` this time.
4. Decide, with you, what to do about the rest of the repo's already-mixed import state: the
   config fix only guarantees *newly touched* files converge correctly — it doesn't retroactively
   reformat the ~9 other files currently sitting in the same inconsistent state. Options: (a)
   leave them for the next real `--fix` pass that happens to touch them, (b) run one repo-wide
   `gci write`/`goimports -w` pass now to bring everything in line immediately. Not deciding this
   without you since it touches many files at once.
5. Present the `.golangci.yml` diff (and, if you pick 5b, the resulting formatting diff) for
   commit-draft review before committing, per standing convention. Two separate, already-pending
   uncommitted changes stay untouched by this: the `backend/scanner/scanner.go` /
   `backend/fixer/fixer.go` GOROOT fix (still awaiting its own commit confirmation) and the
   stray `backend/api/fixes.go.bak.2026-08-20` file.

## Verification
Step 2 is the isolated proof (config alone, no app involved). Step 3 is the end-to-end proof
(same real API flow that showed `"failed"` before now shows `"passed"` for a previously-broken
case) — together these confirm the fix at both the tool level and the app level.
