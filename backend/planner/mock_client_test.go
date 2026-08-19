package planner

import (
	"context"
	"testing"
)

func TestMockClient_GeneratePlan(t *testing.T) {
	tests := []struct {
		name          string
		req           PlanRequest
		wantErr       bool
		wantRiskLevel string
		wantFileCount int
	}{
		{
			name:    "no issues is an error",
			req:     PlanRequest{Issues: nil},
			wantErr: true,
		},
		{
			name: "single non-security issue defaults to low risk",
			req: PlanRequest{Issues: []IssueContext{
				{FilePath: "a.go", Line: 1, Linter: "revive", Rule: "package-comments", Message: "should have a comment"},
			}},
			wantRiskLevel: "low",
			wantFileCount: 1,
		},
		{
			name: "a gosec-family linter escalates risk to high",
			req: PlanRequest{Issues: []IssueContext{
				{FilePath: "a.go", Line: 1, Linter: "revive", Rule: "package-comments", Message: "x"},
				{FilePath: "b.go", Line: 2, Linter: "gosec", Rule: "G304", Message: "path traversal"},
			}},
			wantRiskLevel: "high",
			wantFileCount: 2,
		},
		{
			name: "duplicate file paths are deduplicated in FilesImpacted",
			req: PlanRequest{Issues: []IssueContext{
				{FilePath: "a.go", Line: 1, Linter: "revive", Rule: "r1", Message: "x"},
				{FilePath: "a.go", Line: 2, Linter: "revive", Rule: "r2", Message: "y"},
			}},
			wantRiskLevel: "low",
			wantFileCount: 1,
		},
	}

	c := NewMockClient()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.GeneratePlan(context.Background(), tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.RiskLevel != tt.wantRiskLevel {
				t.Errorf("RiskLevel = %q, want %q", result.RiskLevel, tt.wantRiskLevel)
			}
			if len(result.FilesImpacted) != tt.wantFileCount {
				t.Errorf("len(FilesImpacted) = %d, want %d", len(result.FilesImpacted), tt.wantFileCount)
			}
			if result.RootCause == "" || result.RecommendedFix == "" || result.CurrentBehavior == "" {
				t.Errorf("expected RootCause/RecommendedFix/CurrentBehavior to be non-empty, got %+v", result)
			}
			if len(result.TestPlan) == 0 {
				t.Errorf("expected at least one TestCase, got none")
			}
			if result.GeneratedBy == "" {
				t.Errorf("expected GeneratedBy to be set")
			}
		})
	}
}
