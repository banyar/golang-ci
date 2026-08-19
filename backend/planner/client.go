// Package planner builds AI-generated fix plans for batches of LintIssue
// rows. See golangci/plans/12-component-design.md's "Plan Service" role.
package planner

import "context"

// IssueContext is the per-issue data an AIClient needs to reason about a
// fix -- everything actually available on a persisted LintIssue row, plus
// the scan's RepoRef (needed to read real source for code context/caller
// analysis -- see FulfillPlan, which loads this from the parent LintScan).
type IssueContext struct {
	FilePath string
	Line     int
	Column   int
	Linter   string
	Rule     string
	Message  string
	RepoRef  string
}

// PlanRequest batches one or more issues (from the same scan) into a
// single fix-plan request.
type PlanRequest struct {
	Issues []IssueContext
}

// TestCase is one Given/When/Then row of a FixPlan's test plan.
type TestCase struct {
	Given string `json:"given"`
	When  string `json:"when"`
	Then  string `json:"then"`
}

// ImpactInfo is the "Impact analysis" section: which symbol the fix
// touches and who calls it, mirroring cmd/lint-fixed-plan.md Rule 7's
// AST-based caller analysis (see planner/impact.go).
type ImpactInfo struct {
	AffectedFile    string   `json:"affected_file"`
	AffectedPackage string   `json:"affected_package"`
	AffectedSymbol  string   `json:"affected_symbol"`
	Callers         []string `json:"callers"`
}

// PlanResult is an AI-generated fix plan, matching FixPlan's
// stakeholder-visible fields (04-prd.md FR-4 / 05-ui-design.md's Plan
// Viewer, extended 2026-08-19 to match before-fixed/*.md's section depth)
// plus GeneratedBy for traceability.
type PlanResult struct {
	RootCause       string
	CurrentBehavior string
	RecommendedFix  string
	// RootCauseMy/CurrentBehaviorMy/RecommendedFixMy are Burmese counterparts
	// to the 3 fields above, for the UI's language toggle. The fields added
	// below have no Burmese counterpart -- code, paths, and commands aren't
	// translated in before-fixed/*.md either, only its narrative prose is.
	RootCauseMy       string
	CurrentBehaviorMy string
	RecommendedFixMy  string
	RiskLevel         string // low|medium|high
	BreakingChange    bool
	FilesImpacted     []string
	TestPlan          []TestCase
	GeneratedBy       string

	// CodeContext is the ±10 lines around the flagged line, flagged line
	// marked "▶" -- same convention as before-fixed/*.md §2.
	CodeContext string
	// FixStrategyCode is a concrete code-block suggestion (before-fixed/*.md §4),
	// not prose -- see rule_templates.go.
	FixStrategyCode string
	// BeforeSnippet/AfterSnippet are before-fixed/*.md §5's preview pair.
	BeforeSnippet string
	AfterSnippet  string
	// SideEffects is before-fixed/*.md §6.
	SideEffects []string
	// ImpactAnalysis is before-fixed/*.md §7.
	ImpactAnalysis ImpactInfo
	// RecommendedTestCommands is before-fixed/*.md §8 (distinct from
	// TestPlan's Given/When/Then rows, which is a separate acceptance
	// concept already in the original schema).
	RecommendedTestCommands []string
	// AcceptanceCriteria is before-fixed/*.md §9.
	AcceptanceCriteria []string
}

// AIClient generates a structured fix plan for a batch of lint issues.
// The production implementation would call Claude; this pass ships only
// MockClient -- wiring a real client is a follow-up once a real
// ANTHROPIC_API_KEY is available (never pasted into chat).
type AIClient interface {
	GeneratePlan(ctx context.Context, req PlanRequest) (PlanResult, error)
}
