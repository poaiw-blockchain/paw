package api

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// AdvancedRateLimiter implements sophisticated rate limiting with multiple strategies
type AdvancedRateLimiter struct {
	config      *RateLimitConfig
	auditLogger *AuditLogger

	// IP-based limiters
	ipLimiters     *sync.Map // map[string]*IPLimiter
	ipBlacklist    *sync.Map // map[string]time.Time (IP -> unblock time)
	ipWhitelist    map[string]bool
	whitelistCIDRs []*net.IPNet
	blacklistCIDRs []*net.IPNet

	// Account-based limiters
	accountLimiters *sync.Map // map[string]*AccountLimiter

	// Endpoint-based limiters
	endpointLimiters map[string]*rate.Limiter

	// Adaptive tracking
	userBehavior *sync.Map // map[string]*BehaviorTracker

	// Cleanup
	stopChan chan struct{}
	mu       sync.RWMutex
}

// IPLimiter tracks rate limits for an IP address
type IPLimiter struct {
	limiter    *rate.Limiter
	lastSeen   time.Time
	violations int
	blocked    bool
	blockUntil time.Time
	mu         sync.RWMutex
}

// AccountLimiter tracks rate limits for an authenticated account
type AccountLimiter struct {
	userID        string
	tier          string
	minuteLimiter *rate.Limiter
	hourLimiter   *rate.Limiter
	dayLimiter    *rate.Limiter
	concurrent    int
	maxConcurrent int
	lastSeen      time.Time
	mu            sync.RWMutex
}

// BehaviorTracker tracks user behavior for adaptive rate limiting
type BehaviorTracker struct {
	userID             string
	successCount       int
	failureCount       int
	trustLevel         int
	suspicionLevel     int
	lastActivity       time.Time
	consecutiveSuccess int
	consecutiveFailure int
	mu                 sync.RWMutex
}

// NewAdvancedRateLimiter creates a new advanced rate limiter
func NewAdvancedRateLimiter(config *RateLimitConfig, auditLogger *AuditLogger) (*AdvancedRateLimiter, error) {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid rate limit config: %w", err)
	}

	arl := &AdvancedRateLimiter{
		config:           config,
		auditLogger:      auditLogger,
		ipLimiters:       &sync.Map{},
		ipBlacklist:      &sync.Map{},
		ipWhitelist:      make(map[string]bool),
		accountLimiters:  &sync.Map{},
		endpointLimiters: make(map[string]*rate.Limiter),
		userBehavior:     &sync.Map{},
		stopChan:         make(chan struct{}),
	}

	// Parse CIDR ranges
	if config.IPConfig != nil {
		for _, cidr := range config.IPConfig.WhitelistCIDRs {
			_, ipNet, err := net.ParseCIDR(cidr)
			if err == nil {
				arl.whitelistCIDRs = append(arl.whitelistCIDRs, ipNet)
			}
		}
		for _, cidr := range config.IPConfig.BlacklistCIDRs {
			_, ipNet, err := net.ParseCIDR(cidr)
			if err == nil {
				arl.blacklistCIDRs = append(arl.blacklistCIDRs, ipNet)
			}
		}

		// Populate IP whitelist
		for _, ip := range config.IPConfig.WhitelistIPs {
			arl.ipWhitelist[ip] = true
		}

		// Populate IP blacklist
		for _, ip := range config.IPConfig.BlacklistIPs {
			arl.ipBlacklist.Store(ip, time.Now().Add(365*24*time.Hour)) // Block for a year
		}
	}

	// Initialize endpoint limiters
	for path, endpointLimit := range config.EndpointLimits {
		if endpointLimit.Enabled {
			key := path
			arl.endpointLimiters[key] = rate.NewLimiter(
				rate.Limit(endpointLimit.RPS),
				endpointLimit.Burst,
			)
		}
	}

	// Start cleanup goroutine
	go arl.cleanupRoutine()

	return arl, nil
}

