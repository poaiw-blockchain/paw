package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	Port                int
	MonitorInterval     time.Duration
	MetricsRetention    time.Duration
	IncidentRetention   time.Duration
	BlockchainRPCURL    string
	APIEndpoint         string
	WebSocketEndpoint   string
	ExplorerEndpoint    string
	FaucetEndpoint      string
	AlertEmail          string
	SMTPServer          string
	SMTPPort            int
	SMTPUsername        string
	SMTPPassword        string
	IncidentWebhookURL  string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	port := getEnvAsInt("PORT", 8080)
	monitorInterval := getEnvAsDuration("MONITOR_INTERVAL", 30*time.Second)
	metricsRetention := getEnvAsDuration("METRICS_RETENTION", 7*24*time.Hour)
	incidentRetention := getEnvAsDuration("INCIDENT_RETENTION", 90*24*time.Hour)

	return &Config{
		Port:                port,
		MonitorInterval:     monitorInterval,
		MetricsRetention:    metricsRetention,
		IncidentRetention:   incidentRetention,
		BlockchainRPCURL:    getEnv("BLOCKCHAIN_RPC_URL", "http://localhost:26657"),
		APIEndpoint:         getEnv("API_ENDPOINT", "http://localhost:1317"),
		WebSocketEndpoint:   getEnv("WEBSOCKET_ENDPOINT", "ws://localhost:26657/websocket"),
		ExplorerEndpoint:    getEnv("EXPLORER_ENDPOINT", "http://localhost:3000"),
		FaucetEndpoint:      getEnv("FAUCET_ENDPOINT", "http://localhost:8000"),
		AlertEmail:          getEnv("ALERT_EMAIL", ""),
		SMTPServer:          getEnv("SMTP_SERVER", ""),
		SMTPPort:            getEnvAsInt("SMTP_PORT", 587),
		SMTPUsername:        getEnv("SMTP_USERNAME", ""),
		SMTPPassword:        getEnv("SMTP_PASSWORD", ""),
		IncidentWebhookURL:  getEnv("INCIDENT_WEBHOOK_URL", ""),
	}, nil
}

// Helper functions to get environment variables with defaults
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
