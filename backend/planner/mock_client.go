package planner

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// MockClient is a deterministic, no-network AIClient. It synthesizes a
// plausible plan from the real issue data actually available, so the
// rest of the pipeline (caching, approval, apply -- M3+) can be built and
// tested against real shapes without a real ANTHROPIC_API_KEY or cost.
type MockClient struct{}

// NewMockClient returns a ready-to-use MockClient.
func NewMockClient() *MockClient { return &MockClient{} }

// GeneratePlan synthesizes a deterministic fix plan from req's issues.
func (c *MockClient) GeneratePlan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	if len(req.Issues) == 0 {
		return PlanResult{}, errors.New("mock AI client: no issues in request")
	}

	first := req.Issues[0]

	fileSet := make(map[string]struct{}, len(req.Issues))
	riskLevel := "low"
	for _, iss := range req.Issues {
		fileSet[iss.FilePath] = struct{}{}
		if strings.Contains(strings.ToLower(iss.Linter), "sec") {
			riskLevel = "high"
		}
	}
	files := make([]string, 0, len(fileSet))
	for f := range fileSet {
		files = append(files, f)
	}
	sort.Strings(files)

	text := synthesizeText(first)
	rt := buildRuleTemplate(first)

	return PlanResult{
		RootCause:         text.rootCause,
		CurrentBehavior:   text.currentBehavior,
		RecommendedFix:    text.recommendedFix,
		RootCauseMy:       text.rootCauseMy,
		CurrentBehaviorMy: text.currentBehaviorMy,
		RecommendedFixMy:  text.recommendedFixMy,
		RiskLevel:         riskLevel,
		BreakingChange:    false,
		FilesImpacted:     files,
		TestPlan: []TestCase{
			{
				Given: "the fix described above has been applied",
				When:  fmt.Sprintf("golangci-lint is re-run on %s", first.FilePath),
				Then: fmt.Sprintf(
					"the %s issue on line %d no longer appears",
					first.Rule,
					first.Line,
				),
			},
		},
		GeneratedBy: "mock-v1",

		// Sections added 2026-08-19 to match before-fixed/*.md's depth --
		// see planner.PlanResult's doc comments for what each maps to.
		CodeContext:     buildCodeContext(first.RepoRef, first.FilePath, first.Line),
		FixStrategyCode: rt.fixStrategyCode,
		BeforeSnippet:   flaggedLine(first.RepoRef, first.FilePath, first.Line),
		AfterSnippet:    rt.fixStrategyCode,
		SideEffects:     rt.sideEffects,
		ImpactAnalysis:  analyzeImpact(ctx, first.RepoRef, first.FilePath, first.Line),
		RecommendedTestCommands: []string{
			goTestCommand(first.FilePath),
			"make test-unit TEST_FUNC=ALL",
		},
		AcceptanceCriteria: acceptanceCriteria(first),
	}, nil
}

type bilingualPlanText struct {
	rootCause, currentBehavior, recommendedFix       string
	rootCauseMy, currentBehaviorMy, recommendedFixMy string
}

// synthesizeText builds the English and Burmese RootCause/CurrentBehavior/
// RecommendedFix sentences for issue, split out of GeneratePlan to keep it
// short -- both languages use the same deterministic-template approach.
func synthesizeText(issue IssueContext) bilingualPlanText {
	return bilingualPlanText{
		rootCause: fmt.Sprintf(
			"%s (%s) at %s:%d — %s",
			issue.Linter, issue.Rule, issue.FilePath, issue.Line, issue.Message,
		),
		currentBehavior: fmt.Sprintf(
			"The %s condition reported by %s is present as-is; no fix has been applied yet.",
			issue.Rule, issue.Linter,
		),
		recommendedFix: fmt.Sprintf(
			"Address the %s finding reported by %s.",
			issue.Rule, issue.Linter,
		),
		rootCauseMy: fmt.Sprintf(
			"%s (%s) — %s:%d တွင် %s",
			issue.Linter, issue.Rule, issue.FilePath, issue.Line, issue.Message,
		),
		currentBehaviorMy: fmt.Sprintf(
			"%s မှ တွေ့ရှိသော %s အခြေအနေကို ယခုအထိ ပြင်ဆင်ခြင်း မပြုရသေးပါ။",
			issue.Linter, issue.Rule,
		),
		recommendedFixMy: fmt.Sprintf(
			"%s မှ တွေ့ရှိသော %s ပြဿနာကို ဖြေရှင်းပါ။",
			issue.Linter, issue.Rule,
		),
	}
}

// goTestCommand builds the "same package" test command (before-fixed/*.md
// §8 priority 1) for filePath. filepath.Dir returns "." for a root-level
// file, which would otherwise produce the malformed "go test ././...".
func goTestCommand(filePath string) string {
	dir := filepath.Dir(filePath)
	if dir == "." {
		return "go test ./..."
	}
	return fmt.Sprintf("go test ./%s/...", dir)
}
