# Golangci-lint Rule များ — Myanmar ရှင်းလင်းချက်

**ရည်ရွယ်ချက်:** Linter တစ်ခုချင်းစီ ဘာစစ်သလဲ၊ ဘာကြောင့် rule ရှိသလဲ၊ ဥပမာ၊ ပြင်နည်း

---

## အကျဉ်းချုပ် ဇယား

| Linter | စစ်ဆေးသည့်အချက် | အရေးကြီးမှု |
|---|---|---|
| `gosec` | Security vulnerabilities | 🔴 ချက်ချင်းပြင်ရ |
| `errcheck` | Error return value မစစ်ထားတာ | ⚠️ မကြာမီပြင်ရ |
| `funlen` | Function ရှည်လွန်းတာ | ⚠️ မကြာမီပြင်ရ |
| `revive` | Go style convention | 🟡 နောက်မှပြင်ရ |
| `errorlint` | Error comparison မှားတာ | 🟡 နောက်မှပြင်ရ |
| `gci` | Import grouping မမှန်တာ | 🟡 နောက်မှပြင်ရ |
| `goimports` | Import ပေါ်ပိုနေ / ပျောက်နေတာ | 🟡 နောက်မှပြင်ရ |
| `gofumpt` | Formatting strict rules | 🟡 နောက်မှပြင်ရ |
| `golines` | Line ရှည်လွန်းတာ | 🟡 နောက်မှပြင်ရ |
| `whitespace` | Extra blank lines | 🟡 နောက်မှပြင်ရ |
| `mnd` | Magic number သုံးနေတာ | 🟡 နောက်မှပြင်ရ |
| `gocritic` | Style + performance hints | 🟡 နောက်မှပြင်ရ |
| `ineffassign` | Assign လုပ်ပြီး မသုံးဘဲ | 🟡 နောက်မှပြင်ရ |

---

## 1. `gosec` — Security Issues 🔴

**ဘာစစ်သလဲ:** Go code ထဲမှာ security vulnerability ဖြစ်နိုင်သောနေရာများ

**ဘာကြောင့် rule ရှိသလဲ:** Security bug တစ်ခု production မှာ ဖြစ်ရင် customer data ထိခိုက်မှု၊ system compromise ဖြစ်နိုင်သည်

### G304 — Path Traversal

```go
// ❌ မကောင်း — user input ကို တိုက်ရိုက်သုံး
func readConfig(name string) {
    data, _ := os.ReadFile(name) // name = "../../etc/passwd" ဖြစ်နိုင်
}

// ✅ ကောင်း — validate ပြီးသုံး
func readConfig(name string) {
    clean := filepath.Clean(name)
    if !strings.HasPrefix(clean, "/allowed/dir/") {
        return fmt.Errorf("invalid path")
    }
    data, err := os.ReadFile(clean)
}
```

### G115 — Integer Overflow

```go
// ❌ မကောင်း
var count int = 99999999999
var x int32 = int32(count) // overflow ဖြစ်နိုင်

// ✅ ကောင်း
if count > math.MaxInt32 {
    return fmt.Errorf("value too large")
}
var x int32 = int32(count)
```

### G301 / G302 — File Permissions

```go
// ❌ မကောင်း
os.MkdirAll(path, 0755) // world-readable
os.OpenFile(path, os.O_CREATE, 0644)

// ✅ ကောင်း
os.MkdirAll(path, 0750) // group read only
os.OpenFile(path, os.O_CREATE, 0600) // owner only
```

### G706 — Log Injection

```go
// ❌ မကောင်း
log.Printf("user: " + userInput) // userInput မှာ newline ထည့်ပြီး fake log entries ဖန်တီးနိုင်

// ✅ ကောင်း
log.Printf("user: %q", userInput) // %q သည် escape လုပ်ပေး
```

---

## 2. `errcheck` — Unchecked Errors ⚠️

