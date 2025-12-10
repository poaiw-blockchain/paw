package protocol

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"cosmossdk.io/log"
	"github.com/paw-chain/paw/p2p/reputation"
)

// ProtocolConfig configures the protocol manager
type ProtocolConfig struct {
	// Network
	ChainID      string
	NodeID       string
	ListenAddr   string
	GenesisHash  []byte
	Capabilities []string

	// Protocol settings
	ProtocolVersion  uint8
	HandshakeTimeout time.Duration
	PingInterval     time.Duration
	MaxPeers         int
	MaxInboundPeers  int
	MaxOutboundPeers int

	// Gossip config
	GossipConfig GossipConfig

	// Sync config
	SyncConfig SyncConfig

	// Timeouts
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultProtocolConfig returns default protocol configuration
func DefaultProtocolConfig() ProtocolConfig {
	return ProtocolConfig{
		ProtocolVersion:  CurrentProtocolVersion,
		HandshakeTimeout: 10 * time.Second,
		PingInterval:     30 * time.Second,
		MaxPeers:         50,
		MaxInboundPeers:  25,
		MaxOutboundPeers: 25,
		GossipConfig:     DefaultGossipConfig(),
		SyncConfig:       DefaultSyncConfig(),
		ReadTimeout:      30 * time.Second,
		WriteTimeout:     30 * time.Second,
		IdleTimeout:      5 * time.Minute,
	}
}

// ProtocolManager manages the P2P protocol lifecycle
type ProtocolManager struct {
	config        ProtocolConfig
	reputationMgr *reputation.Manager
	logger        log.Logger

	// Protocol components
	gossip   *GossipProtocol
	sync     *SyncProtocol
	handlers *ProtocolHandlers

	// Peer connections
	peers         map[string]*Peer
	peersMu       sync.RWMutex
	inboundCount  int
	outboundCount int

	// Network listener
	listener   net.Listener
	listenerMu sync.Mutex

	// Protocol negotiation
	supportedVersions []uint8

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	metrics *ProtocolMetrics
}

// Peer represents a connected peer
type Peer struct {
	ID           string
	Address      string
	Conn         net.Conn
	Inbound      bool
	Version      uint8
	Capabilities []string
	GenesisHash  []byte
	BestHeight   int64
	BestHash     []byte
	ConnectedAt  time.Time
	LastSeen     time.Time
	LastPing     time.Time

	// Communication
	sendChan chan Message
	recvChan chan Message

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu sync.RWMutex
}

// ProtocolMetrics tracks protocol statistics
type ProtocolMetrics struct {
	PeersConnected    int64
	PeersDisconnected int64
	MessagesReceived  int64
	MessagesSent      int64
	BytesReceived     int64
	BytesSent         int64
	HandshakesFailed  int64
	ProtocolErrors    int64
	mu                sync.RWMutex
}

// NewProtocolManager creates a new protocol manager
func NewProtocolManager(
	config ProtocolConfig,
	reputationMgr *reputation.Manager,
	logger log.Logger,
) (*ProtocolManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	pm := &ProtocolManager{
		config:            config,
		reputationMgr:     reputationMgr,
		logger:            logger,
		peers:             make(map[string]*Peer),
		supportedVersions: []uint8{CurrentProtocolVersion},
		ctx:               ctx,
		cancel:            cancel,
		metrics:           &ProtocolMetrics{},
	}

	// Initialize protocol components
	pm.gossip = NewGossipProtocol(config.GossipConfig, reputationMgr, logger)
	pm.sync = NewSyncProtocol(config.SyncConfig, logger)
	pm.handlers = NewProtocolHandlers(reputationMgr, logger)

	// Set up handler callbacks
	pm.setupHandlerCallbacks()

	logger.Info("protocol manager created",
		"chain_id", config.ChainID,
		"node_id", config.NodeID,
		"protocol_version", config.ProtocolVersion,
	)

	return pm, nil
}

// Start starts the protocol manager
func (pm *ProtocolManager) Start() error {
	pm.logger.Info("starting protocol manager")

	// Start network listener if configured
	if pm.config.ListenAddr != "" {
		if err := pm.startListener(); err != nil {
			return fmt.Errorf("failed to start listener: %w", err)
		}
	}

	// Start background tasks
	pm.startBackgroundTasks()

	pm.logger.Info("protocol manager started")
	return nil
}

// Stop stops the protocol manager
func (pm *ProtocolManager) Stop() error {
	pm.logger.Info("stopping protocol manager")

	// Stop accepting new connections
	pm.stopListener()

	// Stop background tasks
	pm.cancel()
	pm.wg.Wait()

	// Disconnect all peers
	pm.disconnectAllPeers()

	// Stop protocol components
	pm.gossip.Stop()
	pm.sync.Stop()

	pm.logger.Info("protocol manager stopped")
	return nil
}

// ConnectPeer connects to a peer
func (pm *ProtocolManager) ConnectPeer(address string) error {
	pm.peersMu.Lock()
	if pm.outboundCount >= pm.config.MaxOutboundPeers {
		pm.peersMu.Unlock()
		return errors.New("max outbound peers reached")
	}
	pm.outboundCount++
	pm.peersMu.Unlock()

	conn, err := net.DialTimeout("tcp", address, pm.config.HandshakeTimeout)
	if err != nil {
		pm.peersMu.Lock()
		pm.outboundCount--
		pm.peersMu.Unlock()
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	peer, err := pm.setupPeer(conn, false)
	if err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			pm.logger.Error("error closing failed outbound connection", "error", closeErr)
		}
		pm.peersMu.Lock()
		pm.outboundCount--
		pm.peersMu.Unlock()
		return fmt.Errorf("failed to setup peer: %w", err)
	}

	pm.logger.Info("connected to peer",
		"peer_id", peer.ID,
		"address", address,
		"height", peer.BestHeight,
	)

	return nil
}

