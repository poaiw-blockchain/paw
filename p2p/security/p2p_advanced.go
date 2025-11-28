package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Task 143: P2P Peer Reputation Persistence

// ReputationStore handles persistent storage of peer reputations
type ReputationStore struct {
	filePath   string
	reputations map[string]*PeerReputation
	mu         sync.RWMutex
}

// PeerReputation represents a peer's reputation data
type PeerReputation struct {
	PeerID          string
	Score           float64
	LastSeen        time.Time
	SuccessfulMsgs  uint64
	FailedMsgs      uint64
	Uptime          time.Duration
	Latency         time.Duration
	BandwidthShared uint64
	Violations      uint64
	BlacklistedUntil *time.Time
}

// NewReputationStore creates a new reputation store
func NewReputationStore(filePath string) (*ReputationStore, error) {
	rs := &ReputationStore{
		filePath:    filePath,
		reputations: make(map[string]*PeerReputation),
	}

	// Load existing reputations
	if err := rs.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Start periodic save
	go rs.periodicSave()

	return rs, nil
}

// Load loads reputations from disk
func (rs *ReputationStore) Load() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	data, err := os.ReadFile(rs.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &rs.reputations)
}

// Save saves reputations to disk
func (rs *ReputationStore) Save() error {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	data, err := json.MarshalIndent(rs.reputations, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(rs.filePath, data, 0600)
}

// periodicSave saves reputations periodically
func (rs *ReputationStore) periodicSave() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if err := rs.Save(); err != nil {
			// Log error but don't crash
			fmt.Printf("Failed to save reputations: %v\n", err)
		}
	}
}

// GetReputation returns a peer's reputation
func (rs *ReputationStore) GetReputation(peerID string) (*PeerReputation, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	rep, exists := rs.reputations[peerID]
	return rep, exists
}

// UpdateReputation updates a peer's reputation
func (rs *ReputationStore) UpdateReputation(peerID string, update func(*PeerReputation)) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	rep, exists := rs.reputations[peerID]
	if !exists {
		rep = &PeerReputation{
			PeerID:   peerID,
			Score:    50.0, // Neutral starting score
			LastSeen: time.Now(),
		}
		rs.reputations[peerID] = rep
	}

	update(rep)
	rep.LastSeen = time.Now()
}

// RemoveReputation removes a peer's reputation
func (rs *ReputationStore) RemoveReputation(peerID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	delete(rs.reputations, peerID)
}

// GetTopPeers returns the top N peers by reputation
func (rs *ReputationStore) GetTopPeers(n int) []*PeerReputation {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	// Create slice of all reputations
	peers := make([]*PeerReputation, 0, len(rs.reputations))
	for _, rep := range rs.reputations {
		// Skip blacklisted peers
		if rep.BlacklistedUntil != nil && time.Now().Before(*rep.BlacklistedUntil) {
			continue
		}
		peers = append(peers, rep)
	}

	// Sort by score (simple bubble sort for small N)
	for i := 0; i < len(peers)-1; i++ {
		for j := 0; j < len(peers)-i-1; j++ {
			if peers[j].Score < peers[j+1].Score {
				peers[j], peers[j+1] = peers[j+1], peers[j]
			}
		}
	}

	// Return top N
	if n > len(peers) {
		n = len(peers)
	}
	return peers[:n]
}

// Task 144: P2P DDoS Protection

// DDoSProtector provides DDoS protection
type DDoSProtector struct {
	peerLimiters map[string]*rate.Limiter
	globalLimiter *rate.Limiter
	mu           sync.RWMutex
	maxPeersPerIP map[string]int
	ipPeerCount  map[string]int
	blacklist    map[string]time.Time
}

