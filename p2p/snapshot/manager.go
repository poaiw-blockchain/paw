package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"cosmossdk.io/log"
)

// Manager handles snapshot creation, storage, and retrieval
const maxUint32Value = ^uint32(0)

func intToUint32(field string, value int) (uint32, error) {
	if value < 0 || value > int(maxUint32Value) {
		return 0, fmt.Errorf("%s %d exceeds uint32 range", field, value)
	}
	return uint32(value), nil
}

type Manager struct {
	config      *ManagerConfig
	logger      log.Logger
	snapshotDir string
	mu          sync.RWMutex

	// Snapshot tracking
	snapshots      map[int64]*Snapshot
	latestSnapshot *Snapshot
}

// ManagerConfig configures the snapshot manager
type ManagerConfig struct {
	// Snapshot directory
	SnapshotDir string

	// Snapshot creation
	SnapshotInterval   uint64 // Blocks between snapshots (e.g., 1000)
	SnapshotKeepRecent uint32 // Number of recent snapshots to keep

	// Chunk settings
	ChunkSize uint32 // Bytes per chunk (default 16MB)

	// Pruning
	PruneOldSnapshots  bool
	MinSnapshotsToKeep uint32

	// Chain info
	ChainID string
}

// DefaultManagerConfig returns default manager configuration
func DefaultManagerConfig(dataDir string) *ManagerConfig {
	return &ManagerConfig{
		SnapshotDir:        filepath.Join(dataDir, "snapshots"),
		SnapshotInterval:   1000,
		SnapshotKeepRecent: 10,
		ChunkSize:          DefaultChunkSize,
		PruneOldSnapshots:  true,
		MinSnapshotsToKeep: 2,
	}
}

// NewManager creates a new snapshot manager
func NewManager(config *ManagerConfig, logger log.Logger) (*Manager, error) {
	if config.SnapshotDir == "" {
		return nil, fmt.Errorf("snapshot directory not specified")
	}

	// Create snapshot directory
	if err := os.MkdirAll(config.SnapshotDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	// Create chunks subdirectory
	chunksDir := filepath.Join(config.SnapshotDir, "chunks")
	if err := os.MkdirAll(chunksDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create chunks directory: %w", err)
	}

	manager := &Manager{
		config:      config,
		logger:      logger,
		snapshotDir: config.SnapshotDir,
		snapshots:   make(map[int64]*Snapshot),
	}

	// Load existing snapshots
	if err := manager.loadSnapshots(); err != nil {
		logger.Error("failed to load existing snapshots", "error", err)
	}

	logger.Info("snapshot manager initialized",
		"dir", config.SnapshotDir,
		"interval", config.SnapshotInterval,
		"keep_recent", config.SnapshotKeepRecent)

	return manager, nil
}

// CreateSnapshot creates a new snapshot at the given height
func (m *Manager) CreateSnapshot(height int64, stateData []byte, appHash, validatorHash, consensusHash []byte) (*Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("creating snapshot",
		"height", height,
		"data_size", len(stateData))

	startTime := time.Now()

	// Check if snapshot already exists
	if _, exists := m.snapshots[height]; exists {
		return nil, fmt.Errorf("snapshot already exists at height %d", height)
	}

	// Split state data into chunks
	chunkSize := int(m.config.ChunkSize)
	chunks := SplitIntoChunks(stateData, chunkSize)

	// Calculate chunk hashes
	chunkHashes := make([][]byte, len(chunks))
	for i, chunk := range chunks {
		chunkHashes[i] = HashData(chunk)
	}

	chunkCount := len(chunks)
	numChunks, err := intToUint32("chunk count", chunkCount)
	if err != nil {
		return nil, err
	}

	// Create snapshot metadata
	snapshot := &Snapshot{
		Height:        height,
		Hash:          HashData(stateData),
		Timestamp:     time.Now().Unix(),
		Format:        SnapshotFormatV1,
		ChainID:       m.config.ChainID,
		NumChunks:     numChunks,
		ChunkHashes:   chunkHashes,
		AppHash:       appHash,
		ValidatorHash: validatorHash,
		ConsensusHash: consensusHash,
		VotingPower:   0, // Will be set by validator signatures
		TotalPower:    0, // Will be set by validator signatures
	}

	// Validate snapshot
	if err := snapshot.Validate(); err != nil {
		return nil, fmt.Errorf("snapshot validation failed: %w", err)
	}

	// Save chunks to disk
	for i, chunk := range chunks {
		chunkIndex, err := intToUint32("chunk index", i)
		if err != nil {
			return nil, err
		}
		if err := m.saveChunk(snapshot, chunkIndex, chunk); err != nil {
			return nil, fmt.Errorf("failed to save chunk %d: %w", i, err)
		}
	}

	// Save snapshot metadata
	if err := m.saveMetadata(snapshot); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	// Track snapshot
	m.snapshots[height] = snapshot
	m.latestSnapshot = snapshot

	// Cleanup old snapshots if configured
	if m.config.PruneOldSnapshots {
		m.pruneOldSnapshots()
	}

	duration := time.Since(startTime)
	m.logger.Info("snapshot created successfully",
		"height", height,
		"num_chunks", len(chunks),
		"size_mb", len(stateData)/(1024*1024),
		"duration_ms", duration.Milliseconds())

	return snapshot, nil
}

// LoadSnapshot loads a snapshot at the given height
func (m *Manager) LoadSnapshot(height int64) (*Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check in-memory cache
	if snapshot, exists := m.snapshots[height]; exists {
		return snapshot, nil
	}

	// Load from disk
	metadataPath := m.metadataPath(height)
	data, err := m.readFileSafe(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot metadata: %w", err)
	}

	snapshot, err := DeserializeSnapshot(data)
	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// LoadChunk loads a specific chunk of a snapshot
func (m *Manager) LoadChunk(height int64, chunkIndex uint32) (*SnapshotChunk, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Load chunk from disk
	chunkPath := m.chunkPath(height, chunkIndex)
	data, err := m.readFileSafe(chunkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunk: %w", err)
	}

	chunk := &SnapshotChunk{
		Height: height,
		Index:  chunkIndex,
		Data:   data,
		Hash:   HashData(data),
	}

	// Validate chunk
	if err := chunk.Validate(); err != nil {
		return nil, fmt.Errorf("chunk validation failed: %w", err)
	}

	return chunk, nil
}

// GetLatestSnapshot returns the latest snapshot
func (m *Manager) GetLatestSnapshot() *Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.latestSnapshot
}

// GetSnapshots returns all available snapshots
func (m *Manager) GetSnapshots() []*Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshots := make([]*Snapshot, 0, len(m.snapshots))
	for _, snapshot := range m.snapshots {
		snapshots = append(snapshots, snapshot)
	}

	// Sort by height (descending)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Height > snapshots[j].Height
	})

	return snapshots
}

