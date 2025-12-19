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

// Bootstrapper handles initial network bootstrap
type Bootstrapper struct {
	config           *DiscoveryConfig
	logger           log.Logger
	addressBook      *AddressBook
	peerManager      *PeerManager
	mu               sync.RWMutex
	seedPeers        map[reputation.PeerID]bool
	requiredPeers    int
	bootstrapTimeout time.Duration
	bootstrapped     bool
	bootstrapAttempt int
	lastAttempt      time.Time
}

// NewBootstrapper creates a new bootstrapper
func NewBootstrapper(
	config *DiscoveryConfig,
	addressBook *AddressBook,
	peerManager *PeerManager,
	logger log.Logger,
) *Bootstrapper {
	return &Bootstrapper{
		config:           config,
		logger:           logger,
		addressBook:      addressBook,
		peerManager:      peerManager,
		seedPeers:        make(map[reputation.PeerID]bool),
		requiredPeers:    3, // Minimum peers needed to consider bootstrapped
		bootstrapTimeout: 2 * time.Minute,
	}
}

// Bootstrap performs initial network bootstrap
func (b *Bootstrapper) Bootstrap(ctx context.Context) error {
	b.mu.Lock()
	if b.bootstrapped {
		b.mu.Unlock()
		b.logger.Info("already bootstrapped")
		return nil
	}
	b.bootstrapAttempt++
	b.lastAttempt = time.Now()
	attempt := b.bootstrapAttempt
	b.mu.Unlock()

	b.logger.Info("starting bootstrap", "attempt", attempt)

	// Step 1: Parse and add seed nodes
	b.addSeedNodes()

	// Step 2: Parse and add bootstrap nodes
	b.addBootstrapNodes()

	// Step 3: Parse and add persistent peers
	b.addPersistentPeers()

	// Step 4: Connect to initial peers
	if err := b.connectInitialPeers(ctx); err != nil {
		return fmt.Errorf("failed to connect initial peers: %w", err)
	}

	// Step 5: Wait for minimum connections
	if err := b.waitForConnections(ctx); err != nil {
		return fmt.Errorf("failed to establish minimum connections: %w", err)
	}

	b.mu.Lock()
	b.bootstrapped = true
	b.mu.Unlock()

	inbound, outbound := b.peerManager.NumPeers()
	b.logger.Info("bootstrap completed successfully",
		"peers", inbound+outbound,
		"attempt", attempt)

	return nil
}

// IsBootstrapped returns whether bootstrap is complete
func (b *Bootstrapper) IsBootstrapped() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.bootstrapped
}

// ResetBootstrap resets bootstrap state (for testing or recovery)
func (b *Bootstrapper) ResetBootstrap() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.bootstrapped = false
	b.bootstrapAttempt = 0
	b.logger.Info("bootstrap state reset")
}

// GetBootstrapInfo returns bootstrap information
func (b *Bootstrapper) GetBootstrapInfo() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	inbound, outbound := b.peerManager.NumPeers()

	return map[string]interface{}{
		"bootstrapped":      b.bootstrapped,
		"bootstrap_attempt": b.bootstrapAttempt,
		"last_attempt":      b.lastAttempt,
		"connected_peers":   inbound + outbound,
		"required_peers":    b.requiredPeers,
		"seed_peers":        len(b.seedPeers),
	}
}

// Internal methods

// addSeedNodes parses and adds seed nodes to address book
func (b *Bootstrapper) addSeedNodes() {
	if len(b.config.Seeds) == 0 {
		b.logger.Info("no seed nodes configured")
		return
	}

	b.logger.Info("adding seed nodes", "count", len(b.config.Seeds))

	for _, seedAddr := range b.config.Seeds {
		addr, err := ParseNetAddr(seedAddr)
		if err != nil {
			b.logger.Error("failed to parse seed address", "address", seedAddr, "error", err)
			continue
		}

		addr.Source = PeerSourceSeed
		if err := b.addressBook.AddAddress(addr); err != nil {
			b.logger.Error("failed to add seed address", "address", seedAddr, "error", err)
			continue
		}

		b.seedPeers[addr.ID] = true
		b.logger.Debug("added seed node", "peer_id", addr.ID, "address", addr.NetAddr())
	}
}

