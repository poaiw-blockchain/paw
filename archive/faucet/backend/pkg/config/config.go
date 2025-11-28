package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	Port        string
	Environment string
	CORSOrigins []string

	// Blockchain configuration
	NodeRPC         string
	ChainID         string
	FaucetMnemonic  string
	FaucetAddress   string
	Denom           string
	AmountPerRequest int64

	// Database configuration
	DatabaseURL string

	// Redis configuration
	RedisURL string

	// Rate limiting configuration
	RateLimitPerIP      int
	RateLimitPerAddress int
	RateLimitWindow     time.Duration

	// Captcha configuration
	HCaptchaSecret string

	// Transaction configuration
	GasLimit       uint64
	GasPrice       string
	TransactionMemo string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),
		CORSOrigins: strings.Split(getEnv("CORS_ORIGINS", "*"), ","),

		NodeRPC:         getEnv("NODE_RPC", "http://localhost:26657"),
		ChainID:         getEnv("CHAIN_ID", "paw-testnet-1"),
		FaucetMnemonic:  getEnv("FAUCET_MNEMONIC", ""),
		FaucetAddress:   getEnv("FAUCET_ADDRESS", ""),
		Denom:           getEnv("DENOM", "upaw"),
		AmountPerRequest: getEnvAsInt64("AMOUNT_PER_REQUEST", 100000000), // 100 PAW

		DatabaseURL: getEnv("DATABASE_URL", "postgres://faucet:faucet@localhost:5432/faucet?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),

		RateLimitPerIP:      getEnvAsInt("RATE_LIMIT_PER_IP", 10),
		RateLimitPerAddress: getEnvAsInt("RATE_LIMIT_PER_ADDRESS", 1),
		RateLimitWindow:     time.Duration(getEnvAsInt("RATE_LIMIT_WINDOW_HOURS", 24)) * time.Hour,

		HCaptchaSecret: getEnv("HCAPTCHA_SECRET", ""),

		GasLimit:       uint64(getEnvAsInt("GAS_LIMIT", 200000)),
		GasPrice:       getEnv("GAS_PRICE", "0.025upaw"),
		TransactionMemo: getEnv("TRANSACTION_MEMO", "PAW Testnet Faucet"),
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.NodeRPC == "" {
		return errors.New("NODE_RPC is required")
	}

	if c.ChainID == "" {
		return errors.New("CHAIN_ID is required")
	}

	if c.FaucetMnemonic == "" && c.FaucetAddress == "" {
		return errors.New("either FAUCET_MNEMONIC or FAUCET_ADDRESS is required")
	}

	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	if c.RedisURL == "" {
		return errors.New("REDIS_URL is required")
	}

	if c.AmountPerRequest <= 0 {
		return errors.New("AMOUNT_PER_REQUEST must be positive")
	}

	if c.Environment == "production" && c.HCaptchaSecret == "" {
		return errors.New("HCAPTCHA_SECRET is required in production")
	}

	return nil
}

// RateLimitConfig returns rate limit configuration
func (c *Config) RateLimitConfig() map[string]interface{} {
	return map[string]interface{}{
		"per_ip":      c.RateLimitPerIP,
		"per_address": c.RateLimitPerAddress,
		"window":      c.RateLimitWindow,
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// getEnvAsInt64 gets an environment variable as an int64 or returns a default value
func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}
