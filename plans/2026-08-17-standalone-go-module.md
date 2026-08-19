# Make `golangci/` its own Go module (drop the `rt-external-api/` import prefix)

## Context

Import paths under `golangci/backend/` currently read
`rt-external-api/golangci/backend/<pkg>` because `golangci/` is just a
subdirectory of the `rt-external-api` module (`RT/go.mod`). The user wants
imports to read `golangci/backend/<pkg>` instead. Go import paths are always
`<module>/<subpath>`, so the only way to get that is to make `golangci/` its
own Go module (own `go.mod`, `module golangci`).

The blocker found during exploration: 6 files in `backend/api/` import
`rt-external-api/frontiir/utils` — a file living *outside* `golangci/`
(`RT/frontiir/utils/utils.go`, copied in from a sibling project earlier
today). A separate module can't reach outside its own tree, so this file has
to be copied into `golangci/`'s own tree. The user has explicitly accepted
this copy.

A second, unrelated pre-existing issue surfaced during exploration:
`backend/api/router.go` was modified outside this session to add full
Swagger integration (`docs.SwaggerInfo`, `/swagger/*any` route, and
`@Summary`/`@Router`/etc. annotations across every handler in
`backend/api/*.go`), importing `rt-external-api/docs` — but `swag init` was
never run, so that package doesn't exist and `go build` is currently broken
regardless of this task. The user asked to also run `swag init` in this
session to finish that, generating `golangci/docs/`.

**Bonus side-effect**: this also resolves the "should `.env` be split"
question from two turns ago. Once the backend runs with `CWD = golangci/`
(its new module root, see step 4), `main.go`'s existing
`_ = godotenv.Load()` (which only ever checks the *current directory* for
`.env`) will correctly find the existing `golangci/.env` — which *already*
contains both `LINT_CONFIG` (lint tooling) and the backend's
`GOLANGCI_MYSQL_DB_*` / `GOLANGCI_REDIS_*` / `GOLANGCI_DASHBOARD_PORT` vars.
No new `.env` file needed after all.

## Steps

**1. Copy the vendored dependency in**
`frontiir/utils/utils.go` → `golangci/backend/frontiir/utils/utils.go`
(verbatim copy, package name stays `utils`, keep its existing "vendored from
rt-external-api-v1, kept identical" header comment).

**2. New `golangci/go.mod`** — `module golangci`, `go 1.25.0` /
`toolchain go1.26.2` (matches `RT/go.mod`), requiring the same versions
already pinned in `RT/go.mod`: `github.com/gin-gonic/gin`,
`github.com/go-redis/redis/v8`, `github.com/joho/godotenv`,
`go.uber.org/zap`, `gorm.io/driver/mysql`, `gorm.io/gorm`,
`github.com/swaggo/files`, `github.com/swaggo/gin-swagger`,
`github.com/swaggo/swag`, `git.frontiir.net/sa-dev/frontiirgopackages`. Run
`go mod tidy` inside `golangci/` (same private-module auth already
working in this environment per today's earlier `RT/go.mod` setup) to
generate `go.sum`.

**3. Rewrite import paths** in the same files touched by the earlier
backend-consolidation task, now to the shorter form:
- `rt-external-api/golangci/backend/<pkg>` → `golangci/backend/<pkg>` —
  all 12 files (`api`, `worker`, `storage`, `planner`, `scanner`, `fixer`,
  `parser` subpackages)
- `rt-external-api/frontiir/utils` → `golangci/backend/frontiir/utils` —
  the 6 files in `backend/api/`
- `rt-external-api/docs` → `golangci/docs` — `router.go` only

**4. Drop the `golangci/` prefix from hardcoded runtime path literals**,
since the binary's CWD at runtime will now be `golangci/` itself (the new
module root):
- `backend/cmd/dashboard/main.go`: `"golangci/backend/config/permissions.json"`
  → `"backend/config/permissions.json"` (and the matching log-message
  strings), same treatment for `severity-mapping.json`
- Comment-only references in `worker/severity.go`, `api/middleware.go`,
  `parser/parser.go`, and `config/permissions.json`'s `_comment` field —
  drop the prefix there too, for accuracy

**5. Run `swag init`** from within `golangci/`, after the import rewrites
are in place (so it parses the final, correct paths):
```
swag init -g backend/cmd/dashboard/main.go -d . -o docs --parseDependency --parseInternal
```
Adjust flags empirically if it doesn't pick up `backend/api/*.go`'s
annotations or resolve struct references like `storage.LintScan` used in
`@Success` tags. Verify `docs/docs.go` compiles and `router.go`'s
`docs.SwaggerInfo.BasePath` reference resolves.

**6. Update `golangci/Dockerfile`** to treat `golangci/` as its own
self-contained build context:
- New documented build command: `cd golangci && docker build -t golangci-dashboard .`
  (was `docker build -f golangci/Dockerfile -t golangci-dashboard .` from `RT/`)
- `COPY go.* ./`, `COPY . .`, the build target (`./backend/cmd/dashboard`),
  and the runtime `COPY .../backend/config/ ./backend/config/` all drop
  the `golangci/` path prefix since the context root now *is* `golangci/`
- Drop the now-unnecessary `RUN grep -v '^replace' go.mod ...` line — the
  new minimal `golangci/go.mod` has no `replace` directive
- Update the descriptive top-of-file comments to match

**7. Fix `golangci/cmd/golangci-report.sh`'s `run_linter()`** (surgical,
~6 lines) so `make lint` keeps working now that `golangci/` is a separate
module — `./golangci/...` run from `RT/` would match zero packages once
Go's nested-module exclusion applies (same reason `RT/rt-external-api-v1/`
etc. are invisible to `RT/`'s own `./...` today). Fix: resolve `$cfg` to an
absolute path (mirroring the existing `abs_json` pattern just above it),
then invoke `golangci-lint` inside a `(cd golangci && golangci-lint run
--config "$abs_cfg" --output.json.path "$abs_json" ./...)` subshell instead
of the current `./golangci/...` pattern run from the outer CWD. **Nothing
else in the script changes** — `REPORT_DIR`'s default, the `.env`/
`LINT_CONFIG` loader (`load_dotenv_lint_config`), and the `Makefile`'s
`cd .. && bash golangci/cmd/golangci-report.sh` invocation all stay exactly
as they are today.

