package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter manages API rate limiting
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	cache    CacheInterface
}

// CacheInterface defines the cache operations needed for rate limiting
type CacheInterface interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Increment(ctx context.Context, key string) (int64, error)
	SetWithExpiry(ctx context.Context, key string, value []byte, expiry time.Duration) error
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute  int
	Burst              int
	EnableIPLimit      bool
	EnableAPIKeyLimit  bool
	EnableGlobalLimit  bool
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rps int, burst int, cache CacheInterface) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(rps),
		burst:    burst,
		cache:    cache,
	}
}

// GetLimiter returns a rate limiter for the given key
func (rl *RateLimiter) GetLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[key] = limiter
	}

	return limiter
}

// CleanupStale removes stale limiters
func (rl *RateLimiter) CleanupStale() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// In production, implement proper cleanup logic
	// For now, clear all if too many
	if len(rl.limiters) > 10000 {
		rl.limiters = make(map[string]*rate.Limiter)
	}
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client identifier (IP address)
		clientIP := c.ClientIP()

		// Get API key if present
		apiKey := c.GetHeader("X-API-Key")

		// Use API key as identifier if present, otherwise use IP
		identifier := clientIP
		if apiKey != "" {
			identifier = "apikey:" + apiKey
		}

		// Get rate limiter for this client
		clientLimiter := limiter.GetLimiter(identifier)

		if !clientLimiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate_limit_exceeded",
				"message": "Too many requests. Please try again later.",
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// DistributedRateLimiter implements Redis-backed rate limiting
type DistributedRateLimiter struct {
	cache  CacheInterface
	limit  int64
	window time.Duration
}

// NewDistributedRateLimiter creates a distributed rate limiter
func NewDistributedRateLimiter(cache CacheInterface, requestsPerMinute int) *DistributedRateLimiter {
	return &DistributedRateLimiter{
		cache:  cache,
		limit:  int64(requestsPerMinute),
		window: time.Minute,
	}
}

// Allow checks if a request should be allowed
func (drl *DistributedRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	now := time.Now()
	windowKey := fmt.Sprintf("ratelimit:%s:%d", key, now.Unix()/60)

	// Increment counter
	count, err := drl.cache.Increment(ctx, windowKey)
	if err != nil {
		// On error, allow the request (fail open)
		return true, err
	}

	// Set expiry on first increment
	if count == 1 {
		drl.cache.SetWithExpiry(ctx, windowKey, []byte("1"), drl.window*2)
	}

	return count <= drl.limit, nil
}

