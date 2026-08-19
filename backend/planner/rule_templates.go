package planner

import (
	"fmt"
	"strings"
)

// ruleTemplate is FixStrategyCode + SideEffects for one linter (or, for
// gosec, one G-code sub-rule) -- before-fixed/*.md §4/§6. Same
// enumerated-rule-based approach already established in
// cmd/golangci-report.sh's JQ_REASON/JQ_REASON_MY embedded jq functions,
// just in Go instead of jq.
type ruleTemplate struct {
	fixStrategyCode string
	sideEffects     []string
}

// acceptanceCriteria is before-fixed/*.md §9 -- one fixed checklist
// regardless of linter, per cmd/lint-fixed-plan.md's own spec ("Plan
// complete ဖြစ်ရန် အောက်ပါ အားလုံး ပြည့်မီရမည်").
func acceptanceCriteria(issue IssueContext) []string {
	return []string{
		fmt.Sprintf(
			"Issue (%s at %s:%d) lint error မတွေ့တော့ပါ",
			issue.Linter,
			issue.FilePath,
			issue.Line,
		),
		"golangci-lint run မှာ new issue မတိုးပါ",
		"make test-unit TEST_FUNC=ALL pass ဖြစ်ပါသည်",
		"Business logic ပြောင်းလဲမှု မရှိပါ",
		"//nolint directive မပါဝင်ပါ",
	}
}

// buildRuleTemplate dispatches on issue.Linter (and, for gosec, the G-code
// in issue.Rule/Message) to a concrete fix template. Falls back to a
// generic, honest "review manually" template for any linter not covered
// -- never fabricates a plausible-looking fix for a rule this function
// doesn't actually know how to fix.
func buildRuleTemplate(issue IssueContext) ruleTemplate {
	switch issue.Linter {
	case "errcheck":
		return ruleTemplate{
			fixStrategyCode: "if err := <call>; err != nil {\n" +
				"    return err\n" +
				"}",
			sideEffects: []string{
				"Callers must now handle the returned error explicitly",
				"Function signature may need an error return added if it doesn't already have one",
			},
		}
	case "gosec":
		return gosecTemplate(issue)
	case "revive":
		return ruleTemplate{
			fixStrategyCode: "// " + issue.Message + "\n(add a doc comment above the exported identifier)",
			sideEffects:     []string{"None — comment-only change"},
		}
	case "gocyclo", "nestif":
		return ruleTemplate{
			fixStrategyCode: "// Extract the nested/complex block above into a separate helper function.",
			sideEffects: []string{
				"Function behavior must remain identical after extraction",
				"Add/keep unit tests covering every branch moved into the helper",
			},
		}
	case "ineffassign":
		return ruleTemplate{
			fixStrategyCode: "// Remove the ineffective assignment, or use `_ = <expr>` if the discard is intentional.",
			sideEffects:     []string{"None expected if the assignment was truly unused"},
		}
	case "misspell":
		return ruleTemplate{
			fixStrategyCode: "// Correct the misspelled word in place.",
			sideEffects:     []string{"None — text-only change"},
		}
	default:
		return ruleTemplate{
			fixStrategyCode: fmt.Sprintf(
				"// No fix template known for %s -- review the %s finding manually.",
				issue.Linter,
				issue.Rule,
			),
			sideEffects: []string{"Unknown — review manually before applying"},
		}
	}
}

// gosecTemplate covers the G-codes cmd/golangci-report.sh's own
// JQ_REASON/JQ_REASON_MY already document (same set, so a plan and a lint
// report explain a given gosec finding consistently).
func gosecTemplate(issue IssueContext) ruleTemplate {
	msg := issue.Rule + " " + issue.Message
	switch {
	case strings.Contains(msg, "G304"):
		return ruleTemplate{
			fixStrategyCode: "clean := filepath.Clean(inputPath)\n" +
				"allowedBase := filepath.Clean(baseDir)\n" +
				"if !strings.HasPrefix(clean, allowedBase+string(filepath.Separator)) {\n" +
				"    return fmt.Errorf(\"path outside allowed directory: %s\", clean)\n" +
				"}\n" +
				"data, err := os.ReadFile(clean)",
			sideEffects: []string{
				"Callers passing a path outside the allowed base directory will now get an error instead of silently reading the file",
				"Config file loaders should verify their configured paths still fall under the allowed base",
			},
		}
	case strings.Contains(msg, "G115"):
		return ruleTemplate{
			fixStrategyCode: "if v > math.MaxInt32 || v < math.MinInt32 {\n" +
				"    return fmt.Errorf(\"value %d out of int32 range\", v)\n" +
				"}\n" +
				"safe := int32(v)",
			sideEffects: []string{
				"Callers passing an out-of-range value now get an explicit error instead of a silent overflow",
			},
		}
	case strings.Contains(msg, "G301"), strings.Contains(msg, "G302"):
		return ruleTemplate{
			fixStrategyCode: "os.MkdirAll(dir, 0750) // or os.WriteFile(path, data, 0600)",
			sideEffects: []string{
				"Existing files/directories created with looser permissions are not retroactively fixed",
			},
		}
	case strings.Contains(msg, "G706"):
		return ruleTemplate{
			fixStrategyCode: "logger.Info(\"event\", zap.String(\"user_input\", sanitize(userInput)))",
			sideEffects: []string{
				"Log output format changes -- update any log-scraping/alerting rules that parse this line",
			},
		}
	default:
		return ruleTemplate{
			fixStrategyCode: fmt.Sprintf(
				"// Address the gosec %s finding: %s",
				issue.Rule,
				issue.Message,
			),
			sideEffects: []string{
				"Unknown — review the specific gosec rule's documentation before applying",
			},
		}
	}
}
