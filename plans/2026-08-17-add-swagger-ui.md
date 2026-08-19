# Add Swagger UI to the golangci dashboard API

## Context

The golangci dashboard's REST API (`golangci/backend/api/*.go`, verified
working end-to-end in the previous task) has zero API documentation —
`plans/16-review-checklist.md` flags this explicitly as open item **H1**
("No golangci handler has `@Router`/`@Summary`"). The user wants a Swagger
UI so the 11 real endpoints (+4 stub routes) are browsable/testable from a
browser instead of hand-typed curl commands.

This repo (`rt-external-api`, at `RT/go.mod`) already has an established,
working Swagger convention in the sibling project `rt-external-api-v1` —
reuse it exactly rather than inventing a new pattern:
- `frontiir/main.go` carries the general `@title`/`@version`/`@BasePath`/
  `@securityDefinitions.apikey` annotation block
- `frontiir/api/controllers/*.go` handlers carry per-operation annotations
  (`@Summary`, `@Tags`, `@Security`, `@Param`, `@Success`, `@Failure`,
  `@Router`)
- `frontiir/api/router.go` wires `ginSwagger.WrapHandler(swaggerFiles.Handler,
  ginSwagger.URL("/swagger/doc.json"))` at `/swagger/*any`
- `Makefile`'s `setup` target: `swag init --dir frontiir/` (scans that whole
  subtree recursively — main.go + controllers together) then builds
- Deps already used there: `github.com/swaggo/swag v1.16.4`,
  `github.com/swaggo/gin-swagger v1.6.0`, `github.com/swaggo/files v1.0.1` —
  all three already present in the local Go module cache
  (`~/go-workspace/pkg/mod/github.com/swaggo/`), so `go get`/`go mod tidy`
  resolve fully offline (this sandbox has no outbound internet, confirmed
  earlier). `swag` CLI itself is already installed at
  `~/go-workspace/bin/swag` (v1.16.4).

## Approach

Mirror the sibling's pattern 1:1, scoped to `golangci/backend/`:

1. **Dependencies** — `go get` the 3 exact pinned versions above into
   `RT/go.mod`/`go.sum` (module `rt-external-api`).

2. **General API block** — add to `golangci/backend/cmd/dashboard/main.go`
   above `func main()`:
   ```go
   // @title           Golangci Lint Dashboard API
   // @version         1.0
   // @description     Scan -> Plan -> Approve -> Fix -> Rollback REST API for golangci-lint issue triage.
   // @BasePath        /api/v1
   // @securityDefinitions.apikey  ApiKeyAuth
   // @in                          header
   // @name                        X-API-Key
   ```
   (`ApiKeyAuth`/`X-API-Key` instead of the sibling's `BearerAuth` — this API
   authenticates via `X-API-Key`, not a Bearer token; see
   `golangci/backend/api/middleware.go`'s `RoleAuth`.)

3. **Per-handler annotations** — one doc comment block per handler, same
   shape throughout:
   ```go
   // CreateScan handles POST /scans. ...(existing comment kept)
   // @Summary      Start a lint scan
   // @Description  ...
   // @Tags         Scans
   // @Security     ApiKeyAuth
   // @Accept       json
   // @Produce      json
   // @Param        body body createScanRequest true "Scan target"
   // @Success      202 {object} map[string]any
   // @Failure      400 {object} utils.RestErr
   // @Failure      409 {object} utils.RestErr
   // @Router       /scans [post]
   func CreateScan(...) gin.HandlerFunc {
   ```
   Applied to all 11 real handlers:

   | File | Handlers | Tags |
   |---|---|---|
   | `golangci/backend/api/scans.go` | `CreateScan`, `GetScan`, `GetScanIssues`, `GetIssue` | Scans / Issues |
   | `golangci/backend/api/plans.go` | `CreatePlan`, `GetPlan`, `ApprovePlan` | Plans |
   | `golangci/backend/api/fixes.go` | `CreateFix`, `GetFix` | Fixes |
   | `golangci/backend/api/rollback.go` | `CreateRollback`, `GetRollback` | Rollbacks |

   GET handlers document their real return type (`storage.LintScan`,
   `storage.FixPlan`, etc. — already read, all exported structs); POST
   handlers document the existing ad hoc `gin.H{...}` shape as
   `map[string]any` (no new response DTOs — out of scope, keeps this a
   docs-only change).

   The 4 stub routes (`history/scans`, `history/fixes`, `history/rollbacks`,
   `config` in `golangci/backend/api/router.go`) share one `stub` function —
   swag supports multiple `@Router` lines in a single doc block, so one
   combined annotation above `stub` documents all 4 as `501` with a shared
   "not implemented" description.

4. **Wire the router** — `golangci/backend/api/router.go`:
   ```go
   import (
       ...
       "rt-external-api/docs"
       swaggerFiles "github.com/swaggo/files"
       ginSwagger "github.com/swaggo/gin-swagger"
   )
   ```
   after the `/healthz` route: set `docs.SwaggerInfo.BasePath = APIPrefix`
   and add `r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
   ginSwagger.URL("/swagger/doc.json")))`.

5. **Codegen** — run `swag init --dir golangci/backend/` from `RT/`
   (module root), matching the sibling's exact invocation style, scoped to
   `golangci/backend/` instead of `frontiir/`. Default output `RT/docs/`
   (new package, `rt-external-api/docs` — same import path convention the
   sibling already uses, no naming collision since `RT/docs/` doesn't
   exist yet).

6. **Makefile** — add a `swagger-init` target (`##@ Dashboard` section,
   alongside `dashboard-run`/`dashboard-build`) running the `swag init`
   command above, and mention it in `dashboard-run`'s help text (regenerate
   after changing annotations, same manual-step convention the sibling
   uses — not wired into every build automatically).

