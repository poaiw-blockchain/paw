package reputation

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ReputationTestSuite provides comprehensive reputation system testing
type ReputationTestSuite struct {
	suite.Suite
	storage Storage
	manager *Manager
	scorer  *Scorer
	logger  log.Logger
}

func (s *ReputationTestSuite) SetupTest() {
	s.logger = log.NewNopLogger()
	s.storage = NewMemoryStorage()

	config := DefaultManagerConfig()
	config.ScoreDecayInterval = 100 * time.Millisecond // Fast for testing

	var err error
	s.manager, err = NewManager(s.storage, &config, s.logger)
	s.Require().NoError(err)

	scoringConfig := DefaultScoringConfig()
	s.scorer = NewScorer(DefaultScoreWeights(), &scoringConfig)
}

func (s *ReputationTestSuite) TearDownTest() {
	if s.manager != nil {
		_ = s.manager.Close()
	}
}

// Test peer scoring algorithm
func (s *ReputationTestSuite) TestScoringAlgorithm() {
	t := s.T()

	// Test new peer starts with neutral score
	rep := &PeerReputation{
		PeerID:    "peer1",
		FirstSeen: time.Now(),
		LastSeen:  time.Now(),
		Metrics:   PeerMetrics{},
	}

	score := s.scorer.CalculateScore(rep)
	require.InDelta(t, 50.0, score, 5.0, "new peer should have near-neutral score")

	// Test uptime scoring
	rep.Metrics.TotalUptime = 1 * time.Hour
	rep.Metrics.ConnectionCount = 1
	rep.Metrics.DisconnectionCount = 0
	rep.FirstSeen = time.Now().Add(-2 * time.Hour)
	score = s.scorer.CalculateScore(rep)
	require.Greater(t, score, 30.0, "good uptime should increase score")

	// Test message validity scoring
	rep.Metrics.ValidMessages = 95
	rep.Metrics.InvalidMessages = 5
	rep.Metrics.TotalMessages = 100
	rep.Metrics.ValidMessageRatio = 0.95
	score = s.scorer.CalculateScore(rep)
	require.Greater(t, score, 50.0, "high message validity should increase score")

	// Test latency scoring
	rep.Metrics.AvgResponseLatency = 100 * time.Millisecond
	rep.Metrics.LatencyMeasurements = 10
	score = s.scorer.CalculateScore(rep)
	require.Greater(t, score, 60.0, "low latency should increase score")

	// Test block propagation scoring
	rep.Metrics.BlocksPropagated = 100
	rep.Metrics.FastBlockCount = 90
	rep.Metrics.AvgBlockPropagation = 500 * time.Millisecond
	score = s.scorer.CalculateScore(rep)
	require.Greater(t, score, 70.0, "fast block propagation should increase score")

	// Test violation penalties
	rep.Metrics.ProtocolViolations = 10
	score = s.scorer.CalculateScore(rep)
	require.Less(t, score, 100.0, "violations should decrease score")
}

// Test reputation decay over time
func (s *ReputationTestSuite) TestScoreDecay() {
	t := s.T()

	// Create peer with high score
	event := &PeerEvent{
		PeerID:    "decay-peer",
		EventType: EventTypeConnected,
		Timestamp: time.Now(),
	}
	require.NoError(t, s.manager.RecordEvent(event))

	// Send valid messages to build reputation
	for i := 0; i < 100; i++ {
		event = &PeerEvent{
			PeerID:    "decay-peer",
			EventType: EventTypeValidMessage,
			Timestamp: time.Now(),
			Data:      EventData{MessageSize: 100},
		}
		require.NoError(t, s.manager.RecordEvent(event))
	}

	rep, err := s.manager.GetReputation("decay-peer")
	require.NoError(t, err)
	initialScore := rep.Score

	// Wait for decay to apply
	time.Sleep(200 * time.Millisecond)

	// Modify last seen to trigger decay
	rep.LastSeen = time.Now().Add(-25 * time.Hour)
	require.NoError(t, s.storage.Save(rep))

	// Trigger decay manually
	s.manager.applyScoreDecay()

	rep, err = s.manager.GetReputation("decay-peer")
	require.NoError(t, err)
	require.Less(t, rep.Score, initialScore, "score should decay for inactive peers")
}

