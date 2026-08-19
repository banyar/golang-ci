package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
	"golangci/backend/frontiir/utils"
	"golangci/backend/storage"
	"golangci/backend/worker"
)

type createFixRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
}

// CreateFix handles POST /fixes. Rule BR-3 (10-business-rules.md):
// applying is only allowed once a plan is "approved". Also acquires the
// same per-repo/branch lock CreateScan uses (Rule BR-1: one scan OR fix
// at a time per repo+branch) -- checked here, at the API layer, so a
// locked repo+branch fails fast with 409 rather than deep in the worker.
// @Summary      Apply an approved fix plan
// @Description  Rule BR-3: the plan must be "approved". Rule BR-1: one scan or fix at a time per repo+branch.
// @Tags         Fixes
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        body body createFixRequest true "Approved plan to apply"
// @Success      202 {object} map[string]any
// @Failure      400 {object} utils.RestErr
// @Failure      404 {object} utils.RestErr
// @Failure      409 {object} utils.RestErr "plan not approved, or a scan/fix already running for this repo+branch"
// @Router       /fixes [post]
func CreateFix(db *gorm.DB, rdb *redis.Client, lock *worker.Lock) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createFixRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			restErr := utils.InputErr(err.Error())
			restErr.Code = "VALIDATION_FAILED"
			c.JSON(restErr.Status, restErr)
			return
		}

		ctx := c.Request.Context()
		var plan storage.FixPlan
		if err := db.WithContext(ctx).
			Preload("Issues").
			First(&plan, "id = ?", req.PlanID).
			Error; err != nil {
			writeLookupErr(c, "plan", req.PlanID, err)
			return
		}
		if plan.Status != "approved" {
			restErr := utils.ConflictErr("plan %s is %q, not approved", plan.ID, plan.Status)
			restErr.Code = "PLAN_NOT_APPROVED"
			c.JSON(restErr.Status, restErr)
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
		history := storage.FixHistory{
			PlanID:    plan.ID,
			AppliedBy: fmt.Sprintf("%v", role),
			Result:    "applying",
		}
		if err := db.WithContext(ctx).Create(&history).Error; err != nil {
			_ = lock.Unlock(ctx, lockKey)
			c.JSON(http.StatusInternalServerError, utils.InternalErr("create fix history: %v", err))
			return
		}

		job := worker.FixJob{
			FixHistoryID: history.ID,
			PlanID:       plan.ID,
			RepoRef:      scan.RepoRef,
			Branch:       scan.Branch,
		}
		if err := worker.EnqueueFix(ctx, rdb, job); err != nil {
			// The history row was created but will never be processed --
			// mark it failed and release the lock rather than leave both
			// stuck (same class of bug as CreateScan/CreatePlan's
			// enqueue-failure handling).
			_ = lock.Unlock(ctx, lockKey)
			if markErr := db.WithContext(ctx).
				Model(&history).
				Update("result", "failed").
				Error; markErr != nil {
				c.JSON(
					http.StatusInternalServerError,
					utils.InternalErr(
						"enqueue fix: %v (and failed to mark it failed: %v)",
						err,
						markErr,
					),
				)
				return
			}
			c.JSON(http.StatusInternalServerError, utils.InternalErr("enqueue fix: %v", err))
			return
		}

		c.JSON(http.StatusAccepted, gin.H{"id": history.ID, "result": history.Result})
	}
}

// GetFix handles GET /fixes/:id.
// @Summary      Get a fix application's result by ID
// @Tags         Fixes
// @Security     ApiKeyAuth
// @Produce      json
// @Param        id path string true "Fix history ID"
// @Success      200 {object} storage.FixHistory
// @Failure      404 {object} utils.RestErr
// @Router       /fixes/{id} [get]
func GetFix(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var history storage.FixHistory
		if err := db.WithContext(c.Request.Context()).
			First(&history, "id = ?", c.Param("id")).
			Error; err != nil {
			writeLookupErr(c, "fix", c.Param("id"), err)
			return
		}
		c.JSON(http.StatusOK, history)
	}
}
