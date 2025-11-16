package discovery

import (
	"fmt"
	"net"
	"time"

	"github.com/paw-chain/paw/p2p/reputation"
)

// PeerAddr represents a peer network address with metadata
type PeerAddr struct {
	ID         reputation.PeerID `json:"id"`
	Address    string            `json:"address"`
	Port       uint16            `json:"port"`
	LastSeen   time.Time         `json:"last_seen"`
	FirstSeen  time.Time         `json:"first_seen"`
	Attempts   int               `json:"attempts"`
	LastDialed time.Time         `json:"last_dialed"`
	Source     PeerSource        `json:"source"`
	Bucket     int               `json:"bucket"`
}

// PeerSource indicates how we learned about a peer
type PeerSource int

const (
	PeerSourceUnknown PeerSource = iota
	PeerSourceSeed
	PeerSourceBootstrap
	PeerSourcePEX        // Peer Exchange
	PeerSourceManual     // Manually added
	PeerSourcePersistent // From persistent_peers config
	PeerSourceInbound    // Incoming connection
)

func (ps PeerSource) String() string {
	switch ps {
	case PeerSourceSeed:
		return "seed"
	case PeerSourceBootstrap:
		return "bootstrap"
	case PeerSourcePEX:
		return "pex"
	case PeerSourceManual:
		return "manual"
	case PeerSourcePersistent:
		return "persistent"
	case PeerSourceInbound:
		return "inbound"
	default:
		return "unknown"
	}
}

// PeerConnection represents an active peer connection
type PeerConnection struct {
	PeerAddr     *PeerAddr
	ConnectedAt  time.Time
	Outbound     bool
	Persistent   bool
	LastActivity time.Time
	BytesSent    uint64
	BytesRecv    uint64
	MessagesRecv uint64
	MessagesSent uint64
}

// DiscoveryConfig configures the peer discovery service
type DiscoveryConfig struct {
	// Seed nodes
	Seeds []string

	// Bootstrap nodes (required initial connections)
	BootstrapNodes []string

	// Persistent peers (always maintain connection)
	PersistentPeers []string

	// Private peer IDs (won't be gossiped)
	PrivatePeerIDs []string

	// Unconditional peer IDs (always connect, ignore reputation)
	UnconditionalPeerIDs []string

	// Connection limits
	MaxInboundPeers  int
	MaxOutboundPeers int
	MaxPeers         int

	// Discovery settings
	EnablePEX                   bool // Peer Exchange Protocol
	PEXInterval                 time.Duration
	MinOutboundPeers            int
	DialTimeout                 time.Duration
	HandshakeTimeout            time.Duration
	PersistentPeerMaxDialPeriod time.Duration

	// Address book settings
	AddressBookStrict bool
	AddressBookSize   int

	// Reconnection
	EnableAutoReconnect  bool
	ReconnectInterval    time.Duration
	MaxReconnectAttempts int

	// Health checks
	PingInterval      time.Duration
	PingTimeout       time.Duration
	InactivityTimeout time.Duration

	// Listen address
	ListenAddress   string
	ExternalAddress string
}

// DefaultDiscoveryConfig returns default discovery configuration
func DefaultDiscoveryConfig() DiscoveryConfig {
	return DiscoveryConfig{
		Seeds:                       []string{},
		BootstrapNodes:              []string{},
		PersistentPeers:             []string{},
		PrivatePeerIDs:              []string{},
		UnconditionalPeerIDs:        []string{},
		MaxInboundPeers:             50,
		MaxOutboundPeers:            50,
		MaxPeers:                    100,
		EnablePEX:                   true,
		PEXInterval:                 30 * time.Second,
		MinOutboundPeers:            10,
		DialTimeout:                 10 * time.Second,
		HandshakeTimeout:            20 * time.Second,
		PersistentPeerMaxDialPeriod: 0, // Exponential backoff
		AddressBookStrict:           true,
		AddressBookSize:             1000,
		EnableAutoReconnect:         true,
		ReconnectInterval:           30 * time.Second,
		MaxReconnectAttempts:        10,
		PingInterval:                60 * time.Second,
		PingTimeout:                 30 * time.Second,
		InactivityTimeout:           5 * time.Minute,
		ListenAddress:               "tcp://0.0.0.0:26656",
		ExternalAddress:             "",
	}
}

