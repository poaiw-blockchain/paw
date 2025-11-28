package snapshot

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// Snapshot represents a state snapshot at a specific height
type Snapshot struct {
	// Snapshot metadata
	Height    int64     `json:"height"`
	Hash      []byte    `json:"hash"`
	Timestamp int64     `json:"timestamp"`
	Format    uint32    `json:"format"`    // Snapshot format version
	ChainID   string    `json:"chain_id"`  // Chain identifier

	// Chunk information
	NumChunks   uint32   `json:"num_chunks"`
	ChunkHashes [][]byte `json:"chunk_hashes"`

	// State metadata
	AppHash        []byte `json:"app_hash"`         // Application state hash
	ValidatorHash  []byte `json:"validator_hash"`   // Validator set hash
	ConsensusHash  []byte `json:"consensus_hash"`   // Consensus params hash

	// Validation
	Signature      []byte   `json:"signature"`        // Snapshot signature
	ValidatorSigs  [][]byte `json:"validator_sigs"`   // Validator signatures (BFT proof)
	SignedBy       []string `json:"signed_by"`        // List of validator IDs
	VotingPower    int64    `json:"voting_power"`     // Total voting power of signers
	TotalPower     int64    `json:"total_power"`      // Total validator voting power
}

// SnapshotChunk represents a chunk of snapshot data
type SnapshotChunk struct {
	Height int64  `json:"height"`  // Snapshot height
	Index  uint32 `json:"index"`   // Chunk index
	Data   []byte `json:"data"`    // Chunk data
	Hash   []byte `json:"hash"`    // Chunk hash
}

// SnapshotMetadata contains snapshot discovery information
type SnapshotMetadata struct {
	Height        int64     `json:"height"`
	Hash          []byte    `json:"hash"`
	NumChunks     uint32    `json:"num_chunks"`
	Format        uint32    `json:"format"`
	ChainID       string    `json:"chain_id"`
	Timestamp     time.Time `json:"timestamp"`
	VotingPower   int64     `json:"voting_power"`
	TotalPower    int64     `json:"total_power"`
}

// SnapshotRequest represents a request for snapshot information
type SnapshotRequest struct {
	MinHeight int64  `json:"min_height"` // Minimum acceptable height
	MaxHeight int64  `json:"max_height"` // Maximum height
	ChainID   string `json:"chain_id"`   // Chain identifier
}

// SnapshotOffer represents a peer's snapshot offer
type SnapshotOffer struct {
	PeerID        string           `json:"peer_id"`
	Snapshot      *SnapshotMetadata `json:"snapshot"`
	ReceivedAt    time.Time        `json:"received_at"`
	Reliability   float64          `json:"reliability"` // Peer reliability score
}

// ChunkRequest represents a request for a snapshot chunk
type ChunkRequest struct {
	Height int64  `json:"height"`
	Index  uint32 `json:"index"`
}

// Constants
const (
	// DefaultChunkSize is the default chunk size (16 MB)
	DefaultChunkSize = 16 * 1024 * 1024

	// MinChunkSize is the minimum chunk size (1 MB)
	MinChunkSize = 1 * 1024 * 1024

	// MaxChunkSize is the maximum chunk size (64 MB)
	MaxChunkSize = 64 * 1024 * 1024

	// SnapshotFormatV1 is the current snapshot format version
	SnapshotFormatV1 = 1

	// MinValidatorSignatures is minimum validator signatures required (2/3+)
	MinValidatorSignaturesFraction = 0.67
)

