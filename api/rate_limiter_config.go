package api

import (
	"fmt"
	"time"
)

// RateLimitConfig holds the complete rate limiting configuration
type RateLimitConfig struct {
	// Global settings
	Enabled         bool          `yaml:"enabled" json:"enabled"`
	DefaultRPS      int           `yaml:"default_rps" json:"default_rps"`
	DefaultBurst    int           `yaml:"default_burst" json:"default_burst"`
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`

	// Per-endpoint rate limits
	EndpointLimits map[string]*EndpointLimit `yaml:"endpoint_limits" json:"endpoint_limits"`

	// Account tier limits
	AccountTiers map[string]*TierLimit `yaml:"account_tiers" json:"account_tiers"`

	// Adaptive rate limiting settings
	AdaptiveConfig *AdaptiveConfig `yaml:"adaptive" json:"adaptive"`

	// IP-based settings
	IPConfig *IPConfig `yaml:"ip_config" json:"ip_config"`

	// Audit integration
	AuditEnabled bool `yaml:"audit_enabled" json:"audit_enabled"`
}

// EndpointLimit defines rate limits for a specific endpoint
type EndpointLimit struct {
	Path          string `yaml:"path" json:"path"`
	Method        string `yaml:"method" json:"method"` // GET, POST, etc. Empty means all methods
	RPS           int    `yaml:"rps" json:"rps"`
	Burst         int    `yaml:"burst" json:"burst"`
	Enabled       bool   `yaml:"enabled" json:"enabled"`
	RequireAuth   bool   `yaml:"require_auth" json:"require_auth"`
	SkipIPLimit   bool   `yaml:"skip_ip_limit" json:"skip_ip_limit"` // Don't apply IP-based limits
	CustomMessage string `yaml:"custom_message" json:"custom_message"`
}

// TierLimit defines rate limits for an account tier
type TierLimit struct {
	Name              string `yaml:"name" json:"name"`
	RequestsPerHour   int    `yaml:"requests_per_hour" json:"requests_per_hour"`
	RequestsPerDay    int    `yaml:"requests_per_day" json:"requests_per_day"`
	RequestsPerMinute int    `yaml:"requests_per_minute" json:"requests_per_minute"`
	BurstSize         int    `yaml:"burst_size" json:"burst_size"`
	ConcurrentReqs    int    `yaml:"concurrent_requests" json:"concurrent_requests"`
	Priority          int    `yaml:"priority" json:"priority"` // Higher priority = less restrictive
}

// AdaptiveConfig holds configuration for adaptive rate limiting
type AdaptiveConfig struct {
	Enabled              bool          `yaml:"enabled" json:"enabled"`
	TrustThreshold       int           `yaml:"trust_threshold" json:"trust_threshold"`           // Successful requests before trust increase
	SuspicionThreshold   int           `yaml:"suspicion_threshold" json:"suspicion_threshold"`   // Failed requests before rate decrease
	TrustMultiplier      float64       `yaml:"trust_multiplier" json:"trust_multiplier"`         // Multiplier for trusted users (e.g., 2.0 = 2x limit)
	SuspicionMultiplier  float64       `yaml:"suspicion_multiplier" json:"suspicion_multiplier"` // Multiplier for suspicious users (e.g., 0.5 = 50% limit)
	DecayInterval        time.Duration `yaml:"decay_interval" json:"decay_interval"`             // How often to decay trust/suspicion scores
	MaxTrustLevel        int           `yaml:"max_trust_level" json:"max_trust_level"`
	MaxSuspicionLevel    int           `yaml:"max_suspicion_level" json:"max_suspicion_level"`
	ResetAfterGoodPeriod time.Duration `yaml:"reset_after_good_period" json:"reset_after_good_period"`
}

// IPConfig holds IP-based rate limiting configuration
type IPConfig struct {
	Enabled            bool          `yaml:"enabled" json:"enabled"`
	DefaultRPS         int           `yaml:"default_rps" json:"default_rps"`
	DefaultBurst       int           `yaml:"default_burst" json:"default_burst"`
	WhitelistIPs       []string      `yaml:"whitelist_ips" json:"whitelist_ips"`
	BlacklistIPs       []string      `yaml:"blacklist_ips" json:"blacklist_ips"`
	WhitelistCIDRs     []string      `yaml:"whitelist_cidrs" json:"whitelist_cidrs"`
	BlacklistCIDRs     []string      `yaml:"blacklist_cidrs" json:"blacklist_cidrs"`
	BlockDuration      time.Duration `yaml:"block_duration" json:"block_duration"`
	AutoBlockThreshold int           `yaml:"auto_block_threshold" json:"auto_block_threshold"` // Auto-block after N violations
}

// DefaultRateLimitConfig returns a default rate limiting configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled:         true,
		DefaultRPS:      50,
		DefaultBurst:    100,
		CleanupInterval: 5 * time.Minute,
		EndpointLimits: map[string]*EndpointLimit{
			"/api/auth/login": {
				Path:          "/api/auth/login",
				Method:        "POST",
				RPS:           5,
				Burst:         10,
				Enabled:       true,
				CustomMessage: "Login rate limit exceeded. Please try again later.",
			},
			"/api/auth/register": {
				Path:          "/api/auth/register",
				Method:        "POST",
				RPS:           2,
				Burst:         5,
				Enabled:       true,
				CustomMessage: "Registration rate limit exceeded.",
			},
			"/api/orders/create": {
				Path:        "/api/orders/create",
				Method:      "POST",
				RPS:         10,
				Burst:       20,
				Enabled:     true,
				RequireAuth: true,
			},
			"/api/wallet/send": {
				Path:        "/api/wallet/send",
				Method:      "POST",
				RPS:         5,
				Burst:       10,
				Enabled:     true,
				RequireAuth: true,
			},
			"/api/market/stats": {
				Path:        "/api/market/stats",
				Method:      "GET",
				RPS:         100,
				Burst:       200,
				Enabled:     true,
				SkipIPLimit: false,
			},
			"/health": {
				Path:    "/health",
				Method:  "GET",
				RPS:     1000,
				Burst:   2000,
				Enabled: false, // No limit on health checks
			},
		},
		AccountTiers: map[string]*TierLimit{
			"free": {
				Name:              "free",
				RequestsPerHour:   1000,
				RequestsPerDay:    10000,
				RequestsPerMinute: 20,
				BurstSize:         50,
				ConcurrentReqs:    10,
				Priority:          1,
			},
			"premium": {
				Name:              "premium",
				RequestsPerHour:   10000,
				RequestsPerDay:    100000,
				RequestsPerMinute: 100,
				BurstSize:         200,
				ConcurrentReqs:    50,
				Priority:          5,
			},
			"enterprise": {
				Name:              "enterprise",
				RequestsPerHour:   100000,
				RequestsPerDay:    1000000,
				RequestsPerMinute: 1000,
				BurstSize:         2000,
				ConcurrentReqs:    200,
				Priority:          10,
			},
		},
		AdaptiveConfig: &AdaptiveConfig{
			Enabled:              true,
			TrustThreshold:       100,
			SuspicionThreshold:   10,
			TrustMultiplier:      2.0,
			SuspicionMultiplier:  0.5,
			DecayInterval:        1 * time.Hour,
			MaxTrustLevel:        5,
			MaxSuspicionLevel:    5,
			ResetAfterGoodPeriod: 24 * time.Hour,
		},
		IPConfig: &IPConfig{
			Enabled:            true,
			DefaultRPS:         100,
			DefaultBurst:       200,
			WhitelistIPs:       []string{},
			BlacklistIPs:       []string{},
			WhitelistCIDRs:     []string{"127.0.0.0/8", "::1/128"}, // Localhost
			BlacklistCIDRs:     []string{},
			BlockDuration:      1 * time.Hour,
			AutoBlockThreshold: 100,
		},
		AuditEnabled: true,
	}
}

// Validate validates the rate limit configuration
func (c *RateLimitConfig) Validate() error {
	if c.DefaultRPS <= 0 {
		return fmt.Errorf("default_rps must be greater than 0")
	}
	if c.DefaultBurst <= 0 {
		return fmt.Errorf("default_burst must be greater than 0")
	}
	if c.CleanupInterval <= 0 {
		c.CleanupInterval = 5 * time.Minute
	}

	// Validate endpoint limits
	for path, limit := range c.EndpointLimits {
		if limit.Enabled {
			if limit.RPS <= 0 {
				return fmt.Errorf("endpoint %s: rps must be greater than 0", path)
			}
			if limit.Burst <= 0 {
				return fmt.Errorf("endpoint %s: burst must be greater than 0", path)
			}
		}
	}

	// Validate tier limits
	for tier, limit := range c.AccountTiers {
		if limit.RequestsPerMinute <= 0 {
			return fmt.Errorf("tier %s: requests_per_minute must be greater than 0", tier)
		}
		if limit.BurstSize <= 0 {
			return fmt.Errorf("tier %s: burst_size must be greater than 0", tier)
		}
	}

	// Validate adaptive config
	if c.AdaptiveConfig != nil && c.AdaptiveConfig.Enabled {
		if c.AdaptiveConfig.TrustMultiplier <= 0 {
			return fmt.Errorf("adaptive.trust_multiplier must be greater than 0")
		}
		if c.AdaptiveConfig.SuspicionMultiplier <= 0 || c.AdaptiveConfig.SuspicionMultiplier > 1 {
			return fmt.Errorf("adaptive.suspicion_multiplier must be between 0 and 1")
		}
	}

	return nil
}

// GetEndpointLimit returns the rate limit for a specific endpoint
func (c *RateLimitConfig) GetEndpointLimit(method, path string) *EndpointLimit {
	// Try exact match with method
	key := path
	if limit, ok := c.EndpointLimits[key]; ok && (limit.Method == "" || limit.Method == method) {
		return limit
	}

	// Check for wildcard matches
	for _, limit := range c.EndpointLimits {
		if matchesPattern(path, limit.Path) && (limit.Method == "" || limit.Method == method) {
			return limit
		}
	}

	return nil
}

// GetTierLimit returns the rate limit for a specific tier
func (c *RateLimitConfig) GetTierLimit(tier string) *TierLimit {
	if limit, ok := c.AccountTiers[tier]; ok {
		return limit
	}
	// Default to free tier
	return c.AccountTiers["free"]
}

// matchesPattern checks if a path matches a pattern (supports wildcards)
func matchesPattern(path, pattern string) bool {
	// Simple wildcard matching
	// For more complex patterns, use regexp
	if pattern == "*" {
		return true
	}
	if pattern == path {
		return true
	}
	// Add more sophisticated matching if needed
	return false
}

// RateLimitHeaders represents the standard rate limit headers
type RateLimitHeaders struct {
	Limit      int   `json:"limit"`
	Remaining  int   `json:"remaining"`
	Reset      int64 `json:"reset"`                 // Unix timestamp
	RetryAfter int   `json:"retry_after,omitempty"` // Seconds
}

// ToHeaders converts to HTTP headers map
func (h *RateLimitHeaders) ToHeaders() map[string]string {
	headers := map[string]string{
		"X-RateLimit-Limit":     fmt.Sprintf("%d", h.Limit),
		"X-RateLimit-Remaining": fmt.Sprintf("%d", h.Remaining),
		"X-RateLimit-Reset":     fmt.Sprintf("%d", h.Reset),
	}
	if h.RetryAfter > 0 {
		headers["Retry-After"] = fmt.Sprintf("%d", h.RetryAfter)
	}
	return headers
}
