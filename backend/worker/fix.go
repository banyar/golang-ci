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

const fixQueueKey = "golangci:fix_jobs"

// FixJob is a unit of apply work -- the FixHistory row already exists
// (result="applying"); the worker applies the fix and verifies it via an
// inline rescan.
type FixJob struct {
	FixHistoryID string `json:"fix_history_id"`
	PlanID       string `json:"plan_id"`
	RepoRef      string `json:"repo_ref"`
	Branch       string `json:"branch"`
}

// EnqueueFix pushes a fix job onto its Redis-backed queue.
func EnqueueFix(ctx context.Context, rdb *redis.Client, job FixJob) error {
	b, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return rdb.RPush(ctx, fixQueueKey, b).Err()
}

// RunFixes consumes fix jobs until ctx is cancelled. Same shape as
// RunScans/RunPlans, run as its own goroutine against its own queue.
func (w *Worker) RunFixes(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		res, err := w.rdb.BLPop(ctx, 5*time.Second, fixQueueKey).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue
			}
			w.logger.Error("worker: fix BLPop failed", zap.Error(err))
			continue
		}

		var job FixJob
		if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
			w.logger.Error("worker: bad fix job payload", zap.Error(err))
			continue
		}
		w.processFix(ctx, job)
	}
}

func (w *Worker) processFix(ctx context.Context, job FixJob) {
	defer func() {
		if err := w.lock.Unlock(ctx, LockKey(job.RepoRef, job.Branch)); err != nil {
			w.logger.Error(
				"worker: unlock failed",
				zap.Error(err),
				zap.String("fix_history_id", job.FixHistoryID),
			)
		}
	}()

	var plan storage.FixPlan
	if err := w.db.WithContext(ctx).
		Preload("Issues").
		First(&plan, "id = ?", job.PlanID).
		Error; err != nil {
		w.failFix(ctx, job.FixHistoryID, "load plan", err)
		return
	}

	fixBranch := "lint-fix/" + job.PlanID
	commitSHA, changed, err := fixer.Apply(ctx, job.RepoRef, job.Branch, fixBranch)
	if err != nil {
		// An infra-level failure (worktree/golangci-lint execution) --
		// distinct from "ran fine, nothing was autofixable". The plan
		// stays "approved" so a retry is possible; it does not become
		// "applied".
		w.failFix(ctx, job.FixHistoryID, "apply", err)
		return
	}

	result := "failed" // default: no evidence the issue is actually gone
	var postScanID string

	if changed {
		raw, scanErr := scanner.Run(ctx, job.RepoRef, fixBranch)
		if scanErr != nil {
			w.logger.Error(
				"worker: post-fix rescan failed",
				zap.Error(scanErr),
				zap.String("fix_history_id", job.FixHistoryID),
			)
		} else {
			// scanID is set to "" here since the LintScan row doesn't exist
			// yet (its ID is assigned by GORM's BeforeCreate hook, only
			// available after Create below) -- backfilled onto each issue
			// once postScan.ID is known.
			postIssues, parseErr := parser.Parse("", raw, w.severityMap, w.defaultSeverity)
			if parseErr != nil {
				w.logger.Error(
					"worker: post-fix parse failed",
					zap.Error(parseErr),
					zap.String("fix_history_id", job.FixHistoryID),
				)
			} else {
				postScan := storage.LintScan{
					RepoRef:     job.RepoRef,
					Branch:      fixBranch,
					TriggeredBy: "system:fix-verification",
					Status:      "success",
				}
				if err := w.db.WithContext(ctx).Create(&postScan).Error; err != nil {
					w.logger.Error(
						"worker: persist post-fix scan failed",
						zap.Error(err),
						zap.String("fix_history_id", job.FixHistoryID),
					)
				} else {
					postScanID = postScan.ID
					for i := range postIssues {
						postIssues[i].ScanID = postScan.ID
					}
					if len(postIssues) > 0 {
						if err := w.db.WithContext(ctx).Create(&postIssues).Error; err != nil {
							w.logger.Error(
								"worker: persist post-fix issues failed",
								zap.Error(err),
								zap.String("fix_history_id", job.FixHistoryID),
							)
						}
					}
					if !anyFingerprintSurvives(plan.Issues, postIssues) {
						result = "passed"
					}
				}
			}
		}
	}

	issueStatus := "reopened"
	if result == "passed" {
		issueStatus = "resolved"
	}
	for _, iss := range plan.Issues {
		if err := w.db.WithContext(ctx).Model(&storage.LintIssue{}).Where("id = ?", iss.ID).
			Update("status", issueStatus).Error; err != nil {
			w.logger.Error(
				"worker: update issue status failed",
				zap.Error(err),
				zap.String("issue_id", iss.ID),
			)
		}
	}

	updates := map[string]any{
		"result":           result,
		"branch_name":      fixBranch,
		"diff_ref":         commitSHA,
		"post_fix_scan_id": postScanID,
	}
	if len(plan.Issues) > 0 {
		updates["pre_fix_scan_id"] = plan.Issues[0].ScanID
	}
	if err := w.db.WithContext(ctx).
		Model(&storage.FixHistory{}).
		Where("id = ?", job.FixHistoryID).
		Updates(updates).
		Error; err != nil {
		w.logger.Error(
			"worker: update fix history failed",
			zap.Error(err),
			zap.String("fix_history_id", job.FixHistoryID),
		)
	}
	if err := w.db.WithContext(ctx).
		Model(&storage.FixPlan{}).
		Where("id = ?", job.PlanID).
		Update("status", "applied").
		Error; err != nil {
		w.logger.Error(
			"worker: update plan status failed",
			zap.Error(err),
			zap.String("plan_id", job.PlanID),
		)
	}

	w.logger.Info(
		"worker: fix complete",
		zap.String("fix_history_id", job.FixHistoryID),
		zap.String("result", result),
		zap.Bool("changed", changed),
	)
}

func (w *Worker) failFix(ctx context.Context, fixHistoryID, stage string, cause error) {
	w.logger.Error(
		"worker: fix "+stage+" failed",
		zap.Error(cause),
		zap.String("fix_history_id", fixHistoryID),
	)
	if err := w.db.WithContext(ctx).
		Model(&storage.FixHistory{}).
		Where("id = ?", fixHistoryID).
		Update("result", "failed").
		Error; err != nil {
		w.logger.Error(
			"worker: update fix history to failed also failed",
			zap.Error(err),
			zap.String("fix_history_id", fixHistoryID),
		)
	}
}

// anyFingerprintSurvives reports whether any of the plan's original
// issues can still be found (by fingerprint) in the post-fix scan.
func anyFingerprintSurvives(original, postFix []storage.LintIssue) bool {
	postFP := make(map[string]struct{}, len(postFix))
	for _, iss := range postFix {
		postFP[iss.Fingerprint] = struct{}{}
	}
	for _, iss := range original {
		if _, ok := postFP[iss.Fingerprint]; ok {
			return true
		}
	}
	return false
}
