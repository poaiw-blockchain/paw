package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cosmossdk.io/log"
	"github.com/paw-chain/paw/p2p/reputation"
)

// Service is the main peer discovery service
type Service struct {
	config  DiscoveryConfig
	logger  log.Logger
	dataDir string
	mu      sync.RWMutex

	// Components
	addressBook  *AddressBook
	peerManager  *PeerManager
	bootstrapper *Bootstrapper
	seedCrawler  *SeedCrawler
	repManager   *reputation.Manager

	// PEX (Peer Exchange)
	pexEnabled   bool
	lastPEX      time.Time
	pexPeersSent uint64
	pexPeersRecv uint64

	// Discovery state
	started bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// Statistics
	stats DiscoveryStats

	// Event handlers
	onPeerDiscovered func(*PeerAddr)
	onPeerConnected  func(reputation.PeerID, bool)
	onPeerLost       func(reputation.PeerID)
}

// NewService creates a new peer discovery service
func NewService(
	config DiscoveryConfig,
	dataDir string,
	repManager *reputation.Manager,
	logger log.Logger,
) (*Service, error) {
	// Validate configuration
	helper := NewBootstrapHelper(logger)
	if err := helper.ValidateBootstrapConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create address book
	addressBook, err := NewAddressBook(config, dataDir+"/discovery", logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create address book: %w", err)
	}

	// Create peer manager
	peerManager := NewPeerManager(config, addressBook, repManager, logger)

	// Create bootstrapper
	bootstrapper := NewBootstrapper(config, addressBook, peerManager, logger)

	// Create seed crawler
	seedCrawler := NewSeedCrawler(addressBook, logger)

	ctx, cancel := context.WithCancel(context.Background())

	s := &Service{
		config:       config,
		logger:       logger,
		dataDir:      dataDir,
		addressBook:  addressBook,
		peerManager:  peerManager,
		bootstrapper: bootstrapper,
		seedCrawler:  seedCrawler,
		repManager:   repManager,
		pexEnabled:   config.EnablePEX,
		ctx:          ctx,
		cancel:       cancel,
		stats: DiscoveryStats{
			StartTime: time.Now(),
		},
	}

	// Set up peer manager callbacks
	peerManager.SetCallbacks(s.handlePeerConnected, s.handlePeerDisconnected)

	logger.Info("discovery service created",
		"max_peers", config.MaxPeers,
		"pex_enabled", config.EnablePEX,
		"seeds", len(config.Seeds))

	return s, nil
}

// Start starts the discovery service
func (s *Service) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("discovery service already started")
	}
	s.started = true
	s.mu.Unlock()

	s.logger.Info("starting discovery service")

	// Start background tasks
	s.wg.Add(3)
	go s.pexLoop()
	go s.discoveryLoop()
	go s.statsUpdater()

	// Perform initial bootstrap
	if err := s.bootstrapper.Bootstrap(s.ctx); err != nil {
		s.logger.Error("bootstrap failed", "error", err)
		// Don't fail startup, we'll retry in discoveryLoop
	}

	s.logger.Info("discovery service started")
	return nil
}

// Stop stops the discovery service
func (s *Service) Stop() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	s.logger.Info("stopping discovery service")

	// Cancel context
	s.cancel()

	// Wait for background tasks
	s.wg.Wait()

	// Close components
	if err := s.peerManager.Close(); err != nil {
		s.logger.Error("failed to close peer manager", "error", err)
	}

	if err := s.addressBook.Close(); err != nil {
		s.logger.Error("failed to close address book", "error", err)
	}

	s.mu.Lock()
	s.started = false
	s.mu.Unlock()

	s.logger.Info("discovery service stopped")
	return nil
}

// AddPeer adds a newly connected peer
func (s *Service) AddPeer(peerID reputation.PeerID, address string, outbound bool) error {
	// Parse address
	addr, err := ParseNetAddr(address)
	if err != nil {
		// If parsing fails, create a basic address
		addr = &PeerAddr{
			ID:        peerID,
			Address:   address,
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
		}
	}

	if outbound {
		addr.Source = PeerSourceManual
	} else {
		addr.Source = PeerSourceInbound
	}

	// Add to peer manager
	return s.peerManager.AddPeer(peerID, addr, outbound)
}

// RemovePeer removes a disconnected peer
func (s *Service) RemovePeer(peerID reputation.PeerID, reason string) {
	s.peerManager.RemovePeer(peerID, reason)
}

// GetPeers returns all connected peers
func (s *Service) GetPeers() []PeerInfo {
	return s.peerManager.GetPeerInfo()
}

// GetPeerCount returns the number of connected peers
func (s *Service) GetPeerCount() (inbound, outbound int) {
	return s.peerManager.NumPeers()
}

// HasPeer checks if a peer is connected
func (s *Service) HasPeer(peerID reputation.PeerID) bool {
	return s.peerManager.HasPeer(peerID)
}

