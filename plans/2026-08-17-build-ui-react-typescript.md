# Build golangci/ui/ — React + TypeScript frontend

## Context

The golangci dashboard's backend (`golangci/backend/`, its own Go module,
verified working with Swagger docs at `/swagger/index.html`) has 11 real
REST endpoints but no browser UI — `plans/16-review-checklist.md` item L3
("Build `ui/`") has been deferred at every milestone until now. The user
wants it built as a new `golangci/ui/` folder using React + TypeScript.

The 5 screens and their exact fields/actions/state model are already fully
specced in `plans/05-ui-design.md` (sourced from the original architecture
artifact) — this plan implements that spec directly, not a new design:
**Dashboard → Issue List → Plan Viewer → Fix Progress → History**, with an
explicit rule that state is server-backed (`FixPlan.status`,
`FixHistory.result`, `RollbackHistory`), not locally owned — the UI polls
and renders, it doesn't manage its own state machine.

No existing React/TypeScript project exists anywhere in this repo tree to
mirror conventions from (checked `rt-5-web/` — it's Perl/RT, not React), so
this establishes a fresh, standard, minimal-dependency setup. `node
v18.19.1`/`npm 9.2.0` are installed; `npm ping` confirms registry access
(unlike Docker Hub earlier, npm's registry is reachable from this sandbox).

## Stack decisions

- **Scaffold**: `npm create vite@latest ui -- --template react-ts`, run
  from `golangci/` → lands at `golangci/ui/`. Vite over CRA/Next — this is
  a pure SPA talking to an existing separate Go API, no SSR/routing-on-
  server need.
- **Routing**: `react-router-dom` — 5 screens map 1:1 to 5 routes.
- **Data fetching/polling**: `@tanstack/react-query` — the design doc's own
  state model (`idle → requesting → ready|error`, polled via `GET
  /plans/:id`; `queued → applying → rescanning → passed|failed` polled via
  `GET /fixes/:id`) is exactly react-query's `refetchInterval` polling
  pattern. Avoids hand-rolling polling/loading/error state.
- **Typed API client**: generate TypeScript types directly from the
  Swagger output we just built — `npx openapi-typescript
  golangci/docs/swagger.json -o ui/src/api/schema.ts`, then a thin
  `fetch` wrapper (`ui/src/api/client.ts`) typed against those schemas.
  Reuses the swagger work instead of hand-duplicating request/response
  shapes in TypeScript.
- **Auth**: no login screen exists in the spec (API-key based, not
  session-based) — a small "API Key" field in the top nav bar, persisted
  to `localStorage`, attached as `X-API-Key` on every request in
  `client.ts`. Matches `golangci/backend/api/middleware.go`'s `RoleAuth`.
- **Dev-server networking**: Vite dev proxy (`vite.config.ts`) forwards
  `/api` → `http://localhost:8081`, so the dev server needs no CORS
  changes on the Go backend. Production serving (reverse proxy or the Go
  binary serving the built static files) is **out of scope** for this pass
  — flagged below.
- **Styling**: plain CSS (one `index.css`, per-page class names) — no UI
  kit. The screens are simple tables/forms/tiles; a component library
  would be more dependency weight than this scope justifies.

## Structure

```
golangci/ui/
  src/
    api/
      schema.ts        # generated from swagger.json (openapi-typescript)
      client.ts        # fetch wrapper: base URL, X-API-Key header, typed calls
    pages/
      Dashboard.tsx     # Screen 1 — summary tiles + Scan Lint action
      IssueList.tsx     # Screen 2 — filterable/selectable table
      PlanViewer.tsx    # Screen 3 — AI fix plan preview + approve/reject
      FixProgress.tsx   # Screen 4 — 4-step stepper, polls GET /fixes/:id
      History.tsx       # Screen 5 — 3 tabs: scans/fixes/rollbacks
    components/
      SeverityBadge.tsx
      StatusStepper.tsx
      ApiKeyBar.tsx     # top nav: shows/edits the stored X-API-Key
    App.tsx             # router + layout + QueryClientProvider
    main.tsx
  vite.config.ts        # dev proxy to :8081
  package.json
```

Each `pages/*.tsx` implements exactly the fields/columns/actions from its
numbered section in `plans/05-ui-design.md` — no invented fields.

## State ownership (per plans/05-ui-design.md, implemented as-is)

| State | Owned by | Implementation |
|---|---|---|
| Issue selection | UI | `sessionStorage`-backed `Set<issue_id>` in `IssueList.tsx`; cleared on scan change |
| Plan loading | `FixPlan.status` | `useQuery` polling `GET /plans/:id` |
| Fix progress | `FixHistory.result` | `useQuery` polling `GET /fixes/:id` |
| Rollback | `RollbackHistory` | `useMutation` (`POST /rollbacks`) + polling `GET /rollbacks/:id` |

## Steps

1. Scaffold via `npm create vite@latest ui -- --template react-ts` from
   `golangci/`.
2. `npm install react-router-dom @tanstack/react-query` (+ dev dep
   `openapi-typescript`).
3. Generate `src/api/schema.ts` from `golangci/docs/swagger.json`.
4. Write `src/api/client.ts` (base URL from `import.meta.env.VITE_API_BASE`
   default `/api/v1`, `X-API-Key` header from `localStorage`).
5. Write `App.tsx` (router + `QueryClientProvider` + `ApiKeyBar`) and the 5
   page components per the table above.
6. `vite.config.ts`: dev proxy `/api` and `/swagger` → `localhost:8081`.
7. Add a `ui-dev` Makefile target (`##@ Dashboard` section) —
   `cd ui && npm run dev` — alongside the existing `dashboard-run`.

## Verification

1. `npm run build` in `ui/` — TypeScript compiles clean, no type errors
   against the generated schema.
2. With the Go dashboard already running (`make dashboard-run`), run
   `npm run dev` in `ui/`, open the dev server URL.
3. Enter a real API key (from `golangci/backend/config/permissions.json`)
   in the nav bar; confirm Dashboard tiles load (or show a clean
   empty/zero state — no seed data exists yet).
4. Confirm the dev proxy actually reaches the Go API: a request in the
   browser network tab to `/api/v1/...` should show a `200`/expected JSON,
   not a CORS error or 404.
5. Click through all 5 screens at least once each, confirming no console
   errors and that each renders its spec'd fields/columns.

## Result (2026-08-17) — all steps completed

- Scaffolded via `npm create vite@6 ui -- --template react-ts` (Node
  18.19.1 can't run the latest `create-vite`'s Node 20+-only build, so
  pinned to v6). Dropped the scaffolded ESLint tooling entirely — its
  transitive deps also require Node 20+, and lint wasn't part of this
  task's scope.
- `swag`'s Swagger 2.0 output isn't accepted by `openapi-typescript` v7
  (OpenAPI 3.x only) — added `swagger2openapi` as a conversion step
  (`npm run gen:api`: swagger2openapi → /tmp → openapi-typescript →
  `src/api/schema.ts`).
- Built `client.ts` (typed fetch wrapper, X-API-Key from localStorage),
  all 5 pages per `plans/05-ui-design.md`, 3 shared components, `App.tsx`
  (router + react-query provider), `index.css`, `vite.config.ts` (dev
  proxy to :8081).
- Added `make ui-dev` (`##@ Dashboard` section).
- Verified: `npm run build` (tsc + vite build) clean; dev server up on
  :5173; proxy confirmed reaching the real Go backend (`403`/`404` real
  RBAC responses through the proxy, not CORS/404 proxy errors); all 9
  source modules transform cleanly via Vite's dev server (no syntax/type
  errors surfaced).
- **Known gap, disclosed, not resolved**: no actual browser was used to
  visually click through the 5 screens — only module-transform success,
  typecheck, production build, and API-proxy correctness were verified.
  Frontend "works" claim is scoped to that evidence, not visual/UX
  confirmation.
- `/frontiir-review` run afterward found: MEDIUM (no request timeout in
  `client.ts`), LOW (`History.tsx` duplicates the fetch/auth logic instead
  of reusing `client.ts`'s `request()`, ignoring `VITE_API_BASE` override),
  NIT (`ApiKeyBar`'s `setTimeout` has no unmount cleanup). **User decision:
  leave all three as-is for now** (not applied this session).

## Explicitly out of scope

- Production static-file serving (reverse proxy in front of both Go API +
  built UI, or the Go binary serving `ui/dist`) — separate follow-up once
  the UI itself works in dev.
- The Swagger-route auth gap flagged in the last review (`RoleAuth` on
  `/swagger/*any`) — separate, already offered, still awaiting your call.
- Seeding demo data — screens will render real-but-empty state against the
  current DB until a scan is actually triggered through the UI itself.
