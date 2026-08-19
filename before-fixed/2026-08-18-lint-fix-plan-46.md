# Lint fix plan — issue #46: `gosec`

**Generated:** 2026-08-18
**Source report:** `golangci/linter-report/golangci-default/2026-08-18_23-02-33.json`

---

## 1. Issue summary

| Field | Value |
|---|---|
| Issue number | 46 |
| Linter | `gosec` |
| File | `../golangci/backend/api/middleware.go` |
| Line | 31 |
| Column | 12 |
| Message | G304: Potential file inclusion via variable |

---

## 2. Original code context

Line 31 (±10) — marker `▶` = flagged line.

```go
(file not readable: ../golangci/backend/api/middleware.go)
```

**Flagged source line:**

```go
	b, err := os.ReadFile(path)
```

---

## 3. Root cause analysis

**gosec G304:** `os.Open` / `os.ReadFile` မှာ user-controlled သို့မဟုတ် variable path ကို တိုက်ရိုက်သုံးနေသည်။ Path traversal attack (`../../etc/passwd`) ဖြစ်နိုင်သည်။

---

## 4. Fix strategy

Path ကို `filepath.Clean()` ဖြင့် normalize လုပ်ပြီး `strings.HasPrefix()` ဖြင့် allowed base directory ထဲ ရှိမရှိ စစ်မည်။

```go
clean := filepath.Clean(inputPath)
allowedBase := filepath.Clean(baseDir)
if !strings.HasPrefix(clean, allowedBase+string(filepath.Separator)) {
    return fmt.Errorf("path outside allowed directory: %s", clean)
}
data, err := os.ReadFile(clean)
```

---

## 5. Before → After preview

> Code ကို modify မလုပ်ရသေးပါ။ ဒီ section သည် proposed change ၏ preview သာ ဖြစ်သည်။

### Before

```go
	b, err := os.ReadFile(path)
```

### After

```go
clean := filepath.Clean(inputPath)
allowedBase := filepath.Clean(baseDir)
if !strings.HasPrefix(clean, allowedBase+string(filepath.Separator)) {
    return fmt.Errorf("path outside allowed directory: %s", clean)
}
data, err := os.ReadFile(clean)
```

---

## 6. Possible side effects

* Allowed directory ထဲ မရှိသော path ရှိ callerများ error ရမည် — integration test စစ်ဆေးပါ
* Config file loader ဆိုရင် startup failure ဖြစ်နိုင် — path constants စစ်ဆေးပါ

---

## 7. Impact analysis

**Affected file:** `../golangci/backend/api/middleware.go`
**Affected package:** `../golangci/backend/api`
**Base symbol:** `middleware`

**Callers / references found:**

* No external callers found — change is local to `../golangci/backend/api`

> Verify caller behavior does not change after the fix.

---

## 8. Recommended tests

Priority order:

```bash
# 1. Same package
go test ./../golangci/backend/api/...
make test-unit TEST_FUNC=ALL

# 2. Full lint re-check — confirm issue #46 is resolved
GOROOT=/usr/local/go golangci-lint run --config .golangci.yml ./../golangci/backend/api/...

# 3. Full test suite
make test-unit TEST_FUNC=ALL
```

---

## 9. Acceptance criteria

- [ ] Issue #46 (`gosec` at `../golangci/backend/api/middleware.go:31`) lint error မတွေ့တော့ပါ
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
