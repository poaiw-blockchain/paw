package protocol

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"cosmossdk.io/log"
)

// SyncConfig configures sync protocol behavior
type SyncConfig struct {
	// Sync parameters
	MaxBlocksPerRequest int
	MaxPeersPerSync     int
	SyncTimeout         time.Duration
	BlockRequestTimeout time.Duration
	StateRequestTimeout time.Duration

	// Catchup parameters
	CatchupBatchSize     int
	CatchupConcurrency   int
	CatchupRetryAttempts int
	CatchupRetryDelay    time.Duration

	// Performance
	MaxConcurrentRequests int
	PipelineDepth         int

	// Validation
	ValidateBlockHeaders bool
	ValidateStateProofs  bool
}

// DefaultSyncConfig returns default sync configuration
func DefaultSyncConfig() SyncConfig {
	return SyncConfig{
		MaxBlocksPerRequest:   500,
		MaxPeersPerSync:       5,
		SyncTimeout:           30 * time.Second,
		BlockRequestTimeout:   10 * time.Second,
		StateRequestTimeout:   15 * time.Second,
		CatchupBatchSize:      100,
		CatchupConcurrency:    3,
		CatchupRetryAttempts:  3,
		CatchupRetryDelay:     2 * time.Second,
		MaxConcurrentRequests: 10,
		PipelineDepth:         5,
		ValidateBlockHeaders:  true,
		ValidateStateProofs:   true,
	}
}

// SyncProtocol manages blockchain synchronization
type SyncProtocol struct {
	config SyncConfig
	logger log.Logger

	// Current sync state
	state         SyncState
	currentHeight int64
	targetHeight  int64
	isSyncing     bool
	stateMu       sync.RWMutex

	// Peer sync status
	peerStatus   map[string]*PeerSyncStatus
	peerStatusMu sync.RWMutex

	// Block requests
	pendingRequests   map[int64]*BlockRequest
	pendingRequestsMu sync.RWMutex

	// Callbacks
	getLocalHeight     func() (int64, error)
	getLocalBlock      func(height int64) ([]byte, error)
	validateBlock      func(height int64, data []byte) error
	applyBlock         func(height int64, data []byte) error
	getStateSnapshot   func(height int64) ([]byte, error)
	applyStateSnapshot func(height int64, data []byte) error

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	metrics *SyncMetrics
}

// SyncState represents the current sync state
type SyncState int

const (
	SyncStateIdle SyncState = iota
	SyncStateDiscovery
	SyncStateBlockSync
	SyncStateStateSync
	SyncStateCatchup
	SyncStateComplete
)

func (s SyncState) String() string {
	switch s {
	case SyncStateIdle:
		return "Idle"
	case SyncStateDiscovery:
		return "Discovery"
	case SyncStateBlockSync:
		return "BlockSync"
	case SyncStateStateSync:
		return "StateSync"
	case SyncStateCatchup:
		return "Catchup"
	case SyncStateComplete:
		return "Complete"
	default:
		return "Unknown"
	}
}

// PeerSyncStatus tracks sync status for a peer
type PeerSyncStatus struct {
	PeerID         string
	Height         int64
	LastUpdate     time.Time
	BestHash       []byte
	IsSyncing      bool
	Reliability    float64 // 0-1
	RequestsSent   int
	RequestsFailed int
}

// BlockRequest represents a pending block request
type BlockRequest struct {
	Height      int64
	PeerID      string
	RequestTime time.Time
	Retries     int
	Done        chan []byte
	Error       chan error
}

// SyncMetrics tracks sync statistics
type SyncMetrics struct {
	BlocksSynced         int64
	BlocksValidated      int64
	BlocksApplied        int64
	SyncErrors           int64
	StateSnapshotsSynced int64
	TotalSyncTime        time.Duration
	LastSyncTime         time.Time
	mu                   sync.RWMutex
}

// NewSyncProtocol creates a new sync protocol instance
func NewSyncProtocol(config SyncConfig, logger log.Logger) *SyncProtocol {
	ctx, cancel := context.WithCancel(context.Background())

	sp := &SyncProtocol{
		config:          config,
		logger:          logger,
		state:           SyncStateIdle,
		peerStatus:      make(map[string]*PeerSyncStatus),
		pendingRequests: make(map[int64]*BlockRequest),
		ctx:             ctx,
		cancel:          cancel,
		metrics:         &SyncMetrics{},
	}

	return sp
}

