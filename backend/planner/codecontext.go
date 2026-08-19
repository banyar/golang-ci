package planner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// codeContextRadius matches before-fixed/*.md §2's own convention ("Line
// N (±10)").
const codeContextRadius = 10

// resolveInRepo joins repoRef+filePath and rejects the result if it
// doesn't actually resolve to somewhere under repoRef -- the same
// filepath.Clean+HasPrefix check rule_templates.go's own gosec G304
// template recommends for exactly this class of bug (path traversal via
// an unvalidated joined path). filePath is scanner-produced and already
// constrained to be repo-relative today, but nothing here should rely on
// that invariant holding for every future caller.
func resolveInRepo(repoRef, filePath string) (string, error) {
	full := filepath.Clean(filepath.Join(repoRef, filePath))
	base := filepath.Clean(repoRef)
	if full != base && !strings.HasPrefix(full, base+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside repo: %s", filePath)
	}
	return full, nil
}

// buildCodeContext reads repoRef/filePath and returns the ±10 lines around
// line, with the flagged line prefixed "▶" -- same marker convention as
// before-fixed/*.md §2. A missing/unreadable file (or one resolving
// outside repoRef) returns an honest fallback string (never fabricated
// code), matching that document's own "(file not readable: ...)" wording
// for the same failure case.
func buildCodeContext(repoRef, filePath string, line int) string {
	full, err := resolveInRepo(repoRef, filePath)
	if err != nil {
		return fmt.Sprintf("(file not readable: %s)", filePath)
	}
	b, err := os.ReadFile(full)
	if err != nil {
		return fmt.Sprintf("(file not readable: %s)", filePath)
	}

	lines := strings.Split(string(b), "\n")
	if line < 1 || line > len(lines) {
		return fmt.Sprintf("(line %d out of range for %s, %d lines)", line, filePath, len(lines))
	}

	start := max(line-codeContextRadius, 1)
	end := min(line+codeContextRadius, len(lines))

	var out strings.Builder
	fmt.Fprintf(&out, "Line %d (±%d) — marker \"▶\" = flagged line.\n\n", line, codeContextRadius)
	for i := start; i <= end; i++ {
		marker := "  "
		if i == line {
			marker = "▶ "
		}
		fmt.Fprintf(&out, "%s%4d  %s\n", marker, i, lines[i-1])
	}
	return out.String()
}

// flaggedLine returns just line's own text (trimmed), for PlanResult's
// BeforeSnippet -- the same failure/fallback handling as buildCodeContext.
func flaggedLine(repoRef, filePath string, line int) string {
	full, err := resolveInRepo(repoRef, filePath)
	if err != nil {
		return fmt.Sprintf("(file not readable: %s)", filePath)
	}
	b, err := os.ReadFile(full)
	if err != nil {
		return fmt.Sprintf("(file not readable: %s)", filePath)
	}
	lines := strings.Split(string(b), "\n")
	if line < 1 || line > len(lines) {
		return fmt.Sprintf("(line %d out of range for %s)", line, filePath)
	}
	return strings.TrimRight(lines[line-1], "\r")
}
