package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	assert.NotNil(t, config)
	assert.True(t, config.Enabled)
	assert.Greater(t, config.DefaultRPS, 0)
	assert.Greater(t, config.DefaultBurst, 0)
	assert.NotNil(t, config.EndpointLimits)
	assert.NotNil(t, config.AccountTiers)
	assert.NotNil(t, config.AdaptiveConfig)
	assert.NotNil(t, config.IPConfig)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *RateLimitConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultRateLimitConfig(),
			wantErr: false,
		},
		{
			name: "invalid default rps",
			config: &RateLimitConfig{
				DefaultRPS:   -1,
				DefaultBurst: 100,
			},
			wantErr: true,
		},
		{
			name: "invalid default burst",
			config: &RateLimitConfig{
				DefaultRPS:   100,
				DefaultBurst: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid endpoint rps",
			config: &RateLimitConfig{
				DefaultRPS:   100,
				DefaultBurst: 100,
				EndpointLimits: map[string]*EndpointLimit{
					"/test": {
						Path:    "/test",
						RPS:     -1,
						Burst:   10,
						Enabled: true,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewAdvancedRateLimiter(t *testing.T) {
	config := DefaultRateLimitConfig()
	limiter, err := NewAdvancedRateLimiter(config, nil)

	require.NoError(t, err)
	require.NotNil(t, limiter)
	assert.NotNil(t, limiter.config)
	assert.NotNil(t, limiter.ipLimiters)
	assert.NotNil(t, limiter.accountLimiters)
	assert.NotNil(t, limiter.endpointLimiters)

	// Cleanup
	limiter.Close()
}

func TestIPBasedRateLimiting(t *testing.T) {
	config := DefaultRateLimitConfig()
	config.IPConfig.DefaultRPS = 2
	config.IPConfig.DefaultBurst = 2

	limiter, err := NewAdvancedRateLimiter(config, nil)
	require.NoError(t, err)
	defer limiter.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		allowed, headers, err := limiter.CheckLimit(c)
		if !allowed || err != nil {
			if headers != nil {
				for k, v := range headers.ToHeaders() {
					c.Header(k, v)
				}
			}
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}

	// Third request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestEndpointSpecificRateLimiting(t *testing.T) {
	config := DefaultRateLimitConfig()
	config.EndpointLimits["/api/test"] = &EndpointLimit{
		Path:    "/api/test",
		Method:  "GET",
		RPS:     1,
		Burst:   1,
		Enabled: true,
	}

	limiter, err := NewAdvancedRateLimiter(config, nil)
	require.NoError(t, err)
	defer limiter.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		allowed, headers, err := limiter.CheckLimit(c)
		if !allowed || err != nil {
			if headers != nil {
				for k, v := range headers.ToHeaders() {
					c.Header(k, v)
				}
			}
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	})

	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// First request should succeed
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second request should be rate limited
	req = httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestAccountBasedRateLimiting(t *testing.T) {
	config := DefaultRateLimitConfig()
	config.AccountTiers["test"] = &TierLimit{
		Name:              "test",
		RequestsPerMinute: 5,
		BurstSize:         5,
		ConcurrentReqs:    2,
	}

	limiter, err := NewAdvancedRateLimiter(config, nil)
	require.NoError(t, err)
	defer limiter.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		// Simulate authenticated user
		c.Set("user_id", "user123")
		c.Set("tier", "test")

		allowed, headers, err := limiter.CheckLimit(c)
		if !allowed || err != nil {
			if headers != nil {
				for k, v := range headers.ToHeaders() {
					c.Header(k, v)
				}
			}
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}

		// Decrement on completion
		defer limiter.DecrementConcurrent("user123")
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		// Simulate some processing time to test concurrent limit
		time.Sleep(50 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test concurrent request limit
	var wg sync.WaitGroup
	successCount := 0
	rateLimitCount := 0
	var mu sync.Mutex

	// Start 5 requests concurrently (limit is 2)
	startChan := make(chan struct{})
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startChan // Wait for signal to start
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.3:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			mu.Lock()
			if w.Code == http.StatusOK {
				successCount++
			} else if w.Code == http.StatusTooManyRequests {
				rateLimitCount++
			}
			mu.Unlock()
		}()
	}

	// Start all requests at the same time
	close(startChan)
	wg.Wait()

	// At least some requests should be rate limited
	assert.Greater(t, rateLimitCount, 0, "Should rate limit some requests")
	assert.LessOrEqual(t, successCount, 5, "Should not exceed total requests")
}

func TestIPWhitelisting(t *testing.T) {
	config := DefaultRateLimitConfig()
	config.IPConfig.WhitelistIPs = []string{"192.168.1.100"}
	config.IPConfig.DefaultRPS = 1
	config.IPConfig.DefaultBurst = 1

	limiter, err := NewAdvancedRateLimiter(config, nil)
	require.NoError(t, err)
	defer limiter.Close()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.RemoteAddr = "192.168.1.100:12345"

	// Whitelisted IP should always be allowed
	for i := 0; i < 10; i++ {
		allowed, _, err := limiter.CheckLimit(c)
		assert.True(t, allowed)
		assert.NoError(t, err)
	}
}

func TestIPBlacklisting(t *testing.T) {
	config := DefaultRateLimitConfig()
	config.IPConfig.BlacklistIPs = []string{"10.0.0.1"}

	limiter, err := NewAdvancedRateLimiter(config, nil)
	require.NoError(t, err)
	defer limiter.Close()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.RemoteAddr = "10.0.0.1:12345"

	// Blacklisted IP should always be blocked
	allowed, _, err := limiter.CheckLimit(c)
	assert.False(t, allowed)
	assert.Error(t, err)
}

func TestAdaptiveRateLimiting(t *testing.T) {
	config := DefaultRateLimitConfig()
	config.AdaptiveConfig.Enabled = true
	config.AdaptiveConfig.TrustThreshold = 5
	config.AdaptiveConfig.TrustMultiplier = 2.0

	limiter, err := NewAdvancedRateLimiter(config, nil)
	require.NoError(t, err)
	defer limiter.Close()

	userID := "user456"

	// Record successful requests
	for i := 0; i < 10; i++ {
		limiter.RecordSuccess(userID)
	}

	// Check multiplier increased
	multiplier := limiter.getAdaptiveMultiplier(userID)
	assert.Greater(t, multiplier, 1.0, "Multiplier should increase after successful requests")

	// Record failures
	for i := 0; i < 15; i++ {
		limiter.RecordFailure(userID)
	}

	// Check multiplier decreased
	multiplier = limiter.getAdaptiveMultiplier(userID)
	assert.Less(t, multiplier, 1.0, "Multiplier should decrease after failed requests")
}

func TestAutoIPBlocking(t *testing.T) {
	config := DefaultRateLimitConfig()
	config.IPConfig.DefaultRPS = 1
	config.IPConfig.DefaultBurst = 1
	config.IPConfig.AutoBlockThreshold = 3
	config.IPConfig.BlockDuration = 1 * time.Second

	limiter, err := NewAdvancedRateLimiter(config, nil)
	require.NoError(t, err)
	defer limiter.Close()

	gin.SetMode(gin.TestMode)

	// Exceed rate limit multiple times to trigger auto-block
	for i := 0; i < 5; i++ {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.RemoteAddr = "10.0.0.2:12345"

		limiter.CheckLimit(c)
		time.Sleep(50 * time.Millisecond)
	}

	// IP should now be blocked
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.RemoteAddr = "10.0.0.2:12345"

	allowed, _, _ := limiter.CheckLimit(c)
	assert.False(t, allowed, "IP should be auto-blocked")

	// Wait for block to expire
	time.Sleep(1200 * time.Millisecond)

	// Should be unblocked now
	allowed, _, _ = limiter.CheckLimit(c)
	assert.True(t, allowed, "IP should be unblocked after duration")
}

func TestRateLimitHeaders(t *testing.T) {
	headers := &RateLimitHeaders{
		Limit:      100,
		Remaining:  50,
		Reset:      time.Now().Unix() + 60,
		RetryAfter: 30,
	}

	headerMap := headers.ToHeaders()

	assert.Equal(t, "100", headerMap["X-RateLimit-Limit"])
	assert.Equal(t, "50", headerMap["X-RateLimit-Remaining"])
	assert.NotEmpty(t, headerMap["X-RateLimit-Reset"])
	assert.Equal(t, "30", headerMap["Retry-After"])
}

func TestConcurrentAccess(t *testing.T) {
	config := DefaultRateLimitConfig()
	limiter, err := NewAdvancedRateLimiter(config, nil)
	require.NoError(t, err)
	defer limiter.Close()

	gin.SetMode(gin.TestMode)

	var wg sync.WaitGroup
	iterations := 100

	// Test concurrent access from multiple IPs
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/test", nil)
			c.Request.RemoteAddr = "192.168.1." + string(rune(index%255)) + ":12345"

			// Should not panic
			limiter.CheckLimit(c)
		}(i)
	}

	wg.Wait()
}

func TestCleanupRoutine(t *testing.T) {
	config := DefaultRateLimitConfig()
	config.CleanupInterval = 100 * time.Millisecond

	limiter, err := NewAdvancedRateLimiter(config, nil)
	require.NoError(t, err)
	defer limiter.Close()

	gin.SetMode(gin.TestMode)

	// Create some limiters
	for i := 0; i < 10; i++ {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.RemoteAddr = "10.0.0." + string(rune(i)) + ":12345"
		limiter.CheckLimit(c)
	}

	initialStats := limiter.GetStats()
	initialCount := initialStats["ip_limiters"].(int)

	assert.Greater(t, initialCount, 0, "Should have IP limiters")

	// Wait for cleanup (requires items to be inactive for 10 minutes in real implementation)
	// For testing, we just verify the cleanup routine doesn't panic
	time.Sleep(200 * time.Millisecond)
}

func TestGetStats(t *testing.T) {
	config := DefaultRateLimitConfig()
	limiter, err := NewAdvancedRateLimiter(config, nil)
	require.NoError(t, err)
	defer limiter.Close()

	stats := limiter.GetStats()

	assert.NotNil(t, stats)
	assert.Contains(t, stats, "ip_limiters")
	assert.Contains(t, stats, "account_limiters")
	assert.Contains(t, stats, "behavior_trackers")
	assert.Contains(t, stats, "blacklisted_ips")
	assert.Contains(t, stats, "enabled")
	assert.True(t, stats["enabled"].(bool))
}

func TestGetEndpointLimit(t *testing.T) {
	config := DefaultRateLimitConfig()

	limit := config.GetEndpointLimit("POST", "/api/auth/login")
	assert.NotNil(t, limit)
	assert.Equal(t, "/api/auth/login", limit.Path)

	limit = config.GetEndpointLimit("GET", "/nonexistent")
	assert.Nil(t, limit)
}

func TestGetTierLimit(t *testing.T) {
	config := DefaultRateLimitConfig()

	limit := config.GetTierLimit("premium")
	assert.NotNil(t, limit)
	assert.Equal(t, "premium", limit.Name)

	// Non-existent tier should return free tier
	limit = config.GetTierLimit("nonexistent")
	assert.NotNil(t, limit)
	assert.Equal(t, "free", limit.Name)
}

// Benchmark tests

func BenchmarkIPRateLimiting(b *testing.B) {
	config := DefaultRateLimitConfig()
	limiter, _ := NewAdvancedRateLimiter(config, nil)
	defer limiter.Close()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.RemoteAddr = "192.168.1.1:12345"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.CheckLimit(c)
	}
}

func BenchmarkAccountRateLimiting(b *testing.B) {
	config := DefaultRateLimitConfig()
	limiter, _ := NewAdvancedRateLimiter(config, nil)
	defer limiter.Close()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.RemoteAddr = "192.168.1.1:12345"
	c.Set("user_id", "user123")
	c.Set("tier", "premium")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.CheckLimit(c)
		limiter.DecrementConcurrent("user123")
	}
}

func BenchmarkAdaptiveTracking(b *testing.B) {
	config := DefaultRateLimitConfig()
	limiter, _ := NewAdvancedRateLimiter(config, nil)
	defer limiter.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			limiter.RecordSuccess("user123")
		} else {
			limiter.RecordFailure("user123")
		}
	}
}