// CheckLimit checks if a request should be allowed
func (arl *AdvancedRateLimiter) CheckLimit(c *gin.Context) (allowed bool, headers *RateLimitHeaders, err error) {
	if !arl.config.Enabled {
		return true, nil, nil
	}

	ip := c.ClientIP()
	path := c.Request.URL.Path
	method := c.Request.Method
	userID := c.GetString("user_id")

	// Check IP blacklist first
	if arl.isIPBlacklisted(ip) {
		if arl.auditLogger != nil {
			arl.auditLogger.LogSecurityEvent(
				"rate_limit_blacklist",
				"critical",
				"blocked_request",
				"blocked",
				fmt.Sprintf("IP %s is blacklisted", ip),
				map[string]interface{}{
					"ip":   ip,
					"path": path,
				},
			)
		}
		return false, nil, fmt.Errorf("IP address is blacklisted")
	}

	// Check IP whitelist
	if arl.isIPWhitelisted(ip) {
		return true, nil, nil
	}

	// Get endpoint-specific limit
	endpointLimit := arl.config.GetEndpointLimit(method, path)

	// Check endpoint-specific rate limit
	if endpointLimit != nil && endpointLimit.Enabled {
		allowed, headers = arl.checkEndpointLimit(endpointLimit, c)
		if !allowed {
			if arl.auditLogger != nil {
				arl.auditLogger.LogRateLimitExceeded(c, userID, fmt.Sprintf("endpoint:%s", path))
			}
			return false, headers, nil
		}
	}

	// Check IP-based rate limit (if not skipped by endpoint config)
	if endpointLimit == nil || !endpointLimit.SkipIPLimit {
		allowed, headers = arl.checkIPLimit(ip)
		if !allowed {
			if arl.auditLogger != nil {
				arl.auditLogger.LogRateLimitExceeded(c, userID, "ip")
			}
			return false, headers, nil
		}
	}

	// Check account-based rate limit (if authenticated)
	if userID != "" {
		tier := c.GetString("tier")
		if tier == "" {
			tier = "free"
		}
		allowed, headers = arl.checkAccountLimit(userID, tier)
		if !allowed {
			if arl.auditLogger != nil {
				arl.auditLogger.LogRateLimitExceeded(c, userID, fmt.Sprintf("account:%s", tier))
			}
			return false, headers, nil
		}
	}

	return true, headers, nil
}

// checkEndpointLimit checks the rate limit for a specific endpoint
func (arl *AdvancedRateLimiter) checkEndpointLimit(endpointLimit *EndpointLimit, c *gin.Context) (bool, *RateLimitHeaders) {
	key := endpointLimit.Path
	limiter, exists := arl.endpointLimiters[key]

	if !exists {
		return true, nil
	}

	// Get reservation
	reservation := limiter.Reserve()
	if !reservation.OK() {
		return false, &RateLimitHeaders{
			Limit:      endpointLimit.RPS,
			Remaining:  0,
			Reset:      time.Now().Add(time.Second).Unix(),
			RetryAfter: 1,
		}
	}

	delay := reservation.Delay()
	if delay > 0 {
		reservation.Cancel()
		retryAfter := int(delay.Seconds()) + 1
		return false, &RateLimitHeaders{
			Limit:      endpointLimit.RPS,
			Remaining:  0,
			Reset:      time.Now().Add(delay).Unix(),
			RetryAfter: retryAfter,
		}
	}

	return true, &RateLimitHeaders{
		Limit:     endpointLimit.RPS,
		Remaining: endpointLimit.Burst - 1,
		Reset:     time.Now().Add(time.Second).Unix(),
	}
}