// DisconnectPeer disconnects a peer
func (pm *ProtocolManager) DisconnectPeer(peerID string) error {
	pm.peersMu.Lock()
	peer, exists := pm.peers[peerID]
	if !exists {
		pm.peersMu.Unlock()
		return fmt.Errorf("peer not found: %s", peerID)
	}
	delete(pm.peers, peerID)

	if peer.Inbound {
		pm.inboundCount--
	} else {
		pm.outboundCount--
	}
	pm.peersMu.Unlock()

	peer.cancel()
	peer.wg.Wait()
	if err := peer.Conn.Close(); err != nil {
		pm.logger.Error("error closing peer connection", "peer_id", peerID, "error", err)
	}

	pm.gossip.RemovePeer(peerID)
	pm.handlers.CleanupPeer(peerID)

	pm.recordPeerDisconnected()

	pm.logger.Info("peer disconnected", "peer_id", peerID)

	return nil
}

// BroadcastBlock broadcasts a block to all peers
func (pm *ProtocolManager) BroadcastBlock(height int64, hash, blockData []byte) error {
	msg := &BlockMessage{
		Height:    height,
		Hash:      hash,
		BlockData: blockData,
		Source:    pm.config.NodeID,
	}

	return pm.gossip.GossipBlock(msg)
}

// BroadcastTx broadcasts a transaction to all peers
func (pm *ProtocolManager) BroadcastTx(txHash, txData []byte) error {
	msg := &TxMessage{
		TxHash: txHash,
		TxData: txData,
		Source: pm.config.NodeID,
	}

	return pm.gossip.GossipTx(msg)
}

// BroadcastPeers broadcasts peer addresses
func (pm *ProtocolManager) BroadcastPeers(peers []PeerAddress) error {
	msg := &PeerListMessage{
		Peers: peers,
	}

	return pm.gossip.GossipPeers(msg)
}

// SyncToHeight synchronizes blockchain to target height
func (pm *ProtocolManager) SyncToHeight(targetHeight int64) error {
	return pm.sync.StartSync(targetHeight)
}

// GetPeerCount returns the number of connected peers
func (pm *ProtocolManager) GetPeerCount() int {
	pm.peersMu.RLock()
	defer pm.peersMu.RUnlock()
	return len(pm.peers)
}

// GetPeers returns information about connected peers
func (pm *ProtocolManager) GetPeers() []PeerInfo {
	pm.peersMu.RLock()
	defer pm.peersMu.RUnlock()

	peers := make([]PeerInfo, 0, len(pm.peers))
	for _, peer := range pm.peers {
		peer.mu.RLock()
		info := PeerInfo{
			ID:          peer.ID,
			Address:     peer.Address,
			Inbound:     peer.Inbound,
			Version:     peer.Version,
			BestHeight:  peer.BestHeight,
			ConnectedAt: peer.ConnectedAt,
			LastSeen:    peer.LastSeen,
		}
		peer.mu.RUnlock()
		peers = append(peers, info)
	}

	return peers
}

