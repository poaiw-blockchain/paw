package p2p

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"cosmossdk.io/log"

	"github.com/paw-chain/paw/p2p/discovery"
	"github.com/paw-chain/paw/p2p/reputation"
)

func toUint16Length(field string, length int) (uint16, error) {
	if length < 0 || length > math.MaxUint16 {
		return 0, fmt.Errorf("%s length %d exceeds uint16 range", field, length)
	}
	return uint16(length), nil
}

func toUint32Length(field string, length int) (uint32, error) {
	if length < 0 || length > math.MaxUint32 {
		return 0, fmt.Errorf("%s length %d exceeds uint32 range", field, length)
	}
	return uint32(length), nil
}

// Node represents a P2P network node
type Node struct {
	config  *NodeConfig
	logger  log.Logger
	dataDir string
	mu      sync.RWMutex

	// Core components
	discoveryService *discovery.Service
	repManager       *reputation.Manager

	// Node state
	started    bool
	nodeID     reputation.PeerID
	listenAddr string

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Message handlers
	messageHandlers map[string]MessageHandler
	handlerMu       sync.RWMutex

	// Event handlers
	onPeerConnected    func(reputation.PeerID)
	onPeerDisconnected func(reputation.PeerID)
	onPeerDiscovered   func(*discovery.PeerAddr)
}

// NodeConfig configures the P2P node
type NodeConfig struct {
	// Node identity
	NodeID  string
	Moniker string
	Version string

	// Network configuration
	ListenAddress   string
	ExternalAddress string
	Seeds           []string
	PersistentPeers []string
	BootstrapNodes  []string

	// Connection limits
	MaxInboundPeers  int
	MaxOutboundPeers int
	MaxPeers         int

	// Discovery settings
	EnablePEX        bool
	PEXInterval      time.Duration
	MinOutboundPeers int
	DialTimeout      time.Duration
	HandshakeTimeout time.Duration

	// Address book
	AddressBookStrict bool

	// Reputation
	EnableReputation bool
	ReputationConfig reputation.ManagerConfig

	// Security
	PrivatePeerIDs       []string
	UnconditionalPeerIDs []string

	// State sync configuration
	StateSync StateSyncConfig

	// Data directory
	DataDir string
}

// StateSyncConfig configures state synchronization
type StateSyncConfig struct {
	// Enable state sync
	Enable bool

	// Trust parameters (for light client verification)
	RPCServers  []string      // Trusted RPC servers for light client
	TrustHeight int64         // Trusted block height
	TrustHash   string        // Trusted block hash (hex)
	TrustPeriod time.Duration // Trust period for light client

	// Discovery settings
	DiscoveryTime     time.Duration // Time to discover snapshots
	MinSnapshotOffers int           // Minimum offers before selection

	// Chunk download settings
	ChunkRequestTimeout time.Duration // Timeout for chunk requests
	ChunkFetchers       uint32        // Parallel chunk downloads

	// Snapshot settings
	SnapshotInterval   uint64 // Blocks between snapshots
	SnapshotKeepRecent uint32 // Number of recent snapshots to keep
	SnapshotDir        string // Directory for snapshots (default: <DataDir>/snapshots)
}

// DefaultNodeConfig returns default node configuration
func DefaultNodeConfig() NodeConfig {
	return NodeConfig{
		Moniker:              "paw-node",
		Version:              "1.0.0",
		ListenAddress:        "tcp://0.0.0.0:26656",
		ExternalAddress:      "",
		Seeds:                []string{},
		PersistentPeers:      []string{},
		BootstrapNodes:       []string{},
		MaxInboundPeers:      50,
		MaxOutboundPeers:     50,
		MaxPeers:             100,
		EnablePEX:            true,
		PEXInterval:          30 * time.Second,
		MinOutboundPeers:     10,
		DialTimeout:          10 * time.Second,
		HandshakeTimeout:     20 * time.Second,
		AddressBookStrict:    true,
		EnableReputation:     true,
		ReputationConfig:     reputation.DefaultManagerConfig(),
		PrivatePeerIDs:       []string{},
		UnconditionalPeerIDs: []string{},
		StateSync:            DefaultStateSyncConfig(),
		DataDir:              ".paw/p2p",
	}
}