// addBootstrapNodes parses and adds bootstrap nodes to address book
func (b *Bootstrapper) addBootstrapNodes() {
	if len(b.config.BootstrapNodes) == 0 {
		b.logger.Info("no bootstrap nodes configured")
		return
	}

	b.logger.Info("adding bootstrap nodes", "count", len(b.config.BootstrapNodes))

	for _, bootstrapAddr := range b.config.BootstrapNodes {
		addr, err := ParseNetAddr(bootstrapAddr)
		if err != nil {
			b.logger.Error("failed to parse bootstrap address", "address", bootstrapAddr, "error", err)
			continue
		}

		addr.Source = PeerSourceBootstrap
		if err := b.addressBook.AddAddress(addr); err != nil {
			b.logger.Error("failed to add bootstrap address", "address", bootstrapAddr, "error", err)
			continue
		}

		b.logger.Debug("added bootstrap node", "peer_id", addr.ID, "address", addr.NetAddr())
	}
}

// addPersistentPeers parses and adds persistent peers to address book
func (b *Bootstrapper) addPersistentPeers() {
	if len(b.config.PersistentPeers) == 0 {
		b.logger.Info("no persistent peers configured")
		return
	}

	b.logger.Info("adding persistent peers", "count", len(b.config.PersistentPeers))

	for _, persistentAddr := range b.config.PersistentPeers {
		addr, err := ParseNetAddr(persistentAddr)
		if err != nil {
			b.logger.Error("failed to parse persistent peer address", "address", persistentAddr, "error", err)
			continue
		}

		addr.Source = PeerSourcePersistent
		if err := b.addressBook.AddAddress(addr); err != nil {
			b.logger.Error("failed to add persistent peer address", "address", persistentAddr, "error", err)
			continue
		}

		b.logger.Debug("added persistent peer", "peer_id", addr.ID, "address", addr.NetAddr())
	}
}

// connectInitialPeers connects to initial set of peers
func (b *Bootstrapper) connectInitialPeers(ctx context.Context) error {
	b.logger.Info("connecting to initial peers")

	// Get addresses to connect to
	// Priority: Bootstrap > Seeds > Persistent > Others
	filter := func(addr *PeerAddr) bool {
		return !b.peerManager.HasPeer(addr.ID) && !b.addressBook.IsBanned(addr.ID)
	}

	// Get best addresses
	numToConnect := b.requiredPeers * 3 // Try to connect to 3x required peers
	addrs := b.addressBook.GetBestAddresses(numToConnect, filter)

	if len(addrs) == 0 {
		return fmt.Errorf("no addresses available for connection")
	}

	b.logger.Info("attempting to connect to peers", "count", len(addrs))

	// Queue dials
	for _, addr := range addrs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			b.peerManager.DialPeer(addr)
		}
	}

	return nil
}

// waitForConnections waits for minimum connections to be established
func (b *Bootstrapper) waitForConnections(ctx context.Context) error {
	b.logger.Info("waiting for minimum connections", "required", b.requiredPeers)

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, b.bootstrapTimeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			inbound, outbound := b.peerManager.NumPeers()
			total := inbound + outbound
			if total < b.requiredPeers {
				return fmt.Errorf("bootstrap timeout: only %d/%d peers connected", total, b.requiredPeers)
			}
			return nil

		case <-ticker.C:
			inbound, outbound := b.peerManager.NumPeers()
			total := inbound + outbound

			b.logger.Debug("bootstrap progress",
				"connected", total,
				"required", b.requiredPeers,
				"inbound", inbound,
				"outbound", outbound)

			if total >= b.requiredPeers {
				b.logger.Info("minimum connections established",
					"total", total,
					"inbound", inbound,
					"outbound", outbound)
				return nil
			}

			// If we're not making progress, try connecting to more peers
			if total < b.requiredPeers {
				b.connectMorePeers(ctx)
			}
		}
	}
}

// connectMorePeers attempts to connect to additional peers
func (b *Bootstrapper) connectMorePeers(ctx context.Context) {
	filter := func(addr *PeerAddr) bool {
		return !b.peerManager.HasPeer(addr.ID) && !b.addressBook.IsBanned(addr.ID)
	}

	// Get a few more addresses to try
	addrs := b.addressBook.GetRandomAddresses(5, filter)

	for _, addr := range addrs {
		select {
		case <-ctx.Done():
			return
		default:
			b.peerManager.DialPeer(addr)
		}
	}
}

// SeedCrawler handles crawling seed nodes for more peers
type SeedCrawler struct {
	logger      log.Logger
	addressBook *AddressBook
	mu          sync.Mutex

	// Crawl state
	lastCrawl  time.Time
	crawlCount int
	peersFound int
}

