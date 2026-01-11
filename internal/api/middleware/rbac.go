package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mikedewar/stablerisk/pkg/models"
	"go.uber.org/zap"
)

// RBACMiddleware handles role-based access control
type RBACMiddleware struct {
	logger *zap.Logger
}

// NewRBACMiddleware creates a new RBAC middleware
func NewRBACMiddleware(logger *zap.Logger) *RBACMiddleware {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &RBACMiddleware{
		logger: logger,
	}
}

// RequireRole checks if user has one of the required roles
func (m *RBACMiddleware) RequireRole(allowedRoles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := GetRole(c)
		if userRole == "" {
			m.logger.Warn("RBAC check failed: no role in context",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method))
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Access denied: authentication required",
			})
			c.Abort()
			return
		}

		// Check if user has one of the allowed roles
		role := models.Role(userRole)
		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				m.logger.Debug("RBAC check passed",
					zap.String("user_id", GetUserID(c)),
					zap.String("role", userRole),
					zap.String("path", c.Request.URL.Path))
				c.Next()
				return
			}
		}

		// Access denied
		m.logger.Warn("RBAC check failed: insufficient permissions",
			zap.String("user_id", GetUserID(c)),
			zap.String("user_role", userRole),
			zap.Strings("allowed_roles", rolesToStrings(allowedRoles)),
			zap.String("path", c.Request.URL.Path))

		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Access denied: insufficient permissions",
		})
		c.Abort()
	}
}

// RequireAdmin checks if user is an admin
func (m *RBACMiddleware) RequireAdmin() gin.HandlerFunc {
	return m.RequireRole(models.RoleAdmin)
}

// RequireAnalyst checks if user is admin or analyst
func (m *RBACMiddleware) RequireAnalyst() gin.HandlerFunc {
	return m.RequireRole(models.RoleAdmin, models.RoleAnalyst)
}

// RequireViewer checks if user is admin, analyst, or viewer (any authenticated user)
func (m *RBACMiddleware) RequireViewer() gin.HandlerFunc {
	return m.RequireRole(models.RoleAdmin, models.RoleAnalyst, models.RoleViewer)
}

// HasPermission checks if the current user has a specific permission
// This is a helper function for more granular permission checks
func HasPermission(c *gin.Context, permission Permission) bool {
	role := models.Role(GetRole(c))
	return roleHasPermission(role, permission)
}

// Permission represents a specific action permission
type Permission string

const (
	// Read permissions
	PermissionReadOutliers      Permission = "read:outliers"
	PermissionReadTransactions  Permission = "read:transactions"
	PermissionReadStatistics    Permission = "read:statistics"
	PermissionReadUsers         Permission = "read:users"

	// Write permissions
	PermissionWriteOutliers     Permission = "write:outliers"
	PermissionTriggerDetection  Permission = "trigger:detection"
	PermissionManageUsers       Permission = "manage:users"
	PermissionManageSystem      Permission = "manage:system"
)

// roleHasPermission checks if a role has a specific permission
func roleHasPermission(role models.Role, permission Permission) bool {
	switch role {
	case models.RoleAdmin:
		// Admin has all permissions
		return true

	case models.RoleAnalyst:
		// Analyst can read everything and write outliers (acknowledge)
		switch permission {
		case PermissionReadOutliers,
			PermissionReadTransactions,
			PermissionReadStatistics,
			PermissionWriteOutliers,
			PermissionTriggerDetection:
			return true
		default:
			return false
		}

	case models.RoleViewer:
		// Viewer can only read
		switch permission {
		case PermissionReadOutliers,
			PermissionReadTransactions,
			PermissionReadStatistics:
			return true
		default:
			return false
		}

	default:
		return false
	}
}

// rolesToStrings converts role slice to string slice for logging
func rolesToStrings(roles []models.Role) []string {
	result := make([]string, len(roles))
	for i, role := range roles {
		result[i] = string(role)
	}
	return result
}