// checkIPLimit checks the IP-based rate limit
func (arl *AdvancedRateLimiter) checkIPLimit(ip string) (bool, *RateLimitHeaders) {
	if !arl.config.IPConfig.Enabled {
		return true, nil
	}

	limiter := arl.getOrCreateIPLimiter(ip)

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	// Check if IP is blocked
	if limiter.blocked && time.Now().Before(limiter.blockUntil) {
		return false, &RateLimitHeaders{
			Limit:      0,
			Remaining:  0,
			Reset:      limiter.blockUntil.Unix(),
			RetryAfter: int(time.Until(limiter.blockUntil).Seconds()) + 1,
		}
	} else if limiter.blocked {
		// Unblock if time has passed
		limiter.blocked = false
		limiter.violations = 0
	}

	limiter.lastSeen = time.Now()

	// Check rate limit
	if !limiter.limiter.Allow() {
		limiter.violations++

		// Auto-block if threshold exceeded
		if arl.config.IPConfig.AutoBlockThreshold > 0 &&
			limiter.violations >= arl.config.IPConfig.AutoBlockThreshold {
			limiter.blocked = true
			limiter.blockUntil = time.Now().Add(arl.config.IPConfig.BlockDuration)
			arl.ipBlacklist.Store(ip, limiter.blockUntil)

			if arl.auditLogger != nil {
				arl.auditLogger.LogSecurityEvent(
					"ip_auto_blocked",
					"critical",
					"auto_block",
					"blocked",
					fmt.Sprintf("IP %s auto-blocked after %d violations", ip, limiter.violations),
					map[string]interface{}{
						"ip":         ip,
						"violations": limiter.violations,
					},
				)
			}
		}

		return false, &RateLimitHeaders{
			Limit:      arl.config.IPConfig.DefaultRPS,
			Remaining:  0,
			Reset:      time.Now().Add(time.Second).Unix(),
			RetryAfter: 1,
		}
	}

	return true, &RateLimitHeaders{
		Limit:     arl.config.IPConfig.DefaultRPS,
		Remaining: arl.config.IPConfig.DefaultBurst - 1,
		Reset:     time.Now().Add(time.Second).Unix(),
	}
}

// checkAccountLimit checks the account-based rate limit
func (arl *AdvancedRateLimiter) checkAccountLimit(userID, tier string) (bool, *RateLimitHeaders) {
	limiter := arl.getOrCreateAccountLimiter(userID, tier)

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	limiter.lastSeen = time.Now()

	// Check concurrent requests
	if limiter.concurrent >= limiter.maxConcurrent {
		return false, &RateLimitHeaders{
			Limit:      limiter.maxConcurrent,
			Remaining:  0,
			Reset:      time.Now().Add(time.Second).Unix(),
			RetryAfter: 1,
		}
	}

	// Apply adaptive multiplier
	multiplier := arl.getAdaptiveMultiplier(userID)

	// Check minute limit
	if !limiter.minuteLimiter.Allow() {
		return false, &RateLimitHeaders{
			Limit:      int(float64(limiter.minuteLimiter.Limit()) * multiplier),
			Remaining:  0,
			Reset:      time.Now().Add(time.Minute).Unix(),
			RetryAfter: 60,
		}
	}

	// Increment concurrent counter
	limiter.concurrent++

	tierLimit := arl.config.GetTierLimit(tier)
	return true, &RateLimitHeaders{
		Limit:     int(float64(tierLimit.RequestsPerMinute) * multiplier),
		Remaining: tierLimit.BurstSize - limiter.concurrent,
		Reset:     time.Now().Add(time.Minute).Unix(),
	}
}

// DecrementConcurrent decrements the concurrent request counter for an account
func (arl *AdvancedRateLimiter) DecrementConcurrent(userID string) {
	if userID == "" {
		return
	}

	limiterInterface, ok := arl.accountLimiters.Load(userID)
	if !ok {
		return
	}

	limiter := limiterInterface.(*AccountLimiter)
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	if limiter.concurrent > 0 {
		limiter.concurrent--
	}
}

// RecordSuccess records a successful request for adaptive rate limiting
func (arl *AdvancedRateLimiter) RecordSuccess(userID string) {
	if !arl.config.AdaptiveConfig.Enabled || userID == "" {
		return
	}

	tracker := arl.getOrCreateBehaviorTracker(userID)
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	tracker.successCount++
	tracker.consecutiveSuccess++
	tracker.consecutiveFailure = 0
	tracker.lastActivity = time.Now()

	// Increase trust level
	if tracker.consecutiveSuccess >= arl.config.AdaptiveConfig.TrustThreshold {
		if tracker.trustLevel < arl.config.AdaptiveConfig.MaxTrustLevel {
			tracker.trustLevel++
		}
		tracker.consecutiveSuccess = 0
	}

	// Decrease suspicion level
	if tracker.suspicionLevel > 0 {
		tracker.suspicionLevel--
	}
}

