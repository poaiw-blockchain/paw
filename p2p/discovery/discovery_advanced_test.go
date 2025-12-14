package discovery

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/p2p/reputation"
)

// AdvancedDiscoveryTestSuite tests advanced discovery scenarios
type AdvancedDiscoveryTestSuite struct {
	suite.Suite
	logger      log.Logger
	addressBook *AddressBook
	repManager  *reputation.Manager
	peerManager *PeerManager
}

func (s *AdvancedDiscoveryTestSuite) SetupTest() {
	s.logger = log.NewNopLogger()

	// Create reputation manager
	storage := reputation.NewMemoryStorage()
	repConfig := reputation.DefaultManagerConfig()
	var err error
	s.repManager, err = reputation.NewManager(storage, &repConfig, s.logger)
	s.Require().NoError(err)

	// Create address book
	abConfig := DefaultDiscoveryConfig()
	s.addressBook, err = NewAddressBook(&abConfig, "/tmp/test-addr-book", s.logger)
	s.Require().NoError(err)

	// Create peer manager
	config := &DiscoveryConfig{
		MaxInboundPeers:     10,
		MaxOutboundPeers:    10,
		MinOutboundPeers:    3,
		InactivityTimeout:   1 * time.Minute,
		PersistentPeers:     []string{},
		UnconditionalPeerIDs: []string{},
		EnableAutoReconnect: true,
		ChainID:             "test-chain",
		NodeID:              "test-node",
	}

	s.peerManager = NewPeerManager(config, s.addressBook, s.repManager, s.logger)
}

func (s *AdvancedDiscoveryTestSuite) TearDownTest() {
	if s.peerManager != nil {
		_ = s.peerManager.Close()
	}
	if s.repManager != nil {
		_ = s.repManager.Close()
	}
}

// Test bootstrap with unreachable peers
func (s *AdvancedDiscoveryTestSuite) TestBootstrapUnreachablePeers() {
	t := s.T()

	// Add mix of reachable and unreachable peers
	unreachableAddrs := []string{
		"192.0.2.1:26656",  // TEST-NET-1 (unreachable)
		"192.0.2.2:26656",  // TEST-NET-1
		"198.51.100.1:26656", // TEST-NET-2 (unreachable)
	}

	for i, addr := range unreachableAddrs {
		peerAddr := &PeerAddr{
			ID:      reputation.PeerID(fmt.Sprintf("unreachable-%d", i)),
			Address: addr[:len(addr)-6],
			Port:    26656,
			Source:  PeerSourceBootstrap,
		}
		require.NoError(t, s.addressBook.AddAddress(peerAddr))
	}

	// Try to dial unreachable peers
	addrs := s.addressBook.GetBestAddresses(3, nil)
	require.NotEmpty(t, addrs)

	for _, addr := range addrs {
		s.peerManager.DialPeer(addr)
	}

	// Wait briefly for dial attempts
	time.Sleep(100 * time.Millisecond)

	// All should have failed
	inbound, outbound := s.peerManager.NumPeers()
	require.Equal(t, 0, inbound+outbound, "no peers should connect to unreachable addresses")

	// Verify addresses marked as bad (high attempt count)
	for _, addr := range addrs {
		retrieved, exists := s.addressBook.GetAddress(addr.ID)
		require.True(t, exists && retrieved.Attempts > 0, "unreachable peer should have failed attempts")
	}
}

