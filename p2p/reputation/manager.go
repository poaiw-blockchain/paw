package reputation

import (
	"fmt"
	"sync"
	"time"

	"cosmossdk.io/log"
)

// Manager manages peer reputation tracking and decisions
type Manager struct {
	storage Storage
	scorer  *Scorer
	config  ManagerConfig
	logger  log.Logger

	// In-memory state
	peers   map[PeerID]*PeerReputation
	peersMu sync.RWMutex

	// Subnet and geographic tracking
	subnetStats map[string]*SubnetStats
	geoStats    map[string]*GeographicStats
	statsMu     sync.RWMutex

	// Whitelist and blacklist
	whitelist map[PeerID]bool
	blacklist map[PeerID]bool
	listsMu   sync.RWMutex

	// Background tasks
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Metrics
	metrics *Metrics
}

// ManagerConfig configures the reputation manager
type ManagerConfig struct {
	// Scoring
	ScoreWeights  ScoreWeights
	ScoringConfig ScoringConfig

	// Security limits
	MaxPeersPerSubnet      int // Max peers from same /24 subnet
	MaxPeersPerCountry     int // Max peers from same country
	MinGeographicDiversity int // Min number of different countries
	MaxPeersPerASN         int // Max peers from same ASN

	// Ban settings
	EnableAutoBan   bool
	TempBanDuration time.Duration
	MaxTempBans     int // Convert to permanent after this many temp bans

	// Maintenance
	SnapshotInterval   time.Duration
	CleanupInterval    time.Duration
	CleanupAge         time.Duration
	ScoreDecayInterval time.Duration

	// Performance
	EnableGeoLookup        bool
	GeoLookupCacheDuration time.Duration
}

// DefaultManagerConfig returns default manager configuration
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		ScoreWeights:  DefaultScoreWeights(),
		ScoringConfig: DefaultScoringConfig(),

		MaxPeersPerSubnet:      10,
		MaxPeersPerCountry:     50,
		MinGeographicDiversity: 3,
		MaxPeersPerASN:         15,

		EnableAutoBan:   true,
		TempBanDuration: 24 * time.Hour,
		MaxTempBans:     3,

		SnapshotInterval:   1 * time.Hour,
		CleanupInterval:    24 * time.Hour,
		CleanupAge:         30 * 24 * time.Hour, // 30 days
		ScoreDecayInterval: 1 * time.Hour,

		EnableGeoLookup:        false, // Disabled by default (requires external service)
		GeoLookupCacheDuration: 7 * 24 * time.Hour,
	}
}

// NewManager creates a new reputation manager
func NewManager(storage Storage, config ManagerConfig, logger log.Logger) (*Manager, error) {
	m := &Manager{
		storage:     storage,
		scorer:      NewScorer(config.ScoreWeights, config.ScoringConfig),
		config:      config,
		logger:      logger,
		peers:       make(map[PeerID]*PeerReputation),
		subnetStats: make(map[string]*SubnetStats),
		geoStats:    make(map[string]*GeographicStats),
		whitelist:   make(map[PeerID]bool),
		blacklist:   make(map[PeerID]bool),
		stopChan:    make(chan struct{}),
		metrics:     NewMetrics(),
	}

	// Load existing data
	if err := m.loadState(); err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Start background tasks
	m.startBackgroundTasks()

	logger.Info("reputation manager started", "peers", len(m.peers))
	return m, nil
}

