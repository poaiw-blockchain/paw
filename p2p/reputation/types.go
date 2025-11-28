package reputation

import (
	"net"
	"time"
)

// PeerID represents a unique identifier for a peer
type PeerID string

// PeerReputation stores reputation data for a single peer
type PeerReputation struct {
	PeerID    PeerID    `json:"peer_id"`
	Address   string    `json:"address"`
	Score     float64   `json:"score"` // 0-100
	LastSeen  time.Time `json:"last_seen"`
	FirstSeen time.Time `json:"first_seen"`

	// Behavior metrics
	Metrics PeerMetrics `json:"metrics"`

	// Ban status
	BanStatus BanInfo `json:"ban_status"`

	// Trust level
	TrustLevel TrustLevel `json:"trust_level"`

	// Geographic and network info
	NetworkInfo NetworkInfo `json:"network_info"`
}

// PeerMetrics tracks peer behavior metrics
type PeerMetrics struct {
	// Uptime tracking
	TotalUptime        time.Duration `json:"total_uptime"`
	LastUptimeUpdate   time.Time     `json:"last_uptime_update"`
	ConnectionCount    int64         `json:"connection_count"`
	DisconnectionCount int64         `json:"disconnection_count"`

	// Message statistics
	ValidMessages     int64   `json:"valid_messages"`
	InvalidMessages   int64   `json:"invalid_messages"`
	TotalMessages     int64   `json:"total_messages"`
	ValidMessageRatio float64 `json:"valid_message_ratio"` // cached

	// Performance metrics
	AvgResponseLatency  time.Duration `json:"avg_response_latency"`
	MaxResponseLatency  time.Duration `json:"max_response_latency"`
	MinResponseLatency  time.Duration `json:"min_response_latency"`
	LatencyMeasurements int64         `json:"latency_measurements"`

	// Block propagation
	BlocksPropagated    int64         `json:"blocks_propagated"`
	AvgBlockPropagation time.Duration `json:"avg_block_propagation"`
	FastBlockCount      int64         `json:"fast_block_count"` // blocks < 1s

	// Protocol violations
	ProtocolViolations int64 `json:"protocol_violations"`
	MalformedMessages  int64 `json:"malformed_messages"`
	DoubleSignAttempts int64 `json:"double_sign_attempts"`
	InvalidBlockProps  int64 `json:"invalid_block_proposals"`
	SpamAttempts       int64 `json:"spam_attempts"`

	// DoS and security violations
	OversizedMessages   int64     `json:"oversized_messages"`    // Count of oversized message attempts
	BandwidthViolations int64     `json:"bandwidth_violations"`  // Excessive bandwidth usage
	SecurityEvents      int64     `json:"security_events"`       // Chain ID mismatch, identity spoofing, etc.
	LastViolation       time.Time `json:"last_violation"`        // Timestamp of last violation
	ViolationStreak     int64     `json:"violation_streak"`      // Consecutive violations without recovery
	TotalPenaltyPoints  int64     `json:"total_penalty_points"`  // Accumulated penalty points

	// Resource usage
	BytesSent     int64 `json:"bytes_sent"`
	BytesReceived int64 `json:"bytes_received"`

	// Recent behavior (sliding window - last 24h)
	RecentScores []ScoreSnapshot `json:"recent_scores"`
}

// ScoreSnapshot captures score at a point in time
type ScoreSnapshot struct {
	Timestamp time.Time `json:"timestamp"`
	Score     float64   `json:"score"`
	Reason    string    `json:"reason,omitempty"`
}

// BanInfo tracks ban status
type BanInfo struct {
	IsBanned      bool      `json:"is_banned"`
	BanType       BanType   `json:"ban_type"`
	BannedAt      time.Time `json:"banned_at,omitempty"`
	BanExpires    time.Time `json:"ban_expires,omitempty"`
	BanReason     string    `json:"ban_reason,omitempty"`
	BanCount      int       `json:"ban_count"`
	IsWhitelisted bool      `json:"is_whitelisted"`
}

// BanType defines types of bans
type BanType int

const (
	BanTypeNone BanType = iota
	BanTypeTemporary
	BanTypePermanent
)

func (bt BanType) String() string {
	switch bt {
	case BanTypeNone:
		return "none"
	case BanTypeTemporary:
		return "temporary"
	case BanTypePermanent:
		return "permanent"
	default:
		return "unknown"
	}
}

// TrustLevel represents peer trust classification
type TrustLevel int

const (
	TrustLevelUnknown TrustLevel = iota
	TrustLevelUntrusted
	TrustLevelLow
	TrustLevelMedium
	TrustLevelHigh
	TrustLevelWhitelisted
)

func (tl TrustLevel) String() string {
	switch tl {
	case TrustLevelUnknown:
		return "unknown"
	case TrustLevelUntrusted:
		return "untrusted"
	case TrustLevelLow:
		return "low"
	case TrustLevelMedium:
		return "medium"
	case TrustLevelHigh:
		return "high"
	case TrustLevelWhitelisted:
		return "whitelisted"
	default:
		return "unknown"
	}
}