// DiscoveryStats tracks discovery statistics
type DiscoveryStats struct {
	// Peer counts
	TotalPeers      int
	InboundPeers    int
	OutboundPeers   int
	PersistentPeers int

	// Connection stats
	TotalConnections     uint64
	FailedConnections    uint64
	SuccessfulHandshakes uint64
	FailedHandshakes     uint64

	// Peer exchange
	PEXMessages     uint64
	PEXPeersLearned uint64
	PEXPeersShared  uint64

	// Address book
	KnownAddresses int
	GoodAddresses  int
	TriedAddresses int

	// Network traffic
	TotalBytesSent uint64
	TotalBytesRecv uint64

	// Timestamps
	StartTime     time.Time
	LastPEXTime   time.Time
	LastReconnect time.Time
}

// PeerInfo provides detailed peer information
type PeerInfo struct {
	ID              reputation.PeerID
	Address         string
	ConnectedAt     time.Time
	Outbound        bool
	Persistent      bool
	Source          PeerSource
	LastActivity    time.Time
	BytesSent       uint64
	BytesRecv       uint64
	ReputationScore float64
	Version         string
	Moniker         string
}

// DialResult represents the result of a dial attempt
type DialResult struct {
	PeerID    reputation.PeerID
	Address   string
	Success   bool
	Error     error
	Latency   time.Duration
	Timestamp time.Time
}

// PeerFilter is a function that determines if a peer should be considered
type PeerFilter func(*PeerAddr) bool

// ParseNetAddr parses a network address string
func ParseNetAddr(addr string) (*PeerAddr, error) {
	// Expected format: "id@host:port" or "host:port"
	var peerID string
	var hostPort string

	// Check for ID prefix
	parts := splitByAt(addr)
	if len(parts) == 2 {
		peerID = parts[0]
		hostPort = parts[1]
	} else if len(parts) == 1 {
		hostPort = parts[0]
	} else {
		return nil, fmt.Errorf("invalid address format: %s", addr)
	}

	// Parse host:port
	host, portStr, err := net.SplitHostPort(hostPort)
	if err != nil {
		return nil, fmt.Errorf("failed to parse host:port: %w", err)
	}

	// Parse port
	var port uint16
	_, err = fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %s", portStr)
	}

	// Resolve hostname if needed
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve host: %w", err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no IPs found for host: %s", host)
	}

	// Use first IP
	ipAddr := ips[0].String()

	now := time.Now()
	return &PeerAddr{
		ID:        reputation.PeerID(peerID),
		Address:   ipAddr,
		Port:      port,
		FirstSeen: now,
		LastSeen:  now,
		Source:    PeerSourceUnknown,
	}, nil
}

// NetAddr returns the network address string
func (pa *PeerAddr) NetAddr() string {
	if pa.ID != "" {
		return fmt.Sprintf("%s@%s:%d", pa.ID, pa.Address, pa.Port)
	}
	return fmt.Sprintf("%s:%d", pa.Address, pa.Port)
}

// HostPort returns host:port string
func (pa *PeerAddr) HostPort() string {
	return fmt.Sprintf("%s:%d", pa.Address, pa.Port)
}

// IsRoutable checks if the address is routable (not local/private)
func (pa *PeerAddr) IsRoutable() bool {
	ip := net.ParseIP(pa.Address)
	if ip == nil {
		return false
	}

	// Check for non-routable addresses
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}

	// Check for private IP ranges
	if isPrivateIP(ip) {
		return false
	}

	return true
}

// isPrivateIP checks if an IP is in a private range
func isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7", // IPv6 unique local
	}

	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// splitByAt splits a string by '@' character
func splitByAt(s string) []string {
	result := make([]string, 0, 2)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '@' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

// PeerScore calculates a score for peer selection (higher is better)
func (pa *PeerAddr) PeerScore(now time.Time) float64 {
	score := 100.0

	// Penalize for failed attempts
	if pa.Attempts > 0 {
		score -= float64(pa.Attempts) * 5.0
	}

	// Penalize for not being seen recently
	timeSinceLastSeen := now.Sub(pa.LastSeen)
	if timeSinceLastSeen > 24*time.Hour {
		score -= 30.0
	} else if timeSinceLastSeen > time.Hour {
		score -= 10.0
	}

	// Bonus for being seen recently
	if timeSinceLastSeen < 10*time.Minute {
		score += 10.0
	}

	// Bonus for trusted sources
	switch pa.Source {
	case PeerSourceSeed, PeerSourceBootstrap, PeerSourcePersistent:
		score += 20.0
	case PeerSourceManual:
		score += 15.0
	case PeerSourcePEX:
		score += 5.0
	}

	// Ensure score stays in reasonable range
	if score < 0 {
		score = 0
	}

	return score
}