// GetSnapshotsInRange returns snapshots within a height range
func (m *Manager) GetSnapshotsInRange(minHeight, maxHeight int64) []*Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshots := make([]*Snapshot, 0)
	for height, snapshot := range m.snapshots {
		if height >= minHeight && height <= maxHeight {
			snapshots = append(snapshots, snapshot)
		}
	}

	// Sort by height (descending)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Height > snapshots[j].Height
	})

	return snapshots
}

// HasSnapshot checks if a snapshot exists at the given height
func (m *Manager) HasSnapshot(height int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.snapshots[height]
	return exists
}

// DeleteSnapshot deletes a snapshot
func (m *Manager) DeleteSnapshot(height int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot, exists := m.snapshots[height]
	if !exists {
		return fmt.Errorf("snapshot not found at height %d", height)
	}

	// Delete chunks
	for i := uint32(0); i < snapshot.NumChunks; i++ {
		chunkPath := m.chunkPath(height, i)
		if err := os.Remove(chunkPath); err != nil && !os.IsNotExist(err) {
			m.logger.Error("failed to delete chunk", "height", height, "index", i, "error", err)
		}
	}

	// Delete metadata
	metadataPath := m.metadataPath(height)
	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		m.logger.Error("failed to delete metadata", "height", height, "error", err)
	}

	// Remove from tracking
	delete(m.snapshots, height)

	m.logger.Info("snapshot deleted", "height", height)
	return nil
}

// RestoreFromSnapshot restores state from a snapshot
func (m *Manager) RestoreFromSnapshot(snapshot *Snapshot) ([]byte, error) {
	m.logger.Info("restoring from snapshot", "height", snapshot.Height)

	// Load all chunks
	chunks := make([][]byte, snapshot.NumChunks)
	for i := uint32(0); i < snapshot.NumChunks; i++ {
		chunk, err := m.LoadChunk(snapshot.Height, i)
		if err != nil {
			return nil, fmt.Errorf("failed to load chunk %d: %w", i, err)
		}

		// Verify chunk hash
		if !bytesEqual(chunk.Hash, snapshot.ChunkHashes[i]) {
			return nil, fmt.Errorf("chunk %d hash mismatch", i)
		}

		chunks[i] = chunk.Data
	}

	// Combine chunks
	stateData := CombineChunks(chunks)

	// Verify state data hash
	computedHash := HashData(stateData)
	if !bytesEqual(computedHash, snapshot.Hash) {
		return nil, fmt.Errorf("restored state hash mismatch")
	}

	m.logger.Info("snapshot restored successfully",
		"height", snapshot.Height,
		"size_mb", len(stateData)/(1024*1024))

	return stateData, nil
}

