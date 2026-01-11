package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mikedewar/stablerisk/internal/api/middleware"
	"github.com/mikedewar/stablerisk/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestRBACMiddleware_RequireAdmin_Success(t *testing.T) {
	rbacMiddleware := middleware.NewRBACMiddleware(nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin", func(c *gin.Context) {
		c.Set(middleware.ContextKeyRole, string(models.RoleAdmin))
	}, rbacMiddleware.RequireAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRBACMiddleware_RequireAdmin_Forbidden(t *testing.T) {
	rbacMiddleware := middleware.NewRBACMiddleware(nil)

	tests := []struct {
		name string
		role models.Role
	}{
		{"analyst", models.RoleAnalyst},
		{"viewer", models.RoleViewer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.GET("/admin", func(c *gin.Context) {
				c.Set(middleware.ContextKeyRole, string(tt.role))
			}, rbacMiddleware.RequireAdmin(), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
			})

			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

func TestRBACMiddleware_RequireAnalyst_Success(t *testing.T) {
	rbacMiddleware := middleware.NewRBACMiddleware(nil)

	tests := []struct {
		name string
		role models.Role
	}{
		{"admin", models.RoleAdmin},
		{"analyst", models.RoleAnalyst},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.GET("/analyst", func(c *gin.Context) {
				c.Set(middleware.ContextKeyRole, string(tt.role))
			}, rbacMiddleware.RequireAnalyst(), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "analyst access granted"})
			})

			req := httptest.NewRequest(http.MethodGet, "/analyst", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestRBACMiddleware_RequireAnalyst_Forbidden(t *testing.T) {
	rbacMiddleware := middleware.NewRBACMiddleware(nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/analyst", func(c *gin.Context) {
		c.Set(middleware.ContextKeyRole, string(models.RoleViewer))
	}, rbacMiddleware.RequireAnalyst(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "analyst access granted"})
	})

	req := httptest.NewRequest(http.MethodGet, "/analyst", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRBACMiddleware_RequireViewer_Success(t *testing.T) {
	rbacMiddleware := middleware.NewRBACMiddleware(nil)

	tests := []struct {
		name string
		role models.Role
	}{
		{"admin", models.RoleAdmin},
		{"analyst", models.RoleAnalyst},
		{"viewer", models.RoleViewer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.GET("/viewer", func(c *gin.Context) {
				c.Set(middleware.ContextKeyRole, string(tt.role))
			}, rbacMiddleware.RequireViewer(), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "viewer access granted"})
			})

			req := httptest.NewRequest(http.MethodGet, "/viewer", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestRBACMiddleware_RequireRole_NoRoleInContext(t *testing.T) {
	rbacMiddleware := middleware.NewRBACMiddleware(nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", rbacMiddleware.RequireAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHasPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		role       models.Role
		permission middleware.Permission
		expected   bool
	}{
		// Admin permissions
		{"admin read outliers", models.RoleAdmin, middleware.PermissionReadOutliers, true},
		{"admin write outliers", models.RoleAdmin, middleware.PermissionWriteOutliers, true},
		{"admin manage users", models.RoleAdmin, middleware.PermissionManageUsers, true},
		{"admin manage system", models.RoleAdmin, middleware.PermissionManageSystem, true},

		// Analyst permissions
		{"analyst read outliers", models.RoleAnalyst, middleware.PermissionReadOutliers, true},
		{"analyst write outliers", models.RoleAnalyst, middleware.PermissionWriteOutliers, true},
		{"analyst trigger detection", models.RoleAnalyst, middleware.PermissionTriggerDetection, true},
		{"analyst manage users", models.RoleAnalyst, middleware.PermissionManageUsers, false},
		{"analyst manage system", models.RoleAnalyst, middleware.PermissionManageSystem, false},

		// Viewer permissions
		{"viewer read outliers", models.RoleViewer, middleware.PermissionReadOutliers, true},
		{"viewer read statistics", models.RoleViewer, middleware.PermissionReadStatistics, true},
		{"viewer write outliers", models.RoleViewer, middleware.PermissionWriteOutliers, false},
		{"viewer trigger detection", models.RoleViewer, middleware.PermissionTriggerDetection, false},
		{"viewer manage users", models.RoleViewer, middleware.PermissionManageUsers, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				c.Set(middleware.ContextKeyRole, string(tt.role))
				hasPermission := middleware.HasPermission(c, tt.permission)
				assert.Equal(t, tt.expected, hasPermission)
				c.JSON(http.StatusOK, gin.H{"has_permission": hasPermission})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}
