#!/usr/bin/env bash
# golangci-report.sh — Frontiir Go Lint Report Generator v3.0
#
# Runs golangci-lint for each config file resolved via GOLANGCI_CONFIGS or
# golangci/.env's LINT_CONFIG (see Configuration below), generating per-config
# reports under $REPORT_DIR/<config-name>/:
#
#   {TS}.json       Raw golangci-lint JSON output
#   {TS}-en.md      English Markdown report
#   {TS}-my.md      Myanmar Markdown report
#   {TS}.html       Self-contained HTML report (dark-mode aware)
#   {TS}.sarif      SARIF 2.1.0 (GitLab / GitHub / VSCode)
#
# A combined summary-{TS}.md is written to $REPORT_DIR/.
#
# Configuration — all overridable via environment variables:
#   GOLANGCI_CONFIGS  Space-separated config files (overrides golangci/.env)
#   golangci/.env     LINT_CONFIG=<path> single-config override, used when
#                     GOLANGCI_CONFIGS is unset. Required once present — no
#                     silent fallback.
#   REPORT_DIR        Output base directory       (default: golangci/linter-report)
#   KEEP_HISTORY      Run-sets to retain;         (default: 0 = delete all old)
#                     0 = delete all; N = keep last N
#   ENABLE_ENGLISH    Generate English report     (default: true)
#   ENABLE_MYANMAR    Generate Myanmar report     (default: true)
#   ENABLE_HTML       Generate HTML report        (default: true)
#   ENABLE_SARIF      Generate SARIF report       (default: true)
#
# Run from project root:
#   bash golangci/cmd/golangci-report.sh
#   make lint   (from golangci/ directory)

set -euo pipefail

# ── Configuration ──────────────────────────────────────────────────────────────
REPORT_DIR="${REPORT_DIR:-golangci/linter-report}"
KEEP_HISTORY="${KEEP_HISTORY:-0}"
ENABLE_ENGLISH="${ENABLE_ENGLISH:-true}"
ENABLE_MYANMAR="${ENABLE_MYANMAR:-true}"
ENABLE_HTML="${ENABLE_HTML:-true}"
ENABLE_SARIF="${ENABLE_SARIF:-true}"

# ── Shared globals (set by functions, read by others) ──────────────────────────
CONFIG_LIST=()
FAILED_CONFIGS=()
SUMMARY_ROWS=()
HAS_LINT_ISSUES=0

REPORT_JSON=""
REPORT_EN=""
REPORT_MY=""
REPORT_HTML=""
REPORT_SARIF=""
LINT_VERSION=""
LINT_EXIT_CODE=0
LINT_ELAPSED=0
TOTAL_ELAPSED=0
DATE_DISPLAY=""
TS=""
TOTAL=0
COUNT_CRITICAL=0
COUNT_HIGH=0
COUNT_MEDIUM=0
COUNT_LOW=0
COUNT_INFO=0
FILE_COUNT=0
PKG_COUNT=0
RISK_SCORE=0
META_GO_VERSION=""
META_OS=""
META_ARCH=""
META_HOSTNAME=""
META_USER=""
META_GIT_BRANCH=""
META_GIT_COMMIT=""
META_GIT_AUTHOR=""

# ── Formatting helpers ─────────────────────────────────────────────────────────
RED='\033[0;31m'
GRN='\033[0;32m'
YLW='\033[1;33m'
BLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

_step_n=0
_step_total=0

step() {
  _step_n=$(( _step_n + 1 ))
  printf "${DIM}[%s]${NC} ${BLD}Step %d/%d${NC} — %s\n" \
    "$(date +%H:%M:%S)" "$_step_n" "$_step_total" "$1"
}
ok()   { printf "  ${GRN}✓${NC}  %s\n" "$1"; }
warn() { printf "  ${YLW}⚠${NC}  %s\n" "$1"; }
fail() { printf "  ${RED}✗${NC}  %s\n" "$1" >&2; }

# ── 0a. Config override via golangci/.env (LINT_CONFIG=<path>) ────────────────
# Priority: GOLANGCI_CONFIGS (env) > golangci/.env (LINT_CONFIG).
# .env is parsed (grep/cut), never sourced — never executed as shell code.
# Hard requirement: once golangci/.env exists, LINT_CONFIG must resolve to a
# real file. No silent fallback, ever.
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

  # Expand a leading ~ against $HOME (bare ~ or ~/...; ~user is not supported).
  # shellcheck disable=SC2088 # intentional literal-string match, not expansion
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

# ── Helper: derive a filesystem-safe subdir name from a config path ────────────
# .golangci.yml          → golangci-default
# .golangci_v1.yml       → golangci-v1
# .golangci_security.yml → golangci-security
derive_config_name() {
  local f
  f="${1##*/}"    # basename
  f="${f#.}"      # strip leading dot
  f="${f%.yml}"   # strip .yml
  f="${f%.yaml}"  # strip .yaml
  f="${f//_/-}"   # underscores → hyphens
  if [ "$f" = "golangci" ]; then
    f="golangci-default"
  fi
  printf '%s' "$f"
}

# ── 1. Dependency check ────────────────────────────────────────────────────────
check_dependencies() {
  local missing=0
  printf "${BLD}Checking dependencies...${NC}\n"
  for cmd in golangci-lint jq awk find go; do
    if command -v "$cmd" > /dev/null 2>&1; then
      ok "$cmd"
    else
      fail "$cmd not found — install it before running this script"
      missing=$(( missing + 1 ))
    fi
  done
  if [ "$missing" -gt 0 ]; then
    printf "\n${RED}%d missing dependency/dependencies. Aborting.${NC}\n" "$missing" >&2
    exit 2
  fi
  echo
}

# ── 2. Config check ────────────────────────────────────────────────────────────
check_config() {
  local cfg="$1"
  if [ ! -f "$cfg" ]; then
    fail "Config file not found: $cfg"
    return 1
  fi
  ok "Config found: $cfg"
}

