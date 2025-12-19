package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config controls monitor targets and schedule.
type Config struct {
	Port          int
	CheckInterval time.Duration
	HTTPTimeout   time.Duration
	RPCEndpoint   string
	RESTEndpoint  string
	GRPCEndpoint  string
	ExplorerURL   string
	FaucetHealth  string
	MetricsURL    string
}

func loadConfig() Config {
	return Config{
		Port:          getEnvInt("STATUS_PAGE_PORT", 11090),
		CheckInterval: getEnvDuration("STATUS_PAGE_INTERVAL", 15*time.Second),
		HTTPTimeout:   getEnvDuration("STATUS_PAGE_TIMEOUT", 6*time.Second),
		RPCEndpoint:   getEnv("STATUS_PAGE_RPC_URL", "http://localhost:26657"),
		RESTEndpoint:  getEnv("STATUS_PAGE_REST_URL", "http://localhost:1317"),
		GRPCEndpoint:  getEnv("STATUS_PAGE_GRPC_ADDR", "localhost:9090"),
		ExplorerURL:   getEnv("STATUS_PAGE_EXPLORER_URL", "http://localhost:11080"),
		FaucetHealth:  getEnv("STATUS_PAGE_FAUCET_HEALTH", "http://localhost:8000/api/v1/health"),
		MetricsURL:    getEnv("STATUS_PAGE_METRICS_URL", "http://localhost:26661/metrics"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func (c Config) Validate() error {
	if c.CheckInterval < 5*time.Second {
		return fmt.Errorf("STATUS_PAGE_INTERVAL too low")
	}
	if c.HTTPTimeout <= 0 {
		return fmt.Errorf("STATUS_PAGE_TIMEOUT must be positive")
	}
	return nil
}
