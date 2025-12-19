package protocol

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/p2p/snapshot"
)

// MockPeerManager implements PeerManager interface for testing
type MockPeerManager struct {
	snapshots   map[string]*snapshot.SnapshotMetadata
	chunks      map[string]map[int64]map[uint32][]byte
	malicious   map[string]bool
	queryDelay  time.Duration
	chunkDelay  time.Duration
	reliability map[string]float64
}

func NewMockPeerManager() *MockPeerManager {
	return &MockPeerManager{
		snapshots:   make(map[string]*snapshot.SnapshotMetadata),
		chunks:      make(map[string]map[int64]map[uint32][]byte),
		malicious:   make(map[string]bool),
		reliability: make(map[string]float64),
	}
}

func (m *MockPeerManager) QueryPeerSnapshot(ctx context.Context, peerID string) (*snapshot.SnapshotMetadata, error) {
	if m.queryDelay > 0 {
		time.Sleep(m.queryDelay)
	}

	if m.malicious[peerID] {
		return nil, fmt.Errorf("malicious peer")
	}

	meta, exists := m.snapshots[peerID]
	if !exists {
		return nil, fmt.Errorf("no snapshot available")
	}

	return meta, nil
}

func (m *MockPeerManager) RequestChunk(ctx context.Context, peerID string, height int64, chunkIndex uint32) ([]byte, error) {
	if m.chunkDelay > 0 {
		time.Sleep(m.chunkDelay)
	}

	if m.malicious[peerID] {
		return nil, fmt.Errorf("malicious peer")
	}

	peerChunks, exists := m.chunks[peerID]
	if !exists {
		return nil, fmt.Errorf("peer has no chunks")
	}

	heightChunks, exists := peerChunks[height]
	if !exists {
		return nil, fmt.Errorf("peer doesn't have height %d", height)
	}

	data, exists := heightChunks[chunkIndex]
	if !exists {
		return nil, fmt.Errorf("peer doesn't have chunk %d", chunkIndex)
	}

	return data, nil
}

func (m *MockPeerManager) GetAvailablePeers() []string {
	peers := make([]string, 0, len(m.snapshots))
	for peerID := range m.snapshots {
		peers = append(peers, peerID)
	}
	return peers
}

func (m *MockPeerManager) ReportMaliciousPeer(peerID string, reason string) {
	m.malicious[peerID] = true
}

func (m *MockPeerManager) AddPeerSnapshot(peerID string, meta *snapshot.SnapshotMetadata) {
	m.snapshots[peerID] = meta
}

func (m *MockPeerManager) AddPeerChunk(peerID string, height int64, chunkIndex uint32, data []byte) {
	if m.chunks[peerID] == nil {
		m.chunks[peerID] = make(map[int64]map[uint32][]byte)
	}
	if m.chunks[peerID][height] == nil {
		m.chunks[peerID][height] = make(map[uint32][]byte)
	}
	m.chunks[peerID][height][chunkIndex] = data
}

func (m *MockPeerManager) GetPeerReliability(peerID string) float64 {
	if val, ok := m.reliability[peerID]; ok {
		return val
	}
	return 1.0
}

func (m *MockPeerManager) SetPeerReliability(peerID string, score float64) {
	m.reliability[peerID] = score
}

// Test snapshot discovery
func TestStateSyncDiscovery(t *testing.T) {
	logger := log.NewNopLogger()
	config := DefaultStateSyncConfig()
	config.DiscoveryTime = 2 * time.Second
	config.MinSnapshotOffers = 2

	peerMgr := NewMockPeerManager()

	// Create snapshot metadata
	meta := &snapshot.SnapshotMetadata{
		Height:      1000,
		Hash:        []byte("test-hash"),
		NumChunks:   10,
		Format:      snapshot.SnapshotFormatV1,
		ChainID:     "paw-testnet",
		Timestamp:   time.Now(),
		VotingPower: 100,
		TotalPower:  100,
	}

	// Add snapshots from multiple peers
	peerMgr.AddPeerSnapshot("peer1", meta)
	peerMgr.AddPeerSnapshot("peer2", meta)
	peerMgr.AddPeerSnapshot("peer3", meta)

	ssp := NewStateSyncProtocol(config, nil, peerMgr, logger)

	ctx := context.Background()
	offers, err := ssp.discoverSnapshots(ctx)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(offers), 2)
	require.Equal(t, int64(1000), offers[0].Snapshot.Height)
}

