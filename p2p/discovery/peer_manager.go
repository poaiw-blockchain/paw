package discovery

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"cosmossdk.io/log"
	"github.com/paw-chain/paw/p2p/reputation"
)

// PeerManager manages active peer connections
type PeerManager struct {
	config      DiscoveryConfig
	logger      log.Logger
	addressBook *AddressBook
	repManager  *reputation.Manager
	mu          sync.RWMutex

	// Active connections
	peers map[reputation.PeerID]*PeerConnection

	// Connection tracking
	inboundCount  int
	outboundCount int

	// Persistent peer tracking
	persistentPeers    map[reputation.PeerID]bool
	unconditionalPeers map[reputation.PeerID]bool
	persistentAttempts map[reputation.PeerID]int
	persistentLastDial map[reputation.PeerID]time.Time

	// Dial queue
	dialQueue      chan *PeerAddr
	dialResults    chan *DialResult
	dialInFlight   map[reputation.PeerID]bool
	dialInflightMu sync.Mutex

	// Background tasks
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Statistics
	stats struct {
		totalConnections    uint64
		totalDisconnections uint64
		failedDials         uint64
		successfulDials     uint64
		rejectedInbound     uint64
	}

	// Callbacks
	onPeerConnected    func(peerID reputation.PeerID, outbound bool)
	onPeerDisconnected func(peerID reputation.PeerID)
}

// NewPeerManager creates a new peer manager
func NewPeerManager(
	config DiscoveryConfig,
	addressBook *AddressBook,
	repManager *reputation.Manager,
	logger log.Logger,
) *PeerManager {
	ctx, cancel := context.WithCancel(context.Background())

	pm := &PeerManager{
		config:             config,
		logger:             logger,
		addressBook:        addressBook,
		repManager:         repManager,
		peers:              make(map[reputation.PeerID]*PeerConnection),
		persistentPeers:    make(map[reputation.PeerID]bool),
		unconditionalPeers: make(map[reputation.PeerID]bool),
		persistentAttempts: make(map[reputation.PeerID]int),
		persistentLastDial: make(map[reputation.PeerID]time.Time),
		dialQueue:          make(chan *PeerAddr, 100),
		dialResults:        make(chan *DialResult, 100),
		dialInFlight:       make(map[reputation.PeerID]bool),
		ctx:                ctx,
		cancel:             cancel,
	}

	// Mark persistent peers
	for _, peerIDStr := range config.PersistentPeers {
		pm.persistentPeers[reputation.PeerID(peerIDStr)] = true
	}

	// Mark unconditional peers
	for _, peerIDStr := range config.UnconditionalPeerIDs {
		pm.unconditionalPeers[reputation.PeerID(peerIDStr)] = true
	}

	// Start background tasks
	pm.wg.Add(3)
	go pm.dialWorker()
	go pm.resultProcessor()
	go pm.maintenanceLoop()

	logger.Info("peer manager initialized",
		"max_inbound", config.MaxInboundPeers,
		"max_outbound", config.MaxOutboundPeers,
		"persistent_peers", len(pm.persistentPeers))

	return pm
}

// AddPeer adds a new peer connection (without network conn - for compatibility)
func (pm *PeerManager) AddPeer(peerID reputation.PeerID, addr *PeerAddr, outbound bool) error {
	return pm.addPeerWithConn(peerID, addr, nil, outbound)
}

