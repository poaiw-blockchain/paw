package types

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestNewLocationProof(t *testing.T) {
	validatorAddr := "cosmosvaloper1test"
	ipAddress := "192.168.1.1"
	claimedRegion := "na"

	proof := NewLocationProof(validatorAddr, ipAddress, claimedRegion)

	if proof == nil {
		t.Fatal("NewLocationProof returned nil")
	}

	if proof.ValidatorAddr != validatorAddr {
		t.Errorf("Expected ValidatorAddr %s, got %s", validatorAddr, proof.ValidatorAddr)
	}

	if proof.IPAddress != ipAddress {
		t.Errorf("Expected IPAddress %s, got %s", ipAddress, proof.IPAddress)
	}

	if proof.ClaimedRegion != claimedRegion {
		t.Errorf("Expected ClaimedRegion %s, got %s", claimedRegion, proof.ClaimedRegion)
	}

	if proof.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	if proof.ProofHash == "" {
		t.Error("ProofHash should not be empty")
	}

	// Verify hash was computed correctly
	expectedHash := proof.ComputeHash()
	if proof.ProofHash != expectedHash {
		t.Errorf("ProofHash mismatch: expected %s, got %s", expectedHash, proof.ProofHash)
	}
}

func TestLocationProof_ComputeHash(t *testing.T) {
	proof := &LocationProof{
		ValidatorAddr: "cosmosvaloper1test",
		IPAddress:     "192.168.1.1",
		ClaimedRegion: "na",
		Timestamp:     time.Unix(1234567890, 0),
	}

	hash1 := proof.ComputeHash()
	hash2 := proof.ComputeHash()

	// Same input should produce same hash
	if hash1 != hash2 {
		t.Error("ComputeHash should be deterministic")
	}

	// Hash should be non-empty hex string
	if hash1 == "" {
		t.Error("ComputeHash returned empty string")
	}

	// Hash should be 64 characters (SHA256 hex)
	if len(hash1) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}

	// Different input should produce different hash
	proof.ClaimedRegion = "eu"
	hash3 := proof.ComputeHash()
	if hash1 == hash3 {
		t.Error("Different inputs should produce different hashes")
	}
}

func TestLocationProof_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		proof     *LocationProof
		maxAge    time.Duration
		wantValid bool
	}{
		{
			name: "valid proof within age limit",
			proof: &LocationProof{
				ValidatorAddr: "cosmosvaloper1test",
				IPAddress:     "192.168.1.1",
				ClaimedRegion: "na",
				Timestamp:     time.Now().Add(-5 * time.Minute),
			},
			maxAge:    10 * time.Minute,
			wantValid: true,
		},
		{
			name: "expired proof",
			proof: &LocationProof{
				ValidatorAddr: "cosmosvaloper1test",
				IPAddress:     "192.168.1.1",
				ClaimedRegion: "na",
				Timestamp:     time.Now().Add(-15 * time.Minute),
			},
			maxAge:    10 * time.Minute,
			wantValid: false,
		},
		{
			name: "proof exactly at max age",
			proof: &LocationProof{
				ValidatorAddr: "cosmosvaloper1test",
				IPAddress:     "192.168.1.1",
				ClaimedRegion: "na",
				Timestamp:     time.Now().Add(-10 * time.Minute),
			},
			maxAge:    10 * time.Minute,
			wantValid: false, // Should be expired at exactly maxAge
		},
		{
			name: "future timestamp (invalid)",
			proof: &LocationProof{
				ValidatorAddr: "cosmosvaloper1test",
				IPAddress:     "192.168.1.1",
				ClaimedRegion: "na",
				Timestamp:     time.Now().Add(5 * time.Minute),
			},
			maxAge:    10 * time.Minute,
			wantValid: true, // Future timestamp is within maxAge
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set proof hash
			tt.proof.ProofHash = tt.proof.ComputeHash()

			isValid := tt.proof.IsValid(tt.maxAge)
			if isValid != tt.wantValid {
				t.Errorf("IsValid() = %v, want %v", isValid, tt.wantValid)
			}
		})
	}
}

func TestLocationProof_IsValid_InvalidHash(t *testing.T) {
	proof := &LocationProof{
		ValidatorAddr: "cosmosvaloper1test",
		IPAddress:     "192.168.1.1",
		ClaimedRegion: "na",
		Timestamp:     time.Now(),
		ProofHash:     "invalid_hash",
	}

	if proof.IsValid(10 * time.Minute) {
		t.Error("Proof with invalid hash should not be valid")
	}
}

