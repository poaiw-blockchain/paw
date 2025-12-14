package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the control center backend
type Config struct {
	// Server configuration
	Environment    string `yaml:"environment"`
	HTTPPort       int    `yaml:"http_port"`
	WebSocketPort  int    `yaml:"websocket_port"`

	// Database configuration
	DatabaseURL string `yaml:"database_url"`
	RedisURL    string `yaml:"redis_url"`

	// Authentication configuration
	JWTSecret       string        `yaml:"jwt_secret"`
	TokenExpiration time.Duration `yaml:"token_expiration"`
	AdminWhitelist  []string      `yaml:"admin_whitelist"`

	// Integration URLs
	RPCURL          string `yaml:"rpc_url"`
	PrometheusURL   string `yaml:"prometheus_url"`
	GrafanaURL      string `yaml:"grafana_url"`
	AlertmanagerURL string `yaml:"alertmanager_url"`
	AnalyticsURL    string `yaml:"analytics_url"`

	// Rate limiting
	RateLimitAdmin int `yaml:"rate_limit_admin"`  // requests per minute
	RateLimitRead  int `yaml:"rate_limit_read"`   // requests per minute

	// Circuit breaker configuration
	CircuitBreakerTimeout time.Duration `yaml:"circuit_breaker_timeout"`
	AutoRecovery          bool          `yaml:"auto_recovery"`

	// Audit log configuration
	AuditLogRetention int `yaml:"audit_log_retention"` // days
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	cfg := &Config{
		// Defaults
		Environment:           "development",
		HTTPPort:              11201,
		WebSocketPort:         11202,
		TokenExpiration:       30 * time.Minute,
		RateLimitAdmin:        10,
		RateLimitRead:         100,
		CircuitBreakerTimeout: 5 * time.Minute,
		AutoRecovery:          false,
		AuditLogRetention:     0, // Indefinite
	}

	// Load from config file if exists
	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Override with environment variables
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		cfg.Environment = env
	}

	if port := os.Getenv("HTTP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.HTTPPort = p
		}
	}

	if port := os.Getenv("WEBSOCKET_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.WebSocketPort = p
		}
	}

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.DatabaseURL = dbURL
	}

	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		cfg.RedisURL = redisURL
	}

	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.JWTSecret = secret
	}

	if whitelist := os.Getenv("ADMIN_WHITELIST"); whitelist != "" {
		cfg.AdminWhitelist = strings.Split(whitelist, ",")
	}

	if rpcURL := os.Getenv("RPC_URL"); rpcURL != "" {
		cfg.RPCURL = rpcURL
	}

	if promURL := os.Getenv("PROMETHEUS_URL"); promURL != "" {
		cfg.PrometheusURL = promURL
	}

	if grafanaURL := os.Getenv("GRAFANA_URL"); grafanaURL != "" {
		cfg.GrafanaURL = grafanaURL
	}

	if amURL := os.Getenv("ALERTMANAGER_URL"); amURL != "" {
		cfg.AlertmanagerURL = amURL
	}

	if analyticsURL := os.Getenv("ANALYTICS_URL"); analyticsURL != "" {
		cfg.AnalyticsURL = analyticsURL
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	if cfg.RPCURL == "" {
		return nil, fmt.Errorf("RPC_URL is required")
	}

	return cfg, nil
}

// IsProductionIP checks if an IP is allowed for admin access
func (c *Config) IsProductionIP(ip string) bool {
	if len(c.AdminWhitelist) == 0 {
		return true // No whitelist configured
	}

	for _, allowedIP := range c.AdminWhitelist {
		if matchIP(ip, allowedIP) {
			return true
		}
	}

	return false
}

// matchIP checks if an IP matches a pattern (supports CIDR notation)
func matchIP(ip, pattern string) bool {
	// Simple implementation - in production, use net.ParseCIDR and proper matching
	if pattern == "*" {
		return true
	}

	// Exact match
	if ip == pattern {
		return true
	}

	// CIDR notation (basic support for /16 and /24)
	if strings.Contains(pattern, "/") {
		parts := strings.Split(pattern, "/")
		if len(parts) != 2 {
			return false
		}

		subnet := parts[0]
		mask := parts[1]

		ipParts := strings.Split(ip, ".")
		subnetParts := strings.Split(subnet, ".")

		switch mask {
		case "8":
			return ipParts[0] == subnetParts[0]
		case "16":
			return ipParts[0] == subnetParts[0] && ipParts[1] == subnetParts[1]
		case "24":
			return ipParts[0] == subnetParts[0] && ipParts[1] == subnetParts[1] && ipParts[2] == subnetParts[2]
		}
	}

	return false
}