// Validate validates snapshot metadata
func (s *Snapshot) Validate() error {
	if s.Height <= 0 {
		return fmt.Errorf("invalid height: %d", s.Height)
	}

	if len(s.Hash) == 0 {
		return fmt.Errorf("missing snapshot hash")
	}

	if s.NumChunks == 0 {
		return fmt.Errorf("invalid number of chunks: %d", s.NumChunks)
	}

	if len(s.ChunkHashes) != int(s.NumChunks) {
		return fmt.Errorf("chunk hash count mismatch: expected %d, got %d",
			s.NumChunks, len(s.ChunkHashes))
	}

	if s.ChainID == "" {
		return fmt.Errorf("missing chain ID")
	}

	if len(s.AppHash) == 0 {
		return fmt.Errorf("missing app hash")
	}

	// Validate BFT proof (2/3+ validator signatures)
	if s.TotalPower > 0 {
		signatureFraction := float64(s.VotingPower) / float64(s.TotalPower)
		if signatureFraction < MinValidatorSignaturesFraction {
			return fmt.Errorf("insufficient validator signatures: %.2f%% < %.2f%%",
				signatureFraction*100, MinValidatorSignaturesFraction*100)
		}
	}

	return nil
}

// IsTrusted checks if snapshot has sufficient validator signatures
func (s *Snapshot) IsTrusted() bool {
	if s.TotalPower == 0 {
		return false
	}

	signatureFraction := float64(s.VotingPower) / float64(s.TotalPower)
	return signatureFraction >= MinValidatorSignaturesFraction
}

// Validate validates chunk data
func (c *SnapshotChunk) Validate() error {
	if c.Height <= 0 {
		return fmt.Errorf("invalid height: %d", c.Height)
	}

	if len(c.Data) == 0 {
		return fmt.Errorf("empty chunk data")
	}

	if len(c.Hash) == 0 {
		return fmt.Errorf("missing chunk hash")
	}

	// Verify chunk hash matches data
	computedHash := sha256.Sum256(c.Data)
	if !bytesEqual(c.Hash, computedHash[:]) {
		return fmt.Errorf("chunk hash mismatch")
	}

	return nil
}

// ComputeHash computes the hash of snapshot metadata
func (s *Snapshot) ComputeHash() []byte {
	// Serialize critical fields
	data := fmt.Sprintf("%d:%s:%x:%x:%x",
		s.Height,
		s.ChainID,
		s.AppHash,
		s.ValidatorHash,
		s.ConsensusHash,
	)

	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

// ToMetadata converts Snapshot to SnapshotMetadata
func (s *Snapshot) ToMetadata() *SnapshotMetadata {
	return &SnapshotMetadata{
		Height:      s.Height,
		Hash:        s.Hash,
		NumChunks:   s.NumChunks,
		Format:      s.Format,
		ChainID:     s.ChainID,
		Timestamp:   time.Unix(s.Timestamp, 0),
		VotingPower: s.VotingPower,
		TotalPower:  s.TotalPower,
	}
}

// Serialize serializes the snapshot to JSON
func (s *Snapshot) Serialize() ([]byte, error) {
	return json.Marshal(s)
}

// DeserializeSnapshot deserializes a snapshot from JSON
func DeserializeSnapshot(data []byte) (*Snapshot, error) {
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to deserialize snapshot: %w", err)
	}

	return &snapshot, nil
}

// bytesEqual compares two byte slices
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

// HashData computes SHA256 hash of data
func HashData(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// SplitIntoChunks splits data into chunks of specified size
func SplitIntoChunks(data []byte, chunkSize int) [][]byte {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	totalSize := len(data)
	numChunks := (totalSize + chunkSize - 1) / chunkSize

	chunks := make([][]byte, numChunks)

	for i := 0; i < numChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize

		if end > totalSize {
			end = totalSize
		}

		chunks[i] = data[start:end]
	}

	return chunks
}

// CombineChunks combines chunks into a single data blob
func CombineChunks(chunks [][]byte) []byte {
	// Calculate total size
	totalSize := 0
	for _, chunk := range chunks {
		totalSize += len(chunk)
	}

	// Allocate result buffer
	result := make([]byte, totalSize)

	// Copy chunks
	offset := 0
	for _, chunk := range chunks {
		copy(result[offset:], chunk)
		offset += len(chunk)
	}

	return result
}
