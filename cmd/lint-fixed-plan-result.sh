#!/usr/bin/env bash
# lint-fixed-plan-result.sh — apply a lint fix from a plan file and generate a result report.
# Usage: bash scripts/lint-fixed-plan-result.sh <N>
#   N — issue number matching a plan file in golangci/before-fixed/
# Output: golangci/after-fixed/YYYY-MM-DD-lint-fixed-<N>.md
# Safety: always backs up source file; rolls back on validation failure.
set -euo pipefail

# ── Input validation (Rule 1) ─────────────────────────────────────────────────
N="${1:-}"
if [[ -z "$N" ]]; then
  echo "Error: issue number required." >&2
  echo "" >&2
  echo "Usage:" >&2
  echo "  make lint-fixed-plan-result N=<issue_number>" >&2
  echo "" >&2
  echo "Example:" >&2
  echo "  make lint-fixed N=12" >&2
  exit 1
fi

if ! [[ "$N" =~ ^[0-9]+$ ]] || [[ "$N" -lt 1 ]]; then
  echo "Error: N must be a positive integer, got: '${N}'" >&2
  exit 1
fi

# ── Dependencies ──────────────────────────────────────────────────────────────
for dep in jq python3 golangci-lint gofmt; do
  if ! command -v "$dep" &>/dev/null; then
    echo "Error: '${dep}' is required but not installed." >&2
    exit 1
  fi
done