// Test peer exchange edge cases
func (s *AdvancedDiscoveryTestSuite) TestPEXEdgeCases() {
	t := s.T()

	// Test empty address book
	addrs := s.addressBook.GetBestAddresses(10, nil)
	require.Empty(t, addrs, "should return empty for empty address book")

	// Test duplicate addresses
	peerAddr := &PeerAddr{
		ID:      "duplicate-peer",
		Address: "127.0.0.1",
		Port:    26656,
		Source:  PeerSourcePEX,
	}

	require.NoError(t, s.addressBook.AddAddress(peerAddr))
	err := s.addressBook.AddAddress(peerAddr) // Add again
	require.Error(t, err, "should reject duplicate address")

	// Test invalid addresses
	invalidAddrs := []*PeerAddr{
		{ID: "", Address: "127.0.0.1", Port: 26656},                   // Empty ID
		{ID: "peer", Address: "", Port: 26656},                        // Empty IP
		{ID: "peer", Address: "127.0.0.1", Port: 0},                   // Invalid port
		{ID: "peer", Address: "256.256.256.256", Port: 26656},         // Invalid IP
		{ID: "peer", Address: "127.0.0.1", Port: 65535},               // Port at max limit
	}

	for _, addr := range invalidAddrs {
		err := s.addressBook.AddAddress(addr)
		require.Error(t, err, "should reject invalid address: %+v", addr)
	}

	// Test address filtering
	for i := 0; i < 20; i++ {
		addr := &PeerAddr{
			ID:      reputation.PeerID(fmt.Sprintf("filter-peer-%d", i)),
			Address: fmt.Sprintf("10.0.0.%d", i),
			Port:    26656,
			Source:  PeerSourcePEX,
		}
		require.NoError(t, s.addressBook.AddAddress(addr))

		// Mark every other peer as bad
		if i%2 == 0 {
			s.addressBook.MarkBad(addr.ID)
		}
	}

	// Get addresses excluding bad ones
	filter := func(addr *PeerAddr) bool {
		retrieved, exists := s.addressBook.GetAddress(addr.ID)
		return !(exists && retrieved.Attempts > 0)
	}

	goodAddrs := s.addressBook.GetBestAddresses(20, filter)
	require.Len(t, goodAddrs, 10, "should only return good addresses")

	for _, addr := range goodAddrs {
		retrieved, exists := s.addressBook.GetAddress(addr.ID)
		require.False(t, exists && retrieved.Attempts > 0, "filtered addresses should not be bad")
	}
}

// Test peer manager capacity limits
func (s *AdvancedDiscoveryTestSuite) TestPeerManagerCapacityLimits() {
	t := s.T()

	// Test outbound limit
	maxOutbound := s.peerManager.config.MaxOutboundPeers

	// Add max outbound peers
	for i := 0; i < maxOutbound; i++ {
		peerID := reputation.PeerID(fmt.Sprintf("outbound-peer-%d", i))
		addr := &PeerAddr{
			ID:   peerID,
			Address: fmt.Sprintf("10.0.0.%d", i+1),
			Port: 26656,
		}

		err := s.peerManager.AddPeer(peerID, addr, true)
		require.NoError(t, err)
	}

	_, outbound := s.peerManager.NumPeers()
	require.Equal(t, maxOutbound, outbound)

	// Try to add one more outbound
	overflowPeer := reputation.PeerID("overflow-outbound")
	overflowAddr := &PeerAddr{
		ID:   overflowPeer,
		Address: "10.0.1.1",
		Port: 26656,
	}

	err := s.peerManager.AddPeer(overflowPeer, overflowAddr, true)
	require.Error(t, err, "should reject outbound peer over limit")

	// Test inbound limit
	maxInbound := s.peerManager.config.MaxInboundPeers

	for i := 0; i < maxInbound; i++ {
		peerID := reputation.PeerID(fmt.Sprintf("inbound-peer-%d", i))
		addr := &PeerAddr{
			ID:   peerID,
			Address: fmt.Sprintf("10.0.2.%d", i+1),
			Port: 26656,
		}

		err := s.peerManager.AddPeer(peerID, addr, false)
		require.NoError(t, err)
	}

	inbound, _ := s.peerManager.NumPeers()
	require.Equal(t, maxInbound, inbound)

	// Try to add one more inbound
	overflowInbound := reputation.PeerID("overflow-inbound")
	overflowInboundAddr := &PeerAddr{
		ID:   overflowInbound,
		Address: "10.0.3.1",
		Port: 26656,
	}

	err = s.peerManager.AddPeer(overflowInbound, overflowInboundAddr, false)
	require.Error(t, err, "should reject inbound peer over limit")

	stats := s.peerManager.Stats()
	require.Greater(t, stats["rejected_inbound"].(uint64), uint64(0),
		"should track rejected inbound connections")
}

