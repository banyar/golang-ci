// Package scanner runs golangci-lint against an isolated git worktree of a
// target repo+branch. See golangci/plans/03-proposed-workflow.md and
// golangci/plans/12-component-design.md for the "Scan Service" role this
// implements.
package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Run scans repoPath's branch and returns golangci-lint's raw JSON output,
// with each issue's file path normalized to a project-relative path (see
// normalizeIssuePaths).
//
// repoPath must be a local git repository already on disk (M2 scope is
// local-path only -- see golangci/plans/2026-08-04-golangci-m2-implementation.md
// for why remote-clone support is deferred).
//
// A temporary git worktree is used so the caller's actual checkout is
// never touched -- concurrent scans on different branches don't collide.
// golangci-lint's own exit code is not treated as failure: it exits
// non-zero when it finds issues, which is the normal, expected case. Only
// a missing/unparseable output file counts as a real execution failure.
func Run(ctx context.Context, repoPath, branch string) ([]byte, error) {
	if err := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "--is-inside-work-tree").
		Run(); err != nil {
		return nil, fmt.Errorf("%s is not a git repository: %w", repoPath, err)
	}

	// Sibling of repoPath, not system temp: some repos (e.g. rt-external-api's
	// go.work) reference sibling modules by relative path ("../rtdatacore").
	// A worktree anywhere else breaks that resolution when golangci-lint
	// loads packages -- "cannot load module ../rtdatacore ... no such file
	// or directory" -- because the relative path is resolved from the
	// worktree's own location, not repoPath's.
	tmpDir, err := os.MkdirTemp(filepath.Dir(repoPath), "golangci-scan-*")
	if err != nil {
		return nil, fmt.Errorf("create scan worktree dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// --detach: git refuses "worktree add <dir> <branch>" whenever <branch>
	// is already checked out somewhere (including repoPath's own primary
	// checkout) -- "fatal: '<branch>' is already used by worktree at ...".
	// That's the common case (scanning main while main is checked out), so
	// we always add in detached-HEAD state at that branch's current commit.
	addCmd := exec.CommandContext(
		ctx,
		"git",
		"-C",
		repoPath,
		"worktree",
		"add",
		"--detach",
		tmpDir,
		branch,
	)
	if out, err := addCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git worktree add (branch %q): %w (%s)", branch, err, string(out))
	}
	defer exec.Command("git", "-C", repoPath, "worktree", "remove", tmpDir, "--force").
		Run()
		//nolint:errcheck // best-effort cleanup, RemoveAll above is the fallback

	jsonPath := filepath.Join(tmpDir, "lint.json")
	// --path-mode abs: without it, paths default to relative-to-CWD, which
	// renders differently depending on which base golangci-lint resolved
	// the file against (see normalizeIssuePaths) -- e.g.
	// "../golangci-scan-<random>/pkg/foo.go", a different string for the
	// same logical file on every single scan. That silently breaks
	// fingerprint-based comparison across scans (M4 discovered this: a
	// post-fix rescan's issues never matched the pre-fix ones because the
	// paths never matched, even for files nothing had touched). abs mode
	// reports a stable absolute path instead; normalizeIssuePaths below
	// rewrites that into a clean, portable, project-relative value.
	lintCmd := exec.CommandContext(
		ctx,
		"golangci-lint",
		"run",
		"--path-mode",
		"abs",
		"--output.json.path",
		jsonPath,
		"./...",
	)
	lintCmd.Dir = tmpDir
	// GOLANGCI_LINT_CACHE: the default lint-result cache is keyed by file
	// content + config and shared across every invocation on this
	// machine -- a previous scan of byte-identical content (even from an
	// unrelated repo) can serve a stale result here. A fresh,
	// per-invocation cache dir avoids that (found during M4 verification
	// of fixer.Apply, which has the same exposure).
	cacheDir, err := os.MkdirTemp("", "golangci-scan-cache-*")
	if err != nil {
		return nil, fmt.Errorf("create scan cache dir: %w", err)
	}
	defer os.RemoveAll(cacheDir)
	lintCmd.Env = append(os.Environ(), "GOLANGCI_LINT_CACHE="+cacheDir)
	_ = lintCmd.Run() // non-zero exit here just means "found issues" -- not a failure

	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf(
			"golangci-lint produced no output (execution error, not just lint findings): %w",
			err,
		)
	}

	normalized, err := normalizeIssuePaths(raw, tmpDir, repoPath)
	if err != nil {
		return nil, fmt.Errorf("normalize issue paths: %w", err)
	}
	return normalized, nil
}

// normalizeIssuePaths rewrites each issue's Pos.Filename (an absolute
// path, per --path-mode abs) to a project-relative path (e.g. "pkg/foo.go"),
// so the same logical file gets the same FilePath across every scan --
// needed for fingerprint-based comparison (M4's post-fix rescan) to work
// at all.
//
// Empirically, which absolute path golangci-lint reports for a worktree
// file is not consistent: it has been observed resolving to the worktree
// itself (rooted at tmpDir) and, in other runs, back to repoPath's own
// checkout (git worktrees share history, and some underlying tooling
// appears to report a "canonical" path depending on caching state). Both
// bases are tried; whichever actually contains the file wins.
//
// Uses map[string]any for the per-issue shape so every other field
// (SourceLines, LineRange, ExpectNoLint, ...) passes through untouched --
// this package doesn't need to know the full schema, only the one field
// it must fix up.
func normalizeIssuePaths(raw []byte, tmpDir, repoPath string) ([]byte, error) {
	var out struct {
		Issues []map[string]any `json:"Issues"`
		Report json.RawMessage  `json:"Report"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}

	for _, issue := range out.Issues {
		pos, ok := issue["Pos"].(map[string]any)
		if !ok {
			continue
		}
		filename, ok := pos["Filename"].(string)
		if !ok || !filepath.IsAbs(filename) {
			continue
		}
		if rel, err := filepath.Rel(tmpDir, filename); err == nil && !strings.HasPrefix(rel, "..") {
			pos["Filename"] = rel
			continue
		}
		if rel, err := filepath.Rel(
			repoPath,
			filename,
		); err == nil &&
			!strings.HasPrefix(rel, "..") {
			pos["Filename"] = rel
		}
	}

	return json.Marshal(out)
}