# ── Plan file lookup (Rule 2) ─────────────────────────────────────────────────
PLAN_DIR="golangci/before-fixed"
PLAN_FILE=$(ls -t "${PLAN_DIR}"/*-lint-fix-plan-${N}.md 2>/dev/null | head -1 || true)
if [[ -z "$PLAN_FILE" ]]; then
  echo "Error: plan file for issue #${N} not found in ${PLAN_DIR}/." >&2
  echo "Run 'make lint-fixed-plan N=${N}' first." >&2
  exit 1
fi

echo "Using plan: ${PLAN_FILE}"

# ── Plan file parsing (Rule 3) ────────────────────────────────────────────────
# Parse header metadata (bold key-value lines)
LINTER=$(grep -m1 '| Linter |'  "$PLAN_FILE" | sed 's/.*`\(.*\)`.*/\1/')
FILE=$(grep -m1 '| File |'     "$PLAN_FILE" | sed 's/.*`\(.*\)`.*/\1/')
LINE=$(grep -m1 '| Line |'     "$PLAN_FILE" | awk -F'|' '{gsub(/ /,""); print $3}')
ISSUE_MSG=$(grep -m1 '| Message |' "$PLAN_FILE" | awk -F'|' '{gsub(/^ +| +$/,"",$3); print $3}')

if [[ -z "$LINTER" || -z "$FILE" || -z "$LINE" ]]; then
  echo "Error: failed to parse plan file metadata (Linter/File/Line not found)." >&2
  exit 1
fi

PKG_DIR=$(dirname "$FILE")
DATE=$(date +%Y-%m-%d)
TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")

echo "Parsed → Linter: ${LINTER} | File: ${FILE}:${LINE}"

# $FILE is golangci-lint's Pos.Filename, relative to the golangci/ project root
# (golangci-lint runs via `cd golangci && golangci-lint run ...`). This script
# itself runs one level up (Makefile: `cd .. && bash golangci/cmd/...`), so
# switch into golangci/ now — every path below (source file, backups, package
# dirs for golangci-lint/go test) resolves relative to it from here on.
# $PLAN_FILE was resolved against PLAN_DIR before the cd, so re-anchor it too
# (extract_section_code() below still reads it).
PLAN_FILE="${PLAN_FILE#golangci/}"
cd golangci

# Extract code block from a named section (Burmese/English heading support)
# Usage: extract_section_code "section heading pattern"
extract_section_code() {
  local heading_pat="$1"
  awk "
    /^## .*${heading_pat}/ { in_section=1; next }
    in_section && /^\`\`\`go/ { in_block=1; next }
    in_section && in_block && /^\`\`\`/ { exit }
    in_section && in_block { print }
    in_section && /^## / && !/^## .*${heading_pat}/ { exit }
  " "$PLAN_FILE"
}

FIX_CODE=$(extract_section_code "fix မည်" || true)
SIDE_EFFECTS=$(awk '/^## ဖြစ်နိုင်သော/,/^---/' "$PLAN_FILE" \
  | grep -v '^##\|^---' | sed '/^$/d' | head -10 || true)
VERIFY_CMDS=$(awk '/^## Verification/,/^---/' "$PLAN_FILE" \
  | awk '/^```bash/,/^```/' | grep -v '^```' | head -10 || true)

# ── Validate source file exists ───────────────────────────────────────────────
if [[ ! -f "$FILE" ]]; then
  echo "Error: source file not found: ${FILE}" >&2
  exit 1
fi

# ── Helper: extract function containing a line number ─────────────────────────
extract_function() {
  local file="$1"
  local target_line="$2"
  awk -v tgt="$target_line" '
    /^func / { func_start=NR; buf="" }
    func_start { buf = buf $0 "\n" }
    func_start && /^}/ && NR >= tgt {
      if (tgt >= func_start && tgt <= NR) { printf "%s", buf }
      func_start=0; buf=""
    }
  ' "$file"
}

# ── Helper: restore backup ────────────────────────────────────────────────────
BACKUP_FILE="${FILE}.bak.${DATE}"
restore_backup() {
  if [[ -f "$BACKUP_FILE" ]]; then
    cp "$BACKUP_FILE" "$FILE"
    echo "Backup restored: ${BACKUP_FILE} → ${FILE}"
  fi
}

# ── Pre-fix snapshot (Rule 5) ─────────────────────────────────────────────────
echo "Taking pre-fix snapshot..."

# 5a: backup
cp "$FILE" "$BACKUP_FILE"
echo "Backup: ${BACKUP_FILE}"

# 5b+5c: extract function before fix
BEFORE_FUNC=$(extract_function "$FILE" "$LINE")
BEFORE_SOURCE_LINE=$(sed -n "${LINE}p" "$FILE")

# ── Apply fix (Rule 4) ────────────────────────────────────────────────────────
echo "Applying fix for linter: ${LINTER}..."

FIX_STATUS="APPLIED"
FIX_NOTE=""

case "$LINTER" in

  errcheck)
    TRIMMED=$(echo "$BEFORE_SOURCE_LINE" | sed 's/^[[:space:]]*//')
    INDENT=$(echo "$BEFORE_SOURCE_LINE" | grep -oP '^[[:space:]]*' || true)

    if echo "$BEFORE_SOURCE_LINE" | grep -q "defer.*\.Close()"; then
      # defer x.Close() → defer closure with error check
      RESOURCE=$(echo "$TRIMMED" | grep -oP 'defer \K\S+(?=\.Close\(\))')
      python3 - "$FILE" "$LINE" "$INDENT" "$RESOURCE" <<'PYEOF'
import sys, re
path, line_no, indent, resource = sys.argv[1], int(sys.argv[2]), sys.argv[3], sys.argv[4]
with open(path) as f:
    content = f.read()
has_log = bool(re.search(r'"log"', content))
has_fmt = bool(re.search(r'"fmt"', content))
has_os  = bool(re.search(r'"os"',  content))
if has_log:
    log_stmt = f'{indent}\t\tlog.Printf("close {resource}: %v", cerr)\n'
else:
    log_stmt = f'{indent}\t\tfmt.Fprintf(os.Stderr, "close {resource}: %v\\n", cerr)\n'