// PeerInfo contains information about a peer
type PeerInfo struct {
	ID          string
	Address     string
	Inbound     bool
	Version     uint8
	BestHeight  int64
	ConnectedAt time.Time
	LastSeen    time.Time
}

// Internal methods

func (pm *ProtocolManager) startListener() error {
	pm.listenerMu.Lock()
	defer pm.listenerMu.Unlock()

	listener, err := net.Listen("tcp", pm.config.ListenAddr)
	if err != nil {
		return err
	}

	pm.listener = listener

	pm.wg.Add(1)
	go pm.acceptLoop()

	pm.logger.Info("listening for peers", "address", pm.config.ListenAddr)

	return nil
}

func (pm *ProtocolManager) stopListener() {
	pm.listenerMu.Lock()
	defer pm.listenerMu.Unlock()

	if pm.listener != nil {
		if err := pm.listener.Close(); err != nil {
			pm.logger.Error("error closing listener", "error", err)
		}
		pm.listener = nil
	}
}

func (pm *ProtocolManager) acceptLoop() {
	defer pm.wg.Done()

	for {
		conn, err := pm.listener.Accept()
		if err != nil {
			select {
			case <-pm.ctx.Done():
				return
			default:
				pm.logger.Error("accept error", "error", err)
				continue
			}
		}

		// Check if we can accept more inbound peers
			pm.peersMu.Lock()
			if pm.inboundCount >= pm.config.MaxInboundPeers {
				pm.peersMu.Unlock()
				if err := conn.Close(); err != nil {
					pm.logger.Error("error closing inbound connection", "error", err)
				}
				continue
			}
		pm.inboundCount++
		pm.peersMu.Unlock()

		// Setup peer in background
		go func() {
			peer, err := pm.setupPeer(conn, true)
			if err != nil {
				pm.logger.Error("failed to setup peer", "error", err)
				if closeErr := conn.Close(); closeErr != nil {
					pm.logger.Error("error closing rejected connection", "error", closeErr)
				}
				pm.peersMu.Lock()
				pm.inboundCount--
				pm.peersMu.Unlock()
				return
			}

			pm.logger.Info("accepted peer connection",
				"peer_id", peer.ID,
				"address", peer.Address,
			)
		}()
	}
}