# ── 3. Report directory ────────────────────────────────────────────────────────
prepare_report_dir() {
  local dir="$1"
  mkdir -p "$dir"

  if [ "${KEEP_HISTORY}" -eq 0 ]; then
    find "$dir" -maxdepth 1 \
      \( -name "*.json" -o -name "*-en.md" -o -name "*-my.md" \
         -o -name "*.html" -o -name "*.sarif" \) \
      -delete 2>/dev/null || true
    ok "Old reports cleared (${dir})"
  else
    local existing_count
    existing_count=$(find "$dir" -maxdepth 1 \
      -name "????-??-??_??-??-??.json" | wc -l | tr -d ' ')
    local to_delete=$(( existing_count - KEEP_HISTORY + 1 ))
    if [ "$to_delete" -gt 0 ]; then
      find "$dir" -maxdepth 1 \
        -name "????-??-??_??-??-??.json" | sort | head -n "$to_delete" | \
        while IFS= read -r ts_json; do
          local ts
          ts=$(basename "${ts_json%.json}")
          find "$dir" -maxdepth 1 -name "${ts}*" -delete 2>/dev/null || true
        done
      ok "Pruned $to_delete old run(s); keeping last $(( KEEP_HISTORY - 1 )) + current"
    else
      ok "History within limit (${existing_count}/${KEEP_HISTORY})"
    fi
  fi
}

# ── 4. Metadata collection ─────────────────────────────────────────────────────
collect_metadata() {
  META_GO_VERSION=$(go version 2>/dev/null | awk '{print $3}' || echo "unknown")
  META_OS=$(uname -s 2>/dev/null || echo "unknown")
  META_ARCH=$(uname -m 2>/dev/null || echo "unknown")
  META_HOSTNAME=$(hostname 2>/dev/null || echo "unknown")
  META_USER=$(id -un 2>/dev/null || echo "unknown")

  if git rev-parse --git-dir > /dev/null 2>&1; then
    META_GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    META_GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    META_GIT_AUTHOR=$(git log -1 --format="%an" 2>/dev/null || echo "unknown")
  else
    META_GIT_BRANCH="—"
    META_GIT_COMMIT="—"
    META_GIT_AUTHOR="—"
  fi

  ok "${META_GO_VERSION} | ${META_OS}/${META_ARCH} | branch: ${META_GIT_BRANCH} @ ${META_GIT_COMMIT}"
}