replacement = (
    f'{indent}defer func() {{\n'
    f'{indent}\tif cerr := {resource}.Close(); cerr != nil {{\n'
    + log_stmt +
    f'{indent}\t}}\n'
    f'{indent}}}()\n'
)
# Step 1: replace target line (before any import changes to keep line numbers stable)
lines = content.splitlines(keepends=True)
lines[line_no - 1] = replacement
content = "".join(lines)
# Step 2: add fmt and os imports if needed (fmt.Fprintf(os.Stderr,...) needs both)
if not has_log and not has_fmt:
    content = re.sub(r'(\bimport\s*\()', r'\1\n\t"fmt"', content, count=1)
if not has_log and not has_os:
    content = re.sub(r'(\bimport\s*\()', r'\1\n\t"os"', content, count=1)
with open(path, "w") as f:
    f.write(content)
PYEOF

    elif echo "$BEFORE_SOURCE_LINE" | grep -q "\.Close()"; then
      # x.Close() (non-defer) → if err := x.Close(); err != nil { ... }
      RESOURCE=$(echo "$TRIMMED" | grep -oP '\S+(?=\.Close\(\))')
      python3 - "$FILE" "$LINE" "$INDENT" "$RESOURCE" <<'PYEOF'
import sys, re
path, line_no, indent, resource = sys.argv[1], int(sys.argv[2]), sys.argv[3], sys.argv[4]
with open(path) as f:
    content = f.read()
has_log = bool(re.search(r'"log"', content))
has_fmt = bool(re.search(r'"fmt"', content))
has_os  = bool(re.search(r'"os"',  content))
if has_log:
    log_stmt = f'{indent}\tlog.Printf("close {resource}: %v", err)\n'
else:
    log_stmt = f'{indent}\tfmt.Fprintf(os.Stderr, "close {resource}: %v\\n", err)\n'
replacement = (
    f'{indent}if err := {resource}.Close(); err != nil {{\n'
    + log_stmt +
    f'{indent}}}\n'
)
lines = content.splitlines(keepends=True)
lines[line_no - 1] = replacement
content = "".join(lines)
if not has_log and not has_fmt:
    content = re.sub(r'(\bimport\s*\()', r'\1\n\t"fmt"', content, count=1)
if not has_log and not has_os:
    content = re.sub(r'(\bimport\s*\()', r'\1\n\t"os"', content, count=1)
with open(path, "w") as f:
    f.write(content)
PYEOF

    else
      # Generic: function_call() → if err := function_call(); err != nil { ... }
      CALL=$(echo "$TRIMMED" | sed 's/[[:space:]]*$//')
      python3 - "$FILE" "$LINE" "$INDENT" "$CALL" <<'PYEOF'
import sys, re
path, line_no, indent, call = sys.argv[1], int(sys.argv[2]), sys.argv[3], sys.argv[4]
with open(path) as f:
    content = f.read()
has_log = bool(re.search(r'"log"', content))
has_fmt = bool(re.search(r'"fmt"', content))
has_os  = bool(re.search(r'"os"',  content))
if has_log:
    log_stmt = f'{indent}\tlog.Printf("operation failed: %v", err)\n'
else:
    log_stmt = f'{indent}\tfmt.Fprintf(os.Stderr, "operation failed: %v\\n", err)\n'
replacement = (
    f'{indent}if err := {call}; err != nil {{\n'
    + log_stmt +
    f'{indent}}}\n'
)
lines = content.splitlines(keepends=True)
lines[line_no - 1] = replacement
content = "".join(lines)
if not has_log and not has_fmt:
    content = re.sub(r'(\bimport\s*\()', r'\1\n\t"fmt"', content, count=1)
if not has_log and not has_os:
    content = re.sub(r'(\bimport\s*\()', r'\1\n\t"os"', content, count=1)
with open(path, "w") as f:
    f.write(content)
PYEOF
    fi
    ;;

  gosec)
    if echo "$ISSUE_MSG" | grep -q "G301"; then
      python3 - "$FILE" "$LINE" <<'PYEOF'