// RecordEvent records a peer event and updates reputation
func (m *Manager) RecordEvent(event PeerEvent) error {
	m.peersMu.Lock()
	defer m.peersMu.Unlock()

	// Get or create peer reputation
	rep, exists := m.peers[event.PeerID]
	if !exists {
		rep = m.createNewPeer(event.PeerID, "")
	}

	// Apply event to update metrics
	m.scorer.ApplyEvent(rep, event)

	// Check if peer should be banned
	if m.config.EnableAutoBan && !rep.BanStatus.IsWhitelisted {
		shouldBan, banType, reason := m.scorer.ShouldBan(rep)
		if shouldBan {
			m.banPeer(rep, banType, reason)
		}
	}

	// Update statistics
	m.updateStats(rep)

	// Save to storage
	if err := m.storage.Save(rep); err != nil {
		m.logger.Error("failed to save peer reputation", "peer_id", event.PeerID, "error", err)
		return err
	}

	// Update metrics
	m.metrics.RecordEvent(event.EventType)
	m.metrics.UpdateScore(event.PeerID, rep.Score)

	return nil
}

// GetReputation returns reputation for a peer
func (m *Manager) GetReputation(peerID PeerID) (*PeerReputation, error) {
	m.peersMu.RLock()
	rep, exists := m.peers[peerID]
	m.peersMu.RUnlock()

	if exists {
		// Return a copy
		repCopy := *rep
		return &repCopy, nil
	}

	// Try loading from storage
	rep, err := m.storage.Load(peerID)
	if err != nil {
		return nil, err
	}

	if rep != nil {
		// Cache it
		m.peersMu.Lock()
		m.peers[peerID] = rep
		m.peersMu.Unlock()
	}

	return rep, nil
}

// ShouldAcceptPeer determines if a new peer connection should be accepted
func (m *Manager) ShouldAcceptPeer(peerID PeerID, address string) (bool, string) {
	m.peersMu.RLock()
	defer m.peersMu.RUnlock()

	// Check blacklist
	m.listsMu.RLock()
	if m.blacklist[peerID] {
		m.listsMu.RUnlock()
		return false, "peer is blacklisted"
	}
	isWhitelisted := m.whitelist[peerID]
	m.listsMu.RUnlock()

	// Whitelisted peers always accepted
	if isWhitelisted {
		return true, ""
	}

	// Check existing reputation
	rep, exists := m.peers[peerID]
	if exists {
		// Check if banned
		if rep.BanStatus.IsBanned {
			if rep.BanStatus.BanType == BanTypePermanent {
				return false, "peer is permanently banned"
			}
			if time.Now().Before(rep.BanStatus.BanExpires) {
				return false, fmt.Sprintf("peer is temporarily banned until %s", rep.BanStatus.BanExpires)
			}
			// Ban expired, clear it
			rep.BanStatus.IsBanned = false
		}

		// Check reputation score
		if rep.Score < 30.0 {
			return false, "peer reputation too low"
		}
	}

	// Extract network info
	subnet := ParseSubnet(address)
	if subnet == "" {
		return false, "invalid peer address"
	}

	// Check subnet limits
	m.statsMu.RLock()
	if stats, ok := m.subnetStats[subnet]; ok {
		if stats.PeerCount >= m.config.MaxPeersPerSubnet {
			m.statsMu.RUnlock()
			return false, fmt.Sprintf("subnet %s has too many peers (%d)", subnet, stats.PeerCount)
		}
	}
	m.statsMu.RUnlock()

	// All checks passed
	return true, ""
}

// GetTopPeers returns the N highest-reputation peers
func (m *Manager) GetTopPeers(n int, minScore float64) []*PeerReputation {
	m.peersMu.RLock()
	defer m.peersMu.RUnlock()

	// Filter and collect peers
	candidates := make([]*PeerReputation, 0, len(m.peers))
	for _, rep := range m.peers {
		if rep.Score >= minScore && !rep.BanStatus.IsBanned {
			repCopy := *rep
			candidates = append(candidates, &repCopy)
		}
	}

	// Sort by score (bubble sort for small lists, good enough)
	for i := 0; i < len(candidates)-1; i++ {
		for j := 0; j < len(candidates)-i-1; j++ {
			if candidates[j].Score < candidates[j+1].Score {
				candidates[j], candidates[j+1] = candidates[j+1], candidates[j]
			}
		}
	}

	// Return top N
	if n > len(candidates) {
		n = len(candidates)
	}

	return candidates[:n]
}