// NewSeedCrawler creates a new seed crawler
func NewSeedCrawler(addressBook *AddressBook, logger log.Logger) *SeedCrawler {
	return &SeedCrawler{
		logger:      logger,
		addressBook: addressBook,
	}
}

// CrawlSeeds crawls seed nodes to discover more peers
func (sc *SeedCrawler) CrawlSeeds(ctx context.Context, seedAddrs []string) error {
	sc.mu.Lock()
	sc.lastCrawl = time.Now()
	sc.crawlCount++
	sc.mu.Unlock()

	sc.logger.Info("crawling seed nodes", "count", len(seedAddrs))

	for _, seedAddr := range seedAddrs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := sc.crawlSeed(ctx, seedAddr); err != nil {
				sc.logger.Error("failed to crawl seed", "address", seedAddr, "error", err)
				continue
			}
		}
	}

	sc.logger.Info("seed crawl completed", "peers_found", sc.peersFound)
	return nil
}

// crawlSeed crawls a single seed node
func (sc *SeedCrawler) crawlSeed(ctx context.Context, seedAddr string) error {
	addr, err := ParseNetAddr(seedAddr)
	if err != nil {
		return fmt.Errorf("failed to parse seed address: %w", err)
	}

	sc.logger.Debug("crawling seed", "address", addr.NetAddr())

	// Create connection timeout context (5 seconds)
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Connect to seed node via TCP
	var dialer net.Dialer
	conn, err := dialer.DialContext(dialCtx, "tcp", addr.HostPort())
	if err != nil {
		return fmt.Errorf("failed to connect to seed: %w", err)
	}
	defer conn.Close()

	sc.logger.Debug("connected to seed", "address", addr.HostPort())

	// Set read timeout (10 seconds)
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Send peer discovery request
	// Protocol: [1 byte: message type = 0x01 for peer request]
	requestMsg := []byte{0x01}
	if _, err := conn.Write(requestMsg); err != nil {
		return fmt.Errorf("failed to send peer request: %w", err)
	}

	sc.logger.Debug("sent peer discovery request")

	// Read response header: [2 bytes: peer count]
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return fmt.Errorf("failed to read response header: %w", err)
	}

	peerCount := uint16(header[0])<<8 | uint16(header[1])
	sc.logger.Debug("received peer count", "count", peerCount)

	// Limit peer count to prevent abuse
	if peerCount > 1000 {
		return fmt.Errorf("peer count too high: %d", peerCount)
	}

	// Read peer addresses
	peersAdded := 0
	for i := uint16(0); i < peerCount; i++ {
		// Read peer address format: [1 byte: ID length][ID][2 bytes: port][1 byte: IP length][IP]
		// Read ID length
		idLenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, idLenBuf); err != nil {
			sc.logger.Error("failed to read peer ID length", "error", err)
			continue
		}
		idLen := idLenBuf[0]

		if idLen > 128 {
			sc.logger.Warn("peer ID too long", "length", idLen)
			continue
		}

		// Read ID
		idBuf := make([]byte, idLen)
		if _, err := io.ReadFull(conn, idBuf); err != nil {
			sc.logger.Error("failed to read peer ID", "error", err)
			continue
		}

		// Read port (2 bytes)
		portBuf := make([]byte, 2)
		if _, err := io.ReadFull(conn, portBuf); err != nil {
			sc.logger.Error("failed to read peer port", "error", err)
			continue
		}
		port := uint16(portBuf[0])<<8 | uint16(portBuf[1])

		// Read IP length
		ipLenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, ipLenBuf); err != nil {
			sc.logger.Error("failed to read IP length", "error", err)
			continue
		}
		ipLen := ipLenBuf[0]

		if ipLen > 64 {
			sc.logger.Warn("IP address too long", "length", ipLen)
			continue
		}

		// Read IP
		ipBuf := make([]byte, ipLen)
		if _, err := io.ReadFull(conn, ipBuf); err != nil {
			sc.logger.Error("failed to read IP", "error", err)
			continue
		}

		// Parse and validate IP
		ipAddr := net.ParseIP(string(ipBuf))
		if ipAddr == nil {
			sc.logger.Warn("invalid IP address", "ip", string(ipBuf))
			continue
		}

		// Create peer address
		peerAddr := &PeerAddr{
			ID:        reputation.PeerID(string(idBuf)),
			Address:   ipAddr.String(),
			Port:      port,
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
			Source:    PeerSourceSeed,
		}

		// Validate peer address
		if !peerAddr.IsRoutable() {
			sc.logger.Debug("skipping non-routable peer", "address", peerAddr.NetAddr())
			continue
		}

		// Add to address book
		if err := sc.addressBook.AddAddress(peerAddr); err != nil {
			sc.logger.Debug("failed to add peer address", "address", peerAddr.NetAddr(), "error", err)
			continue
		}

		peersAdded++
		sc.logger.Debug("added peer from seed", "address", peerAddr.NetAddr())
	}

	sc.mu.Lock()
	sc.peersFound += peersAdded
	sc.mu.Unlock()

	sc.logger.Info("seed crawl completed", "seed", addr.HostPort(), "peers_added", peersAdded)
	return nil
}