// GetSnapshotStats returns snapshot statistics
func (m *Manager) GetSnapshotStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var totalSize int64
	heights := make([]int64, 0, len(m.snapshots))

	for height := range m.snapshots {
		heights = append(heights, height)
	}

	// Calculate total disk usage
	for _, snapshot := range m.snapshots {
		for i := uint32(0); i < snapshot.NumChunks; i++ {
			chunkPath := m.chunkPath(snapshot.Height, i)
			if info, err := os.Stat(chunkPath); err == nil {
				totalSize += info.Size()
			}
		}
	}

	sort.Slice(heights, func(i, j int) bool {
		return heights[i] < heights[j]
	})

	stats := map[string]interface{}{
		"total_snapshots": len(m.snapshots),
		"total_size_mb":   totalSize / (1024 * 1024),
		"snapshot_dir":    m.snapshotDir,
		"heights":         heights,
	}

	if m.latestSnapshot != nil {
		stats["latest_height"] = m.latestSnapshot.Height
		stats["latest_timestamp"] = time.Unix(m.latestSnapshot.Timestamp, 0)
	}

	return stats
}

// Private methods

// saveChunk saves a chunk to disk
func (m *Manager) saveChunk(snapshot *Snapshot, chunkIndex uint32, data []byte) error {
	chunkPath := m.chunkPath(snapshot.Height, chunkIndex)

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(chunkPath), 0o750); err != nil {
		return fmt.Errorf("failed to create chunk directory: %w", err)
	}

	// Write chunk
	if err := os.WriteFile(chunkPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write chunk: %w", err)
	}

	return nil
}

// saveMetadata saves snapshot metadata to disk
func (m *Manager) saveMetadata(snapshot *Snapshot) error {
	data, err := snapshot.Serialize()
	if err != nil {
		return err
	}

	metadataPath := m.metadataPath(snapshot.Height)
	if err := os.WriteFile(metadataPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// loadSnapshots loads all snapshots from disk
func (m *Manager) loadSnapshots() error {
	// Read snapshot directory
	entries, err := os.ReadDir(m.snapshotDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	// Load each snapshot metadata file
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only load .json files
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Read metadata
		path := filepath.Join(m.snapshotDir, entry.Name())
		data, err := m.readFileSafe(path)
		if err != nil {
			m.logger.Error("failed to read snapshot metadata", "file", entry.Name(), "error", err)
			continue
		}

		// Deserialize
		var snapshot Snapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			m.logger.Error("failed to deserialize snapshot", "file", entry.Name(), "error", err)
			continue
		}

		// Track snapshot
		m.snapshots[snapshot.Height] = &snapshot

		// Update latest
		if m.latestSnapshot == nil || snapshot.Height > m.latestSnapshot.Height {
			m.latestSnapshot = &snapshot
		}
	}

	m.logger.Info("loaded snapshots from disk", "count", len(m.snapshots))
	return nil
}

func (m *Manager) readFileSafe(path string) ([]byte, error) {
	cleanBase := filepath.Clean(m.snapshotDir)
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, cleanBase+string(os.PathSeparator)) && cleanPath != cleanBase {
		return nil, fmt.Errorf("snapshot path %s escapes base %s", cleanPath, cleanBase)
	}
	return os.ReadFile(cleanPath)
}

// pruneOldSnapshots removes old snapshots
func (m *Manager) pruneOldSnapshots() {
	if len(m.snapshots) <= int(m.config.MinSnapshotsToKeep) {
		return
	}

	// Get sorted heights
	heights := make([]int64, 0, len(m.snapshots))
	for height := range m.snapshots {
		heights = append(heights, height)
	}

	sort.Slice(heights, func(i, j int) bool {
		return heights[i] > heights[j]
	})

	// Keep only recent snapshots
	keepCount := int(m.config.SnapshotKeepRecent)
	if keepCount < int(m.config.MinSnapshotsToKeep) {
		keepCount = int(m.config.MinSnapshotsToKeep)
	}

	// Delete old snapshots
	for i := keepCount; i < len(heights); i++ {
		height := heights[i]
		if err := m.DeleteSnapshot(height); err != nil {
			m.logger.Error("failed to prune snapshot", "height", height, "error", err)
		} else {
			m.logger.Info("pruned old snapshot", "height", height)
		}
	}
}

// metadataPath returns the path to a snapshot metadata file
func (m *Manager) metadataPath(height int64) string {
	return filepath.Join(m.snapshotDir, fmt.Sprintf("snapshot-%d.json", height))
}

// chunkPath returns the path to a chunk file
func (m *Manager) chunkPath(height int64, chunkIndex uint32) string {
	return filepath.Join(m.snapshotDir, "chunks", fmt.Sprintf("%d-%d.chunk", height, chunkIndex))
}