// addPeerWithConn adds a new peer connection with network connection
func (pm *PeerManager) addPeerWithConn(peerID reputation.PeerID, addr *PeerAddr, netConn net.Conn, outbound bool) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if already connected
	if _, exists := pm.peers[peerID]; exists {
		return fmt.Errorf("peer already connected")
	}

	// Check connection limits
	if outbound {
		if pm.outboundCount >= pm.config.MaxOutboundPeers {
			return fmt.Errorf("outbound peer limit reached")
		}
	} else {
		if pm.inboundCount >= pm.config.MaxInboundPeers {
			pm.stats.rejectedInbound++
			return fmt.Errorf("inbound peer limit reached")
		}

		// Check if we should accept this inbound peer
		if pm.repManager != nil {
			shouldAccept, reason := pm.repManager.ShouldAcceptPeer(peerID, addr.Address)
			if !shouldAccept {
				pm.stats.rejectedInbound++
				return fmt.Errorf("peer rejected by reputation system: %s", reason)
			}
		}
	}

	// Create connection
	conn := &PeerConnection{
		PeerAddr:     addr,
		ConnectedAt:  time.Now(),
		Outbound:     outbound,
		Persistent:   pm.persistentPeers[peerID],
		LastActivity: time.Now(),
		Conn:         netConn,
	}

	pm.peers[peerID] = conn
	pm.stats.totalConnections++

	if outbound {
		pm.outboundCount++
	} else {
		pm.inboundCount++
	}

	// Mark address as good
	pm.addressBook.MarkGood(peerID)

	// Reset persistent peer tracking
	if pm.persistentPeers[peerID] {
		pm.persistentAttempts[peerID] = 0
	}

	pm.logger.Info("peer connected",
		"peer_id", peerID,
		"address", addr.NetAddr(),
		"outbound", outbound,
		"persistent", conn.Persistent,
		"total_peers", len(pm.peers))

	// Trigger callback
	if pm.onPeerConnected != nil {
		pm.onPeerConnected(peerID, outbound)
	}

	// Record connection event
	if pm.repManager != nil {
		pm.repManager.RecordEvent(reputation.PeerEvent{
			PeerID:    peerID,
			EventType: reputation.EventTypeConnected,
			Timestamp: time.Now(),
		})
	}

	return nil
}

// RemovePeer removes a peer connection
func (pm *PeerManager) RemovePeer(peerID reputation.PeerID, reason string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	conn, exists := pm.peers[peerID]
	if !exists {
		return
	}

	// Close network connection if exists
	if conn.Conn != nil {
		if err := conn.Conn.Close(); err != nil {
			pm.logger.Debug("error closing connection", "peer_id", peerID, "error", err)
		}
	}

	delete(pm.peers, peerID)
	pm.stats.totalDisconnections++

	if conn.Outbound {
		pm.outboundCount--
	} else {
		pm.inboundCount--
	}

	pm.logger.Info("peer disconnected",
		"peer_id", peerID,
		"reason", reason,
		"persistent", conn.Persistent,
		"total_peers", len(pm.peers))

	// Trigger callback
	if pm.onPeerDisconnected != nil {
		pm.onPeerDisconnected(peerID)
	}

	// Record disconnection event
	if pm.repManager != nil {
		pm.repManager.RecordEvent(reputation.PeerEvent{
			PeerID:    peerID,
			EventType: reputation.EventTypeDisconnected,
			Timestamp: time.Now(),
		})
	}

	// Schedule reconnect for persistent peers
	if conn.Persistent && pm.config.EnableAutoReconnect {
		pm.scheduleReconnect(peerID, conn.PeerAddr)
	}
}

// GetPeer returns a peer connection
func (pm *PeerManager) GetPeer(peerID reputation.PeerID) (*PeerConnection, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	conn, exists := pm.peers[peerID]
	return conn, exists
}

// GetAllPeers returns all active peer connections
func (pm *PeerManager) GetAllPeers() []*PeerConnection {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peers := make([]*PeerConnection, 0, len(pm.peers))
	for _, conn := range pm.peers {
		peers = append(peers, conn)
	}

	return peers
}

// GetPeerInfo returns detailed peer information
func (pm *PeerManager) GetPeerInfo() []PeerInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	info := make([]PeerInfo, 0, len(pm.peers))

	for peerID, conn := range pm.peers {
		pi := PeerInfo{
			ID:           peerID,
			Address:      conn.PeerAddr.NetAddr(),
			ConnectedAt:  conn.ConnectedAt,
			Outbound:     conn.Outbound,
			Persistent:   conn.Persistent,
			Source:       conn.PeerAddr.Source,
			LastActivity: conn.LastActivity,
			BytesSent:    conn.BytesSent,
			BytesRecv:    conn.BytesRecv,
		}

		// Add reputation score if available
		if pm.repManager != nil {
			if rep, err := pm.repManager.GetReputation(peerID); err == nil && rep != nil {
				pi.ReputationScore = rep.Score
			}
		}

		info = append(info, pi)
	}

	return info
}

