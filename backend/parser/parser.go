// Package parser normalizes golangci-lint's raw JSON output into
// storage.LintIssue rows. See golangci/plans/07-database-design.md for the
// entity shape and golangci/plans/2026-08-04-golangci-m2-implementation.md
// for the real (v2) JSON schema this was verified against.
package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"golangci/backend/storage"
)

type rawIssue struct {
	FromLinter string `json:"FromLinter"`
	Text       string `json:"Text"`
	Pos        struct {
		Filename string `json:"Filename"`
		Line     int    `json:"Line"`
		Column   int    `json:"Column"`
	} `json:"Pos"`
}

type rawOutput struct {
	Issues []rawIssue `json:"Issues"`
}

// Parse normalizes raw golangci-lint JSON into LintIssue rows for scanID.
// golangci-lint v2 has no separate "rule" field -- linters that report one
// (e.g. revive's "package-comments: ...") embed it as a "<rule>: <message>"
// prefix in Text; splitRule pulls it back out. severityMap/defaultSeverity
// come from backend/config/severity-mapping.json -- golangci-lint's own
// Severity field is not useful (it's a literal "error" for everything by
// default), confirming 07-database-design.md's config-driven-severity
// rationale. Returns an empty (not nil) slice when there are no issues.
func Parse(
	scanID string,
	raw []byte,
	severityMap map[string]string,
	defaultSeverity string,
) ([]storage.LintIssue, error) {
	var out rawOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}

	issues := make([]storage.LintIssue, 0, len(out.Issues))
	for _, ri := range out.Issues {
		rule, message := splitRule(ri.Text, ri.FromLinter)

		severity, ok := severityMap[ri.FromLinter]
		if !ok {
			severity = defaultSeverity
		}

		issues = append(issues, storage.LintIssue{
			ScanID:      scanID,
			FilePath:    ri.Pos.Filename,
			Line:        ri.Pos.Line,
			Column:      ri.Pos.Column,
			Linter:      ri.FromLinter,
			Rule:        rule,
			Message:     message,
			ReasonMy:    reasonMy(ri.FromLinter, ri.Text),
			Severity:    severity,
			Fingerprint: fingerprint(ri.Pos.Filename, rule, message),
			Status:      "open",
		})
	}
	return issues, nil
}

// splitRule splits golangci-lint v2's "<rule>: <message>" Text convention.
// Falls back to the linter name as the rule when Text has no such prefix.
func splitRule(text, fallbackRule string) (rule, message string) {
	if idx := strings.Index(text, ": "); idx > 0 {
		return text[:idx], text[idx+2:]
	}
	return fallbackRule, text
}

// fingerprint tracks whether an issue is "the same" across scans even if
// its line number shifts (golangci/plans/07-database-design.md).
func fingerprint(filePath, rule, message string) string {
	sum := sha256.Sum256([]byte(filePath + rule + message))
	return hex.EncodeToString(sum[:])[:16]
}
