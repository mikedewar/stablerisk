package security

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mikedewar/stablerisk/pkg/models"
)

// JWTManager handles JWT token generation and validation
type JWTManager struct {
	secretKey          []byte
	issuer             string
	audience           string
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

// Claims represents JWT claims
type Claims struct {
	UserID   string      `json:"user_id"`
	Username string      `json:"username"`
	Role     models.Role `json:"role"`
	jwt.RegisteredClaims
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey          string
	Issuer             string
	Audience           string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(config JWTConfig) *JWTManager {
	return &JWTManager{
		secretKey:          []byte(config.SecretKey),
		issuer:             config.Issuer,
		audience:           config.Audience,
		accessTokenExpiry:  config.AccessTokenExpiry,
		refreshTokenExpiry: config.RefreshTokenExpiry,
	}
}

// GenerateAccessToken generates an access token
func (m *JWTManager) GenerateAccessToken(user *models.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// GenerateRefreshToken generates a refresh token
func (m *JWTManager) GenerateRefreshToken(user *models.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ValidateToken validates a JWT token and returns the claims
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// GetAccessTokenExpiry returns the access token expiry duration
func (m *JWTManager) GetAccessTokenExpiry() time.Duration {
	return m.accessTokenExpiry
}

// GetRefreshTokenExpiry returns the refresh token expiry duration
func (m *JWTManager) GetRefreshTokenExpiry() time.Duration {
	return m.refreshTokenExpiry
}
