package models

import "time"

// User represents a system user
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	PasswordHash string    `json:"-"` // Never expose in JSON
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastLogin    time.Time `json:"last_login,omitempty"`
	IsActive     bool      `json:"is_active"`
}

// Role represents user roles
type Role string

const (
	RoleAdmin   Role = "admin"   // Full access
	RoleAnalyst Role = "analyst" // Read outliers, transactions, stats; trigger detection
	RoleViewer  Role = "viewer"  // Read-only access
)

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         *User  `json:"user"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