func (pm *ProtocolManager) setupPeer(conn net.Conn, inbound bool) (*Peer, error) {
	// Set deadline for handshake
	if err := conn.SetDeadline(time.Now().Add(pm.config.HandshakeTimeout)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to set handshake deadline: %w", err)
	}

	var handshake *HandshakeMessage
	var peerID string

	if inbound {
		// Wait for handshake
		envelope, err := ReadEnvelope(conn)
		if err != nil {
			return nil, fmt.Errorf("failed to read handshake: %w", err)
		}

		msg, err := UnmarshalMessage(envelope)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal handshake: %w", err)
		}

		var ok bool
		handshake, ok = msg.(*HandshakeMessage)
		if !ok {
			return nil, errors.New("expected handshake message")
		}

		peerID = handshake.NodeID

		// Send handshake ack
		ack := &HandshakeAckMessage{
			Accepted: true,
			NodeID:   pm.config.NodeID,
		}

		ackEnv, err := MarshalEnvelope(ack)
		if err != nil {
			return nil, err
		}

		if err := WriteEnvelope(conn, ackEnv); err != nil {
			return nil, err
		}

	} else {
		// Send handshake
		handshake = &HandshakeMessage{
			ProtocolVersion: pm.config.ProtocolVersion,
			ChainID:         pm.config.ChainID,
			NodeID:          pm.config.NodeID,
			ListenAddr:      pm.config.ListenAddr,
			Capabilities:    pm.config.Capabilities,
			GenesisHash:     pm.config.GenesisHash,
			Timestamp:       time.Now().Unix(),
		}

		envelope, err := MarshalEnvelope(handshake)
		if err != nil {
			return nil, err
		}

		if err := WriteEnvelope(conn, envelope); err != nil {
			return nil, err
		}

		// Wait for ack
		ackEnv, err := ReadEnvelope(conn)
		if err != nil {
			return nil, fmt.Errorf("failed to read handshake ack: %w", err)
		}

		ackMsg, err := UnmarshalMessage(ackEnv)
		if err != nil {
			return nil, err
		}

		ack, ok := ackMsg.(*HandshakeAckMessage)
		if !ok || !ack.Accepted {
			return nil, fmt.Errorf("handshake rejected: %s", ack.Reason)
		}

		peerID = ack.NodeID
	}

	// Validate handshake
	if handshake.ChainID != pm.config.ChainID {
		return nil, fmt.Errorf("chain ID mismatch: expected %s, got %s",
			pm.config.ChainID, handshake.ChainID)
	}

	// Check reputation
	if pm.reputationMgr != nil {
		accepted, reason := pm.reputationMgr.ShouldAcceptPeer(
			reputation.PeerID(peerID),
			conn.RemoteAddr().String(),
		)
		if !accepted {
			return nil, fmt.Errorf("peer rejected by reputation system: %s", reason)
		}
	}

	// Clear deadline
	if err := conn.SetDeadline(time.Time{}); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to clear handshake deadline: %w", err)
	}

	// Create peer
	ctx, cancel := context.WithCancel(pm.ctx)
	peer := &Peer{
		ID:           peerID,
		Address:      conn.RemoteAddr().String(),
		Conn:         conn,
		Inbound:      inbound,
		Version:      handshake.ProtocolVersion,
		Capabilities: handshake.Capabilities,
		GenesisHash:  handshake.GenesisHash,
		BestHeight:   handshake.BestHeight,
		BestHash:     handshake.BestHash,
		ConnectedAt:  time.Now(),
		LastSeen:     time.Now(),
		sendChan:     make(chan Message, 100),
		recvChan:     make(chan Message, 100),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Register peer
	pm.peersMu.Lock()
	pm.peers[peerID] = peer
	pm.peersMu.Unlock()

	// Add to gossip network
	if err := pm.gossip.AddPeer(peerID, peer.Address, !inbound); err != nil {
		pm.logger.Error("failed to add peer to gossip", "peer_id", peerID, "error", err)
		peer.cancel()
		if closeErr := peer.Conn.Close(); closeErr != nil {
			pm.logger.Error("error closing connection after gossip failure", "peer_id", peerID, "error", closeErr)
		}
		pm.peersMu.Lock()
		delete(pm.peers, peerID)
		pm.peersMu.Unlock()
		return nil, err
	}

	// Start peer workers
	peer.wg.Add(2)
	go pm.peerReadLoop(peer)
	go pm.peerWriteLoop(peer)

	pm.recordPeerConnected()

	return peer, nil
}

func (pm *ProtocolManager) peerReadLoop(peer *Peer) {
	defer peer.wg.Done()

	for {
		// Set read deadline
		if err := peer.Conn.SetReadDeadline(time.Now().Add(pm.config.ReadTimeout)); err != nil {
			pm.logger.Error("failed to set read deadline", "peer_id", peer.ID, "error", err)
			pm.safeDisconnectPeer(peer.ID, "read deadline failed")
			return
		}

		envelope, err := ReadEnvelope(peer.Conn)
		if err != nil {
			if err != io.EOF {
				pm.logger.Error("read error", "peer_id", peer.ID, "error", err)
			}
			pm.safeDisconnectPeer(peer.ID, "read failure")
			return
		}

		msg, err := UnmarshalMessage(envelope)
		if err != nil {
			pm.logger.Error("unmarshal error", "peer_id", peer.ID, "error", err)
			continue
		}

		// Update last seen
		peer.mu.Lock()
		peer.LastSeen = time.Now()
		peer.mu.Unlock()

		// Handle message
		if err := pm.handlers.HandleMessage(pm.ctx, peer.ID, msg); err != nil {
			pm.logger.Error("handler error",
				"peer_id", peer.ID,
				"message_type", msg.Type().String(),
				"error", err,
			)
		}

		pm.recordMessageReceived()
	}
}