// DefaultStateSyncConfig returns default state sync configuration
func DefaultStateSyncConfig() StateSyncConfig {
	return StateSyncConfig{
		Enable:              false, // Disabled by default
		RPCServers:          []string{},
		TrustHeight:         0,
		TrustHash:           "",
		TrustPeriod:         7 * 24 * time.Hour, // 7 days
		DiscoveryTime:       10 * time.Second,
		MinSnapshotOffers:   3,
		ChunkRequestTimeout: 30 * time.Second,
		ChunkFetchers:       4,
		SnapshotInterval:    1000, // Every 1000 blocks
		SnapshotKeepRecent:  10,
		SnapshotDir:         "", // Will be set to <DataDir>/snapshots
	}
}

// MessageHandler handles incoming messages from peers
type MessageHandler func(peerID reputation.PeerID, msg []byte) error

// NewNode creates a new P2P node
func NewNode(config *NodeConfig, logger log.Logger) (*Node, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.DataDir == "" {
		return nil, fmt.Errorf("data directory not specified")
	}

	cfg := *config

	ctx, cancel := context.WithCancel(context.Background())

	node := &Node{
		config:          &cfg,
		logger:          logger,
		dataDir:         cfg.DataDir,
		nodeID:          reputation.PeerID(cfg.NodeID),
		listenAddr:      cfg.ListenAddress,
		ctx:             ctx,
		cancel:          cancel,
		messageHandlers: make(map[string]MessageHandler),
	}

	// Initialize reputation manager if enabled
	var repManager *reputation.Manager
	if cfg.EnableReputation {
		storage, err := reputation.NewFileStorage(
			reputation.DefaultFileStorageConfig(cfg.DataDir),
			logger,
		)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create reputation storage: %w", err)
		}

		repManager, err = reputation.NewManager(storage, &cfg.ReputationConfig, logger)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create reputation manager: %w", err)
		}

		node.repManager = repManager
	}

	// Initialize discovery service
	discoveryConfig := discovery.DiscoveryConfig{
		Seeds:                       cfg.Seeds,
		BootstrapNodes:              cfg.BootstrapNodes,
		PersistentPeers:             cfg.PersistentPeers,
		PrivatePeerIDs:              cfg.PrivatePeerIDs,
		UnconditionalPeerIDs:        cfg.UnconditionalPeerIDs,
		MaxInboundPeers:             cfg.MaxInboundPeers,
		MaxOutboundPeers:            cfg.MaxOutboundPeers,
		MaxPeers:                    cfg.MaxPeers,
		EnablePEX:                   cfg.EnablePEX,
		PEXInterval:                 cfg.PEXInterval,
		MinOutboundPeers:            cfg.MinOutboundPeers,
		DialTimeout:                 cfg.DialTimeout,
		HandshakeTimeout:            cfg.HandshakeTimeout,
		PersistentPeerMaxDialPeriod: 0,
		AddressBookStrict:           cfg.AddressBookStrict,
		AddressBookSize:             1000,
		EnableAutoReconnect:         true,
		ReconnectInterval:           30 * time.Second,
		MaxReconnectAttempts:        10,
		PingInterval:                60 * time.Second,
		PingTimeout:                 30 * time.Second,
		InactivityTimeout:           5 * time.Minute,
		ListenAddress:               cfg.ListenAddress,
		ExternalAddress:             cfg.ExternalAddress,
	}

	discoveryService, err := discovery.NewService(&discoveryConfig, cfg.DataDir, repManager, logger)
	if err != nil {
		cancel()
		if repManager != nil {
			if closeErr := repManager.Close(); closeErr != nil {
				logger.Error("failed to close reputation manager", "error", closeErr)
			}
		}
		return nil, fmt.Errorf("failed to create discovery service: %w", err)
	}

	node.discoveryService = discoveryService

	// Set up event handlers
	discoveryService.SetEventHandlers(
		node.handlePeerDiscovered,
		node.handlePeerConnected,
		node.handlePeerDisconnected,
	)

	// Wire messaging between discovery and the node
	discoveryService.SetMessageSender(node.SendMessage)
	discoveryService.GetPeerManager().SetMessageHandler(node.handlePeerMessage)
	node.RegisterMessageHandler(discovery.PEXMessageType, node.handlePEXMessage)

	logger.Info("P2P node created",
		"node_id", cfg.NodeID,
		"listen_address", cfg.ListenAddress,
		"max_peers", cfg.MaxPeers,
		"reputation_enabled", cfg.EnableReputation)

	return node, nil
}