**ဘာစစ်သလဲ:** Function က error return ပေးသော်လည်း caller က ignore လုပ်ထားသောနေရာ

**ဘာကြောင့် rule ရှိသလဲ:** Error ကို ignore လုပ်ရင် operation fail ဖြစ်သောကြောင့် data corrupt ဖြစ်နိုင်၊ resource leak ဖြစ်နိုင်

```go
// ❌ မကောင်း
defer f.Close()           // Close error ကို ignore
json.Unmarshal(data, &v)  // parse fail ကို သိမ်းမထား

// ✅ ကောင်း
defer func() {
    if err := f.Close(); err != nil {
        log.Printf("close failed: %v", err)
    }
}()

if err := json.Unmarshal(data, &v); err != nil {
    return fmt.Errorf("parse: %w", err)
}
```

---

## 3. `funlen` — Function Too Long ⚠️

**ဘာစစ်သလဲ:** Function body ၏ line count နှင့် statement count

**Rule:** ≤ 80 lines / ≤ 50 statements

**ဘာကြောင့် rule ရှိသလဲ:** Function ရှည်လွန်းရင် test ရေးရ ခက်သည်၊ logic ဖတ်ရ ခက်သည်၊ single responsibility ချိုးဖောက်နေသည်

```go
// ❌ မကောင်း — အရာအားလုံး function တစ်ခုထဲ
func ProcessOrder(order Order) error {
    // validate ... (20 lines)
    // calculate ... (20 lines)
    // save to db ... (20 lines)
    // send email ... (20 lines)
    // log audit ... (10 lines)
}

// ✅ ကောင်း — တစ်ခုချင်းစီ ခွဲ
func ProcessOrder(order Order) error {
    if err := validateOrder(order); err != nil { return err }
    total := calculateTotal(order)
    if err := saveOrder(order, total); err != nil { return err }
    return notifyCustomer(order)
}
```

---

## 4. `revive` — Style Violations 🟡

**ဘာစစ်သလဲ:** Go community style convention — golint ၏ successor

**အဓိကစစ်သောအချက်များ:**

```go
// ❌ exported function မှာ comment မပါ
func GetTicket(id int) *Ticket { ... }

// ✅
// GetTicket returns a ticket by its ID.
func GetTicket(id int) *Ticket { ... }

// ❌ unused function parameter
func handle(w http.ResponseWriter, r *http.Request, ctx context.Context) {
    // ctx ကို မသုံးဘဲ
}

// ❌ stuttering name
package ticket
type TicketService struct{} // ticket.TicketService — "ticket" ထပ်နေ

// ✅
type Service struct{} // ticket.Service
```

---

## 5. `errorlint` — Error Comparison 🟡

**ဘာစစ်သလဲ:** Error ကို `==` ဖြင့် တိုက်ရိုက် compare လုပ်နေသောနေရာ

**ဘာကြောင့် rule ရှိသလဲ:** Wrapped error (`%w`) ကို `==` ဖြင့် compare လုပ်ရင် match မဖြစ်ဘဲ bug ဖြစ်သည်

```go
// ❌ မကောင်း
if err == sql.ErrNoRows { ... }           // wrapped ဖြစ်နေရင် false ပြန်
if err == context.DeadlineExceeded { ... }

// ✅ ကောင်း
if errors.Is(err, sql.ErrNoRows) { ... }           // wrapped layer အားလုံး ဖြတ်စစ်
if errors.Is(err, context.DeadlineExceeded) { ... }
```

---

## 6. `gci` — Import Grouping 🟡

**ဘာစစ်သလဲ:** Import block ၏ grouping order

**Rule:** stdlib → external packages → internal/local packages ဆိုပြီး blank line ဖြင့် ခွဲရမည်