// NumPeers returns the number of connected peers
func (pm *PeerManager) NumPeers() (inbound, outbound int) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.inboundCount, pm.outboundCount
}

// HasPeer checks if a peer is connected
func (pm *PeerManager) HasPeer(peerID reputation.PeerID) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	_, exists := pm.peers[peerID]
	return exists
}

// UpdateActivity updates the last activity time for a peer
func (pm *PeerManager) UpdateActivity(peerID reputation.PeerID) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if conn, exists := pm.peers[peerID]; exists {
		conn.LastActivity = time.Now()
	}
}

// UpdateTraffic updates traffic statistics for a peer
func (pm *PeerManager) UpdateTraffic(peerID reputation.PeerID, bytesSent, bytesRecv uint64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if conn, exists := pm.peers[peerID]; exists {
		conn.BytesSent += bytesSent
		conn.BytesRecv += bytesRecv
	}
}

// DialPeer queues a peer for dialing
func (pm *PeerManager) DialPeer(addr *PeerAddr) {
	pm.dialInflightMu.Lock()
	// Check if already dialing
	if pm.dialInFlight[addr.ID] {
		pm.dialInflightMu.Unlock()
		return
	}
	pm.dialInFlight[addr.ID] = true
	pm.dialInflightMu.Unlock()

	select {
	case pm.dialQueue <- addr:
		// Queued successfully
	default:
		// Queue full, mark as not in flight
		pm.dialInflightMu.Lock()
		delete(pm.dialInFlight, addr.ID)
		pm.dialInflightMu.Unlock()
	}
}

// SetCallbacks sets peer event callbacks
func (pm *PeerManager) SetCallbacks(
	onConnected func(peerID reputation.PeerID, outbound bool),
	onDisconnected func(peerID reputation.PeerID),
) {
	pm.onPeerConnected = onConnected
	pm.onPeerDisconnected = onDisconnected
}

// Stats returns peer manager statistics
func (pm *PeerManager) Stats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return map[string]interface{}{
		"total_peers":          len(pm.peers),
		"inbound_peers":        pm.inboundCount,
		"outbound_peers":       pm.outboundCount,
		"total_connections":    pm.stats.totalConnections,
		"total_disconnections": pm.stats.totalDisconnections,
		"failed_dials":         pm.stats.failedDials,
		"successful_dials":     pm.stats.successfulDials,
		"rejected_inbound":     pm.stats.rejectedInbound,
	}
}

// Close shuts down the peer manager
func (pm *PeerManager) Close() error {
	pm.logger.Info("closing peer manager")

	// Cancel context to stop workers
	pm.cancel()

	// Wait for workers to finish
	pm.wg.Wait()

	// Close channels
	close(pm.dialQueue)
	close(pm.dialResults)

	pm.logger.Info("peer manager closed")
	return nil
}

// Internal methods

// dialWorker processes dial requests
func (pm *PeerManager) dialWorker() {
	defer pm.wg.Done()

	for {
		select {
		case addr := <-pm.dialQueue:
			pm.performDial(addr)

		case <-pm.ctx.Done():
			return
		}
	}
}

