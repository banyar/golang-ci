# Lint fix plan — issue #50: `gci`

**Generated:** 2026-07-10
**Source report:** `golangci/linter-report/golangci-default/2026-07-10_16-20-00.json`

---

## 1. Issue summary

| Field | Value |
|---|---|
| Issue number | 50 |
| Linter | `gci` |
| File | `redmine/redmine_get_issue.go` |
| Line | 111 |
| Column | 1 |
| Message | File is not properly formatted |

---

## 2. Original code context

Line 111 (±10) — marker `▶` = flagged line.

```go
   101  	Name     string `json:"name"`     // field name (e.g. "status_id", "assigned_to_id")
   102  	OldValue string `json:"old_value"`
   103  	NewValue string `json:"new_value"`
   104  }
   105  
   106  // Journal - Issue ၌ ဖြစ်ပေါ်သော activity တစ်ခု
   107  // Comment ထည့်သည် သို့မဟုတ် field ပြောင်းသည် တိုင်း journal entry တစ်ခု ဖြစ်ပေါ်သည်
   108  type Journal struct {
   109  	ID           int             `json:"id"`
   110  	User         RedmineRef      `json:"user"`
▶  111  	Notes        string          `json:"notes"`         // comment text; field change ကိုသာ ပြုလုပ်ပါက empty
   112  	CreatedOn    time.Time       `json:"created_on"`
   113  	PrivateNotes bool            `json:"private_notes"` // true ဖြစ်ပါက member only မြင်ရသည်
   114  	Details      []JournalDetail `json:"details"`       // ပြောင်းလဲသော fields list
   115  }
   116  
   117  // Attachment - Issue ၌ attach လုပ်ထားသော file တစ်ခု
   118  type Attachment struct {
   119  	ID          int        `json:"id"`
   120  	Filename    string     `json:"filename"`
   121  	Filesize    int64      `json:"filesize"`    // bytes
```

**Flagged source line:**

```go
	Notes        string          `json:"notes"`         // comment text; field change ကိုသာ ပြုလုပ်ပါက empty
```

---

## 3. Root cause analysis

**gci rule:** `File is not properly formatted` — linter documentation ကြည့်ပြီး root cause စစ်ဆေးပါ။

---

## 4. Fix strategy

gci documentation ကို ကြည့်ပြီး fix pattern ရွေးချယ်မည်။

```go
// TODO: apply fix per gci rule
```

---

## 5. Before → After preview

> Code ကို modify မလုပ်ရသေးပါ။ ဒီ section သည် proposed change ၏ preview သာ ဖြစ်သည်။

### Before

```go
	Notes        string          `json:"notes"`         // comment text; field change ကိုသာ ပြုလုပ်ပါက empty
```

### After

```go
// TODO: apply fix per gci rule
```

---

## 6. Possible side effects

* Unknown — code review လိုအပ်သည်

---

## 7. Impact analysis

**Affected file:** `redmine/redmine_get_issue.go`
**Affected package:** `redmine`
**Base symbol:** `redmine_get_issue`

**Callers / references found:**

* No external callers found — change is local to `redmine`

> Verify caller behavior does not change after the fix.

---

## 8. Recommended tests

Priority order:

```bash
# 1. Same package
go test ./redmine/...
make test-unit TEST_FUNC=ALL

# 2. Full lint re-check — confirm issue #50 is resolved
GOROOT=/usr/local/go golangci-lint run --config .golangci.yml ./redmine/...

# 3. Full test suite
make test-unit TEST_FUNC=ALL
```

---

## 9. Acceptance criteria

- [ ] Issue #50 (`gci` at `redmine/redmine_get_issue.go:111`) lint error မတွေ့တော့ပါ
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