// Start starts the P2P node
func (n *Node) Start() error {
	n.mu.Lock()
	if n.started {
		n.mu.Unlock()
		return fmt.Errorf("node already started")
	}
	n.started = true
	n.mu.Unlock()

	n.logger.Info("starting P2P node", "node_id", n.nodeID)

	// Start discovery service
	if err := n.discoveryService.Start(); err != nil {
		n.mu.Lock()
		n.started = false
		n.mu.Unlock()
		return fmt.Errorf("failed to start discovery service: %w", err)
	}

	// Start background tasks
	n.wg.Add(1)
	go n.mainLoop()

	n.logger.Info("P2P node started successfully")
	return nil
}

// Stop stops the P2P node
func (n *Node) Stop() error {
	n.mu.Lock()
	if !n.started {
		n.mu.Unlock()
		return nil
	}
	n.mu.Unlock()

	n.logger.Info("stopping P2P node")

	// Cancel context
	n.cancel()

	// Wait for background tasks
	n.wg.Wait()

	// Stop discovery service
	if err := n.discoveryService.Stop(); err != nil {
		n.logger.Error("failed to stop discovery service", "error", err)
	}

	// Close reputation manager
	if n.repManager != nil {
		if err := n.repManager.Close(); err != nil {
			n.logger.Error("failed to close reputation manager", "error", err)
		}
	}

	n.mu.Lock()
	n.started = false
	n.mu.Unlock()

	n.logger.Info("P2P node stopped")
	return nil
}

// GetNodeID returns the node's peer ID
func (n *Node) GetNodeID() reputation.PeerID {
	return n.nodeID
}

// GetPeers returns all connected peers
func (n *Node) GetPeers() []discovery.PeerInfo {
	return n.discoveryService.GetPeers()
}

// GetPeerCount returns the number of connected peers
func (n *Node) GetPeerCount() (inbound, outbound int) {
	return n.discoveryService.GetPeerCount()
}

// HasPeer checks if a peer is connected
func (n *Node) HasPeer(peerID reputation.PeerID) bool {
	return n.discoveryService.HasPeer(peerID)
}

// P2P message size limits to prevent DoS attacks
const (
	MaxP2PMessageSize = 10 * 1024 * 1024 // 10MB for general messages
	MaxBlockSize      = 21 * 1024 * 1024 // 21MB for blocks (aligned with Cosmos SDK)
)

// handlePeerMessage dispatches inbound peer messages to registered handlers.
// Security: Validates message size before processing to prevent DoS attacks.
func (n *Node) handlePeerMessage(peerID reputation.PeerID, msgType string, data []byte) {
	// SECURITY FIX: Validate message size before processing
	// This prevents memory exhaustion DoS attacks via oversized messages
	maxSize := MaxP2PMessageSize

	// Allow larger messages for block propagation
	if msgType == "block" || msgType == "block_announce" || msgType == "block_response" {
		maxSize = MaxBlockSize
	}

	if len(data) > maxSize {
		n.logger.Warn("rejecting oversized message - potential DoS attack",
			"peer_id", peerID,
			"type", msgType,
			"size", len(data),
			"max_allowed", maxSize)

		// Report misbehavior to reputation system
		if n.repManager != nil {
			event := reputation.PeerEvent{
				PeerID:    peerID,
				EventType: reputation.EventTypeOversizedMessage,
				Timestamp: time.Now(),
				Data: reputation.EventData{
					MessageSize:   int64(len(data)),
					ViolationType: "oversized_message",
					Details:       fmt.Sprintf("message size %d exceeds max %d for type %s", len(data), maxSize, msgType),
				},
			}
			if err := n.repManager.RecordEvent(&event); err != nil {
				n.logger.Error("failed to record oversized message event", "peer_id", peerID, "error", err)
			}
		}

		// Disconnect peer for repeated violations
		n.ReportPeerMisbehavior(peerID, fmt.Sprintf("oversized message: %d bytes (max %d)", len(data), maxSize))
		return
	}

	n.handlerMu.RLock()
	handler := n.messageHandlers[msgType]
	n.handlerMu.RUnlock()

	if handler == nil {
		n.logger.Debug("dropping message without handler",
			"peer_id", peerID,
			"type", msgType)
		return
	}

	if err := handler(peerID, data); err != nil {
		n.logger.Error("message handler failed",
			"peer_id", peerID,
			"type", msgType,
			"error", err)
	}
}