// NewDDoSProtector creates a new DDoS protector
func NewDDoSProtector(globalRate, burstRate int) *DDoSProtector {
	dp := &DDoSProtector{
		peerLimiters: make(map[string]*rate.Limiter),
		globalLimiter: rate.NewLimiter(rate.Limit(globalRate), burstRate),
		maxPeersPerIP: make(map[string]int),
		ipPeerCount:  make(map[string]int),
		blacklist:    make(map[string]time.Time),
	}

	// Start cleanup goroutine
	go dp.cleanup()

	return dp
}

// CheckRateLimit checks if a peer is within rate limits
func (dp *DDoSProtector) CheckRateLimit(peerID string) error {
	// Check global rate limit
	if !dp.globalLimiter.Allow() {
		return errors.New("global rate limit exceeded")
	}

	// Check peer-specific rate limit
	dp.mu.RLock()
	limiter, exists := dp.peerLimiters[peerID]
	dp.mu.RUnlock()

	if !exists {
		// Create new limiter for peer (100 msgs/sec, burst 200)
		limiter = rate.NewLimiter(100, 200)
		dp.mu.Lock()
		dp.peerLimiters[peerID] = limiter
		dp.mu.Unlock()
	}

	if !limiter.Allow() {
		return fmt.Errorf("peer %s rate limit exceeded", peerID)
	}

	return nil
}

// CheckBlacklist checks if a peer/IP is blacklisted
func (dp *DDoSProtector) CheckBlacklist(identifier string) error {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	if until, exists := dp.blacklist[identifier]; exists {
		if time.Now().Before(until) {
			return fmt.Errorf("peer/IP blacklisted until %v", until)
		}
		// Blacklist expired, will be cleaned up
	}

	return nil
}

// Blacklist adds a peer/IP to blacklist
func (dp *DDoSProtector) Blacklist(identifier string, duration time.Duration) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	dp.blacklist[identifier] = time.Now().Add(duration)
}

// CheckConnectionLimit checks if IP has too many connections
func (dp *DDoSProtector) CheckConnectionLimit(ip string, maxPerIP int) error {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	count := dp.ipPeerCount[ip]
	if count >= maxPerIP {
		return fmt.Errorf("IP %s has reached connection limit %d", ip, maxPerIP)
	}

	dp.ipPeerCount[ip] = count + 1
	return nil
}

// ReleaseConnection releases a connection slot for an IP
func (dp *DDoSProtector) ReleaseConnection(ip string) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if count := dp.ipPeerCount[ip]; count > 0 {
		dp.ipPeerCount[ip] = count - 1
	}
}

// cleanup removes expired entries
func (dp *DDoSProtector) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		dp.mu.Lock()

		// Clean expired blacklist entries
		now := time.Now()
		for id, until := range dp.blacklist {
			if now.After(until) {
				delete(dp.blacklist, id)
			}
		}

		// Clean inactive peer limiters
		for peerID, limiter := range dp.peerLimiters {
			// Remove limiters that haven't been used recently
			if limiter.Tokens() >= float64(limiter.Burst()) {
				delete(dp.peerLimiters, peerID)
			}
		}

		dp.mu.Unlock()
	}
}

// Task 145: P2P Connection Limiting

// ConnectionManager manages P2P connections
type ConnectionManager struct {
	maxInbound  int
	maxOutbound int
	maxTotal    int
	inbound     map[string]struct{}
	outbound    map[string]struct{}
	mu          sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(maxInbound, maxOutbound, maxTotal int) *ConnectionManager {
	return &ConnectionManager{
		maxInbound:  maxInbound,
		maxOutbound: maxOutbound,
		maxTotal:    maxTotal,
		inbound:     make(map[string]struct{}),
		outbound:    make(map[string]struct{}),
	}
}

// AcceptInbound checks if an inbound connection can be accepted
func (cm *ConnectionManager) AcceptInbound(peerID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	total := len(cm.inbound) + len(cm.outbound)
	if total >= cm.maxTotal {
		return errors.New("maximum total connections reached")
	}

	if len(cm.inbound) >= cm.maxInbound {
		return errors.New("maximum inbound connections reached")
	}

	cm.inbound[peerID] = struct{}{}
	return nil
}

// AcceptOutbound checks if an outbound connection can be made
func (cm *ConnectionManager) AcceptOutbound(peerID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	total := len(cm.inbound) + len(cm.outbound)
	if total >= cm.maxTotal {
		return errors.New("maximum total connections reached")
	}

	if len(cm.outbound) >= cm.maxOutbound {
		return errors.New("maximum outbound connections reached")
	}

	cm.outbound[peerID] = struct{}{}
	return nil
}

// ReleaseInbound releases an inbound connection
func (cm *ConnectionManager) ReleaseInbound(peerID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.inbound, peerID)
}

