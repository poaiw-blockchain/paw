package middleware

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/paw-chain/paw/control-center/admin-api/types"
)

// RateLimiter manages rate limiting for API endpoints
type RateLimiter struct {
	limiters map[string]*userLimiter
	mu       sync.RWMutex
	config   RateLimitConfig
}

type userLimiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	WriteOperationsPerMinute int
	ReadOperationsPerMinute  int
	BurstMultiplier          int
	CleanupInterval          time.Duration
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	if config.WriteOperationsPerMinute == 0 {
		config.WriteOperationsPerMinute = 10
	}
	if config.ReadOperationsPerMinute == 0 {
		config.ReadOperationsPerMinute = 100
	}
	if config.BurstMultiplier == 0 {
		config.BurstMultiplier = 2
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 10 * time.Minute
	}

	rl := &RateLimiter{
		limiters: make(map[string]*userLimiter),
		config:   config,
	}

	// Start background cleanup goroutine
	go rl.cleanup()

	return rl
}

// GetLimiter returns a rate limiter for the given user/IP combination
func (rl *RateLimiter) GetLimiter(key string, limit rate.Limit, burst int) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = &userLimiter{
			limiter:    rate.NewLimiter(limit, burst),
			lastAccess: time.Now(),
		}
		rl.limiters[key] = limiter
	} else {
		limiter.lastAccess = time.Now()
	}

	return limiter.limiter
}

func burstForPerMinute(perMinute int, multiplier int) int {
	if multiplier <= 0 {
		multiplier = 1
	}
	if perMinute <= 0 {
		return 1
	}

	base := int(math.Ceil(float64(perMinute) / 60.0))
	if base < 1 {
		base = 1
	}
	return base * multiplier
}

// cleanup removes stale limiters periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, limiter := range rl.limiters {
			if now.Sub(limiter.lastAccess) > rl.config.CleanupInterval*2 {
				delete(rl.limiters, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware creates a rate limiting middleware
func (rl *RateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Determine identifier (user ID or IP address)
		var identifier string
		if userID, exists := c.Get("user_id"); exists {
			identifier = fmt.Sprintf("user:%s", userID)
		} else {
			identifier = fmt.Sprintf("ip:%s", c.ClientIP())
		}

		// Determine rate limit based on operation type and role
		var perMinute int

		// Check if this is a write operation
		isWrite := c.Request.Method != "GET" && c.Request.Method != "HEAD" && c.Request.Method != "OPTIONS"

		if isWrite {
			// Write operations have stricter limits
			perMinute = rl.config.WriteOperationsPerMinute

			// Adjust based on role
			if roleValue, exists := c.Get("role"); exists {
				if role, ok := roleValue.(types.Role); ok {
					switch role {
					case types.RoleSuperUser:
						perMinute *= 2 // SuperUser gets 2x rate
					case types.RoleAdmin:
						perMinute = int(math.Round(float64(perMinute) * 1.5)) // Admin gets 1.5x rate
					case types.RoleOperator:
						// Use default rate
					case types.RoleReadOnly:
						// ReadOnly should not perform write operations
						c.JSON(http.StatusForbidden, gin.H{
							"error":   "forbidden",
							"message": "Read-only users cannot perform write operations",
						})
						c.Abort()
						return
					}
				}
			}
		} else {
			// Read operations have more relaxed limits
			perMinute = rl.config.ReadOperationsPerMinute
		}

		limit := rate.Limit(float64(perMinute) / 60.0)
		burst := burstForPerMinute(perMinute, rl.config.BurstMultiplier)

		// Get or create limiter for this identifier
		limiter := rl.GetLimiter(identifier, limit, burst)

		if !limiter.Allow() {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", perMinute))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))
			c.Header("Retry-After", "60")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"message":     "Too many requests. Please try again later.",
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", perMinute))

		c.Next()
	}
}

// PerEndpointRateLimiter creates a rate limiter that applies different limits per endpoint
type PerEndpointRateLimiter struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
	config   map[string]RateLimitConfig
}

// NewPerEndpointRateLimiter creates a new per-endpoint rate limiter
func NewPerEndpointRateLimiter(config map[string]RateLimitConfig) *PerEndpointRateLimiter {
	limiters := make(map[string]*RateLimiter)
	for endpoint, cfg := range config {
		limiters[endpoint] = NewRateLimiter(cfg)
	}

	return &PerEndpointRateLimiter{
		limiters: limiters,
		config:   config,
	}
}

// Middleware creates a rate limiting middleware for specific endpoints
func (perl *PerEndpointRateLimiter) Middleware(endpoint string) gin.HandlerFunc {
	perl.mu.RLock()
	limiter, exists := perl.limiters[endpoint]
	perl.mu.RUnlock()

	if !exists {
		// Use default config if endpoint not configured
		limiter = NewRateLimiter(RateLimitConfig{
			WriteOperationsPerMinute: 10,
			ReadOperationsPerMinute:  100,
		})
		perl.mu.Lock()
		perl.limiters[endpoint] = limiter
		perl.mu.Unlock()
	}

	return limiter.RateLimitMiddleware()
}

// SlidingWindowRateLimiter implements a sliding window rate limiter
type SlidingWindowRateLimiter struct {
	windows map[string]*slidingWindow
	mu      sync.RWMutex
	limit   int
	window  time.Duration
}

type slidingWindow struct {
	requests []time.Time
	mu       sync.Mutex
}

// NewSlidingWindowRateLimiter creates a new sliding window rate limiter
func NewSlidingWindowRateLimiter(requestsPerWindow int, windowDuration time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		windows: make(map[string]*slidingWindow),
		limit:   requestsPerWindow,
		window:  windowDuration,
	}
}

// Allow checks if a request should be allowed
func (swrl *SlidingWindowRateLimiter) Allow(key string) bool {
	swrl.mu.Lock()
	window, exists := swrl.windows[key]
	if !exists {
		window = &slidingWindow{
			requests: make([]time.Time, 0),
		}
		swrl.windows[key] = window
	}
	swrl.mu.Unlock()

	window.mu.Lock()
	defer window.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-swrl.window)

	// Remove old requests outside the window
	validRequests := make([]time.Time, 0)
	for _, req := range window.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	window.requests = validRequests

	// Check if limit exceeded
	if len(window.requests) >= swrl.limit {
		return false
	}

	// Add current request
	window.requests = append(window.requests, now)
	return true
}

// Middleware creates a middleware for sliding window rate limiting
func (swrl *SlidingWindowRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var identifier string
		if userID, exists := c.Get("user_id"); exists {
			identifier = fmt.Sprintf("user:%s", userID)
		} else {
			identifier = fmt.Sprintf("ip:%s", c.ClientIP())
		}

		if !swrl.Allow(identifier) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"message":     fmt.Sprintf("Maximum %d requests per %s exceeded", swrl.limit, swrl.window),
				"retry_after": int(swrl.window.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
