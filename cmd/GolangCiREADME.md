# `.golangci.yml` — Linter Rules ရှင်းလင်းချက် (မြန်မာဘာသာ)

**golangci-lint version:** v2.x  
**config file:** `.golangci.yml`  
**ရည်ရွယ်ချက်:** ဤ project မှာ enable လုပ်ထားသော linter တစ်ခုချင်းစီ ဘာကိုစစ်ဆေးသည်၊ ဘာကြောင့် လိုအပ်သည် ဆိုတာ ရှင်းပြသော reference document

---

## Run configuration

```yaml
run:
  timeout: 5m
```

`golangci-lint run` command ကို အများဆုံး **5 မိနစ်** ကြာခွင့်ပေးသည်။ ဒီထက် ကြာသွားရင် timeout ဖြစ်ပြီး fail ဖြစ်မည်။

---

## Linter အုပ်စုများ

### အုပ်စု 1 — Core (မဖြစ်မနေ စစ်ရသည်)

#### `errcheck` — Error မစစ်ထားတာ စစ်ဆေးသည်

Go မှာ function တိုင်းနီးပါး `error` return ပြန်သည်။ ထို `error` ကို **ignore** လုပ်ပါက runtime မှာ silent failure ဖြစ်နိုင်သည် — program ပျက်သွားနိုင်ပေမဲ့ ဘယ်နေရာမှာ ဆိုတာ မသိဘဲ ဆက်သွားနိုင်သည်။

```go
// BAD — errcheck ကျရှုံးသည်
f.Close()

// GOOD — error စစ်ထားသည်
if err := f.Close(); err != nil {
    log.Printf("close failed: %v", err)
}
```

---

#### `govet` — Suspicious code pattern စစ်ဆေးသည်

Go compiler ကိုယ်တိုင် detect မလုပ်သော **ဖြစ်နိုင်ချေရှိသော bug** များကို စစ်သည်။

စစ်ဆေးသောနေရာများ —

| ပြဿနာ | ဥပမာ |
|---|---|
| `printf` format string နှင့် argument မကိုက် | `fmt.Printf("%d", "hello")` |
| Loop variable ကို goroutine ထဲ capture | `go func() { use(x) }()` in loop |
| Struct copy lock | `mu sync.Mutex` ပါသော struct ကို copy |
| Unreachable code | `return` ပြီးနောက် statement |

```go
// BAD — format string bug
fmt.Printf("%d", "not a number")

// BAD — loop variable capture
for _, v := range items {
    go func() { process(v) }() // v always captures last value
}
```

---

#### `staticcheck` — Comprehensive static analysis

Go ecosystem မှာ အကြီးကြီးသော static analyzer — compiler ထက် ပိုနက်နဲစွာ စစ်သည်။

စစ်ဆေးသောနေရာများ —

| Category | ဥပမာ |
|---|---|
| Deprecated API သုံးနေတာ | `ioutil.ReadFile` (Go 1.16 ကတည်းက deprecated) |
| Dead code | ဘယ်တော့မှ execute မဖြစ်နိုင်သော branch |
| Incorrect API usage | `strings.Replace` ကို `-1` မဟုတ်ဘဲ မှားသုံးတာ |
| Inefficient code | `len(x) == 0` အစား `x == nil` |
| Race conditions | concurrent access pattern |

---

#### `ineffassign` — အသုံးမဝင်သော assignment စစ်ဆေးသည်

Variable တစ်ခုကို value assign လုပ်ပြီး **ထို value ကို ဘယ်တော့မှ မသုံးဘဲ** နောက်ထပ် assign ပြန်သုံးသောကြောင့် ပထမ assignment က dead code ဖြစ်နေသည်ကို စစ်သည်။

```go
// BAD — ပထမ err assign က အသုံးမဝင်
err := doA()
err = doB() // ပထမ err ကို ဘယ်တော့မှ မစစ်ဘဲ overwrite

// GOOD
if err := doA(); err != nil {
    return err
}
if err := doB(); err != nil {
    return err
}
```

---

### အုပ်စု 2 — Style & Consistency

#### `misspell` — Typo (စာလုံးပေါင်းမှား) စစ်ဆေးသည်

**Setting:** `locale: US` — US English စာလုံးပေါင်းစည်းမျဉ်း သုံးသည်

Code ထဲ identifier, comment, string literal မှာ **အဖြစ်များသော English typo** များ စစ်သည်။ Auto-fix လုပ်နိုင်သည်။

```go
// BAD
// initialised the config  ← "initialised" သည် British, US: "initialized"

// GOOD
// initialized the config
```

> `golangci-lint run --fix` သုံးပါက auto-correct လုပ်ပေးသည်။

