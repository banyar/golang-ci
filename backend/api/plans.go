package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"golangci/backend/frontiir/utils"
	"golangci/backend/planner"
	"golangci/backend/storage"
	"golangci/backend/worker"
)

type createPlanRequest struct {
	IssueIDs []string `json:"issue_ids" binding:"required,min=1"`
}

// CreatePlan handles POST /plans. A cache hit (same issue batch already
// planned, golangci/plans/13-test-plan.md scenario 12) returns 200 with
// the ready plan immediately -- no async work happened, so 202 would be
// misleading. A cache miss creates a "generating" FixPlan and enqueues
// the AI work, returning 202.
// @Summary      Request an AI-generated fix plan
// @Description  Cache hit (same issue batch already planned) returns 200 with the ready plan immediately; a cache miss enqueues generation and returns 202.
// @Tags         Plans
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        body body createPlanRequest true "Issue IDs to plan a fix for"
// @Success      200 {object} storage.FixPlan "cache hit -- plan already ready"
// @Success      202 {object} map[string]any   "cache miss -- plan generation enqueued"
// @Failure      400 {object} utils.RestErr
// @Failure      404 {object} utils.RestErr
// @Router       /plans [post]
func CreatePlan(svc *planner.Service, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createPlanRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			restErr := utils.InputErr(err.Error())
			restErr.Code = "VALIDATION_FAILED"
			c.JSON(restErr.Status, restErr)
			return
		}

		ctx := c.Request.Context()
		plan, cacheHit, err := svc.RequestPlan(ctx, req.IssueIDs)
		if err != nil {
			writePlanRequestErr(c, err)
			return
		}

		if cacheHit {
			c.JSON(http.StatusOK, plan)
			return
		}

		if err := worker.EnqueuePlan(ctx, rdb, worker.PlanJob{PlanID: plan.ID}); err != nil {
			// The plan row was created but will never be processed --
			// mark it failed rather than leave it stuck at "generating"
			// forever (same class of bug as CreateScan's enqueue-failure
			// handling in golangci/api/scans.go).
			if markErr := svc.MarkFailed(ctx, plan.ID); markErr != nil {
				c.JSON(
					http.StatusInternalServerError,
					utils.InternalErr(
						"enqueue plan: %v (and failed to mark it failed: %v)",
						err,
						markErr,
					),
				)
				return
			}
			c.JSON(http.StatusInternalServerError, utils.InternalErr("enqueue plan: %v", err))
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"id": plan.ID, "status": plan.Status})
	}
}

// GetPlan handles GET /plans/:id.
// @Summary      Get a fix plan by ID
// @Tags         Plans
// @Security     ApiKeyAuth
// @Produce      json
// @Param        id path string true "Plan ID"
// @Success      200 {object} storage.FixPlan
// @Failure      404 {object} utils.RestErr
// @Router       /plans/{id} [get]
func GetPlan(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var plan storage.FixPlan
		err := db.WithContext(c.Request.Context()).
			Preload("Issues").
			First(&plan, "id = ?", c.Param("id")).
			Error
		if err != nil {
			writeLookupErr(c, "plan", c.Param("id"), err)
			return
		}
		c.JSON(http.StatusOK, plan)
	}
}

type approvePlanRequest struct {
	// Approve defaults to true (JSON omitted -> zero value -> approve) --
	// the endpoint's own name is "approve"; explicit {"approve": false} is
	// how a reject happens (06-api-design.md has no separate reject route).
	Approve   *bool `json:"approve"`
	Confirmed bool  `json:"confirmed"`
}

// ApprovePlan handles POST /plans/:id/approve. Rule BR-3 (approval gate)
// and Rule BR-5 (confirmation required for high-risk/breaking plans) --
// both from golangci/plans/10-business-rules.md.
// @Summary      Approve or reject a fix plan
// @Description  Rule BR-3: applying a fix requires an approved plan. Rule BR-5: high-risk/breaking-change plans require explicit confirmed=true.
// @Tags         Plans
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        id   path string             true "Plan ID"
// @Param        body body approvePlanRequest true  "approve defaults to true; set false to reject"
// @Success      200 {object} storage.FixPlan
// @Failure      400 {object} utils.RestErr "confirmation required for high-risk/breaking plans"
// @Failure      404 {object} utils.RestErr
// @Failure      409 {object} utils.RestErr "plan is not pending an approval decision"
// @Router       /plans/{id}/approve [post]
func ApprovePlan(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req approvePlanRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			restErr := utils.InputErr(err.Error())
			restErr.Code = "VALIDATION_FAILED"
			c.JSON(restErr.Status, restErr)
			return
		}
		approve := req.Approve == nil || *req.Approve

		ctx := c.Request.Context()
		var plan storage.FixPlan
		if err := db.WithContext(ctx).First(&plan, "id = ?", c.Param("id")).Error; err != nil {
			writeLookupErr(c, "plan", c.Param("id"), err)
			return
		}

		if plan.Status != "pending" {
			restErr := utils.ConflictErr(
				"plan %s is %q, not pending an approval decision",
				plan.ID,
				plan.Status,
			)
			restErr.Code = "PLAN_NOT_PENDING"
			c.JSON(restErr.Status, restErr)
			return
		}

		if approve && (plan.RiskLevel == "high" || plan.BreakingChange) && !req.Confirmed {
			restErr := utils.InputErr(
				fmt.Sprintf(
					"plan %s is high-risk/breaking-change and requires explicit confirmation",
					plan.ID,
				),
			)
			restErr.Code = "CONFIRMATION_REQUIRED"
			c.JSON(restErr.Status, restErr)
			return
		}

		role, _ := c.Get("role")
		now := time.Now()
		newStatus := "rejected"
		if approve {
			newStatus = "approved"
		}
		updates := map[string]any{
			"status":      newStatus,
			"approved_by": fmt.Sprintf("%v", role),
			"approved_at": now,
		}
		if err := db.WithContext(ctx).Model(&plan).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, utils.InternalErr("update plan: %v", err))
			return
		}

		plan.Status = newStatus
		plan.ApprovedBy = fmt.Sprintf("%v", role)
		plan.ApprovedAt = &now
		c.JSON(http.StatusOK, plan)
	}
}

func writePlanRequestErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, planner.ErrIssueNotFound):
		c.JSON(http.StatusNotFound, utils.NotFound("%s", err.Error()))
	case errors.Is(err, planner.ErrCrossScanBatch):
		restErr := utils.InputErr(err.Error())
		restErr.Code = "VALIDATION_FAILED"
		c.JSON(restErr.Status, restErr)
	default:
		c.JSON(http.StatusInternalServerError, utils.InternalErr("request plan: %v", err))
	}
}
