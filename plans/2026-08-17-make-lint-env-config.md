# Make `make lint` use an external `.golangci.yml` via `.env`

## Context

`RT/golangci/` is a standalone lint-report tool (no `go.mod` of its own) whose
`make lint` target delegates to `golangci/cmd/golangci-report.sh`, which reads
`golangci/lint-configs.txt` to decide which `.golangci.yml`(s) to run. During
investigation (separate from this task) we found the project's own
`.golangci.yml` no longer exists at the parent project root, so the current
`make lint` silently reports "0 issues" without ever actually linting anything
— the script treats a missing config file as a non-fatal skip.

Rather than recreating a local `.golangci.yml`, the user wants `make lint` to
point at an **external, already-existing** config that belongs to a different
project (`rt-external-api-with-auto-remote-resolved/.golangci.yml`), configured
via a local `.env` file — without copying that file into this project, and
without silently falling back to old behavior if the override is missing or
broken.

The script already has a `GOLANGCI_CONFIGS` env-var override that takes
priority over `lint-configs.txt` (pre-existing, untouched by this change) —
this task adds a second, `.env`-driven override sitting below it, and fixes a
real pre-existing bug where `make lint` never fails even when lint issues are
found.

## Decisions made with the user

- If `golangci/.env` is missing entirely, or present but `LINT_CONFIG` is
  blank/unset, or `LINT_CONFIG` points at a non-existent file → **hard fail**
  with a clear error and non-zero exit. No fallback to `lint-configs.txt` in
  any of these cases (this makes `lint-configs.txt`'s automatic multi-config
  flow unreachable from `make lint` going forward, except by manually setting
  `GOLANGCI_CONFIGS` — accepted tradeoff per explicit user choice).
- The exit-code fix (`make lint` fails when lint issues are found, not just on
  execution crashes) applies script-wide — the only caller of this script is
  `make lint`, so there's no meaningful way/reason to scope it narrower.
- Ship `golangci/.env.example` (committed) alongside the gitignored `golangci/.env`.
- Logic lives in `golangci-report.sh` (bash, already shellcheck'd, already owns
  config-list resolution), not in the `Makefile` recipe (which runs under
  `/bin/sh` per Make's default `SHELL`, making tilde-expansion / `#`-containing
  parameter expansion fragile there). **The Makefile itself is not modified.**

## Files to change

### 1. New: `golangci/.env` (gitignored — contains a machine-local absolute path)
```
# golangci/.env — local override, NOT committed (see .gitignore).
# LINT_CONFIG is REQUIRED once this file exists — no silent fallback to
# golangci/lint-configs.txt. Supports absolute paths, ~ expansion, and paths
# relative to golangci/. Do not quote the value.
LINT_CONFIG=~/FrontiirProjects/RT/rt-external-api-with-auto-remote-resolved/.golangci.yml
```

### 2. New: `golangci/.env.example` (committed)
```
# Copy to .env and set LINT_CONFIG to a real golangci-lint config path.
# Once golangci/.env exists, LINT_CONFIG must be set and point to a real file —
# there is no silent fallback.
LINT_CONFIG=/absolute/path/to/some-other-project/.golangci.yml
```

### 3. `golangci/.gitignore` — append
```

# local lint-config override (may contain machine-local absolute paths)
.env
```

### 4. `golangci/cmd/golangci-report.sh`

**Doc comment (near top, config env-vars list):** mention `golangci/.env` /
`LINT_CONFIG` and the priority order (`GOLANGCI_CONFIGS` > `.env` `LINT_CONFIG`).

**New global** next to the existing `CONFIG_LIST=()` / `FAILED_CONFIGS=()`:
```bash
HAS_LINT_ISSUES=0
```

