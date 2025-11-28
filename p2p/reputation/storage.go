package reputation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cosmossdk.io/log"
)

// Storage handles persistence of peer reputation data
type Storage interface {
	// Save saves a peer reputation
	Save(rep *PeerReputation) error

	// Load loads a peer reputation
	Load(peerID PeerID) (*PeerReputation, error)

	// LoadAll loads all peer reputations
	LoadAll() (map[PeerID]*PeerReputation, error)

	// Delete deletes a peer reputation
	Delete(peerID PeerID) error

	// SaveSnapshot saves a complete snapshot
	SaveSnapshot(snapshot *ReputationSnapshot) error

	// LoadLatestSnapshot loads the most recent snapshot
	LoadLatestSnapshot() (*ReputationSnapshot, error)

	// Cleanup removes old data
	Cleanup(olderThan time.Time) error

	// Close closes the storage
	Close() error
}

// FileStorage implements Storage using JSON files
type FileStorage struct {
	dataDir       string
	snapshotDir   string
	logger        log.Logger
	mu            sync.RWMutex
	writeCache    map[PeerID]*PeerReputation
	cacheSize     int
	flushInterval time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// FileStorageConfig configures file storage
type FileStorageConfig struct {
	DataDir       string
	CacheSize     int
	FlushInterval time.Duration
	EnableCache   bool
}

// DefaultFileStorageConfig returns default file storage config
func DefaultFileStorageConfig(homeDir string) FileStorageConfig {
	return FileStorageConfig{
		DataDir:       filepath.Join(homeDir, "data", "p2p", "reputation"),
		CacheSize:     1000,
		FlushInterval: 30 * time.Second,
		EnableCache:   true,
	}
}

// NewFileStorage creates a new file-based storage
func NewFileStorage(config FileStorageConfig, logger log.Logger) (*FileStorage, error) {
	// Create directories
	dataDir := config.DataDir
	snapshotDir := filepath.Join(dataDir, "snapshots")

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	if err := os.MkdirAll(snapshotDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	fs := &FileStorage{
		dataDir:       dataDir,
		snapshotDir:   snapshotDir,
		logger:        logger,
		writeCache:    make(map[PeerID]*PeerReputation),
		cacheSize:     config.CacheSize,
		flushInterval: config.FlushInterval,
		stopChan:      make(chan struct{}),
	}

	// Start background flusher if cache enabled
	if config.EnableCache {
		fs.wg.Add(1)
		go fs.backgroundFlusher()
	}

	return fs, nil
}

// Save saves a peer reputation
func (fs *FileStorage) Save(rep *PeerReputation) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Update cache
	fs.writeCache[rep.PeerID] = rep

	// Flush if cache is full
	if len(fs.writeCache) >= fs.cacheSize {
		return fs.flushCache()
	}

	return nil
}

// Load loads a peer reputation
func (fs *FileStorage) Load(peerID PeerID) (*PeerReputation, error) {
	fs.mu.RLock()
	// Check cache first
	if rep, ok := fs.writeCache[peerID]; ok {
		fs.mu.RUnlock()
		return rep, nil
	}
	fs.mu.RUnlock()

	// Load from disk
	filePath := fs.getPeerFilePath(peerID)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Peer not found
		}
		return nil, fmt.Errorf("failed to read peer file: %w", err)
	}

	var rep PeerReputation
	if err := json.Unmarshal(data, &rep); err != nil {
		return nil, fmt.Errorf("failed to unmarshal peer data: %w", err)
	}

	return &rep, nil
}

// LoadAll loads all peer reputations
func (fs *FileStorage) LoadAll() (map[PeerID]*PeerReputation, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	peers := make(map[PeerID]*PeerReputation)

	// Load from disk
	entries, err := os.ReadDir(fs.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(fs.dataDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			fs.logger.Error("failed to read peer file", "file", entry.Name(), "error", err)
			continue
		}

		var rep PeerReputation
		if err := json.Unmarshal(data, &rep); err != nil {
			fs.logger.Error("failed to unmarshal peer data", "file", entry.Name(), "error", err)
			continue
		}

		peers[rep.PeerID] = &rep
	}

	// Merge with cache (cache takes precedence)
	for peerID, rep := range fs.writeCache {
		peers[peerID] = rep
	}

	return peers, nil
}

// Delete deletes a peer reputation
func (fs *FileStorage) Delete(peerID PeerID) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Remove from cache
	delete(fs.writeCache, peerID)

	// Remove from disk
	filePath := fs.getPeerFilePath(peerID)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete peer file: %w", err)
	}

	return nil
}

// SaveSnapshot saves a complete snapshot
func (fs *FileStorage) SaveSnapshot(snapshot *ReputationSnapshot) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	filename := fmt.Sprintf("snapshot_%d.json", snapshot.Timestamp.Unix())
	filePath := filepath.Join(fs.snapshotDir, filename)

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write snapshot: %w", err)
	}

	fs.logger.Info("saved reputation snapshot", "peers", snapshot.TotalPeers, "file", filename)
	return nil
}

// LoadLatestSnapshot loads the most recent snapshot
func (fs *FileStorage) LoadLatestSnapshot() (*ReputationSnapshot, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entries, err := os.ReadDir(fs.snapshotDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	if len(entries) == 0 {
		return nil, nil // No snapshots
	}

	// Find latest snapshot (they're named with timestamps)
	var latestEntry os.DirEntry
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if latestEntry == nil || entry.Name() > latestEntry.Name() {
			latestEntry = entry
		}
	}

	if latestEntry == nil {
		return nil, nil
	}

	filePath := filepath.Join(fs.snapshotDir, latestEntry.Name())
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot: %w", err)
	}

	var snapshot ReputationSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	return &snapshot, nil
}

