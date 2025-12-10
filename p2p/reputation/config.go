package reputation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config is the main configuration for the reputation system
type Config struct {
	// Enable/disable reputation system
	Enabled bool `json:"enabled"`

	// Storage configuration
	Storage StorageConfig `json:"storage"`

	// Scoring configuration
	Scoring ScoringConfigJSON `json:"scoring"`

	// Manager configuration
	Manager ManagerConfigJSON `json:"manager"`

	// Security configuration
	Security SecurityConfig `json:"security"`

	// Whitelist of trusted peers (never banned)
	Whitelist []string `json:"whitelist"`
}

// StorageConfig configures storage
type StorageConfig struct {
	Type          string        `json:"type"` // "file" or "memory"
	DataDir       string        `json:"data_dir"`
	CacheSize     int           `json:"cache_size"`
	FlushInterval time.Duration `json:"flush_interval"`
	EnableCache   bool          `json:"enable_cache"`
}

// ScoringConfigJSON is JSON-serializable scoring config
type ScoringConfigJSON struct {
	// Weights
	UptimeWeight          float64 `json:"uptime_weight"`
	MessageValidityWeight float64 `json:"message_validity_weight"`
	LatencyWeight         float64 `json:"latency_weight"`
	BlockPropWeight       float64 `json:"block_propagation_weight"`
	ViolationPenalty      float64 `json:"violation_penalty"`

	// Time windows
	ScoreDecayPeriod time.Duration `json:"score_decay_period"`
	ScoreDecayFactor float64       `json:"score_decay_factor"`
	MinScoreHistory  time.Duration `json:"min_score_history"`

	// Thresholds
	MinUptimeForGoodScore  time.Duration `json:"min_uptime_for_good_score"`
	MaxLatencyForGoodScore time.Duration `json:"max_latency_for_good_score"`
	MinValidMessageRatio   float64       `json:"min_valid_message_ratio"`
	FastBlockThreshold     time.Duration `json:"fast_block_threshold"`

	// Penalties
	ViolationPenaltyScore   float64 `json:"violation_penalty_score"`
	DoubleSignPenalty       float64 `json:"double_sign_penalty"`
	InvalidBlockPenalty     float64 `json:"invalid_block_penalty"`
	SpamPenalty             float64 `json:"spam_penalty"`
	MalformedMessagePenalty float64 `json:"malformed_message_penalty"`

	// Scoring caps
	MaxScore          float64 `json:"max_score"`
	MinScore          float64 `json:"min_score"`
	NewPeerStartScore float64 `json:"new_peer_start_score"`
}

// ManagerConfigJSON is JSON-serializable manager config
type ManagerConfigJSON struct {
	// Security limits
	MaxPeersPerSubnet      int `json:"max_peers_per_subnet"`
	MaxPeersPerCountry     int `json:"max_peers_per_country"`
	MinGeographicDiversity int `json:"min_geographic_diversity"`
	MaxPeersPerASN         int `json:"max_peers_per_asn"`

	// Ban settings
	EnableAutoBan   bool          `json:"enable_auto_ban"`
	TempBanDuration time.Duration `json:"temp_ban_duration"`
	MaxTempBans     int           `json:"max_temp_bans"`

	// Maintenance
	SnapshotInterval   time.Duration `json:"snapshot_interval"`
	CleanupInterval    time.Duration `json:"cleanup_interval"`
	CleanupAge         time.Duration `json:"cleanup_age"`
	ScoreDecayInterval time.Duration `json:"score_decay_interval"`

	// Performance
	EnableGeoLookup        bool          `json:"enable_geo_lookup"`
	GeoLookupCacheDuration time.Duration `json:"geo_lookup_cache_duration"`
}

// SecurityConfig configures security features
type SecurityConfig struct {
	// Sybil attack prevention
	MaxNewPeersPerHour     int     `json:"max_new_peers_per_hour"`
	MaxPeersFromNewSubnets int     `json:"max_peers_from_new_subnets"`
	NewPeerScoreThreshold  float64 `json:"new_peer_score_threshold"`

	// Eclipse attack prevention
	RequireGeoDiversity   bool    `json:"require_geo_diversity"`
	MinDifferentCountries int     `json:"min_different_countries"`
	MaxPercentFromCountry float64 `json:"max_percent_from_country"`

	// Rate limiting
	EnableRateLimiting      bool          `json:"enable_rate_limiting"`
	MaxMessagesPerSecond    int           `json:"max_messages_per_second"`
	MaxBlocksPerSecond      int           `json:"max_blocks_per_second"`
	RateLimitWindowDuration time.Duration `json:"rate_limit_window_duration"`

	// Connection limits
	MaxInboundConnections  int `json:"max_inbound_connections"`
	MaxOutboundConnections int `json:"max_outbound_connections"`
}