// GetDiversePeers returns peers ensuring geographic diversity
func (m *Manager) GetDiversePeers(n int, minScore float64) []*PeerReputation {
	m.peersMu.RLock()
	defer m.peersMu.RUnlock()

	// Group peers by country
	byCountry := make(map[string][]*PeerReputation)
	for _, rep := range m.peers {
		if rep.Score >= minScore && !rep.BanStatus.IsBanned {
			country := rep.NetworkInfo.Country
			if country == "" {
				country = "unknown"
			}
			byCountry[country] = append(byCountry[country], rep)
		}
	}

	// Round-robin selection from different countries
	result := make([]*PeerReputation, 0, n)
	countries := make([]string, 0, len(byCountry))
	for country := range byCountry {
		countries = append(countries, country)
	}

	idx := 0
	for len(result) < n && len(byCountry) > 0 {
		country := countries[idx%len(countries)]
		peers := byCountry[country]

		if len(peers) > 0 {
			// Take best peer from this country
			best := peers[0]
			for _, p := range peers {
				if p.Score > best.Score {
					best = p
				}
			}

			repCopy := *best
			result = append(result, &repCopy)

			// Remove selected peer
			newPeers := make([]*PeerReputation, 0, len(peers)-1)
			for _, p := range peers {
				if p.PeerID != best.PeerID {
					newPeers = append(newPeers, p)
				}
			}

			if len(newPeers) > 0 {
				byCountry[country] = newPeers
			} else {
				delete(byCountry, country)
				// Remove country from list
				newCountries := make([]string, 0, len(countries)-1)
				for _, c := range countries {
					if c != country {
						newCountries = append(newCountries, c)
					}
				}
				countries = newCountries
			}
		}

		idx++
	}

	return result
}

// BanPeer manually bans a peer
func (m *Manager) BanPeer(peerID PeerID, duration time.Duration, reason string) error {
	m.peersMu.Lock()
	defer m.peersMu.Unlock()

	rep, exists := m.peers[peerID]
	if !exists {
		rep = m.createNewPeer(peerID, "")
	}

	banType := BanTypeTemporary
	if duration == 0 {
		banType = BanTypePermanent
	}

	m.banPeer(rep, banType, reason)

	if banType == BanTypeTemporary {
		rep.BanStatus.BanExpires = time.Now().Add(duration)
	}

	return m.storage.Save(rep)
}

// UnbanPeer manually unbans a peer
func (m *Manager) UnbanPeer(peerID PeerID) error {
	m.peersMu.Lock()
	defer m.peersMu.Unlock()

	rep, exists := m.peers[peerID]
	if !exists {
		return fmt.Errorf("peer not found")
	}

	rep.BanStatus.IsBanned = false
	rep.BanStatus.BanType = BanTypeNone
	rep.BanStatus.BanExpires = time.Time{}

	return m.storage.Save(rep)
}

// AddToWhitelist adds a peer to whitelist
func (m *Manager) AddToWhitelist(peerID PeerID) {
	m.listsMu.Lock()
	defer m.listsMu.Unlock()

	m.whitelist[peerID] = true

	m.peersMu.Lock()
	if rep, exists := m.peers[peerID]; exists {
		rep.BanStatus.IsWhitelisted = true
		rep.TrustLevel = TrustLevelWhitelisted
	}
	m.peersMu.Unlock()
}

// RemoveFromWhitelist removes a peer from whitelist
func (m *Manager) RemoveFromWhitelist(peerID PeerID) {
	m.listsMu.Lock()
	defer m.listsMu.Unlock()

	delete(m.whitelist, peerID)

	m.peersMu.Lock()
	if rep, exists := m.peers[peerID]; exists {
		rep.BanStatus.IsWhitelisted = false
		rep.TrustLevel = CalculateTrustLevel(rep.Score, false)
	}
	m.peersMu.Unlock()
}

