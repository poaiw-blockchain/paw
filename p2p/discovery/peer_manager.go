package discovery

import (
	"context"
	"fmt"
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

// AddPeer adds a new peer connection
func (pm *PeerManager) AddPeer(peerID reputation.PeerID, addr *PeerAddr, outbound bool) error {
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

	// Simulate dial (in real implementation, this would use actual networking)
	// For now, we just track the attempt and result
	startTime := time.Now()

	// TODO: Implement actual TCP/networking dial here
	// This is a placeholder that would be replaced with real networking code
	success := pm.simulateDial(addr)

	latency := time.Since(startTime)

	result := &DialResult{
		PeerID:    addr.ID,
		Address:   addr.NetAddr(),
		Success:   success,
		Latency:   latency,
		Timestamp: time.Now(),
	}

	if !success {
		result.Error = fmt.Errorf("dial failed")
	}

	// Send result
	select {
	case pm.dialResults <- result:
	case <-pm.ctx.Done():
	}
}

// simulateDial simulates a dial attempt (placeholder)
// In a real implementation, this would perform actual network connection
func (pm *PeerManager) simulateDial(addr *PeerAddr) bool {
	// This is a placeholder - real implementation would:
	// 1. Open TCP connection to addr.HostPort()
	// 2. Perform handshake
	// 3. Verify peer identity
	// 4. Return success/failure
	return false // Always fails in simulation mode
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
