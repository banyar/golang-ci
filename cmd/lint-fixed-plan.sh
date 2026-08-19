#!/usr/bin/env bash
# lint-fixed-plan-v1.sh — generate a structured Markdown fix-plan for a golangci-lint issue.
# Usage: bash scripts/lint-fixed-plan-v1.sh <N>
#   N — 1-based issue number from the latest golangci JSON report
# Output: plans/YYYY-MM-DD-lint-fix-<N>.md
# Does NOT modify source code, git state, or add //nolint directives.
set -euo pipefail

# ── Input validation ──────────────────────────────────────────────────────────
N="${1:-}"
if [[ -z "$N" ]]; then
  echo "Error: issue number required." >&2
  echo "" >&2
  echo "Usage:" >&2
  echo "  make lint-fixed-plan N=<issue_number>" >&2
  echo "" >&2
  echo "Example:" >&2
  echo "  make lint-fixed-plan N=12" >&2
  exit 1
fi

if ! [[ "$N" =~ ^[0-9]+$ ]] || [[ "$N" -lt 1 ]]; then
  echo "Error: N must be a positive integer, got: '${N}'" >&2
  exit 1
fi

# ── Dependencies check ────────────────────────────────────────────────────────
if ! command -v jq &>/dev/null; then
  echo "Error: jq is required but not installed." >&2
  exit 1
fi

# ── Locate latest JSON report ─────────────────────────────────────────────────
REPORT_DIR="${REPORT_DIR:-golangci/linter-report}"
LATEST_JSON=$(find "$REPORT_DIR" -maxdepth 2 -name "*.json" 2>/dev/null | sort -r | head -1)
if [[ -z "$LATEST_JSON" ]]; then
  echo "Error: no lint JSON report found in ${REPORT_DIR}/. Run 'make lint' first." >&2
  exit 1
fi

