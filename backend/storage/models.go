package storage

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// newID returns a random 16-byte hex string, used as the primary key for
// every model below. golangci/plans/07-database-design.md specifies a
// string PK for all entities; this avoids adding a uuid dependency that
// isn't otherwise needed in this module. Returns an error instead of
// panicking — BeforeCreate hooks run as part of request handling, not
// main()/init(), so a rand-source failure must propagate, not crash.
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// LintScan tracks a single golangci-lint run for a repo+branch.
// See golangci/plans/07-database-design.md and 08-validation-rules.md
// for the decided status enum.
//
// json tags match 06-api-design.md/07-database-design.md's snake_case
// field names (e.g. repo_ref, not RepoRef) -- found missing during M3
// when a live response came back in Go's PascalCase instead.
type LintScan struct {
	ID           string    `gorm:"primaryKey;size:32" json:"id"`
	RepoRef      string    `                          json:"repo_ref"`
	Branch       string    `                          json:"branch"`
	CommitSHA    string    `                          json:"commit_sha"`
	TriggeredBy  string    `                          json:"triggered_by"`
	Status       string    `gorm:"size:16"            json:"status"` // running|success|failed
	ConfigHash   string    `                          json:"config_hash"`
	RawOutputRef string    `                          json:"raw_output_ref"`
	CreatedAt    time.Time `                          json:"created_at"`
	UpdatedAt    time.Time `                          json:"updated_at"`
}

func (m *LintScan) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// LintIssue is one normalized issue row produced by the Parser Service
// from a LintScan's raw golangci-lint JSON output.
type LintIssue struct {
	ID          string    `gorm:"primaryKey;size:32" json:"id"`
	ScanID      string    `gorm:"size:32;index"      json:"scan_id"`
	FilePath    string    `                          json:"file_path"`
	Line        int       `                          json:"line"`
	Column      int       `                          json:"column"`
	Linter      string    `                          json:"linter"`
	Rule        string    `                          json:"rule"`
	Message     string    `gorm:"type:text"          json:"message"`   // golangci-lint's issue text, minus the rule prefix -- see parser.Parse
	ReasonMy    string    `gorm:"type:text"          json:"reason_my"` // Burmese "why this breaks the rule" explanation -- see parser.reasonMy
	Severity    string    `gorm:"size:16"            json:"severity"`  // critical|high|medium|low|info
	Fingerprint string    `gorm:"size:64;index"      json:"fingerprint"`
	Status      string    `gorm:"size:16"            json:"status"` // open|planned|fix_applied|resolved|reopened|ignored
	CreatedAt   time.Time `                          json:"created_at"`
	UpdatedAt   time.Time `                          json:"updated_at"`

	Plans []FixPlan `gorm:"many2many:fix_plan_issues;" json:"plans,omitempty"`
}

