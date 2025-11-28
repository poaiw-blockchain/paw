package protocol

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/paw-chain/paw/p2p/snapshot"
)

// downloadSnapshotChunks downloads all snapshot chunks in parallel with Byzantine fault tolerance
func (ssp *StateSyncProtocol) downloadSnapshotChunks(ctx context.Context, snap *snapshot.Snapshot) error {
	ssp.logger.Info("downloading snapshot chunks",
		"height", snap.Height,
		"num_chunks", snap.NumChunks,
		"fetchers", ssp.config.ChunkFetchers)

	numChunks := snap.NumChunks

	// Initialize downloaded chunks map
	ssp.chunksMu.Lock()
	ssp.downloadedChunks = make(map[uint32]bool, numChunks)
	ssp.chunksMu.Unlock()

	// Create work queue
	chunkQueue := make(chan uint32, numChunks)
	resultChan := make(chan *chunkResult, numChunks)
	errorChan := make(chan error, 1)

	// Fill chunk queue
	for i := uint32(0); i < numChunks; i++ {
		chunkQueue <- i
	}
	close(chunkQueue)

	// Start chunk fetchers
	var wg sync.WaitGroup
	fetcherCtx, fetcherCancel := context.WithCancel(ctx)
	defer fetcherCancel()

	for i := uint32(0); i < ssp.config.ChunkFetchers; i++ {
		wg.Add(1)
		go ssp.chunkFetcher(fetcherCtx, &wg, snap, chunkQueue, resultChan, errorChan)
	}

	// Wait for all chunks to be downloaded
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	downloadedCount := uint32(0)
	chunks := make(map[uint32]*snapshot.SnapshotChunk)

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				// All chunks downloaded
				if downloadedCount == numChunks {
					ssp.logger.Info("all chunks downloaded successfully",
						"count", downloadedCount,
						"bytes", atomic.LoadUint64(&ssp.bytesReceived))

					// Store chunks for application
					ssp.storeChunks(snap, chunks)
					return nil
				}

				return fmt.Errorf("download incomplete: %d/%d chunks",
					downloadedCount, numChunks)
			}

			if result.err != nil {
				fetcherCancel()
				return fmt.Errorf("chunk download failed: %w", result.err)
			}

			// Store chunk
			chunks[result.chunk.Index] = result.chunk
			downloadedCount++

			// Update metrics
			atomic.AddInt64(&ssp.metrics.ChunksDownloaded, 1)
			atomic.AddInt64(&ssp.metrics.BytesDownloaded, int64(len(result.chunk.Data)))

			// Mark as downloaded
			ssp.chunksMu.Lock()
			ssp.downloadedChunks[result.chunk.Index] = true
			ssp.chunksMu.Unlock()

			// Log progress
			if downloadedCount%10 == 0 || downloadedCount == numChunks {
				progress := float64(downloadedCount) / float64(numChunks) * 100
				ssp.logger.Info("download progress",
					"chunks", fmt.Sprintf("%d/%d", downloadedCount, numChunks),
					"progress", fmt.Sprintf("%.1f%%", progress),
					"bytes_mb", atomic.LoadUint64(&ssp.bytesReceived)/(1024*1024))
			}

		case err := <-errorChan:
			fetcherCancel()
			return fmt.Errorf("fatal download error: %w", err)

		case <-ctx.Done():
			fetcherCancel()
			return ctx.Err()
		}
	}
}

// chunkResult represents the result of a chunk download
type chunkResult struct {
	chunk *snapshot.SnapshotChunk
	err   error
}