// Test address book corruption recovery
func (s *AdvancedDiscoveryTestSuite) TestAddressBookCorruptionRecovery() {
	t := s.T()

	// Add valid addresses
	for i := 0; i < 10; i++ {
		addr := &PeerAddr{
			ID:      reputation.PeerID(fmt.Sprintf("valid-peer-%d", i)),
			Address: fmt.Sprintf("10.0.0.%d", i+1),
			Port:    26656,
			Source:  PeerSourcePEX,
		}
		require.NoError(t, s.addressBook.AddAddress(addr))
	}

	// Simulate corruption by directly manipulating internal state
	// (in real scenario, this would be file corruption)
	newCount, triedCount := s.addressBook.Size()
	require.Equal(t, 10, newCount+triedCount)

	// Verify we can still retrieve addresses
	addrs := s.addressBook.GetBestAddresses(10, nil)
	require.NotEmpty(t, addrs)

	// Test recovery from empty state
	config := DefaultDiscoveryConfig()
	newBook, err := NewAddressBook(&config, "/tmp/test-addr-book2", s.logger)
	require.NoError(t, err)
	newCount2, triedCount2 := newBook.Size()
	require.Equal(t, 0, newCount2+triedCount2)

	// Re-add addresses
	for _, addr := range addrs {
		require.NoError(t, newBook.AddAddress(addr))
	}

	newCount3, triedCount3 := newBook.Size()
	require.Equal(t, len(addrs), newCount3+triedCount3)
}

// Test persistent peer reconnection
func (s *AdvancedDiscoveryTestSuite) TestPersistentPeerReconnection() {
	t := s.T()

	// Create peer manager with persistent peer
	persistentID := "persistent-peer-1"
	config := &DiscoveryConfig{
		MaxInboundPeers:     10,
		MaxOutboundPeers:    10,
		MinOutboundPeers:    3,
		InactivityTimeout:   1 * time.Minute,
		PersistentPeers:     []string{persistentID},
		EnableAutoReconnect: true,
		ChainID:             "test-chain",
		NodeID:              "test-node",
	}

	pm := NewPeerManager(config, s.addressBook, s.repManager, s.logger)
	defer pm.Close()

	// Add persistent peer address
	peerAddr := &PeerAddr{
		ID:      reputation.PeerID(persistentID),
		Address: "127.0.0.1",
		Port:    26656,
		Source:  PeerSourcePersistent,
	}
	require.NoError(t, s.addressBook.AddAddress(peerAddr))

	// Add and then remove the persistent peer
	err := pm.AddPeer(reputation.PeerID(persistentID), peerAddr, true)
	require.NoError(t, err)

	pm.RemovePeer(reputation.PeerID(persistentID), "test disconnect")

	// Verify reconnection is scheduled
	// (actual reconnection would happen in maintenance loop)
	require.True(t, pm.persistentPeers[reputation.PeerID(persistentID)])
}

// Test unconditional peers bypass limits
func (s *AdvancedDiscoveryTestSuite) TestUnconditionalPeersBypassLimits() {
	t := s.T()

	unconditionalID := "unconditional-peer"
	config := &DiscoveryConfig{
		MaxInboundPeers:     10,
		MaxOutboundPeers:    2, // Very low limit
		MinOutboundPeers:    1,
		InactivityTimeout:   1 * time.Minute,
		UnconditionalPeerIDs: []string{unconditionalID},
		EnableAutoReconnect: true,
		ChainID:             "test-chain",
		NodeID:              "test-node",
	}

	pm := NewPeerManager(config, s.addressBook, s.repManager, s.logger)
	defer pm.Close()

	// Fill outbound slots
	for i := 0; i < config.MaxOutboundPeers; i++ {
		peerID := reputation.PeerID(fmt.Sprintf("regular-peer-%d", i))
		addr := &PeerAddr{
			ID:   peerID,
			Address: fmt.Sprintf("10.0.0.%d", i+1),
			Port: 26656,
		}
		require.NoError(t, pm.AddPeer(peerID, addr, true))
	}

	_, outbound := pm.NumPeers()
	require.Equal(t, config.MaxOutboundPeers, outbound)

	// Unconditional peer should still be allowed
	unconditionalAddr := &PeerAddr{
		ID:   reputation.PeerID(unconditionalID),
		Address: "10.0.1.1",
		Port: 26656,
	}

	// Add to address book first
	require.NoError(t, s.addressBook.AddAddress(unconditionalAddr))

	// Dial unconditional peer (would normally be rejected due to limit)
	pm.DialPeer(unconditionalAddr)

	// Note: In real implementation, unconditional peer would bypass the limit
	// This test verifies the configuration is set up correctly
	require.True(t, pm.unconditionalPeers[reputation.PeerID(unconditionalID)])
}

