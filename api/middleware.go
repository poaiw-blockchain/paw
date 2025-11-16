package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// AuthMiddleware validates JWT tokens
func (s *Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Authorization header required",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := s.authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Invalid or expired token",
				Details: err.Error(),
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("address", claims.Address)

		c.Next()
	}
}

// CORSMiddleware handles CORS
func (s *Server) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range s.config.CORSOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else if len(s.config.CORSOrigins) > 0 {
			c.Writer.Header().Set("Access-Control-Allow-Origin", s.config.CORSOrigins[0])
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware implements basic rate limiting (legacy)
// Deprecated: Use AdvancedRateLimitMiddleware for production
func RateLimitMiddleware(rps int) gin.HandlerFunc {
	// Create rate limiters per IP
	limiters := &sync.Map{}

	return func(c *gin.Context) {
		ip := c.ClientIP()

		// Get or create limiter for this IP
		limiterInterface, _ := limiters.LoadOrStore(ip, rate.NewLimiter(rate.Limit(rps), rps*2))
		limiter := limiterInterface.(*rate.Limiter)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, ErrorResponse{
				Error: "Rate limit exceeded",
				Code:  "RATE_LIMIT",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdvancedRateLimitMiddleware implements advanced rate limiting with all features
func AdvancedRateLimitMiddleware(limiter *AdvancedRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil {
			c.Next()
			return
		}

		// Check rate limit
		allowed, headers, err := limiter.CheckLimit(c)

		// Set rate limit headers
		if headers != nil {
			for k, v := range headers.ToHeaders() {
				c.Header(k, v)
			}
		}

		// Handle rate limit exceeded
		if !allowed || err != nil {
			statusCode := http.StatusTooManyRequests
			errorMsg := "Rate limit exceeded. Please try again later."

			// Get custom message for endpoint if available
			endpointLimit := limiter.config.GetEndpointLimit(c.Request.Method, c.Request.URL.Path)
			if endpointLimit != nil && endpointLimit.CustomMessage != "" {
				errorMsg = endpointLimit.CustomMessage
			}

			if err != nil {
				errorMsg = err.Error()
			}

			c.JSON(statusCode, ErrorResponse{
				Error: errorMsg,
				Code:  "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		// Track request completion for adaptive limiting
		userID := c.GetString("user_id")
		if userID != "" {
			defer limiter.DecrementConcurrent(userID)
		}

		// Process request
		c.Next()

		// Record success/failure for adaptive rate limiting
		if userID != "" {
			statusCode := c.Writer.Status()
			if statusCode >= 200 && statusCode < 400 {
				limiter.RecordSuccess(userID)
			} else if statusCode >= 400 {
				limiter.RecordFailure(userID)
			}
		}
	}
}

// LoggerMiddleware logs HTTP requests
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log after request
		end := time.Now()
		latency := end.Sub(start)

		statusCode := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()

		if query != "" {
			path = path + "?" + query
		}

		// Color code status
		var statusColor string
		switch {
		case statusCode >= 200 && statusCode < 300:
			statusColor = "\033[32m" // Green
		case statusCode >= 300 && statusCode < 400:
			statusColor = "\033[36m" // Cyan
		case statusCode >= 400 && statusCode < 500:
			statusColor = "\033[33m" // Yellow
		default:
			statusColor = "\033[31m" // Red
		}
		resetColor := "\033[0m"

		fmt.Printf("%s[%d]%s %s | %13v | %15s | %-7s %s\n",
			statusColor, statusCode, resetColor,
			end.Format("2006/01/02 15:04:05"),
			latency,
			clientIP,
			method,
			path,
		)
	}
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("Panic recovered: %v\n", err)
				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Error: "Internal server error",
					Code:  "INTERNAL_ERROR",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateOrderID() // Reuse existing function
		}
		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'")
		c.Writer.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
		c.Writer.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Next()
	}
}

// RequestSizeLimitMiddleware limits request body size
func RequestSizeLimitMiddleware(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{
				Error: "Request body too large",
				Code:  "REQUEST_TOO_LARGE",
			})
			c.Abort()
			return
		}

		// Also set a reader limit to prevent memory exhaustion
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	}
}

// HTTPSRedirectMiddleware redirects HTTP to HTTPS in production
func HTTPSRedirectMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request is over HTTPS
		if c.Request.TLS == nil && c.Request.Header.Get("X-Forwarded-Proto") != "https" {
			// Allow localhost without HTTPS for development
			if c.Request.Host != "localhost" && !strings.HasPrefix(c.Request.Host, "127.0.0.1") {
				target := "https://" + c.Request.Host + c.Request.RequestURI
				c.Redirect(http.StatusMovedPermanently, target)
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// APIKeyMiddleware validates API keys for sensitive operations
func (s *Server) APIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")

		// For high-value operations, require API key
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "API key required for this operation",
				Code:  "API_KEY_REQUIRED",
			})
			c.Abort()
			return
		}

		// Validate API key (in production, check against database)
		if !s.validateAPIKey(apiKey) {
			if s.auditLogger != nil {
				s.auditLogger.LogSecurityEvent(
					"invalid_api_key",
					"warning",
					"api_key_validation",
					"failure",
					fmt.Sprintf("Invalid API key attempt from IP: %s", c.ClientIP()),
					map[string]interface{}{
						"ip": c.ClientIP(),
					},
				)
			}

			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Invalid API key",
				Code:  "INVALID_API_KEY",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// validateAPIKey validates an API key (placeholder implementation)
func (s *Server) validateAPIKey(apiKey string) bool {
	// In production:
	// 1. Hash the API key
	// 2. Check against database
	// 3. Verify not expired
	// 4. Check rate limits specific to this key

	// For now, accept any non-empty key in development
	// In production, this MUST be replaced with proper validation
	return len(apiKey) >= 32
}

// TimeoutMiddleware sets a timeout for requests
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a context with timeout
		ctx := c.Copy().Request.Context()
		cancel := func() {}
		if timeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, timeout)
		}
		defer cancel()

		// Replace request context
		c.Request = c.Request.WithContext(ctx)

		// Process request
		finished := make(chan struct{})
		go func() {
			c.Next()
			finished <- struct{}{}
		}()

		select {
		case <-finished:
			// Request completed normally
		case <-ctx.Done():
			// Request timed out
			c.JSON(http.StatusRequestTimeout, ErrorResponse{
				Error: "Request timeout",
				Code:  "TIMEOUT",
			})
			c.Abort()
		}
	}
}

// CompressionMiddleware adds gzip compression (if needed)
// Note: Gin has built-in gzip middleware that can be used instead
func CompressionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if client accepts gzip
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// For production, use gin-contrib/gzip middleware
		c.Next()
	}
}