// DefaultConfig returns default configuration
func DefaultConfig(homeDir string) Config {
	return Config{
		Enabled: true,

		Storage: StorageConfig{
			Type:          "file",
			DataDir:       filepath.Join(homeDir, "data", "p2p", "reputation"),
			CacheSize:     1000,
			FlushInterval: 30 * time.Second,
			EnableCache:   true,
		},

		Scoring: ScoringConfigJSON{
			UptimeWeight:          0.25,
			MessageValidityWeight: 0.30,
			LatencyWeight:         0.20,
			BlockPropWeight:       0.15,
			ViolationPenalty:      0.10,

			ScoreDecayPeriod: 24 * time.Hour,
			ScoreDecayFactor: 0.95,
			MinScoreHistory:  7 * 24 * time.Hour,

			MinUptimeForGoodScore:  1 * time.Hour,
			MaxLatencyForGoodScore: 500 * time.Millisecond,
			MinValidMessageRatio:   0.95,
			FastBlockThreshold:     1 * time.Second,

			ViolationPenaltyScore:   5.0,
			DoubleSignPenalty:       100.0,
			InvalidBlockPenalty:     20.0,
			SpamPenalty:             10.0,
			MalformedMessagePenalty: 2.0,

			MaxScore:          100.0,
			MinScore:          0.0,
			NewPeerStartScore: 50.0,
		},

		Manager: ManagerConfigJSON{
			MaxPeersPerSubnet:      10,
			MaxPeersPerCountry:     50,
			MinGeographicDiversity: 3,
			MaxPeersPerASN:         15,

			EnableAutoBan:   true,
			TempBanDuration: 24 * time.Hour,
			MaxTempBans:     3,

			SnapshotInterval:   1 * time.Hour,
			CleanupInterval:    24 * time.Hour,
			CleanupAge:         30 * 24 * time.Hour,
			ScoreDecayInterval: 1 * time.Hour,

			EnableGeoLookup:        false,
			GeoLookupCacheDuration: 7 * 24 * time.Hour,
		},

		Security: SecurityConfig{
			MaxNewPeersPerHour:     100,
			MaxPeersFromNewSubnets: 20,
			NewPeerScoreThreshold:  30.0,

			RequireGeoDiversity:   true,
			MinDifferentCountries: 3,
			MaxPercentFromCountry: 0.40, // Max 40% from one country

			EnableRateLimiting:      true,
			MaxMessagesPerSecond:    100,
			MaxBlocksPerSecond:      10,
			RateLimitWindowDuration: 10 * time.Second,

			MaxInboundConnections:  50,
			MaxOutboundConnections: 50,
		},

		Whitelist: []string{},
	}
}

// LoadConfig loads configuration from file
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath) // #nosec G304 - configuration path supplied by operator
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, filePath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ToScoringConfig converts JSON config to internal config
func (c *ScoringConfigJSON) ToScoringConfig() ScoringConfig {
	return ScoringConfig{
		ScoreDecayPeriod:        c.ScoreDecayPeriod,
		ScoreDecayFactor:        c.ScoreDecayFactor,
		MinScoreHistory:         c.MinScoreHistory,
		MinUptimeForGoodScore:   c.MinUptimeForGoodScore,
		MaxLatencyForGoodScore:  c.MaxLatencyForGoodScore,
		MinValidMessageRatio:    c.MinValidMessageRatio,
		FastBlockThreshold:      c.FastBlockThreshold,
		ViolationPenalty:        c.ViolationPenaltyScore,
		DoubleSignPenalty:       c.DoubleSignPenalty,
		InvalidBlockPenalty:     c.InvalidBlockPenalty,
		SpamPenalty:             c.SpamPenalty,
		MalformedMessagePenalty: c.MalformedMessagePenalty,
		MaxScore:                c.MaxScore,
		MinScore:                c.MinScore,
		NewPeerStartScore:       c.NewPeerStartScore,
	}
}

// ToScoreWeights converts JSON config to score weights
func (c *ScoringConfigJSON) ToScoreWeights() ScoreWeights {
	return ScoreWeights{
		UptimeWeight:          c.UptimeWeight,
		MessageValidityWeight: c.MessageValidityWeight,
		LatencyWeight:         c.LatencyWeight,
		BlockPropWeight:       c.BlockPropWeight,
		ViolationPenalty:      c.ViolationPenalty,
	}
}

// ToManagerConfig converts JSON config to internal config
func (c *ManagerConfigJSON) ToManagerConfig(scoringConfig ScoringConfig, scoreWeights ScoreWeights) ManagerConfig {
	return ManagerConfig{
		ScoreWeights:           scoreWeights,
		ScoringConfig:          scoringConfig,
		MaxPeersPerSubnet:      c.MaxPeersPerSubnet,
		MaxPeersPerCountry:     c.MaxPeersPerCountry,
		MinGeographicDiversity: c.MinGeographicDiversity,
		MaxPeersPerASN:         c.MaxPeersPerASN,
		EnableAutoBan:          c.EnableAutoBan,
		TempBanDuration:        c.TempBanDuration,
		MaxTempBans:            c.MaxTempBans,
		SnapshotInterval:       c.SnapshotInterval,
		CleanupInterval:        c.CleanupInterval,
		CleanupAge:             c.CleanupAge,
		ScoreDecayInterval:     c.ScoreDecayInterval,
		EnableGeoLookup:        c.EnableGeoLookup,
		GeoLookupCacheDuration: c.GeoLookupCacheDuration,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Storage.CacheSize < 0 {
		return fmt.Errorf("cache size must be >= 0")
	}

	if c.Scoring.MaxScore <= c.Scoring.MinScore {
		return fmt.Errorf("max score must be greater than min score")
	}

	if c.Scoring.NewPeerStartScore < c.Scoring.MinScore || c.Scoring.NewPeerStartScore > c.Scoring.MaxScore {
		return fmt.Errorf("new peer start score must be between min and max score")
	}

	if c.Scoring.ScoreDecayFactor < 0 || c.Scoring.ScoreDecayFactor > 1 {
		return fmt.Errorf("score decay factor must be between 0 and 1")
	}

	if c.Manager.MaxPeersPerSubnet < 1 {
		return fmt.Errorf("max peers per subnet must be >= 1")
	}

	if c.Security.MaxPercentFromCountry < 0 || c.Security.MaxPercentFromCountry > 1 {
		return fmt.Errorf("max percent from country must be between 0 and 1")
	}

	return nil
}
