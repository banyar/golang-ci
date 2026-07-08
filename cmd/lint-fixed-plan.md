# lint-fixed-plan command specification

## Command

```bash
make lint-fixed-plan N=<issue_number>
```

---

## Input validation

`N` parameter မပါလျှင် command ကို terminate လုပ်ရမည်။ Usage message ပြရမည်။

```text
Error: issue number required

Usage:
make lint-fixed-plan N=<issue_number>

Example:
make lint-fixed-plan N=12
```

---

## Source data

Latest golangci-lint JSON report ကို `golangci/` folder ထဲမှ ရှာရမည်။

```text
golangci/2026-06-23_16-02-53.json
```

Issue number `N` ကို JSON report မှ extract လုပ်ရမည်။

---

## Plan file output

```text
plans/YYYY-MM-DD-lint-fix-<N>.md
```

Example:

```text
plans/2026-06-23-lint-fix-12.md
```

---

## Plan content

Plan file တွင် အောက်ပါ section ၉ ခု ပါရမည်။

---

### 1. Issue summary

| Field | Value |
|---|---|
| Issue number | N |
| Linter | e.g. errcheck |
| File | e.g. frontiir/api/router.go |
| Line | e.g. 16 |
| Message | e.g. Error return value of `r.SetTrustedProxies` is not checked |

---

### 2. Original code context

Issue line ±10 lines ကို ဖော်ပြရမည်။

```go
// original code shown here
```

---

### 3. Root cause analysis

Lint issue ဘာကြောင့်ဖြစ်သည်ကို linter rule အလိုက် ရှင်းပြရမည်။

| Linter | Explanation |
|---|---|
| `errcheck` | Function မှ return လုပ်သော error ကို မစစ်ဆေးဘဲ ignore လုပ်ထားသည် |
| `gosec` | Security risk ဖြစ်သော pattern တွေ့ရှိသည် (path traversal, overflow, loose permissions) |
| `revive` | Go convention နှင့် မကိုက်ညီသော style ရှိသည် |
| `gocyclo` | Function cyclomatic complexity limit ကျော်လွန်သည် |
| `nestif` | Nested if depth limit ကျော်လွန်သည် |
| `ineffassign` | Variable assign လုပ်ထားသော်လည်း နောက်မှ မသုံးဘဲ |
| `misspell` | စာလုံးပေါင်း မှားနေသည် |

---

### 4. Fix strategy

Issue ကို ဘယ်လို ဖြေရှင်းမည်ကို အသေးစိတ် ရေးရမည်။

Example (errcheck):

```go
if err := r.SetTrustedProxies(nil); err != nil {
    return nil
}
```

---

### 5. Before → After preview

Code ကို တကယ်မပြင်ရသေးပါ။ Proposed change ကို preview အဖြစ်သာ ထုတ်ပေးရမည်။

#### Before

```go
r.SetTrustedProxies(nil)
```

#### After

```go
if err := r.SetTrustedProxies(nil); err != nil {
    return nil
}
```

---

### 6. Possible side effects

Fix ပြုလုပ်ပါက ဖြစ်နိုင်သော အကျိုးသက်ရောက်မှုများကို ရေးရမည်။

Examples:

* additional error handling required
* behavior changes
* permission changes
* API output changes

---

### 7. Impact analysis

အောက်ပါအရာများကို စစ်ဆေးရမည်။

* caller functions
* related package
* related middleware
* related interfaces
* related integration points

---

### 8. Recommended tests

Priority order:

1. same package tests
2. caller package tests
3. integration tests
4. full test suite

Example:

```bash
go test ./frontiir/api/...
go test ./frontiir/middleware/...
make test-unit TEST_FUNC=ALL
```

---

### 9. Acceptance criteria

Plan complete ဖြစ်ရန် အောက်ပါ အားလုံး ပြည့်မီရမည်:

* lint issue ကို ဖြေရှင်းနိုင်ရမည်
* new lint issue မတိုးရ
* tests pass ရမည်
* business logic မပြောင်းရ

---

## Important rules

| Rule | Details |
|---|---|
| Plan only | ဤ command သည် plan file ကိုသာ generate လုပ်ရမည် |
| No code changes | Source code ကို modify မလုပ်ရ |
| No git changes | Git files မပြောင်းရ |
| No suppression | Lint issue ကို suppress မလုပ်ရ |
| No nolint | `//nolint` comment မထည့်ရ |