func (m *LintIssue) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// FixPlan is an AI-generated (or, later, cached) fix plan covering one or
// more LintIssue rows from the same scan.
type FixPlan struct {
	ID              string `gorm:"primaryKey;size:32" json:"id"`
	RootCause       string `gorm:"type:text"          json:"root_cause"`
	CurrentBehavior string `gorm:"type:text"          json:"current_behavior"` // required by 04-prd.md FR-4 / 05-ui-design.md's Plan Viewer -- missing from the original M1 schema
	RecommendedFix  string `gorm:"type:text"          json:"recommended_fix"`
	// RootCauseMy/CurrentBehaviorMy/RecommendedFixMy are Burmese counterparts
	// to the 3 fields above, for the UI's language toggle -- additive, the
	// English fields are unchanged and remain the fallback when these are empty.
	RootCauseMy       string          `gorm:"type:text" json:"root_cause_my"`
	CurrentBehaviorMy string          `gorm:"type:text" json:"current_behavior_my"`
	RecommendedFixMy  string          `gorm:"type:text" json:"recommended_fix_my"`
	RiskLevel         string          `gorm:"size:16"   json:"risk_level"` // low|medium|high
	BreakingChange    bool            `                 json:"breaking_change"`
	FilesImpacted     json.RawMessage `gorm:"type:json" json:"files_impacted"`
	TestPlan          json.RawMessage `gorm:"type:json" json:"test_plan"`
	// CodeContext/FixStrategyCode/BeforeSnippet/AfterSnippet/SideEffects/
	// ImpactAnalysis/RecommendedTestCommands/AcceptanceCriteria added
	// 2026-08-19 to match before-fixed/*.md's section depth (§2, §4-9) --
	// see planner.PlanResult's matching doc comments for what populates each.
	CodeContext             string          `gorm:"type:text" json:"code_context"`
	FixStrategyCode         string          `gorm:"type:text" json:"fix_strategy_code"`
	BeforeSnippet           string          `gorm:"type:text" json:"before_snippet"`
	AfterSnippet            string          `gorm:"type:text" json:"after_snippet"`
	SideEffects             json.RawMessage `gorm:"type:json" json:"side_effects"`
	ImpactAnalysis          json.RawMessage `gorm:"type:json" json:"impact_analysis"`
	RecommendedTestCommands json.RawMessage `gorm:"type:json" json:"recommended_test_commands"`
	AcceptanceCriteria      json.RawMessage `gorm:"type:json" json:"acceptance_criteria"`
	// generating|pending|approved|rejected|applied|failed -- "generating"
	// and "failed" added in M3 for the async plan-job sub-states (a client
	// polling GET /plans/:id needs to tell "still working" from
	// "permanently failed, e.g. AI Layer unavailable" apart); the other 4
	// are 08-validation-rules.md's originally decided enum.
	Status      string `gorm:"size:16" json:"status"`
	GeneratedBy string `               json:"generated_by"`
	// BatchFingerprint/PromptVersion implement the Risk register's own
	// mitigation ("cache plan by issue+source fingerprint; store model +
	// prompt version on FixPlan") -- absent from the original M1 schema.
	BatchFingerprint string `gorm:"size:64;index" json:"batch_fingerprint"`
	PromptVersion    string `gorm:"size:32"       json:"prompt_version"`
	// ApprovedBy/ApprovedAt implement Rule BR-3's "approver identity +
	// timestamp recorded" requirement (10-business-rules.md) -- absent
	// from the original M1 schema. ApprovedAt is nil until approve/reject.
	ApprovedBy string     `json:"approved_by,omitempty"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	Issues []LintIssue `gorm:"many2many:fix_plan_issues;" json:"issues,omitempty"`
}

func (m *FixPlan) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// FixHistory records one apply attempt of a FixPlan.
type FixHistory struct {
	ID            string    `gorm:"primaryKey;size:32" json:"id"`
	PlanID        string    `gorm:"size:32;index"      json:"plan_id"`
	AppliedBy     string    `                          json:"applied_by"`
	BranchName    string    `                          json:"branch_name"`
	DiffRef       string    `                          json:"diff_ref"`
	PreFixScanID  string    `gorm:"size:32"            json:"pre_fix_scan_id"`
	PostFixScanID string    `gorm:"size:32"            json:"post_fix_scan_id"`
	Result        string    `gorm:"size:16"            json:"result"` // applying|passed|failed -- "applying" added in M4, apply is worker-queued like scan/plan
	CreatedAt     time.Time `                          json:"created_at"`
	UpdatedAt     time.Time `                          json:"updated_at"`
}

func (m *FixHistory) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// RollbackHistory records a revert of a FixHistory entry.
type RollbackHistory struct {
	ID                 string `gorm:"primaryKey;size:32" json:"id"`
	FixHistoryID       string `gorm:"size:32;index"      json:"fix_history_id"`
	RolledBackBy       string `                          json:"rolled_back_by"`
	RevertCommitSHA    string `                          json:"revert_commit_sha"`
	PostRollbackScanID string `gorm:"size:32"            json:"post_rollback_scan_id"`
	Reason             string `gorm:"type:text"          json:"reason"`
	// reverting|done|conflict|failed -- added when POST /rollbacks was
	// implemented: 11-sequence-diagram.md's own Flow 4 describes two
	// outcomes (revert succeeds vs conflicts) that need to be
	// distinguishable, and apply is worker-queued like scan/plan/fix so
	// needs an in-flight state too, but no such field existed.
	Result    string    `gorm:"size:16" json:"result"`
	CreatedAt time.Time `               json:"created_at"`
	UpdatedAt time.Time `               json:"updated_at"`
}

func (m *RollbackHistory) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// Configuration holds config-driven settings (severity mapping, model
// choice, per-repo overrides) scoped either "global" or "repo".
type Configuration struct {
	ID        string          `gorm:"primaryKey;size:32" json:"id"`
	Key       string          `gorm:"size:128;index"     json:"key"`
	Value     json.RawMessage `gorm:"type:json"          json:"value"`
	Scope     string          `gorm:"size:16"            json:"scope"` // global|repo
	CreatedAt time.Time       `                          json:"created_at"`
	UpdatedAt time.Time       `                          json:"updated_at"`
}

func (m *Configuration) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// AllModels returns every model this package owns, for AutoMigrate.
func AllModels() []any {
	return []any{
		&LintScan{},
		&LintIssue{},
		&FixPlan{},
		&FixHistory{},
		&RollbackHistory{},
		&Configuration{},
	}
}