// SendMessage sends a message to a peer
func (n *Node) SendMessage(peerID reputation.PeerID, msgType string, data []byte) error {
	// Look up peer connection
	peerMgr := n.discoveryService.GetPeerManager()
	conn, exists := peerMgr.GetPeer(peerID)
	if !exists {
		return fmt.Errorf("peer not connected: %s", peerID)
	}

	if conn.Conn == nil {
		return fmt.Errorf("peer connection is nil: %s", peerID)
	}

	// Serialize message: [4 bytes length][msgType length (2 bytes)][msgType][data]
	msgTypeBytes := []byte(msgType)
	msgTypeLen, err := toUint16Length("message type", len(msgTypeBytes))
	if err != nil {
		return err
	}

	totalPayloadLen := 2 + len(msgTypeBytes) + len(data)
	totalLen, err := toUint32Length("message payload", totalPayloadLen)
	if err != nil {
		return err
	}

	// Create message buffer
	buf := make([]byte, 4+2+len(msgTypeBytes)+len(data))

	// Write total length (4 bytes)
	buf[0] = byte(totalLen >> 24)
	buf[1] = byte(totalLen >> 16)
	buf[2] = byte(totalLen >> 8)
	buf[3] = byte(totalLen)

	// Write msgType length (2 bytes)
	buf[4] = byte(msgTypeLen >> 8)
	buf[5] = byte(msgTypeLen)

	// Write msgType
	copy(buf[6:], msgTypeBytes)

	// Write data
	copy(buf[6+len(msgTypeBytes):], data)

	// Thread-safe write with timeout
	conn.WriteMu.Lock()
	defer conn.WriteMu.Unlock()

	// Set write timeout (5 seconds)
	if err := conn.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		n.logger.Error("failed to set write deadline", "peer_id", peerID, "error", err)
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send over network connection
	written, err := conn.Conn.Write(buf)
	if err != nil {
		n.logger.Error("failed to write message",
			"peer_id", peerID,
			"type", msgType,
			"error", err)

		// Handle specific network errors
		if isNetworkError(err) {
			// Remove disconnected peer
			n.discoveryService.RemovePeer(peerID, "network write error")
		}

		return fmt.Errorf("network write failed: %w", err)
	}

	if written != len(buf) {
		return fmt.Errorf("incomplete write: %d/%d bytes", written, len(buf))
	}

	// Update peer activity and traffic
	n.discoveryService.UpdatePeerActivity(peerID)
	if written < 0 {
		return fmt.Errorf("invalid write length %d", written)
	}
	peerMgr.UpdateTraffic(peerID, uint64(written), 0)

	// Update message counter
	conn.MessagesSent++

	n.logger.Debug("message sent successfully",
		"peer_id", peerID,
		"type", msgType,
		"size", len(data),
		"total_bytes", written)

	return nil
}

// isNetworkError checks if an error is a network-related error
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	// Check for common network errors
	errStr := err.Error()
	return contains(errStr, "connection reset") ||
		contains(errStr, "broken pipe") ||
		contains(errStr, "connection refused") ||
		contains(errStr, "timeout") ||
		contains(errStr, "connection closed")
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				len(s) > len(substr)*2 && findSubstring(s, substr)))
}

// findSubstring performs simple substring search
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BroadcastMessage broadcasts a message to all connected peers
func (n *Node) BroadcastMessage(msgType string, data []byte) error {
	peers := n.GetPeers()

	n.logger.Debug("broadcasting message",
		"type", msgType,
		"size", len(data),
		"peer_count", len(peers))

	for i := range peers {
		peer := &peers[i]
		if err := n.SendMessage(peer.ID, msgType, data); err != nil {
			n.logger.Error("failed to send broadcast message",
				"peer_id", peer.ID,
				"error", err)
		}
	}

	return nil
}

// RegisterMessageHandler registers a handler for a message type
func (n *Node) RegisterMessageHandler(msgType string, handler MessageHandler) {
	n.handlerMu.Lock()
	defer n.handlerMu.Unlock()

	n.messageHandlers[msgType] = handler
	n.logger.Debug("registered message handler", "type", msgType)
}

// UnregisterMessageHandler unregisters a message handler
func (n *Node) UnregisterMessageHandler(msgType string) {
	n.handlerMu.Lock()
	defer n.handlerMu.Unlock()

	delete(n.messageHandlers, msgType)
	n.logger.Debug("unregistered message handler", "type", msgType)
}

func (n *Node) handlePEXMessage(peerID reputation.PeerID, payload []byte) error {
	return n.discoveryService.HandlePEXMessage(peerID, payload)
}