// RecordFailure records a failed request for adaptive rate limiting
func (arl *AdvancedRateLimiter) RecordFailure(userID string) {
	if !arl.config.AdaptiveConfig.Enabled || userID == "" {
		return
	}

	tracker := arl.getOrCreateBehaviorTracker(userID)
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	tracker.failureCount++
	tracker.consecutiveFailure++
	tracker.consecutiveSuccess = 0
	tracker.lastActivity = time.Now()

	// Increase suspicion level
	if tracker.consecutiveFailure >= arl.config.AdaptiveConfig.SuspicionThreshold {
		if tracker.suspicionLevel < arl.config.AdaptiveConfig.MaxSuspicionLevel {
			tracker.suspicionLevel++
		}
		tracker.consecutiveFailure = 0

		// Log suspicious activity
		if arl.auditLogger != nil && tracker.suspicionLevel >= 3 {
			arl.auditLogger.LogSuspiciousActivity(
				&gin.Context{},
				userID,
				"high_failure_rate",
				fmt.Sprintf("User has suspicion level %d", tracker.suspicionLevel),
			)
		}
	}

	// Decrease trust level
	if tracker.trustLevel > 0 {
		tracker.trustLevel--
	}
}

// getAdaptiveMultiplier returns the rate limit multiplier based on user behavior
func (arl *AdvancedRateLimiter) getAdaptiveMultiplier(userID string) float64 {
	if !arl.config.AdaptiveConfig.Enabled || userID == "" {
		return 1.0
	}

	trackerInterface, ok := arl.userBehavior.Load(userID)
	if !ok {
		return 1.0
	}

	tracker := trackerInterface.(*BehaviorTracker)
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	multiplier := 1.0

	// Apply trust multiplier
	if tracker.trustLevel > 0 {
		multiplier *= (1.0 + float64(tracker.trustLevel)*0.2)
		if multiplier > arl.config.AdaptiveConfig.TrustMultiplier {
			multiplier = arl.config.AdaptiveConfig.TrustMultiplier
		}
	}

	// Apply suspicion multiplier
	if tracker.suspicionLevel > 0 {
		multiplier *= (1.0 - float64(tracker.suspicionLevel)*0.1)
		if multiplier < arl.config.AdaptiveConfig.SuspicionMultiplier {
			multiplier = arl.config.AdaptiveConfig.SuspicionMultiplier
		}
	}

	return multiplier
}

// Helper functions

func (arl *AdvancedRateLimiter) getOrCreateIPLimiter(ip string) *IPLimiter {
	limiterInterface, _ := arl.ipLimiters.LoadOrStore(ip, &IPLimiter{
		limiter: rate.NewLimiter(
			rate.Limit(arl.config.IPConfig.DefaultRPS),
			arl.config.IPConfig.DefaultBurst,
		),
		lastSeen: time.Now(),
	})
	return limiterInterface.(*IPLimiter)
}

func (arl *AdvancedRateLimiter) getOrCreateAccountLimiter(userID, tier string) *AccountLimiter {
	limiterInterface, _ := arl.accountLimiters.LoadOrStore(userID, func() *AccountLimiter {
		tierLimit := arl.config.GetTierLimit(tier)
		return &AccountLimiter{
			userID:        userID,
			tier:          tier,
			minuteLimiter: rate.NewLimiter(rate.Limit(tierLimit.RequestsPerMinute)/60.0, tierLimit.BurstSize),
			hourLimiter:   rate.NewLimiter(rate.Limit(tierLimit.RequestsPerHour)/3600.0, tierLimit.BurstSize),
			dayLimiter:    rate.NewLimiter(rate.Limit(tierLimit.RequestsPerDay)/86400.0, tierLimit.BurstSize),
			maxConcurrent: tierLimit.ConcurrentReqs,
			lastSeen:      time.Now(),
		}
	}())
	return limiterInterface.(*AccountLimiter)
}

