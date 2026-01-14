package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"time"
)

// LocationProof represents cryptographic proof of validator geographic location
// This is used to prevent validators from spoofing their location
type LocationProof struct {
	ValidatorAddr string    `json:"validator_addr"`
	IPAddress     string    `json:"ip_address"`
	ClaimedRegion string    `json:"claimed_region"`
	Timestamp     time.Time `json:"timestamp"`
	ProofHash     string    `json:"proof_hash"` // SHA256 hash of proof data
	Signature     []byte    `json:"signature"`  // Validator signature
}

// NewLocationProof creates a new location proof
func NewLocationProof(validatorAddr, ipAddress, claimedRegion string) *LocationProof {
	now := time.Now()
	proof := &LocationProof{
		ValidatorAddr: validatorAddr,
		IPAddress:     ipAddress,
		ClaimedRegion: claimedRegion,
		Timestamp:     now,
	}
	proof.ProofHash = proof.ComputeHash()
	return proof
}

// ComputeHash computes the SHA256 hash of the location proof
func (lp *LocationProof) ComputeHash() string {
	data := fmt.Sprintf("%s:%s:%s:%d",
		lp.ValidatorAddr,
		lp.IPAddress,
		lp.ClaimedRegion,
		lp.Timestamp.Unix(),
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// IsValid checks if the location proof is valid and not expired
func (lp *LocationProof) IsValid(maxAge time.Duration) bool {
	// Check if proof is too old
	if time.Since(lp.Timestamp) > maxAge {
		return false
	}

	// Verify hash
	expectedHash := lp.ComputeHash()
	if lp.ProofHash != expectedHash {
		return false
	}

	return true
}

// ValidateBasic performs basic validation of location proof
func (lp *LocationProof) ValidateBasic() error {
	if lp.ValidatorAddr == "" {
		return fmt.Errorf("validator address cannot be empty")
	}

	if lp.IPAddress == "" {
		return fmt.Errorf("IP address cannot be empty")
	}

	if lp.ClaimedRegion == "" {
		return fmt.Errorf("claimed region cannot be empty")
	}

	if lp.Timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
	}

	if lp.ProofHash == "" {
		return fmt.Errorf("proof hash cannot be empty")
	}

	// Verify hash matches
	expectedHash := lp.ComputeHash()
	if lp.ProofHash != expectedHash {
		return fmt.Errorf("proof hash mismatch: expected %s, got %s", expectedHash, lp.ProofHash)
	}

	return nil
}

// LocationEvidence represents evidence of validator location over time
// This helps detect if validators are spoofing location by tracking history
type LocationEvidence struct {
	ValidatorAddr string            `json:"validator_addr"`
	Proofs        []*LocationProof  `json:"proofs"`
	VerifiedAt    time.Time         `json:"verified_at"`
	VerifiedBy    string            `json:"verified_by"` // Optional: external verifier
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// AddProof adds a location proof to the evidence
func (le *LocationEvidence) AddProof(proof *LocationProof) error {
	if err := proof.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid proof: %w", err)
	}

	if proof.ValidatorAddr != le.ValidatorAddr {
		return fmt.Errorf("proof validator address mismatch")
	}

	le.Proofs = append(le.Proofs, proof)
	return nil
}

// GetLatestProof returns the most recent location proof
func (le *LocationEvidence) GetLatestProof() *LocationProof {
	if len(le.Proofs) == 0 {
		return nil
	}

	latest := le.Proofs[0]
	for _, proof := range le.Proofs {
		if proof.Timestamp.After(latest.Timestamp) {
			latest = proof
		}
	}

	return latest
}

// IsConsistent checks if location proofs are consistent over time
// Returns true if all recent proofs show same region
func (le *LocationEvidence) IsConsistent(maxAge time.Duration) bool {
	if len(le.Proofs) == 0 {
		return false
	}

	var region string
	recentProofs := 0

	for _, proof := range le.Proofs {
		if time.Since(proof.Timestamp) > maxAge {
			continue
		}

		if region == "" {
			region = proof.ClaimedRegion
		} else if proof.ClaimedRegion != region {
			// Location changed - suspicious
			return false
		}

		recentProofs++
	}

	// Need at least 1 recent proof
	return recentProofs > 0
}

// DetectLocationJumps detects if validator location has changed suspiciously
// Returns true if location changed too frequently (indicating spoofing)
func (le *LocationEvidence) DetectLocationJumps(threshold int, period time.Duration) bool {
	if len(le.Proofs) < 2 {
		return false
	}

	changes := 0
	var lastRegion string
	cutoff := time.Now().Add(-period)

	for _, proof := range le.Proofs {
		if proof.Timestamp.Before(cutoff) {
			continue
		}

		if lastRegion != "" && proof.ClaimedRegion != lastRegion {
			changes++
		}
		lastRegion = proof.ClaimedRegion
	}

	return changes >= threshold
}

// GeographicDistribution represents the distribution of validators across regions
type GeographicDistribution struct {
	RegionCounts   map[string]int `json:"region_counts"`
	TotalCount     int            `json:"total_count"`
	UniqueRegions  int            `json:"unique_regions"`
	DiversityScore float64        `json:"diversity_score"` // 0-1, higher is better
}

// NewGeographicDistribution creates a new geographic distribution tracker
func NewGeographicDistribution() *GeographicDistribution {
	return &GeographicDistribution{
		RegionCounts: make(map[string]int),
	}
}

// AddRegion adds a validator's region to the distribution
func (gd *GeographicDistribution) AddRegion(region string) {
	gd.RegionCounts[region]++
	gd.TotalCount++
	gd.calculateDiversity()
}

// calculateDiversity computes diversity score using Shannon entropy
// Higher score = more evenly distributed validators
func (gd *GeographicDistribution) calculateDiversity() {
	gd.UniqueRegions = len(gd.RegionCounts)

	if gd.TotalCount == 0 || gd.UniqueRegions == 0 {
		gd.DiversityScore = 0
		return
	}

	// Shannon entropy: H = -sum(p_i * log2(p_i))
	entropy := 0.0
	for _, count := range gd.RegionCounts {
		if count > 0 {
			p := float64(count) / float64(gd.TotalCount)
			entropy -= p * log2(p)
		}
	}

	// Normalize to 0-1 scale
	// Maximum entropy is log2(n) where n is number of regions
	maxEntropy := log2(float64(gd.UniqueRegions))
	if maxEntropy > 0 {
		gd.DiversityScore = entropy / maxEntropy
	} else {
		gd.DiversityScore = 0
	}
}

// log2 computes logarithm base 2
func log2(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// log2(x) = ln(x) / ln(2)
	return logNatural(x) / logNatural(2)
}

// logNatural computes natural logarithm
// Uses the standard math library for accurate results
func logNatural(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return math.Log(x)
}

// IsSufficient checks if geographic diversity is sufficient
func (gd *GeographicDistribution) IsSufficient(minRegions int, minDiversityScore float64) bool {
	return gd.UniqueRegions >= minRegions && gd.DiversityScore >= minDiversityScore
}
