package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/paw-chain/paw/control-center/admin-api/types"
)

// RBACMiddleware provides role-based access control
type RBACMiddleware struct {
	auditLogger AuditLogger
}

// NewRBACMiddleware creates a new RBAC middleware
func NewRBACMiddleware(auditLogger AuditLogger) *RBACMiddleware {
	return &RBACMiddleware{
		auditLogger: auditLogger,
	}
}

// RequireRole creates a middleware that requires a specific minimum role
func (rbac *RBACMiddleware) RequireRole(minRole types.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "No role information found",
			})
			c.Abort()
			return
		}

		userRole, ok := roleValue.(types.Role)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Invalid role format",
			})
			c.Abort()
			return
		}

		if !hasMinimumRole(userRole, minRole) {
			userID, _ := c.Get("user_id")
			username, _ := c.Get("username")

			rbac.auditLogger.LogAction(
				getString(userID),
				getString(username),
				"access_denied",
				c.Request.URL.Path,
				c.ClientIP(),
				map[string]interface{}{
					"required_role": minRole,
					"user_role":     userRole,
					"method":        c.Request.Method,
				},
				false,
				fmt.Errorf("insufficient role: %s required, user has %s", minRole, userRole),
			)

			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": fmt.Sprintf("Minimum role '%s' is required", minRole),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermissions creates a middleware that requires specific permissions
func (rbac *RBACMiddleware) RequirePermissions(permissions ...types.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "No role information found",
			})
			c.Abort()
			return
		}

		userRole, ok := roleValue.(types.Role)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Invalid role format",
			})
			c.Abort()
			return
		}

		// Check if user has all required permissions
		missingPermissions := []types.Permission{}
		for _, perm := range permissions {
			if !userRole.HasPermission(perm) {
				missingPermissions = append(missingPermissions, perm)
			}
		}

		if len(missingPermissions) > 0 {
			userID, _ := c.Get("user_id")
			username, _ := c.Get("username")

			rbac.auditLogger.LogAction(
				getString(userID),
				getString(username),
				"permission_denied",
				c.Request.URL.Path,
				c.ClientIP(),
				map[string]interface{}{
					"required_permissions": permissions,
					"missing_permissions":  missingPermissions,
					"user_role":            userRole,
					"method":               c.Request.Method,
				},
				false,
				fmt.Errorf("missing permissions: %v", missingPermissions),
			)

			c.JSON(http.StatusForbidden, gin.H{
				"error":              "forbidden",
				"message":            "Insufficient permissions",
				"required":           permissions,
				"missing":            missingPermissions,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission creates a middleware that requires at least one of the specified permissions
func (rbac *RBACMiddleware) RequireAnyPermission(permissions ...types.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "No role information found",
			})
			c.Abort()
			return
		}

		userRole, ok := roleValue.(types.Role)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Invalid role format",
			})
			c.Abort()
			return
		}

		// Check if user has at least one required permission
		hasAny := false
		for _, perm := range permissions {
			if userRole.HasPermission(perm) {
				hasAny = true
				break
			}
		}

		if !hasAny {
			userID, _ := c.Get("user_id")
			username, _ := c.Get("username")

			rbac.auditLogger.LogAction(
				getString(userID),
				getString(username),
				"permission_denied",
				c.Request.URL.Path,
				c.ClientIP(),
				map[string]interface{}{
					"required_any_of": permissions,
					"user_role":       userRole,
					"method":          c.Request.Method,
				},
				false,
				fmt.Errorf("user does not have any of the required permissions: %v", permissions),
			)

			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "At least one of the required permissions is needed",
				"required_any_of": permissions,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireExactRole creates a middleware that requires an exact role match
func (rbac *RBACMiddleware) RequireExactRole(role types.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "No role information found",
			})
			c.Abort()
			return
		}

		userRole, ok := roleValue.(types.Role)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Invalid role format",
			})
			c.Abort()
			return
		}

		if userRole != role {
			userID, _ := c.Get("user_id")
			username, _ := c.Get("username")

			rbac.auditLogger.LogAction(
				getString(userID),
				getString(username),
				"access_denied",
				c.Request.URL.Path,
				c.ClientIP(),
				map[string]interface{}{
					"required_role": role,
					"user_role":     userRole,
					"method":        c.Request.Method,
				},
				false,
				fmt.Errorf("exact role match required: %s, user has %s", role, userRole),
			)

			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": fmt.Sprintf("Exact role '%s' is required", role),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyRole creates a middleware that allows any of the specified roles
func (rbac *RBACMiddleware) RequireAnyRole(roles ...types.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "No role information found",
			})
			c.Abort()
			return
		}

		userRole, ok := roleValue.(types.Role)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Invalid role format",
			})
			c.Abort()
			return
		}

		// Check if user has any of the allowed roles
		hasRole := false
		for _, allowedRole := range roles {
			if userRole == allowedRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			userID, _ := c.Get("user_id")
			username, _ := c.Get("username")

			rbac.auditLogger.LogAction(
				getString(userID),
				getString(username),
				"access_denied",
				c.Request.URL.Path,
				c.ClientIP(),
				map[string]interface{}{
					"allowed_roles": roles,
					"user_role":     userRole,
					"method":        c.Request.Method,
				},
				false,
				fmt.Errorf("user role %s not in allowed roles: %v", userRole, roles),
			)

			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "User role is not authorized for this endpoint",
				"allowed_roles": roles,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// hasMinimumRole checks if user role meets the minimum required role
func hasMinimumRole(userRole, minRole types.Role) bool {
	roleLevel := map[types.Role]int{
		types.RoleReadOnly:  1,
		types.RoleOperator:  2,
		types.RoleAdmin:     3,
		types.RoleSuperUser: 4,
	}

	return roleLevel[userRole] >= roleLevel[minRole]
}

// getString safely converts interface{} to string
func getString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