// performDial attempts to dial a peer
func (pm *PeerManager) performDial(addr *PeerAddr) {
	defer func() {
		pm.dialInflightMu.Lock()
		delete(pm.dialInFlight, addr.ID)
		pm.dialInflightMu.Unlock()
	}()

	// Check if already connected
	if pm.HasPeer(addr.ID) {
		return
	}

	// Check outbound limit
	pm.mu.RLock()
	outboundFull := pm.outboundCount >= pm.config.MaxOutboundPeers
	pm.mu.RUnlock()

	if outboundFull {
		// Unless it's an unconditional peer
		if !pm.unconditionalPeers[addr.ID] {
			return
		}
	}

	pm.logger.Debug("dialing peer", "peer_id", addr.ID, "address", addr.NetAddr())

	// Mark attempt
	pm.addressBook.MarkAttempt(addr.ID)

	startTime := time.Now()

	// Perform actual TCP dial with timeout
	conn, err := pm.dialPeerTCP(addr)

	latency := time.Since(startTime)

	result := &DialResult{
		PeerID:    addr.ID,
		Address:   addr.NetAddr(),
		Success:   err == nil,
		Latency:   latency,
		Timestamp: time.Now(),
		Error:     err,
	}

	if err == nil && conn != nil {
		// Successfully dialed and handshaked
		// Add peer to connection pool
		if addErr := pm.addPeerWithConn(addr.ID, addr, conn, true); addErr != nil {
			pm.logger.Error("failed to add peer after dial", "peer_id", addr.ID, "error", addErr)
			conn.Close()
			result.Success = false
			result.Error = addErr
		} else {
			// Start message reader goroutine for this peer
			pm.wg.Add(1)
			go pm.readPeerMessages(addr.ID, conn)
		}
	}

	// Send result
	select {
	case pm.dialResults <- result:
	case <-pm.ctx.Done():
	}
}

// dialPeerTCP performs the actual TCP dial and handshake
func (pm *PeerManager) dialPeerTCP(addr *PeerAddr) (net.Conn, error) {
	// Parse and validate address
	if addr.Port == 0 {
		return nil, fmt.Errorf("invalid port: 0")
	}

	hostPort := addr.HostPort()

	// Create TCP connection with timeout (5 seconds)
	dialer := net.Dialer{
		Timeout: 5 * time.Second,
	}

	conn, err := dialer.Dial("tcp", hostPort)
	if err != nil {
		return nil, fmt.Errorf("TCP dial failed: %w", err)
	}

	pm.logger.Debug("TCP connection established", "address", hostPort)

	// Perform protocol handshake
	if err := pm.performHandshake(conn, addr); err != nil {
		conn.Close()
		return nil, fmt.Errorf("handshake failed: %w", err)
	}

	pm.logger.Debug("handshake completed", "peer_id", addr.ID)

	return conn, nil
}