// DistributedRateLimitMiddleware creates a distributed rate limiting middleware
func DistributedRateLimitMiddleware(limiter *DistributedRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		apiKey := c.GetHeader("X-API-Key")

		identifier := clientIP
		if apiKey != "" {
			identifier = "apikey:" + apiKey
		}

		allowed, err := limiter.Allow(c.Request.Context(), identifier)
		if err != nil {
			// Log error but allow request
			c.Next()
			return
		}

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate_limit_exceeded",
				"message": "Too many requests. Please try again later.",
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// TokenBucketLimiter implements token bucket algorithm
type TokenBucketLimiter struct {
	capacity  int64
	tokens    map[string]*tokenBucket
	mu        sync.RWMutex
	refillRate time.Duration
}

type tokenBucket struct {
	tokens     int64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucketLimiter creates a new token bucket rate limiter
func NewTokenBucketLimiter(capacity int, refillRate time.Duration) *TokenBucketLimiter {
	limiter := &TokenBucketLimiter{
		capacity:   int64(capacity),
		tokens:     make(map[string]*tokenBucket),
		refillRate: refillRate,
	}

	// Start background cleanup
	go limiter.cleanup()

	return limiter
}

// Allow checks if a request is allowed
func (tbl *TokenBucketLimiter) Allow(key string) bool {
	tbl.mu.RLock()
	bucket, exists := tbl.tokens[key]
	tbl.mu.RUnlock()

	if !exists {
		tbl.mu.Lock()
		bucket = &tokenBucket{
			tokens:     tbl.capacity,
			lastRefill: time.Now(),
		}
		tbl.tokens[key] = bucket
		tbl.mu.Unlock()
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int64(elapsed / tbl.refillRate)

	if tokensToAdd > 0 {
		bucket.tokens += tokensToAdd
		if bucket.tokens > tbl.capacity {
			bucket.tokens = tbl.capacity
		}
		bucket.lastRefill = now
	}

	// Check if we have tokens available
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// cleanup removes stale buckets
func (tbl *TokenBucketLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for range ticker.C {
		tbl.mu.Lock()
		now := time.Now()
		for key, bucket := range tbl.tokens {
			bucket.mu.Lock()
			if now.Sub(bucket.lastRefill) > time.Hour {
				delete(tbl.tokens, key)
			}
			bucket.mu.Unlock()
		}
		tbl.mu.Unlock()
	}
}

// SlidingWindowLimiter implements sliding window rate limiting
type SlidingWindowLimiter struct {
	cache      CacheInterface
	limit      int64
	windowSize time.Duration
}

// NewSlidingWindowLimiter creates a sliding window rate limiter
func NewSlidingWindowLimiter(cache CacheInterface, limit int, windowSize time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		cache:      cache,
		limit:      int64(limit),
		windowSize: windowSize,
	}
}

// Allow checks if a request is allowed using sliding window algorithm
func (swl *SlidingWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	now := time.Now()
	currentWindow := now.Unix() / int64(swl.windowSize.Seconds())
	previousWindow := currentWindow - 1

	currentKey := fmt.Sprintf("ratelimit:sw:%s:%d", key, currentWindow)
	previousKey := fmt.Sprintf("ratelimit:sw:%s:%d", key, previousWindow)

	// Get counts for current and previous windows
	currentCount, err := swl.cache.Increment(ctx, currentKey)
	if err != nil {
		return true, err // Fail open
	}

	// Set TTL on first increment
	if currentCount == 1 {
		swl.cache.SetWithExpiry(ctx, currentKey, []byte("1"), swl.windowSize*2)
	}

	// Get previous window count
	var previousCount int64
	previousData, err := swl.cache.Get(ctx, previousKey)
	if err == nil {
		fmt.Sscanf(string(previousData), "%d", &previousCount)
	}

	// Calculate weighted count
	windowProgress := float64(now.Unix()%int64(swl.windowSize.Seconds())) / float64(swl.windowSize.Seconds())
	weightedCount := int64(float64(previousCount)*(1-windowProgress)) + currentCount

	return weightedCount <= swl.limit, nil
}

// AdaptiveRateLimiter adjusts rate limits based on system load
type AdaptiveRateLimiter struct {
	baseLimiter *TokenBucketLimiter
	load        float64
	mu          sync.RWMutex
}

// NewAdaptiveRateLimiter creates an adaptive rate limiter
func NewAdaptiveRateLimiter(baseCapacity int, refillRate time.Duration) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		baseLimiter: NewTokenBucketLimiter(baseCapacity, refillRate),
		load:        0.5, // Start at 50% load
	}
}

// SetLoad sets the current system load (0.0 to 1.0)
func (arl *AdaptiveRateLimiter) SetLoad(load float64) {
	arl.mu.Lock()
	defer arl.mu.Unlock()
	arl.load = load
}

// Allow checks if request is allowed based on current load
func (arl *AdaptiveRateLimiter) Allow(key string) bool {
	arl.mu.RLock()
	load := arl.load
	arl.mu.RUnlock()

	// Under high load, be more strict
	if load > 0.8 {
		// Reduce effective rate by 50%
		if !arl.baseLimiter.Allow(key) {
			return false
		}
		// Require two tokens for high load
		return arl.baseLimiter.Allow(key)
	}

	return arl.baseLimiter.Allow(key)
}