// GetCrawlStats returns crawl statistics
func (sc *SeedCrawler) GetCrawlStats() map[string]interface{} {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	return map[string]interface{}{
		"last_crawl":  sc.lastCrawl,
		"crawl_count": sc.crawlCount,
		"peers_found": sc.peersFound,
	}
}

// BootstrapHelper provides utility functions for bootstrap
type BootstrapHelper struct {
	logger log.Logger
}

// NewBootstrapHelper creates a new bootstrap helper
func NewBootstrapHelper(logger log.Logger) *BootstrapHelper {
	return &BootstrapHelper{
		logger: logger,
	}
}

// ValidateBootstrapConfig validates bootstrap configuration.
func (bh *BootstrapHelper) ValidateBootstrapConfig(config *DiscoveryConfig) error {
	// Check if we have any way to discover peers
	if len(config.Seeds) == 0 &&
		len(config.BootstrapNodes) == 0 &&
		len(config.PersistentPeers) == 0 {
		bh.logger.Warn("no seeds, bootstrap nodes, or persistent peers configured - node may not connect to network")
	}

	// Validate addresses
	for _, addr := range config.Seeds {
		if _, err := ParseNetAddr(addr); err != nil {
			return fmt.Errorf("invalid seed address %s: %w", addr, err)
		}
	}

	for _, addr := range config.BootstrapNodes {
		if _, err := ParseNetAddr(addr); err != nil {
			return fmt.Errorf("invalid bootstrap address %s: %w", addr, err)
		}
	}

	for _, addr := range config.PersistentPeers {
		if _, err := ParseNetAddr(addr); err != nil {
			return fmt.Errorf("invalid persistent peer address %s: %w", addr, err)
		}
	}

	// Validate connection limits
	if config.MaxInboundPeers < 1 {
		return fmt.Errorf("max_inbound_peers must be at least 1")
	}

	if config.MaxOutboundPeers < 1 {
		return fmt.Errorf("max_outbound_peers must be at least 1")
	}

	if config.MinOutboundPeers > config.MaxOutboundPeers {
		return fmt.Errorf("min_outbound_peers cannot exceed max_outbound_peers")
	}

	bh.logger.Info("bootstrap configuration validated")
	return nil
}

// GenerateDefaultSeeds returns default seed nodes for PAW network
// Production seed node addresses with DNS and IP fallbacks
func (bh *BootstrapHelper) GenerateDefaultSeeds() []string {
	return []string{
		// DNS-based seed nodes (primary)
		"seed1.paw.network:26656",
		"seed2.paw.network:26656",
		"seed3.paw.network:26656",

		// IP-based fallback seed nodes (secondary)
		// These are example IPs - in production, use actual deployed seed node IPs
		"35.184.142.101:26656", // Primary seed node (US East)
		"34.89.156.233:26656",  // Secondary seed node (Europe West)
		"35.201.123.45:26656",  // Tertiary seed node (Asia Pacific)

		// Additional regional seed nodes for redundancy
		"seed-us-west.paw.network:26656",
		"seed-eu-central.paw.network:26656",
		"seed-asia-east.paw.network:26656",
	}
}

// GetRecommendedMinPeers returns recommended minimum peer count based on network size
func (bh *BootstrapHelper) GetRecommendedMinPeers(networkSize string) int {
	switch networkSize {
	case "small": // < 10 nodes
		return 2
	case "medium": // 10-100 nodes
		return 5
	case "large": // > 100 nodes
		return 10
	default:
		return 3
	}
}
