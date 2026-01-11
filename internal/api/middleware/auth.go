package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mikedewar/stablerisk/internal/security"
	"go.uber.org/zap"
)

const (
	// ContextKeyUserID is the context key for user ID
	ContextKeyUserID = "user_id"
	// ContextKeyUsername is the context key for username
	ContextKeyUsername = "username"
	// ContextKeyRole is the context key for user role
	ContextKeyRole = "user_role"
	// ContextKeyClaims is the context key for JWT claims
	ContextKeyClaims = "jwt_claims"
)

// AuthMiddleware creates authentication middleware
type AuthMiddleware struct {
	jwtManager *security.JWTManager
	logger     *zap.Logger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtManager *security.JWTManager, logger *zap.Logger) *AuthMiddleware {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &AuthMiddleware{
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// Authenticate validates JWT token and adds claims to context
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractToken(c)
		if token == "" {
			m.logger.Debug("Missing authentication token",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Missing authentication token",
			})
			c.Abort()
			return
		}

		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil {
			m.logger.Debug("Invalid authentication token",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Add claims to context for downstream handlers
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUsername, claims.Username)
		c.Set(ContextKeyRole, claims.Role)
		c.Set(ContextKeyClaims, claims)

		m.logger.Debug("User authenticated",
			zap.String("user_id", claims.UserID),
			zap.String("username", claims.Username),
			zap.String("role", string(claims.Role)))

		c.Next()
	}
}

// Optional makes authentication optional - validates token if present but doesn't require it
func (m *AuthMiddleware) Optional() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractToken(c)
		if token == "" {
			// No token provided, continue without authentication
			c.Next()
			return
		}

		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil {
			// Invalid token, but don't block request
			m.logger.Debug("Invalid token in optional auth",
				zap.Error(err))
			c.Next()
			return
		}

		// Add claims to context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUsername, claims.Username)
		c.Set(ContextKeyRole, claims.Role)
		c.Set(ContextKeyClaims, claims)

		c.Next()
	}
}

// extractToken extracts JWT token from Authorization header or query parameter
func (m *AuthMiddleware) extractToken(c *gin.Context) string {
	// Try Authorization header first (Bearer token)
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// Try query parameter (for WebSocket connections)
	token := c.Query("token")
	if token != "" {
		return token
	}

	// Try X-Auth-Token header (alternative)
	token = c.GetHeader("X-Auth-Token")
	if token != "" {
		return token
	}

	return ""
}

// GetUserID retrieves user ID from context
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get(ContextKeyUserID); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// GetUsername retrieves username from context
func GetUsername(c *gin.Context) string {
	if username, exists := c.Get(ContextKeyUsername); exists {
		if name, ok := username.(string); ok {
			return name
		}
	}
	return ""
}

// GetRole retrieves user role from context
func GetRole(c *gin.Context) string {
	if role, exists := c.Get(ContextKeyRole); exists {
		if r, ok := role.(string); ok {
			return r
		}
	}
	return ""
}

// GetClaims retrieves JWT claims from context
func GetClaims(c *gin.Context) *security.Claims {
	if claims, exists := c.Get(ContextKeyClaims); exists {
		if c, ok := claims.(*security.Claims); ok {
			return c
		}
	}
	return nil
}