// StartSync initiates blockchain synchronization
func (sp *SyncProtocol) StartSync(targetHeight int64) error {
	sp.stateMu.Lock()
	defer sp.stateMu.Unlock()

	if sp.isSyncing {
		return errors.New("sync already in progress")
	}

	localHeight, err := sp.getLocalHeight()
	if err != nil {
		return fmt.Errorf("failed to get local height: %w", err)
	}

	if targetHeight <= localHeight {
		sp.logger.Info("already at target height", "local", localHeight, "target", targetHeight)
		return nil
	}

	sp.currentHeight = localHeight
	sp.targetHeight = targetHeight
	sp.isSyncing = true
	sp.state = SyncStateDiscovery

	sp.logger.Info("starting sync",
		"current_height", localHeight,
		"target_height", targetHeight,
		"blocks_to_sync", targetHeight-localHeight,
	)

	// Start sync worker
	sp.wg.Add(1)
	go sp.syncWorker()

	return nil
}

// StopSync stops ongoing synchronization
func (sp *SyncProtocol) StopSync() {
	sp.stateMu.Lock()
	defer sp.stateMu.Unlock()

	if !sp.isSyncing {
		return
	}

	sp.logger.Info("stopping sync")
	sp.isSyncing = false
	sp.state = SyncStateIdle
}

// UpdatePeerStatus updates sync status for a peer
func (sp *SyncProtocol) UpdatePeerStatus(peerID string, height int64, bestHash []byte, isSyncing bool) {
	sp.peerStatusMu.Lock()
	defer sp.peerStatusMu.Unlock()

	status, exists := sp.peerStatus[peerID]
	if !exists {
		status = &PeerSyncStatus{
			PeerID:      peerID,
			Reliability: 1.0,
		}
		sp.peerStatus[peerID] = status
	}

	status.Height = height
	status.BestHash = bestHash
	status.IsSyncing = isSyncing
	status.LastUpdate = time.Now()
}

// RequestBlocks requests blocks from a peer
func (sp *SyncProtocol) RequestBlocks(peerID string, fromHeight, toHeight int64) error {
	if toHeight < fromHeight {
		return errors.New("invalid height range")
	}

	blockCount := toHeight - fromHeight + 1
	if blockCount > int64(sp.config.MaxBlocksPerRequest) {
		return fmt.Errorf("too many blocks requested: %d", blockCount)
	}

	sp.logger.Debug("requesting blocks",
		"peer_id", peerID,
		"from", fromHeight,
		"to", toHeight,
		"count", blockCount,
	)

	// Create requests for each block
	for height := fromHeight; height <= toHeight; height++ {
		sp.createBlockRequest(peerID, height)
	}

	return nil
}

// RequestState requests state snapshot from a peer
func (sp *SyncProtocol) RequestState(peerID string, height int64) error {
	sp.logger.Info("requesting state snapshot", "peer_id", peerID, "height", height)

	// State sync implementation would go here
	// For now, return not implemented
	return errors.New("state sync not implemented")
}

// HandleBlockResponse processes a block response
func (sp *SyncProtocol) HandleBlockResponse(peerID string, height int64, blockData []byte) error {
	sp.pendingRequestsMu.Lock()
	req, exists := sp.pendingRequests[height]
	sp.pendingRequestsMu.Unlock()

	if !exists {
		sp.logger.Warn("received unrequested block", "peer_id", peerID, "height", height)
		return errors.New("unrequested block")
	}

	if req.PeerID != peerID {
		sp.logger.Warn("block from wrong peer",
			"expected", req.PeerID,
			"got", peerID,
			"height", height,
		)
		return errors.New("block from wrong peer")
	}

	// Validate block if configured
	if sp.config.ValidateBlockHeaders && sp.validateBlock != nil {
		if err := sp.validateBlock(height, blockData); err != nil {
			sp.logger.Error("block validation failed",
				"peer_id", peerID,
				"height", height,
				"error", err,
			)

			// Update peer reliability
			sp.updatePeerReliability(peerID, false)

			select {
			case req.Error <- err:
			default:
			}

			return err
		}
	}

	// Send block data to requester
	select {
	case req.Done <- blockData:
		sp.logger.Debug("block received", "peer_id", peerID, "height", height)
		sp.updatePeerReliability(peerID, true)
	case <-time.After(time.Second):
		sp.logger.Warn("block response channel timeout", "height", height)
	}

	// Clean up request
	sp.pendingRequestsMu.Lock()
	delete(sp.pendingRequests, height)
	sp.pendingRequestsMu.Unlock()

	return nil
}