// ReleaseOutbound releases an outbound connection
func (cm *ConnectionManager) ReleaseOutbound(peerID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.outbound, peerID)
}

// GetConnectionCounts returns current connection counts
func (cm *ConnectionManager) GetConnectionCounts() (inbound, outbound, total int) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return len(cm.inbound), len(cm.outbound), len(cm.inbound) + len(cm.outbound)
}

// Task 146: P2P Peer Discovery Security

// SecureDiscovery provides secure peer discovery
type SecureDiscovery struct {
	trustedBootstrapPeers []string
	peerStore            *ReputationStore
	ddosProtector        *DDoSProtector
	minReputation        float64
	mu                   sync.RWMutex
}

// NewSecureDiscovery creates a new secure discovery
func NewSecureDiscovery(bootstrapPeers []string, reputationStore *ReputationStore, ddosProtector *DDoSProtector) *SecureDiscovery {
	return &SecureDiscovery{
		trustedBootstrapPeers: bootstrapPeers,
		peerStore:            reputationStore,
		ddosProtector:        ddosProtector,
		minReputation:        30.0, // Minimum score to connect
	}
}

// VerifyPeer verifies if a peer is safe to connect to
func (sd *SecureDiscovery) VerifyPeer(peerID, peerAddr string) error {
	// Check blacklist
	if err := sd.ddosProtector.CheckBlacklist(peerID); err != nil {
		return err
	}

	// Check reputation
	if rep, exists := sd.peerStore.GetReputation(peerID); exists {
		if rep.Score < sd.minReputation {
			return fmt.Errorf("peer reputation too low: %.2f < %.2f", rep.Score, sd.minReputation)
		}

		// Check if blacklisted
		if rep.BlacklistedUntil != nil && time.Now().Before(*rep.BlacklistedUntil) {
			return errors.New("peer is blacklisted")
		}
	}

	return nil
}

// GetBootstrapPeers returns trusted bootstrap peers
func (sd *SecureDiscovery) GetBootstrapPeers() []string {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return append([]string{}, sd.trustedBootstrapPeers...)
}

// AddBootstrapPeer adds a trusted bootstrap peer
func (sd *SecureDiscovery) AddBootstrapPeer(peer string) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.trustedBootstrapPeers = append(sd.trustedBootstrapPeers, peer)
}

// Task 147: P2P NAT Traversal

// NATTraversal handles NAT traversal using various techniques
type NATTraversal struct {
	publicIP   string
	publicPort int
	stunServers []string
	mu         sync.RWMutex
}

// NewNATTraversal creates a new NAT traversal handler
func NewNATTraversal(stunServers []string) *NATTraversal {
	return &NATTraversal{
		stunServers: stunServers,
	}
}

// DiscoverPublicAddress discovers the public IP and port using STUN
func (nt *NATTraversal) DiscoverPublicAddress(localPort int) error {
	// In a production implementation, this would use a STUN library
	// For now, provide a simplified version

	// Try to detect public IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	nt.mu.Lock()
	nt.publicIP = localAddr.IP.String()
	nt.publicPort = localPort
	nt.mu.Unlock()

	return nil
}

// GetPublicAddress returns the discovered public address
func (nt *NATTraversal) GetPublicAddress() (string, int) {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	return nt.publicIP, nt.publicPort
}