// Test snapshot selection with Byzantine agreement
func TestStateSyncSelection(t *testing.T) {
	logger := log.NewNopLogger()
	config := DefaultStateSyncConfig()

	peerMgr := NewMockPeerManager()
	ssp := NewStateSyncProtocol(config, nil, peerMgr, logger)

	// Create two different snapshots
	meta1 := &snapshot.SnapshotMetadata{
		Height:      1000,
		Hash:        []byte("hash-1"),
		NumChunks:   10,
		Format:      snapshot.SnapshotFormatV1,
		ChainID:     "paw-testnet",
		VotingPower: 100,
		TotalPower:  100,
	}

	meta2 := &snapshot.SnapshotMetadata{
		Height:      1000,
		Hash:        []byte("hash-2"), // Different hash (malicious)
		NumChunks:   10,
		Format:      snapshot.SnapshotFormatV1,
		ChainID:     "paw-testnet",
		VotingPower: 100,
		TotalPower:  100,
	}

	// 4 peers agree on meta1, 1 peer has different hash
	offers := []*snapshot.SnapshotOffer{
		{PeerID: "peer1", Snapshot: meta1},
		{PeerID: "peer2", Snapshot: meta1},
		{PeerID: "peer3", Snapshot: meta1},
		{PeerID: "peer4", Snapshot: meta1},
		{PeerID: "peer5", Snapshot: meta2}, // Byzantine peer
	}

	selected, err := ssp.selectBestSnapshot(offers)

	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, int64(1000), selected.Height)
	// Should select the one with 80% agreement (4/5)
}

func TestStateSyncDiscoveryIncludesReliability(t *testing.T) {
	logger := log.NewNopLogger()
	config := DefaultStateSyncConfig()
	config.MinSnapshotOffers = 2

	peerMgr := NewMockPeerManager()
	meta := &snapshot.SnapshotMetadata{
		Height:    2000,
		Hash:      []byte("reliability-test"),
		NumChunks: 2,
		ChainID:   "paw-testnet",
	}

	for _, peer := range []string{"peerA", "peerB"} {
		peerMgr.AddPeerSnapshot(peer, meta)
	}

	peerMgr.SetPeerReliability("peerA", 0.25)
	peerMgr.SetPeerReliability("peerB", 0.9)

	ssp := NewStateSyncProtocol(config, nil, peerMgr, logger)
	offers, err := ssp.discoverSnapshots(context.Background())

	require.NoError(t, err)
	require.Len(t, offers, 2)

	reliabilities := make(map[string]float64)
	for _, offer := range offers {
		reliabilities[offer.PeerID] = offer.Reliability
	}

	require.InDelta(t, 0.25, reliabilities["peerA"], 0.0001)
	require.InDelta(t, 0.9, reliabilities["peerB"], 0.0001)
}

