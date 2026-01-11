package middleware

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mikedewar/stablerisk/internal/security"
	"go.uber.org/zap"
)

// AuditMiddleware handles audit logging for compliance
type AuditMiddleware struct {
	auditLogger *security.AuditLogger
	logger      *zap.Logger
}

// NewAuditMiddleware creates a new audit middleware
func NewAuditMiddleware(auditLogger *security.AuditLogger, logger *zap.Logger) *AuditMiddleware {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &AuditMiddleware{
		auditLogger: auditLogger,
		logger:      logger,
	}
}

// Log creates an audit log for each request
func (m *AuditMiddleware) Log() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record start time
		start := time.Now()

		// Capture request body if present (for write operations)
		var requestBody string
		if c.Request.Body != nil && shouldLogBody(c.Request.Method) {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				requestBody = string(bodyBytes)
				// Restore the body for downstream handlers
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// Create a custom response writer to capture status code
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Process request
		c.Next()

		// Record end time
		duration := time.Since(start)

		// Get user info from context (may be empty for unauthenticated requests)
		userID := GetUserID(c)
		if userID == "" {
			userID = "anonymous"
		}

		// Determine action
		action := fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)

		// Determine resource (path)
		resource := c.Request.URL.Path

		// Determine status
		status := fmt.Sprintf("%d", blw.Status())

		// Get client IP
		ipAddress := c.ClientIP()

		// Build details
		details := map[string]interface{}{
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"query":       c.Request.URL.RawQuery,
			"user_agent":  c.Request.UserAgent(),
			"duration_ms": duration.Milliseconds(),
			"status_code": blw.Status(),
		}

		// Add request body for write operations (excluding sensitive endpoints)
		if requestBody != "" && !isSensitiveEndpoint(c.Request.URL.Path) {
			details["request_body"] = requestBody
		}

		// Add error if request failed
		if len(c.Errors) > 0 {
			details["errors"] = c.Errors.String()
		}

		// Add query parameters if present
		if len(c.Request.URL.Query()) > 0 {
			details["query_params"] = c.Request.URL.Query()
		}

		// Log to audit system
		m.auditLogger.Log(userID, action, resource, status, ipAddress, details)

		// Also log to structured logger for immediate visibility
		m.logger.Info("API request",
			zap.String("user_id", userID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", blw.Status()),
			zap.Duration("duration", duration),
			zap.String("ip", ipAddress))
	}
}

// bodyLogWriter is a custom response writer that captures the status code
type bodyLogWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

// Write captures the response body
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

// WriteString captures the response body
func (w *bodyLogWriter) WriteString(s string) (int, error) {
	return w.ResponseWriter.WriteString(s)
}

// Status returns the status code
func (w *bodyLogWriter) Status() int {
	if w.status == 0 {
		return w.ResponseWriter.Status()
	}
	return w.status
}

// WriteHeader captures the status code
func (w *bodyLogWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// shouldLogBody determines if we should log request body for this method
func shouldLogBody(method string) bool {
	return method == "POST" || method == "PUT" || method == "PATCH"
}

// isSensitiveEndpoint checks if an endpoint contains sensitive data that shouldn't be logged
func isSensitiveEndpoint(path string) bool {
	sensitiveEndpoints := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/refresh",
		"/api/v1/users/password",
	}

	for _, endpoint := range sensitiveEndpoints {
		if path == endpoint {
			return true
		}
	}

	return false
}
