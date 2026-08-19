package planner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"golangci/backend/storage"
)

// PromptVersion is stored on every FixPlan (Risk register mitigation:
// "store model + prompt version on FixPlan", golangci/plans/14-risk-analysis.md).
// Bump this whenever the prompt-building logic changes meaningfully.
const PromptVersion = "planner-mock-v1"

var (
	// ErrIssueNotFound is returned when one or more requested issue_ids
	// don't exist.
	ErrIssueNotFound = errors.New("one or more issue_ids not found")
	// ErrCrossScanBatch is returned when the requested issues span more
	// than one scan (decided default, golangci/plans/13-test-plan.md scenario 9).
	ErrCrossScanBatch = errors.New("issue_ids must all belong to the same scan")
)

// Service owns the plan-generation business logic: same-scan validation,
// batch-fingerprint caching, and (once queued) calling the AIClient and
// persisting its result.
type Service struct {
	db *gorm.DB
	ai AIClient
}

// NewService constructs a Service backed by db and ai.
func NewService(db *gorm.DB, ai AIClient) *Service {
	return &Service{db: db, ai: ai}
}

// RequestPlan validates issueIDs and either returns an existing cached
// FixPlan (cacheHit=true, no AI call) or creates a new one in
// "generating" state (cacheHit=false — the caller must enqueue a plan
// job; Service intentionally has no queue dependency, to avoid an
// import cycle with golangci/worker, which itself calls FulfillPlan).
func (s *Service) RequestPlan(
	ctx context.Context,
	issueIDs []string,
) (*storage.FixPlan, bool, error) {
	uniqueIDs := dedupe(issueIDs)

	var issues []storage.LintIssue
	if err := s.db.WithContext(ctx).Where("id IN ?", uniqueIDs).Find(&issues).Error; err != nil {
		return nil, false, fmt.Errorf("load issues: %w", err)
	}
	if len(issues) != len(uniqueIDs) {
		return nil, false, ErrIssueNotFound
	}

	scanID := issues[0].ScanID
	for _, iss := range issues {
		if iss.ScanID != scanID {
			return nil, false, ErrCrossScanBatch
		}
	}

	fp := batchFingerprint(issues)

	var existing storage.FixPlan
	err := s.db.WithContext(ctx).
		Preload("Issues").
		Where("batch_fingerprint = ?", fp).
		First(&existing).
		Error
	switch {
	case err == nil:
		return &existing, true, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		// fall through to create a new plan
	default:
		return nil, false, fmt.Errorf("lookup cached plan: %w", err)
	}

	plan := storage.FixPlan{
		Status:           "generating",
		BatchFingerprint: fp,
		PromptVersion:    PromptVersion,
	}
	if err := s.db.WithContext(ctx).Create(&plan).Error; err != nil {
		return nil, false, fmt.Errorf("create plan: %w", err)
	}
	// Append via Association rather than Create-with-nested-struct, so
	// GORM doesn't also try to upsert the (already-persisted) LintIssue
	// rows themselves.
	if err := s.db.WithContext(ctx).Model(&plan).Association("Issues").Append(&issues); err != nil {
		return nil, false, fmt.Errorf("associate issues: %w", err)
	}
	plan.Issues = issues

	return &plan, false, nil
}

// MarkFailed flips a plan to "failed". Used when a plan job could not
// even be enqueued (so FulfillPlan will never run) -- the caller is
// already reporting a different (enqueue) failure to its client, so this
// error should be wrapped alongside it, not dropped.
func (s *Service) MarkFailed(ctx context.Context, planID string) error {
	return s.db.WithContext(ctx).
		Model(&storage.FixPlan{}).
		Where("id = ?", planID).
		Update("status", "failed").
		Error
}