// GetStatistics returns current statistics
func (m *Manager) GetStatistics() Statistics {
	m.peersMu.RLock()
	defer m.peersMu.RUnlock()

	stats := Statistics{
		TotalPeers:        len(m.peers),
		BannedPeers:       0,
		WhitelistedPeers:  len(m.whitelist),
		AvgScore:          0.0,
		ScoreDistribution: make(map[string]int),
		TrustDistribution: make(map[string]int),
	}

	totalScore := 0.0
	for _, rep := range m.peers {
		totalScore += rep.Score

		if rep.BanStatus.IsBanned {
			stats.BannedPeers++
		}

		// Score distribution (buckets)
		switch {
		case rep.Score < 20:
			stats.ScoreDistribution["0-20"]++
		case rep.Score < 40:
			stats.ScoreDistribution["20-40"]++
		case rep.Score < 60:
			stats.ScoreDistribution["40-60"]++
		case rep.Score < 80:
			stats.ScoreDistribution["60-80"]++
		default:
			stats.ScoreDistribution["80-100"]++
		}

		// Trust distribution
		stats.TrustDistribution[rep.TrustLevel.String()]++
	}

	if len(m.peers) > 0 {
		stats.AvgScore = totalScore / float64(len(m.peers))
	}

	return stats
}

// Close shuts down the manager
func (m *Manager) Close() error {
	m.logger.Info("shutting down reputation manager")

	// Stop background tasks
	close(m.stopChan)
	m.wg.Wait()

	// Save final snapshot
	if err := m.saveSnapshot(); err != nil {
		m.logger.Error("failed to save final snapshot", "error", err)
	}

	// Close storage
	if err := m.storage.Close(); err != nil {
		return fmt.Errorf("failed to close storage: %w", err)
	}

	return nil
}

// Internal methods

func (m *Manager) createNewPeer(peerID PeerID, address string) *PeerReputation {
	now := time.Now()

	rep := &PeerReputation{
		PeerID:     peerID,
		Address:    address,
		Score:      m.config.ScoringConfig.NewPeerStartScore,
		FirstSeen:  now,
		LastSeen:   now,
		TrustLevel: TrustLevelUnknown,
		Metrics:    PeerMetrics{},
		BanStatus:  BanInfo{},
		NetworkInfo: NetworkInfo{
			IPAddress: address,
			Subnet:    ParseSubnet(address),
		},
	}

	m.peers[peerID] = rep
	return rep
}

func (m *Manager) banPeer(rep *PeerReputation, banType BanType, reason string) {
	now := time.Now()

	rep.BanStatus.IsBanned = true
	rep.BanStatus.BanType = banType
	rep.BanStatus.BannedAt = now
	rep.BanStatus.BanReason = reason
	rep.BanStatus.BanCount++

	if banType == BanTypeTemporary {
		duration := m.scorer.GetBanDuration(rep)
		rep.BanStatus.BanExpires = now.Add(duration)

		// Upgrade to permanent if too many temp bans
		if rep.BanStatus.BanCount >= m.config.MaxTempBans {
			rep.BanStatus.BanType = BanTypePermanent
			rep.BanStatus.BanExpires = time.Time{}
		}
	}

	m.logger.Info("peer banned",
		"peer_id", rep.PeerID,
		"ban_type", banType.String(),
		"reason", reason,
		"expires", rep.BanStatus.BanExpires,
	)

	m.metrics.RecordBan(banType)
}

