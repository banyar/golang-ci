// Package fixer applies native golangci-lint fixes on an isolated
// lint-fix/* branch. See golangci/plans/12-component-design.md's "Fix
// Service" role and Rule BR-2 (10-business-rules.md): commits only ever
// land on a lint-fix/* branch, never main/master.
//
// Only the native `golangci-lint --fix` path is implemented (M4 scope
// decision, golangci/plans/2026-08-04-golangci-m4-implementation.md) --
// applying an AI-authored patch for non-autofixable issues needs a real
// AIClient that can produce an actual diff, which planner.MockClient
// cannot (it only produces prose). Applying a plan whose issues aren't
// autofixable is not an error: Apply returns changed=false.
package fixer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Apply creates fixBranch (a new branch) from baseBranch in an isolated
// worktree, runs `golangci-lint run --fix`, and commits any resulting
// changes. changed reports whether anything was actually modified.
// fixBranch must already be caller-constructed as a lint-fix/* name --
// Apply does not accept or validate an arbitrary branch target, which is
// how Rule BR-2 is enforced (by construction, not a runtime check).
func Apply(
	ctx context.Context,
	repoPath, baseBranch, fixBranch string,
) (commitSHA string, changed bool, err error) {
	if err := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "--is-inside-work-tree").
		Run(); err != nil {
		return "", false, fmt.Errorf("%s is not a git repository: %w", repoPath, err)
	}

	// Sibling of repoPath, not system temp -- see scanner.Run's identical
	// comment: a go.work file's relative sibling-module references (e.g.
	// "../rtdatacore") only resolve correctly when the worktree sits next
	// to repoPath itself.
	tmpDir, err := os.MkdirTemp(filepath.Dir(repoPath), "golangci-fix-*")
	if err != nil {
		return "", false, fmt.Errorf("create fix worktree dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// A separate scratch dir OUTSIDE the worktree -- the JSON report and
	// lint cache must never live inside tmpDir, or `git add -A` below
	// would commit them as if they were part of the fix (found during
	// M4 verification: an empty lint.json was committed and mistaken for
	// a real change).
	scratchDir, err := os.MkdirTemp("", "golangci-fix-scratch-*")
	if err != nil {
		return "", false, fmt.Errorf("create fix scratch dir: %w", err)
	}
	defer os.RemoveAll(scratchDir)

	// -b creates fixBranch fresh off baseBranch. Unlike scanner.Run's
	// --detach, this branch must be a real, named, checked-out branch so
	// it persists as a reviewable lint-fix/* ref after the worktree (but
	// not the branch) is removed below.
	addCmd := exec.CommandContext(
		ctx,
		"git",
		"-C",
		repoPath,
		"worktree",
		"add",
		"-b",
		fixBranch,
		tmpDir,
		baseBranch,
	)
	if out, err := addCmd.CombinedOutput(); err != nil {
		return "", false, fmt.Errorf("git worktree add -b %s: %w (%s)", fixBranch, err, string(out))
	}
	defer exec.Command("git", "-C", repoPath, "worktree", "remove", tmpDir, "--force").
		Run()
		//nolint:errcheck // best-effort cleanup; the branch itself is untouched by this

	fixPath := filepath.Join(scratchDir, "lint.json")
	fixCmd := exec.CommandContext(
		ctx,
		"golangci-lint",
		"run",
		"--fix",
		"--output.json.path",
		fixPath,
		"./...",
	)
	fixCmd.Dir = tmpDir
	// GOLANGCI_LINT_CACHE: golangci-lint's default lint-result cache is
	// keyed by file content + config, shared across every invocation on
	// this machine -- a previous scan/fix of byte-identical content (even
	// in a completely different repo) can serve a stale "0 issues"
	// result here and silently suppress --fix from doing anything. A
	// fresh, per-invocation cache dir avoids that entirely.
	// GOROOT: see scanner.Run's identical comment -- this process's ambient
	// environment can carry a stale/wrong GOROOT that makes golangci-lint's
	// typecheck pass fail for every package it touches, silently preventing
	// --fix from doing anything to those files. Appended last so it
	// overrides any inherited value.
	fixCmd.Env = append(
		os.Environ(),
		"GOLANGCI_LINT_CACHE="+filepath.Join(scratchDir, "cache"),
		"GOROOT=/usr/local/go",
	)
	_ = fixCmd.Run() // non-zero exit is expected if unfixable issues remain -- not a failure

	statusOut, err := exec.CommandContext(ctx, "git", "-C", tmpDir, "status", "--porcelain").
		Output()
	if err != nil {
		return "", false, fmt.Errorf("git status: %w", err)
	}
	if len(strings.TrimSpace(string(statusOut))) == 0 {
		return "", false, nil // nothing was autofixable -- a valid, non-error outcome
	}

	if out, err := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", "-A").
		CombinedOutput(); err != nil {
		return "", false, fmt.Errorf("git add: %w (%s)", err, string(out))
	}
	commitCmd := exec.CommandContext(
		ctx,
		"git",
		"-C",
		tmpDir,
		"commit",
		"-m",
		"golangci-lint --fix ("+fixBranch+")",
	)
	if out, err := commitCmd.CombinedOutput(); err != nil {
		return "", false, fmt.Errorf("git commit: %w (%s)", err, string(out))
	}

	shaOut, err := exec.CommandContext(ctx, "git", "-C", tmpDir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", true, fmt.Errorf("rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(shaOut)), true, nil
}