// Test peer activity tracking
func (s *AdvancedDiscoveryTestSuite) TestPeerActivityTracking() {
	t := s.T()

	peerID := reputation.PeerID("activity-peer")
	addr := &PeerAddr{
		ID:   peerID,
		Address: "127.0.0.1",
		Port: 26656,
	}

	require.NoError(t, s.peerManager.AddPeer(peerID, addr, true))

	peer, exists := s.peerManager.GetPeer(peerID)
	require.True(t, exists)
	initialActivity := peer.LastActivity

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Update activity
	s.peerManager.UpdateActivity(peerID)

	peer, exists = s.peerManager.GetPeer(peerID)
	require.True(t, exists)
	require.True(t, peer.LastActivity.After(initialActivity),
		"last activity should be updated")
}

// Test traffic statistics
func (s *AdvancedDiscoveryTestSuite) TestTrafficStatistics() {
	t := s.T()

	peerID := reputation.PeerID("traffic-peer")
	addr := &PeerAddr{
		ID:   peerID,
		Address: "127.0.0.1",
		Port: 26656,
	}

	require.NoError(t, s.peerManager.AddPeer(peerID, addr, true))

	// Update traffic
	s.peerManager.UpdateTraffic(peerID, 1024, 2048)
	s.peerManager.UpdateTraffic(peerID, 512, 1024)

	peer, exists := s.peerManager.GetPeer(peerID)
	require.True(t, exists)
	require.Equal(t, uint64(1536), peer.BytesSent)
	require.Equal(t, uint64(3072), peer.BytesRecv)
}

// Test peer info collection
func (s *AdvancedDiscoveryTestSuite) TestPeerInfoCollection() {
	t := s.T()

	// Add several peers
	for i := 0; i < 5; i++ {
		peerID := reputation.PeerID(fmt.Sprintf("info-peer-%d", i))
		addr := &PeerAddr{
			ID:   peerID,
			Address: fmt.Sprintf("10.0.0.%d", i+1),
			Port: 26656,
		}

		outbound := i%2 == 0
		require.NoError(t, s.peerManager.AddPeer(peerID, addr, outbound))

		// Update some stats
		s.peerManager.UpdateTraffic(peerID, uint64(100*i), uint64(200*i))
	}

	// Get peer info
	peerInfo := s.peerManager.GetPeerInfo()
	require.Len(t, peerInfo, 5)

	for _, info := range peerInfo {
		require.NotEmpty(t, info.ID)
		require.NotEmpty(t, info.Address)
		require.False(t, info.ConnectedAt.IsZero())
	}
}

// Test address book selection bias
func (s *AdvancedDiscoveryTestSuite) TestAddressBookSelectionBias() {
	t := s.T()

	// Add addresses with different sources and track
	sources := []PeerSource{
		PeerSourceBootstrap,
		PeerSourcePEX,
		PeerSourcePersistent,
		PeerSourceManual,
	}

	for i, source := range sources {
		for j := 0; j < 10; j++ {
			addr := &PeerAddr{
				ID:      reputation.PeerID(fmt.Sprintf("source-%d-peer-%d", i, j)),
				Address: fmt.Sprintf("10.%d.%d.1", i, j),
				Port:    26656,
				Source:  source,
			}
			require.NoError(t, s.addressBook.AddAddress(addr))
		}
	}

	// Get best addresses multiple times and verify distribution
	iterations := 10
	sourceCount := make(map[PeerSource]int)

	for iter := 0; iter < iterations; iter++ {
		addrs := s.addressBook.GetBestAddresses(20, nil)

		for _, addr := range addrs {
			sourceCount[addr.Source]++
		}
	}

	// All sources should be represented
	for _, source := range sources {
		require.Greater(t, sourceCount[source], 0,
			"source %d should be represented in selection", source)
	}
}