func (pm *ProtocolManager) peerWriteLoop(peer *Peer) {
	defer peer.wg.Done()

	ticker := time.NewTicker(pm.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case msg := <-peer.sendChan:
			// Set write deadline
			if err := peer.Conn.SetWriteDeadline(time.Now().Add(pm.config.WriteTimeout)); err != nil {
				pm.logger.Error("failed to set write deadline", "peer_id", peer.ID, "error", err)
				pm.safeDisconnectPeer(peer.ID, "write deadline failed")
				return
			}

			envelope, err := MarshalEnvelope(msg)
			if err != nil {
				pm.logger.Error("marshal error", "peer_id", peer.ID, "error", err)
				continue
			}

			if err := WriteEnvelope(peer.Conn, envelope); err != nil {
				pm.logger.Error("write error", "peer_id", peer.ID, "error", err)
				pm.safeDisconnectPeer(peer.ID, "write failure")
				return
			}

			pm.recordMessageSent()

		case <-ticker.C:
			// Send ping
			peer.mu.Lock()
			peer.LastPing = time.Now()
			peer.mu.Unlock()

		case <-peer.ctx.Done():
			return
		}
	}
}

func (pm *ProtocolManager) disconnectAllPeers() {
	pm.peersMu.Lock()
	peers := make([]*Peer, 0, len(pm.peers))
	for _, peer := range pm.peers {
		peers = append(peers, peer)
	}
	pm.peers = make(map[string]*Peer)
	pm.inboundCount = 0
	pm.outboundCount = 0
	pm.peersMu.Unlock()

	for _, peer := range peers {
		peer.cancel()
		peer.wg.Wait()
		if err := peer.Conn.Close(); err != nil {
			pm.logger.Error("error closing peer during shutdown", "peer_id", peer.ID, "error", err)
		}
	}
}

func (pm *ProtocolManager) startBackgroundTasks() {
	// Peer maintenance task
	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pm.maintainPeers()
			case <-pm.ctx.Done():
				return
			}
		}
	}()
}

func (pm *ProtocolManager) maintainPeers() {
	now := time.Now()

	pm.peersMu.RLock()
	var idlePeers []string
	for id, peer := range pm.peers {
		peer.mu.RLock()
		lastSeen := peer.LastSeen
		peer.mu.RUnlock()

		if now.Sub(lastSeen) > pm.config.IdleTimeout {
			idlePeers = append(idlePeers, id)
		}
	}
	pm.peersMu.RUnlock()

	for _, id := range idlePeers {
		pm.logger.Info("disconnecting idle peer", "peer_id", id)
		pm.safeDisconnectPeer(id, "idle timeout")
	}
}

func (pm *ProtocolManager) setupHandlerCallbacks() {
	// Set up handler callbacks to integrate with other components
	pm.handlers.SetHandshakeHandler(func(peerID string, msg *HandshakeMessage) (*HandshakeAckMessage, error) {
		return &HandshakeAckMessage{
			Accepted: true,
			NodeID:   pm.config.NodeID,
		}, nil
	})
}

func (pm *ProtocolManager) safeDisconnectPeer(peerID, context string) {
	if err := pm.DisconnectPeer(peerID); err != nil {
		pm.logger.Error("failed to disconnect peer", "peer_id", peerID, "context", context, "error", err)
	}
}

// Metrics

func (pm *ProtocolManager) recordPeerConnected() {
	pm.metrics.mu.Lock()
	defer pm.metrics.mu.Unlock()
	pm.metrics.PeersConnected++
}

func (pm *ProtocolManager) recordPeerDisconnected() {
	pm.metrics.mu.Lock()
	defer pm.metrics.mu.Unlock()
	pm.metrics.PeersDisconnected++
}

func (pm *ProtocolManager) recordMessageReceived() {
	pm.metrics.mu.Lock()
	defer pm.metrics.mu.Unlock()
	pm.metrics.MessagesReceived++
}

func (pm *ProtocolManager) recordMessageSent() {
	pm.metrics.mu.Lock()
	defer pm.metrics.mu.Unlock()
	pm.metrics.MessagesSent++
}

func (pm *ProtocolManager) GetMetrics() ProtocolMetrics {
	pm.metrics.mu.RLock()
	defer pm.metrics.mu.RUnlock()
	return ProtocolMetrics{
		PeersConnected:    pm.metrics.PeersConnected,
		PeersDisconnected: pm.metrics.PeersDisconnected,
		MessagesReceived:  pm.metrics.MessagesReceived,
		MessagesSent:      pm.metrics.MessagesSent,
		BytesReceived:     pm.metrics.BytesReceived,
		BytesSent:         pm.metrics.BytesSent,
		HandshakesFailed:  pm.metrics.HandshakesFailed,
		ProtocolErrors:    pm.metrics.ProtocolErrors,
	}
}
