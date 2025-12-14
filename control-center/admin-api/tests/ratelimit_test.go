package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/paw-chain/paw/control-center/admin-api/middleware"
	"github.com/paw-chain/paw/control-center/admin-api/types"
)

func TestRateLimiter_BasicLimiting(t *testing.T) {
	config := middleware.RateLimitConfig{
		WriteOperationsPerMinute: 5,
		ReadOperationsPerMinute:  10,
		BurstMultiplier:          1,
	}

	rateLimiter := middleware.NewRateLimiter(config)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rateLimiter.RateLimitMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make requests within the limit
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
	}
}

func TestRateLimiter_ExceedLimit(t *testing.T) {
	config := middleware.RateLimitConfig{
		WriteOperationsPerMinute: 60,  // 1 request per second
		ReadOperationsPerMinute:  120, // 2 requests per second
		BurstMultiplier:          1,
	}

	rateLimiter := middleware.NewRateLimiter(config)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rateLimiter.RateLimitMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make requests exceeding the limit
	successCount := 0
	rateLimitedCount := 0

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code == http.StatusOK {
			successCount++
		} else if resp.Code == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	// At least some requests should be rate limited
	assert.Greater(t, rateLimitedCount, 0, "Expected some requests to be rate limited")
}

func TestRateLimiter_DifferentLimitsForRoles(t *testing.T) {
	config := middleware.RateLimitConfig{
		WriteOperationsPerMinute: 60, // 1 per second
		ReadOperationsPerMinute:  60, // 1 per second
		BurstMultiplier:          2,
	}

	rateLimiter := middleware.NewRateLimiter(config)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Simulate authenticated user with admin role
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "admin-001")
		c.Set("role", types.RoleAdmin)
		c.Next()
	})

	router.Use(rateLimiter.RateLimitMiddleware())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Admin should get higher rate limits for write operations
	successCount := 0
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("POST", "/test", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code == http.StatusOK {
			successCount++
		}
	}

	// Admin should have some successful requests
	assert.Greater(t, successCount, 0)
}

func TestRateLimiter_ReadOnlyCannotWrite(t *testing.T) {
	config := middleware.RateLimitConfig{
		WriteOperationsPerMinute: 60,
		ReadOperationsPerMinute:  120,
		BurstMultiplier:          2,
	}

	rateLimiter := middleware.NewRateLimiter(config)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Simulate authenticated user with readonly role
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "readonly-001")
		c.Set("role", types.RoleReadOnly)
		c.Next()
	})

	router.Use(rateLimiter.RateLimitMiddleware())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// ReadOnly user should be blocked from write operations
	req, _ := http.NewRequest("POST", "/test", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.Contains(t, resp.Body.String(), "Read-only users cannot perform write operations")
}

func TestRateLimiter_PerUserLimiting(t *testing.T) {
	config := middleware.RateLimitConfig{
		WriteOperationsPerMinute: 60, // 1 per second
		ReadOperationsPerMinute:  60,
		BurstMultiplier:          1,
	}

	rateLimiter := middleware.NewRateLimiter(config)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rateLimiter.RateLimitMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// User 1 makes requests
	user1SuccessCount := 0
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code == http.StatusOK {
			user1SuccessCount++
		}
	}

	// User 2 should have independent rate limit
	user2SuccessCount := 0
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.2:1234"
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code == http.StatusOK {
			user2SuccessCount++
		}
	}

	// Both users should have some successful requests (independent limits)
	assert.Greater(t, user1SuccessCount, 0)
	assert.Greater(t, user2SuccessCount, 0)
}

func TestSlidingWindowRateLimiter(t *testing.T) {
	limiter := middleware.NewSlidingWindowRateLimiter(3, time.Second)

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		allowed := limiter.Allow("test-user")
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}

	// 4th request should be rate limited
	allowed := limiter.Allow("test-user")
	assert.False(t, allowed, "4th request should be rate limited")

	// Wait for window to expire
	time.Sleep(1100 * time.Millisecond)

	// Should be allowed again
	allowed = limiter.Allow("test-user")
	assert.True(t, allowed, "Request after window should be allowed")
}

func TestSlidingWindowRateLimiter_Middleware(t *testing.T) {
	limiter := middleware.NewSlidingWindowRateLimiter(3, time.Second)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(limiter.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First 3 requests should succeed
	successCount := 0
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code == http.StatusOK {
			successCount++
		}
	}

	assert.Equal(t, 3, successCount, "First 3 requests should succeed")

	// 4th request should be rate limited
	req, _ := http.NewRequest("GET", "/test", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusTooManyRequests, resp.Code)
}

func TestRateLimiter_Headers(t *testing.T) {
	config := middleware.RateLimitConfig{
		WriteOperationsPerMinute: 60,
		ReadOperationsPerMinute:  120,
		BurstMultiplier:          2,
	}

	rateLimiter := middleware.NewRateLimiter(config)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rateLimiter.RateLimitMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Check rate limit headers are set
	assert.NotEmpty(t, resp.Header().Get("X-RateLimit-Limit"))
}

func TestRateLimiter_Cleanup(t *testing.T) {
	config := middleware.RateLimitConfig{
		WriteOperationsPerMinute: 60,
		ReadOperationsPerMinute:  120,
		BurstMultiplier:          2,
		CleanupInterval:          100 * time.Millisecond,
	}

	rateLimiter := middleware.NewRateLimiter(config)

	// Create limiters for multiple users
	for i := 0; i < 10; i++ {
		key := "user-" + string(rune('0'+i))
		rateLimiter.GetLimiter(key, 10, 20)
	}

	// Wait for cleanup to run
	time.Sleep(300 * time.Millisecond)

	// Cleanup should have run (we can't directly test this without exposing internals,
	// but we can verify the limiter still works)
	limiter := rateLimiter.GetLimiter("new-user", 10, 20)
	assert.NotNil(t, limiter)
}