// chunkFetcher downloads chunks from peers with retries and verification
func (ssp *StateSyncProtocol) chunkFetcher(
	ctx context.Context,
	wg *sync.WaitGroup,
	snap *snapshot.Snapshot,
	chunkQueue <-chan uint32,
	resultChan chan<- *chunkResult,
	errorChan chan<- error,
) {
	defer wg.Done()

	for {
		select {
		case chunkIndex, ok := <-chunkQueue:
			if !ok {
				return // No more chunks
			}

			// Download chunk with retries
			chunk, err := ssp.downloadChunkWithRetry(ctx, snap, chunkIndex)
			if err != nil {
				// Fatal error
				select {
				case errorChan <- fmt.Errorf("failed to download chunk %d: %w", chunkIndex, err):
				default:
				}
				return
			}

			// Verify chunk if configured
			if ssp.config.VerifyAllChunks {
				if err := ssp.verifyChunk(chunk, snap, chunkIndex); err != nil {
					select {
					case errorChan <- fmt.Errorf("chunk %d verification failed: %w", chunkIndex, err):
					default:
					}
					return
				}

				atomic.AddInt64(&ssp.metrics.ChunksVerified, 1)
			}

			// Send result
			select {
			case resultChan <- &chunkResult{chunk: chunk}:
			case <-ctx.Done():
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// downloadChunkWithRetry downloads a chunk with retry logic
func (ssp *StateSyncProtocol) downloadChunkWithRetry(
	ctx context.Context,
	snap *snapshot.Snapshot,
	chunkIndex uint32,
) (*snapshot.SnapshotChunk, error) {
	var lastErr error

	for attempt := 0; attempt < ssp.config.ChunkRetryAttempts; attempt++ {
		if attempt > 0 {
			ssp.logger.Debug("retrying chunk download",
				"chunk", chunkIndex,
				"attempt", attempt+1)

			// Exponential backoff
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// Select peer for download
		peerID, err := ssp.selectPeerForChunk(chunkIndex)
		if err != nil {
			lastErr = err
			continue
		}

		// Download chunk from peer
		chunk, err := ssp.downloadChunkFromPeer(ctx, peerID, snap, chunkIndex)
		if err != nil {
			lastErr = err
			ssp.logger.Warn("chunk download failed",
				"chunk", chunkIndex,
				"peer", peerID,
				"error", err)

			// Report potentially malicious peer
			ssp.reportSuspiciousPeer(peerID, "chunk download failed")
			continue
		}

		// Success
		atomic.AddUint64(&ssp.chunksDownloaded, 1)
		atomic.AddUint64(&ssp.bytesReceived, uint64(len(chunk.Data)))

		return chunk, nil
	}

	return nil, fmt.Errorf("max retries exceeded for chunk %d: %w", chunkIndex, lastErr)
}

// downloadChunkFromPeer downloads a specific chunk from a peer
func (ssp *StateSyncProtocol) downloadChunkFromPeer(
	ctx context.Context,
	peerID string,
	snap *snapshot.Snapshot,
	chunkIndex uint32,
) (*snapshot.SnapshotChunk, error) {
	// Set request timeout
	reqCtx, cancel := context.WithTimeout(ctx, ssp.config.ChunkRequestTimeout)
	defer cancel()

	// Request chunk from peer
	data, err := ssp.peerManager.RequestChunk(reqCtx, peerID, snap.Height, chunkIndex)
	if err != nil {
		return nil, fmt.Errorf("peer request failed: %w", err)
	}

	// Create chunk
	chunk := &snapshot.SnapshotChunk{
		Height: snap.Height,
		Index:  chunkIndex,
		Data:   data,
		Hash:   snapshot.HashData(data),
	}

	return chunk, nil
}

// selectPeerForChunk selects the best peer for downloading a chunk
func (ssp *StateSyncProtocol) selectPeerForChunk(chunkIndex uint32) (string, error) {
	ssp.peerOffersMu.RLock()
	defer ssp.peerOffersMu.RUnlock()

	if len(ssp.peerOffers) == 0 {
		return "", fmt.Errorf("no peers available")
	}

	// Get selected snapshot details
	ssp.stateMu.RLock()
	selectedHeight := ssp.selectedSnapshot.Height
	selectedHash := fmt.Sprintf("%x", ssp.selectedSnapshot.Hash)
	ssp.stateMu.RUnlock()

	// Find peers that offered this snapshot
	var validPeers []string

	for peerID, offer := range ssp.peerOffers {
		// Skip malicious peers
		ssp.maliciousPeersMu.RLock()
		isMalicious := ssp.maliciousPeers[peerID]
		ssp.maliciousPeersMu.RUnlock()

		if isMalicious {
			continue
		}

		// Check if peer has the right snapshot
		if offer.Snapshot.Height == selectedHeight {
			offerHash := fmt.Sprintf("%x", offer.Snapshot.Hash)
			if offerHash == selectedHash {
				validPeers = append(validPeers, peerID)
			}
		}
	}

	if len(validPeers) == 0 {
		return "", fmt.Errorf("no valid peers for chunk %d", chunkIndex)
	}

	// Simple round-robin selection (could be improved with load balancing)
	peerIndex := int(chunkIndex) % len(validPeers)
	return validPeers[peerIndex], nil
}

// verifyChunk verifies a downloaded chunk
func (ssp *StateSyncProtocol) verifyChunk(
	chunk *snapshot.SnapshotChunk,
	snap *snapshot.Snapshot,
	chunkIndex uint32,
) error {
	// Validate chunk structure
	if err := chunk.Validate(); err != nil {
		return fmt.Errorf("chunk validation failed: %w", err)
	}

	// Verify chunk index
	if chunk.Index != chunkIndex {
		return fmt.Errorf("chunk index mismatch: expected %d, got %d",
			chunkIndex, chunk.Index)
	}

	// Verify chunk height
	if chunk.Height != snap.Height {
		return fmt.Errorf("chunk height mismatch: expected %d, got %d",
			snap.Height, chunk.Height)
	}

	// Verify chunk hash against snapshot metadata (if available)
	if chunkIndex < uint32(len(snap.ChunkHashes)) && snap.ChunkHashes[chunkIndex] != nil {
		expectedHash := snap.ChunkHashes[chunkIndex]
		if !bytesEqual(chunk.Hash, expectedHash) {
			return fmt.Errorf("chunk hash mismatch")
		}
	}

	return nil
}

// storeChunks stores downloaded chunks for later application
func (ssp *StateSyncProtocol) storeChunks(snap *snapshot.Snapshot, chunks map[uint32]*snapshot.SnapshotChunk) {
	mgr := ssp.getSnapshotManager()
	if mgr == nil {
		return
	}

	// Store each chunk
	for i := uint32(0); i < snap.NumChunks; i++ {
		chunk, exists := chunks[i]
		if !exists {
			ssp.logger.Error("missing chunk during storage", "index", i)
			continue
		}

		// Update snapshot chunk hash if not set
		if i < uint32(len(snap.ChunkHashes)) && snap.ChunkHashes[i] == nil {
			snap.ChunkHashes[i] = chunk.Hash
		}
	}
}

// applySnapshot reconstructs state from downloaded chunks
func (ssp *StateSyncProtocol) applySnapshot(snap *snapshot.Snapshot) ([]byte, error) {
	ssp.logger.Info("applying snapshot", "height", snap.Height)

	mgr := ssp.getSnapshotManager()
	if mgr == nil {
		return nil, fmt.Errorf("snapshot manager not available")
	}

	// Restore state from snapshot
	stateData, err := mgr.RestoreFromSnapshot(snap)
	if err != nil {
		return nil, fmt.Errorf("failed to restore from snapshot: %w", err)
	}

	// Apply state using callback
	if ssp.applySnapshotCallback != nil {
		if err := ssp.applySnapshotCallback(snap.Height, stateData); err != nil {
			return nil, fmt.Errorf("failed to apply state: %w", err)
		}
	}

	ssp.logger.Info("snapshot applied successfully",
		"height", snap.Height,
		"size_mb", len(stateData)/(1024*1024))

	return stateData, nil
}

// verifyAppliedState verifies the applied state matches the snapshot
func (ssp *StateSyncProtocol) verifyAppliedState(snap *snapshot.Snapshot, stateData []byte) error {
	ssp.logger.Info("verifying applied state", "height", snap.Height)

	// Verify state data hash
	computedHash := snapshot.HashData(stateData)
	if !bytesEqual(computedHash, snap.Hash) {
		return fmt.Errorf("state hash mismatch after application")
	}

	// Verify app hash using callback
	if ssp.verifyStateCallback != nil {
		if err := ssp.verifyStateCallback(snap.Height, snap.AppHash); err != nil {
			return fmt.Errorf("app hash verification failed: %w", err)
		}
	}

	ssp.logger.Info("state verification passed", "height", snap.Height)
	return nil
}

// getSnapshotManager safely gets the snapshot manager
func (ssp *StateSyncProtocol) getSnapshotManager() *snapshot.Manager {
	if ssp.snapshotMgr == nil {
		return nil
	}
	return ssp.snapshotMgr
}

// reportSuspiciousPeer reports a peer that may be malicious
func (ssp *StateSyncProtocol) reportSuspiciousPeer(peerID string, reason string) {
	ssp.maliciousPeersMu.Lock()
	defer ssp.maliciousPeersMu.Unlock()

	// Track malicious behavior
	if !ssp.maliciousPeers[peerID] {
		ssp.maliciousPeers[peerID] = true
		atomic.AddInt64(&ssp.metrics.MaliciousPeersFound, 1)

		ssp.logger.Warn("peer marked as suspicious",
			"peer_id", peerID,
			"reason", reason)

		// Report to peer manager
		ssp.peerManager.ReportMaliciousPeer(peerID, reason)
	}

	// Check if too many malicious peers
	if len(ssp.maliciousPeers) >= ssp.config.MaxMaliciousPeers {
		ssp.logger.Error("too many malicious peers detected",
			"count", len(ssp.maliciousPeers),
			"max", ssp.config.MaxMaliciousPeers)
	}
}

// setState updates the state sync state
func (ssp *StateSyncProtocol) setState(state StateSyncState) {
	ssp.stateMu.Lock()
	defer ssp.stateMu.Unlock()

	ssp.state = state
	ssp.logger.Debug("state sync state changed", "state", state.String())
}

// GetState returns the current state sync state
func (ssp *StateSyncProtocol) GetState() StateSyncState {
	ssp.stateMu.RLock()
	defer ssp.stateMu.RUnlock()

	return ssp.state
}

// GetProgress returns download progress
func (ssp *StateSyncProtocol) GetProgress() (downloaded, total uint32) {
	ssp.chunksMu.RLock()
	defer ssp.chunksMu.RUnlock()

	if ssp.selectedSnapshot == nil {
		return 0, 0
	}

	return uint32(len(ssp.downloadedChunks)), ssp.selectedSnapshot.NumChunks
}

// GetMetrics returns state sync metrics
func (ssp *StateSyncProtocol) GetMetrics() StateSyncMetrics {
	ssp.metrics.mu.RLock()
	defer ssp.metrics.mu.RUnlock()

	return StateSyncMetrics{
		SnapshotsDiscovered: ssp.metrics.SnapshotsDiscovered,
		ChunksDownloaded:    ssp.metrics.ChunksDownloaded,
		ChunksVerified:      ssp.metrics.ChunksVerified,
		BytesDownloaded:     ssp.metrics.BytesDownloaded,
		PeersQueried:        ssp.metrics.PeersQueried,
		MaliciousPeersFound: ssp.metrics.MaliciousPeersFound,
		DownloadTime:        ssp.metrics.DownloadTime,
		VerificationTime:    ssp.metrics.VerificationTime,
		TotalTime:           ssp.metrics.TotalTime,
	}
}

// SetApplySnapshotCallback sets the callback for applying snapshots
func (ssp *StateSyncProtocol) SetApplySnapshotCallback(fn func(int64, []byte) error) {
	ssp.applySnapshotCallback = fn
}

// SetVerifyStateCallback sets the callback for verifying state
func (ssp *StateSyncProtocol) SetVerifyStateCallback(fn func(int64, []byte) error) {
	ssp.verifyStateCallback = fn
}

// Stop stops the state sync protocol
func (ssp *StateSyncProtocol) Stop() {
	ssp.logger.Info("stopping state sync protocol")
	ssp.cancel()
	ssp.wg.Wait()
}

// Helper function
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