func (arl *AdvancedRateLimiter) getOrCreateBehaviorTracker(userID string) *BehaviorTracker {
	trackerInterface, _ := arl.userBehavior.LoadOrStore(userID, &BehaviorTracker{
		userID:       userID,
		lastActivity: time.Now(),
	})
	return trackerInterface.(*BehaviorTracker)
}

func (arl *AdvancedRateLimiter) isIPWhitelisted(ip string) bool {
	// Check exact match
	if arl.ipWhitelist[ip] {
		return true
	}

	// Check CIDR ranges
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, ipNet := range arl.whitelistCIDRs {
		if ipNet.Contains(parsedIP) {
			return true
		}
	}

	return false
}

func (arl *AdvancedRateLimiter) isIPBlacklisted(ip string) bool {
	// Check blacklist map
	unblockTimeInterface, ok := arl.ipBlacklist.Load(ip)
	if ok {
		unblockTime := unblockTimeInterface.(time.Time)
		if time.Now().Before(unblockTime) {
			return true
		}
		// Remove from blacklist if time has passed
		arl.ipBlacklist.Delete(ip)
	}

	// Check CIDR ranges
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, ipNet := range arl.blacklistCIDRs {
		if ipNet.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// cleanupRoutine periodically cleans up old limiters and trackers
func (arl *AdvancedRateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(arl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			arl.cleanup()
		case <-arl.stopChan:
			return
		}
	}
}

func (arl *AdvancedRateLimiter) cleanup() {
	now := time.Now()
	cleanupThreshold := 10 * time.Minute

	// Cleanup IP limiters
	arl.ipLimiters.Range(func(key, value interface{}) bool {
		limiter := value.(*IPLimiter)
		limiter.mu.RLock()
		lastSeen := limiter.lastSeen
		limiter.mu.RUnlock()

		if now.Sub(lastSeen) > cleanupThreshold {
			arl.ipLimiters.Delete(key)
		}
		return true
	})

	// Cleanup account limiters
	arl.accountLimiters.Range(func(key, value interface{}) bool {
		limiter := value.(*AccountLimiter)
		limiter.mu.RLock()
		lastSeen := limiter.lastSeen
		limiter.mu.RUnlock()

		if now.Sub(lastSeen) > cleanupThreshold {
			arl.accountLimiters.Delete(key)
		}
		return true
	})

	// Cleanup behavior trackers
	arl.userBehavior.Range(func(key, value interface{}) bool {
		tracker := value.(*BehaviorTracker)
		tracker.mu.RLock()
		lastActivity := tracker.lastActivity
		tracker.mu.RUnlock()

		if now.Sub(lastActivity) > cleanupThreshold {
			arl.userBehavior.Delete(key)
		}
		return true
	})

	// Cleanup IP blacklist
	arl.ipBlacklist.Range(func(key, value interface{}) bool {
		unblockTime := value.(time.Time)
		if now.After(unblockTime) {
			arl.ipBlacklist.Delete(key)
		}
		return true
	})
}

// Close stops the rate limiter
func (arl *AdvancedRateLimiter) Close() {
	close(arl.stopChan)
}

// GetStats returns statistics about the rate limiter
func (arl *AdvancedRateLimiter) GetStats() map[string]interface{} {
	ipCount := 0
	accountCount := 0
	behaviorCount := 0
	blacklistCount := 0

	arl.ipLimiters.Range(func(_, _ interface{}) bool {
		ipCount++
		return true
	})

	arl.accountLimiters.Range(func(_, _ interface{}) bool {
		accountCount++
		return true
	})

	arl.userBehavior.Range(func(_, _ interface{}) bool {
		behaviorCount++
		return true
	})

	arl.ipBlacklist.Range(func(_, _ interface{}) bool {
		blacklistCount++
		return true
	})

	return map[string]interface{}{
		"ip_limiters":       ipCount,
		"account_limiters":  accountCount,
		"behavior_trackers": behaviorCount,
		"blacklisted_ips":   blacklistCount,
		"enabled":           arl.config.Enabled,
	}
}