// ReportPeerMisbehavior reports peer misbehavior
func (n *Node) ReportPeerMisbehavior(peerID reputation.PeerID, reason string) {
	n.discoveryService.ReportPeerMisbehavior(peerID, reason)
}

// BanPeer bans a peer
func (n *Node) BanPeer(peerID reputation.PeerID, duration time.Duration) {
	n.discoveryService.BanPeer(peerID, duration)
}

// UnbanPeer unbans a peer
func (n *Node) UnbanPeer(peerID reputation.PeerID) {
	n.discoveryService.UnbanPeer(peerID)
}

// GetStats returns node statistics
func (n *Node) GetStats() map[string]interface{} {
	stats := n.discoveryService.GetStats()

	inbound, outbound := n.GetPeerCount()

	return map[string]interface{}{
		"node_id":        n.nodeID,
		"listen_address": n.listenAddr,
		"started":        n.started,
		"peers": map[string]interface{}{
			"total":    inbound + outbound,
			"inbound":  inbound,
			"outbound": outbound,
		},
		"discovery": map[string]interface{}{
			"total_connections":     stats.TotalConnections,
			"failed_connections":    stats.FailedConnections,
			"successful_handshakes": stats.SuccessfulHandshakes,
			"known_addresses":       stats.KnownAddresses,
			"pex_enabled":           n.config.EnablePEX,
			"pex_messages":          stats.PEXMessages,
			"pex_peers_learned":     stats.PEXPeersLearned,
		},
		"reputation_enabled": n.repManager != nil,
	}
}

// GetDiscoveryStats returns detailed discovery statistics
func (n *Node) GetDiscoveryStats() discovery.DiscoveryStats {
	return n.discoveryService.GetStats()
}

// GetReputationStats returns reputation statistics
func (n *Node) GetReputationStats() reputation.Statistics {
	if n.repManager == nil {
		return reputation.Statistics{}
	}
	return n.repManager.GetStatistics()
}

// IsBootstrapped returns whether the node is bootstrapped
func (n *Node) IsBootstrapped() bool {
	return n.discoveryService.IsBootstrapped()
}

// SetPeerEventHandlers sets peer event handlers
func (n *Node) SetPeerEventHandlers(
	onConnected func(reputation.PeerID),
	onDisconnected func(reputation.PeerID),
	onDiscovered func(*discovery.PeerAddr),
) {
	n.onPeerConnected = onConnected
	n.onPeerDisconnected = onDisconnected
	n.onPeerDiscovered = onDiscovered
}

// Internal methods

// mainLoop is the main node event loop
func (n *Node) mainLoop() {
	defer n.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			n.performMaintenance()

		case <-n.ctx.Done():
			return
		}
	}
}

// performMaintenance performs periodic maintenance
func (n *Node) performMaintenance() {
	// Log periodic status
	inbound, outbound := n.GetPeerCount()
	total := inbound + outbound

	n.logger.Debug("node status",
		"total_peers", total,
		"inbound", inbound,
		"outbound", outbound,
		"bootstrapped", n.IsBootstrapped())

	// Additional maintenance tasks could go here
	// - Check peer health
	// - Update metrics
	// - Clean up stale connections
	// etc.
}

// handlePeerDiscovered handles peer discovery events
func (n *Node) handlePeerDiscovered(addr *discovery.PeerAddr) {
	n.logger.Debug("peer discovered",
		"peer_id", addr.ID,
		"address", addr.NetAddr(),
		"source", addr.Source.String())

	if n.onPeerDiscovered != nil {
		n.onPeerDiscovered(addr)
	}
}

// handlePeerConnected handles peer connection events
func (n *Node) handlePeerConnected(peerID reputation.PeerID, outbound bool) {
	n.logger.Info("peer connected",
		"peer_id", peerID,
		"outbound", outbound)

	if n.onPeerConnected != nil {
		n.onPeerConnected(peerID)
	}
}

// handlePeerDisconnected handles peer disconnection events
func (n *Node) handlePeerDisconnected(peerID reputation.PeerID) {
	n.logger.Info("peer disconnected", "peer_id", peerID)

	if n.onPeerDisconnected != nil {
		n.onPeerDisconnected(peerID)
	}
}

// GetDiscoveryService returns the discovery service (for testing/inspection)
func (n *Node) GetDiscoveryService() *discovery.Service {
	return n.discoveryService
}

// GetReputationManager returns the reputation manager (for testing/inspection)
func (n *Node) GetReputationManager() *reputation.Manager {
	return n.repManager
}