// performHandshake performs the protocol handshake
func (pm *PeerManager) performHandshake(conn net.Conn, addr *PeerAddr) error {
	// Set handshake timeout (10 seconds)
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("failed to set deadline: %w", err)
	}

	// Protocol handshake format:
	// Send: [1 byte: protocol version][32 bytes: chain ID][32 bytes: node ID]
	// Receive: [1 byte: protocol version][32 bytes: chain ID][32 bytes: node ID]

	const protocolVersion = 0x01
	const handshakeSize = 1 + 32 + 32 // 65 bytes

	// Prepare handshake message
	handshake := make([]byte, handshakeSize)
	handshake[0] = protocolVersion

	// SECURITY FIX: Use actual chain ID from configuration, not listen address
	// This prevents chain ID spoofing attacks
	chainID := []byte(pm.config.ChainID)
	if pm.config.ChainID == "" {
		pm.logger.Error("chain ID not configured - security risk")
		return fmt.Errorf("chain ID not configured in handshake")
	}
	if len(chainID) > 32 {
		chainID = chainID[:32]
	}
	copy(handshake[1:33], chainID)

	// SECURITY FIX: Use node ID derived from P2P key, not listen address
	// This ensures proper peer identity verification
	nodeID := []byte(pm.config.NodeID)
	if pm.config.NodeID == "" {
		pm.logger.Error("node ID not configured - security risk")
		return fmt.Errorf("node ID not configured in handshake")
	}
	if len(nodeID) > 32 {
		nodeID = nodeID[:32]
	}
	copy(handshake[33:65], nodeID)

	// Send handshake
	if _, err := conn.Write(handshake); err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	pm.logger.Debug("sent handshake",
		"version", protocolVersion,
		"chain_id", pm.config.ChainID,
		"node_id", pm.config.NodeID)

	// Read handshake response
	response := make([]byte, handshakeSize)
	if _, err := io.ReadFull(conn, response); err != nil {
		return fmt.Errorf("failed to read handshake response: %w", err)
	}

	// Verify protocol version
	peerVersion := response[0]
	if peerVersion != protocolVersion {
		pm.logger.Warn("protocol version mismatch",
			"expected", protocolVersion,
			"received", peerVersion,
			"peer_id", addr.ID)
		return fmt.Errorf("protocol version mismatch: expected %d, got %d", protocolVersion, peerVersion)
	}

	// SECURITY FIX: Strict chain ID validation to prevent cross-chain attacks
	peerChainID := response[1:33]
	if !bytesEqual(chainID, peerChainID) {
		peerChainIDStr := string(bytesToString(peerChainID))
		pm.logger.Error("chain ID mismatch - potential spoofing attack",
			"expected_chain_id", pm.config.ChainID,
			"peer_chain_id", peerChainIDStr,
			"peer_id", addr.ID,
			"peer_address", addr.NetAddr())

		// Record security event
		if pm.repManager != nil {
			pm.repManager.RecordEvent(reputation.PeerEvent{
				PeerID:    addr.ID,
				EventType: reputation.EventTypeSecurity,
				Timestamp: time.Now(),
			})
		}

		return fmt.Errorf("chain ID mismatch: expected %s, got %s", pm.config.ChainID, peerChainIDStr)
	}

	// Extract and validate peer node ID
	peerNodeID := string(bytesToString(response[33:65]))
	if peerNodeID == "" {
		pm.logger.Warn("empty peer node ID received", "peer_id", addr.ID)
		return fmt.Errorf("invalid peer node ID: empty")
	}

	// Verify peer node ID matches expected ID from address
	if addr.ID != "" && addr.ID != reputation.PeerID(peerNodeID) {
		pm.logger.Error("peer node ID mismatch - potential identity spoofing",
			"expected", addr.ID,
			"received", peerNodeID)
		return fmt.Errorf("peer node ID mismatch: expected %s, got %s", addr.ID, peerNodeID)
	}

	pm.logger.Debug("handshake completed successfully",
		"peer_version", peerVersion,
		"peer_chain_id", string(bytesToString(peerChainID)),
		"peer_node_id", peerNodeID)

	// Clear deadline for normal operation
	if err := conn.SetDeadline(time.Time{}); err != nil {
		return fmt.Errorf("failed to clear deadline: %w", err)
	}

	return nil
}

