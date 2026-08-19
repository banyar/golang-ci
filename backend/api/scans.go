package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"golangci/backend/frontiir/utils"
	"golangci/backend/storage"
	"golangci/backend/worker"
)

type createScanRequest struct {
	RepoRef string `json:"repo_ref" binding:"required"`
	Branch  string `json:"branch"   binding:"required"`
}

// scanLockTTL bounds how long a lock can be held if a worker dies mid-scan
// without releasing it -- prevents a permanently stuck repo+branch.
const scanLockTTL = 30 * time.Minute

// CreateScan handles POST /scans. See golangci/plans/06-api-design.md and
// Rule BR-1 (golangci/plans/10-business-rules.md): one scan/fix at a time
// per repo+branch, enforced here via the Redis lock before any DB write.
// @Summary      Start a lint scan
// @Description  Enqueues a golangci-lint scan for repo_ref+branch. Rule BR-1: only one scan or fix may run at a time per repo+branch.
// @Tags         Scans
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        body body createScanRequest true "Scan target"
// @Success      202 {object} map[string]any
// @Failure      400 {object} utils.RestErr
// @Failure      409 {object} utils.RestErr "a scan or fix is already running for this repo+branch"
// @Router       /scans [post]
func CreateScan(db *gorm.DB, rdb *redis.Client, lock *worker.Lock) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createScanRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			restErr := utils.InputErr(err.Error())
			restErr.Code = "VALIDATION_FAILED"
			c.JSON(restErr.Status, restErr)
			return
		}

		ctx := c.Request.Context()
		lockKey := worker.LockKey(req.RepoRef, req.Branch)
		claimed, err := lock.TryLock(ctx, lockKey, scanLockTTL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utils.InternalErr("lock check failed: %v", err))
			return
		}
		if !claimed {
			restErr := utils.ConflictErr(
				"a scan or fix is already running for %s:%s",
				req.RepoRef,
				req.Branch,
			)
			restErr.Code = "SCAN_LOCKED"
			c.JSON(restErr.Status, restErr)
			return
		}

		role, _ := c.Get("role")
		scan := storage.LintScan{
			RepoRef:     req.RepoRef,
			Branch:      req.Branch,
			Status:      "running",
			TriggeredBy: fmt.Sprintf("%v", role),
		}
		if err := db.WithContext(ctx).Create(&scan).Error; err != nil {
			_ = lock.Unlock(ctx, lockKey)
			c.JSON(http.StatusInternalServerError, utils.InternalErr("create scan: %v", err))
			return
		}

		job := worker.Job{ScanID: scan.ID, RepoRef: scan.RepoRef, Branch: scan.Branch}
		if err := worker.Enqueue(ctx, rdb, job); err != nil {
			// The scan row was created but will never be processed — delete it
			// rather than leave it stuck at status="running" forever.
			db.WithContext(ctx).Delete(&scan)
			_ = lock.Unlock(ctx, lockKey)
			c.JSON(http.StatusInternalServerError, utils.InternalErr("enqueue scan: %v", err))
			return
		}

		c.JSON(http.StatusAccepted, gin.H{"id": scan.ID, "status": scan.Status})
	}
}

// GetScan handles GET /scans/:id.
// @Summary      Get a scan by ID
// @Tags         Scans
// @Security     ApiKeyAuth
// @Produce      json
// @Param        id path string true "Scan ID"
// @Success      200 {object} storage.LintScan
// @Failure      404 {object} utils.RestErr
// @Router       /scans/{id} [get]
func GetScan(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var scan storage.LintScan
		if err := db.WithContext(c.Request.Context()).
			First(&scan, "id = ?", c.Param("id")).
			Error; err != nil {
			writeLookupErr(c, "scan", c.Param("id"), err)
			return
		}
		c.JSON(http.StatusOK, scan)
	}
}

// GetScanIssues handles GET /scans/:id/issues, paginated per the decided
// default in golangci/plans/13-test-plan.md scenario 25.
// @Summary      List issues found by a scan
// @Tags         Scans
// @Security     ApiKeyAuth
// @Produce      json
// @Param        id     path  string true  "Scan ID"
// @Param        limit  query int    false "Max results (default 50, capped at 200)"
// @Param        offset query int    false "Pagination offset (default 0)"
// @Success      200 {array}  storage.LintIssue
// @Failure      404 {object} utils.RestErr
// @Router       /scans/{id}/issues [get]
func GetScanIssues(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var scan storage.LintScan
		if err := db.WithContext(ctx).First(&scan, "id = ?", c.Param("id")).Error; err != nil {
			writeLookupErr(c, "scan", c.Param("id"), err)
			return
		}

		limit, offset := paginationParams(c)

		issues := make([]storage.LintIssue, 0)
		if err := db.WithContext(ctx).
			Where("scan_id = ?", scan.ID).
			Limit(limit).
			Offset(offset).
			Find(&issues).
			Error; err != nil {
			c.JSON(http.StatusInternalServerError, utils.InternalErr("list issues: %v", err))
			return
		}
		c.JSON(http.StatusOK, issues)
	}
}

// GetIssue handles GET /issues/:id.
// @Summary      Get a lint issue by ID
// @Tags         Issues
// @Security     ApiKeyAuth
// @Produce      json
// @Param        id path string true "Issue ID"
// @Success      200 {object} storage.LintIssue
// @Failure      404 {object} utils.RestErr
// @Router       /issues/{id} [get]
func GetIssue(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var issue storage.LintIssue
		if err := db.WithContext(c.Request.Context()).
			First(&issue, "id = ?", c.Param("id")).
			Error; err != nil {
			writeLookupErr(c, "issue", c.Param("id"), err)
			return
		}
		c.JSON(http.StatusOK, issue)
	}
}

func writeLookupErr(c *gin.Context, kind, id string, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, utils.NotFound("%s %s not found", kind, id))
		return
	}
	c.JSON(http.StatusInternalServerError, utils.InternalErr("lookup %s: %v", kind, err))
}

// paginationParams reads limit/offset query params, defaulting to 50/0 and
// capping limit at 200 (golangci/plans/13-test-plan.md scenario 25).
func paginationParams(c *gin.Context) (limit, offset int) {
	limit, offset = 50, 0
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 200 {
		limit = 200
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}
