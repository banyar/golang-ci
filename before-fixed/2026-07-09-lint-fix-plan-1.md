# Lint fix plan — issue #1: `errcheck`

**Generated:** 2026-07-09
**Source report:** `golangci/linter-report/golangci-default/2026-07-09_10-38-20.json`

---

## 1. Issue summary

| Field | Value |
|---|---|
| Issue number | 1 |
| Linter | `errcheck` |
| File | `frontiir/api/controllers/analytics_controller.go` |
| Line | 286 |
| Column | 15 |
| Message | Error return value of `f.Close` is not checked |

---

## 2. Original code context

Line 286 (±10) — marker `▶` = flagged line.

```go
   276  func readLogFile(path string) ([]AnalyticsLogEntry, error) {
   277  	clean := filepath.Clean(path)
   278  	allowedBase := filepath.Clean(getLogDir())
   279  	if !strings.HasPrefix(clean, allowedBase+string(filepath.Separator)) {
   280  		return nil, fmt.Errorf("path outside allowed directory: %s", clean)
   281  	}
   282  	f, err := os.Open(clean)
   283  	if err != nil {
   284  		return nil, err
   285  	}
▶  286  	defer f.Close()
   287  
   288  	var entries []AnalyticsLogEntry
   289  	scanner := bufio.NewScanner(f)
   290  	for scanner.Scan() {
   291  		line := scanner.Text()
   292  		if line == "" {
   293  			continue
   294  		}
   295  		var entry AnalyticsLogEntry
   296  		if err := json.Unmarshal([]byte(line), &entry); err == nil {
```

**Flagged source line:**

```go
	defer f.Close()
```

---

## 3. Root cause analysis

**errcheck rule:** function မှ return လုပ်သော `error` ကို caller မှ မစစ်ဆေးဘဲ ignore လုပ်ထားသည်။ Production မှာ disk full / network error တွေကို သိရှိနိုင်မည်မဟုတ် — silent failure ဖြစ်သည်။

---

## 4. Fix strategy

defer statement ကို error-checking closure ဖြင့် wrap လုပ်မည်။ Close error ကို zap logger ဖြင့် log ထုတ်မည်။

```go
defer func() {
    if cerr := resource.Close(); cerr != nil {
        log.Printf("close %s: %v", resourceName, cerr)
    }
}()
```

---

## 5. Before → After preview

> Code ကို modify မလုပ်ရသေးပါ။ ဒီ section သည် proposed change ၏ preview သာ ဖြစ်သည်။

### Before

```go
	defer f.Close()
```

### After

```go
defer func() {
    if cerr := resource.Close(); cerr != nil {
        log.Printf("close %s: %v", resourceName, cerr)
    }
}()
```

---

## 6. Possible side effects

* Error visibility တိုးသည် — behavior မပြောင်း
* Defer close error ဆိုရင် resource cleanup path မပြောင်း
* Test coverage ထည့်မည်ဆိုရင် error injection mock လိုနိုင်သည်

---

## 7. Impact analysis

**Affected file:** `frontiir/api/controllers/analytics_controller.go`
**Affected package:** `frontiir/api/controllers`
**Base symbol:** `f.Close`

**Callers / references found:**

* `frontiir/middleware/api_analytics.go:74`
* `frontiir/middleware/api_analytics.go:86`

> Verify caller behavior does not change after the fix.

---

## 8. Recommended tests

Priority order:

```bash
# 1. Same package
go test ./frontiir/api/controllers/...
make test-unit TEST_FUNC=ALL

# 2. Full lint re-check — confirm issue #1 is resolved
GOROOT=/usr/local/go golangci-lint run --config .golangci.yml ./frontiir/api/controllers/...

# 3. Full test suite
make test-unit TEST_FUNC=ALL
```

---

## 9. Acceptance criteria

- [ ] Issue #1 (`errcheck` at `frontiir/api/controllers/analytics_controller.go:286`) lint error မတွေ့တော့ပါ
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