## Verification

1. `go build ./golangci/...` — confirms annotations don't break compilation
   (swag comments are just `//` comments, zero runtime risk either way).
2. `make swagger-init` — confirm `docs/docs.go` generates with no swag
   parse errors and lists all 15 documented paths (11 real + 4 stub, shared
   block counts as 4 `@Router` entries).
3. `make dashboard-run` (reuses the already-provisioned `RT/.env` DB/Redis
   from the previous verification task).
4. `curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8081/swagger/index.html`
   → expect `200`.
5. `curl -s http://127.0.0.1:8081/swagger/doc.json | jq '.paths | keys'` →
   expect all 15 paths listed, matching `router.go`.
6. Spot-check one documented endpoint's shape in the JSON (e.g. `/scans`
   POST has the `createScanRequest` body schema and `ApiKeyAuth` security
   requirement).

## Result (2026-08-17) — all steps completed

- Deps added: `github.com/swaggo/swag@v1.16.4`, `github.com/swaggo/gin-swagger@v1.6.0`,
  `github.com/swaggo/files@v1.0.1` (`go get` + `go mod tidy`, both fully offline
  from the local module cache).
- General API block added to `golangci/backend/cmd/dashboard/main.go`.
- Per-operation annotations added to all 11 real handlers
  (`scans.go`, `plans.go`, `fixes.go`, `rollback.go`) plus one combined
  4-`@Router` block on the shared `stub` handler in `router.go`.
- `router.go` wired: `docs.SwaggerInfo.BasePath = APIPrefix`, plus
  `GET /swagger/*any` via `ginSwagger.WrapHandler`.
- **Actual working `swag init` invocation** (differs from the plan's first
  guess — `--dir golangci/backend/` alone wasn't enough since `main.go`
  lives one level deeper than the sibling's `frontiir/main.go`, and
  `utils.RestErr` / `json.RawMessage` live outside that subtree):
  ```
  swag init --dir golangci/backend/,frontiir/utils \
    --generalInfo cmd/dashboard/main.go --parseDependency
  ```
  Added as `make swagger-init` in `golangci/Makefile`'s `##@ Dashboard`
  section, using this exact command.
- Generated `RT/docs/{docs.go,swagger.json,swagger.yaml}` — 15 paths total
  (11 real + 4 stub).
- Verified end-to-end: killed a stray orphaned `go run` child process from
  an earlier test (holding port 8081), rebuilt, restarted via
  `make dashboard-run`:
  - `GET /healthz` → `200`
  - `GET /swagger/index.html` → `200`
  - `GET /swagger/doc.json` → all 15 paths present
  - `/scans` POST → `security: [{"ApiKeyAuth": []}]`, `tags: ["Scans"]`,
    `parameters: ["body"]`, `responses: [202, 400, 409]` — matches the
    handler's actual behavior exactly.

## Explicitly out of scope

- New typed response DTOs to replace the existing `gin.H{...}` ad hoc maps
  (would be a separate, larger refactor).
- Docker image changes (`golangci/Dockerfile`) — this task only touches the
  already-verified host-native `make dashboard-run` path.
- Building the actual `ui/` frontend — unrelated, separate task already
  discussed.