```go
// ❌ မကောင်း — အားလုံးရောနေ
import (
    "git.frontiir.net/sa-dev/rtdatacore"
    "fmt"
    "github.com/gin-gonic/gin"
    "os"
)

// ✅ ကောင်း — group ခွဲထားသည်
import (
    "fmt"        // stdlib
    "os"

    "github.com/gin-gonic/gin"  // external

    "git.frontiir.net/sa-dev/rtdatacore"  // internal
)
```

**Auto-fix:** `gci write ./...`

---

## 7. `goimports` — Import Management 🟡

**ဘာစစ်သလဲ:** Import ပေါ်ပိုနေသောနေရာ၊ လိုအပ်သော import မပါသောနေရာ၊ gci grouping

**`gci` နဲ့ ကွာချက်:**

| | `gci` | `goimports` |
|---|---|---|
| **စစ်သောအချက်** | Group order သာ | Group order + unused + missing |
| **Auto-fix** | `gci write` | `goimports -w` |

```go
// ❌ မကောင်း — "fmt" သုံးထားသော်လည်း import မပါ
import "os"
func main() {
    fmt.Println("hello") // compile error ဖြစ်မည်
}

// ❌ မကောင်း — import ထည့်ထားသော်လည်း မသုံး
import (
    "fmt"
    "os"  // မသုံးဘဲ
)
```

**Auto-fix:** `goimports -w ./...`

---

## 8. `gofumpt` — Strict Formatting 🟡

**ဘာစစ်သလဲ:** `gofmt` ထက် stricter formatting rules

**`gofmt` နဲ့ ကွာချက်:** Extra blank lines, multiline composite literals, function grouping

```go
// ❌ gofumpt မကောင်း — function body ထဲ ပထမဆုံး line blank
func hello() {

    fmt.Println("hi")
}

// ✅ gofumpt ကောင်း
func hello() {
    fmt.Println("hi")
}

// ❌ မကောင်း — struct literal မှာ last element ၏ trailing comma မပါ
p := Person{
    Name: "Ko Ko",
    Age: 30
}

// ✅ ကောင်း
p := Person{
    Name: "Ko Ko",
    Age:  30,
}
```

**Auto-fix:** `gofumpt -w ./...`

---

## 9. `golines` — Line Length 🟡

**ဘာစစ်သလဲ:** Line တစ်ကြောင်း ၏ character count (default max: 120)

**ဘာကြောင့် rule ရှိသလဲ:** ရှည်သော line တွေသည် horizontal scroll လိုစေသည်၊ code review မှာ ဖတ်ရ ခက်သည်

```go
// ❌ မကောင်း — line ရှည်လွန်း
func (s *TicketService) GetTicketsByCpeIdWithPaginationAndFilters(ctx context.Context, cpeId string, page int, pageSize int, filters map[string]interface{}) ([]*Ticket, error) {

// ✅ ကောင်း — wrap လုပ်ထားသည်
func (s *TicketService) GetTicketsByCpeIdWithPaginationAndFilters(
    ctx context.Context,
    cpeId string,
    page int,
    pageSize int,
    filters map[string]interface{},
) ([]*Ticket, error) {
```

**Auto-fix:** `golines -w ./...`

---

## 10. `whitespace` — Extra Blank Lines 🟡

**ဘာစစ်သလဲ:** Function / if / for block တွေ၏ ထိပ်နှင့် အောက်ဆုံးမှာ ပိုနေသော blank line

```go
// ❌ မကောင်း — function ထဲ ထိပ်မှာ blank line ပို
func process() {

    doWork()
}

// ❌ မကောင်း — if block ၏ ပထမနှင့် နောက်ဆုံး blank line ပို
if err != nil {

    return err

}

// ✅ ကောင်း
func process() {
    doWork()
}

if err != nil {
    return err
}
```

**Auto-fix:** `golangci-lint run --fix`

---

## 11. `mnd` — Magic Numbers 🟡

**ဘာစစ်သလဲ:** Code ထဲမှာ ရှင်းချက်မပါဘဲ သုံးနေသော literal number (magic number)