# ── Extract issue #N (1-based, sorted by linter → file → line) ───────────────
ISSUE_JSON=$(jq --argjson n "$N" '
  (.Issues // [])
  | sort_by(.FromLinter, .Pos.Filename, .Pos.Line)
  | to_entries
  | map(.value + {_num: (.key + 1)})
  | map(select(._num == $n))
  | first // empty
' "$LATEST_JSON")

if [[ -z "$ISSUE_JSON" ]]; then
  TOTAL=$(jq '(.Issues // []) | length' "$LATEST_JSON")
  echo "Error: issue #${N} not found in ${LATEST_JSON} (total issues: ${TOTAL})." >&2
  exit 1
fi

LINTER=$(jq -r '.FromLinter'        <<< "$ISSUE_JSON")
FILE=$(jq -r '.Pos.Filename'        <<< "$ISSUE_JSON")
LINE=$(jq -r '.Pos.Line'            <<< "$ISSUE_JSON")
COL=$(jq -r '.Pos.Column'           <<< "$ISSUE_JSON")
TEXT=$(jq -r '.Text'                <<< "$ISSUE_JSON")
SOURCE_LINE=$(jq -r '(.SourceLines // ["(source unavailable)"])[0]' <<< "$ISSUE_JSON")

# golangci-lint records Pos.Filename relative to the golangci/ project root
# (it's invoked via `cd golangci && golangci-lint run ...`), but this script
# itself runs one level up (Makefile does `cd .. && bash golangci/cmd/...`),
# so disk access needs the golangci/ prefix that $FILE lacks.
SRC_FILE="golangci/${FILE}"

# ── Read code context (LINE ±10) ──────────────────────────────────────────────
CONTEXT_START=$(( LINE > 10 ? LINE - 10 : 1 ))
CONTEXT_END=$(( LINE + 10 ))
if [[ -f "$SRC_FILE" ]]; then
  CODE_CONTEXT=$(awk \
    -v s="$CONTEXT_START" -v e="$CONTEXT_END" -v hl="$LINE" \
    'NR>=s && NR<=e { marker="  "; if(NR==hl) marker="▶ "; printf "%s%4d  %s\n", marker, NR, $0 }' \
    "$SRC_FILE")
else
  CODE_CONTEXT="(file not readable: ${FILE})"
fi

# ── Impact analysis: find callers ─────────────────────────────────────────────
BASE=$(basename "$FILE" .go)
PKG_DIR=$(dirname "$FILE")
SYMBOL=$(echo "$TEXT" | grep -oP '`\K[^`]+(?=`)' | head -1 || true)

CALLERS=""
if [[ -n "$SYMBOL" ]]; then
  CALLERS=$(grep -rn "$SYMBOL" --include="*.go" frontiir/ 2>/dev/null \
    | grep -v "^${FILE}:" \
    | awk -F: '{printf "* `%s:%s`\n", $1, $2}' \
    | head -10 || true)
fi
[[ -z "$CALLERS" ]] && CALLERS="* No external callers found — change is local to \`${PKG_DIR}\`"

# ── Linter-specific fix templates ─────────────────────────────────────────────
ROOT_CAUSE=""
FIX_STRATEGY=""
BEFORE_CODE="$SOURCE_LINE"
AFTER_CODE=""
SIDE_EFFECTS=""

case "$LINTER" in
  errcheck)
    ROOT_CAUSE="Go では関数がエラーを返す場合、呼び出し側で必ずそのエラーを確認しなければなりません。
\`errcheck\` linter သည် \`${TEXT}\` ကို တွေ့ရှိသည် — return error ကို \`_\` ဖြင့် discard သို့မဟုတ် ဆက်မလုပ်ဘဲ ignore လုပ်ထားသောကြောင့် production မှာ silent failure ဖြစ်နိုင်သည်။"
    ROOT_CAUSE="**errcheck rule:** function မှ return လုပ်သော \`error\` ကို caller မှ မစစ်ဆေးဘဲ ignore လုပ်ထားသည်။ Production မှာ disk full / network error တွေကို သိရှိနိုင်မည်မဟုတ် — silent failure ဖြစ်သည်။"

    if echo "$SOURCE_LINE" | grep -q "defer"; then
      FIX_STRATEGY="defer statement ကို error-checking closure ဖြင့် wrap လုပ်မည်။ Close error ကို zap logger ဖြင့် log ထုတ်မည်။"
      AFTER_CODE='defer func() {
    if cerr := resource.Close(); cerr != nil {
        log.Printf("close %s: %v", resourceName, cerr)
    }
}()'
    else
      FIX_STRATEGY="Function call ၏ error return ကို \`if err :=\` pattern ဖြင့် ရယူပြီး handle လုပ်မည်။"
      AFTER_CODE='if err := operation(); err != nil {
    // handle or return
    return fmt.Errorf("operation failed: %w", err)
}'
    fi
    SIDE_EFFECTS="* Error visibility တိုးသည် — behavior မပြောင်း
* Defer close error ဆိုရင် resource cleanup path မပြောင်း
* Test coverage ထည့်မည်ဆိုရင် error injection mock လိုနိုင်သည်"
    ;;

  gosec)
    if echo "$TEXT" | grep -q "G304"; then
      ROOT_CAUSE="**gosec G304:** \`os.Open\` / \`os.ReadFile\` မှာ user-controlled သို့မဟုတ် variable path ကို တိုက်ရိုက်သုံးနေသည်။ Path traversal attack (\`../../etc/passwd\`) ဖြစ်နိုင်သည်။"
      FIX_STRATEGY="Path ကို \`filepath.Clean()\` ဖြင့် normalize လုပ်ပြီး \`strings.HasPrefix()\` ဖြင့် allowed base directory ထဲ ရှိမရှိ စစ်မည်။"
      AFTER_CODE='clean := filepath.Clean(inputPath)
allowedBase := filepath.Clean(baseDir)
if !strings.HasPrefix(clean, allowedBase+string(filepath.Separator)) {
    return fmt.Errorf("path outside allowed directory: %s", clean)
}
data, err := os.ReadFile(clean)'
      SIDE_EFFECTS="* Allowed directory ထဲ မရှိသော path ရှိ callerများ error ရမည် — integration test စစ်ဆေးပါ
* Config file loader ဆိုရင် startup failure ဖြစ်နိုင် — path constants စစ်ဆေးပါ"

    elif echo "$TEXT" | grep -q "G301"; then
      ROOT_CAUSE="**gosec G301:** Directory ကို \`0755\` (world-readable+executable) ဖြင့် ဖန်တီးနေသည်။ \`0750\` (owner+group only) သုံးသင့်သည်။"
      FIX_STRATEGY="Permission argument ကို \`0755\` မှ \`0750\` သို့ ပြောင်းမည်။"
      AFTER_CODE='if err := os.MkdirAll(dirPath, 0750); err != nil {
    return fmt.Errorf("mkdir %s: %w", dirPath, err)
}'
      SIDE_EFFECTS="* Other users (world) သည် directory ကို list/execute မနိုင်တော့ပါ
* Log aggregator / monitoring agent သည် log directory ကို ဖတ်နိုင်ဖို့ same group ထဲ ရှိရမည်"

    elif echo "$TEXT" | grep -q "G302"; then
      ROOT_CAUSE="**gosec G302:** File ကို \`0644\` (world-readable) ဖြင့် ဖန်တီးနေသည်။ Log file များသည် \`0600\` (owner only) ဖြင့် ဖန်တီးသင့်သည်။"
      FIX_STRATEGY="Permission argument ကို \`0644\` မှ \`0600\` သို့ ပြောင်းမည်။"
      AFTER_CODE='f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)'
      SIDE_EFFECTS="* Group / world မှ file ကို ဖတ်မရတော့ပါ
* External log reader process ရှိသည်ဆိုရင် permission စစ်ဆေးပါ"

    elif echo "$TEXT" | grep -q "G115"; then
      ROOT_CAUSE="**gosec G115:** \`int\` (64-bit) မှ \`int32\` (32-bit) conversion မှာ runtime overflow ဖြစ်နိုင်သည်။ 2,147,483,647 ကျော်သော value ဆိုရင် wrap-around ဖြစ်သည်။"
      FIX_STRATEGY="Conversion မတိုင်ခင် \`math.MaxInt32\` နှင့် bound check လုပ်မည်။ Out-of-range ဆိုရင် error return ပြန်မည်။"
      AFTER_CODE='import "math"

if val > math.MaxInt32 || val < math.MinInt32 {
    return 0, fmt.Errorf("value %d overflows int32", val)
}
result := int32(val)'
      SIDE_EFFECTS="* Overflow case မှာ error return ဖြစ်သောကြောင့် caller error handling လိုသည်
* Proto/gRPC struct ဆိုရင် upstream ကို int32 boundary enforce လုပ်ပေးသည်"

    elif echo "$TEXT" | grep -q "G706"; then
      ROOT_CAUSE="**gosec G706:** Log statement မှာ user-controlled / external data ကို sanitize မလုပ်ဘဲ တိုက်ရိုက် log ထုတ်နေသည်။ Log injection (newline injection) ဖြစ်နိုင်သည်။"
      FIX_STRATEGY="Log မတိုင်ခင် \`strings.NewReplacer\` ဖြင့် newline / carriage-return characters ကို strip လုပ်မည်။"
      AFTER_CODE='sanitized := strings.NewReplacer("\n", " ", "\r", " ").Replace(externalInput)
log.Printf("[INFO] host: %s", sanitized)'
      SIDE_EFFECTS="* Log format မပြောင်း — newline character သာ space ဖြစ်သွားသည်
* Multi-line log injection မဖြစ်နိုင်တော့သောကြောင့် SIEM log parsing မပျက်တော့"

    elif echo "$TEXT" | grep -q "G117"; then
      ROOT_CAUSE="**gosec G117:** JSON-serialized struct မှာ \`api_key\` / \`password\` နှင့် ဆင်တူသော field name ရှိသည် — secret leak ဖြစ်နိုင်သည်။"
      FIX_STRATEGY="Field ကို JSON serialization မှ exclude လုပ်ရန် \`json:\"-\"\` tag သုံးမည် သို့မဟုတ် field ကို rename မည်။"
      AFTER_CODE='// Option A: exclude from JSON output entirely
APIKey string `json:"-"`

// Option B: rename to non-sensitive key
AuthHeader string `json:"auth_header"`'
      SIDE_EFFECTS="* JSON API response schema ပြောင်းနိုင်သည် — API consumer စစ်ဆေးပါ
* Option A သုံးပါက field ကို JSON မှ မတွေ့နိုင်တော့ပါ"

    else
      ROOT_CAUSE="**gosec rule:** \`${TEXT}\` — security violation တွေ့ရှိသည်။ https://securego.io မှ rule detail ကြည့်ပါ။"
      FIX_STRATEGY="gosec rule documentation ကို ကြည့်ပြီး project context နှင့် ကိုက်ညီသော fix pattern ရွေးချယ်ပါ။"
      AFTER_CODE="// TODO: apply fix per gosec rule — ${TEXT}"
      SIDE_EFFECTS="* Security fix — behavior change ဖြစ်နိုင်သည်"
    fi
    ;;

  gocyclo)
    ROOT_CAUSE="**gocyclo rule:** Function ၏ cyclomatic complexity သည် \`${TEXT}\` — limit 15 ကျော်လွန်သည်။ Branch (if/for/switch/case) အရေအတွက် များသောကြောင့် testing / reasoning ခက်ခဲသည်။"
    FIX_STRATEGY="Function ကို logical section တစ်ခုချင်းကို named helper function တွေ ခွဲမည်။ Each helper ၏ complexity ≤ 15 ဖြစ်ရမည်။"
    AFTER_CODE='// Extract each major branch into a focused helper:
func handleCaseA(ctx context.Context, data Data) error {
    // ...
}
func handleCaseB(ctx context.Context, data Data) error {
    // ...
}

// Orchestrator stays thin:
func mainFunc(ctx context.Context, data Data) error {
    switch data.Type {
    case TypeA:
        return handleCaseA(ctx, data)
    default:
        return handleCaseB(ctx, data)
    }
}'
    SIDE_EFFECTS="* New unexported helper functions ထပ်ထွက်မည် — package-internal ဆိုရင် breaking change မဟုတ်ပါ
* Test ကို helper level ထိ ချျပြီးစစ်ဆေးနိုင်မည်"
    ;;

  nestif)
    ROOT_CAUSE="**nestif rule:** If block ၏ nesting depth သည် limit ကျော်လွန်သည်။ Deep nesting သည် logic ကို ခြေရာခံရခက်ခဲစေပြီး bug ဖြစ်နှုန်းကို မြင့်သည်။"
    FIX_STRATEGY="Early-return (guard clause) pattern ဖြင့် nested if block တွေကို flatten လုပ်မည်။ Happy path ကို function ၏ main body ထဲ ချန်ထားမည်။"
    AFTER_CODE='// Before: deep nesting
if condA {
    if condB {
        doWork()
    }
}

// After: guard clauses (early return)
if !condA {
    return
}
if !condB {
    return
}
doWork()'
    SIDE_EFFECTS="* Logic မပြောင်း — code structure သာ ပြောင်းသည်
* Readability တိုးသောကြောင့် future bug ဖြစ်နှုန်း ကျသည်"
    ;;

  ineffassign)
    ROOT_CAUSE="**ineffassign rule:** Variable ကို value assign လုပ်ထားသော်လည်း ထို value ကို နောက်မှ မသုံးဘဲ overwrite သို့မဟုတ် function ပြန်ထွက်နေသည် — dead code ဖြစ်သည်။"
    FIX_STRATEGY="Option A: Assignment ကိုဖျက်ပြီး declaration ကိုပါ remove လုပ်ပါ (ထို variable ကို လုံးဝမသုံးရင်)။ Option B: ထို value ကို logic မှာ သုံးအောင် ပြင်ပါ။"
    AFTER_CODE='// Option A: remove unused assignment entirely
// (delete the line)

// Option B: use the computed value
result := compute()
if result != expectedValue {
    return fmt.Errorf("unexpected result: %v", result)
}'
    SIDE_EFFECTS="* Dead code ဖျက်သောကြောင့် observable behavior မပြောင်း
* Option B ဆိုရင် logic path ထပ်ထွက်နိုင်သည် — test စစ်ဆေးပါ"
    ;;

  misspell)
    ROOT_CAUSE="**misspell rule:** Identifier / comment / string literal မှာ US-English spelling မှားနေသည်: \`${TEXT}\`"
    FIX_STRATEGY="Misspelled word ကို correct spelling ဖြင့် replace မည်။ Comment မှာဆိုရင် auto-fix သုံးနိုင်သည်။ Exported identifier ဆိုရင် callers ကိုပါ update ရမည်။"
    AFTER_CODE='# Auto-fix (comments, strings):
golangci-lint run --config .golangci.yml --fix

# Exported identifier — manual rename required:
# Before: func Occured() {}
# After:  func Occurred() {}'
    SIDE_EFFECTS="* Comment / unexported identifier ဆိုရင် side effect မရှိ
* Exported identifier ဆိုရင် import လုပ်ထားသော package တွေကို rename လိုနိုင်သည် — grep လုပ်ပါ"
    ;;

  revive)
    ROOT_CAUSE="**revive rule:** \`${TEXT}\` — Go convention (Effective Go / revive ruleset) နှင့် မကိုက်ညီသော pattern တွေ့ရှိသည်။"
    FIX_STRATEGY="revive rule document ကို ကြည့်ပြီး project convention နှင့် ကိုက်ညီသော fix ရွေးချယ်မည်။ Cognitive complexity rule ဆိုရင် logic ကို helper function တွေ ခွဲမည်။"
    AFTER_CODE='// cognitive-complexity: extract complex conditions
func isValidState(s State) bool {
    return s == Active || s == Pending
}

// unused-parameter: use _ for intentionally unused params
func Handler(_ *gin.Context) { /* ... */ }'
    SIDE_EFFECTS="* Style change — logic မပြောင်း
* Exported API rename ဆိုရင် breaking change ဖြစ်နိုင် — callers စစ်ဆေးပါ"
    ;;

  funlen)
    ROOT_CAUSE="**funlen rule:** Function သည် \`${TEXT}\` — configured limit (80 lines) ကျော်လွန်သည်။ Long function တွေသည် testing / review ခက်ခဲသည်။"
    FIX_STRATEGY="Logical section တစ်ခုချင်းကို named helper function ခွဲမည်: input validation, data fetch, response build တို့ကို split လုပ်မည်။"
    AFTER_CODE='func validateRequest(req Request) error { /* ... */ }
func fetchData(ctx context.Context, req Request) (Data, error) { /* ... */ }
func buildResponse(data Data) Response { /* ... */ }

func Handler(ctx context.Context, req Request) (Response, error) {
    if err := validateRequest(req); err != nil {
        return Response{}, err
    }
    data, err := fetchData(ctx, req)
    if err != nil {
        return Response{}, err
    }
    return buildResponse(data), nil
}'
    SIDE_EFFECTS="* New unexported helpers ထပ်ထွက်မည် — logic မပြောင်း
* Test ကို helper level ထိ ချျပြီးစစ်ဆေးနိုင်မည်"
    ;;

  bodyclose)
    ROOT_CAUSE="**bodyclose rule:** HTTP response body (\`resp.Body\`) ကို \`defer resp.Body.Close()\` ဖြင့် close မလုပ်ဘဲ ချန်ထားသည် — goroutine / fd leak ဖြစ်နိုင်သည်။"
    FIX_STRATEGY="HTTP call ပြီးနောက် ချက်ချင်း \`defer resp.Body.Close()\` ထည့်မည်။ Error check ပြီးမှ defer လုပ်ရမည်။"
    AFTER_CODE='resp, err := http.Get(url)
if err != nil {
    return err
}
defer func() {
    if cerr := resp.Body.Close(); cerr != nil {
        log.Printf("body close: %v", cerr)
    }
}()'
    SIDE_EFFECTS="* Resource leak ကာကွယ်ပေးသည် — behavior မပြောင်း
* Long-running service မှာ connection pool exhaustion ကာကွယ်ပေးသည်"
    ;;

  staticcheck)
    ROOT_CAUSE="**staticcheck rule:** \`${TEXT}\` — static analysis ဖြင့် bug / deprecated API / unreachable code တွေ့ရှိသည်။"
    FIX_STRATEGY="staticcheck rule ID ကို https://staticcheck.dev/docs/checks မှ ကြည့်ပြီး project context နှင့် ကိုက်ညီသော fix ရွေးချယ်မည်။"
    AFTER_CODE="// TODO: apply fix per staticcheck rule — ${TEXT}"
    SIDE_EFFECTS="* Rule အပေါ်မူတည်၍ behavior change ဖြစ်နိုင်သည်"
    ;;

  govet)
    ROOT_CAUSE="**govet rule:** \`${TEXT}\` — Go compiler vet tool ဖြင့် suspicious construct တွေ့ရှိသည် (printf format mismatch, struct alignment, etc.)。"
    FIX_STRATEGY="govet message ကို ဖတ်ပြီး format string / struct layout ကို ပြင်မည်။"
    AFTER_CODE="// Fix per govet message: ${TEXT}"
    SIDE_EFFECTS="* Printf format mismatch fix ဆိုရင် output string ပြောင်းနိုင်သည် — log / error message test စစ်ဆေးပါ"
    ;;

  *)
    ROOT_CAUSE="**${LINTER} rule:** \`${TEXT}\` — linter documentation ကြည့်ပြီး root cause စစ်ဆေးပါ။"
    FIX_STRATEGY="${LINTER} documentation ကို ကြည့်ပြီး fix pattern ရွေးချယ်မည်။"
    AFTER_CODE="// TODO: apply fix per ${LINTER} rule"
    SIDE_EFFECTS="* Unknown — code review လိုအပ်သည်"
    ;;
esac

# ── Determine test commands from affected package ─────────────────────────────
TEST_PKG="./${PKG_DIR}/..."
TEST_COMMANDS="go test ${TEST_PKG}
make test-unit TEST_FUNC=ALL"

# ── Output plan file ──────────────────────────────────────────────────────────
PLANS_DIR="golangci/before-fixed"
mkdir -p "$PLANS_DIR"
DATE=$(date +%Y-%m-%d)
PLAN_FILE="${PLANS_DIR}/${DATE}-lint-fix-plan-${N}.md"

cat > "$PLAN_FILE" <<PLAN
# Lint fix plan — issue #${N}: \`${LINTER}\`

**Generated:** ${DATE}
**Source report:** \`${LATEST_JSON}\`

---

## 1. Issue summary

| Field | Value |
|---|---|
| Issue number | ${N} |
| Linter | \`${LINTER}\` |
| File | \`${FILE}\` |
| Line | ${LINE} |
| Column | ${COL} |
| Message | ${TEXT} |

---

## 2. Original code context

Line ${LINE} (±10) — marker \`▶\` = flagged line.

\`\`\`go
${CODE_CONTEXT}
\`\`\`

**Flagged source line:**

\`\`\`go
${SOURCE_LINE}
\`\`\`

---

## 3. Root cause analysis

${ROOT_CAUSE}

---

## 4. Fix strategy

${FIX_STRATEGY}

\`\`\`go
${AFTER_CODE}
\`\`\`

---

## 5. Before → After preview

> Code ကို modify မလုပ်ရသေးပါ။ ဒီ section သည် proposed change ၏ preview သာ ဖြစ်သည်။

### Before

\`\`\`go
${BEFORE_CODE}
\`\`\`

### After

\`\`\`go
${AFTER_CODE}
\`\`\`

---

## 6. Possible side effects

${SIDE_EFFECTS}

---

## 7. Impact analysis

**Affected file:** \`${FILE}\`
**Affected package:** \`${PKG_DIR}\`
**Base symbol:** \`${SYMBOL:-${BASE}}\`

**Callers / references found:**

${CALLERS}

> Verify caller behavior does not change after the fix.

---

## 8. Recommended tests

Priority order:

\`\`\`bash
# 1. Same package
${TEST_COMMANDS}

# 2. Full lint re-check — confirm issue #${N} is resolved
GOROOT=/usr/local/go golangci-lint run --config .golangci.yml ./${PKG_DIR}/...

# 3. Full test suite
make test-unit TEST_FUNC=ALL
\`\`\`

---

## 9. Acceptance criteria

- [ ] Issue #${N} (\`${LINTER}\` at \`${FILE}:${LINE}\`) lint error မတွေ့တော့ပါ
- [ ] \`golangci-lint run\` မှာ new issue မတိုးပါ
- [ ] \`make test-unit TEST_FUNC=ALL\` pass ဖြစ်ပါသည်
- [ ] Business logic ပြောင်းလဲမှု မရှိပါ
- [ ] \`//nolint\` directive မပါဝင်ပါ

---

## Important rules

> ဤ plan file သည် analysis document သာ ဖြစ်သည်။
>
> * Source code ကို modify မလုပ်ရ
> * Git state မပြောင်းရ
> * Lint issue ကို \`//nolint\` ဖြင့် suppress မလုပ်ရ
PLAN

echo "Plan created: ${PLAN_FILE}"