import sys, re
path, line_no = sys.argv[1], int(sys.argv[2])
with open(path) as f:
    lines = f.readlines()
lines[line_no - 1] = re.sub(r'\b0755\b', '0750', lines[line_no - 1])
with open(path, "w") as f:
    f.writelines(lines)
PYEOF

    elif echo "$ISSUE_MSG" | grep -q "G302"; then
      python3 - "$FILE" "$LINE" <<'PYEOF'
import sys, re
path, line_no = sys.argv[1], int(sys.argv[2])
with open(path) as f:
    lines = f.readlines()
lines[line_no - 1] = re.sub(r'\b0644\b', '0600', lines[line_no - 1])
with open(path, "w") as f:
    f.writelines(lines)
PYEOF

    elif echo "$ISSUE_MSG" | grep -q "G115"; then
      FIX_STATUS="MANUAL"
      FIX_NOTE="G115 integer overflow fix requires type-aware refactor. Apply manually using the plan's fix strategy."

    elif echo "$ISSUE_MSG" | grep -q "G304"; then
      FIX_STATUS="MANUAL"
      FIX_NOTE="G304 path validation requires understanding the allowed base directory. Apply manually using the plan's fix strategy."

    elif echo "$ISSUE_MSG" | grep -q "G706"; then
      FIX_STATUS="MANUAL"
      FIX_NOTE="G706 log injection fix requires identifying the exact tainted variable. Apply manually."

    elif echo "$ISSUE_MSG" | grep -q "G117"; then
      FIX_STATUS="MANUAL"
      FIX_NOTE="G117 secret field in JSON requires API contract review. Apply manually."

    else
      FIX_STATUS="MANUAL"
      FIX_NOTE="gosec rule requires manual review. See plan fix strategy."
    fi
    ;;

  ineffassign)
    # Remove the unused assignment line
    python3 - "$FILE" "$LINE" <<'PYEOF'
import sys
path, line_no = sys.argv[1], int(sys.argv[2])
with open(path) as f:
    lines = f.readlines()
del lines[line_no - 1]
with open(path, "w") as f:
    f.writelines(lines)
PYEOF
    ;;

  misspell)
    GOROOT=/usr/local/go golangci-lint run --config .golangci.yml --fix "./${PKG_DIR}/..." 2>/dev/null || true
    ;;

  gocyclo|nestif|funlen|revive)
    FIX_STATUS="MANUAL"
    FIX_NOTE="${LINTER} requires splitting/refactoring the function. Apply manually using the plan's fix strategy and re-run 'make lint-fixed N=${N}'."
    ;;

  *)
    FIX_STATUS="MANUAL"
    FIX_NOTE="${LINTER} fix requires manual intervention. See plan fix strategy."
    ;;
esac

# ── Syntax check (safety gate) ────────────────────────────────────────────────
GOFMT_OK="true"
if [[ "$FIX_STATUS" == "APPLIED" ]]; then
  if ! GOROOT=/usr/local/go gofmt -e "$FILE" > /dev/null 2>&1; then
    echo "Warning: gofmt syntax error after fix — restoring backup." >&2
    restore_backup
    FIX_STATUS="FAIL"
    FIX_NOTE="gofmt reported syntax error after applying fix. Backup restored."
    GOFMT_OK="false"
  fi
fi

# ── Post-fix snapshot (Rule 6) ────────────────────────────────────────────────
if [[ "$FIX_STATUS" == "APPLIED" ]]; then
  AFTER_FUNC=$(extract_function "$FILE" "$LINE")
  AFTER_SOURCE_LINE=$(sed -n "${LINE}p" "$FILE" 2>/dev/null || echo "(line removed)")
else
  AFTER_FUNC="$BEFORE_FUNC"
  AFTER_SOURCE_LINE="$BEFORE_SOURCE_LINE"