**New function**, added just before the existing `# ── 0. Load config list ──` section,
and `load_config_list()` rewritten to use it as the sole non-`GOLANGCI_CONFIGS` path
(replaces the current `lint-configs.txt`-reading `else` branch):
```bash
# ── 0a. Config override via golangci/.env (LINT_CONFIG=<path>) ────────────────
# Priority: GOLANGCI_CONFIGS (env) > golangci/.env (LINT_CONFIG).
# .env is parsed (grep/cut), never sourced — never executed as shell code.
# Hard requirement: once golangci/.env exists (or is required, see below),
# LINT_CONFIG must resolve to a real file. No silent fallback, ever.
load_dotenv_lint_config() {
  local env_file="golangci/.env"
  if [ ! -f "$env_file" ]; then
    fail "${env_file} not found — create it with LINT_CONFIG=<path-to-.golangci.yml>, or set GOLANGCI_CONFIGS to override explicitly"
    exit 2
  fi

  local raw
  raw=$(grep -E '^\s*LINT_CONFIG\s*=' "$env_file" | tail -n1 | cut -d '=' -f2- || true)
  raw=$(printf '%s' "$raw" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')

  if [ -z "$raw" ]; then
    fail "${env_file} exists but LINT_CONFIG is not set — no silent fallback"
    exit 2
  fi

  # Expand a leading ~ against $HOME (bare ~ or ~/...; ~user is not supported)
  case "$raw" in
    "~")   raw="$HOME" ;;
    "~/"*) raw="${HOME}${raw#\~}" ;;
  esac

  # Relative paths resolve relative to golangci/ (this project's own dir),
  # invariant to how make/the script is invoked.
  case "$raw" in
    /*) : ;;
    *) raw="golangci/${raw}" ;;
  esac

  if [ ! -f "$raw" ]; then
    fail "LINT_CONFIG (from ${env_file}) points to a missing file: ${raw}"
    exit 2
  fi

  CONFIG_LIST=("$raw")
  ok "Using LINT_CONFIG from ${env_file}: ${raw}"
}

# ── 0. Load config list ────────────────────────────────────────────────────────
load_config_list() {
  if [ -n "${GOLANGCI_CONFIGS:-}" ]; then
    read -ra CONFIG_LIST <<< "$GOLANGCI_CONFIGS"
  else
    load_dotenv_lint_config
  fi
  if [ "${#CONFIG_LIST[@]}" -eq 0 ]; then
    fail "No config files defined"
    exit 2
  fi
  ok "${#CONFIG_LIST[@]} config(s): ${CONFIG_LIST[*]}"
}
```
This removes the current `lint-configs.txt`-reading branch entirely (per the
accepted tradeoff above). `golangci/lint-configs.txt` itself is left on disk,
untouched, just no longer read automatically.

**Main loop — mark when issues are found** (existing `row_status` branch):
```diff
     if [ "$LINT_EXIT_CODE" -eq 0 ]; then
       row_status="✓ Clean"
     else
       row_status="⚠️ Issues"
+      HAS_LINT_ISSUES=1
     fi
```

**End of `main()` — fail the script (and thus `make lint`) when issues were found**,
after the existing `FAILED_CONFIGS` check:
```diff
   if [ "${#FAILED_CONFIGS[@]}" -gt 0 ]; then
     printf "\n${RED}%d config(s) failed: %s${NC}\n" \
       "${#FAILED_CONFIGS[@]}" "${FAILED_CONFIGS[*]}" >&2
     exit 1
   fi
+
+  if [ "$HAS_LINT_ISSUES" -eq 1 ]; then
+    printf "\n${YLW}Lint issues were found — failing (see reports above).${NC}\n" >&2
+    exit 1
+  fi
 }
```

### 5. `golangci/Makefile` — no change
`make` already propagates a non-zero recipe exit code as its own failure, so
`make lint`'s exit code automatically matches the script's once the fix above
lands.

## Known pre-existing limitations not fixed by this change (out of scope)
- `LINT_CONFIG` / `GOLANGCI_CONFIGS` paths containing spaces are unsupported
  (whitespace-splitting is pre-existing behavior).