// readPeerMessages reads messages from a peer connection
func (pm *PeerManager) readPeerMessages(peerID reputation.PeerID, conn net.Conn) {
	defer pm.wg.Done()
	defer func() {
		conn.Close()
		pm.RemovePeer(peerID, "connection closed")
	}()

	pm.logger.Debug("starting message reader", "peer_id", peerID)

	for {
		select {
		case <-pm.ctx.Done():
			return
		default:
			// Set read timeout (30 seconds)
			if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
				pm.logger.Error("failed to set read deadline", "peer_id", peerID, "error", err)
				return
			}

			// Read message header: [4 bytes: total length]
			header := make([]byte, 4)
			if _, err := io.ReadFull(conn, header); err != nil {
				if err != io.EOF {
					pm.logger.Debug("failed to read message header", "peer_id", peerID, "error", err)
				}
				return
			}

			// Parse message length
			msgLen := uint32(header[0])<<24 | uint32(header[1])<<16 | uint32(header[2])<<8 | uint32(header[3])

			// SECURITY FIX: DoS protection - penalize oversized messages
			const maxMessageSize = 10 * 1024 * 1024 // 10 MB max
			if msgLen > maxMessageSize {
				pm.logger.Error("message too large - potential DoS attack",
					"peer_id", peerID,
					"length", msgLen,
					"max_allowed", maxMessageSize)

				// Penalize peer in reputation system with specific event type
				if pm.repManager != nil {
					pm.repManager.RecordEvent(reputation.PeerEvent{
						PeerID:    peerID,
						EventType: reputation.EventTypeOversizedMessage,
						Timestamp: time.Now(),
						Data: reputation.EventData{
							MessageSize:   int64(msgLen),
							ViolationType: "oversized_message",
							Details:       fmt.Sprintf("message size %d exceeds max %d", msgLen, maxMessageSize),
						},
					})

					// Check if peer should be banned after repeated violations
					rep, err := pm.repManager.GetReputation(peerID)
					if err == nil && rep != nil {
						// Ban threshold based on reputation score and penalty points
						if rep.Score < 20 || rep.Metrics.TotalPenaltyPoints > 100 {
							banDuration := 24 * time.Hour

							// Escalate ban duration for repeat offenders
							if rep.Metrics.OversizedMessages > 5 {
								banDuration = 7 * 24 * time.Hour // 7 days
							}
							if rep.Metrics.OversizedMessages > 10 {
								banDuration = 30 * 24 * time.Hour // 30 days
							}

							pm.logger.Warn("banning peer for repeated DoS attempts",
								"peer_id", peerID,
								"reputation_score", rep.Score,
								"oversized_messages", rep.Metrics.OversizedMessages,
								"ban_duration", banDuration)

							pm.repManager.BanPeer(peerID, banDuration, "oversized message attack")
						}
					}
				}

				// Record security event for monitoring and alerting
				pm.logger.Info("security_event",
					"event_type", "oversized_message_attack",
					"peer_id", peerID,
					"message_size", msgLen,
					"max_size", maxMessageSize,
					"severity", "high")

				return
			}

			// Read full message
			msgBuf := make([]byte, msgLen)
			if _, err := io.ReadFull(conn, msgBuf); err != nil {
				pm.logger.Error("failed to read message", "peer_id", peerID, "error", err)
				return
			}

			// Update traffic statistics
			pm.UpdateTraffic(peerID, 0, uint64(4+msgLen))
			pm.UpdateActivity(peerID)

			// Parse message type and data
			if msgLen < 2 {
				pm.logger.Warn("message too short", "peer_id", peerID, "length", msgLen)
				continue
			}

			msgTypeLen := uint16(msgBuf[0])<<8 | uint16(msgBuf[1])
			if uint32(msgTypeLen) > msgLen-2 {
				pm.logger.Warn("invalid message type length", "peer_id", peerID)
				continue
			}

			msgType := string(msgBuf[2 : 2+msgTypeLen])
			msgData := msgBuf[2+msgTypeLen:]

			pm.logger.Debug("received message", "peer_id", peerID, "type", msgType, "size", len(msgData))

			// Update message counter
			pm.mu.Lock()
			if peer, exists := pm.peers[peerID]; exists {
				peer.MessagesRecv++
			}
			pm.mu.Unlock()

			// Message handling would be done by upper layers
			// For now, just log the message
		}
	}
}

// Helper functions

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func bytesToString(b []byte) []byte {
	// Trim null bytes
	for i, v := range b {
		if v == 0 {
			return b[:i]
		}
	}
	return b
}

// resultProcessor processes dial results
func (pm *PeerManager) resultProcessor() {
	defer pm.wg.Done()

	for {
		select {
		case result := <-pm.dialResults:
			pm.handleDialResult(result)

		case <-pm.ctx.Done():
			return
		}
	}
}

