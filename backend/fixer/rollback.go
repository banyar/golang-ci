package fixer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Revert runs `git revert --no-edit <diffRef>` on fixBranch (which must
// already exist -- created by an earlier Apply) in an isolated worktree.
// conflict reports whether the revert hit a merge conflict; when it does,
// the revert is aborted (`git revert --abort`) rather than forced -- Rule
// BR-4 (golangci/plans/10-business-rules.md): no reset/force-push, ever.
// conflictDetail carries git's own conflict output for RollbackHistory.Reason.
func Revert(
	ctx context.Context,
	repoPath, fixBranch, diffRef string,
) (commitSHA string, conflict bool, conflictDetail string, err error) {
	if err := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "--is-inside-work-tree").
		Run(); err != nil {
		return "", false, "", fmt.Errorf("%s is not a git repository: %w", repoPath, err)
	}

	tmpDir, err := os.MkdirTemp("", "golangci-rollback-*")
	if err != nil {
		return "", false, "", fmt.Errorf("create rollback worktree dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// No -b: fixBranch already exists (created by an earlier Apply, whose
	// own worktree has since been removed -- the branch itself was never
	// left checked out anywhere, so this attaches to it cleanly). Attached
	// (not --detach, unlike scanner.Run) since the revert commit must
	// land on this exact branch, not a throwaway detached HEAD.
	addCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "worktree", "add", tmpDir, fixBranch)
	if out, err := addCmd.CombinedOutput(); err != nil {
		return "", false, "", fmt.Errorf(
			"git worktree add %s: %w (%s)",
			fixBranch,
			err,
			string(out),
		)
	}
	defer exec.Command("git", "-C", repoPath, "worktree", "remove", tmpDir, "--force").
		Run()
		//nolint:errcheck // best-effort cleanup, RemoveAll above is the fallback

	revertCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "revert", "--no-edit", diffRef)
	out, revertErr := revertCmd.CombinedOutput()
	if revertErr != nil {
		// Never force through a conflict -- abort cleanly and report it.
		_ = exec.CommandContext(ctx, "git", "-C", tmpDir, "revert", "--abort").Run()
		return "", true, strings.TrimSpace(string(out)), nil
	}

	shaOut, err := exec.CommandContext(ctx, "git", "-C", tmpDir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", false, "", fmt.Errorf("rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(shaOut)), false, "", nil
}