// DiscoverPeers actively discovers new peers
func (s *Service) DiscoverPeers(ctx context.Context, count int) ([]*PeerAddr, error) {
	s.logger.Debug("discovering peers", "count", count)

	// Get addresses from address book
	filter := func(addr *PeerAddr) bool {
		return !s.peerManager.HasPeer(addr.ID) && !s.addressBook.IsBanned(addr.ID)
	}

	addrs := s.addressBook.GetBestAddresses(count, filter)

	if len(addrs) == 0 {
		// Try crawling seeds for more peers
		if len(s.config.Seeds) > 0 {
			if err := s.seedCrawler.CrawlSeeds(ctx, s.config.Seeds); err != nil {
				s.logger.Error("seed crawl failed", "error", err)
			}

			// Try again after crawl
			addrs = s.addressBook.GetBestAddresses(count, filter)
		}
	}

	s.logger.Debug("discovered peers", "count", len(addrs))
	return addrs, nil
}

// SharePeers shares peer addresses for PEX
func (s *Service) SharePeers(count int) []*PeerAddr {
	if !s.pexEnabled {
		return nil
	}

	addrs := s.addressBook.GetAddressesForSharing(count)

	s.mu.Lock()
	s.pexPeersSent += uint64(len(addrs))
	s.stats.PEXPeersShared += uint64(len(addrs))
	s.mu.Unlock()

	return addrs
}

// ReceivePeers receives peer addresses from PEX
func (s *Service) ReceivePeers(addrs []*PeerAddr) {
	if !s.pexEnabled {
		return
	}

	s.logger.Debug("received peers via PEX", "count", len(addrs))

	// Add to address book
	s.addressBook.AddAddresses(addrs)

	s.mu.Lock()
	s.pexPeersRecv += uint64(len(addrs))
	s.stats.PEXPeersLearned += uint64(len(addrs))
	s.stats.PEXMessages++
	s.mu.Unlock()
}

// UpdatePeerActivity updates peer activity timestamp
func (s *Service) UpdatePeerActivity(peerID reputation.PeerID) {
	s.peerManager.UpdateActivity(peerID)
}

// ReportPeerMisbehavior reports peer misbehavior
func (s *Service) ReportPeerMisbehavior(peerID reputation.PeerID, reason string) {
	s.logger.Info("peer misbehavior reported", "peer_id", peerID, "reason", reason)

	// Mark address as bad
	s.addressBook.MarkBad(peerID)

	// Remove peer connection
	s.peerManager.RemovePeer(peerID, reason)

	// Ban peer if using reputation manager
	if s.repManager != nil {
		if err := s.repManager.BanPeer(peerID, 24*time.Hour, reason); err != nil {
			s.logger.Error("failed to ban peer", "peer_id", peerID, "error", err)
		}
	}
}

// BanPeer bans a peer
func (s *Service) BanPeer(peerID reputation.PeerID, duration time.Duration) {
	s.addressBook.Ban(peerID, duration)
	s.peerManager.RemovePeer(peerID, "banned")

	if s.repManager != nil {
		s.repManager.BanPeer(peerID, duration, "manual ban")
	}
}

// UnbanPeer unbans a peer
func (s *Service) UnbanPeer(peerID reputation.PeerID) {
	s.addressBook.Unban(peerID)

	if s.repManager != nil {
		s.repManager.UnbanPeer(peerID)
	}
}

// GetStats returns discovery statistics
func (s *Service) GetStats() DiscoveryStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Update current stats
	inbound, outbound := s.peerManager.NumPeers()
	s.stats.InboundPeers = inbound
	s.stats.OutboundPeers = outbound
	s.stats.TotalPeers = inbound + outbound

	newAddrs, triedAddrs := s.addressBook.Size()
	s.stats.KnownAddresses = newAddrs + triedAddrs
	s.stats.GoodAddresses = triedAddrs
	s.stats.TriedAddresses = triedAddrs

	return s.stats
}

// GetAddressBookStats returns address book statistics
func (s *Service) GetAddressBookStats() map[string]interface{} {
	return s.addressBook.Stats()
}

// GetPeerManagerStats returns peer manager statistics
func (s *Service) GetPeerManagerStats() map[string]interface{} {
	return s.peerManager.Stats()
}

// GetBootstrapInfo returns bootstrap information
func (s *Service) GetBootstrapInfo() map[string]interface{} {
	return s.bootstrapper.GetBootstrapInfo()
}

// IsBootstrapped returns whether the node is bootstrapped
func (s *Service) IsBootstrapped() bool {
	return s.bootstrapper.IsBootstrapped()
}

// SetEventHandlers sets event handler callbacks
func (s *Service) SetEventHandlers(
	onDiscovered func(*PeerAddr),
	onConnected func(reputation.PeerID, bool),
	onLost func(reputation.PeerID),
) {
	s.onPeerDiscovered = onDiscovered
	s.onPeerConnected = onConnected
	s.onPeerLost = onLost
}