// FulfillPlan calls the AIClient for a "generating" plan and persists the
// result. Called by the worker after a plan job is dequeued -- never
// inline in the request handler, since a real AI call is not instant.
func (s *Service) FulfillPlan(ctx context.Context, planID string) error {
	var plan storage.FixPlan
	if err := s.db.WithContext(ctx).
		Preload("Issues").
		First(&plan, "id = ?", planID).
		Error; err != nil {
		return fmt.Errorf("load plan %s: %w", planID, err)
	}

	// RepoRef is needed for code context/impact analysis (see
	// planner.PlanResult's CodeContext/ImpactAnalysis) -- plan.Issues is
	// already validated same-scan in RequestPlan, so one lookup covers
	// every issue in the batch.
	var scan storage.LintScan
	if err := s.db.WithContext(ctx).
		First(&scan, "id = ?", plan.Issues[0].ScanID).
		Error; err != nil {
		return fmt.Errorf("load scan for plan %s: %w", planID, err)
	}

	req := PlanRequest{Issues: make([]IssueContext, len(plan.Issues))}
	for i, iss := range plan.Issues {
		req.Issues[i] = IssueContext{
			FilePath: iss.FilePath,
			Line:     iss.Line,
			Column:   iss.Column,
			Linter:   iss.Linter,
			Rule:     iss.Rule,
			Message:  iss.Message,
			RepoRef:  scan.RepoRef,
		}
	}

	result, err := s.ai.GeneratePlan(ctx, req)
	if err != nil {
		if updateErr := s.db.WithContext(ctx).
			Model(&plan).
			Update("status", "failed").
			Error; updateErr != nil {
			return fmt.Errorf(
				"generate plan: %w (and failed to mark plan as failed: %w)",
				err,
				updateErr,
			)
		}
		return fmt.Errorf("generate plan %s: %w", planID, err)
	}

	filesImpacted, err := json.Marshal(result.FilesImpacted)
	if err != nil {
		return fmt.Errorf("marshal files_impacted: %w", err)
	}
	testPlan, err := json.Marshal(result.TestPlan)
	if err != nil {
		return fmt.Errorf("marshal test_plan: %w", err)
	}
	sideEffects, err := json.Marshal(result.SideEffects)
	if err != nil {
		return fmt.Errorf("marshal side_effects: %w", err)
	}
	impactAnalysis, err := json.Marshal(result.ImpactAnalysis)
	if err != nil {
		return fmt.Errorf("marshal impact_analysis: %w", err)
	}
	recommendedTestCommands, err := json.Marshal(result.RecommendedTestCommands)
	if err != nil {
		return fmt.Errorf("marshal recommended_test_commands: %w", err)
	}
	acceptanceCriteria, err := json.Marshal(result.AcceptanceCriteria)
	if err != nil {
		return fmt.Errorf("marshal acceptance_criteria: %w", err)
	}

	updates := map[string]any{
		"root_cause":                result.RootCause,
		"current_behavior":          result.CurrentBehavior,
		"recommended_fix":           result.RecommendedFix,
		"root_cause_my":             result.RootCauseMy,
		"current_behavior_my":       result.CurrentBehaviorMy,
		"recommended_fix_my":        result.RecommendedFixMy,
		"risk_level":                result.RiskLevel,
		"breaking_change":           result.BreakingChange,
		"files_impacted":            filesImpacted,
		"test_plan":                 testPlan,
		"status":                    "pending",
		"generated_by":              result.GeneratedBy,
		"code_context":              result.CodeContext,
		"fix_strategy_code":         result.FixStrategyCode,
		"before_snippet":            result.BeforeSnippet,
		"after_snippet":             result.AfterSnippet,
		"side_effects":              sideEffects,
		"impact_analysis":           impactAnalysis,
		"recommended_test_commands": recommendedTestCommands,
		"acceptance_criteria":       acceptanceCriteria,
	}
	return s.db.WithContext(ctx).Model(&plan).Updates(updates).Error
}

// dedupe removes duplicate ids while preserving first-seen order. Without
// this, a caller passing a duplicate issue_id would make len(issues) <
// len(issueIDs) even though every id is valid, wrongly reporting
// ErrIssueNotFound.
func dedupe(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// batchFingerprint hashes the sorted set of issue fingerprints so a plan
// requested for the same batch (regardless of issue_ids order) is
// recognized as a cache hit.
func batchFingerprint(issues []storage.LintIssue) string {
	fps := make([]string, len(issues))
	for i, iss := range issues {
		fps[i] = iss.Fingerprint
	}
	sort.Strings(fps)
	sum := sha256.Sum256([]byte(strings.Join(fps, ",")))
	return hex.EncodeToString(sum[:])
}
