package worker

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"golangci/backend/fixer"
	"golangci/backend/parser"
	"golangci/backend/scanner"
	"golangci/backend/storage"
)

const rollbackQueueKey = "golangci:rollback_jobs"

// RollbackJob is a unit of revert work -- the RollbackHistory row already
// exists (result="reverting"); the worker performs the git revert and a
// confirmation rescan.
type RollbackJob struct {
	RollbackHistoryID string `json:"rollback_history_id"`
	RepoRef           string `json:"repo_ref"`
	Branch            string `json:"branch"` // original base branch -- the lock key, not the fix branch
	FixBranch         string `json:"fix_branch"`
	DiffRef           string `json:"diff_ref"`
}

// EnqueueRollback pushes a rollback job onto its Redis-backed queue.
func EnqueueRollback(ctx context.Context, rdb *redis.Client, job RollbackJob) error {
	b, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return rdb.RPush(ctx, rollbackQueueKey, b).Err()
}

// RunRollbacks consumes rollback jobs until ctx is cancelled. Same shape
// as RunScans/RunPlans/RunFixes, its own goroutine against its own queue.
func (w *Worker) RunRollbacks(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		res, err := w.rdb.BLPop(ctx, 5*time.Second, rollbackQueueKey).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue
			}
			w.logger.Error("worker: rollback BLPop failed", zap.Error(err))
			continue
		}

		var job RollbackJob
		if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
			w.logger.Error("worker: bad rollback job payload", zap.Error(err))
			continue
		}
		w.processRollback(ctx, job)
	}
}

func (w *Worker) processRollback(ctx context.Context, job RollbackJob) {
	defer func() {
		if err := w.lock.Unlock(ctx, LockKey(job.RepoRef, job.Branch)); err != nil {
			w.logger.Error(
				"worker: unlock failed",
				zap.Error(err),
				zap.String("rollback_history_id", job.RollbackHistoryID),
			)
		}
	}()

	commitSHA, conflict, conflictDetail, err := fixer.Revert(
		ctx,
		job.RepoRef,
		job.FixBranch,
		job.DiffRef,
	)
	if err != nil {
		w.failRollback(ctx, job.RollbackHistoryID, err.Error())
		return
	}

	if conflict {
		updates := map[string]any{
			"result": "conflict",
			"reason": "revert conflicts with a later commit on " + job.FixBranch + ": " + conflictDetail,
		}
		if err := w.db.WithContext(ctx).
			Model(&storage.RollbackHistory{}).
			Where("id = ?", job.RollbackHistoryID).
			Updates(updates).
			Error; err != nil {
			w.logger.Error(
				"worker: update rollback history (conflict) failed",
				zap.Error(err),
				zap.String("rollback_history_id", job.RollbackHistoryID),
			)
		}
		w.logger.Info(
			"worker: rollback conflict, manual resolution required",
			zap.String("rollback_history_id", job.RollbackHistoryID),
		)
		return
	}

	// Confirmation rescan (11-sequence-diagram.md Flow 4): evidence that
	// reverting actually restored the pre-fix issue state. Best-effort --
	// its failure doesn't undo the revert that already succeeded.
	var postScanID string
	raw, scanErr := scanner.Run(ctx, job.RepoRef, job.FixBranch)
	if scanErr != nil {
		w.logger.Error(
			"worker: post-rollback confirmation rescan failed",
			zap.Error(scanErr),
			zap.String("rollback_history_id", job.RollbackHistoryID),
		)
	} else {
		postIssues, parseErr := parser.Parse("", raw, w.severityMap, w.defaultSeverity)
		if parseErr != nil {
			w.logger.Error(
				"worker: post-rollback parse failed",
				zap.Error(parseErr),
				zap.String("rollback_history_id", job.RollbackHistoryID),
			)
		} else {
			postScan := storage.LintScan{
				RepoRef:     job.RepoRef,
				Branch:      job.FixBranch,
				TriggeredBy: "system:rollback-confirmation",
				Status:      "success",
			}
			if err := w.db.WithContext(ctx).Create(&postScan).Error; err != nil {
				w.logger.Error(
					"worker: persist post-rollback scan failed",
					zap.Error(err),
					zap.String("rollback_history_id", job.RollbackHistoryID),
				)
			} else {
				postScanID = postScan.ID
				for i := range postIssues {
					postIssues[i].ScanID = postScan.ID
				}
				if len(postIssues) > 0 {
					if err := w.db.WithContext(ctx).Create(&postIssues).Error; err != nil {
						w.logger.Error(
							"worker: persist post-rollback issues failed",
							zap.Error(err),
							zap.String("rollback_history_id", job.RollbackHistoryID),
						)
					}
				}
			}
		}
	}

	updates := map[string]any{
		"result":                "done",
		"revert_commit_sha":     commitSHA,
		"post_rollback_scan_id": postScanID,
	}
	if err := w.db.WithContext(ctx).
		Model(&storage.RollbackHistory{}).
		Where("id = ?", job.RollbackHistoryID).
		Updates(updates).
		Error; err != nil {
		w.logger.Error(
			"worker: update rollback history failed",
			zap.Error(err),
			zap.String("rollback_history_id", job.RollbackHistoryID),
		)
	}
	w.logger.Info(
		"worker: rollback complete",
		zap.String("rollback_history_id", job.RollbackHistoryID),
	)
}

func (w *Worker) failRollback(ctx context.Context, rollbackHistoryID, reason string) {
	w.logger.Error(
		"worker: rollback failed",
		zap.String("rollback_history_id", rollbackHistoryID),
		zap.String("reason", reason),
	)
	updates := map[string]any{"result": "failed", "reason": reason}
	if err := w.db.WithContext(ctx).
		Model(&storage.RollbackHistory{}).
		Where("id = ?", rollbackHistoryID).
		Updates(updates).
		Error; err != nil {
		w.logger.Error(
			"worker: update rollback history to failed also failed",
			zap.Error(err),
			zap.String("rollback_history_id", rollbackHistoryID),
		)
	}
}
