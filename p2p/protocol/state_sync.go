package protocol

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"cosmossdk.io/log"

	"github.com/paw-chain/paw/p2p/snapshot"
)

// StateSyncProtocol handles fast state synchronization using snapshots
type StateSyncProtocol struct {
	config *StateSyncConfig
	logger log.Logger

	// Snapshot management
	snapshotMgr *snapshot.Manager

	// Peer communication
	peerManager PeerManager

	// State sync tracking
	state            StateSyncState
	stateMu          sync.RWMutex
	selectedSnapshot *snapshot.Snapshot
	downloadedChunks map[uint32]bool
	chunksMu         sync.RWMutex

	// Byzantine detection
	peerOffers       map[string]*snapshot.SnapshotOffer // peerID -> offer
	peerOffersMu     sync.RWMutex
	maliciousPeers   map[string]bool
	maliciousPeersMu sync.RWMutex

	// Progress tracking
	chunksDownloaded uint64
	bytesReceived    uint64
	snapshotHeight   int64
	startTime        time.Time

	// Callbacks
	applySnapshotCallback func(height int64, data []byte) error
	verifyStateCallback   func(height int64, appHash []byte) error

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	metrics *StateSyncMetrics
}

// StateSyncConfig configures state sync behavior
type StateSyncConfig struct {
	// Trust parameters (for light client verification)
	TrustHeight int64         // Trusted block height
	TrustHash   []byte        // Trusted block hash
	TrustPeriod time.Duration // Trust period for light client

	// Chunk download settings
	ChunkFetchers       uint32        // Parallel chunk downloads
	ChunkSize           uint32        // Bytes per chunk
	ChunkRequestTimeout time.Duration // Timeout for chunk requests
	ChunkRetryAttempts  int           // Max retries per chunk

	// Discovery settings
	DiscoveryTime     time.Duration // Time to discover snapshots
	MinSnapshotOffers int           // Minimum offers before selection
	RequireBFTProof   bool          // Require 2/3+ validator signatures

	// Byzantine fault tolerance
	MinPeerAgreement  float64 // Minimum peer agreement (e.g., 0.67 for 2/3+)
	MaxMaliciousPeers int     // Max malicious peers before abort
	VerifyAllChunks   bool    // Verify all chunks against hashes

	// Fallback settings
	FallbackToBlockSync bool          // Fall back to block sync on failure
	StateTimeout        time.Duration // Total state sync timeout
}

// DefaultStateSyncConfig returns default state sync configuration
func DefaultStateSyncConfig() *StateSyncConfig {
	return &StateSyncConfig{
		TrustPeriod:         7 * 24 * time.Hour, // 7 days
		ChunkFetchers:       4,
		ChunkSize:           16 * 1024 * 1024, // 16 MB
		ChunkRequestTimeout: 30 * time.Second,
		ChunkRetryAttempts:  3,
		DiscoveryTime:       10 * time.Second,
		MinSnapshotOffers:   3,
		RequireBFTProof:     true,
		MinPeerAgreement:    0.67, // 2/3+
		MaxMaliciousPeers:   3,
		VerifyAllChunks:     true,
		FallbackToBlockSync: true,
		StateTimeout:        10 * time.Minute,
	}
}

const (
	defaultReliabilityScore = 0.5
	minReliabilityScore     = 0.05
)

// StateSyncState represents state sync state
type StateSyncState int

const (
	StateSyncStateIdle StateSyncState = iota
	StateSyncStateDiscovering
	StateSyncStateSelecting
	StateSyncStateDownloading
	StateSyncStateApplying
	StateSyncStateVerifying
	StateSyncStateComplete
	StateSyncStateFailed
)