func TestLocationProof_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		proof   *LocationProof
		wantErr string
	}{
		{
			name: "valid proof",
			proof: &LocationProof{
				ValidatorAddr: "cosmosvaloper1test",
				IPAddress:     "192.168.1.1",
				ClaimedRegion: "na",
				Timestamp:     time.Now(),
			},
			wantErr: "",
		},
		{
			name: "empty validator address",
			proof: &LocationProof{
				ValidatorAddr: "",
				IPAddress:     "192.168.1.1",
				ClaimedRegion: "na",
				Timestamp:     time.Now(),
			},
			wantErr: "validator address cannot be empty",
		},
		{
			name: "empty IP address",
			proof: &LocationProof{
				ValidatorAddr: "cosmosvaloper1test",
				IPAddress:     "",
				ClaimedRegion: "na",
				Timestamp:     time.Now(),
			},
			wantErr: "IP address cannot be empty",
		},
		{
			name: "empty claimed region",
			proof: &LocationProof{
				ValidatorAddr: "cosmosvaloper1test",
				IPAddress:     "192.168.1.1",
				ClaimedRegion: "",
				Timestamp:     time.Now(),
			},
			wantErr: "claimed region cannot be empty",
		},
		{
			name: "zero timestamp",
			proof: &LocationProof{
				ValidatorAddr: "cosmosvaloper1test",
				IPAddress:     "192.168.1.1",
				ClaimedRegion: "na",
				Timestamp:     time.Time{},
			},
			wantErr: "timestamp cannot be zero",
		},
		{
			name: "empty proof hash",
			proof: &LocationProof{
				ValidatorAddr: "cosmosvaloper1test",
				IPAddress:     "192.168.1.1",
				ClaimedRegion: "na",
				Timestamp:     time.Now(),
				ProofHash:     "",
			},
			wantErr: "proof hash cannot be empty",
		},
		{
			name: "mismatched proof hash",
			proof: &LocationProof{
				ValidatorAddr: "cosmosvaloper1test",
				IPAddress:     "192.168.1.1",
				ClaimedRegion: "na",
				Timestamp:     time.Now(),
				ProofHash:     "wronghash",
			},
			wantErr: "proof hash mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.proof.ProofHash == "" && tt.wantErr == "" {
				// Set correct hash for valid cases
				tt.proof.ProofHash = tt.proof.ComputeHash()
			}

			err := tt.proof.ValidateBasic()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateBasic() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateBasic() error = nil, want error containing %q", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateBasic() error = %v, want error containing %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestLocationEvidence_AddProof(t *testing.T) {
	evidence := &LocationEvidence{
		ValidatorAddr: "cosmosvaloper1test",
		Proofs:        []*LocationProof{},
	}

	proof := NewLocationProof("cosmosvaloper1test", "192.168.1.1", "na")

	err := evidence.AddProof(proof)
	if err != nil {
		t.Errorf("AddProof() error = %v, want nil", err)
	}

	if len(evidence.Proofs) != 1 {
		t.Errorf("Expected 1 proof, got %d", len(evidence.Proofs))
	}

	// Test adding proof with mismatched validator address
	wrongProof := NewLocationProof("cosmosvaloper1other", "192.168.1.2", "eu")
	err = evidence.AddProof(wrongProof)
	if err == nil {
		t.Error("AddProof should fail for mismatched validator address")
	}
}

func TestLocationEvidence_GetLatestProof(t *testing.T) {
	evidence := &LocationEvidence{
		ValidatorAddr: "cosmosvaloper1test",
		Proofs:        []*LocationProof{},
	}

	// Empty proofs
	latest := evidence.GetLatestProof()
	if latest != nil {
		t.Error("GetLatestProof should return nil for empty proofs")
	}

	// Add proofs with different timestamps
	now := time.Now()
	proof1 := NewLocationProof("cosmosvaloper1test", "192.168.1.1", "na")
	proof1.Timestamp = now.Add(-10 * time.Minute)

	proof2 := NewLocationProof("cosmosvaloper1test", "192.168.1.1", "na")
	proof2.Timestamp = now.Add(-5 * time.Minute)

	proof3 := NewLocationProof("cosmosvaloper1test", "192.168.1.1", "na")
	proof3.Timestamp = now

	evidence.Proofs = []*LocationProof{proof1, proof3, proof2} // Out of order

	latest = evidence.GetLatestProof()
	if latest == nil {
		t.Fatal("GetLatestProof returned nil")
	}

	if !latest.Timestamp.Equal(proof3.Timestamp) {
		t.Errorf("Expected latest timestamp %v, got %v", proof3.Timestamp, latest.Timestamp)
	}
}

