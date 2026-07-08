# Lint fix plan — issue #3: `errcheck`

**Generated:** 2026-07-07
**Source report:** `golangci/linter-report/2026-07-07_23-10-02.json`

---

## 1. Issue summary

| Field | Value |
|---|---|
| Issue number | 3 |
| Linter | `errcheck` |
| File | `frontiir/services/customfields_service.go` |
| Line | 58 |
| Column | 18 |
| Message | Error return value of `cache.CacheSet` is not checked |

---

## 2. Original code context

Line 58 (±10) — marker `▶` = flagged line.

```go
    48  		return nil, utils.ErrResponse(cfvErr)
    49  	}
    50  
    51  	resResult := &response.FBytesRootCauseResponse{
    52  		Status: http.StatusOK,
    53  		Data:   convertToRootCauseGroups(customFieldValues),
    54  	}
    55  
    56  	if utils.GetEnvBool("APPLY_CACHE") {
    57  		if displayKey, ok := GetDisplayKey(name); ok {
▶   58  			cache.CacheSet(resResult, os.Getenv("CACHE_PREFIX")+displayKey)
    59  		}
    60  	}
    61  	return resResult, nil
    62  }
    63  
    64  // GetCustomFieldByName is a backward-compatible package-level wrapper used by controllers
    65  // and integration tests that call services.GetCustomFieldByName directly.
    66  func GetCustomFieldByName(name string, filter string) (*response.FBytesRootCauseResponse, *utils.RestErr) {
    67  	return CustomFieldsSvc.GetCustomFieldByName(name, filter)
    68  }
```

**Flagged source line:**

```go
			cache.CacheSet(resResult, os.Getenv("CACHE_PREFIX")+displayKey)
```

---

## 3. Root cause analysis

**errcheck rule:** function မှ return လုပ်သော `error` ကို caller မှ မစစ်ဆေးဘဲ ignore လုပ်ထားသည်။ Production မှာ disk full / network error တွေကို သိရှိနိုင်မည်မဟုတ် — silent failure ဖြစ်သည်။

---

## 4. Fix strategy

Function call ၏ error return ကို `if err :=` pattern ဖြင့် ရယူပြီး handle လုပ်မည်။

```go
if err := operation(); err != nil {
    // handle or return
    return fmt.Errorf("operation failed: %w", err)
}
```

---

## 5. Before → After preview

> Code ကို modify မလုပ်ရသေးပါ။ ဒီ section သည် proposed change ၏ preview သာ ဖြစ်သည်။

### Before

```go
			cache.CacheSet(resResult, os.Getenv("CACHE_PREFIX")+displayKey)
```

### After

```go
if err := operation(); err != nil {
    // handle or return
    return fmt.Errorf("operation failed: %w", err)
}
```

---

## 6. Possible side effects

* Error visibility တိုးသည် — behavior မပြောင်း
* Defer close error ဆိုရင် resource cleanup path မပြောင်း
* Test coverage ထည့်မည်ဆိုရင် error injection mock လိုနိုင်သည်

---

## 7. Impact analysis

**Affected file:** `frontiir/services/customfields_service.go`
**Affected package:** `frontiir/services`
**Base symbol:** `cache.CacheSet`

**Callers / references found:**

* `frontiir/services/ticket_service.go:101`
* `frontiir/services/customfieldsvalues_service.go:65`

> Verify caller behavior does not change after the fix.

---

## 8. Recommended tests

Priority order:

```bash
# 1. Same package
go test ./frontiir/services/...
make test-unit TEST_FUNC=ALL

# 2. Full lint re-check — confirm issue #3 is resolved
GOROOT=/usr/local/go golangci-lint run --config .golangci.yml ./frontiir/services/...

# 3. Full test suite
make test-unit TEST_FUNC=ALL
```

---

## 9. Acceptance criteria

- [ ] Issue #3 (`errcheck` at `frontiir/services/customfields_service.go:58`) lint error မတွေ့တော့ပါ
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