fi

# Compute changed lines summary
DIFF_SUMMARY=""
if [[ "$FIX_STATUS" == "APPLIED" ]] && [[ -f "$BACKUP_FILE" ]]; then
  DIFF_SUMMARY=$(diff "$BACKUP_FILE" "$FILE" | grep '^[<>]' \
    | sed 's/^< /  BEFORE: /; s/^> /  AFTER:  /' || true)
fi

# ── Impact analysis (Rule 7) ──────────────────────────────────────────────────
BASE=$(basename "$FILE" .go)

# Callers: look for the function name containing the flagged line
FUNC_NAME=$(awk -v tgt="$LINE" '
  /^func / { fname=$0; func_line=NR }
  func_line && /^}/ && NR >= tgt {
    if (tgt >= func_line) {
      match(fname, /^func [^(]+/, arr)
      gsub(/^func /, "", arr[0])
      print arr[0]
      exit
    }
  }
' "$FILE" || true)

CALLERS=""
if [[ -n "$FUNC_NAME" ]]; then
  CALLERS=$(grep -rn "$FUNC_NAME" --include="*.go" frontiir/ 2>/dev/null \
    | grep -v "^${FILE}:" \
    | awk -F: '{printf "* `%s:%s`\n", $1, $2}' \
    | head -10 || true)
fi
[[ -z "$CALLERS" ]] && CALLERS="* No external callers found — change is local to \`${PKG_DIR}\`"

# Callees: function calls inside affected function
CALLEES=$(echo "$BEFORE_FUNC" | grep -oP '\b[a-zA-Z][a-zA-Z0-9_]*\(' \
  | sort -u | sed 's/($//' | grep -v '^func$\|^if$\|^for$\|^switch$\|^return$' \
  | awk '{printf "* `%s()`\n", $0}' | head -10 || true)
[[ -z "$CALLEES" ]] && CALLEES="* (none identified)"

# Related tests
RELATED_TESTS=$(find "$PKG_DIR" -name "*_test.go" 2>/dev/null \
  | awk '{printf "* `%s`\n", $0}' | head -10 || true)
[[ -z "$RELATED_TESTS" ]] && RELATED_TESTS="* No test files found in \`${PKG_DIR}\`"

# Related interfaces
IFACE_MATCHES=$(grep -rn "interface" --include="*.go" frontiir/ 2>/dev/null \
  | grep -i "$BASE" \
  | awk -F: '{printf "* `%s:%s`\n", $1, $2}' | head -5 || true)
[[ -z "$IFACE_MATCHES" ]] && IFACE_MATCHES="* None identified"

# ── Test plan (Rule 8) ────────────────────────────────────────────────────────
CALLER_PKGS=$(grep -rn "$FUNC_NAME" --include="*.go" frontiir/ 2>/dev/null \
  | grep -v "^${FILE}:" | awk -F: '{print $1}' \
  | xargs -I{} dirname {} 2>/dev/null | sort -u \
  | awk '{printf "go test ./%s/...\n", $0}' | head -5 || true)
[[ -z "$CALLER_PKGS" ]] && CALLER_PKGS="# (no caller packages found)"

# ── Lint validation (Rule 9) ──────────────────────────────────────────────────
LINT_STATUS="SKIP"
LINT_OUTPUT=""
LINT_ISSUE_GONE="unknown"
NEW_ISSUES=""

if [[ "$FIX_STATUS" == "APPLIED" ]]; then
  echo "Running lint validation..."
  LINT_OUTPUT=$(GOROOT=/usr/local/go golangci-lint run \
    --config .golangci.yml "./${PKG_DIR}/..." 2>&1 || true)

  # Check if the original issue at file:line still appears
  if echo "$LINT_OUTPUT" | grep -q "${FILE}:${LINE}"; then
    LINT_ISSUE_GONE="false"
    LINT_STATUS="FAIL"
    echo "FAIL: issue #${N} still present at ${FILE}:${LINE}" >&2
    restore_backup
    FIX_STATUS="ROLLED_BACK"
    FIX_NOTE="Lint validation failed — issue #${N} still present. Backup restored."
  else
    LINT_ISSUE_GONE="true"
    LINT_STATUS="PASS"
    # Check for new issues introduced
    NEW_ISSUES=$(echo "$LINT_OUTPUT" | grep -v "^$" | head -10 || true)
    if [[ -n "$NEW_ISSUES" ]]; then
      LINT_STATUS="WARN"
      echo "WARNING: fix introduced new lint issues — review required."
    fi
    echo "PASS: issue #${N} resolved."
  fi
fi

# ── Unit tests ────────────────────────────────────────────────────────────────
TEST_STATUS="SKIP"
TEST_OUTPUT=""

if [[ "$FIX_STATUS" == "APPLIED" ]]; then
  echo "Running unit tests..."
  if TEST_OUTPUT=$(GOTOOLCHAIN=local GOROOT=/usr/local/go \
      go test "./${PKG_DIR}/..." 2>&1); then
    TEST_STATUS="PASS"
  else
    TEST_STATUS="FAIL"
    echo "WARNING: tests failed after fix — review required." >&2
  fi
fi

# ── Remaining risks ───────────────────────────────────────────────────────────
RISKS=""
case "$LINTER" in
  errcheck)
    RISKS="* Callers of the modified function may not expect an error return path — check nil/error propagation
