package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mikedewar/stablerisk/internal/api/middleware"
	"github.com/mikedewar/stablerisk/internal/security"
	"github.com/mikedewar/stablerisk/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestJWTManager() *security.JWTManager {
	return security.NewJWTManager(security.JWTConfig{
		SecretKey:          "test-secret-key-32-characters!!",
		Issuer:             "stablerisk-test",
		Audience:           "stablerisk-api-test",
		AccessTokenExpiry:  1 * time.Hour,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	})
}

func TestAuthMiddleware_Authenticate_Success(t *testing.T) {
	jwtManager := setupTestJWTManager()
	authMiddleware := middleware.NewAuthMiddleware(jwtManager, nil)

	// Generate a valid token
	user := &models.User{
		ID:       "test-user-id",
		Username: "testuser",
		Role:     models.RoleAdmin,
	}
	token, err := jwtManager.GenerateAccessToken(user)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", authMiddleware.Authenticate(), func(c *gin.Context) {
		userID := middleware.GetUserID(c)
		username := middleware.GetUsername(c)
		role := middleware.GetRole(c)

		c.JSON(http.StatusOK, gin.H{
			"user_id":  userID,
			"username": username,
			"role":     role,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_Authenticate_MissingToken(t *testing.T) {
	jwtManager := setupTestJWTManager()
	authMiddleware := middleware.NewAuthMiddleware(jwtManager, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", authMiddleware.Authenticate(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_Authenticate_InvalidToken(t *testing.T) {
	jwtManager := setupTestJWTManager()
	authMiddleware := middleware.NewAuthMiddleware(jwtManager, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", authMiddleware.Authenticate(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_Authenticate_QueryToken(t *testing.T) {
	jwtManager := setupTestJWTManager()
	authMiddleware := middleware.NewAuthMiddleware(jwtManager, nil)

	// Generate a valid token
	user := &models.User{
		ID:       "test-user-id",
		Username: "testuser",
		Role:     models.RoleAdmin,
	}
	token, err := jwtManager.GenerateAccessToken(user)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws", authMiddleware.Authenticate(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "authenticated"})
	})

	req := httptest.NewRequest(http.MethodGet, "/ws?token="+token, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_Optional_WithToken(t *testing.T) {
	jwtManager := setupTestJWTManager()
	authMiddleware := middleware.NewAuthMiddleware(jwtManager, nil)

	user := &models.User{
		ID:       "test-user-id",
		Username: "testuser",
		Role:     models.RoleAdmin,
	}
	token, err := jwtManager.GenerateAccessToken(user)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/public", authMiddleware.Optional(), func(c *gin.Context) {
		userID := middleware.GetUserID(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_Optional_WithoutToken(t *testing.T) {
	jwtManager := setupTestJWTManager()
	authMiddleware := middleware.NewAuthMiddleware(jwtManager, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/public", authMiddleware.Optional(), func(c *gin.Context) {
		userID := middleware.GetUserID(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_GetHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, "user-123")
		c.Set(middleware.ContextKeyUsername, "testuser")
		c.Set(middleware.ContextKeyRole, "admin")

		userID := middleware.GetUserID(c)
		username := middleware.GetUsername(c)
		role := middleware.GetRole(c)

		assert.Equal(t, "user-123", userID)
		assert.Equal(t, "testuser", username)
		assert.Equal(t, "admin", role)

		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