---

#### `revive` — golint replacement, extensible style linter

`golint` ကို replace လုပ်ထားသော ပိုစွမ်းဆောင်ရည်ကောင်းသော linter — Go community style convention လိုက်နာမှု စစ်သည်။

**ဤ project ရဲ့ settings:**

```yaml
revive:
  rules:
    - name: exported
      disabled: true          # exported function မှာ comment မရှိသော warning — disable လုပ်ထားသည် (existing codebase မှာ noisy)
    - name: unused-parameter
      severity: warning       # Function parameter ကို မသုံးတာ warning ပေးသည်
    - name: cognitive-complexity
      arguments: [20]         # Function တစ်ခုချင်း cognitive complexity ≤ 20 ဖြစ်ရမည်
```

**Cognitive complexity** ဆိုသည်မှာ code ကို **လူ** တစ်ယောက် ဖတ်ပြီး နားလည်ရ ခက်လောက်တဲ့ပမာဏ — `if`, `for`, `switch`, logical operator (`&&`, `||`) တွေ တိုးလာသည်နှင့်အမျှ တက်သည်။

```go
// BAD — unused parameter
func process(id int, name string) int { // name မသုံးဘဲ
    return id * 2
}

// GOOD
func process(id int) int {
    return id * 2
}
```

---

### အုပ်စု 3 — Bugs & Reliability

#### `gosec` — Security vulnerability စစ်ဆေးသည်

Go code ထဲ **security issue** များကို စစ်သည်။ OWASP / CWE standard ကိုရည်ညွှန်းသည်။

**ဤ project မှာ enable လုပ်ထားသော rule ID များ:**

| Rule ID | စစ်ဆေးသောအချက် | ဘာကြောင့် အရေးကြီးတာ |
|---|---|---|
| G301 | Directory permission > 0750 | World-readable directory — sensitive data ထိခိုက်နိုင် |
| G302 | File permission > 0600 | Group/world ဖတ်နိုင် — credential file ဖြစ်ရင် အန္တရာယ်ရှိ |
| G304 | File path variable မစစ်ဘဲ open | Path traversal attack (CWE-22) — `../../etc/passwd` ကိုဖတ်နိုင် |
| G115 | Integer overflow conversion | `int` → `int32` / `uint` မှာ overflow ဖြစ်နိုင် |
| G117 | Secret field JSON serialization | `api_key` ဟု JSON serialize လုပ်သော field — credential leak ဖြစ်နိုင် |
| G706 | Log injection | User input ကို sanitize မလုပ်ဘဲ log ထည့်သောကြောင့် log forging ဖြစ်နိုင် |

**G104 (unhandled errors) ကို exclude လုပ်ထားသည်** — `errcheck` linter ကဆောင်ရွက်ပြီးသားဖြစ်သောကြောင့် duplicated မဖြစ်စေရန်။

```go
// BAD — G304: path validate မလုပ်
data, _ := os.ReadFile(userInputPath)

// GOOD — path clean + validate လုပ်
clean := filepath.Clean(userInputPath)
if !strings.HasPrefix(clean, allowedDir) {
    return fmt.Errorf("path not allowed: %s", clean)
}
data, err := os.ReadFile(clean)
```

---

#### `bodyclose` — HTTP response body မပိတ်တာ စစ်ဆေးသည်

`http.Get()` / `http.Do()` ဖြင့် HTTP request ပေးပြီးနောက် **`resp.Body.Close()` ကို defer မလုပ်ဘဲ** ချန်ထားပါက memory/connection leak ဖြစ်သည်ကို စစ်သည်။

```go
// BAD — body မပိတ်ဘဲ
resp, err := http.Get(url)
if err != nil {
    return err
}
data, _ := io.ReadAll(resp.Body)

// GOOD — defer close
resp, err := http.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()
data, err := io.ReadAll(resp.Body)
```

---

#### `nilerr` — nil error မဟုတ်ဘဲ nil return စစ်ဆေးသည်

`err != nil` စစ်ပြီးနောက် `return nil, nil` ပြန်သောကြောင့် error ကို caller ကို မပို့ဘဲ **ဖုံးကွယ်နေသည်** ကို စစ်သည်။

```go
// BAD — error ကို silent ဖုံးကွယ်
if err != nil {
    return nil, nil // caller မသိ
}

// GOOD — error ကို caller ကိုပို့
if err != nil {
    return nil, fmt.Errorf("query failed: %w", err)
}
```

---

### အုပ်စု 4 — Complexity

#### `gocyclo` — Cyclomatic complexity စစ်ဆေးသည်

**Setting:** `min-complexity: 15`

