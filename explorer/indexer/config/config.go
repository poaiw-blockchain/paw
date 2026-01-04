package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration for the indexer
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Chain    ChainConfig    `yaml:"chain"`
	Indexer  IndexerConfig  `yaml:"indexer"`
	API      APIConfig      `yaml:"api"`
	Metrics  MetricsConfig  `yaml:"metrics"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Database        string        `yaml:"database"`
	SSLMode         string        `yaml:"ssl_mode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	Password     string        `yaml:"password"`
	DB           int           `yaml:"db"`
	PoolSize     int           `yaml:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns"`
	CacheTTL     time.Duration `yaml:"cache_ttl"`
}

// ChainConfig holds blockchain node configuration
type ChainConfig struct {
	ChainID       string        `yaml:"chain_id"`
	RPCURL        string        `yaml:"rpc_url"`
	GRPCURL       string        `yaml:"grpc_url"`
	WSUrl         string        `yaml:"ws_url"`
	Timeout       time.Duration `yaml:"timeout"`
	RetryAttempts int           `yaml:"retry_attempts"`
	RetryDelay    time.Duration `yaml:"retry_delay"`
}

// IndexerConfig holds indexer-specific configuration
type IndexerConfig struct {
	StartHeight              int64         `yaml:"start_height"`
	BatchSize                int           `yaml:"batch_size"`
	Workers                  int           `yaml:"workers"`
	BlockBuffer              int           `yaml:"block_buffer"`
	IndexBlocks              bool          `yaml:"index_blocks"`
	IndexTransactions        bool          `yaml:"index_transactions"`
	IndexEvents              bool          `yaml:"index_events"`
	IndexAccounts            bool          `yaml:"index_accounts"`
	IndexValidators          bool          `yaml:"index_validators"`
	IndexDEX                 bool          `yaml:"index_dex"`
	IndexOracle              bool          `yaml:"index_oracle"`
	IndexCompute             bool          `yaml:"index_compute"`
	SyncInterval             time.Duration `yaml:"sync_interval"`
	EnableHistoricalIndexing bool          `yaml:"enable_historical_indexing"`
	HistoricalBatchSize      int           `yaml:"historical_batch_size"`
	ParallelFetches          int           `yaml:"parallel_fetches"`
	MaxRetries               int           `yaml:"max_retries"`
	RetryDelay               time.Duration `yaml:"retry_delay"`
}

// APIConfig holds API server configuration
type APIConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	EnableGraphQL   bool          `yaml:"enable_graphql"`
	EnableREST      bool          `yaml:"enable_rest"`
	EnableWebSocket bool          `yaml:"enable_websocket"`
	CORSOrigins     []string      `yaml:"cors_origins"`
	RateLimit       int           `yaml:"rate_limit"`
	Timeout         time.Duration `yaml:"timeout"`
}

// MetricsConfig holds metrics server configuration
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Override with environment variables
	cfg.applyEnvOverrides()

	return &cfg, nil
}

// applyEnvOverrides applies environment variable overrides
func (c *Config) applyEnvOverrides() {
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		c.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		fmt.Sscanf(dbPort, "%d", &c.Database.Port)
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		c.Database.User = dbUser
	}
	if dbPass := os.Getenv("DB_PASSWORD"); dbPass != "" {
		c.Database.Password = dbPass
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		c.Database.Database = dbName
	}

	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		c.Redis.Host = redisHost
	}
	if redisPort := os.Getenv("REDIS_PORT"); redisPort != "" {
		fmt.Sscanf(redisPort, "%d", &c.Redis.Port)
	}
	if redisPass := os.Getenv("REDIS_PASSWORD"); redisPass != "" {
		c.Redis.Password = redisPass
	}

	if rpcURL := os.Getenv("RPC_URL"); rpcURL != "" {
		c.Chain.RPCURL = rpcURL
	}
	if grpcURL := os.Getenv("GRPC_URL"); grpcURL != "" {
		c.Chain.GRPCURL = grpcURL
	}
	if wsURL := os.Getenv("WS_URL"); wsURL != "" {
		c.Chain.WSUrl = wsURL
	}
	if chainID := os.Getenv("CHAIN_ID"); chainID != "" {
		c.Chain.ChainID = chainID
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Database validation
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Port == 0 {
		return fmt.Errorf("database port is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	// Redis validation
	if c.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
	}
	if c.Redis.Port == 0 {
		return fmt.Errorf("redis port is required")
	}

	// Chain validation
	if c.Chain.ChainID == "" {
		return fmt.Errorf("chain ID is required")
	}
	if c.Chain.RPCURL == "" {
		return fmt.Errorf("RPC URL is required")
	}
	if c.Chain.GRPCURL == "" {
		return fmt.Errorf("gRPC URL is required")
	}

	// Indexer validation
	if c.Indexer.BatchSize <= 0 {
		c.Indexer.BatchSize = 100
	}
	if c.Indexer.Workers <= 0 {
		c.Indexer.Workers = 4
	}
	if c.Indexer.BlockBuffer <= 0 {
		c.Indexer.BlockBuffer = 1000
	}

	// API validation
	if c.API.Port == 0 {
		return fmt.Errorf("API port is required")
	}
	if c.API.RateLimit <= 0 {
		c.API.RateLimit = 100
	}

	// Metrics validation
	if c.Metrics.Enabled && c.Metrics.Port == 0 {
		return fmt.Errorf("metrics port is required when metrics are enabled")
	}

	return nil
}

// GetConnectionString returns the PostgreSQL connection string
func (c *DatabaseConfig) GetConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}

// GetRedisAddr returns the Redis connection address
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