**ဘာကြောင့် rule ရှိသလဲ:** `42`, `86400`, `3` ဆိုသော number တွေ ဘာကိုဆိုလိုသလဲ ဖတ်ကြည့်ရုံနဲ့ မသိနိုင်

```go
// ❌ မကောင်း — 86400 ဆိုတာ ဘာကို ဆိုလိုသလဲ?
if age > 18 {
    token.Expiry = time.Now().Add(86400 * time.Second)
}

// ✅ ကောင်း — named constant သုံး
const (
    MinAdultAge     = 18
    TokenTTLSeconds = 86400 // 24 hours
)

if age > MinAdultAge {
    token.Expiry = time.Now().Add(TokenTTLSeconds * time.Second)
}
```

---

## 12. `gocritic` — Style + Performance Hints 🟡

**ဘာစစ်သလဲ:** Performance, style, diagnostic အမျိုးအစား ၃ ခုပါသော opinionated checks

**အဓိကဥပမာများ:**

```go
// ❌ appendAssign — append result ကို တူသော variable ပြန်မသိမ်း
var s []string
append(s, "a") // result ကို discard လုပ်နေ

// ✅
s = append(s, "a")

// ❌ sloppyLen — len() ကို 0 နဲ့ > မသုံးဘဲ != သုံးနေ
if len(items) != 0 { ... }

// ✅
if len(items) > 0 { ... }

// ❌ rangeValCopy — range မှာ large struct ကို value copy လုပ်နေ
for _, ticket := range tickets { // tickets []Ticket — copy ဖြစ်နေ
    process(ticket)
}

// ✅
for i := range tickets {
    process(&tickets[i]) // pointer သုံး
}
```

---

## 13. `ineffassign` — Ineffectual Assignments 🟡

**ဘာစစ်သလဲ:** Variable ကို assign လုပ်ထားသော်လည်း နောက်ထပ် read မလုပ်မီ overwrite သို့မဟုတ် return ဖြစ်သွားသောနေရာ

**ဘာကြောင့် rule ရှိသလဲ:** Dead code ဖြစ်သောကြောင့် — ဖတ်သောသူကို လှည့်ဖြားသည်

```go
// ❌ မကောင်း
func getStatus() string {
    status := "pending"   // assign လုပ်ထား
    status = "active"     // ဒီနေရာမှာ overwrite — "pending" ကို ဘယ်တော့မှ မသုံးဘဲ
    return status
}

// ❌ မကောင်း
err := doA()
err = doB() // doA ၏ err ကို check မလုပ်ဘဲ overwrite

// ✅ ကောင်း
if err := doA(); err != nil {
    return err
}
if err := doB(); err != nil {
    return err
}
```

---

## Auto-fix လုပ်နိုင်သော Linter များ

| Linter | Auto-fix command |
|---|---|
| `gci` | `gci write ./...` |
| `goimports` | `goimports -w ./...` |
| `gofumpt` | `gofumpt -w ./...` |
| `golines` | `golines -w ./...` |
| `whitespace` | `golangci-lint run --fix` |
| `misspell` | `golangci-lint run --fix` |

**တစ်ကြိမ်တည်း auto-fix လုပ်ရန်:**
```bash
golangci-lint run --config .golangci.yml --fix
```

> Manual ပြင်ရမည့်အရာ: `gosec`, `errcheck`, `funlen`, `mnd`, `gocritic`, `ineffassign`, `errorlint`

---

## Priority အလိုက် ပြင်ဆင်ရမည့် အစီအစဉ်

```
Phase 1 — ချက်ချင်း  : gosec (security)
Phase 2 — ဦးစားပေး  : errcheck, funlen, errorlint
Phase 3 — Auto-fix   : gci, goimports, gofumpt, golines, whitespace
Phase 4 — နောက်မှ    : mnd, gocritic, revive, ineffassign
```