func (s StateSyncState) String() string {
	switch s {
	case StateSyncStateIdle:
		return "Idle"
	case StateSyncStateDiscovering:
		return "Discovering"
	case StateSyncStateSelecting:
		return "Selecting"
	case StateSyncStateDownloading:
		return "Downloading"
	case StateSyncStateApplying:
		return "Applying"
	case StateSyncStateVerifying:
		return "Verifying"
	case StateSyncStateComplete:
		return "Complete"
	case StateSyncStateFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// StateSyncMetrics tracks state sync statistics
type StateSyncMetrics struct {
	SnapshotsDiscovered int64
	ChunksDownloaded    int64
	ChunksVerified      int64
	BytesDownloaded     int64
	PeersQueried        int64
	MaliciousPeersFound int64
	DownloadTime        time.Duration
	VerificationTime    time.Duration
	TotalTime           time.Duration
	mu                  sync.RWMutex
}

// PeerManager interface for peer communication
type PeerManager interface {
	QueryPeerSnapshot(ctx context.Context, peerID string) (*snapshot.SnapshotMetadata, error)
	RequestChunk(ctx context.Context, peerID string, height int64, chunkIndex uint32) ([]byte, error)
	GetAvailablePeers() []string
	ReportMaliciousPeer(peerID string, reason string)
	GetPeerReliability(peerID string) float64
}

// NewStateSyncProtocol creates a new state sync protocol instance
func NewStateSyncProtocol(
	config *StateSyncConfig,
	snapshotMgr *snapshot.Manager,
	peerMgr PeerManager,
	logger log.Logger,
) *StateSyncProtocol {
	ctx, cancel := context.WithCancel(context.Background())

	return &StateSyncProtocol{
		config:           config,
		logger:           logger,
		snapshotMgr:      snapshotMgr,
		peerManager:      peerMgr,
		state:            StateSyncStateIdle,
		downloadedChunks: make(map[uint32]bool),
		peerOffers:       make(map[string]*snapshot.SnapshotOffer),
		maliciousPeers:   make(map[string]bool),
		ctx:              ctx,
		cancel:           cancel,
		metrics:          &StateSyncMetrics{},
	}
}

// StartStateSync initiates fast state synchronization
func (ssp *StateSyncProtocol) StartStateSync(ctx context.Context) error {
	ssp.stateMu.Lock()
	if ssp.state != StateSyncStateIdle {
		ssp.stateMu.Unlock()
		return fmt.Errorf("state sync already in progress: %s", ssp.state)
	}
	ssp.state = StateSyncStateDiscovering
	ssp.startTime = time.Now()
	ssp.stateMu.Unlock()

	ssp.logger.Info("starting state sync")

	// Set timeout for entire state sync process
	syncCtx := ctx
	if ssp.config.StateTimeout > 0 {
		var cancel context.CancelFunc
		syncCtx, cancel = context.WithTimeout(ctx, ssp.config.StateTimeout)
		defer cancel()
	}

	// 1. Discover available snapshots from peers
	snapshots, err := ssp.discoverSnapshots(syncCtx)
	if err != nil {
		ssp.setState(StateSyncStateFailed)
		return fmt.Errorf("snapshot discovery failed: %w", err)
	}

	if len(snapshots) == 0 {
		ssp.setState(StateSyncStateFailed)
		if ssp.config.FallbackToBlockSync {
			ssp.logger.Info("no snapshots available, falling back to block sync")
			return errors.New("no snapshots available")
		}
		return errors.New("no snapshots available and fallback disabled")
	}

	ssp.metrics.mu.Lock()
	ssp.metrics.SnapshotsDiscovered = int64(len(snapshots))
	ssp.metrics.mu.Unlock()

	// 2. Select best snapshot (highest height with 2/3+ agreement)
	ssp.setState(StateSyncStateSelecting)
	bestSnapshot, err := ssp.selectBestSnapshot(snapshots)
	if err != nil {
		ssp.setState(StateSyncStateFailed)
		return fmt.Errorf("snapshot selection failed: %w", err)
	}

	ssp.selectedSnapshot = bestSnapshot
	ssp.snapshotHeight = bestSnapshot.Height

	ssp.logger.Info("selected snapshot",
		"height", bestSnapshot.Height,
		"chunks", bestSnapshot.NumChunks,
		"hash", fmt.Sprintf("%x", bestSnapshot.Hash[:8]))

	// 3. Verify snapshot against trusted state
	if err := ssp.verifySnapshot(bestSnapshot); err != nil {
		ssp.setState(StateSyncStateFailed)
		return fmt.Errorf("snapshot verification failed: %w", err)
	}

	// 4. Download snapshot chunks in parallel
	ssp.setState(StateSyncStateDownloading)
	downloadStart := time.Now()

	if err := ssp.downloadSnapshotChunks(syncCtx, bestSnapshot); err != nil {
		ssp.setState(StateSyncStateFailed)
		return fmt.Errorf("chunk download failed: %w", err)
	}

	ssp.metrics.mu.Lock()
	ssp.metrics.DownloadTime = time.Since(downloadStart)
	ssp.metrics.mu.Unlock()

	// 5. Apply snapshot to state
	ssp.setState(StateSyncStateApplying)
	stateData, err := ssp.applySnapshot(bestSnapshot)
	if err != nil {
		ssp.setState(StateSyncStateFailed)
		return fmt.Errorf("snapshot application failed: %w", err)
	}

	// 6. Verify applied state
	ssp.setState(StateSyncStateVerifying)
	verifyStart := time.Now()

	if err := ssp.verifyAppliedState(bestSnapshot, stateData); err != nil {
		ssp.setState(StateSyncStateFailed)
		return fmt.Errorf("state verification failed: %w", err)
	}

	ssp.metrics.mu.Lock()
	ssp.metrics.VerificationTime = time.Since(verifyStart)
	ssp.metrics.TotalTime = time.Since(ssp.startTime)
	ssp.metrics.mu.Unlock()

	ssp.setState(StateSyncStateComplete)

	ssp.logger.Info("state sync completed successfully",
		"height", bestSnapshot.Height,
		"chunks", bestSnapshot.NumChunks,
		"download_time", ssp.metrics.DownloadTime,
		"total_time", ssp.metrics.TotalTime)

	return nil
}

// discoverSnapshots queries peers for available snapshots
func (ssp *StateSyncProtocol) discoverSnapshots(ctx context.Context) ([]*snapshot.SnapshotOffer, error) {
	ssp.logger.Info("discovering snapshots from peers")

	peers := ssp.peerManager.GetAvailablePeers()
	if len(peers) == 0 {
		return nil, errors.New("no peers available")
	}

	offerChan := make(chan *snapshot.SnapshotOffer, len(peers))
	var wg sync.WaitGroup

	// Query all peers in parallel
	for _, peerID := range peers {
		wg.Add(1)
		go func(pid string) {
			defer wg.Done()

			// Skip malicious peers
			ssp.maliciousPeersMu.RLock()
			isMalicious := ssp.maliciousPeers[pid]
			ssp.maliciousPeersMu.RUnlock()

			if isMalicious {
				return
			}

			atomic.AddInt64(&ssp.metrics.PeersQueried, 1)

			meta, err := ssp.peerManager.QueryPeerSnapshot(ctx, pid)
			if err != nil {
				ssp.logger.Debug("failed to query peer snapshot",
					"peer_id", pid,
					"error", err)
				return
			}

			if meta != nil {
				reliability := ssp.getPeerReliability(pid)
				offer := &snapshot.SnapshotOffer{
					PeerID:      pid,
					Snapshot:    meta,
					ReceivedAt:  time.Now(),
					Reliability: reliability,
				}

				select {
				case offerChan <- offer:
					ssp.peerOffersMu.Lock()
					ssp.peerOffers[pid] = offer
					ssp.peerOffersMu.Unlock()
				case <-ctx.Done():
				}
			}
		}(peerID)
	}

	// Wait for all queries to complete or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	offers := make([]*snapshot.SnapshotOffer, 0)
	discoveryTimeout := time.After(ssp.config.DiscoveryTime)

	for {
		select {
		case offer := <-offerChan:
			offers = append(offers, offer)

			// If we have enough offers, we can proceed
			if len(offers) >= ssp.config.MinSnapshotOffers {
				// Continue collecting but don't block
			}

		case <-discoveryTimeout:
			ssp.logger.Info("snapshot discovery timeout reached",
				"offers", len(offers))
			return offers, nil

		case <-done:
			// Drain any remaining offers from the channel
			for {
				select {
				case offer := <-offerChan:
					offers = append(offers, offer)
				default:
					ssp.logger.Info("snapshot discovery complete",
						"offers", len(offers),
						"peers_queried", len(peers))
					return offers, nil
				}
			}

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// selectBestSnapshot chooses the optimal snapshot based on Byzantine fault tolerance
func (ssp *StateSyncProtocol) selectBestSnapshot(offers []*snapshot.SnapshotOffer) (*snapshot.Snapshot, error) {
	if len(offers) == 0 {
		return nil, errors.New("no snapshot offers available")
	}

	// Group offers by height and hash
	type SnapshotKey struct {
		Height int64
		Hash   string
	}

	type snapshotGroup struct {
		offers []*snapshot.SnapshotOffer
		weight float64
	}

	offerGroups := make(map[SnapshotKey]*snapshotGroup)
	totalWeight := 0.0

	for _, offer := range offers {
		key := SnapshotKey{
			Height: offer.Snapshot.Height,
			Hash:   fmt.Sprintf("%x", offer.Snapshot.Hash),
		}
		group := offerGroups[key]
		if group == nil {
			group = &snapshotGroup{}
			offerGroups[key] = group
		}

		reliability := normalizeReliability(offer.Reliability)
		group.offers = append(group.offers, offer)
		group.weight += reliability
		totalWeight += reliability
	}

	// Find the snapshot with highest height and sufficient peer agreement
	var bestKey SnapshotKey
	var maxHeight int64
	var maxAgreement float64
	var bestGroup *snapshotGroup

	for key, group := range offerGroups {
		var agreement float64
		if totalWeight > 0 {
			agreement = group.weight / totalWeight
		} else {
			agreement = float64(len(group.offers)) / float64(len(offers))
		}

		// Check if this snapshot meets our requirements
		if agreement >= ssp.config.MinPeerAgreement {
			if key.Height > maxHeight || (key.Height == maxHeight && agreement > maxAgreement) {
				bestKey = key
				maxHeight = key.Height
				maxAgreement = agreement
				bestGroup = group
			}
		}
	}

	if maxHeight == 0 {
		return nil, fmt.Errorf("no snapshot with sufficient peer agreement (min: %.2f%%)",
			ssp.config.MinPeerAgreement*100)
	}

	// Get the full snapshot from one of the agreeing peers
	if bestGroup == nil || len(bestGroup.offers) == 0 {
		return nil, errors.New("no offers for best snapshot")
	}

	// Construct full snapshot from metadata
	meta := bestGroup.offers[0].Snapshot
	fullSnapshot := &snapshot.Snapshot{
		Height:      meta.Height,
		Hash:        meta.Hash,
		NumChunks:   meta.NumChunks,
		Format:      meta.Format,
		ChainID:     meta.ChainID,
		Timestamp:   meta.Timestamp.Unix(),
		VotingPower: meta.VotingPower,
		TotalPower:  meta.TotalPower,
		ChunkHashes: make([][]byte, meta.NumChunks), // Will be populated during download
	}

	// Log selected snapshot (safely handle hash length)
	hashStr := bestKey.Hash
	if len(hashStr) > 16 {
		hashStr = hashStr[:16]
	}

	ssp.logger.Info("selected snapshot",
		"height", maxHeight,
		"hash", hashStr,
		"peer_agreement", fmt.Sprintf("%.1f%%", maxAgreement*100),
		"agreeing_peers", len(bestGroup.offers),
		"total_peers", len(offers))

	return fullSnapshot, nil
}

// verifySnapshot verifies snapshot against trusted state and BFT proofs
func (ssp *StateSyncProtocol) verifySnapshot(snap *snapshot.Snapshot) error {
	ssp.logger.Info("verifying snapshot", "height", snap.Height)

	// 1. Validate snapshot structure
	if err := snap.Validate(); err != nil {
		return fmt.Errorf("snapshot validation failed: %w", err)
	}

	// 2. Check if snapshot is within trust period
	if ssp.config.TrustHeight > 0 {
		if snap.Height < ssp.config.TrustHeight {
			return fmt.Errorf("snapshot height %d below trust height %d",
				snap.Height, ssp.config.TrustHeight)
		}
	}

	// 3. Verify BFT proof (2/3+ validator signatures) if required
	if ssp.config.RequireBFTProof {
		if !snap.IsTrusted() {
			return fmt.Errorf("snapshot lacks sufficient validator signatures")
		}
	}

	// 4. Verify snapshot comes from correct chain
	// Note: Chain ID verification can be added here if needed
	// For now, we rely on peer agreement and BFT proofs

	ssp.logger.Info("snapshot verification passed", "height", snap.Height)
	return nil
}

func (ssp *StateSyncProtocol) getPeerReliability(peerID string) float64 {
	if ssp.peerManager == nil {
		return defaultReliabilityScore
	}

	return normalizeReliability(ssp.peerManager.GetPeerReliability(peerID))
}

func normalizeReliability(value float64) float64 {
	if value <= 0 {
		return defaultReliabilityScore
	}

	if value < minReliabilityScore {
		return minReliabilityScore
	}

	if value > 1 {
		return 1
	}

	return value
}
