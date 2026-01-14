package alerting

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds configuration for the alerting system
type Config struct {
	// Server configuration
	HTTPPort    int    `yaml:"http_port"`
	Environment string `yaml:"environment"`

	// Database configuration
	DatabaseURL string `yaml:"database_url"`
	RedisURL    string `yaml:"redis_url"`

	// Rule engine configuration
	EvaluationInterval  time.Duration `yaml:"evaluation_interval"`
	DefaultForDuration  time.Duration `yaml:"default_for_duration"`
	MaxConcurrentEvals  int           `yaml:"max_concurrent_evals"`
	EnableDeduplication bool          `yaml:"enable_deduplication"`
	DeduplicationWindow time.Duration `yaml:"deduplication_window"`
	EnableGrouping      bool          `yaml:"enable_grouping"`
	GroupingWindow      time.Duration `yaml:"grouping_window"`

	// Notification configuration
	MaxRetries          int           `yaml:"max_retries"`
	RetryBackoff        time.Duration `yaml:"retry_backoff"`
	NotificationTimeout time.Duration `yaml:"notification_timeout"`
	BatchNotifications  bool          `yaml:"batch_notifications"`
	BatchSize           int           `yaml:"batch_size"`
	BatchDelay          time.Duration `yaml:"batch_delay"`

	// Channel configurations
	WebhookConfig WebhookChannelConfig `yaml:"webhook_config"`
	EmailConfig   EmailChannelConfig   `yaml:"email_config"`
	SMSConfig     SMSChannelConfig     `yaml:"sms_config"`
	SlackConfig   SlackChannelConfig   `yaml:"slack_config"`
	DiscordConfig DiscordChannelConfig `yaml:"discord_config"`

	// Integration URLs
	PrometheusURL string `yaml:"prometheus_url"`
	ExplorerURL   string `yaml:"explorer_url"`
	AdminAPIURL   string `yaml:"admin_api_url"`

	// Alert retention
	AlertRetentionDays int `yaml:"alert_retention_days"`

	// Security
	JWTSecret      string   `yaml:"jwt_secret"`
	AdminWhitelist []string `yaml:"admin_whitelist"`
}

// WebhookChannelConfig holds webhook channel configuration
type WebhookChannelConfig struct {
	Timeout            time.Duration     `yaml:"timeout"`
	InsecureSkipVerify bool              `yaml:"insecure_skip_verify"`
	DefaultHeaders     map[string]string `yaml:"default_headers"`
}

// EmailChannelConfig holds email channel configuration
type EmailChannelConfig struct {
	SMTPHost    string `yaml:"smtp_host"`
	SMTPPort    int    `yaml:"smtp_port"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	FromAddress string `yaml:"from_address"`
	FromName    string `yaml:"from_name"`
	UseTLS      bool   `yaml:"use_tls"`
	UseStartTLS bool   `yaml:"use_starttls"`
}

// SMSChannelConfig holds SMS channel configuration (Twilio)
type SMSChannelConfig struct {
	AccountSID  string `yaml:"account_sid"`
	AuthToken   string `yaml:"auth_token"`
	FromNumber  string `yaml:"from_number"`
	APIEndpoint string `yaml:"api_endpoint"`
}

// SlackChannelConfig holds Slack channel configuration
type SlackChannelConfig struct {
	DefaultWebhookURL string `yaml:"default_webhook_url"`
	BotToken          string `yaml:"bot_token"`
	DefaultChannel    string `yaml:"default_channel"`
}

// DiscordChannelConfig holds Discord channel configuration
type DiscordChannelConfig struct {
	DefaultWebhookURL string `yaml:"default_webhook_url"`
	BotToken          string `yaml:"bot_token"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		// Defaults
		HTTPPort:            11210,
		Environment:         "development",
		EvaluationInterval:  10 * time.Second,
		DefaultForDuration:  1 * time.Minute,
		MaxConcurrentEvals:  10,
		EnableDeduplication: true,
		DeduplicationWindow: 5 * time.Minute,
		EnableGrouping:      true,
		GroupingWindow:      1 * time.Minute,
		MaxRetries:          3,
		RetryBackoff:        5 * time.Second,
		NotificationTimeout: 30 * time.Second,
		BatchNotifications:  false,
		BatchSize:           10,
		BatchDelay:          30 * time.Second,
		AlertRetentionDays:  90,
		WebhookConfig: WebhookChannelConfig{
			Timeout:            10 * time.Second,
			InsecureSkipVerify: false,
		},
		EmailConfig: EmailChannelConfig{
			SMTPPort:    587,
			UseTLS:      false,
			UseStartTLS: true,
		},
		SMSConfig: SMSChannelConfig{
			APIEndpoint: "https://api.twilio.com/2010-04-01",
		},
	}

	// Load from config file if exists
	if configFile := os.Getenv("ALERTING_CONFIG_FILE"); configFile != "" {
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

	if port := os.Getenv("ALERTING_HTTP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.HTTPPort = p
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

	if promURL := os.Getenv("PROMETHEUS_URL"); promURL != "" {
		cfg.PrometheusURL = promURL
	}

	if explorerURL := os.Getenv("EXPLORER_URL"); explorerURL != "" {
		cfg.ExplorerURL = explorerURL
	}

	if adminURL := os.Getenv("ADMIN_API_URL"); adminURL != "" {
		cfg.AdminAPIURL = adminURL
	}

	// Email configuration from environment
	if host := os.Getenv("SMTP_HOST"); host != "" {
		cfg.EmailConfig.SMTPHost = host
	}
	if port := os.Getenv("SMTP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.EmailConfig.SMTPPort = p
		}
	}
	if user := os.Getenv("SMTP_USERNAME"); user != "" {
		cfg.EmailConfig.Username = user
	}
	if pass := os.Getenv("SMTP_PASSWORD"); pass != "" {
		cfg.EmailConfig.Password = pass
	}
	if from := os.Getenv("SMTP_FROM_ADDRESS"); from != "" {
		cfg.EmailConfig.FromAddress = from
	}

	// SMS configuration from environment
	if sid := os.Getenv("TWILIO_ACCOUNT_SID"); sid != "" {
		cfg.SMSConfig.AccountSID = sid
	}
	if token := os.Getenv("TWILIO_AUTH_TOKEN"); token != "" {
		cfg.SMSConfig.AuthToken = token
	}
	if from := os.Getenv("TWILIO_FROM_NUMBER"); from != "" {
		cfg.SMSConfig.FromNumber = from
	}

	// Slack configuration from environment
	if webhook := os.Getenv("SLACK_WEBHOOK_URL"); webhook != "" {
		cfg.SlackConfig.DefaultWebhookURL = webhook
	}
	if token := os.Getenv("SLACK_BOT_TOKEN"); token != "" {
		cfg.SlackConfig.BotToken = token
	}

	// Discord configuration from environment
	if webhook := os.Getenv("DISCORD_WEBHOOK_URL"); webhook != "" {
		cfg.DiscordConfig.DefaultWebhookURL = webhook
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}
