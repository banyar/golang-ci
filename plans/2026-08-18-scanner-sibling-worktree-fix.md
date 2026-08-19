# Fix: scanner/fixer worktree location breaks go.work sibling modules

## Bug report

User tried to scan `rt-external-api-with-auto-remote-resolved` (its own
`.golangci.yml` rules, branch `ticket-create-sla`) through the dashboard
UI. Scan status came back `failed`.

## Root cause

Two distinct issues surfaced:

1. `ticket-create-sla` is only a remote-tracking ref
   (`origin/ticket-create-sla`) in that repo, no local branch of that name
   exists — `git worktree add --detach <dir> ticket-create-sla` fails with
   `invalid reference`. Using `origin/ticket-create-sla` as the branch
   value fixed this part on its own.

2. **The real bug**: that project uses a Go workspace (`go.work`)
   referencing sibling modules by relative path:
   ```
   use (
       .
       ../frontiirgopackages
       ../rtdatacore
   )
   ```
   `scanner.Run` (`golangci/backend/scanner/scanner.go`) and `fixer.Apply`
   (`golangci/backend/fixer/fixer.go`) both created their isolated git
   worktree via `os.MkdirTemp("", "golangci-scan-*")` — an arbitrary
   system-temp location, not a sibling of `repoPath`. From there,
   `../frontiirgopackages` and `../rtdatacore` don't resolve to anything,
   so `golangci-lint`'s package loading fails outright:
   ```
   go: cannot load module ../frontiirgopackages listed in go.work file:
       open ../frontiirgopackages/go.mod: no such file or directory
   ```
   Confirmed by reproducing `scanner.Run`'s exact steps by hand — the
   `git worktree add` itself succeeds, `golangci-lint run` is what fails,
   and only for repos with this workspace pattern (the earlier scratch
   demo-repo and `golangci` itself have no `go.work`, so this was never hit
   before).

## Fix

`golangci/backend/scanner/scanner.go` and `golangci/backend/fixer/fixer.go`:
create the git-worktree temp dir as a **sibling of `repoPath`**
(`os.MkdirTemp(filepath.Dir(repoPath), "golangci-scan-*")` /
`"golangci-fix-*"`) instead of system temp. This keeps relative sibling-
module references resolvable, since the worktree now sits at the same
directory depth as `repoPath`'s own siblings. `filepath` was already
imported in both files — no new import needed.

`golangci/backend/fixer/rollback.go`'s `Revert` was **not** changed — it
only runs `git revert`, never invokes `golangci-lint`/`go` tooling, so it
was never affected by this bug; changing it would be an unnecessary,
unrelated edit.

`fixer.go`'s `scratchDir` and `scanner.go`'s `cacheDir` (JSON report output
/ lint cache, not git worktrees) were also left untouched — no relative
sibling-module resolution happens through those paths.

## Verification

1. `go build ./...` — clean.
2. Manually reproduced `scanner.Run`'s exact steps with the new sibling
   temp-dir placement against the real repo/branch — `golangci-lint`
   loaded packages successfully this time, found 378 real issues (per that
   project's own `.golangci.yml`: revive, gci, golines, gosec, mnd,
   gofumpt, whitespace, errorlint, errcheck, funlen, gocritic, goimports,
   staticcheck, ineffassign).
3. Restarted the dashboard (`make dashboard-run`) to pick up the fix.
4. Re-ran the actual failing case through the real REST API (as `admin`):
   `POST /scans` with `repo_ref=rt-external-api-with-auto-remote-resolved`,
   `branch=origin/ticket-create-sla` → polled to `status: success` this
   time (previously `failed`).
5. Confirmed via `GET /scans/:id/issues` and a direct DB query
   (`lint_issues` grouped by linter) that all 378 issues were parsed and
   stored correctly, matching the manual reproduction's linter breakdown.
