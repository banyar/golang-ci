package parser

import "strings"

// linterReasonsMy maps a linter name to its Burmese "why this breaks the
// rule" explanation. Ported from cmd/golangci-report.sh's JQ_REASON_MY jq
// function (reason($l;$t)) -- same wording, so the UI's explanation matches
// what the CLI's -my.md report already says. gosec is handled separately
// by reasonGosecMy, since it branches further on the rule code within Text.
var linterReasonsMy = map[string]string{
	"errcheck":    "Error ကို ignore လုပ်ထား — production တွင် silent failure",
	"gocyclo":     "Complexity 15 ကျော် — test ရေးရ ခက်",
	"funlen":      "Function 80 lines ကျော် — သေးငယ်သော function ခွဲပါ",
	"misspell":    "စာလုံးပေါင်းမှားနေ — searchability ကျဆင်း",
	"nestif":      "Nesting ရှုပ်လွန်း — early-return pattern သုံးပါ",
	"ineffassign": "Assign လုပ်ထားသော value ကို မသုံးဘဲ — dead code",
	"revive":      "Go style convention မလိုက်နာ",
	"staticcheck": "Static bug သို့မဟုတ် deprecated API",
	"govet":       "go vet suspect construct — latent bug ဖြစ်နိုင်",
	"bodyclose":   "HTTP body မပိတ်ထား — load မြင့်လာလျှင် connection leak",
	"nilerr":      "Error path မှာ nil ပြန် — error context ဆုံးရှုံး",
}

// reasonMy returns a Burmese "why this breaks the rule" explanation for an
// issue reported by linter, given its raw golangci-lint text.
func reasonMy(linter, text string) string {
	if linter == "gosec" {
		return reasonGosecMy(text)
	}
	if reason, ok := linterReasonsMy[linter]; ok {
		return reason
	}
	return "Linting rule ချိုးဖောက် — linter docs ကြည့်ပါ"
}

// gosecReasonsMy maps a gosec rule code (found as a substring of Text) to
// its Burmese explanation, checked in this fixed order.
var gosecReasonsMy = []struct {
	code   string
	reason string
}{
	{"G304", "Path ကို validate မလုပ်ဘဲ file I/O — path traversal ဖြစ်နိုင်"},
	{"G115", "int→int32 စစ်မထား — runtime overflow ဖြစ်နိုင်"},
	{"G301", "Directory permission 0750 ထက်ကျယ် — world-readable"},
	{"G302", "File permission 0600 ထက်ကျယ် — group/world ဖတ်နိုင်"},
	{"G706", "Sanitize မလုပ်ဘဲ log ထည့် — log injection ဖြစ်နိုင်"},
	{"G117", "Secret-like JSON key — credential ထိခိုက်နိုင်"},
}

func reasonGosecMy(text string) string {
	for _, r := range gosecReasonsMy {
		if strings.Contains(text, r.code) {
			return r.reason
		}
	}
	return "Security rule ချိုးဖောက် — gosec docs ကြည့်ပါ"
}