- `~user` (someone else's home dir) expansion is not supported, only bare `~`
  and `~/...` against the current `$HOME`.
- `.env` values must not be quoted (no quote-stripping) — documented in the
  file's own comments.
- Report-directory naming (`derive_config_name`) still maps any file literally
  named `.golangci.yml` to `golangci-default` — switching between an external
  `.golangci.yml` and (hypothetically) a local one of the same name on the same
  day would land reports in the same subdirectory. Not fixed; flagging only.

## Addendum (2026-08-17, implemented after initial approval)

After the `.env`/`LINT_CONFIG` mechanism was verified working, `make lint` still
couldn't complete a real run because of pre-existing structural gaps outside
the original scope. With explicit user sign-off at each step, these were also
fixed:

1. **`RT/` had no `go.mod`.** `RT/golangci/`'s own `.go` files turned out to
   hardcode `import "rt-external-api/golangci/..."` and
   `import "rt-external-api/frontiir/utils"` — i.e. this code is a copy of the
   `golangci/` subdirectory from one of `rt-external-api-v1/` or
   `rt-external-api-with-auto-remote-resolved/` (both `module rt-external-api`,
   both containing `golangci/` + `frontiir/` as siblings), missing its
   `frontiir/utils` sibling and its own `go.mod`. Fixed by:
   - Copying `frontiir/utils/utils.go` from `rt-external-api-v1` (not
     `-with-auto-remote-resolved` — the latter's copy is missing
     `ForbiddenErr`/`BadGatewayErr`, which `golangci/api/middleware.go`
     actually calls; proven by a real `undefined: utils.ForbiddenErr` build
     error) to `RT/frontiir/utils/utils.go`.
   - Adding `RT/go.mod` (`module rt-external-api`, matching the siblings) and
     running `go mod tidy` (network + the already-cached private
     `git.frontiir.net/sa-dev/frontiirgopackages` module).
   - This also fixes `./...`'s module-root error as a side effect — nested
     sibling projects each keep their own `go.mod`, so Go automatically
     excludes them from `RT/`'s new module.
2. **An unexplained `golangci/frontiir/` directory** (`main.go` + `utils/`,
   created during this session, matching `rt-external-api-v1`'s `utils.go`
   byte-for-byte) surfaced during verification. Its origin could not be
   confirmed from the session's own command history. Left in place per user
   decision, then — once it was shown to block `make lint` entirely (its
   `main.go` imports several packages, like `frontiir/api`/`frontiir/cache`,
   that were never copied, so it can never compile) — renamed to
   `golangci/_frontiir.orphaned-bak/` (leading `_` makes Go tooling skip it)
   rather than deleted, so nothing is lost if it turns out to matter.
3. **`--output.json.path`, when relative, resolves against the `--config`
   file's directory in golangci-lint v2.12.2 — not the shell's cwd.** Once
   `--config` became an external absolute path, this silently wrote the JSON
   report inside the *other* project instead of `golangci/linter-report/`.
   Fixed in `run_linter()` by resolving `$REPORT_JSON` to an absolute path
   before passing it to `--output.json.path`.

### Additional files touched

- `RT/go.mod` (new), `RT/go.sum` (new, via `go mod tidy`)
- `RT/frontiir/utils/utils.go` (new, copied from `rt-external-api-v1`)
- `RT/golangci/_frontiir.orphaned-bak/` (renamed from the unexplained `golangci/frontiir/`)
- `RT/golangci/cmd/golangci-report.sh` — additional `run_linter()` fix for the
  `--output.json.path` resolution bug

### Final verification

`make lint` (from `RT/golangci/`) now runs the full pipeline end-to-end against
the real dashboard code: 123 real issues found (revive, gosec, golines, mnd,
gofumpt, gci, errcheck, funlen, errorlint, goimports), all 5 report formats
written to `golangci/linter-report/golangci-default/`, `Config` field in the
report correctly shows the external path, and `make lint` exits non-zero
because issues were found (requirement 5).

## Verification (to run after implementation, before declaring done)
1. `cd RT/golangci && cat .env` → confirm `LINT_CONFIG` resolves correctly.
2. `make lint` from `golangci/` → confirm it starts, logs
   `Using LINT_CONFIG from golangci/.env: <resolved external path>`, and the
   generated report's "Config" field shows the full external path (not copied
   into this project).
3. Temporarily rename `golangci/.env` away → `make lint` → expect a clear
   error to stderr and non-zero exit (`echo $?`), no report generated.
4. Restore `.env`, set `LINT_CONFIG=` blank → `make lint` → expect the same
   clear-failure behavior.
5. Restore a valid `LINT_CONFIG`, point it at a nonexistent file → `make lint`
   → expect clear "missing file" error, non-zero exit.
6. With a valid external config: if the external project currently has lint
   issues, confirm `make lint` exits non-zero (`echo $?` after) and prints the
   issues to the terminal; if clean, confirm exit 0.
7. Set `GOLANGCI_CONFIGS=/some/other/config.yml` explicitly → confirm it wins
   over `.env` (per documented priority).
8. `bash golangci/cmd/golangci-report.sh` run directly from `RT/` (bypassing
   `make` entirely) → confirm identical `.env` behavior, proving the logic
   isn't Makefile-dependent.
9. `make -C golangci shellcheck` (existing target) → confirm the edited script
   still passes shellcheck.
</content>
