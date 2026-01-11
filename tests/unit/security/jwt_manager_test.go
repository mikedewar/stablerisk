package security

import (
	"testing"
	"time"

	"github.com/mikedewar/stablerisk/internal/security"
	"github.com/mikedewar/stablerisk/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTManager_GenerateAccessToken(t *testing.T) {
	jwtManager := security.NewJWTManager(security.JWTConfig{
		SecretKey:          "test-secret-key-32-characters!!",
		Issuer:             "stablerisk-test",
		Audience:           "stablerisk-api-test",
		AccessTokenExpiry:  1 * time.Hour,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	})

	user := &models.User{
		ID:       "test-user-id",
		Username: "testuser",
		Role:     models.RoleAdmin,
	}

	token, err := jwtManager.GenerateAccessToken(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTManager_GenerateRefreshToken(t *testing.T) {
	jwtManager := security.NewJWTManager(security.JWTConfig{
		SecretKey:          "test-secret-key-32-characters!!",
		Issuer:             "stablerisk-test",
		Audience:           "stablerisk-api-test",
		AccessTokenExpiry:  1 * time.Hour,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	})

	user := &models.User{
		ID:       "test-user-id",
		Username: "testuser",
		Role:     models.RoleAdmin,
	}

	token, err := jwtManager.GenerateRefreshToken(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTManager_ValidateToken_Success(t *testing.T) {
	jwtManager := security.NewJWTManager(security.JWTConfig{
		SecretKey:          "test-secret-key-32-characters!!",
		Issuer:             "stablerisk-test",
		Audience:           "stablerisk-api-test",
		AccessTokenExpiry:  1 * time.Hour,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	})

	user := &models.User{
		ID:       "test-user-id",
		Username: "testuser",
		Role:     models.RoleAdmin,
	}

	token, err := jwtManager.GenerateAccessToken(user)
	require.NoError(t, err)

	claims, err := jwtManager.ValidateToken(token)
	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, user.Role, claims.Role)
}

func TestJWTManager_ValidateToken_InvalidToken(t *testing.T) {
	jwtManager := security.NewJWTManager(security.JWTConfig{
		SecretKey:          "test-secret-key-32-characters!!",
		Issuer:             "stablerisk-test",
		Audience:           "stablerisk-api-test",
		AccessTokenExpiry:  1 * time.Hour,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	})

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid format", "invalid-token"},
		{"malformed jwt", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := jwtManager.ValidateToken(tt.token)
			assert.Error(t, err)
			assert.Nil(t, claims)
		})
	}
}

func TestJWTManager_ValidateToken_ExpiredToken(t *testing.T) {
	// Create a JWT manager with very short expiry
	jwtManager := security.NewJWTManager(security.JWTConfig{
		SecretKey:          "test-secret-key-32-characters!!",
		Issuer:             "stablerisk-test",
		Audience:           "stablerisk-api-test",
		AccessTokenExpiry:  1 * time.Nanosecond, // Immediately expired
		RefreshTokenExpiry: 1 * time.Nanosecond,
	})

	user := &models.User{
		ID:       "test-user-id",
		Username: "testuser",
		Role:     models.RoleAdmin,
	}

	token, err := jwtManager.GenerateAccessToken(user)
	require.NoError(t, err)

	// Wait a moment to ensure expiration
	time.Sleep(10 * time.Millisecond)

	claims, err := jwtManager.ValidateToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTManager_ValidateToken_WrongSecret(t *testing.T) {
	// Create token with one secret
	jwtManager1 := security.NewJWTManager(security.JWTConfig{
		SecretKey:          "test-secret-key-32-characters!!",
		Issuer:             "stablerisk-test",
		Audience:           "stablerisk-api-test",
		AccessTokenExpiry:  1 * time.Hour,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	})

	user := &models.User{
		ID:       "test-user-id",
		Username: "testuser",
		Role:     models.RoleAdmin,
	}

	token, err := jwtManager1.GenerateAccessToken(user)
	require.NoError(t, err)

	// Try to validate with different secret
	jwtManager2 := security.NewJWTManager(security.JWTConfig{
		SecretKey:          "different-secret-key-32-chars!",
		Issuer:             "stablerisk-test",
		Audience:           "stablerisk-api-test",
		AccessTokenExpiry:  1 * time.Hour,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	})

	claims, err := jwtManager2.ValidateToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTManager_GetExpiry(t *testing.T) {
	accessExpiry := 2 * time.Hour
	refreshExpiry := 14 * 24 * time.Hour

	jwtManager := security.NewJWTManager(security.JWTConfig{
		SecretKey:          "test-secret-key-32-characters!!",
		Issuer:             "stablerisk-test",
		Audience:           "stablerisk-api-test",
		AccessTokenExpiry:  accessExpiry,
		RefreshTokenExpiry: refreshExpiry,
	})

	assert.Equal(t, accessExpiry, jwtManager.GetAccessTokenExpiry())
	assert.Equal(t, refreshExpiry, jwtManager.GetRefreshTokenExpiry())
}

func TestJWTManager_DifferentRoles(t *testing.T) {
	jwtManager := security.NewJWTManager(security.JWTConfig{
		SecretKey:          "test-secret-key-32-characters!!",
		Issuer:             "stablerisk-test",
		Audience:           "stablerisk-api-test",
		AccessTokenExpiry:  1 * time.Hour,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	})

	roles := []models.Role{
		models.RoleAdmin,
		models.RoleAnalyst,
		models.RoleViewer,
	}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			user := &models.User{
				ID:       "test-user-id",
				Username: "testuser",
				Role:     role,
			}

			token, err := jwtManager.GenerateAccessToken(user)
			require.NoError(t, err)

			claims, err := jwtManager.ValidateToken(token)
			require.NoError(t, err)
			assert.Equal(t, role, claims.Role)
		})
	}
}