func (m *Manager) updateStats(rep *PeerReputation) {
	m.statsMu.Lock()
	defer m.statsMu.Unlock()

	// Update subnet stats
	subnet := rep.NetworkInfo.Subnet
	if subnet != "" {
		stats, exists := m.subnetStats[subnet]
		if !exists {
			stats = &SubnetStats{
				Subnet:      subnet,
				LastUpdated: time.Now(),
			}
			m.subnetStats[subnet] = stats
		}

		// Recalculate (simple approach - could be optimized)
		stats.PeerCount = 0
		stats.BannedCount = 0
		totalScore := 0.0

		for _, p := range m.peers {
			if p.NetworkInfo.Subnet == subnet {
				stats.PeerCount++
				totalScore += p.Score
				if p.BanStatus.IsBanned {
					stats.BannedCount++
				}
			}
		}

		if stats.PeerCount > 0 {
			stats.AvgScore = totalScore / float64(stats.PeerCount)
		}
		stats.LastUpdated = time.Now()
	}

	// Update geo stats (if available)
	country := rep.NetworkInfo.Country
	if country != "" {
		stats, exists := m.geoStats[country]
		if !exists {
			stats = &GeographicStats{
				Country:     country,
				LastUpdated: time.Now(),
			}
			m.geoStats[country] = stats
		}

		// Recalculate
		stats.PeerCount = 0
		stats.BannedCount = 0
		totalScore := 0.0

		for _, p := range m.peers {
			if p.NetworkInfo.Country == country {
				stats.PeerCount++
				totalScore += p.Score
				if p.BanStatus.IsBanned {
					stats.BannedCount++
				}
			}
		}

		if stats.PeerCount > 0 {
			stats.AvgScore = totalScore / float64(stats.PeerCount)
		}
		stats.LastUpdated = time.Now()
	}
}

func (m *Manager) loadState() error {
	// Load all peers from storage
	peers, err := m.storage.LoadAll()
	if err != nil {
		return fmt.Errorf("failed to load peers: %w", err)
	}

	m.peersMu.Lock()
	m.peers = peers
	m.peersMu.Unlock()

	// Rebuild statistics
	m.peersMu.RLock()
	for _, rep := range m.peers {
		m.updateStats(rep)

		if rep.BanStatus.IsWhitelisted {
			m.listsMu.Lock()
			m.whitelist[rep.PeerID] = true
			m.listsMu.Unlock()
		}
	}
	m.peersMu.RUnlock()

	return nil
}

func (m *Manager) saveSnapshot() error {
	m.peersMu.RLock()
	defer m.peersMu.RUnlock()

	stats := m.GetStatistics()

	snapshot := &ReputationSnapshot{
		Timestamp:   time.Now(),
		TotalPeers:  stats.TotalPeers,
		BannedPeers: stats.BannedPeers,
		AvgScore:    stats.AvgScore,
		Peers:       make(map[PeerID]*PeerReputation),
	}

	for id, rep := range m.peers {
		repCopy := *rep
		snapshot.Peers[id] = &repCopy
	}

	return m.storage.SaveSnapshot(snapshot)
}

func (m *Manager) startBackgroundTasks() {
	// Snapshot task
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(m.config.SnapshotInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := m.saveSnapshot(); err != nil {
					m.logger.Error("failed to save snapshot", "error", err)
				}
			case <-m.stopChan:
				return
			}
		}
	}()

	// Cleanup task
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(m.config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				olderThan := time.Now().Add(-m.config.CleanupAge)
				if err := m.storage.Cleanup(olderThan); err != nil {
					m.logger.Error("cleanup failed", "error", err)
				}
			case <-m.stopChan:
				return
			}
		}
	}()

	// Score decay task
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(m.config.ScoreDecayInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.applyScoreDecay()
			case <-m.stopChan:
				return
			}
		}
	}()
}

func (m *Manager) applyScoreDecay() {
	m.peersMu.Lock()
	defer m.peersMu.Unlock()

	for _, rep := range m.peers {
		oldScore := rep.Score
		newScore := m.scorer.CalculateScore(rep)

		if newScore != oldScore {
			rep.Score = newScore
			rep.TrustLevel = CalculateTrustLevel(newScore, rep.BanStatus.IsWhitelisted)
		}
	}
}

// Statistics holds reputation statistics
type Statistics struct {
	TotalPeers        int
	BannedPeers       int
	WhitelistedPeers  int
	AvgScore          float64
	ScoreDistribution map[string]int
	TrustDistribution map[string]int
}