// Cleanup removes old data
func (fs *FileStorage) Cleanup(olderThan time.Time) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	count := 0

	// Clean up old peer files
	entries, err := os.ReadDir(fs.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read data directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(fs.dataDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(olderThan) {
			if err := os.Remove(filePath); err != nil {
				fs.logger.Error("failed to remove old peer file", "file", entry.Name(), "error", err)
			} else {
				count++
			}
		}
	}

	// Clean up old snapshots (keep last 30)
	snapshotEntries, err := os.ReadDir(fs.snapshotDir)
	if err != nil {
		return fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	if len(snapshotEntries) > 30 {
		// Sort by name (timestamp-based) and remove oldest
		for i := 0; i < len(snapshotEntries)-30; i++ {
			filePath := filepath.Join(fs.snapshotDir, snapshotEntries[i].Name())
			if err := os.Remove(filePath); err != nil {
				fs.logger.Error("failed to remove old snapshot", "file", snapshotEntries[i].Name(), "error", err)
			}
		}
	}

	fs.logger.Info("cleaned up old reputation data", "files_removed", count, "older_than", olderThan)
	return nil
}

// Close closes the storage and flushes cache
func (fs *FileStorage) Close() error {
	// Stop background flusher
	close(fs.stopChan)
	fs.wg.Wait()

	// Final flush
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return fs.flushCache()
}

// Flush forces a cache flush
func (fs *FileStorage) Flush() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return fs.flushCache()
}

// flushCache writes cached data to disk (must be called with lock held)
func (fs *FileStorage) flushCache() error {
	if len(fs.writeCache) == 0 {
		return nil
	}

	errCount := 0
	for peerID, rep := range fs.writeCache {
		if err := fs.saveToDisk(rep); err != nil {
			fs.logger.Error("failed to save peer to disk", "peer_id", peerID, "error", err)
			errCount++
		}
	}

	// Clear cache
	fs.writeCache = make(map[PeerID]*PeerReputation)

	if errCount > 0 {
		return fmt.Errorf("failed to save %d peers", errCount)
	}

	return nil
}

// saveToDisk saves a single peer to disk (must be called with lock held)
func (fs *FileStorage) saveToDisk(rep *PeerReputation) error {
	filePath := fs.getPeerFilePath(rep.PeerID)

	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal peer data: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write peer file: %w", err)
	}

	return nil
}

// getPeerFilePath returns the file path for a peer
func (fs *FileStorage) getPeerFilePath(peerID PeerID) string {
	filename := fmt.Sprintf("%s.json", peerID)
	return filepath.Join(fs.dataDir, filename)
}

// backgroundFlusher periodically flushes the cache
func (fs *FileStorage) backgroundFlusher() {
	defer fs.wg.Done()

	ticker := time.NewTicker(fs.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fs.mu.Lock()
			if err := fs.flushCache(); err != nil {
				fs.logger.Error("background flush failed", "error", err)
			}
			fs.mu.Unlock()

		case <-fs.stopChan:
			return
		}
	}
}

// MemoryStorage implements Storage using in-memory storage (for testing)
type MemoryStorage struct {
	peers     map[PeerID]*PeerReputation
	snapshots []*ReputationSnapshot
	mu        sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		peers:     make(map[PeerID]*PeerReputation),
		snapshots: make([]*ReputationSnapshot, 0),
	}
}

func (ms *MemoryStorage) Save(rep *PeerReputation) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Make a copy to avoid external modifications
	repCopy := *rep
	ms.peers[rep.PeerID] = &repCopy
	return nil
}

func (ms *MemoryStorage) Load(peerID PeerID) (*PeerReputation, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	rep, ok := ms.peers[peerID]
	if !ok {
		return nil, nil
	}

	// Return a copy
	repCopy := *rep
	return &repCopy, nil
}

func (ms *MemoryStorage) LoadAll() (map[PeerID]*PeerReputation, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	result := make(map[PeerID]*PeerReputation, len(ms.peers))
	for id, rep := range ms.peers {
		repCopy := *rep
		result[id] = &repCopy
	}

	return result, nil
}

func (ms *MemoryStorage) Delete(peerID PeerID) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.peers, peerID)
	return nil
}

func (ms *MemoryStorage) SaveSnapshot(snapshot *ReputationSnapshot) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	snapshotCopy := *snapshot
	ms.snapshots = append(ms.snapshots, &snapshotCopy)
	return nil
}

func (ms *MemoryStorage) LoadLatestSnapshot() (*ReputationSnapshot, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if len(ms.snapshots) == 0 {
		return nil, nil
	}

	snapshot := ms.snapshots[len(ms.snapshots)-1]
	snapshotCopy := *snapshot
	return &snapshotCopy, nil
}

func (ms *MemoryStorage) Cleanup(olderThan time.Time) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Remove peers not seen since olderThan
	for id, rep := range ms.peers {
		if rep.LastSeen.Before(olderThan) {
			delete(ms.peers, id)
		}
	}

	return nil
}

func (ms *MemoryStorage) Close() error {
	return nil
}