func TestStateSyncSelectionWeightsReliability(t *testing.T) {
	logger := log.NewNopLogger()
	config := DefaultStateSyncConfig()
	config.MinPeerAgreement = 0.3

	ssp := NewStateSyncProtocol(config, nil, nil, logger)

	metaTrusted := &snapshot.SnapshotMetadata{
		Height:    1200,
		Hash:      []byte("trusted"),
		NumChunks: 8,
		ChainID:   "paw-testnet",
	}

	metaRisky := &snapshot.SnapshotMetadata{
		Height:    1200,
		Hash:      []byte("risky"),
		NumChunks: 8,
		ChainID:   "paw-testnet",
	}

	offers := []*snapshot.SnapshotOffer{
		{PeerID: "high1", Snapshot: metaTrusted, Reliability: 0.95},
		{PeerID: "high2", Snapshot: metaTrusted, Reliability: 0.85},
		{PeerID: "low1", Snapshot: metaRisky, Reliability: 0.30},
		{PeerID: "low2", Snapshot: metaRisky, Reliability: 0.30},
		{PeerID: "low3", Snapshot: metaRisky, Reliability: 0.30},
	}

	selected, err := ssp.selectBestSnapshot(offers)

	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, int64(1200), selected.Height)
	require.Equal(t, fmt.Sprintf("%x", metaTrusted.Hash), fmt.Sprintf("%x", selected.Hash))
}

