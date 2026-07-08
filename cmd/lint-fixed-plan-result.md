# lint-fixed command specification

## Command

```bash
make lint-fixed N=<issue_number>
```

---

## Rules

### Rule 1 — Input validation

`N` parameter မပါလျှင် command ကို terminate လုပ်ပြီး usage message ပြပါ။

```text
Error: issue number required

Usage:
  make lint-fixed N=<issue_number>

Example:
  make lint-fixed N=12
```

---

### Rule 2 — Plan file lookup

`golangci/before-fixed/` directory ထဲမှ issue number နှင့် ကိုက်ညီသော plan file ကို ရှာပါ။

```text
golangci/before-fixed/*-lint-fix-plan-<N>.md
```

Plan file မတွေ့ပါက error ပြပြီး terminate လုပ်ပါ။

```text
Error: plan file for issue #<N> not found in golangci/before-fixed/
Run 'make lint-fixed-plan N=<N>' first.
```

---

### Rule 3 — Plan file parsing

Plan file ထဲမှ အောက်ပါ section များကို parse လုပ်ပါ။

| Section | Source heading |
|---|---|
| Linter | `## 1. Issue summary` → Linter row |
| File | `## 1. Issue summary` → File row |
| Line | `## 1. Issue summary` → Line row |
| Issue message | `## 1. Issue summary` → Message row |
| Fix strategy | `## 4. Fix strategy` |
| Side effects | `## 6. Possible side effects` |
| Verification | `## 8. Recommended tests` |

---

### Rule 4 — Code modification

Fix Strategy section ကို primary source အဖြစ် သတ်မှတ်ပြီး source file ကို modify လုပ်ပါ။

* Plan file ထဲက code block ကိုသာ reference လုပ်ပြီး fix လုပ်ပါ
* Plan မှာ မပါသော logic မထည့်ရ
* Public API signature မပြောင်းရ
* Business logic မပြောင်းရ

---

### Rule 5 — Pre-fix snapshot

Code မပြင်မီ အောက်ပါ ၃ ခု လုပ်ဆောင်ပါ။

#### 5a. Backup original file

```bash
cp <file> <file>.bak.<date>
```

#### 5b. Extract affected function

Line number ကို အသုံးပြုပြီး affected function scope ကို extract လုပ်ပါ။

#### 5c. Before snapshot

Extract လုပ်ထားသော function ကို result report ထဲ Before Code section မှာ သိမ်းပါ။

---

### Rule 6 — Post-fix snapshot

Code ပြင်ပြီးနောက် အောက်ပါ ၂ ခု လုပ်ဆောင်ပါ။

#### 6a. After snapshot

Modified function ကို result report ထဲ After Code section မှာ သိမ်းပါ။

#### 6b. Changed lines summary

```text
File: frontiir/api/router.go
  Line 16: r.SetTrustedProxies(nil)
       →   if err := r.SetTrustedProxies(nil); err != nil { ... }
```

---

### Rule 7 — Impact analysis

ပြင်ထားသော function ကို AST (Abstract Syntax Tree) ဖြင့် analyze လုပ်ပြီး အောက်ပါ ၄ ခု ရှာပါ။

| Target | Method |
|---|---|
| Callers | `grep -rn <function_name>` ဖြင့် reference ရှာ |
| Callees | Modified function ထဲမှ function call တွေ extract လုပ် |
| Related interfaces | Modified function ၏ signature နှင့် ကိုက်ညီသော interface တွေ ရှာ |
| Related tests | File / function name ပြောင်းသော `_test.go` files ရှာ |

---

### Rule 8 — Test selection

အောက်ပါ priority order ဖြင့် test များ ရွေးချယ်ပါ။

| Priority | Scope | Command example |
|---|---|---|
| 1 | Same package tests | `go test ./frontiir/api/...` |
| 2 | Caller package tests | `go test ./frontiir/middleware/...` |
| 3 | Integration tests affected by changed API | `make test TEST_FUNC=<TestName>` |
| 4 | Full project test suite | `make test-unit TEST_FUNC=ALL` |

---

### Rule 9 — Validation

Fix ပြီးနောက် lint ကို ပြန် run ပြီး issue #N ပျောက်သွားကြောင်း အတည်ပြုပါ။

```bash
GOROOT=/usr/local/go golangci-lint run --config .golangci.yml ./<pkg>/...
```

**Validation outcomes:**