// handleDialResult handles a dial result
func (pm *PeerManager) handleDialResult(result *DialResult) {
	if result.Success {
		pm.mu.Lock()
		pm.stats.successfulDials++
		pm.mu.Unlock()

		pm.logger.Info("dial succeeded",
			"peer_id", result.PeerID,
			"address", result.Address,
			"latency", result.Latency)

		// Note: Actual peer addition would happen in the networking layer
		// after successful handshake, not here

	} else {
		pm.mu.Lock()
		pm.stats.failedDials++
		pm.mu.Unlock()

		pm.logger.Debug("dial failed",
			"peer_id", result.PeerID,
			"address", result.Address,
			"error", result.Error)

		// Mark as bad
		pm.addressBook.MarkBad(result.PeerID)

		// Track persistent peer attempts
		if pm.persistentPeers[result.PeerID] {
			pm.persistentAttempts[result.PeerID]++
			pm.persistentLastDial[result.PeerID] = time.Now()
		}
	}
}

// maintenanceLoop performs periodic maintenance
func (pm *PeerManager) maintenanceLoop() {
	defer pm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.performMaintenance()

		case <-pm.ctx.Done():
			return
		}
	}
}

// performMaintenance performs periodic maintenance tasks
func (pm *PeerManager) performMaintenance() {
	// Check for inactive peers
	pm.checkInactivePeers()

	// Ensure we have enough outbound connections
	pm.ensureMinOutbound()

	// Reconnect to persistent peers
	pm.reconnectPersistent()
}

// checkInactivePeers checks for and removes inactive peers
func (pm *PeerManager) checkInactivePeers() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()
	inactiveThreshold := now.Add(-pm.config.InactivityTimeout)

	for peerID, conn := range pm.peers {
		if conn.LastActivity.Before(inactiveThreshold) {
			// Skip unconditional peers
			if pm.unconditionalPeers[peerID] {
				continue
			}

			pm.logger.Info("removing inactive peer", "peer_id", peerID)
			delete(pm.peers, peerID)
			pm.stats.totalDisconnections++

			if conn.Outbound {
				pm.outboundCount--
			} else {
				pm.inboundCount--
			}
		}
	}
}

// ensureMinOutbound ensures we have minimum outbound connections
func (pm *PeerManager) ensureMinOutbound() {
	pm.mu.RLock()
	needMore := pm.outboundCount < pm.config.MinOutboundPeers
	currentCount := pm.outboundCount
	pm.mu.RUnlock()

	if !needMore {
		return
	}

	needed := pm.config.MinOutboundPeers - currentCount

	pm.logger.Debug("need more outbound peers", "current", currentCount, "needed", needed)

	// Get addresses to dial
	filter := func(addr *PeerAddr) bool {
		return !pm.HasPeer(addr.ID) && !pm.addressBook.IsBanned(addr.ID)
	}

	addrs := pm.addressBook.GetBestAddresses(needed, filter)

	// Queue dials
	for _, addr := range addrs {
		pm.DialPeer(addr)
	}
}

// reconnectPersistent attempts to reconnect to persistent peers
func (pm *PeerManager) reconnectPersistent() {
	now := time.Now()

	for peerID := range pm.persistentPeers {
		// Skip if already connected
		if pm.HasPeer(peerID) {
			continue
		}

		// Check if we should retry
		lastDial := pm.persistentLastDial[peerID]
		if !lastDial.IsZero() {
			attempts := pm.persistentAttempts[peerID]

			// Exponential backoff
			backoff := time.Duration(1<<uint(attempts)) * time.Second
			if backoff > 10*time.Minute {
				backoff = 10 * time.Minute
			}

			if now.Sub(lastDial) < backoff {
				continue // Too soon to retry
			}
		}

		// Get address
		addr, exists := pm.addressBook.GetAddress(peerID)
		if !exists {
			continue
		}

		pm.logger.Debug("reconnecting to persistent peer", "peer_id", peerID)
		pm.DialPeer(addr)
	}
}

// scheduleReconnect schedules a reconnect attempt for a peer
func (pm *PeerManager) scheduleReconnect(peerID reputation.PeerID, addr *PeerAddr) {
	// This would typically use a timer, but for simplicity we rely on
	// the maintenance loop to handle reconnection
	pm.logger.Debug("scheduled reconnect", "peer_id", peerID)
}