// syncWorker performs the actual synchronization
func (sp *SyncProtocol) syncWorker() {
	defer sp.wg.Done()

	startTime := time.Now()

	for {
		sp.stateMu.RLock()
		isSyncing := sp.isSyncing
		currentHeight := sp.currentHeight
		targetHeight := sp.targetHeight
		sp.stateMu.RUnlock()

		if !isSyncing {
			break
		}

		if currentHeight >= targetHeight {
			sp.completeSyn()
			break
		}

		// Select best peers for sync
		peers := sp.selectSyncPeers()
		if len(peers) == 0 {
			sp.logger.Warn("no peers available for sync")
			time.Sleep(5 * time.Second)
			continue
		}

		// Determine sync strategy
		blocksRemaining := targetHeight - currentHeight
		if blocksRemaining > int64(sp.config.CatchupBatchSize)*10 {
			// Use state sync for large gaps (if available)
			sp.stateMu.Lock()
			sp.state = SyncStateStateSync
			sp.stateMu.Unlock()

			if err := sp.performStateSync(peers); err != nil {
				sp.logger.Error("state sync failed", "error", err)
				// Fall back to block sync
			}
		}

		// Perform block sync
		sp.stateMu.Lock()
		sp.state = SyncStateBlockSync
		sp.stateMu.Unlock()

		if err := sp.performBlockSync(peers); err != nil {
			sp.logger.Error("block sync failed", "error", err)
			sp.recordSyncError()
			time.Sleep(sp.config.CatchupRetryDelay)
		}
	}

	sp.metrics.mu.Lock()
	sp.metrics.TotalSyncTime = time.Since(startTime)
	sp.metrics.LastSyncTime = time.Now()
	sp.metrics.mu.Unlock()
}

// performBlockSync syncs blocks incrementally
func (sp *SyncProtocol) performBlockSync(peers []*PeerSyncStatus) error {
	sp.stateMu.RLock()
	currentHeight := sp.currentHeight
	targetHeight := sp.targetHeight
	sp.stateMu.RUnlock()

	batchSize := sp.config.CatchupBatchSize
	fromHeight := currentHeight + 1
	toHeight := fromHeight + int64(batchSize) - 1

	if toHeight > targetHeight {
		toHeight = targetHeight
	}

	sp.logger.Info("syncing blocks",
		"from", fromHeight,
		"to", toHeight,
		"peers", len(peers),
	)

	// Request blocks from peers
	peerIndex := 0
	for height := fromHeight; height <= toHeight; height++ {
		peer := peers[peerIndex%len(peers)]
		peerIndex++

		req := sp.createBlockRequest(peer.PeerID, height)

		// Wait for block or timeout
		select {
		case blockData := <-req.Done:
			if err := sp.applyBlock(height, blockData); err != nil {
				return fmt.Errorf("failed to apply block %d: %w", height, err)
			}

			sp.stateMu.Lock()
			sp.currentHeight = height
			sp.stateMu.Unlock()

			sp.recordBlockSynced()

		case err := <-req.Error:
			return fmt.Errorf("block request failed for height %d: %w", height, err)

		case <-time.After(sp.config.BlockRequestTimeout):
			sp.logger.Warn("block request timeout", "height", height, "peer", peer.PeerID)
			sp.updatePeerReliability(peer.PeerID, false)

			// Retry with different peer
			if req.Retries < sp.config.CatchupRetryAttempts {
				req.Retries++
				sp.logger.Debug("retrying block request", "height", height, "attempt", req.Retries)
				height-- // Retry same height
			} else {
				return fmt.Errorf("max retries exceeded for block %d", height)
			}

		case <-sp.ctx.Done():
			return errors.New("sync cancelled")
		}
	}

	return nil
}

// performStateSync syncs using state snapshots
func (sp *SyncProtocol) performStateSync(peers []*PeerSyncStatus) error {
	sp.logger.Info("state sync not yet implemented")
	return errors.New("state sync not implemented")
}

// selectSyncPeers selects the best peers for synchronization
func (sp *SyncProtocol) selectSyncPeers() []*PeerSyncStatus {
	sp.peerStatusMu.RLock()
	defer sp.peerStatusMu.RUnlock()

	sp.stateMu.RLock()
	targetHeight := sp.targetHeight
	sp.stateMu.RUnlock()

	var eligible []*PeerSyncStatus
	now := time.Now()

	for _, status := range sp.peerStatus {
		// Skip if peer doesn't have the blocks we need
		if status.Height < targetHeight {
			continue
		}

		// Skip if status is stale
		if now.Sub(status.LastUpdate) > 30*time.Second {
			continue
		}

		// Skip unreliable peers
		if status.Reliability < 0.5 {
			continue
		}

		eligible = append(eligible, status)
	}

	// Sort by reliability (simple bubble sort)
	for i := 0; i < len(eligible)-1; i++ {
		for j := 0; j < len(eligible)-i-1; j++ {
			if eligible[j].Reliability < eligible[j+1].Reliability {
				eligible[j], eligible[j+1] = eligible[j+1], eligible[j]
			}
		}
	}

	// Return top N peers
	maxPeers := sp.config.MaxPeersPerSync
	if len(eligible) > maxPeers {
		eligible = eligible[:maxPeers]
	}

	return eligible
}