// NetworkInfo stores network and geographic data
type NetworkInfo struct {
	IPAddress     string    `json:"ip_address"`
	Subnet        string    `json:"subnet"` // /24 subnet
	ASN           int       `json:"asn"`    // Autonomous System Number
	Country       string    `json:"country"`
	Region        string    `json:"region"`
	ISP           string    `json:"isp"`
	LastGeoLookup time.Time `json:"last_geo_lookup"`
}

// ScoreWeights defines weights for scoring algorithm
type ScoreWeights struct {
	UptimeWeight          float64 `json:"uptime_weight"`
	MessageValidityWeight float64 `json:"message_validity_weight"`
	LatencyWeight         float64 `json:"latency_weight"`
	BlockPropWeight       float64 `json:"block_propagation_weight"`
	ViolationPenalty      float64 `json:"violation_penalty"`
}

// DefaultScoreWeights returns default scoring weights
func DefaultScoreWeights() ScoreWeights {
	return ScoreWeights{
		UptimeWeight:          0.25,
		MessageValidityWeight: 0.30,
		LatencyWeight:         0.20,
		BlockPropWeight:       0.15,
		ViolationPenalty:      0.10,
	}
}

// PeerEvent represents an event that affects reputation
type PeerEvent struct {
	PeerID    PeerID    `json:"peer_id"`
	EventType EventType `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      EventData `json:"data,omitempty"`
}

// EventType defines types of peer events
type EventType int

const (
	EventTypeConnected EventType = iota
	EventTypeDisconnected
	EventTypeValidMessage
	EventTypeInvalidMessage
	EventTypeBlockPropagated
	EventTypeProtocolViolation
	EventTypeDoubleSign
	EventTypeInvalidBlock
	EventTypeSpam
	EventTypeLatencyMeasured
	EventTypeSecurity         // Generic security event (chain ID mismatch, etc.)
	EventTypeMisbehavior      // General misbehavior
	EventTypeOversizedMessage // DoS attack via oversized messages
	EventTypeBandwidthAbuse   // Excessive bandwidth usage
)

func (et EventType) String() string {
	switch et {
	case EventTypeConnected:
		return "connected"
	case EventTypeDisconnected:
		return "disconnected"
	case EventTypeValidMessage:
		return "valid_message"
	case EventTypeInvalidMessage:
		return "invalid_message"
	case EventTypeBlockPropagated:
		return "block_propagated"
	case EventTypeProtocolViolation:
		return "protocol_violation"
	case EventTypeDoubleSign:
		return "double_sign"
	case EventTypeInvalidBlock:
		return "invalid_block"
	case EventTypeSpam:
		return "spam"
	case EventTypeLatencyMeasured:
		return "latency_measured"
	case EventTypeSecurity:
		return "security_event"
	case EventTypeMisbehavior:
		return "misbehavior"
	case EventTypeOversizedMessage:
		return "oversized_message"
	case EventTypeBandwidthAbuse:
		return "bandwidth_abuse"
	default:
		return "unknown"
	}
}

// EventData holds event-specific data
type EventData struct {
	Latency       time.Duration `json:"latency,omitempty"`
	MessageSize   int64         `json:"message_size,omitempty"`
	BlockHeight   int64         `json:"block_height,omitempty"`
	ViolationType string        `json:"violation_type,omitempty"`
	Details       string        `json:"details,omitempty"`
}

// ReputationSnapshot represents state at a point in time
type ReputationSnapshot struct {
	Timestamp   time.Time                  `json:"timestamp"`
	TotalPeers  int                        `json:"total_peers"`
	BannedPeers int                        `json:"banned_peers"`
	AvgScore    float64                    `json:"avg_score"`
	Peers       map[PeerID]*PeerReputation `json:"peers"`
}

// SubnetStats tracks statistics per subnet
type SubnetStats struct {
	Subnet      string    `json:"subnet"`
	PeerCount   int       `json:"peer_count"`
	AvgScore    float64   `json:"avg_score"`
	BannedCount int       `json:"banned_count"`
	LastUpdated time.Time `json:"last_updated"`
}

// GeographicStats tracks geographic distribution
type GeographicStats struct {
	Country     string    `json:"country"`
	PeerCount   int       `json:"peer_count"`
	AvgScore    float64   `json:"avg_score"`
	BannedCount int       `json:"banned_count"`
	LastUpdated time.Time `json:"last_updated"`
}

// ParseSubnet extracts /24 subnet from IP address
func ParseSubnet(ipAddr string) string {
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return ""
	}

	// Get /24 subnet
	if ip.To4() != nil {
		// IPv4
		mask := net.CIDRMask(24, 32)
		network := ip.Mask(mask)
		return network.String() + "/24"
	} else {
		// IPv6 - use /48
		mask := net.CIDRMask(48, 128)
		network := ip.Mask(mask)
		return network.String() + "/48"
	}
}

// CalculateTrustLevel determines trust level based on score
func CalculateTrustLevel(score float64, isWhitelisted bool) TrustLevel {
	if isWhitelisted {
		return TrustLevelWhitelisted
	}

	switch {
	case score < 0:
		return TrustLevelUnknown
	case score < 20:
		return TrustLevelUntrusted
	case score < 50:
		return TrustLevelLow
	case score < 75:
		return TrustLevelMedium
	case score <= 100:
		return TrustLevelHigh
	default:
		return TrustLevelUnknown
	}
}