**8. No new `.env` file** — the existing `golangci/.env` already covers
both concerns and will now be auto-loaded correctly by the backend (see
Context above). Nothing to create here.

## Explicitly out of scope (flagged, not touched)

- `lint-fixed-plan.sh` / `lint-fixed-plan-result.sh` — already hardcoded to
  a stale `--config .golangci.yml` per a decision documented earlier today;
  unaffected either way by the module split, not fixed here.
- `RT/frontiir/utils/utils.go` — becomes unused within the `rt-external-api`
  module after this change (confirmed nothing else in `RT/` imports it).
  Left in place, not deleted — deletion wasn't asked for.
- `RT/go.mod`'s now-unused `gin`/`redis`/`zap`/`gorm`/`swaggo` `require`
  entries — not cleaned up here; that module may still need them for other
  work, and tidying it is a separate decision.

## Verification

1. `cd golangci && GOROOT=/usr/local/go go build ./...` — confirms the new
   module resolves entirely on its own, zero dependency on `rt-external-api`.
2. `cd golangci && GOROOT=/usr/local/go go vet ./...` and
   `go test ./backend/planner/...`
3. `cd golangci && GOROOT=/usr/local/go go build -o /tmp/golangci-dashboard ./backend/cmd/dashboard`
   then run it briefly from `golangci/` — expect it to fail at the DB-connect
   step (no local MySQL/Redis here), *not* at the config-file-loading step —
   confirms the de-prefixed path literals resolve correctly.
4. `make lint` from `golangci/` (invocation unchanged) — confirm it still
   runs end-to-end; reported file paths shorten (e.g. `backend/api/router.go`
   instead of `golangci/backend/api/router.go`), issue count should be
   materially the same.
5. `grep -rn "rt-external-api" golangci/ --include="*.go"` — zero hits
   outside historical docs (`plans/`, out of scope per earlier precedent).
6. Confirm `docs/docs.go` (generated by `swag init`) compiles as part of
   step 1's build.
7. Not run here (flag as manual follow-up): a real `docker build` — still
   needs private-registry credentials unavailable in this environment.

## Result (2026-08-17)

All steps completed and verified:
- `golangci/backend/frontiir/utils/utils.go` copied in verbatim; `golangci/go.mod`
  (`module golangci`) created and `go mod tidy` succeeded (`go.sum` generated).
- All import paths rewritten: `rt-external-api/golangci/backend/X` →
  `golangci/backend/X` (12 files), `rt-external-api/frontiir/utils` →
  `golangci/backend/frontiir/utils` (6 files), `rt-external-api/docs` →
  `golangci/docs` (`router.go`). Zero remaining `rt-external-api` path
  references in any `.go` file (one harmless prose comment in `main.go`
  aside).
- Runtime path literals in `main.go` (and matching comments in
  `severity.go`, `middleware.go`, `parser.go`, `permissions.json`) de-prefixed
  to `backend/config/...`.
- `swag init -g backend/cmd/dashboard/main.go -d . -o docs --parseDependency
  --parseInternal` run successfully from `golangci/`; `docs/docs.go` compiles
  and `router.go`'s `docs.SwaggerInfo` reference resolves.
- `Dockerfile` reworked for a self-contained `golangci/` build context; the
  now-unnecessary `replace`-stripping `RUN` line dropped.
- `cmd/golangci-report.sh`'s `run_linter()` fixed for the module split
  (absolute `--config`, `cd golangci` subshell, `go list` output translated
  from module-qualified paths to `./`-relative ones to avoid a doubled
  `golangci/golangci/...` path, `backend/frontiir/...` excluded from the
  scan to preserve the "vendored, not lint-reported" intent).

Verified: `go build ./...`, `go vet ./...`, `go test ./backend/planner/...`
all clean from `golangci/`. Built binary run directly: fails at the
DB-connect step with credentials read from `golangci/.env` (confirms both
the path fix and the `.env` auto-load bonus from the Context section — no
separate `.env` needed). `make lint` runs end-to-end (95 issues, up from 88
at last measurement — the increase traces to the externally-added Swagger
integration in `main.go`/`router.go` that landed *between* the two
measurements, not to this session's changes: none of the new issues are in
`docs/` or `backend/frontiir/`, confirming the module split and scan
exclusion work correctly). `make shellcheck` still fails, but only on
pre-existing warnings in lines this session didn't touch (`golangci-report.sh`'s
long-standing `printf` color-variable style warnings, plus unrelated
warnings in `lint-fixed-plan-result.sh`) — confirmed zero new warnings on
the lines this session added.

Not run: real `docker build` (needs credentials unavailable here).
</content>