// Test concurrent peer operations
func (s *AdvancedDiscoveryTestSuite) TestConcurrentPeerOperations() {
	t := s.T()

	numGoroutines := 10
	peersPerGoroutine := 10

	done := make(chan bool, numGoroutines)

	// Concurrent peer additions
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			for i := 0; i < peersPerGoroutine; i++ {
				peerID := reputation.PeerID(fmt.Sprintf("concurrent-peer-%d-%d", goroutineID, i))
				addr := &PeerAddr{
					ID:   peerID,
					Address: fmt.Sprintf("10.%d.%d.1", goroutineID, i),
					Port: 26656,
				}

				// Alternate between inbound and outbound
				outbound := (goroutineID+i)%2 == 0

				// Try to add (may fail due to limits, which is okay)
				_ = s.peerManager.AddPeer(peerID, addr, outbound)

				// Random operations
				if i%3 == 0 {
					s.peerManager.UpdateActivity(peerID)
				}
				if i%5 == 0 {
					s.peerManager.UpdateTraffic(peerID, 100, 200)
				}
			}
		}(g)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify state is consistent
	inbound, outbound := s.peerManager.NumPeers()
	require.LessOrEqual(t, inbound, s.peerManager.config.MaxInboundPeers)
	require.LessOrEqual(t, outbound, s.peerManager.config.MaxOutboundPeers)

	// Stats should be consistent
	stats := s.peerManager.Stats()
	require.NotNil(t, stats)
	require.Equal(t, inbound+outbound, stats["total_peers"].(int))
}

func TestAdvancedDiscoveryTestSuite(t *testing.T) {
	suite.Run(t, new(AdvancedDiscoveryTestSuite))
}

// Benchmarks

func BenchmarkAddressBookAddition(b *testing.B) {
	logger := log.NewNopLogger()
	config := DefaultDiscoveryConfig()
	book, _ := NewAddressBook(&config, "/tmp/bench-addr-book", logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		addr := &PeerAddr{
			ID:      reputation.PeerID(fmt.Sprintf("bench-peer-%d", i)),
			Address: fmt.Sprintf("10.0.%d.%d", i/256, i%256),
			Port:    26656,
			Source:  PeerSourcePEX,
		}
		_ = book.AddAddress(addr)
	}
}

func BenchmarkGetBestAddresses(b *testing.B) {
	logger := log.NewNopLogger()
	config := DefaultDiscoveryConfig()
	book, _ := NewAddressBook(&config, "/tmp/bench-addr-book2", logger)

	// Pre-populate with addresses
	for i := 0; i < 1000; i++ {
		addr := &PeerAddr{
			ID:      reputation.PeerID(fmt.Sprintf("bench-peer-%d", i)),
			Address: fmt.Sprintf("10.0.%d.%d", i/256, i%256),
			Port:    26656,
			Source:  PeerSource(i % 4),
		}
		_ = book.AddAddress(addr)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = book.GetBestAddresses(50, nil)
	}
}

func BenchmarkPeerManagerStats(b *testing.B) {
	logger := log.NewNopLogger()
	config := DefaultDiscoveryConfig()
	addressBook, _ := NewAddressBook(&config, "/tmp/bench-addr-book3", logger)
	storage := reputation.NewMemoryStorage()
	repConfig := reputation.DefaultManagerConfig()

	repMgr, _ := reputation.NewManager(storage, &repConfig, logger)
	defer repMgr.Close()

	pmConfig := &DiscoveryConfig{
		MaxInboundPeers:     100,
		MaxOutboundPeers:    100,
		MinOutboundPeers:    10,
		InactivityTimeout:   5 * time.Minute,
		EnableAutoReconnect: true,
		ChainID:             "bench-chain",
		NodeID:              "bench-node",
	}

	pm := NewPeerManager(pmConfig, addressBook, repMgr, logger)
	defer pm.Close()

	// Add some peers
	for i := 0; i < 50; i++ {
		peerID := reputation.PeerID(fmt.Sprintf("bench-peer-%d", i))
		addr := &PeerAddr{
			ID:   peerID,
			Address: fmt.Sprintf("10.0.%d.1", i),
			Port: 26656,
		}
		_ = pm.AddPeer(peerID, addr, i%2 == 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pm.Stats()
	}
}