# ── 5. Run linter ──────────────────────────────────────────────────────────────
run_linter() {
  local cfg="$1"
  local t0=$SECONDS
  # golangci-lint v2 resolves a relative --output.json.path against the
  # --config file's directory, not the shell's cwd — fatal once --config is
  # an external, absolute path (as with LINT_CONFIG). Always pass it absolute.
  local abs_json
  abs_json="$(cd "$(dirname "$REPORT_JSON")" && pwd)/$(basename "$REPORT_JSON")"
  # golangci/ is its own Go module (see golangci/go.mod) — golangci-lint
  # must run with it as cwd, so a relative --config also needs resolving
  # to an absolute path before we cd there.
  local abs_cfg
  case "$cfg" in
    /*) abs_cfg="$cfg" ;;
    *) abs_cfg="$(cd "$(dirname "$cfg")" && pwd)/$(basename "$cfg")" ;;
  esac
  # Scan golangci/'s own code, excluding backend/frontiir/utils/ -- a
  # verbatim vendored copy from rt-external-api-v1 (kept identical to that
  # project on purpose). It's still type-checked as a normal dependency,
  # just not lint-reported. `go list` returns module-qualified import paths
  # (e.g. "golangci/backend/fixer") -- golangci-lint needs "./"-relative
  # patterns instead, or it resolves them against cwd and doubles the
  # "golangci/" segment.
  local pkgs=()
  while IFS= read -r pkg; do
    pkgs+=("./${pkg#golangci/}")
  done < <(cd golangci && go list ./... | grep -v '^golangci/backend/frontiir/')
  (cd golangci && golangci-lint run --config "$abs_cfg" --output.json.path "$abs_json" "${pkgs[@]}") \
    && LINT_EXIT_CODE=0 \
    || LINT_EXIT_CODE=$?
  LINT_ELAPSED=$(( SECONDS - t0 ))

  if [ "$LINT_EXIT_CODE" -ge 2 ]; then
    fail "golangci-lint exited with code ${LINT_EXIT_CODE} (execution error, not lint issues)"
    return 2
  fi
  if [ ! -f "$REPORT_JSON" ]; then
    fail "JSON report not written — golangci-lint may have crashed"
    return 2
  fi
}

# ── 6. Compute statistics (runs jq once; sets shared globals) ──────────────────
compute_stats() {
  local raw
  raw=$(jq -r '
    def sev($l):
      if   $l == "gosec"                                              then "critical"
      elif ($l | IN("errcheck","gocyclo","funlen"))                   then "high"
      elif ($l | IN("nestif","bodyclose","nilerr"))                   then "medium"
      elif ($l | IN("revive","ineffassign","staticcheck","govet"))    then "low"
      else "info"
      end;
    (.Issues // []) as $all |
    {
      total:    ($all | length),
      critical: [$all[] | select(sev(.FromLinter) == "critical")] | length,
      high:     [$all[] | select(sev(.FromLinter) == "high")]     | length,
      medium:   [$all[] | select(sev(.FromLinter) == "medium")]   | length,
      low:      [$all[] | select(sev(.FromLinter) == "low")]      | length,
      info:     [$all[] | select(sev(.FromLinter) == "info")]     | length,
      files:    ([$all[] | .Pos.Filename] | unique | length),
      pkgs:     ([$all[] | .Pos.Filename | split("/")[:-1] | join("/")] | unique | length)
    } |
    "\(.total) \(.critical) \(.high) \(.medium) \(.low) \(.info) \(.files) \(.pkgs)"
  ' "$REPORT_JSON")

  read -r TOTAL COUNT_CRITICAL COUNT_HIGH COUNT_MEDIUM COUNT_LOW COUNT_INFO \
           FILE_COUNT PKG_COUNT <<< "$raw"

  RISK_SCORE=$(( COUNT_CRITICAL * 10 + COUNT_HIGH * 5 + COUNT_MEDIUM * 2 + COUNT_LOW ))
  if [ "$RISK_SCORE" -gt 100 ]; then RISK_SCORE=100; fi
}

# ── jq shared definitions (embedded in every jq call that needs them) ──────────
JQ_SEV='
  def sev($l):
    if   $l == "gosec"                                           then "critical"
    elif ($l | IN("errcheck","gocyclo","funlen"))                then "high"
    elif ($l | IN("nestif","bodyclose","nilerr"))                then "medium"
    elif ($l | IN("revive","ineffassign","staticcheck","govet")) then "low"
    else "info"
    end;
  def icon($l):
    if   sev($l) == "critical" then "🔴"
    elif sev($l) == "high"     then "⚠️"
    elif sev($l) == "medium"   then "🟡"
    elif sev($l) == "low"      then "🔵"
    else "ℹ️"
    end;
'

JQ_REASON='
  def reason($l; $t):
    if $l == "gosec" then
      if   ($t | test("G304")) then "User-controlled path in file I/O — path traversal (CWE-22)"
      elif ($t | test("G115")) then "int→int32 without range check — silent overflow"
      elif ($t | test("G301")) then "Directory permissions > 0750 — world-readable"
      elif ($t | test("G302")) then "File permissions > 0600 — group/world-readable"
      elif ($t | test("G706")) then "Unsanitised user data in log — log injection"
      elif ($t | test("G117")) then "Secret-like JSON key — may expose credentials"
      else "Security constraint violated — see gosec docs"
      end
    elif $l == "errcheck"    then "Ignored error — silent failure in production"
    elif $l == "gocyclo"     then "Complexity > 15 — hard to test and review"
    elif $l == "funlen"      then "Function > 80 lines — split into smaller functions"
    elif $l == "misspell"    then "Spelling mistake — reduces searchability"
    elif $l == "nestif"      then "Nesting too deep — use early-return pattern"
    elif $l == "ineffassign" then "Assigned value never read — dead code"
    elif $l == "revive"      then "Go style convention not followed"
    elif $l == "staticcheck" then "Statically detectable bug or deprecated API"
    elif $l == "govet"       then "Suspicious construct — likely a latent bug"
    elif $l == "bodyclose"   then "HTTP response body not closed — connection leak"
    elif $l == "nilerr"      then "nil returned on error path — context lost"
    else "Linting rule violated — see linter docs"
    end;
'

JQ_REASON_MY='
  def reason($l; $t):
    if $l == "gosec" then
      if   ($t | test("G304")) then "Path ကို validate မလုပ်ဘဲ file I/O — path traversal ဖြစ်နိုင်"
      elif ($t | test("G115")) then "int→int32 စစ်မထား — runtime overflow ဖြစ်နိုင်"
      elif ($t | test("G301")) then "Directory permission 0750 ထက်ကျယ် — world-readable"
      elif ($t | test("G302")) then "File permission 0600 ထက်ကျယ် — group/world ဖတ်နိုင်"
      elif ($t | test("G706")) then "Sanitize မလုပ်ဘဲ log ထည့် — log injection ဖြစ်နိုင်"
      elif ($t | test("G117")) then "Secret-like JSON key — credential ထိခိုက်နိုင်"
      else "Security rule ချိုးဖောက် — gosec docs ကြည့်ပါ"
      end
    elif $l == "errcheck"    then "Error ကို ignore လုပ်ထား — production တွင် silent failure"
    elif $l == "gocyclo"     then "Complexity 15 ကျော် — test ရေးရ ခက်"
    elif $l == "funlen"      then "Function 80 lines ကျော် — သေးငယ်သော function ခွဲပါ"
    elif $l == "misspell"    then "စာလုံးပေါင်းမှားနေ — searchability ကျဆင်း"
    elif $l == "nestif"      then "Nesting ရှုပ်လွန်း — early-return pattern သုံးပါ"
    elif $l == "ineffassign" then "Assign လုပ်ထားသော value ကို မသုံးဘဲ — dead code"
    elif $l == "revive"      then "Go style convention မလိုက်နာ"
    elif $l == "staticcheck" then "Static bug သို့မဟုတ် deprecated API"
    elif $l == "govet"       then "go vet suspect construct — latent bug ဖြစ်နိုင်"
    elif $l == "bodyclose"   then "HTTP body မပိတ်ထား — load မြင့်လာလျှင် connection leak"
    elif $l == "nilerr"      then "Error path မှာ nil ပြန် — error context ဆုံးရှုံး"
    else "Linting rule ချိုးဖောက် — linter docs ကြည့်ပါ"
    end;
'

# Helper: top-10 files as Markdown rows
_top_files_md() {
  jq -r '
    (.Issues // []) |
    group_by(.Pos.Filename) |
    map({f: .[0].Pos.Filename, n: length}) |
    sort_by(-.n) | .[0:10] | .[] |
    "| `\(.f)` | \(.n) |"
  ' "$REPORT_JSON"
}

# Helper: top-5 recommendations as numbered Markdown list
_recommendations_md() {
  jq -r '
    def prio($l):
      if   $l == "gosec"       then 1
      elif $l == "errcheck"    then 2
      elif $l == "gocyclo"     then 3
      elif $l == "funlen"      then 4
      elif $l == "nestif"      then 5
      elif $l == "bodyclose"   then 6
      elif $l == "nilerr"      then 7
      elif $l == "revive"      then 8
      elif $l == "ineffassign" then 9
      elif $l == "staticcheck" then 10
      elif $l == "govet"       then 11
      else 99
      end;
    (.Issues // []) |
    group_by(.FromLinter) |
    map({l: .[0].FromLinter, n: length, p: prio(.[0].FromLinter)}) |
    sort_by(.p) | .[0:5] |
    to_entries | .[] |
    "\(.key+1). Fix `\(.value.l)` — \(.value.n) issue(s)"
  ' "$REPORT_JSON"
}

# Helper: risk label (English)
_risk_label_en() {
  if   [ "$RISK_SCORE" -eq 0 ];   then echo "🟢 **Clean** — No issues found."
  elif [ "$RISK_SCORE" -le 20 ];  then echo "🔵 **Low risk** — Minor style issues only."
  elif [ "$RISK_SCORE" -le 50 ];  then echo "🟡 **Medium risk** — Some issues need attention."
  elif [ "$RISK_SCORE" -le 80 ];  then echo "⚠️ **High risk** — Significant issues found."
  else                                  echo "🔴 **Critical risk** — Security/correctness issues require immediate action."
  fi
}

# Helper: risk label (Myanmar)
_risk_label_my() {
  if   [ "$RISK_SCORE" -eq 0 ];   then echo "🟢 **Clean** — Issue မရှိပါ။"
  elif [ "$RISK_SCORE" -le 20 ];  then echo "🔵 **Risk နည်း** — Style issue ကလေးများသာ ရှိ။"
  elif [ "$RISK_SCORE" -le 50 ];  then echo "🟡 **Risk အလယ်** — Issue အချို့ပြင်ရန် လိုအပ်သည်။"
  elif [ "$RISK_SCORE" -le 80 ];  then echo "⚠️ **Risk မြင့်** — Significant issue များ တွေ့ရှိ။"
  else                                  echo "🔴 **Risk အလွန်မြင့်** — Security/correctness issue ချက်ချင်းပြင်ရမည်။"
  fi
}

# ── 7. English Markdown report ─────────────────────────────────────────────────
generate_english_report() {
  local cfg="$1"
  {
    echo "# Golangci-lint Audit Report"
    echo ""
    echo "| Field | Value |"
    echo "|---|---|"
    echo "| **Date** | ${DATE_DISPLAY} |"
    echo "| **golangci-lint** | ${LINT_VERSION} |"
    echo "| **Go version** | ${META_GO_VERSION} |"
    echo "| **OS / Arch** | ${META_OS} / ${META_ARCH} |"
    echo "| **Hostname** | ${META_HOSTNAME} |"
    echo "| **Run by** | ${META_USER} |"
    echo "| **Git branch** | ${META_GIT_BRANCH} |"
    echo "| **Commit** | ${META_GIT_COMMIT} |"
    echo "| **Author** | ${META_GIT_AUTHOR} |"
    echo "| **Config** | \`${cfg}\` |"
    echo "| **Files scanned** | ${FILE_COUNT} |"
    echo "| **Packages** | ${PKG_COUNT} |"
    echo "| **Lint duration** | ${LINT_ELAPSED}s |"
    echo ""
    echo "---"
    echo ""
    echo "## Risk Score: ${RISK_SCORE} / 100"
    echo ""
    echo "> $(_risk_label_en)"
    echo ""
    echo "| Severity | Count | Weight | Contribution |"
    echo "|---|---|---|---|"
    echo "| 🔴 Critical | ${COUNT_CRITICAL} | ×10 | $(( COUNT_CRITICAL * 10 )) |"
    echo "| ⚠️ High | ${COUNT_HIGH} | ×5 | $(( COUNT_HIGH * 5 )) |"
    echo "| 🟡 Medium | ${COUNT_MEDIUM} | ×2 | $(( COUNT_MEDIUM * 2 )) |"
    echo "| 🔵 Low | ${COUNT_LOW} | ×1 | ${COUNT_LOW} |"
    echo "| ℹ️ Info | ${COUNT_INFO} | ×0 | 0 |"
    echo "| **Total** | **${TOTAL}** | | **${RISK_SCORE}/100** |"
    echo ""
    echo "---"
    echo ""
    echo "## Summary by Linter"
    echo ""
    echo "| Linter | Count | Severity | Rule |"
    echo "|---|---|---|---|"
    jq -r "$JQ_SEV"'
      def rule($l):
        if $l == "gosec"         then "No security vulnerabilities (G104–G706)"
        elif $l == "errcheck"    then "Every error return must be checked"
        elif $l == "misspell"    then "No spelling mistakes"
        elif $l == "revive"      then "Go community style conventions"
        elif $l == "gocyclo"     then "Cyclomatic complexity ≤ 15"
        elif $l == "nestif"      then "Nesting complexity ≤ 5"
        elif $l == "ineffassign" then "Every assigned variable must be read"
        elif $l == "funlen"      then "≤ 80 lines / ≤ 50 statements per function"
        elif $l == "bodyclose"   then "Always close HTTP response bodies"
        elif $l == "nilerr"      then "Never return nil on an error path"
        elif $l == "staticcheck" then "No statically detectable bugs"
        elif $l == "govet"       then "No suspicious constructs"
        else "See linter docs"
        end;
      def slabel($l):
        if   sev($l) == "critical" then "🔴 Critical"
        elif sev($l) == "high"     then "⚠️ High"
        elif sev($l) == "medium"   then "🟡 Medium"
        elif sev($l) == "low"      then "🔵 Low"
        else "ℹ️ Info"
        end;
      (.Issues // []) |
      group_by(.FromLinter) |
      map({l: .[0].FromLinter, n: length}) | sort_by(-.n) | .[] |
      "| `\(.l)` | \(.n) | \(slabel(.l)) | \(rule(.l)) |"
    ' "$REPORT_JSON"
    echo ""
    echo "---"
    echo ""

    if [ "$TOTAL" -gt 0 ]; then
      echo "## Top Files with Issues"
      echo ""
      echo "| File | Issues |"
      echo "|---|---|"
      _top_files_md
      echo ""
      echo "---"
      echo ""
    fi

    jq -r "$JQ_SEV$JQ_REASON"'
      (.Issues // []) |
      sort_by(.FromLinter, .Pos.Filename, .Pos.Line) |
      to_entries | map(.value + {idx: (.key+1)}) |
      group_by(.FromLinter) | sort_by(-length) | .[] |
      . as $g | ($g[0].FromLinter) as $l | ($g | length) as $n |
      (
        "## \(icon($l)) \($l) — \($n) issue(s)",
        "",
        "| # | File | Line | Issue | Why it violates the rule |",
        "|---|---|---|---|---|",
        ($g | sort_by(.Pos.Filename, .Pos.Line) | .[] |
          "| **#\(.idx)** | `\(.Pos.Filename)` | \(.Pos.Line) | \(.Text) | \(reason(.FromLinter; .Text)) |"
        ),
        "",
        "---",
        ""
      )
    ' "$REPORT_JSON"

    if [ "$TOTAL" -gt 0 ]; then
      echo "## Recommended Next Actions"
      echo ""
      _recommendations_md
      echo ""
      echo "---"
      echo ""
    fi

    echo "## How to re-run"
    echo ""
    echo "\`\`\`bash"
    echo "make lint"
    echo "golangci-lint run --config ${cfg} --fix"
    echo "\`\`\`"
  } > "$REPORT_EN"
}

# ── 8. Myanmar Markdown report ─────────────────────────────────────────────────
generate_myanmar_report() {
  local cfg="$1"
  {
    echo "# Golangci-lint စစ်ဆေးမှု အစီရင်ခံစာ"
    echo ""
    echo "| အချက် | တန်ဖိုး |"
    echo "|---|---|"
    echo "| **စစ်ဆေးသည့်ရက်** | ${DATE_DISPLAY} |"
    echo "| **golangci-lint** | ${LINT_VERSION} |"
    echo "| **Go version** | ${META_GO_VERSION} |"
    echo "| **OS / Arch** | ${META_OS} / ${META_ARCH} |"
    echo "| **Hostname** | ${META_HOSTNAME} |"
    echo "| **Run လုပ်သူ** | ${META_USER} |"
    echo "| **Git branch** | ${META_GIT_BRANCH} |"
    echo "| **Commit** | ${META_GIT_COMMIT} |"
    echo "| **Author** | ${META_GIT_AUTHOR} |"
    echo "| **Config** | \`${cfg}\` |"
    echo "| **စစ်ဆေးသော files** | ${FILE_COUNT} |"
    echo "| **Packages** | ${PKG_COUNT} |"
    echo "| **Lint ကြာချိန်** | ${LINT_ELAPSED}s |"
    echo ""
    echo "---"
    echo ""
    echo "## Risk Score: ${RISK_SCORE} / 100"
    echo ""
    echo "> $(_risk_label_my)"
    echo ""
    echo "| Severity | ခုရေ | Weight | Score |"
    echo "|---|---|---|---|"
    echo "| 🔴 Critical | ${COUNT_CRITICAL} | ×10 | $(( COUNT_CRITICAL * 10 )) |"
    echo "| ⚠️ High | ${COUNT_HIGH} | ×5 | $(( COUNT_HIGH * 5 )) |"
    echo "| 🟡 Medium | ${COUNT_MEDIUM} | ×2 | $(( COUNT_MEDIUM * 2 )) |"
    echo "| 🔵 Low | ${COUNT_LOW} | ×1 | ${COUNT_LOW} |"
    echo "| ℹ️ Info | ${COUNT_INFO} | ×0 | 0 |"
    echo "| **စုစုပေါင်း** | **${TOTAL}** | | **${RISK_SCORE}/100** |"
    echo ""
    echo "---"
    echo ""
    echo "## Linter အလိုက် အကျဉ်းချုပ်"
    echo ""
    echo "| Category | ခုရေ | အရေးကြီးမှု | ချိုးဖောက်ထားသော Rule |"
    echo "|---|---|---|---|"
    jq -r "$JQ_SEV"'
      def rule($l):
        if $l == "gosec"         then "Security vulnerability မဖြစ်စေရ (G104–G706)"
        elif $l == "errcheck"    then "Error return value တိုင်းကို စစ်ရမည်"
        elif $l == "misspell"    then "Identifier/comment မှာ စာလုံးပေါင်းမှားမရ"
        elif $l == "revive"      then "Go community style convention လိုက်နာရမည်"
        elif $l == "gocyclo"     then "Function complexity ≤ 15 ဖြစ်ရမည်"
        elif $l == "nestif"      then "Nesting complexity ≤ 5 ဖြစ်ရမည်"
        elif $l == "ineffassign" then "Assign လုပ်သော variable တိုင်း ပြန်သုံးရမည်"
        elif $l == "funlen"      then "Function ≤ 80 lines / ≤ 50 statements"
        elif $l == "bodyclose"   then "HTTP response body ကို အမြဲပိတ်ရမည်"
        elif $l == "nilerr"      then "Error path မှာ nil မပြန်ရ"
        elif $l == "staticcheck" then "Static analysis bug မရှိရ"
        elif $l == "govet"       then "go vet ကို ဖြတ်ကျော်ရမည်"
        else "Linter documentation ကြည့်ပါ"
        end;
      def slabel($l):
        if   sev($l) == "critical" then "🔴 ချက်ချင်းပြင်ရ"
        elif sev($l) == "high"     then "⚠️ မကြာမီပြင်ရ"
        elif sev($l) == "medium"   then "🟡 မကြာမီပြင်ရ"
        elif sev($l) == "low"      then "🔵 နောက်မှပြင်ရ"
        else "ℹ️ Optional"
        end;
      (.Issues // []) |
      group_by(.FromLinter) |
      map({l: .[0].FromLinter, n: length}) | sort_by(-.n) | .[] |
      "| `\(.l)` | \(.n) | \(slabel(.l)) | \(rule(.l)) |"
    ' "$REPORT_JSON"
    echo ""
    echo "---"
    echo ""

    if [ "$TOTAL" -gt 0 ]; then
      echo "## Issue အများဆုံး Files"
      echo ""
      echo "| File | Issue ခုရေ |"
      echo "|---|---|"
      _top_files_md
      echo ""
      echo "---"
      echo ""
    fi

    jq -r "$JQ_SEV$JQ_REASON_MY"'
      (.Issues // []) |
      sort_by(.FromLinter, .Pos.Filename, .Pos.Line) |
      to_entries | map(.value + {idx: (.key+1)}) |
      group_by(.FromLinter) | sort_by(-length) | .[] |
      . as $g | ($g[0].FromLinter) as $l | ($g | length) as $n |
      (
        "## \(icon($l)) \($l) — \($n) ခု",
        "",
        "| # | File | Line | ပြဿနာ | ဘာကြောင့် Rule နဲ့ မညီတာ |",
        "|---|---|---|---|---|",
        ($g | sort_by(.Pos.Filename, .Pos.Line) | .[] |
          "| **#\(.idx)** | `\(.Pos.Filename)` | \(.Pos.Line) | \(.Text) | \(reason(.FromLinter; .Text)) |"
        ),
        "",
        "---",
        ""
      )
    ' "$REPORT_JSON"

    if [ "$TOTAL" -gt 0 ]; then
      echo "## ထောက်ပြချက် (Recommended Actions)"
      echo ""
      _recommendations_md
      echo ""
      echo "---"
      echo ""
    fi

    echo "## Lint ပြန်Run နည်း"
    echo ""
    echo "\`\`\`bash"
    echo "make lint"
    echo "golangci-lint run --config ${cfg} --fix"
    echo "\`\`\`"
  } > "$REPORT_MY"
}

# ── 9. HTML report ─────────────────────────────────────────────────────────────
generate_html_report() {
  local issues_html linter_rows top_files_html rec_html risk_color

  issues_html=$(jq -r '
    def sc($l):
      if   $l == "gosec"                                              then "crit"
      elif ($l | IN("errcheck","gocyclo","funlen"))                   then "high"
      elif ($l | IN("nestif","bodyclose","nilerr"))                   then "med"
      elif ($l | IN("revive","ineffassign","staticcheck","govet"))    then "low"
      else "info"
      end;
    (.Issues // []) | sort_by(.FromLinter,.Pos.Filename,.Pos.Line) |
    to_entries | .[] |
    "<tr class=\"\(sc(.value.FromLinter))\"><td>\(.key+1)</td><td><code>\(.value.Pos.Filename)</code></td><td>\(.value.Pos.Line)</td><td><code>\(.value.FromLinter)</code></td><td>\(.value.Text | gsub("<";"&lt;") | gsub(">";"&gt;"))</td></tr>"
  ' "$REPORT_JSON")

  linter_rows=$(jq -r '
    def sc($l):
      if   $l == "gosec"                                              then "crit"
      elif ($l | IN("errcheck","gocyclo","funlen"))                   then "high"
      elif ($l | IN("nestif","bodyclose","nilerr"))                   then "med"
      elif ($l | IN("revive","ineffassign","staticcheck","govet"))    then "low"
      else "info"
      end;
    def sl($l):
      if   sc($l) == "crit" then "🔴 Critical"
      elif sc($l) == "high" then "⚠️ High"
      elif sc($l) == "med"  then "🟡 Medium"
      elif sc($l) == "low"  then "🔵 Low"
      else "ℹ️ Info"
      end;
    (.Issues // []) |
    group_by(.FromLinter) | map({l:.[0].FromLinter,n:length}) | sort_by(-.n) | .[] |
    "<tr class=\"\(sc(.l))\"><td><code>\(.l)</code></td><td>\(.n)</td><td>\(sl(.l))</td></tr>"
  ' "$REPORT_JSON")

  top_files_html=$(jq -r '
    (.Issues // []) |
    group_by(.Pos.Filename) | map({f:.[0].Pos.Filename,n:length}) |
    sort_by(-.n) | .[0:10] | .[] |
    "<tr><td><code>\(.f)</code></td><td>\(.n)</td></tr>"
  ' "$REPORT_JSON")

  rec_html=$(jq -r '
    def prio($l):
      if   $l=="gosec"       then 1 elif $l=="errcheck"    then 2
      elif $l=="gocyclo"     then 3 elif $l=="funlen"      then 4
      elif $l=="nestif"      then 5 elif $l=="bodyclose"   then 6
      elif $l=="nilerr"      then 7 elif $l=="revive"      then 8
      elif $l=="ineffassign" then 9 elif $l=="staticcheck" then 10
      elif $l=="govet"       then 11 else 99 end;
    (.Issues // []) |
    group_by(.FromLinter) |
    map({l:.[0].FromLinter,n:length,p:prio(.[0].FromLinter)}) |
    sort_by(.p) | .[0:5] | to_entries | .[] |
    "<li><code>\(.value.l)</code> — \(.value.n) issue(s)</li>"
  ' "$REPORT_JSON")

  if   [ "$RISK_SCORE" -eq 0 ];   then risk_color="#22c55e"
  elif [ "$RISK_SCORE" -le 20 ];  then risk_color="#3b82f6"
  elif [ "$RISK_SCORE" -le 50 ];  then risk_color="#eab308"
  elif [ "$RISK_SCORE" -le 80 ];  then risk_color="#f97316"
  else                                  risk_color="#ef4444"
  fi

  {
    # Static section: no variable expansion needed
    cat << 'HTML_STATIC'
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
<style>
:root{--bg:#fff;--fg:#111;--muted:#6b7280;--bdr:#e5e7eb;--code:#f3f4f6;
  --crit:#fee2e2;--high:#fef9c3;--med:#fefce8;--low:#eff6ff;--info:#f0fdf4}
@media(prefers-color-scheme:dark){:root{--bg:#0f172a;--fg:#e2e8f0;--muted:#94a3b8;
  --bdr:#1e293b;--code:#1e293b;--crit:#450a0a;--high:#422006;--med:#1c1917;
  --low:#0c1a2e;--info:#052e16}}
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:system-ui,-apple-system,sans-serif;background:var(--bg);
  color:var(--fg);line-height:1.6;padding:2rem;max-width:1400px;margin:0 auto}
h1{font-size:1.75rem;margin-bottom:.35rem}
h2{font-size:1.2rem;margin:2rem 0 .75rem;border-bottom:1px solid var(--bdr);padding-bottom:.4rem}
table{width:100%;border-collapse:collapse;font-size:.875rem;margin-bottom:1.5rem}
th{text-align:left;padding:.5rem .75rem;background:var(--code);
  border-bottom:2px solid var(--bdr);font-weight:600;white-space:nowrap}
td{padding:.45rem .75rem;border-bottom:1px solid var(--bdr);
  vertical-align:top;word-break:break-word}
code{background:var(--code);padding:.1rem .35rem;border-radius:3px;
  font-size:.8rem;font-family:monospace}
.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));
  gap:.75rem;margin:1.25rem 0}
.card{background:var(--code);border-radius:8px;padding:.85rem 1rem}
.card .lbl{font-size:.7rem;color:var(--muted);text-transform:uppercase;
  letter-spacing:.06em}
.card .val{font-size:1rem;font-weight:600;margin-top:.2rem;word-break:break-all}
.risk{display:inline-flex;align-items:center;gap:.5rem;border-radius:9999px;
  padding:.4rem 1.2rem;font-size:1.6rem;font-weight:700;margin:.75rem 0;
  border:2px solid}
.ov{overflow-x:auto}
ol{padding-left:1.5rem}
ol li{margin-bottom:.4rem}
footer{margin-top:3rem;padding-top:.75rem;border-top:1px solid var(--bdr);
  font-size:.75rem;color:var(--muted)}
tr.crit{background:var(--crit)}
tr.high{background:var(--high)}
tr.med{background:var(--med)}
tr.low{background:var(--low)}
tr.info{background:var(--info)}
</style>
HTML_STATIC

    # Dynamic title
    printf '<title>Lint Report — %s</title>\n' "$DATE_DISPLAY"
    printf '</head>\n<body>\n'

    printf '<h1>Golangci-lint Audit Report</h1>\n'
    printf '<p style="color:var(--muted)">%s &bull; %s @ %s &bull; %s</p>\n' \
      "$DATE_DISPLAY" "$META_GIT_BRANCH" "$META_GIT_COMMIT" "$META_GIT_AUTHOR"

    printf '<div class="grid">\n'
    for pair in \
      "Go Version|${META_GO_VERSION}" \
      "OS / Arch|${META_OS} / ${META_ARCH}" \
      "golangci-lint|${LINT_VERSION}" \
      "Lint Duration|${LINT_ELAPSED}s" \
      "Files Scanned|${FILE_COUNT}" \
      "Packages|${PKG_COUNT}" \
      "Run by|${META_USER} @ ${META_HOSTNAME}" \
      "Total Issues|${TOTAL}"
    do
      local lbl="${pair%%|*}"
      local val="${pair#*|}"
      printf '  <div class="card"><div class="lbl">%s</div><div class="val">%s</div></div>\n' \
        "$lbl" "$val"
    done
    printf '</div>\n'

    printf '<h2>Risk Score</h2>\n'
    printf '<div class="risk" style="color:%s;border-color:%s">%s / 100</div>\n' \
      "$risk_color" "$risk_color" "$RISK_SCORE"

    printf '<div class="ov"><table>\n'
    printf '<thead><tr><th>Severity</th><th>Count</th><th>Weight</th><th>Contribution</th></tr></thead>\n'
    printf '<tbody>\n'
    printf '<tr class="crit"><td>🔴 Critical</td><td>%s</td><td>×10</td><td>%s</td></tr>\n' \
      "$COUNT_CRITICAL" "$(( COUNT_CRITICAL * 10 ))"
    printf '<tr class="high"><td>⚠️ High</td><td>%s</td><td>×5</td><td>%s</td></tr>\n' \
      "$COUNT_HIGH" "$(( COUNT_HIGH * 5 ))"
    printf '<tr class="med"><td>🟡 Medium</td><td>%s</td><td>×2</td><td>%s</td></tr>\n' \
      "$COUNT_MEDIUM" "$(( COUNT_MEDIUM * 2 ))"
    printf '<tr class="low"><td>🔵 Low</td><td>%s</td><td>×1</td><td>%s</td></tr>\n' \
      "$COUNT_LOW" "$COUNT_LOW"
    printf '<tr class="info"><td>ℹ️ Info</td><td>%s</td><td>×0</td><td>0</td></tr>\n' "$COUNT_INFO"
    printf '</tbody></table></div>\n'

    printf '<h2>Summary by Linter</h2>\n'
    printf '<div class="ov"><table>\n'
    printf '<thead><tr><th>Linter</th><th>Count</th><th>Severity</th></tr></thead>\n'
    printf '<tbody>\n%s\n</tbody></table></div>\n' "$linter_rows"

    printf '<h2>Top Files with Issues</h2>\n'
    printf '<div class="ov"><table>\n'
    printf '<thead><tr><th>File</th><th>Issues</th></tr></thead>\n'
    printf '<tbody>\n%s\n</tbody></table></div>\n' "$top_files_html"

    printf '<h2>All Issues</h2>\n'
    printf '<div class="ov"><table>\n'
    printf '<thead><tr><th>#</th><th>File</th><th>Line</th><th>Linter</th><th>Issue</th></tr></thead>\n'
    printf '<tbody>\n%s\n</tbody></table></div>\n' "$issues_html"

    printf '<h2>Recommended Next Actions</h2>\n'
    printf '<ol>\n%s\n</ol>\n' "$rec_html"

    printf '<footer>Generated by golangci-report.sh v3.0 &bull; %s &bull; golangci-lint %s</footer>\n' \
      "$DATE_DISPLAY" "$LINT_VERSION"
    printf '</body>\n</html>\n'
  } > "$REPORT_HTML"
}

# ── 10. SARIF 2.1.0 report ─────────────────────────────────────────────────────
generate_sarif_report() {
  jq --arg ver "$LINT_VERSION" '
    def sarif_level($l):
      if   $l == "gosec"                                              then "error"
      elif ($l | IN("errcheck","gocyclo","funlen","nestif","bodyclose","nilerr")) then "warning"
      elif ($l | IN("revive","ineffassign","staticcheck","govet"))    then "note"
      else "note"
      end;
    (.Issues // []) as $issues |
    {
      "$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
      "version": "2.1.0",
      "runs": [{
        "tool": {
          "driver": {
            "name": "golangci-lint",
            "version": $ver,
            "informationUri": "https://golangci-lint.run",
            "rules": (
              $issues | map(.FromLinter) | unique | map({
                "id": .,
                "name": .,
                "shortDescription": {"text": ("golangci-lint: " + .)}
              })
            )
          }
        },
        "results": [
          $issues[] | {
            "ruleId": .FromLinter,
            "level": sarif_level(.FromLinter),
            "message": {"text": .Text},
            "locations": [{
              "physicalLocation": {
                "artifactLocation": {
                  "uri": .Pos.Filename,
                  "uriBaseId": "%SRCROOT%"
                },
                "region": {
                  "startLine": .Pos.Line,
                  "startColumn": (.Pos.Column // 1)
                }
              }
            }]
          }
        ]
      }]
    }
  ' "$REPORT_JSON" > "$REPORT_SARIF"
}

# ── 11. Dashboard summary ──────────────────────────────────────────────────────
print_dashboard() {
  local status_label
  if [ "$LINT_EXIT_CODE" -eq 0 ]; then
    status_label="${GRN}✓ No issues${NC}"
  else
    status_label="${YLW}⚠  Issues found${NC}"
  fi

  printf "\n${BLD}══════════════════════════════════════════════════${NC}\n"
  printf "${BLD}  Lint Dashboard  —  %s${NC}\n" "$DATE_DISPLAY"
  printf "${BLD}══════════════════════════════════════════════════${NC}\n"
  printf "  Status       : "
  printf "$status_label\n"
  printf "  Risk Score   : ${BLD}%s / 100${NC}\n" "$RISK_SCORE"
  printf "  Duration     : %ss  (lint: %ss)\n" "$TOTAL_ELAPSED" "$LINT_ELAPSED"
  printf "  Files        : %s scanned | %s with issues\n" \
    "$FILE_COUNT" \
    "$(jq '[(.Issues // [])[] | .Pos.Filename] | unique | length' "$REPORT_JSON")"
  printf "  Packages     : %s\n" "$PKG_COUNT"
  printf "\n"
  printf "  🔴 Critical  : %s\n" "$COUNT_CRITICAL"
  printf "  ⚠️  High      : %s\n" "$COUNT_HIGH"
  printf "  🟡 Medium    : %s\n" "$COUNT_MEDIUM"
  printf "  🔵 Low       : %s\n" "$COUNT_LOW"
  printf "  ℹ️  Info      : %s\n" "$COUNT_INFO"
  printf "  ──────────────────────────────────────────────────\n"
  printf "  Total Issues : ${BLD}%s${NC}\n" "$TOTAL"
  echo
  if [ "$TOTAL" -gt 0 ]; then
    printf "  Breakdown:\n"
    jq -r '
      (.Issues // []) |
      group_by(.FromLinter) | map({l:.[0].FromLinter,n:length}) | sort_by(-.n) | .[] |
      "    * \(.l): \(.n)"
    ' "$REPORT_JSON"
    echo
  fi
  printf "${BLD}══════════════════════════════════════════════════${NC}\n\n"
  printf "  JSON   : %s\n" "$REPORT_JSON"
  [ "$ENABLE_ENGLISH" = "true" ] && printf "  EN     : %s\n" "$REPORT_EN"
  [ "$ENABLE_MYANMAR" = "true" ] && printf "  MY     : %s\n" "$REPORT_MY"
  [ "$ENABLE_HTML"    = "true" ] && printf "  HTML   : %s\n" "$REPORT_HTML"
  [ "$ENABLE_SARIF"   = "true" ] && printf "  SARIF  : %s\n" "$REPORT_SARIF"
  echo
}

# ── 12. Summary report (all configs combined) ──────────────────────────────────
generate_summary_report() {
  local summary_file="${REPORT_DIR}/summary-${TS}.md"
  local grand_total=0
  local s_cfg s_name s_total s_risk s_status

  mkdir -p "$REPORT_DIR"
  {
    printf "# Lint Run Summary\n\n"
    printf "| Field | Value |\n|---|---|\n"
    printf "| **Date** | %s |\n" "$DATE_DISPLAY"
    printf "| **golangci-lint** | %s |\n" "$LINT_VERSION"
    printf "| **Git branch** | %s @ %s |\n" "$META_GIT_BRANCH" "$META_GIT_COMMIT"
    printf "| **Configs run** | %d |\n\n" "${#CONFIG_LIST[@]}"
    printf '%s\n\n' "---"
    printf "## Results by Config\n\n"
    printf "| Config | Issues | Risk Score | Status |\n|---|---|---|---|\n"
    for row in "${SUMMARY_ROWS[@]}"; do
      IFS='|' read -r s_cfg s_name s_total s_risk s_status <<< "$row"
      printf "| \`%s\` | %s | %s/100 | %s |\n" "$s_cfg" "$s_total" "$s_risk" "$s_status"
      grand_total=$(( grand_total + s_total ))
    done
    printf "| **Total** | **%d** | | |\n\n" "$grand_total"
    printf '%s\n\n' "---"
    printf "## Report Directories\n\n"
    printf "| Config | Directory |\n|---|---|\n"
    for row in "${SUMMARY_ROWS[@]}"; do
      IFS='|' read -r s_cfg s_name s_total s_risk s_status <<< "$row"
      printf "| \`%s\` | \`%s/%s/\` |\n" "$s_cfg" "$REPORT_DIR" "$s_name"
    done
    printf "\n"
    if [ "${#FAILED_CONFIGS[@]}" -gt 0 ]; then
      printf '%s\n\n' "---"
      printf "## Failed Configs\n\n"
      for s_cfg in "${FAILED_CONFIGS[@]}"; do
        printf '%s\n' "- \`${s_cfg}\` — execution error (exit code ≥ 2)"
      done
      printf "\n"
    fi
    printf '%s\n\n' "---"
    printf "## How to re-run\n\n\`\`\`bash\nmake lint\n\`\`\`\n"
  } > "$summary_file"
  ok "$summary_file"
}

# ── Main ───────────────────────────────────────────────────────────────────────
main() {
  TS=$(date +%Y-%m-%d_%H-%M-%S)
  DATE_DISPLAY=$(date +%Y-%m-%d)

  printf "${BLD}Loading config list...${NC}\n"
  load_config_list
  echo

  check_dependencies
  LINT_VERSION=$(golangci-lint --version 2>&1 | awk '{print $4}')

  # Compute per-config step count (fixed across all iterations)
  local per_config_steps=5  # check_config + prepare_dir + metadata + run_linter + compute_stats
  [ "$ENABLE_ENGLISH" = "true" ] && per_config_steps=$(( per_config_steps + 1 ))
  [ "$ENABLE_MYANMAR" = "true" ] && per_config_steps=$(( per_config_steps + 1 ))
  [ "$ENABLE_HTML"    = "true" ] && per_config_steps=$(( per_config_steps + 1 ))
  [ "$ENABLE_SARIF"   = "true" ] && per_config_steps=$(( per_config_steps + 1 ))

  local T0=$SECONDS
  local cfg config_name config_report_dir row_status

  for cfg in "${CONFIG_LIST[@]}"; do
    config_name=$(derive_config_name "$cfg")
    config_report_dir="${REPORT_DIR}/${config_name}"

    _step_n=0
    _step_total=$per_config_steps

    printf "\n${BLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
    printf "${BLD}  Config: %s  →  %s${NC}\n" "$cfg" "$config_name"
    printf "${BLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n\n"

    REPORT_JSON="${config_report_dir}/${TS}.json"
    REPORT_EN="${config_report_dir}/${TS}-en.md"
    REPORT_MY="${config_report_dir}/${TS}-my.md"
    REPORT_HTML="${config_report_dir}/${TS}.html"
    REPORT_SARIF="${config_report_dir}/${TS}.sarif"

    step "Verifying config"
    if ! check_config "$cfg"; then
      warn "Skipping ${cfg} — config file not found (not a failure)"
      continue
    fi

    step "Preparing report directory"
    prepare_report_dir "$config_report_dir"

    step "Collecting metadata"
    collect_metadata

    step "Running golangci-lint"
    if ! run_linter "$cfg"; then
      FAILED_CONFIGS+=("$cfg")
      warn "Skipping ${cfg} — golangci-lint execution error"
      continue
    fi

    step "Computing statistics"
    compute_stats
    ok "${TOTAL} issues | ${FILE_COUNT} files | ${PKG_COUNT} pkgs | Risk: ${RISK_SCORE}/100"

    if [ "$ENABLE_ENGLISH" = "true" ]; then
      step "Generating English report"
      generate_english_report "$cfg"
      ok "$REPORT_EN"
    fi

    if [ "$ENABLE_MYANMAR" = "true" ]; then
      step "Generating Myanmar report"
      generate_myanmar_report "$cfg"
      ok "$REPORT_MY"
    fi

    if [ "$ENABLE_HTML" = "true" ]; then
      step "Generating HTML report"
      generate_html_report
      ok "$REPORT_HTML"
    fi

    if [ "$ENABLE_SARIF" = "true" ]; then
      step "Generating SARIF report"
      generate_sarif_report
      ok "$REPORT_SARIF"
    fi

    if [ "$LINT_EXIT_CODE" -eq 0 ]; then
      row_status="✓ Clean"
    else
      row_status="⚠️ Issues"
      HAS_LINT_ISSUES=1
    fi
    SUMMARY_ROWS+=("${cfg}|${config_name}|${TOTAL}|${RISK_SCORE}|${row_status}")

    TOTAL_ELAPSED=$(( SECONDS - T0 ))
    print_dashboard
  done

  printf "\n${BLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
  printf "${BLD}  Run Summary${NC}\n"
  printf "${BLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n\n"
  generate_summary_report

  if [ "${#FAILED_CONFIGS[@]}" -gt 0 ]; then
    printf "\n${RED}%d config(s) failed: %s${NC}\n" \
      "${#FAILED_CONFIGS[@]}" "${FAILED_CONFIGS[*]}" >&2
    exit 1
  fi

  if [ "$HAS_LINT_ISSUES" -eq 1 ]; then
    printf "\n${YLW}Lint issues were found — failing (see reports above).${NC}\n" >&2
    exit 1
  fi
}

main "$@"
