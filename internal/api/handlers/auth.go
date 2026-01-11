package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mikedewar/stablerisk/internal/security"
	"github.com/mikedewar/stablerisk/pkg/models"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	db         *sql.DB
	jwtManager *security.JWTManager
	logger     *zap.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *sql.DB, jwtManager *security.JWTManager, logger *zap.Logger) *AuthHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &AuthHandler{
		db:         db,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request body",
		})
		return
	}

	// Query user from database
	var user models.User
	err := h.db.QueryRow(`
		SELECT id, username, email, password_hash, role, created_at, updated_at, last_login, is_active
		FROM users
		WHERE username = ? AND is_active = 1
	`, req.Username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLogin,
		&user.IsActive,
	)

	if err == sql.ErrNoRows {
		h.logger.Warn("Login failed: user not found",
			zap.String("username", req.Username))
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Invalid username or password",
		})
		return
	}

	if err != nil {
		h.logger.Error("Database error during login",
			zap.Error(err),
			zap.String("username", req.Username))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to process login",
		})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.logger.Warn("Login failed: invalid password",
			zap.String("username", req.Username))
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Invalid username or password",
		})
		return
	}

	// Generate tokens
	accessToken, err := h.jwtManager.GenerateAccessToken(&user)
	if err != nil {
		h.logger.Error("Failed to generate access token",
			zap.Error(err),
			zap.String("user_id", user.ID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to generate token",
		})
		return
	}

	refreshToken, err := h.jwtManager.GenerateRefreshToken(&user)
	if err != nil {
		h.logger.Error("Failed to generate refresh token",
			zap.Error(err),
			zap.String("user_id", user.ID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to generate token",
		})
		return
	}

	// Update last login time
	_, err = h.db.Exec(`
		UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?
	`, user.ID)
	if err != nil {
		h.logger.Error("Failed to update last login",
			zap.Error(err),
			zap.String("user_id", user.ID))
	}

	h.logger.Info("User logged in successfully",
		zap.String("user_id", user.ID),
		zap.String("username", user.Username))

	// Return response
	c.JSON(http.StatusOK, models.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(h.jwtManager.GetAccessTokenExpiry().Seconds()),
		User:         &user,
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request body",
		})
		return
	}

	// Validate refresh token
	claims, err := h.jwtManager.ValidateToken(req.RefreshToken)
	if err != nil {
		h.logger.Warn("Invalid refresh token",
			zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Invalid or expired refresh token",
		})
		return
	}

	// Query user to ensure still active
	var user models.User
	err = h.db.QueryRow(`
		SELECT id, username, email, role, created_at, updated_at, last_login, is_active
		FROM users
		WHERE id = ? AND is_active = 1
	`, claims.UserID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLogin,
		&user.IsActive,
	)

	if err == sql.ErrNoRows {
		h.logger.Warn("Token refresh failed: user not found or inactive",
			zap.String("user_id", claims.UserID))
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "User not found or inactive",
		})
		return
	}

	if err != nil {
		h.logger.Error("Database error during token refresh",
			zap.Error(err),
			zap.String("user_id", claims.UserID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to refresh token",
		})
		return
	}

	// Generate new access token
	accessToken, err := h.jwtManager.GenerateAccessToken(&user)
	if err != nil {
		h.logger.Error("Failed to generate access token",
			zap.Error(err),
			zap.String("user_id", user.ID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to generate token",
		})
		return
	}

	h.logger.Debug("Token refreshed",
		zap.String("user_id", user.ID))

	c.JSON(http.StatusOK, gin.H{
		"token":      accessToken,
		"expires_in": int64(h.jwtManager.GetAccessTokenExpiry().Seconds()),
	})
}

// GetProfile returns the current user's profile
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	var user models.User
	err := h.db.QueryRow(`
		SELECT id, username, email, role, created_at, updated_at, last_login, is_active
		FROM users
		WHERE id = ?
	`, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLogin,
		&user.IsActive,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "User not found",
		})
		return
	}

	if err != nil {
		h.logger.Error("Database error fetching user profile",
			zap.Error(err),
			zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to fetch profile",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}