| Result | Action |
|---|---|
| Issue #N ပျောက်သွားသည် | PASS — report ထဲ Validation Result မှာ မှတ်တမ်းတင် |
| Issue #N ရှိနေသည် | FAIL — backup restore လုပ်ပြီး report ထဲ failure reason ရေး |
| New lint issues ထွက်လာသည် | FAIL — backup restore လုပ်ပြီး new issues list ကို report ထဲ ထည့် |

---

### Rule 10 — Safety rules

| Rule | Details |
|---|---|
| Lint fix required | Fix သည် issue #N ကို ဖြေရှင်းရမည် |
| Business logic preserved | Business logic မပြောင်းရ |
| API signature preserved | Public function / method signature မပြောင်းရ |
| Behavioral change disclosure | Behavioral change ဖြစ်သည်ဆိုရင် report ထဲ explicit note ထည့်ရမည် |
| No suppression | `//nolint` directive မထည့်ရ |
| Backup always | Original file ကို backup မသိမ်းမချင်း fix မလုပ်ရ |

---

### Rule 11 — Output

Result file ကို အောက်ပါ path ထဲ generate လုပ်ပါ။

```text
golangci/after-fixed/YYYY-MM-DD-lint-fixed-<N>.md
```

Example:

```text
golangci/after-fixed/2026-06-23-lint-fixed-12.md
```

---

## Output file structure

Result file တွင် အောက်ပါ section ၈ ခု ပါရမည်။

### 1. Issue summary

| Field | Value |
|---|---|
| Issue number | N |
| Linter | e.g. errcheck |
| File | e.g. frontiir/api/router.go |
| Line | e.g. 16 |
| Message | e.g. Error return value of `r.SetTrustedProxies` is not checked |
| Plan file | golangci/before-fixed/YYYY-MM-DD-lint-fix-plan-N.md |
| Fixed at | YYYY-MM-DD HH:MM:SS |

---

### 2. Before code

Fix မလုပ်မီ affected function ၏ original code။

```go
// before fix
func InitRouter() *gin.Engine {
    r := gin.Default()
    r.SetTrustedProxies(nil)  // ← flagged line
    ...
}
```

---

### 3. After code

Fix ပြီးနောက် modified function ၏ code။

```go
// after fix
func InitRouter() *gin.Engine {
    r := gin.Default()
    if err := r.SetTrustedProxies(nil); err != nil {
        return nil
    }
    ...
}
```

---

### 4. Modified files

| File | Lines changed | Backup |
|---|---|---|
| `frontiir/api/router.go` | 16 → 18 (+2) | `frontiir/api/router.go.bak.2026-06-23` |

---

### 5. Impact analysis

#### Callers

* No callers found — function is package-internal

#### Callees

* `gin.Default()`
* `r.SetTrustedProxies()`

#### Related interfaces

* None identified

#### Related tests

* `frontiir/api/router_test.go` (if exists)

---

### 6. Test plan

Priority order နှင့် run မည့် test command တွေ ဖော်ပြပါ။

```bash
# Priority 1: same package
go test ./frontiir/api/...

# Priority 2: caller packages
go test ./frontiir/middleware/...

# Priority 3: integration
make test TEST_FUNC=TestInitRouter

# Priority 4: full suite
make test-unit TEST_FUNC=ALL
```

---

### 7. Validation result

```text
Status: PASS

golangci-lint run ./frontiir/api/...
→ Issue #2 (errcheck at frontiir/api/router.go:16): NOT FOUND ✓
→ New issues introduced: 0 ✓

make test-unit TEST_FUNC=ALL
→ All tests passed ✓
```

Failure example:

```text
Status: FAIL

golangci-lint run ./frontiir/api/...
→ Issue #2 (errcheck at frontiir/api/router.go:16): STILL PRESENT
→ Reason: fix pattern did not match actual code structure

Action: backup restored → frontiir/api/router.go.bak.2026-06-23
```

---

### 8. Remaining risks

Fix ပြီးနောက် ဆက်လက် စစ်ဆေးရမည့် အရာများ ဖော်ပြပါ။

Example:

* `InitRouter()` ကို `nil` return ဖြစ်ရင် caller စစ်ဆေးမလုပ်ထားပါက nil pointer panic ဖြစ်နိုင်သည်
* Integration test ထဲမှာ `InitRouter()` result ကို nil check ထည့်ထားခြင်း ရှိ/မရှိ စစ်ဆေးပါ