// Test reputation threshold enforcement
func (s *ReputationTestSuite) TestThresholdEnforcement() {
	t := s.T()

	// Create peer with low reputation
	event := &PeerEvent{
		PeerID:    "low-rep-peer",
		EventType: EventTypeConnected,
		Timestamp: time.Now(),
	}
	require.NoError(t, s.manager.RecordEvent(event))

	// Send many invalid messages to decrease reputation
	for i := 0; i < 100; i++ {
		event = &PeerEvent{
			PeerID:    "low-rep-peer",
			EventType: EventTypeInvalidMessage,
			Timestamp: time.Now(),
		}
		require.NoError(t, s.manager.RecordEvent(event))
	}

	// Check if peer should be accepted
	accepted, reason := s.manager.ShouldAcceptPeer("low-rep-peer", "127.0.0.1:26656")
	require.False(t, accepted, "peer with low reputation should be rejected")
	require.NotEmpty(t, reason)
}

// Test malicious peer detection
func (s *ReputationTestSuite) TestMaliciousPeerDetection() {
	t := s.T()

	testCases := []struct {
		name      string
		events    []EventType
		shouldBan bool
		banType   BanType
	}{
		{
			name:      "double sign attempt",
			events:    []EventType{EventTypeConnected, EventTypeDoubleSign},
			shouldBan: true,
			banType:   BanTypePermanent,
		},
		{
			name: "multiple invalid blocks",
			events: []EventType{
				EventTypeConnected,
				EventTypeInvalidBlock,
				EventTypeInvalidBlock,
				EventTypeInvalidBlock,
			},
			shouldBan: true,
			banType:   BanTypePermanent,
		},
		{
			name: "spam attempts",
			events: []EventType{
				EventTypeConnected,
				EventTypeSpam, EventTypeSpam, EventTypeSpam,
				EventTypeSpam, EventTypeSpam,
			},
			shouldBan: true,
			banType:   BanTypeTemporary,
		},
		{
			name: "oversized messages",
			events: []EventType{
				EventTypeConnected,
				EventTypeOversizedMessage,
				EventTypeOversizedMessage,
				EventTypeOversizedMessage,
			},
			shouldBan: true,
			banType:   BanTypeTemporary,
		},
		{
			name: "security events",
			events: []EventType{
				EventTypeConnected,
				EventTypeSecurity,
				EventTypeSecurity,
			},
			shouldBan: false, // Depends on overall score
			banType:   BanTypeNone,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			peerID := PeerID("malicious-" + tc.name)

			for _, eventType := range tc.events {
				event := &PeerEvent{
					PeerID:    peerID,
					EventType: eventType,
					Timestamp: time.Now(),
				}
				require.NoError(t, s.manager.RecordEvent(event))
			}

			rep, err := s.manager.GetReputation(peerID)
			require.NoError(t, err)

			shouldBan, banType, _ := s.scorer.ShouldBan(rep)
			if tc.shouldBan {
				require.True(t, shouldBan, "peer should be banned for: %s", tc.name)
				require.Equal(t, tc.banType, banType, "incorrect ban type for: %s", tc.name)
			}
		})
	}
}

// Test peer banning and unbanning
func (s *ReputationTestSuite) TestBanningAndUnbanning() {
	t := s.T()

	peerID := PeerID("ban-test-peer")

	// Ban peer temporarily
	duration := 1 * time.Hour
	err := s.manager.BanPeer(peerID, duration, "test ban")
	require.NoError(t, err)

	rep, err := s.manager.GetReputation(peerID)
	require.NoError(t, err)
	require.True(t, rep.BanStatus.IsBanned)
	require.Equal(t, BanTypeTemporary, rep.BanStatus.BanType)
	require.False(t, rep.BanStatus.BanExpires.IsZero())

	// Try to accept banned peer
	accepted, reason := s.manager.ShouldAcceptPeer(peerID, "127.0.0.1:26656")
	require.False(t, accepted)
	require.Contains(t, reason, "banned")

	// Unban peer
	err = s.manager.UnbanPeer(peerID)
	require.NoError(t, err)

	rep, err = s.manager.GetReputation(peerID)
	require.NoError(t, err)
	require.False(t, rep.BanStatus.IsBanned)

	// Test permanent ban
	err = s.manager.BanPeer(peerID, 0, "permanent test ban")
	require.NoError(t, err)

	rep, err = s.manager.GetReputation(peerID)
	require.NoError(t, err)
	require.True(t, rep.BanStatus.IsBanned)
	require.Equal(t, BanTypePermanent, rep.BanStatus.BanType)
}

