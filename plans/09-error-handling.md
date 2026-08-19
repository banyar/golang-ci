# 09 — Error Handling

**Date:** 2026-08-03
**Status:** Draft — **net-new**; grounded in this repo's actual `utils.RestErr` convention, not the Architecture Artifact (which has no error content)
**Source:** `frontiir/utils/utils.go` (`RestErr`, `ArrayRestErr`, and their constructors)
**Dependency:** [06-api-design.md](06-api-design.md) (endpoints), [08-validation-rules.md](08-validation-rules.md) (validation failures)

> **Net-new content notice**: the Architecture Artifact mentions only `202 Accepted` for async operations — no error taxonomy, no 4xx/5xx table, no error-response schema. This file reuses the response shape this repo already has (`utils.RestErr`) instead of inventing a new one.

## Response shape (reused, not new)

```go
type RestErr struct {
    Status  int    `json:"status"`
    Message string `json:"message"`
    Code    string `json:"code,omitempty"` // machine-readable business code, e.g. "SLA_EXPIRED"
    Data    any    `json:"data,omitempty"`
}
```

`Code` is the field this feature will actually make use of — the current codebase mostly leaves it empty, but the golangci dashboard has several distinct failure modes that a generic HTTP status can't disambiguate for the UI (e.g. "locked" vs "not found" are both plausible on the same endpoint).

## Error Catalog

| Scenario | Endpoint(s) | HTTP status | `RestErr.Code` | Constructor to reuse |
|---|---|---|---|---|
| Request body missing/malformed required field | `POST /scans`, `/plans`, `/fixes`, `/rollbacks` | 422 | `VALIDATION_FAILED` | `utils.ValidationErr([]string{...})` |
| Referenced `issue_ids` / `plan_id` / `fix_history_id` not found | `POST /plans`, `/fixes`, `/rollbacks` | 404 | `NOT_FOUND` | `utils.NotFound(...)` |
| Scan/fix already running for this repo+branch (Redis lock held) | `POST /scans`, `POST /fixes` | 409 | `SCAN_LOCKED` | `utils.ConflictErr(...)` |
| Apply requested but `FixPlan.status != approved` | `POST /fixes` | 409 | `PLAN_NOT_APPROVED` | `utils.ConflictErr(...)` — see [10-business-rules.md](10-business-rules.md) Rule BR-3 |
| Apply requested with `risk=High`/`breaking_change=true` but no confirmation flag | `POST /fixes` | 422 | `CONFIRMATION_REQUIRED` | `utils.InputErr(...)` — see [08-validation-rules.md](08-validation-rules.md) |
| Fix attempt targets a branch outside `lint-fix/*` | `POST /fixes` | 403 | `BRANCH_RESTRICTED` | `utils.ForbiddenErr(...)` — **new constructor, decided below** |
| Caller's role below the endpoint's minimum role | any RBAC'd endpoint | 403 | `INSUFFICIENT_ROLE` | `utils.ForbiddenErr(...)` |
| AI Layer (Claude) call fails or times out during plan generation | `POST /plans` | 502 | `AI_LAYER_UNAVAILABLE` | `utils.BadGatewayErr(...)` — **new constructor, decided below** |
| Golangci-lint runner process fails / worktree checkout fails | `POST /scans` | 500 | `SCAN_RUNNER_FAILED` | `utils.InternalErr(...)` |
| Unhandled/unexpected error | any | 500 | *(omit — generic)* | `utils.ErrResponse(err)` — already has GORM-`ErrRecordNotFound` → 404 translation built in |

## Gaps found while grounding this in the real codebase — **Decided (default, 2026-08-04)**

`utils.go` currently has no constructor for 403 Forbidden or 502 Bad Gateway. Decision: add both, mirroring the existing `NotFound`/`ConflictErr` pattern at `frontiir/utils/utils.go:116-129`, rather than overloading an existing status with just a distinguishing `Code` — this keeps `RestErr.Status` itself meaningful at a glance, consistent with every other constructor in the file.

```go
// ForbiddenErr creates a 403 Forbidden error response
func ForbiddenErr(format string, args ...interface{}) *RestErr {
	return &RestErr{
		Status:  http.StatusForbidden,
		Message: fmt.Sprintf(format, args...),
	}
}

// BadGatewayErr creates a 502 Bad Gateway error response (e.g. AI Layer unavailable)
func BadGatewayErr(format string, args ...interface{}) *RestErr {
	return &RestErr{
		Status:  http.StatusBadGateway,
		Message: fmt.Sprintf(format, args...),
	}
}
```

This is an M1-scope task: add both constructors to `frontiir/utils/utils.go`, next to `NotFound`/`ConflictErr`.

## Logging requirements

Per this repo's existing pattern (`utils.go:131-159`'s `ErrResponse`, which already does structured `log.Printf` with caller file/line/func before returning a `RestErr`), every error path above should log: endpoint, scan/plan/fix ID involved, and the underlying error — never swallow silently. This matches `AGENTS.md`'s general rule "request handlers must return `*utils.RestErr`" — no bare error returns.