// SupportsHolePunching checks if hole punching is possible
func (nt *NATTraversal) SupportsHolePunching() bool {
	// In production, detect NAT type and check if hole punching is possible
	// For now, return true
	return true
}

// InitiateHolePunch initiates NAT hole punching with a peer
func (nt *NATTraversal) InitiateHolePunch(ctx context.Context, peerAddr string) error {
	// Production implementation would:
	// 1. Exchange addresses via signaling server
	// 2. Send UDP packets to punch hole
	// 3. Establish direct connection

	// For now, simplified implementation
	conn, err := net.Dial("udp", peerAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Send hole-punch packet
	_, err = conn.Write([]byte("HOLE_PUNCH"))
	return err
}

// Task 148: P2P Relay Node Support

// RelayNode handles relaying messages for peers behind NAT
type RelayNode struct {
	isRelay     bool
	maxClients  int
	clients     map[string]*RelayClient
	mu          sync.RWMutex
	byteLimit   uint64 // Max bytes to relay per client
	timeLimit   time.Duration // Max relay duration per client
}

// RelayClient represents a client using the relay
type RelayClient struct {
	PeerID      string
	ConnectedAt time.Time
	BytesRelayed uint64
}

// NewRelayNode creates a new relay node
func NewRelayNode(isRelay bool, maxClients int, byteLimit uint64, timeLimit time.Duration) *RelayNode {
	return &RelayNode{
		isRelay:    isRelay,
		maxClients: maxClients,
		clients:    make(map[string]*RelayClient),
		byteLimit:  byteLimit,
		timeLimit:  timeLimit,
	}
}

// AcceptRelayClient accepts a new relay client
func (rn *RelayNode) AcceptRelayClient(peerID string) error {
	if !rn.isRelay {
		return errors.New("not a relay node")
	}

	rn.mu.Lock()
	defer rn.mu.Unlock()

	if len(rn.clients) >= rn.maxClients {
		return errors.New("relay at capacity")
	}

	if _, exists := rn.clients[peerID]; exists {
		return errors.New("peer already connected")
	}

	rn.clients[peerID] = &RelayClient{
		PeerID:      peerID,
		ConnectedAt: time.Now(),
	}

	return nil
}

// RelayMessage relays a message between peers
func (rn *RelayNode) RelayMessage(fromPeer, toPeer string, message []byte) error {
	if !rn.isRelay {
		return errors.New("not a relay node")
	}

	rn.mu.Lock()
	defer rn.mu.Unlock()

	// Check if both peers are clients
	fromClient, fromExists := rn.clients[fromPeer]
	if !fromExists {
		return errors.New("source peer not connected")
	}

	if _, toExists := rn.clients[toPeer]; !toExists {
		return errors.New("destination peer not connected")
	}

	// Check byte limit
	messageSize := uint64(len(message))
	if fromClient.BytesRelayed+messageSize > rn.byteLimit {
		return errors.New("byte limit exceeded")
	}

	// Check time limit
	if time.Since(fromClient.ConnectedAt) > rn.timeLimit {
		return errors.New("time limit exceeded")
	}

	// Update relay stats
	fromClient.BytesRelayed += messageSize

	// In production, actually relay the message
	// For now, just validate and update stats

	return nil
}

// ReleaseRelayClient releases a relay client
func (rn *RelayNode) ReleaseRelayClient(peerID string) {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	delete(rn.clients, peerID)
}

// GetRelayStats returns relay statistics
func (rn *RelayNode) GetRelayStats() (clientCount int, totalBytesRelayed uint64) {
	rn.mu.RLock()
	defer rn.mu.RUnlock()

	clientCount = len(rn.clients)
	for _, client := range rn.clients {
		totalBytesRelayed += client.BytesRelayed
	}

	return
}

// IsRelay returns if this node is a relay
func (rn *RelayNode) IsRelay() bool {
	return rn.isRelay
}
