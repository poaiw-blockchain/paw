package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cosmossdk.io/log"
	"github.com/paw-chain/paw/p2p/discovery"
	"github.com/paw-chain/paw/p2p/reputation"
)

// Node represents a P2P network node
type Node struct {
	config  NodeConfig
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

	// Data directory
	DataDir string
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
		DataDir:              ".paw/p2p",
	}
}

// MessageHandler handles incoming messages from peers
type MessageHandler func(peerID reputation.PeerID, msg []byte) error

// NewNode creates a new P2P node
func NewNode(config NodeConfig, logger log.Logger) (*Node, error) {
	// Validate configuration
	if config.DataDir == "" {
		return nil, fmt.Errorf("data directory not specified")
	}

	ctx, cancel := context.WithCancel(context.Background())

	node := &Node{
		config:          config,
		logger:          logger,
		dataDir:         config.DataDir,
		nodeID:          reputation.PeerID(config.NodeID),
		listenAddr:      config.ListenAddress,
		ctx:             ctx,
		cancel:          cancel,
		messageHandlers: make(map[string]MessageHandler),
	}

	// Initialize reputation manager if enabled
	var repManager *reputation.Manager
	if config.EnableReputation {
		storage, err := reputation.NewFileStorage(
			reputation.DefaultFileStorageConfig(config.DataDir),
			logger,
		)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create reputation storage: %w", err)
		}

		repManager, err = reputation.NewManager(storage, config.ReputationConfig, logger)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create reputation manager: %w", err)
		}

		node.repManager = repManager
	}

	// Initialize discovery service
	discoveryConfig := discovery.DiscoveryConfig{
		Seeds:                       config.Seeds,
		BootstrapNodes:              config.BootstrapNodes,
		PersistentPeers:             config.PersistentPeers,
		PrivatePeerIDs:              config.PrivatePeerIDs,
		UnconditionalPeerIDs:        config.UnconditionalPeerIDs,
		MaxInboundPeers:             config.MaxInboundPeers,
		MaxOutboundPeers:            config.MaxOutboundPeers,
		MaxPeers:                    config.MaxPeers,
		EnablePEX:                   config.EnablePEX,
		PEXInterval:                 config.PEXInterval,
		MinOutboundPeers:            config.MinOutboundPeers,
		DialTimeout:                 config.DialTimeout,
		HandshakeTimeout:            config.HandshakeTimeout,
		PersistentPeerMaxDialPeriod: 0,
		AddressBookStrict:           config.AddressBookStrict,
		AddressBookSize:             1000,
		EnableAutoReconnect:         true,
		ReconnectInterval:           30 * time.Second,
		MaxReconnectAttempts:        10,
		PingInterval:                60 * time.Second,
		PingTimeout:                 30 * time.Second,
		InactivityTimeout:           5 * time.Minute,
		ListenAddress:               config.ListenAddress,
		ExternalAddress:             config.ExternalAddress,
	}

	discoveryService, err := discovery.NewService(discoveryConfig, config.DataDir, repManager, logger)
	if err != nil {
		cancel()
		if repManager != nil {
			repManager.Close()
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

	logger.Info("P2P node created",
		"node_id", config.NodeID,
		"listen_address", config.ListenAddress,
		"max_peers", config.MaxPeers,
		"reputation_enabled", config.EnableReputation)

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

// SendMessage sends a message to a peer
func (n *Node) SendMessage(peerID reputation.PeerID, msgType string, data []byte) error {
	// TODO: Implement actual message sending
	// This would involve:
	// 1. Looking up peer connection
	// 2. Serializing message with type
	// 3. Sending over network connection
	// 4. Handling errors and retries

	n.logger.Debug("sending message",
		"peer_id", peerID,
		"type", msgType,
		"size", len(data))

	return nil
}

// BroadcastMessage broadcasts a message to all connected peers
func (n *Node) BroadcastMessage(msgType string, data []byte) error {
	peers := n.GetPeers()

	n.logger.Debug("broadcasting message",
		"type", msgType,
		"size", len(data),
		"peer_count", len(peers))

	for _, peer := range peers {
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

// handleMessage handles incoming messages
func (n *Node) handleMessage(peerID reputation.PeerID, msgType string, data []byte) error {
	n.handlerMu.RLock()
	handler, exists := n.messageHandlers[msgType]
	n.handlerMu.RUnlock()

	if !exists {
		n.logger.Warn("no handler for message type",
			"type", msgType,
			"peer_id", peerID)
		return fmt.Errorf("no handler for message type: %s", msgType)
	}

	// Update peer activity
	n.discoveryService.UpdatePeerActivity(peerID)

	// Call handler
	if err := handler(peerID, data); err != nil {
		n.logger.Error("message handler error",
			"type", msgType,
			"peer_id", peerID,
			"error", err)
		return err
	}

	return nil
}

// GetDiscoveryService returns the discovery service (for testing/inspection)
func (n *Node) GetDiscoveryService() *discovery.Service {
	return n.discoveryService
}

// GetReputationManager returns the reputation manager (for testing/inspection)
func (n *Node) GetReputationManager() *reputation.Manager {
	return n.repManager
}
