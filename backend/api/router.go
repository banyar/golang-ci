package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golangci/backend/frontiir/utils"
	"golangci/backend/planner"
	"golangci/backend/worker"
	"golangci/docs"
	"gorm.io/gorm"
)

// APIPrefix matches this repo's existing versioning convention — see
// frontiir/api/router.go and every frontiir/api/*_routes.go's APIPrefix
// constant. Not a new convention, the one already in the codebase.
const APIPrefix = "/api/v1"

// stub is an M1 placeholder handler. The scanner/parser/planner/fixer/
// history logic behind each route is M2+ scope — see
// golangci/plans/15-implementation-plan.md. It proves the route is
// reachable (and RBAC-checked) without any business logic behind it yet.
// @Summary      Not implemented yet
// @Description  History and config endpoints are M5+ scope -- see golangci/plans/15-implementation-plan.md.
// @Tags         Unimplemented
// @Security     ApiKeyAuth
// @Produce      json
// @Failure      501 {object} utils.RestErr
// @Router       /history/scans [get]
// @Router       /history/fixes [get]
// @Router       /history/rollbacks [get]
// @Router       /config [get]
func stub(c *gin.Context) {
	restErr := &utils.RestErr{
		Status:  http.StatusNotImplemented,
		Message: "not implemented yet — see golangci/plans/15-implementation-plan.md",
		Code:    "NOT_IMPLEMENTED_YET",
	}
	c.JSON(restErr.Status, restErr)
}

// NewRouter registers golangci/plans/06-api-design.md's 14 designed
// endpoints under APIPrefix, each RBAC-guarded per that file's "Min. role"
// column, plus one addition: GET /rollbacks/:id (not in the original 14 --
// added for consistency with /scans, /plans, /fixes, which all have a
// paired GET :id; see golangci/api/rollback.go's doc comment). M2 wired
// the 4 scan/issue endpoints, M3 the 2 plan endpoints, M4 the approve+fix
// endpoints, this pass the 2 rollback endpoints (git write happens in the
// worker, not here). The remaining 4 (history x3, config) stay M1's 501
// stubs until M5+.
func NewRouter(
	db *gorm.DB,
	rdb *redis.Client,
	lock *worker.Lock,
	plannerSvc *planner.Service,
) *gin.Engine {
	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	docs.SwaggerInfo.BasePath = APIPrefix
	r.GET(
		"/swagger/*any",
		ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")),
	)

	v1 := r.Group(APIPrefix)

	v1.POST("/scans", RoleAuth("operator"), CreateScan(db, rdb, lock))
	v1.GET("/scans/:id", RoleAuth("viewer"), GetScan(db))
	v1.GET("/scans/:id/issues", RoleAuth("viewer"), GetScanIssues(db))
	v1.GET("/issues/:id", RoleAuth("viewer"), GetIssue(db))
	v1.POST("/plans", RoleAuth("viewer"), CreatePlan(plannerSvc, rdb))
	v1.GET("/plans/:id", RoleAuth("viewer"), GetPlan(db))
	v1.POST("/plans/:id/approve", RoleAuth("approver"), ApprovePlan(db))
	v1.POST("/fixes", RoleAuth("operator"), CreateFix(db, rdb, lock))
	v1.GET("/fixes/:id", RoleAuth("viewer"), GetFix(db))
	v1.POST("/rollbacks", RoleAuth("operator"), CreateRollback(db, rdb, lock))
	v1.GET("/rollbacks/:id", RoleAuth("viewer"), GetRollback(db))
	v1.GET("/history/scans", RoleAuth("viewer"), stub)
	v1.GET("/history/fixes", RoleAuth("viewer"), stub)
	v1.GET("/history/rollbacks", RoleAuth("viewer"), stub)
	v1.GET("/config", RoleAuth("admin"), stub)

	return r
}