// Test reputation persistence across restarts
func (s *ReputationTestSuite) TestPersistence() {
	t := s.T()

	peerID := PeerID("persist-peer")

	// Create reputation
	for i := 0; i < 10; i++ {
		event := &PeerEvent{
			PeerID:    peerID,
			EventType: EventTypeValidMessage,
			Timestamp: time.Now(),
		}
		require.NoError(t, s.manager.RecordEvent(event))
	}

	// Get initial reputation
	rep1, err := s.manager.GetReputation(peerID)
	require.NoError(t, err)
	require.NotNil(t, rep1)

	// Close and recreate manager (simulating restart)
	require.NoError(t, s.manager.Close())

	config := DefaultManagerConfig()
	s.manager, err = NewManager(s.storage, &config, s.logger)
	require.NoError(t, err)

	// Verify reputation persisted
	rep2, err := s.manager.GetReputation(peerID)
	require.NoError(t, err)
	require.NotNil(t, rep2)
	require.Equal(t, rep1.Score, rep2.Score)
	require.Equal(t, rep1.Metrics.ValidMessages, rep2.Metrics.ValidMessages)
}

// Test concurrent reputation updates
func (s *ReputationTestSuite) TestConcurrentUpdates() {
	t := s.T()

	peerID := PeerID("concurrent-peer")
	numGoroutines := 10
	eventsPerGoroutine := 100

	// Launch concurrent updates
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < eventsPerGoroutine; j++ {
				eventType := EventTypeValidMessage
				if j%10 == 0 {
					eventType = EventTypeInvalidMessage
				}

				event := &PeerEvent{
					PeerID:    peerID,
					EventType: eventType,
					Timestamp: time.Now(),
				}

				if err := s.manager.RecordEvent(event); err != nil {
					t.Errorf("concurrent update failed: %v", err)
				}
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final state
	rep, err := s.manager.GetReputation(peerID)
	require.NoError(t, err)
	require.NotNil(t, rep)

	expectedValid := numGoroutines * eventsPerGoroutine * 9 / 10 // 90% valid
	expectedInvalid := numGoroutines * eventsPerGoroutine / 10   // 10% invalid
	expectedTotal := numGoroutines * eventsPerGoroutine

	require.Equal(t, int64(expectedValid), rep.Metrics.ValidMessages)
	require.Equal(t, int64(expectedInvalid), rep.Metrics.InvalidMessages)
	require.Equal(t, int64(expectedTotal), rep.Metrics.TotalMessages)
}

// Test whitelist functionality
func (s *ReputationTestSuite) TestWhitelist() {
	t := s.T()

	peerID := PeerID("whitelist-peer")

	// Add to whitelist
	s.manager.AddToWhitelist(peerID)

	// Send malicious events
	for i := 0; i < 5; i++ {
		event := &PeerEvent{
			PeerID:    peerID,
			EventType: EventTypeDoubleSign,
			Timestamp: time.Now(),
		}
		require.NoError(t, s.manager.RecordEvent(event))
	}

	// Whitelisted peer should not be banned
	rep, err := s.manager.GetReputation(peerID)
	require.NoError(t, err)
	require.True(t, rep.BanStatus.IsWhitelisted)
	require.False(t, rep.BanStatus.IsBanned)

	// Should be accepted despite low score
	accepted, _ := s.manager.ShouldAcceptPeer(peerID, "127.0.0.1:26656")
	require.True(t, accepted)

	// Remove from whitelist
	s.manager.RemoveFromWhitelist(peerID)

	rep, err = s.manager.GetReputation(peerID)
	require.NoError(t, err)
	require.False(t, rep.BanStatus.IsWhitelisted)
}

// Test subnet limits
func (s *ReputationTestSuite) TestSubnetLimits() {
	t := s.T()

	// Add peers from same subnet
	subnet := "192.168.1.0/24"
	maxPeers := s.manager.config.MaxPeersPerSubnet

	for i := 0; i < maxPeers+5; i++ {
		peerID := PeerID("subnet-peer-" + string(rune('0'+i)))
		addr := "192.168.1." + string(rune('1'+i)) + ":26656"

		event := &PeerEvent{
			PeerID:    peerID,
			EventType: EventTypeConnected,
			Timestamp: time.Now(),
		}
		require.NoError(t, s.manager.RecordEvent(event))

		// Update network info
		rep, err := s.manager.GetReputation(peerID)
		require.NoError(t, err)
		rep.NetworkInfo.IPAddress = addr
		rep.NetworkInfo.Subnet = subnet
		require.NoError(t, s.storage.Save(rep))

		// Force stats update
		s.manager.updateStats(rep)
	}

	// Try to add one more peer from same subnet
	newPeerID := PeerID("subnet-peer-overflow")
	newAddr := "192.168.1.254:26656"

	accepted, reason := s.manager.ShouldAcceptPeer(newPeerID, newAddr)
	require.False(t, accepted)
	require.Contains(t, reason, "subnet")
}

// Test top peers selection
func (s *ReputationTestSuite) TestGetTopPeers() {
	t := s.T()

	// Create peers with varying scores
	for i := 0; i < 10; i++ {
		peerID := PeerID("top-peer-" + string(rune('0'+i)))

		event := &PeerEvent{
			PeerID:    peerID,
			EventType: EventTypeConnected,
			Timestamp: time.Now(),
		}
		require.NoError(t, s.manager.RecordEvent(event))

		// Send varying numbers of valid messages
		for j := 0; j < (i+1)*10; j++ {
			event = &PeerEvent{
				PeerID:    peerID,
				EventType: EventTypeValidMessage,
				Timestamp: time.Now(),
			}
			require.NoError(t, s.manager.RecordEvent(event))
		}
	}

	// Get top 5 peers
	topPeers := s.manager.GetTopPeers(5, 0.0)
	require.Len(t, topPeers, 5)

	// Verify they're sorted by score
	for i := 1; i < len(topPeers); i++ {
		require.GreaterOrEqual(t, topPeers[i-1].Score, topPeers[i].Score,
			"top peers should be sorted by score")
	}
}

// Test diverse peers selection
func (s *ReputationTestSuite) TestGetDiversePeers() {
	t := s.T()

	countries := []string{"US", "UK", "DE", "JP", "AU"}

	// Create peers from different countries
	for i, country := range countries {
		for j := 0; j < 3; j++ {
			peerID := PeerID("diverse-peer-" + country + "-" + string(rune('0'+j)))

			event := &PeerEvent{
				PeerID:    peerID,
				EventType: EventTypeConnected,
				Timestamp: time.Now(),
			}
			require.NoError(t, s.manager.RecordEvent(event))

			// Set country and score
			rep, err := s.manager.GetReputation(peerID)
			require.NoError(t, err)
			rep.NetworkInfo.Country = country
			rep.Score = float64(50 + i*10 + j*5) // Varying scores
			require.NoError(t, s.storage.Save(rep))
		}
	}

	// Get diverse peers
	diversePeers := s.manager.GetDiversePeers(10, 0.0)

	// Count countries represented
	countrySeen := make(map[string]bool)
	for _, peer := range diversePeers {
		countrySeen[peer.NetworkInfo.Country] = true
	}

	// Should have good geographic diversity
	require.GreaterOrEqual(t, len(countrySeen), 3,
		"should have peers from at least 3 different countries")
}

// Test statistics collection
func (s *ReputationTestSuite) TestStatistics() {
	t := s.T()

	// Create various peers
	peers := []struct {
		id     PeerID
		score  float64
		banned bool
	}{
		{"stats-peer-1", 90.0, false},
		{"stats-peer-2", 70.0, false},
		{"stats-peer-3", 50.0, false},
		{"stats-peer-4", 30.0, true},
		{"stats-peer-5", 10.0, true},
	}

	for _, p := range peers {
		event := &PeerEvent{
			PeerID:    p.id,
			EventType: EventTypeConnected,
			Timestamp: time.Now(),
		}
		require.NoError(t, s.manager.RecordEvent(event))

		if p.banned {
			require.NoError(t, s.manager.BanPeer(p.id, 1*time.Hour, "test"))
		}
	}

	// Get statistics
	stats := s.manager.GetStatistics()

	require.Equal(t, len(peers), stats.TotalPeers)
	require.Equal(t, 2, stats.BannedPeers)
	require.Greater(t, stats.AvgScore, 0.0)
	require.NotEmpty(t, stats.ScoreDistribution)
	require.NotEmpty(t, stats.TrustDistribution)
}

// Test ban duration calculation
func (s *ReputationTestSuite) TestBanDurationCalculation() {
	t := s.T()

	rep := &PeerReputation{
		PeerID: "ban-duration-peer",
		BanStatus: BanInfo{
			BanCount: 0,
		},
	}

	// First ban - 1 hour
	duration := s.scorer.GetBanDuration(rep)
	require.Equal(t, 1*time.Hour, duration)

	// Second ban - 2 hours
	rep.BanStatus.BanCount = 1
	duration = s.scorer.GetBanDuration(rep)
	require.Equal(t, 2*time.Hour, duration)

	// Third ban - 4 hours
	rep.BanStatus.BanCount = 2
	duration = s.scorer.GetBanDuration(rep)
	require.Equal(t, 4*time.Hour, duration)

	// Many bans - should cap at 7 days
	rep.BanStatus.BanCount = 20
	duration = s.scorer.GetBanDuration(rep)
	require.Equal(t, 7*24*time.Hour, duration)
}

// Test trust level calculation
func (s *ReputationTestSuite) TestTrustLevelCalculation() {
	t := s.T()

	testCases := []struct {
		score       float64
		whitelisted bool
		expected    TrustLevel
	}{
		{100.0, false, TrustLevelHigh},
		{75.0, false, TrustLevelHigh},
		{50.0, false, TrustLevelMedium},
		{30.0, false, TrustLevelLow},
		{10.0, false, TrustLevelUntrusted},
		{-1.0, false, TrustLevelUnknown},
		{50.0, true, TrustLevelWhitelisted},
	}

	for _, tc := range testCases {
		level := CalculateTrustLevel(tc.score, tc.whitelisted)
		require.Equal(t, tc.expected, level,
			"score %.1f (whitelisted=%v) should be %s",
			tc.score, tc.whitelisted, tc.expected.String())
	}
}

func TestReputationTestSuite(t *testing.T) {
	suite.Run(t, new(ReputationTestSuite))
}

// Benchmarks

func BenchmarkScoreCalculation(b *testing.B) {
	scoringConfig := DefaultScoringConfig()
	scorer := NewScorer(DefaultScoreWeights(), &scoringConfig)

	rep := &PeerReputation{
		PeerID:    "bench-peer",
		FirstSeen: time.Now().Add(-24 * time.Hour),
		LastSeen:  time.Now(),
		Metrics: PeerMetrics{
			TotalUptime:         12 * time.Hour,
			ConnectionCount:     10,
			DisconnectionCount:  2,
			ValidMessages:       1000,
			InvalidMessages:     10,
			TotalMessages:       1010,
			ValidMessageRatio:   0.99,
			AvgResponseLatency:  100 * time.Millisecond,
			LatencyMeasurements: 100,
			BlocksPropagated:    500,
			FastBlockCount:      450,
			ProtocolViolations:  2,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scorer.CalculateScore(rep)
	}
}

func BenchmarkEventRecording(b *testing.B) {
	logger := log.NewNopLogger()
	storage := NewMemoryStorage()
	config := DefaultManagerConfig()

	mgr, err := NewManager(storage, &config, logger)
	if err != nil {
		b.Fatal(err)
	}
	defer mgr.Close()

	event := &PeerEvent{
		PeerID:    "bench-peer",
		EventType: EventTypeValidMessage,
		Timestamp: time.Now(),
		Data:      EventData{MessageSize: 1024},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mgr.RecordEvent(event)
	}
}

func BenchmarkConcurrentEventRecording(b *testing.B) {
	logger := log.NewNopLogger()
	storage := NewMemoryStorage()
	config := DefaultManagerConfig()

	mgr, err := NewManager(storage, &config, logger)
	if err != nil {
		b.Fatal(err)
	}
	defer mgr.Close()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			event := &PeerEvent{
				PeerID:    PeerID("bench-peer-" + string(rune('0'+i%10))),
				EventType: EventTypeValidMessage,
				Timestamp: time.Now(),
			}
			_ = mgr.RecordEvent(event)
			i++
		}
	})
}

func BenchmarkGetTopPeers(b *testing.B) {
	logger := log.NewNopLogger()
	storage := NewMemoryStorage()
	config := DefaultManagerConfig()

	mgr, err := NewManager(storage, &config, logger)
	if err != nil {
		b.Fatal(err)
	}
	defer mgr.Close()

	// Create 100 peers
	for i := 0; i < 100; i++ {
		event := &PeerEvent{
			PeerID:    PeerID("bench-peer-" + string(rune('0'+i))),
			EventType: EventTypeConnected,
			Timestamp: time.Now(),
		}
		_ = mgr.RecordEvent(event)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mgr.GetTopPeers(10, 50.0)
	}
}