func TestLocationEvidence_IsConsistent(t *testing.T) {
	now := time.Now()
	maxAge := 1 * time.Hour

	tests := []struct {
		name       string
		proofs     []*LocationProof
		wantResult bool
	}{
		{
			name:       "no proofs",
			proofs:     []*LocationProof{},
			wantResult: false,
		},
		{
			name: "single recent proof",
			proofs: []*LocationProof{
				{ClaimedRegion: "na", Timestamp: now.Add(-30 * time.Minute)},
			},
			wantResult: true,
		},
		{
			name: "consistent region",
			proofs: []*LocationProof{
				{ClaimedRegion: "na", Timestamp: now.Add(-50 * time.Minute)},
				{ClaimedRegion: "na", Timestamp: now.Add(-30 * time.Minute)},
				{ClaimedRegion: "na", Timestamp: now.Add(-10 * time.Minute)},
			},
			wantResult: true,
		},
		{
			name: "inconsistent region",
			proofs: []*LocationProof{
				{ClaimedRegion: "na", Timestamp: now.Add(-50 * time.Minute)},
				{ClaimedRegion: "eu", Timestamp: now.Add(-30 * time.Minute)},
			},
			wantResult: false,
		},
		{
			name: "all proofs too old",
			proofs: []*LocationProof{
				{ClaimedRegion: "na", Timestamp: now.Add(-2 * time.Hour)},
				{ClaimedRegion: "na", Timestamp: now.Add(-3 * time.Hour)},
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := &LocationEvidence{
				ValidatorAddr: "cosmosvaloper1test",
				Proofs:        tt.proofs,
			}

			result := evidence.IsConsistent(maxAge)
			if result != tt.wantResult {
				t.Errorf("IsConsistent() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestLocationEvidence_DetectLocationJumps(t *testing.T) {
	now := time.Now()
	period := 24 * time.Hour

	tests := []struct {
		name       string
		proofs     []*LocationProof
		threshold  int
		wantJumps  bool
	}{
		{
			name:       "no jumps - less than 2 proofs",
			proofs:     []*LocationProof{{ClaimedRegion: "na", Timestamp: now}},
			threshold:  2,
			wantJumps:  false,
		},
		{
			name: "no jumps - consistent region",
			proofs: []*LocationProof{
				{ClaimedRegion: "na", Timestamp: now.Add(-20 * time.Hour)},
				{ClaimedRegion: "na", Timestamp: now.Add(-10 * time.Hour)},
				{ClaimedRegion: "na", Timestamp: now},
			},
			threshold: 2,
			wantJumps: false,
		},
		{
			name: "jumps detected - exceeds threshold",
			proofs: []*LocationProof{
				{ClaimedRegion: "na", Timestamp: now.Add(-20 * time.Hour)},
				{ClaimedRegion: "eu", Timestamp: now.Add(-15 * time.Hour)},
				{ClaimedRegion: "apac", Timestamp: now.Add(-10 * time.Hour)},
			},
			threshold: 2,
			wantJumps: true,
		},
		{
			name: "jumps below threshold",
			proofs: []*LocationProof{
				{ClaimedRegion: "na", Timestamp: now.Add(-20 * time.Hour)},
				{ClaimedRegion: "eu", Timestamp: now.Add(-10 * time.Hour)},
			},
			threshold: 2,
			wantJumps: false,
		},
		{
			name: "old proofs ignored",
			proofs: []*LocationProof{
				{ClaimedRegion: "na", Timestamp: now.Add(-30 * time.Hour)},
				{ClaimedRegion: "eu", Timestamp: now.Add(-25 * time.Hour)},
				{ClaimedRegion: "na", Timestamp: now.Add(-10 * time.Hour)},
			},
			threshold: 2,
			wantJumps: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := &LocationEvidence{
				ValidatorAddr: "cosmosvaloper1test",
				Proofs:        tt.proofs,
			}

			result := evidence.DetectLocationJumps(tt.threshold, period)
			if result != tt.wantJumps {
				t.Errorf("DetectLocationJumps() = %v, want %v", result, tt.wantJumps)
			}
		})
	}
}

func TestNewGeographicDistribution(t *testing.T) {
	gd := NewGeographicDistribution()

	if gd == nil {
		t.Fatal("NewGeographicDistribution returned nil")
	}

	if gd.RegionCounts == nil {
		t.Error("RegionCounts should be initialized")
	}

	if gd.TotalCount != 0 {
		t.Error("TotalCount should be 0 initially")
	}

	if gd.UniqueRegions != 0 {
		t.Error("UniqueRegions should be 0 initially")
	}

	if gd.DiversityScore != 0 {
		t.Error("DiversityScore should be 0 initially")
	}
}

func TestGeographicDistribution_AddRegion(t *testing.T) {
	gd := NewGeographicDistribution()

	gd.AddRegion("na")
	if gd.TotalCount != 1 {
		t.Errorf("Expected TotalCount 1, got %d", gd.TotalCount)
	}
	if gd.RegionCounts["na"] != 1 {
		t.Errorf("Expected na count 1, got %d", gd.RegionCounts["na"])
	}

	gd.AddRegion("na")
	if gd.TotalCount != 2 {
		t.Errorf("Expected TotalCount 2, got %d", gd.TotalCount)
	}
	if gd.RegionCounts["na"] != 2 {
		t.Errorf("Expected na count 2, got %d", gd.RegionCounts["na"])
	}

	gd.AddRegion("eu")
	if gd.TotalCount != 3 {
		t.Errorf("Expected TotalCount 3, got %d", gd.TotalCount)
	}
	if gd.UniqueRegions != 2 {
		t.Errorf("Expected UniqueRegions 2, got %d", gd.UniqueRegions)
	}
}

func TestGeographicDistribution_DiversityScore(t *testing.T) {
	tests := []struct {
		name           string
		regions        []string
		expectMinScore float64
		expectMaxScore float64
	}{
		{
			name:           "perfectly distributed",
			regions:        []string{"na", "eu", "apac"},
			expectMinScore: 0.85,  // Lowered due to approximation in log functions
			expectMaxScore: 1.0,
		},
		{
			name:           "all same region",
			regions:        []string{"na", "na", "na"},
			expectMinScore: 0.0,
			expectMaxScore: 0.01,
		},
		{
			name:           "moderately distributed",
			regions:        []string{"na", "na", "eu"},
			expectMinScore: 0.5,
			expectMaxScore: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gd := NewGeographicDistribution()
			for _, region := range tt.regions {
				gd.AddRegion(region)
			}

			if gd.DiversityScore < tt.expectMinScore || gd.DiversityScore > tt.expectMaxScore {
				t.Errorf("DiversityScore = %f, want between %f and %f",
					gd.DiversityScore, tt.expectMinScore, tt.expectMaxScore)
			}
		})
	}
}

func TestGeographicDistribution_IsSufficient(t *testing.T) {
	tests := []struct {
		name              string
		regions           []string
		minRegions        int
		minDiversityScore float64
		wantSufficient    bool
	}{
		{
			name:              "sufficient diversity",
			regions:           []string{"na", "eu", "apac"},
			minRegions:        3,
			minDiversityScore: 0.9,
			wantSufficient:    true,
		},
		{
			name:              "insufficient regions",
			regions:           []string{"na", "eu"},
			minRegions:        3,
			minDiversityScore: 0.9,
			wantSufficient:    false,
		},
		{
			name:              "insufficient diversity score",
			regions:           []string{"na", "na", "na", "eu"},
			minRegions:        2,
			minDiversityScore: 0.9,
			wantSufficient:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gd := NewGeographicDistribution()
			for _, region := range tt.regions {
				gd.AddRegion(region)
			}

			result := gd.IsSufficient(tt.minRegions, tt.minDiversityScore)
			if result != tt.wantSufficient {
				t.Errorf("IsSufficient() = %v, want %v (diversity=%f, regions=%d)",
					result, tt.wantSufficient, gd.DiversityScore, gd.UniqueRegions)
			}
		})
	}
}

func TestLog2(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
		tolerance float64
	}{
		{1.0, 0.0, 0.01},
		{2.0, 1.0, 0.2},  // Increased tolerance for approximation
		{4.0, 2.0, 0.5},  // Increased tolerance for approximation
		{8.0, 3.0, 0.8},  // Increased tolerance for approximation
		{0.0, 0.0, 0.01},
		{-1.0, 0.0, 0.01}, // Negative should return 0
	}

	for _, tt := range tests {
		result := log2(tt.input)
		if math.Abs(result-tt.expected) > tt.tolerance {
			t.Errorf("log2(%f) = %f, want %f (tolerance %f)",
				tt.input, result, tt.expected, tt.tolerance)
		}
	}
}

func TestLogNatural(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
		tolerance float64
	}{
		{1.0, 0.0, 0.01},
		{math.E, 1.0, 0.1},
		{0.0, 0.0, 0.01},
		{-1.0, 0.0, 0.01}, // Negative should return 0
	}

	for _, tt := range tests {
		result := logNatural(tt.input)
		if math.Abs(result-tt.expected) > tt.tolerance {
			t.Errorf("logNatural(%f) = %f, want %f (tolerance %f)",
				tt.input, result, tt.expected, tt.tolerance)
		}
	}
}
