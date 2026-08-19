package worker

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"golangci/backend/parser"
	"golangci/backend/planner"
	"golangci/backend/scanner"
	"golangci/backend/storage"
)

// Two independent queues (not one tagged queue) so a slow AI call (once
// planner's AIClient is real) can never head-of-line-block scanning, and
// vice versa -- see golangci/plans/2026-08-04-golangci-m3-implementation.md.
const (
	scanQueueKey = "golangci:scan_jobs"
	planQueueKey = "golangci:plan_jobs"
)

// Job is a unit of scan work handed from the API layer to the worker, off
// the request path (NFR-1/NFR-5, golangci/plans/04-prd.md).
type Job struct {
	ScanID  string `json:"scan_id"`
	RepoRef string `json:"repo_ref"`
	Branch  string `json:"branch"`
}

// Enqueue pushes a scan job onto its Redis-backed queue.
func Enqueue(ctx context.Context, rdb *redis.Client, job Job) error {
	b, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return rdb.RPush(ctx, scanQueueKey, b).Err()
}

// PlanJob is a unit of plan-generation work -- the FixPlan row already
// exists (status="generating"); the worker just needs to fulfill it.
type PlanJob struct {
	PlanID string `json:"plan_id"`
}

// EnqueuePlan pushes a plan job onto its Redis-backed queue.
func EnqueuePlan(ctx context.Context, rdb *redis.Client, job PlanJob) error {
	b, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return rdb.RPush(ctx, planQueueKey, b).Err()
}

// Worker consumes scan and plan jobs and runs them to completion.
type Worker struct {
	rdb             *redis.Client
	lock            *Lock
	db              *gorm.DB
	logger          *zap.Logger
	severityMap     map[string]string
	defaultSeverity string
	planner         *planner.Service
}

// NewWorker constructs a Worker that consumes jobs from both queues, runs
// scans/plans, persists results, and releases the per-repo/branch lock
// (scan jobs only -- plan jobs don't hold that lock).
func NewWorker(
	rdb *redis.Client,
	lock *Lock,
	db *gorm.DB,
	logger *zap.Logger,
	severityMap map[string]string,
	defaultSeverity string,
	plannerSvc *planner.Service,
) *Worker {
	return &Worker{
		rdb:             rdb,
		lock:            lock,
		db:              db,
		logger:          logger,
		severityMap:     severityMap,
		defaultSeverity: defaultSeverity,
		planner:         plannerSvc,
	}
}

// RunScans consumes scan jobs until ctx is cancelled. Uses a short BLPop
// timeout (rather than blocking indefinitely) so shutdown is responsive.
func (w *Worker) RunScans(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		res, err := w.rdb.BLPop(ctx, 5*time.Second, scanQueueKey).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue // timed out with nothing queued, or shutting down
			}
			w.logger.Error("worker: scan BLPop failed", zap.Error(err))
			continue
		}

		var job Job
		if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
			w.logger.Error("worker: bad scan job payload", zap.Error(err))
			continue
		}
		w.processScan(ctx, job)
	}
}

// RunPlans consumes plan jobs until ctx is cancelled. Same shape as
// RunScans, run as an independent goroutine against a separate queue.
func (w *Worker) RunPlans(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		res, err := w.rdb.BLPop(ctx, 5*time.Second, planQueueKey).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue
			}
			w.logger.Error("worker: plan BLPop failed", zap.Error(err))
			continue
		}

		var job PlanJob
		if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
			w.logger.Error("worker: bad plan job payload", zap.Error(err))
			continue
		}
		if err := w.planner.FulfillPlan(ctx, job.PlanID); err != nil {
			w.logger.Error(
				"worker: fulfill plan failed",
				zap.Error(err),
				zap.String("plan_id", job.PlanID),
			)
			continue
		}
		w.logger.Info("worker: plan complete", zap.String("plan_id", job.PlanID))
	}
}

func (w *Worker) processScan(ctx context.Context, job Job) {
	defer func() {
		if err := w.lock.Unlock(ctx, LockKey(job.RepoRef, job.Branch)); err != nil {
			w.logger.Error(
				"worker: unlock failed",
				zap.Error(err),
				zap.String("scan_id", job.ScanID),
			)
		}
	}()

	raw, err := scanner.Run(ctx, job.RepoRef, job.Branch)
	if err != nil {
		w.fail(ctx, job.ScanID, "scan", err)
		return
	}

	issues, err := parser.Parse(job.ScanID, raw, w.severityMap, w.defaultSeverity)
	if err != nil {
		w.fail(ctx, job.ScanID, "parse", err)
		return
	}

	if len(issues) > 0 {
		if err := w.db.WithContext(ctx).Create(&issues).Error; err != nil {
			w.fail(ctx, job.ScanID, "persist issues", err)
			return
		}
	}

	if err := w.db.WithContext(ctx).
		Model(&storage.LintScan{}).
		Where("id = ?", job.ScanID).
		Update("status", "success").
		Error; err != nil {
		w.logger.Error(
			"worker: update scan status failed",
			zap.Error(err),
			zap.String("scan_id", job.ScanID),
		)
		return
	}
	w.logger.Info(
		"worker: scan complete",
		zap.String("scan_id", job.ScanID),
		zap.Int("issues", len(issues)),
	)
}

func (w *Worker) fail(ctx context.Context, scanID, stage string, cause error) {
	w.logger.Error("worker: "+stage+" failed", zap.Error(cause), zap.String("scan_id", scanID))
	if err := w.db.WithContext(ctx).
		Model(&storage.LintScan{}).
		Where("id = ?", scanID).
		Update("status", "failed").
		Error; err != nil {
		w.logger.Error(
			"worker: update scan status to failed also failed",
			zap.Error(err),
			zap.String("scan_id", scanID),
		)
	}
}
