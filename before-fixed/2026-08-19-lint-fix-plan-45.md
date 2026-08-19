# Lint fix plan — issue #45: `gosec`

**Generated:** 2026-08-19
**Source report:** `golangci/linter-report/golangci-default/2026-08-19_00-32-42.json`

---

## 1. Issue summary

| Field | Value |
|---|---|
| Issue number | 45 |
| Linter | `gosec` |
| File | `../golangci/backend/cmd/dashboard/main.go` |
| Line | 102 |
| Column | 10 |
| Message | G112: Potential Slowloris Attack because ReadHeaderTimeout is not configured in the http.Server |

---

## 2. Original code context

Line 102 (±10) — marker `▶` = flagged line.

```go
(file not readable: ../golangci/backend/cmd/dashboard/main.go)
```

**Flagged source line:**

```go
	srv := &http.Server{
```

---

## 3. Root cause analysis

**gosec rule:** `G112: Potential Slowloris Attack because ReadHeaderTimeout is not configured in the http.Server` — security violation တွေ့ရှိသည်။ https://securego.io မှ rule detail ကြည့်ပါ။

---

## 4. Fix strategy

gosec rule documentation ကို ကြည့်ပြီး project context နှင့် ကိုက်ညီသော fix pattern ရွေးချယ်ပါ။

```go
// TODO: apply fix per gosec rule — G112: Potential Slowloris Attack because ReadHeaderTimeout is not configured in the http.Server
```

---

## 5. Before → After preview

> Code ကို modify မလုပ်ရသေးပါ။ ဒီ section သည် proposed change ၏ preview သာ ဖြစ်သည်။

### Before

```go
	srv := &http.Server{
```

### After

```go
// TODO: apply fix per gosec rule — G112: Potential Slowloris Attack because ReadHeaderTimeout is not configured in the http.Server
```

---

## 6. Possible side effects

* Security fix — behavior change ဖြစ်နိုင်သည်

---

## 7. Impact analysis

**Affected file:** `../golangci/backend/cmd/dashboard/main.go`
**Affected package:** `../golangci/backend/cmd/dashboard`
**Base symbol:** `main`

**Callers / references found:**

* No external callers found — change is local to `../golangci/backend/cmd/dashboard`

> Verify caller behavior does not change after the fix.

---

## 8. Recommended tests

Priority order:

```bash
# 1. Same package
go test ./../golangci/backend/cmd/dashboard/...
make test-unit TEST_FUNC=ALL

# 2. Full lint re-check — confirm issue #45 is resolved
GOROOT=/usr/local/go golangci-lint run --config .golangci.yml ./../golangci/backend/cmd/dashboard/...

# 3. Full test suite
make test-unit TEST_FUNC=ALL
```

---

## 9. Acceptance criteria

- [ ] Issue #45 (`gosec` at `../golangci/backend/cmd/dashboard/main.go:102`) lint error မတွေ့တော့ပါ
- [ ] `golangci-lint run` မှာ new issue မတိုးပါ
- [ ] `make test-unit TEST_FUNC=ALL` pass ဖြစ်ပါသည်
- [ ] Business logic ပြောင်းလဲမှု မရှိပါ
- [ ] `//nolint` directive မပါဝင်ပါ

---

## Important rules

> ဤ plan file သည် analysis document သာ ဖြစ်သည်။
>
> * Source code ကို modify မလုပ်ရ
> * Git state မပြောင်းရ
> * Lint issue ကို `//nolint` ဖြင့် suppress မလုပ်ရ