// Test snapshot verification
func TestStateSyncVerification(t *testing.T) {
	logger := log.NewNopLogger()
	config := DefaultStateSyncConfig()
	config.TrustHeight = 500
	config.RequireBFTProof = true

	ssp := NewStateSyncProtocol(config, nil, nil, logger)

	tests := []struct {
		name        string
		snapshot    *snapshot.Snapshot
		expectError bool
	}{
		{
			name: "valid snapshot",
			snapshot: &snapshot.Snapshot{
				Height:        1000,
				Hash:          []byte("test-hash"),
				NumChunks:     10,
				ChunkHashes:   make([][]byte, 10),
				Format:        snapshot.SnapshotFormatV1,
				ChainID:       "paw-testnet",
				AppHash:       []byte("app-hash"),
				ValidatorHash: []byte("val-hash"),
				VotingPower:   70,
				TotalPower:    100,
			},
			expectError: false,
		},
		{
			name: "below trust height",
			snapshot: &snapshot.Snapshot{
				Height:        400, // Below trust height of 500
				Hash:          []byte("test-hash"),
				NumChunks:     10,
				ChunkHashes:   make([][]byte, 10),
				Format:        snapshot.SnapshotFormatV1,
				ChainID:       "paw-testnet",
				AppHash:       []byte("app-hash"),
				ValidatorHash: []byte("val-hash"),
				VotingPower:   70,
				TotalPower:    100,
			},
			expectError: true,
		},
		{
			name: "insufficient BFT proof",
			snapshot: &snapshot.Snapshot{
				Height:        1000,
				Hash:          []byte("test-hash"),
				NumChunks:     10,
				ChunkHashes:   make([][]byte, 10),
				Format:        snapshot.SnapshotFormatV1,
				ChainID:       "paw-testnet",
				AppHash:       []byte("app-hash"),
				ValidatorHash: []byte("val-hash"),
				VotingPower:   50, // Only 50%, needs 67%
				TotalPower:    100,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ssp.verifySnapshot(tt.snapshot)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test chunk download with retries
func TestChunkDownloadWithRetry(t *testing.T) {
	logger := log.NewNopLogger()
	config := DefaultStateSyncConfig()
	config.ChunkRequestTimeout = 1 * time.Second
	config.ChunkRetryAttempts = 3

	peerMgr := NewMockPeerManager()

	// Create test hash
	testHash := []byte("test-hash")

	// Add snapshot offer
	meta := &snapshot.SnapshotMetadata{
		Height:    1000,
		Hash:      testHash,
		NumChunks: 5,
		ChainID:   "paw-testnet",
	}
	peerMgr.AddPeerSnapshot("peer1", meta)

	// Add chunks
	testData := []byte("test chunk data")
	for i := uint32(0); i < 5; i++ {
		peerMgr.AddPeerChunk("peer1", 1000, i, testData)
	}

	ssp := NewStateSyncProtocol(config, nil, peerMgr, logger)

	// Set selected snapshot with same hash
	ssp.selectedSnapshot = &snapshot.Snapshot{
		Height:    1000,
		Hash:      testHash,
		NumChunks: 5,
	}

	// Add peer offer
	ssp.peerOffers["peer1"] = &snapshot.SnapshotOffer{
		PeerID:   "peer1",
		Snapshot: meta,
	}

	ctx := context.Background()
	chunk, err := ssp.downloadChunkWithRetry(ctx, ssp.selectedSnapshot, 0)

	require.NoError(t, err)
	require.NotNil(t, chunk)
	require.Equal(t, uint32(0), chunk.Index)
	require.Equal(t, testData, chunk.Data)
}

// Test parallel chunk download
func TestParallelChunkDownload(t *testing.T) {
	logger := log.NewNopLogger()
	config := DefaultStateSyncConfig()
	config.ChunkFetchers = 4
	config.VerifyAllChunks = false // Skip verification for this test

	peerMgr := NewMockPeerManager()

	// Create test data
	testData := make([][]byte, 20)
	for i := range testData {
		testData[i] = []byte(fmt.Sprintf("chunk-%d-data", i))
	}

	// Add snapshot and chunks
	meta := &snapshot.SnapshotMetadata{
		Height:    1000,
		Hash:      []byte("test-hash"),
		NumChunks: uint32(len(testData)),
		ChainID:   "paw-testnet",
	}

	// Add to multiple peers
	for _, peerID := range []string{"peer1", "peer2", "peer3"} {
		peerMgr.AddPeerSnapshot(peerID, meta)
		for i, data := range testData {
			peerMgr.AddPeerChunk(peerID, 1000, uint32(i), data)
		}
	}

	// Create temporary snapshot manager
	tmpDir := t.TempDir()
	mgr, err := snapshot.NewManager(
		&snapshot.ManagerConfig{
			SnapshotDir:        tmpDir,
			ChunkSize:          snapshot.DefaultChunkSize,
			SnapshotInterval:   1000,
			SnapshotKeepRecent: 10,
			ChainID:            "paw-testnet",
		},
		logger,
	)
	require.NoError(t, err)

	ssp := NewStateSyncProtocol(config, mgr, peerMgr, logger)

	// Set selected snapshot
	snap := &snapshot.Snapshot{
		Height:      1000,
		Hash:        []byte("test-hash"),
		NumChunks:   uint32(len(testData)),
		ChunkHashes: make([][]byte, len(testData)),
		ChainID:     "paw-testnet",
	}

	for i := range testData {
		snap.ChunkHashes[i] = snapshot.HashData(testData[i])
	}

	ssp.selectedSnapshot = snap

	// Add peer offers
	for _, peerID := range []string{"peer1", "peer2", "peer3"} {
		ssp.peerOffers[peerID] = &snapshot.SnapshotOffer{
			PeerID:   peerID,
			Snapshot: meta,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = ssp.downloadSnapshotChunks(ctx, snap)
	require.NoError(t, err)

	// Verify all chunks were downloaded
	require.Equal(t, uint32(len(testData)), uint32(len(ssp.downloadedChunks)))
}

func TestSelectPeerForChunkPrioritizesReliability(t *testing.T) {
	logger := log.NewNopLogger()
	config := DefaultStateSyncConfig()

	ssp := NewStateSyncProtocol(config, nil, nil, logger)

	snap := &snapshot.Snapshot{
		Height: 1500,
		Hash:   []byte("chunk-hash"),
	}

	baseMeta := func() *snapshot.SnapshotMetadata {
		return &snapshot.SnapshotMetadata{
			Height: 1500,
			Hash:   []byte("chunk-hash"),
		}
	}

	ssp.peerOffers["alpha"] = &snapshot.SnapshotOffer{
		PeerID:      "alpha",
		Snapshot:    baseMeta(),
		Reliability: 0.95,
	}
	ssp.peerOffers["bravo"] = &snapshot.SnapshotOffer{
		PeerID:      "bravo",
		Snapshot:    baseMeta(),
		Reliability: 0.6,
	}
	ssp.peerOffers["charlie"] = &snapshot.SnapshotOffer{
		PeerID:      "charlie",
		Snapshot:    baseMeta(),
		Reliability: 0.2,
	}

	counts := map[string]int{"alpha": 0, "bravo": 0, "charlie": 0}
	for i := 0; i < 64; i++ {
		peer, err := ssp.selectPeerForChunk(snap, uint32(i))
		require.NoError(t, err)
		counts[peer]++
	}

	require.Greater(t, counts["alpha"], counts["bravo"])
	require.Greater(t, counts["bravo"], counts["charlie"])
	require.Greater(t, counts["charlie"], 0)
}

// Test Byzantine peer detection
func TestByzantinePeerDetection(t *testing.T) {
	logger := log.NewNopLogger()
	config := DefaultStateSyncConfig()
	config.MaxMaliciousPeers = 2

	peerMgr := NewMockPeerManager()
	ssp := NewStateSyncProtocol(config, nil, peerMgr, logger)

	// Report malicious peers
	ssp.reportSuspiciousPeer("peer1", "bad data")
	ssp.reportSuspiciousPeer("peer2", "timeout")

	require.Equal(t, 2, len(ssp.maliciousPeers))
	require.True(t, ssp.maliciousPeers["peer1"])
	require.True(t, ssp.maliciousPeers["peer2"])
	require.Equal(t, int64(2), ssp.metrics.MaliciousPeersFound)
}

// Test state sync metrics
func TestStateSyncMetrics(t *testing.T) {
	logger := log.NewNopLogger()
	config := DefaultStateSyncConfig()

	ssp := NewStateSyncProtocol(config, nil, nil, logger)

	// Update metrics
	ssp.metrics.SnapshotsDiscovered = 5
	ssp.metrics.ChunksDownloaded = 100
	ssp.metrics.BytesDownloaded = 1024 * 1024 * 100 // 100 MB

	metrics := ssp.GetMetrics()

	require.Equal(t, int64(5), metrics.SnapshotsDiscovered)
	require.Equal(t, int64(100), metrics.ChunksDownloaded)
	require.Equal(t, int64(1024*1024*100), metrics.BytesDownloaded)
}

// Test progress tracking
func TestProgressTracking(t *testing.T) {
	logger := log.NewNopLogger()
	config := DefaultStateSyncConfig()

	ssp := NewStateSyncProtocol(config, nil, nil, logger)

	// Set selected snapshot
	ssp.selectedSnapshot = &snapshot.Snapshot{
		Height:    1000,
		NumChunks: 50,
	}

	// Mark some chunks as downloaded
	ssp.downloadedChunks[0] = true
	ssp.downloadedChunks[1] = true
	ssp.downloadedChunks[2] = true

	downloaded, total := ssp.GetProgress()

	require.Equal(t, uint32(3), downloaded)
	require.Equal(t, uint32(50), total)
}

// Benchmark chunk download
func BenchmarkChunkDownload(b *testing.B) {
	logger := log.NewNopLogger()
	config := DefaultStateSyncConfig()

	peerMgr := NewMockPeerManager()
	testData := make([]byte, 16*1024*1024) // 16 MB chunk

	peerMgr.AddPeerSnapshot("peer1", &snapshot.SnapshotMetadata{
		Height:    1000,
		Hash:      []byte("test"),
		NumChunks: 1,
	})
	peerMgr.AddPeerChunk("peer1", 1000, 0, testData)

	ssp := NewStateSyncProtocol(config, nil, peerMgr, logger)
	ssp.selectedSnapshot = &snapshot.Snapshot{Height: 1000, NumChunks: 1}
	ssp.peerOffers["peer1"] = &snapshot.SnapshotOffer{
		PeerID: "peer1",
		Snapshot: &snapshot.SnapshotMetadata{
			Height: 1000,
			Hash:   []byte("test"),
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ssp.downloadChunkWithRetry(ctx, ssp.selectedSnapshot, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}