// Internal methods

// handlePeerConnected handles peer connection events
func (s *Service) handlePeerConnected(peerID reputation.PeerID, outbound bool) {
	s.mu.Lock()
	s.stats.TotalConnections++
	s.stats.SuccessfulHandshakes++
	s.mu.Unlock()

	s.logger.Debug("peer connected", "peer_id", peerID, "outbound", outbound)

	if s.onPeerConnected != nil {
		s.onPeerConnected(peerID, outbound)
	}
}

// handlePeerDisconnected handles peer disconnection events
func (s *Service) handlePeerDisconnected(peerID reputation.PeerID) {
	s.logger.Debug("peer disconnected", "peer_id", peerID)

	if s.onPeerLost != nil {
		s.onPeerLost(peerID)
	}
}

// pexLoop handles periodic peer exchange
func (s *Service) pexLoop() {
	defer s.wg.Done()

	if !s.pexEnabled {
		return
	}

	ticker := time.NewTicker(s.config.PEXInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performPEX()

		case <-s.ctx.Done():
			return
		}
	}
}

// performPEX performs peer exchange with connected peers
func (s *Service) performPEX() {
	peers := s.peerManager.GetAllPeers()
	if len(peers) == 0 {
		return
	}

	s.logger.Debug("performing PEX", "peer_count", len(peers))

	// Get peers to share
	addrsToShare := s.SharePeers(25)
	if len(addrsToShare) == 0 {
		return
	}

	// In a real implementation, this would send PEX messages to connected peers
	// For now, this is a placeholder showing the structure

	s.mu.Lock()
	s.lastPEX = time.Now()
	s.stats.LastPEXTime = time.Now()
	s.stats.PEXMessages++
	s.mu.Unlock()

	// TODO: Implement actual PEX protocol
	// This would involve:
	// 1. Selecting peers to exchange with
	// 2. Sending PEX request/response messages
	// 3. Processing received peer lists
	// 4. Rate limiting PEX messages
}

// discoveryLoop handles continuous peer discovery
func (s *Service) discoveryLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performDiscovery()

		case <-s.ctx.Done():
			return
		}
	}
}

// performDiscovery performs active peer discovery
func (s *Service) performDiscovery() {
	// Check if we need more peers
	inbound, outbound := s.peerManager.NumPeers()
	total := inbound + outbound

	if total >= s.config.MaxPeers {
		return // Already at capacity
	}

	needed := s.config.MaxPeers - total
	if needed > 10 {
		needed = 10 // Limit discovery batch size
	}

	// Discover new peers
	addrs, err := s.DiscoverPeers(s.ctx, needed)
	if err != nil {
		s.logger.Error("peer discovery failed", "error", err)
		return
	}

	if len(addrs) == 0 {
		// Try re-bootstrapping if we lost all peers
		if !s.bootstrapper.IsBootstrapped() || total == 0 {
			s.logger.Info("re-bootstrapping network")
			if err := s.bootstrapper.Bootstrap(s.ctx); err != nil {
				s.logger.Error("re-bootstrap failed", "error", err)
			}
		}
		return
	}

	// Queue dials for discovered peers
	for _, addr := range addrs {
		s.peerManager.DialPeer(addr)

		if s.onPeerDiscovered != nil {
			s.onPeerDiscovered(addr)
		}
	}

	s.logger.Debug("queued peer dials", "count", len(addrs))
}

// statsUpdater periodically updates statistics
func (s *Service) statsUpdater() {
	defer s.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updateStats()

		case <-s.ctx.Done():
			return
		}
	}
}

// updateStats updates internal statistics
func (s *Service) updateStats() {
	s.mu.Lock()
	defer s.mu.Unlock()

	inbound, outbound := s.peerManager.NumPeers()
	s.stats.InboundPeers = inbound
	s.stats.OutboundPeers = outbound
	s.stats.TotalPeers = inbound + outbound

	newAddrs, triedAddrs := s.addressBook.Size()
	s.stats.KnownAddresses = newAddrs + triedAddrs
	s.stats.GoodAddresses = triedAddrs
	s.stats.TriedAddresses = triedAddrs

	// Get stats from peer manager
	pmStats := s.peerManager.Stats()
	if totalConns, ok := pmStats["total_connections"].(uint64); ok {
		s.stats.TotalConnections = totalConns
	}
	if failedConns, ok := pmStats["failed_dials"].(uint64); ok {
		s.stats.FailedConnections = failedConns
	}
}

// GetAddressBook returns the address book (for testing/inspection)
func (s *Service) GetAddressBook() *AddressBook {
	return s.addressBook
}

// GetPeerManager returns the peer manager (for testing/inspection)
func (s *Service) GetPeerManager() *PeerManager {
	return s.peerManager
}

// GetBootstrapper returns the bootstrapper (for testing/inspection)
func (s *Service) GetBootstrapper() *Bootstrapper {
	return s.bootstrapper
}
