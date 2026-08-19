package api

import (
	"encoding/json"
	"os"

	"github.com/gin-gonic/gin"
	"golangci/backend/frontiir/utils"
)

// roleRank defines the hierarchy decided in golangci/plans/03-proposed-workflow.md
// §Actors & Roles: viewer < approver < operator < admin.
var roleRank = map[string]int{
	"viewer":   1,
	"approver": 2,
	"operator": 3,
	"admin":    4,
}

type permissionsFile struct {
	Keys map[string]string `json:"keys"`
}

var apiKeyRoles map[string]string

// LoadPermissions loads backend/config/permissions.json once at startup.
// This is a self-contained loader (not a reuse of frontiir/middleware's
// api_key_authz.go) — see the M1 plan for why: the dashboard is a
// standalone binary and shouldn't cross-import frontiir/middleware.
func LoadPermissions(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var pf permissionsFile
	if err := json.Unmarshal(b, &pf); err != nil {
		return err
	}
	apiKeyRoles = pf.Keys
	return nil
}

// RoleAuth enforces that the caller's X-API-Key resolves to a role at or
// above minRole. See golangci/plans/12-component-design.md §Security controls
// and golangci/plans/06-api-design.md for the per-endpoint minimum role.
func RoleAuth(minRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			restErr := utils.ForbiddenErr("missing X-API-Key header")
			restErr.Code = "MISSING_API_KEY"
			c.AbortWithStatusJSON(restErr.Status, restErr)
			return
		}

		role, ok := apiKeyRoles[key]
		if !ok {
			restErr := utils.ForbiddenErr("unrecognized API key")
			restErr.Code = "MISSING_API_KEY"
			c.AbortWithStatusJSON(restErr.Status, restErr)
			return
		}

		if roleRank[role] < roleRank[minRole] {
			restErr := utils.ForbiddenErr(
				"role %q is below the required minimum role %q",
				role,
				minRole,
			)
			restErr.Code = "INSUFFICIENT_ROLE"
			c.AbortWithStatusJSON(restErr.Status, restErr)
			return
		}

		c.Set("role", role)
		c.Next()
	}
}
