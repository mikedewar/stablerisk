package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mikedewar/stablerisk/internal/api/handlers"
	"github.com/mikedewar/stablerisk/internal/security"
	"github.com/mikedewar/stablerisk/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Open SQLite with automatic timestamp parsing
	// The go-sqlite3 driver will handle time.Time if we use DATETIME type and proper format
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)

	// Create users table
	// Use DATETIME type for timestamp columns
	_, err = db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			email TEXT,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_login DATETIME,
			is_active INTEGER DEFAULT 1
		)
	`)
	require.NoError(t, err)

	// Insert test user
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
	now := time.Now()
	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, role, created_at, updated_at, last_login, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-user-id", "testuser", "test@example.com", string(passwordHash), "admin", now, now, now, 1)
	require.NoError(t, err)

	return db
}

func setupTestJWTManager() *security.JWTManager {
	return security.NewJWTManager(security.JWTConfig{
		SecretKey:          "test-secret-key-32-characters!!",
		Issuer:             "stablerisk-test",
		Audience:           "stablerisk-api-test",
		AccessTokenExpiry:  1 * time.Hour,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	})
}

func TestAuthHandler_Login_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtManager := setupTestJWTManager()
	handler := handlers.NewAuthHandler(db, jwtManager, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/login", handler.Login)

	loginReq := models.LoginRequest{
		Username: "testuser",
		Password: "testpass123",
	}
	body, _ := json.Marshal(loginReq)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.Token)
	assert.NotEmpty(t, response.RefreshToken)
	assert.Greater(t, response.ExpiresIn, int64(0))
	assert.NotNil(t, response.User)
	assert.Equal(t, "testuser", response.User.Username)
	assert.Equal(t, models.RoleAdmin, response.User.Role)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtManager := setupTestJWTManager()
	handler := handlers.NewAuthHandler(db, jwtManager, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/login", handler.Login)

	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
	}{
		{"wrong password", "testuser", "wrongpassword", http.StatusUnauthorized},
		{"wrong username", "nonexistent", "testpass123", http.StatusUnauthorized},
		{"empty password", "testuser", "", http.StatusBadRequest},      // Required field validation
		{"empty username", "", "testpass123", http.StatusBadRequest}, // Required field validation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loginReq := models.LoginRequest{
				Username: tt.username,
				Password: tt.password,
			}
			body, _ := json.Marshal(loginReq)

			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestAuthHandler_GetProfile_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtManager := setupTestJWTManager()
	handler := handlers.NewAuthHandler(db, jwtManager, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/profile", func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		handler.GetProfile(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var user models.User
	err := json.Unmarshal(w.Body.Bytes(), &user)
	require.NoError(t, err)

	assert.Equal(t, "test-user-id", user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, models.RoleAdmin, user.Role)
	assert.True(t, user.IsActive)
}

func TestAuthHandler_RefreshToken_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtManager := setupTestJWTManager()
	handler := handlers.NewAuthHandler(db, jwtManager, nil)

	// First, create a refresh token
	user := &models.User{
		ID:       "test-user-id",
		Username: "testuser",
		Role:     models.RoleAdmin,
	}
	refreshToken, err := jwtManager.GenerateRefreshToken(user)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/refresh", handler.RefreshToken)

	refreshReq := models.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}
	body, _ := json.Marshal(refreshReq)

	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["token"])
	assert.Greater(t, response["expires_in"], float64(0))
}

func TestAuthHandler_RefreshToken_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtManager := setupTestJWTManager()
	handler := handlers.NewAuthHandler(db, jwtManager, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/refresh", handler.RefreshToken)

	refreshReq := models.RefreshTokenRequest{
		RefreshToken: "invalid-token",
	}
	body, _ := json.Marshal(refreshReq)

	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
