package planner

import (
	"testing"

	"golangci/backend/storage"
)

func TestDedupe(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "no duplicates", in: []string{"a", "b", "c"}, want: []string{"a", "b", "c"}},
		{name: "adjacent duplicate", in: []string{"a", "a", "b"}, want: []string{"a", "b"}},
		{name: "non-adjacent duplicate preserves first-seen order", in: []string{"a", "b", "a"}, want: []string{"a", "b"}},
		{name: "empty", in: []string{}, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupe(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("dedupe(%v) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("dedupe(%v) = %v, want %v", tt.in, got, tt.want)
				}
			}
		})
	}
}

// batchFingerprint is pure (no DB) and safe to unit test directly. The
// cache-hit and AI-failure-path behaviors that depend on Service's *gorm.DB
// are covered by the live end-to-end verification instead, per this repo's
// existing convention that unit tests must not depend on external
// services (AGENTS.md's make test-unit vs make test TEST_FUNC=ALL split).
func TestBatchFingerprint(t *testing.T) {
	a := []storage.LintIssue{{Fingerprint: "fp1"}, {Fingerprint: "fp2"}}
	b := []storage.LintIssue{{Fingerprint: "fp2"}, {Fingerprint: "fp1"}} // reversed order
	c := []storage.LintIssue{{Fingerprint: "fp1"}, {Fingerprint: "fp3"}}

	fpA := batchFingerprint(a)
	fpB := batchFingerprint(b)
	fpC := batchFingerprint(c)

	if fpA != fpB {
		t.Errorf("batchFingerprint should be order-independent: got %q vs %q", fpA, fpB)
	}
	if fpA == fpC {
		t.Errorf("batchFingerprint should differ for a different issue set: both were %q", fpA)
	}
	if fpA == "" {
		t.Errorf("batchFingerprint should never be empty")
	}
}