**Cyclomatic complexity** ဆိုသည်မှာ function တစ်ခုထဲမှာ **logic path** (decision branch) ဘယ်နှစ်ခုရှိသည် ဆိုသော ကိန်းဂဏန်း — `if`, `else`, `for`, `switch case`, `&&`, `||` တိုင်းတိုးသည်။

| Complexity | ဆိုလိုသည် |
|---|---|
| 1–10 | ရိုးရှင်း၊ test ရေးရ လွယ် |
| 11–15 | အနည်းငယ်ရှုပ်သည် |
| **> 15** | **ရှုပ်လွန်းသည် — function ခွဲပါ** |

```go
// BAD — if/else chain ရှည်နေ → complexity တက်
func process(x int) string {
    if x == 1 {
        ...
    } else if x == 2 {
        ...
    } else if x == 3 {
        ...
    } // ဆက်သွားတိုင်း complexity ++ ဖြစ်

// GOOD — lookup table / helper function ခွဲသုံး
func process(x int) string {
    handlers := map[int]func() string{
        1: handleOne,
        2: handleTwo,
        3: handleThree,
    }
    if fn, ok := handlers[x]; ok {
        return fn()
    }
    return ""
}
```

---

#### `funlen` — Function length စစ်ဆေးသည်

**Settings:**

```yaml
funlen:
  lines: 80       # function တစ်ခု ≤ 80 line ဖြစ်ရမည်
  statements: 50  # statement ≤ 50 ခု ဖြစ်ရမည်
```

Function တစ်ခု ကြီးလာလေ —
- ဖတ်ရ ခက်လာသည်
- Bug ဝင်ရ လွယ်လာသည်
- Test ရေးရ ခက်လာသည်

> **Test file မှာ exempt:** `_test.go` file ထဲ test function ကို `funlen` နှင့် `gocyclo` မစစ်ဘဲ ချန်ထားသည် (test case တွေ ရှည်တတ်သောကြောင့်)

---

#### `nestif` — Deeply nested if-block စစ်ဆေးသည်

**Setting:** `min-complexity: 5`

`if` block တွေ တွင်းထဲ တွင်းထဲ ဆင်းသွားသောအခါ code ဖတ်ရ ခက်လာသည်ကို စစ်သည်။

```go
// BAD — 3 level nesting
if a {
    if b {
        if c {
            doThing()
        }
    }
}

// GOOD — early return (guard clause)
if !a {
    return
}
if !b {
    return
}
if !c {
    return
}
doThing()
```

---

## Exclusions (စစ်ဆေးမှုမှ ချန်ထားသော နေရာများ)

```yaml
exclusions:
  generated: lax          # Auto-generated code မှာ lax mode (warnings သာ)
  rules:
    - path: _test\.go
      linters:
        - funlen          # Test file မှာ funlen မစစ်
        - gocyclo         # Test file မှာ gocyclo မစစ်
  paths:
    - vendor              # vendor/ directory စစ်မည်မဟုတ်
    - third_party         # third_party/ directory စစ်မည်မဟုတ်
    - mocks               # mocks/ directory စစ်မည်မဟုတ်
```

---

## Issues limits

```yaml
issues:
  max-issues-per-linter: 50   # Linter တစ်ခုကြောင့် warning ၅၀ ထက် မပြ
  max-same-issues: 10         # တူညီသော issue ကို ၁၀ ခုသာ ပြသည် (console overflow မဖြစ်ရန်)
```

---

## Linter အားလုံး အကျဉ်းချုပ်

| Linter | စစ်ဆေးသောအချက် | Setting |
|---|---|---|
| `errcheck` | Error ignore | — |
| `govet` | Suspicious constructs | — |
| `staticcheck` | Comprehensive static analysis | — |
| `ineffassign` | Dead assignment | — |
| `misspell` | Typo | `locale: US` |
| `revive` | Style / cognitive complexity | `complexity ≤ 20` |
| `gosec` | Security vulnerability | G104 exclude |
| `bodyclose` | HTTP body leak | — |
| `nilerr` | Nil error masking | — |
| `gocyclo` | Cyclomatic complexity | `≤ 15` |
| `funlen` | Function length | `≤ 80 lines / 50 statements` |
| `nestif` | Nested if depth | `≤ complexity 5` |

---

## Command များ

```bash
# Lint စစ်ကြည့်ရန်
make lint

# Lint စစ်ကြည့်ရန် (config file ဖြင့် တိုက်ရိုက်)
golangci-lint run --config .golangci.yml

# Auto-fix လုပ်နိုင်သောနေရာများ ပြင်ရန် (misspell, gofmt, goimports)
golangci-lint run --config .golangci.yml --fix

# Linter list ကြည့်ရန်
golangci-lint linters
```