// createBlockRequest creates a new block request
func (sp *SyncProtocol) createBlockRequest(peerID string, height int64) *BlockRequest {
	req := &BlockRequest{
		Height:      height,
		PeerID:      peerID,
		RequestTime: time.Now(),
		Retries:     0,
		Done:        make(chan []byte, 1),
		Error:       make(chan error, 1),
	}

	sp.pendingRequestsMu.Lock()
	sp.pendingRequests[height] = req
	sp.pendingRequestsMu.Unlock()

	// Update peer stats
	sp.peerStatusMu.Lock()
	if status, exists := sp.peerStatus[peerID]; exists {
		status.RequestsSent++
	}
	sp.peerStatusMu.Unlock()

	return req
}

// updatePeerReliability updates peer reliability score
func (sp *SyncProtocol) updatePeerReliability(peerID string, success bool) {
	sp.peerStatusMu.Lock()
	defer sp.peerStatusMu.Unlock()

	status, exists := sp.peerStatus[peerID]
	if !exists {
		return
	}

	if success {
		// Increase reliability (up to 1.0)
		status.Reliability = (status.Reliability + 1.0) / 2.0
		if status.Reliability > 1.0 {
			status.Reliability = 1.0
		}
	} else {
		// Decrease reliability
		status.Reliability *= 0.8
		status.RequestsFailed++
	}
}

// completeSyn marks synchronization as complete
func (sp *SyncProtocol) completeSyn() {
	sp.stateMu.Lock()
	defer sp.stateMu.Unlock()

	sp.state = SyncStateComplete
	sp.isSyncing = false

	sp.logger.Info("synchronization complete",
		"final_height", sp.currentHeight,
		"blocks_synced", sp.metrics.BlocksSynced,
		"sync_time", sp.metrics.TotalSyncTime,
	)
}

// Metrics recording

func (sp *SyncProtocol) recordBlockSynced() {
	sp.metrics.mu.Lock()
	defer sp.metrics.mu.Unlock()
	sp.metrics.BlocksSynced++
	sp.metrics.BlocksValidated++
	sp.metrics.BlocksApplied++
}

func (sp *SyncProtocol) recordSyncError() {
	sp.metrics.mu.Lock()
	defer sp.metrics.mu.Unlock()
	sp.metrics.SyncErrors++
}

// Getters

func (sp *SyncProtocol) IsSyncing() bool {
	sp.stateMu.RLock()
	defer sp.stateMu.RUnlock()
	return sp.isSyncing
}

func (sp *SyncProtocol) GetSyncState() SyncState {
	sp.stateMu.RLock()
	defer sp.stateMu.RUnlock()
	return sp.state
}

func (sp *SyncProtocol) GetCurrentHeight() int64 {
	sp.stateMu.RLock()
	defer sp.stateMu.RUnlock()
	return sp.currentHeight
}

func (sp *SyncProtocol) GetTargetHeight() int64 {
	sp.stateMu.RLock()
	defer sp.stateMu.RUnlock()
	return sp.targetHeight
}

func (sp *SyncProtocol) GetMetrics() SyncMetrics {
	sp.metrics.mu.RLock()
	defer sp.metrics.mu.RUnlock()
	return SyncMetrics{
		BlocksSynced:         sp.metrics.BlocksSynced,
		BlocksValidated:      sp.metrics.BlocksValidated,
		BlocksApplied:        sp.metrics.BlocksApplied,
		SyncErrors:           sp.metrics.SyncErrors,
		StateSnapshotsSynced: sp.metrics.StateSnapshotsSynced,
		TotalSyncTime:        sp.metrics.TotalSyncTime,
		LastSyncTime:         sp.metrics.LastSyncTime,
	}
}

// Callback setters

func (sp *SyncProtocol) SetGetLocalHeightCallback(fn func() (int64, error)) {
	sp.getLocalHeight = fn
}

func (sp *SyncProtocol) SetGetLocalBlockCallback(fn func(int64) ([]byte, error)) {
	sp.getLocalBlock = fn
}

func (sp *SyncProtocol) SetValidateBlockCallback(fn func(int64, []byte) error) {
	sp.validateBlock = fn
}

func (sp *SyncProtocol) SetApplyBlockCallback(fn func(int64, []byte) error) {
	sp.applyBlock = fn
}

func (sp *SyncProtocol) SetGetStateSnapshotCallback(fn func(int64) ([]byte, error)) {
	sp.getStateSnapshot = fn
}

func (sp *SyncProtocol) SetApplyStateSnapshotCallback(fn func(int64, []byte) error) {
	sp.applyStateSnapshot = fn
}

// Stop stops the sync protocol
func (sp *SyncProtocol) Stop() {
	sp.logger.Info("stopping sync protocol")
	sp.cancel()
	sp.wg.Wait()
	sp.logger.Info("sync protocol stopped")
}
