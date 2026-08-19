package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"golangci/backend/frontiir/utils"
	"golangci/backend/storage"
	"golangci/backend/worker"
)

type createRollbackRequest struct {
	FixHistoryID string `json:"fix_history_id" binding:"required"`
}

// CreateRollback handles POST /rollbacks. Rule BR-4 (10-business-rules.md):
// revert-only, never reset/force-push -- enforced in fixer.Revert, not
// here. This handler's job is validating the request is even eligible:
// the fix must have actually committed something, and it must not have
// already been rolled back (decided default, 13-test-plan.md scenario 29).
// @Summary      Roll back an applied fix
// @Description  Rule BR-4: revert-only, never reset/force-push. Fails if the fix made no commit, or was already rolled back.
// @Tags         Rollbacks
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        body body createRollbackRequest true "Fix to roll back"
// @Success      202 {object} map[string]any
// @Failure      400 {object} utils.RestErr "fix made no commit -- nothing to roll back"
// @Failure      404 {object} utils.RestErr
// @Failure      409 {object} utils.RestErr "already rolled back, or a scan/fix already running for this repo+branch"
// @Router       /rollbacks [post]
func CreateRollback(db *gorm.DB, rdb *redis.Client, lock *worker.Lock) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createRollbackRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			restErr := utils.InputErr(err.Error())
			restErr.Code = "VALIDATION_FAILED"
			c.JSON(restErr.Status, restErr)
			return
		}

		ctx := c.Request.Context()
		var fixHistory storage.FixHistory
		if err := db.WithContext(ctx).
			First(&fixHistory, "id = ?", req.FixHistoryID).
			Error; err != nil {
			writeLookupErr(c, "fix", req.FixHistoryID, err)
			return
		}
		if fixHistory.DiffRef == "" {
			restErr := utils.InputErr(
				fmt.Sprintf("fix %s made no commit (nothing to roll back)", fixHistory.ID),
			)
			restErr.Code = "NOTHING_TO_ROLLBACK"
			c.JSON(restErr.Status, restErr)
			return
		}

		var existing storage.RollbackHistory
		err := db.WithContext(ctx).Where("fix_history_id = ?", fixHistory.ID).First(&existing).Error
		if err == nil {
			restErr := utils.ConflictErr(
				"fix %s has already been rolled back (rollback %s)",
				fixHistory.ID,
				existing.ID,
			)
			restErr.Code = "ALREADY_ROLLED_BACK"
			c.JSON(restErr.Status, restErr)
			return
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(
				http.StatusInternalServerError,
				utils.InternalErr("check existing rollback: %v", err),
			)
			return
		}

		var plan storage.FixPlan
		if err := db.WithContext(ctx).
			Preload("Issues").
			First(&plan, "id = ?", fixHistory.PlanID).
			Error; err != nil {
			c.JSON(
				http.StatusInternalServerError,
				utils.InternalErr("load plan for fix %s: %v", fixHistory.ID, err),
			)
			return
		}
		if len(plan.Issues) == 0 {
			c.JSON(
				http.StatusInternalServerError,
				utils.InternalErr("plan %s has no associated issues", plan.ID),
			)
			return
		}
		var scan storage.LintScan
		if err := db.WithContext(ctx).
			First(&scan, "id = ?", plan.Issues[0].ScanID).
			Error; err != nil {
			c.JSON(
				http.StatusInternalServerError,
				utils.InternalErr("load scan for plan %s: %v", plan.ID, err),
			)
			return
		}

		lockKey := worker.LockKey(scan.RepoRef, scan.Branch)
		claimed, err := lock.TryLock(ctx, lockKey, scanLockTTL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utils.InternalErr("lock check failed: %v", err))
			return
		}
		if !claimed {
			restErr := utils.ConflictErr(
				"a scan or fix is already running for %s:%s",
				scan.RepoRef,
				scan.Branch,
			)
			restErr.Code = "SCAN_LOCKED"
			c.JSON(restErr.Status, restErr)
			return
		}

		role, _ := c.Get("role")
		rollback := storage.RollbackHistory{
			FixHistoryID: fixHistory.ID,
			RolledBackBy: fmt.Sprintf("%v", role),
			Result:       "reverting",
		}
		if err := db.WithContext(ctx).Create(&rollback).Error; err != nil {
			_ = lock.Unlock(ctx, lockKey)
			c.JSON(
				http.StatusInternalServerError,
				utils.InternalErr("create rollback history: %v", err),
			)
			return
		}

		job := worker.RollbackJob{
			RollbackHistoryID: rollback.ID,
			RepoRef:           scan.RepoRef,
			Branch:            scan.Branch,
			FixBranch:         fixHistory.BranchName,
			DiffRef:           fixHistory.DiffRef,
		}
		if err := worker.EnqueueRollback(ctx, rdb, job); err != nil {
			// Same class of bug as CreateScan/CreatePlan/CreateFix's
			// enqueue-failure handling: don't leave the row stuck.
			_ = lock.Unlock(ctx, lockKey)
			if markErr := db.WithContext(ctx).
				Model(&rollback).
				Update("result", "failed").
				Error; markErr != nil {
				c.JSON(
					http.StatusInternalServerError,
					utils.InternalErr(
						"enqueue rollback: %v (and failed to mark it failed: %v)",
						err,
						markErr,
					),
				)
				return
			}
			c.JSON(http.StatusInternalServerError, utils.InternalErr("enqueue rollback: %v", err))
			return
		}

		c.JSON(http.StatusAccepted, gin.H{"id": rollback.ID, "result": rollback.Result})
	}
}

// GetRollback handles GET /rollbacks/:id. Not one of the 14 endpoints in
// golangci/plans/06-api-design.md's original table -- added for
// consistency with /scans, /plans, /fixes, which all have a paired GET
// :id for polling an async operation's own result; without it there was
// no way to check a rollback's outcome before history endpoints exist.
// @Summary      Get a rollback's result by ID
// @Tags         Rollbacks
// @Security     ApiKeyAuth
// @Produce      json
// @Param        id path string true "Rollback ID"
// @Success      200 {object} storage.RollbackHistory
// @Failure      404 {object} utils.RestErr
// @Router       /rollbacks/{id} [get]
func GetRollback(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rollback storage.RollbackHistory
		if err := db.WithContext(c.Request.Context()).
			First(&rollback, "id = ?", c.Param("id")).
			Error; err != nil {
			writeLookupErr(c, "rollback", c.Param("id"), err)
			return
		}
		c.JSON(http.StatusOK, rollback)
	}
}