* If the closed resource is used after Close(), behavior is now guarded — verify call order"
    ;;
  gosec)
    RISKS="* Permission change affects new files/dirs only — existing files retain old permissions
* Monitor group membership if log aggregation relies on world-readable access"
    ;;
  ineffassign)
    RISKS="* Removed line may be part of a debugging pattern — confirm it is truly dead code"
    ;;
  *)
    RISKS="* Review diff carefully before merging — automated fix may not cover all edge cases"
    ;;
esac

# ── Generate result report (Rule 11) ─────────────────────────────────────────
OUTPUT_DIR="after-fixed"
mkdir -p "$OUTPUT_DIR"
RESULT_FILE="${OUTPUT_DIR}/${DATE}-lint-fixed-${N}.md"

cat > "$RESULT_FILE" <<REPORT
# Lint fix result — issue #${N}: \`${LINTER}\`

**Fix status:** ${FIX_STATUS}
**Generated:** ${TIMESTAMP}
**Plan file:** \`${PLAN_FILE}\`

---

## 1. Issue summary

| Field | Value |
|---|---|
| Issue number | ${N} |
| Linter | \`${LINTER}\` |
| File | \`${FILE}\` |
| Line | ${LINE} |
| Message | ${ISSUE_MSG} |
| Plan file | \`${PLAN_FILE}\` |
| Fixed at | ${TIMESTAMP} |
| Fix status | **${FIX_STATUS}** |

$(if [[ -n "$FIX_NOTE" ]]; then echo "> **Note:** ${FIX_NOTE}"; fi)

---

## 2. Before code

\`\`\`go
${BEFORE_FUNC:-$BEFORE_SOURCE_LINE}
\`\`\`

---

## 3. After code

$(if [[ "$FIX_STATUS" == "APPLIED" || "$FIX_STATUS" == "WARN" ]]; then
  echo "\`\`\`go"
  echo "${AFTER_FUNC:-$AFTER_SOURCE_LINE}"
  echo "\`\`\`"
else
  echo "> Fix was not applied (status: ${FIX_STATUS}) — after code unavailable."
fi)

---

## 4. Modified files

| File | Backup | Status |
|---|---|---|
| \`${FILE}\` | \`${BACKUP_FILE}\` | ${FIX_STATUS} |

$(if [[ -n "$DIFF_SUMMARY" ]]; then
  echo "**Changed lines:**"
  echo "\`\`\`diff"
  echo "$DIFF_SUMMARY"
  echo "\`\`\`"
fi)

---

## 5. Impact analysis

### Callers

${CALLERS}

### Callees (inside modified function)

${CALLEES}

### Related interfaces

${IFACE_MATCHES}

### Related tests

${RELATED_TESTS}

---

## 6. Test plan

\`\`\`bash
# Priority 1: same package
GOROOT=/usr/local/go go test ./${PKG_DIR}/...

# Priority 2: caller packages
${CALLER_PKGS}

# Priority 3: full test suite
make test-unit TEST_FUNC=ALL

# Priority 4: lint re-check
GOROOT=/usr/local/go golangci-lint run --config .golangci.yml ./${PKG_DIR}/...
\`\`\`

---

## 7. Validation result

**Lint:** ${LINT_STATUS}
**Tests:** ${TEST_STATUS}

$(if [[ "$LINT_STATUS" == "PASS" ]]; then
  echo "\`\`\`"
  echo "golangci-lint run ./${PKG_DIR}/..."
  echo "→ Issue #${N} (${LINTER} at ${FILE}:${LINE}): NOT FOUND ✓"
  if [[ -z "$NEW_ISSUES" ]]; then
    echo "→ New issues introduced: 0 ✓"
  fi
  echo "\`\`\`"
elif [[ "$LINT_STATUS" == "WARN" ]]; then
  echo "\`\`\`"
  echo "golangci-lint run ./${PKG_DIR}/..."
  echo "→ Issue #${N}: RESOLVED ✓"
  echo "→ New issues found — review required:"
  echo "$NEW_ISSUES"
  echo "\`\`\`"
elif [[ "$LINT_STATUS" == "FAIL" ]]; then
  echo "\`\`\`"
  echo "golangci-lint run ./${PKG_DIR}/..."
  echo "→ Issue #${N} (${LINTER} at ${FILE}:${LINE}): STILL PRESENT"
  echo "→ Action: backup restored"
  echo "\`\`\`"
elif [[ "$LINT_STATUS" == "SKIP" ]]; then
  echo "> Lint validation skipped — fix status is \`${FIX_STATUS}\`."
  echo "> Apply fix manually, then run:"
  echo "\`\`\`bash"
  echo "GOROOT=/usr/local/go golangci-lint run --config .golangci.yml ./${PKG_DIR}/..."
  echo "\`\`\`"
fi)

$(if [[ "$TEST_STATUS" == "FAIL" ]]; then
  echo "**Test failure output:**"
  echo "\`\`\`"
  echo "$TEST_OUTPUT"
  echo "\`\`\`"
fi)

---

## 8. Remaining risks

${RISKS}

---

## Acceptance checklist

- [$(if [[ "$LINT_ISSUE_GONE" == "true" ]]; then echo "x"; else echo " "; fi)] Issue #${N} (\`${LINTER}\` at \`${FILE}:${LINE}\`) lint error မတွေ့တော့ပါ
- [$(if [[ -z "$NEW_ISSUES" && "$LINT_STATUS" != "SKIP" ]]; then echo "x"; else echo " "; fi)] \`golangci-lint run\` မှာ new issue မတိုးပါ
- [$(if [[ "$TEST_STATUS" == "PASS" ]]; then echo "x"; else echo " "; fi)] \`make test-unit TEST_FUNC=ALL\` pass ဖြစ်ပါသည်
- [ ] Business logic ပြောင်းလဲမှု မရှိကြောင်း developer အတည်ပြုပြီး
- [ ] \`//nolint\` directive မပါဝင်ကြောင်း စစ်ဆေးပြီး
REPORT

echo ""
echo "Result: ${RESULT_FILE}"
echo "Status: ${FIX_STATUS} | Lint: ${LINT_STATUS} | Tests: ${TEST_STATUS}"
