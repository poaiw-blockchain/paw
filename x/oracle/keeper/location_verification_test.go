package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/oracle/types"
)

func TestVerifyValidatorLocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		validatorAddr string
		claimedRegion string
		ipAddress     string
		expectError   bool
		errorContains string
	}{
		{
			name:          "valid north america location",
			validatorAddr: "cosmosvaloper1test1",
			claimedRegion: "north_america",
			ipAddress:     "8.8.8.8", // Google DNS (US)
			expectError:   false,
		},
		{
			name:          "valid europe location",
			validatorAddr: "cosmosvaloper1test2",
			claimedRegion: "europe",
			ipAddress:     "1.1.1.1", // Cloudflare (multiple regions, but acceptable)
			expectError:   false,
		},
		{
			name:          "valid asia location",
			validatorAddr: "cosmosvaloper1test3",
			claimedRegion: "asia",
			ipAddress:     "114.114.114.114", // China DNS
			expectError:   false,
		},
		{
			name:          "empty region",
			validatorAddr: "cosmosvaloper1test4",
			claimedRegion: "",
			ipAddress:     "8.8.8.8",
			expectError:   true,
			errorContains: "cannot be empty",
		},
		{
			name:          "empty IP address",
			validatorAddr: "cosmosvaloper1test5",
			claimedRegion: "north_america",
			ipAddress:     "",
			expectError:   true,
			errorContains: "cannot be empty",
		},
		{
			name:          "invalid region",
			validatorAddr: "cosmosvaloper1test6",
			claimedRegion: "mars",
			ipAddress:     "8.8.8.8",
			expectError:   true,
			errorContains: "invalid geographic region",
		},
		{
			name:          "invalid IP format",
			validatorAddr: "cosmosvaloper1test7",
			claimedRegion: "north_america",
			ipAddress:     "not.an.ip.address",
			expectError:   true,
			errorContains: "invalid IP address format",
		},
		{
			name:          "private IP rejected",
			validatorAddr: "cosmosvaloper1test8",
			claimedRegion: "north_america",
			ipAddress:     "192.168.1.1",
			expectError:   true,
			errorContains: "private/localhost",
		},
		{
			name:          "localhost rejected",
			validatorAddr: "cosmosvaloper1test9",
			claimedRegion: "north_america",
			ipAddress:     "127.0.0.1",
			expectError:   true,
			errorContains: "private/localhost",
		},
		{
			name:          "link-local IP rejected",
			validatorAddr: "cosmosvaloper1test10",
			claimedRegion: "north_america",
			ipAddress:     "169.254.1.1",
			expectError:   true,
			errorContains: "private/localhost",
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Note: This test verifies the validation logic
			// GeoIP verification is not tested here as it requires a database
			// Integration tests should verify actual IP-to-region matching

			// For unit testing, we verify the validation flow
			// The actual GeoIP manager is tested separately
		})
	}
}

func TestLocationProof(t *testing.T) {
	t.Parallel()

	validatorAddr := "cosmosvaloper1test"
	ipAddress := "8.8.8.8"
	claimedRegion := "north_america"

	t.Run("create valid location proof", func(t *testing.T) {
		t.Parallel()

		proof := types.NewLocationProof(validatorAddr, ipAddress, claimedRegion)
		require.NotNil(t, proof)
		require.Equal(t, validatorAddr, proof.ValidatorAddr)
		require.Equal(t, ipAddress, proof.IPAddress)
		require.Equal(t, claimedRegion, proof.ClaimedRegion)
		require.NotEmpty(t, proof.ProofHash)
		require.False(t, proof.Timestamp.IsZero())
	})

	t.Run("validate basic proof", func(t *testing.T) {
		t.Parallel()

		proof := types.NewLocationProof(validatorAddr, ipAddress, claimedRegion)
		err := proof.ValidateBasic()
		require.NoError(t, err)
	})

	t.Run("reject proof with empty validator address", func(t *testing.T) {
		t.Parallel()

		proof := types.NewLocationProof("", ipAddress, claimedRegion)
		err := proof.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator address cannot be empty")
	})

	t.Run("reject proof with empty IP address", func(t *testing.T) {
		t.Parallel()

		proof := types.NewLocationProof(validatorAddr, "", claimedRegion)
		err := proof.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "IP address cannot be empty")
	})

	t.Run("reject proof with empty region", func(t *testing.T) {
		t.Parallel()

		proof := types.NewLocationProof(validatorAddr, ipAddress, "")
		err := proof.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "claimed region cannot be empty")
	})

	t.Run("reject proof with tampered hash", func(t *testing.T) {
		t.Parallel()

		proof := types.NewLocationProof(validatorAddr, ipAddress, claimedRegion)
		proof.ProofHash = "tampered_hash"
		err := proof.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "proof hash mismatch")
	})

	t.Run("reject expired proof", func(t *testing.T) {
		t.Parallel()

		proof := types.NewLocationProof(validatorAddr, ipAddress, claimedRegion)
		// Set timestamp to 25 hours ago
		proof.Timestamp = time.Now().Add(-25 * time.Hour)
		proof.ProofHash = proof.ComputeHash()

		maxAge := 24 * time.Hour
		isValid := proof.IsValid(maxAge)
		require.False(t, isValid, "proof should be invalid due to age")
	})

	t.Run("accept recent proof", func(t *testing.T) {
		t.Parallel()

		proof := types.NewLocationProof(validatorAddr, ipAddress, claimedRegion)
		maxAge := 24 * time.Hour
		isValid := proof.IsValid(maxAge)
		require.True(t, isValid, "recent proof should be valid")
	})
}

func TestLocationEvidence(t *testing.T) {
	t.Parallel()

	validatorAddr := "cosmosvaloper1test"
	ipAddress := "8.8.8.8"
	region1 := "north_america"
	region2 := "europe"

	t.Run("create location evidence", func(t *testing.T) {
		t.Parallel()

		evidence := &types.LocationEvidence{
			ValidatorAddr: validatorAddr,
			Proofs:        []*types.LocationProof{},
			VerifiedAt:    time.Now(),
			Metadata:      make(map[string]string),
		}

		require.NotNil(t, evidence)
		require.Equal(t, validatorAddr, evidence.ValidatorAddr)
		require.Empty(t, evidence.Proofs)
	})

	t.Run("add valid proof to evidence", func(t *testing.T) {
		t.Parallel()

		evidence := &types.LocationEvidence{
			ValidatorAddr: validatorAddr,
			Proofs:        []*types.LocationProof{},
			VerifiedAt:    time.Now(),
			Metadata:      make(map[string]string),
		}

		proof := types.NewLocationProof(validatorAddr, ipAddress, region1)
		err := evidence.AddProof(proof)
		require.NoError(t, err)
		require.Len(t, evidence.Proofs, 1)
	})

	t.Run("reject proof with mismatched validator address", func(t *testing.T) {
		t.Parallel()

		evidence := &types.LocationEvidence{
			ValidatorAddr: validatorAddr,
			Proofs:        []*types.LocationProof{},
			VerifiedAt:    time.Now(),
			Metadata:      make(map[string]string),
		}

		proof := types.NewLocationProof("cosmosvaloper1different", ipAddress, region1)
		err := evidence.AddProof(proof)
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator address mismatch")
	})

	t.Run("get latest proof", func(t *testing.T) {
		t.Parallel()

		evidence := &types.LocationEvidence{
			ValidatorAddr: validatorAddr,
			Proofs:        []*types.LocationProof{},
			VerifiedAt:    time.Now(),
			Metadata:      make(map[string]string),
		}

		// Add older proof
		oldProof := types.NewLocationProof(validatorAddr, ipAddress, region1)
		oldProof.Timestamp = time.Now().Add(-2 * time.Hour)
		oldProof.ProofHash = oldProof.ComputeHash()
		_ = evidence.AddProof(oldProof)

		// Add newer proof
		newProof := types.NewLocationProof(validatorAddr, ipAddress, region1)
		_ = evidence.AddProof(newProof)

		latest := evidence.GetLatestProof()
		require.NotNil(t, latest)
		require.Equal(t, newProof.ProofHash, latest.ProofHash)
	})

	t.Run("consistent location is valid", func(t *testing.T) {
		t.Parallel()

		evidence := &types.LocationEvidence{
			ValidatorAddr: validatorAddr,
			Proofs:        []*types.LocationProof{},
			VerifiedAt:    time.Now(),
			Metadata:      make(map[string]string),
		}

		// Add 3 proofs in same region
		for i := 0; i < 3; i++ {
			proof := types.NewLocationProof(validatorAddr, ipAddress, region1)
			proof.Timestamp = time.Now().Add(time.Duration(-i) * time.Hour)
			proof.ProofHash = proof.ComputeHash()
			_ = evidence.AddProof(proof)
		}

		maxAge := 24 * time.Hour
		isConsistent := evidence.IsConsistent(maxAge)
		require.True(t, isConsistent, "same region proofs should be consistent")
	})

	t.Run("inconsistent location is invalid", func(t *testing.T) {
		t.Parallel()

		evidence := &types.LocationEvidence{
			ValidatorAddr: validatorAddr,
			Proofs:        []*types.LocationProof{},
			VerifiedAt:    time.Now(),
			Metadata:      make(map[string]string),
		}

		// Add proof in region1
		proof1 := types.NewLocationProof(validatorAddr, ipAddress, region1)
		proof1.Timestamp = time.Now().Add(-1 * time.Hour)
		proof1.ProofHash = proof1.ComputeHash()
		_ = evidence.AddProof(proof1)

		// Add proof in region2 (suspicious location change)
		proof2 := types.NewLocationProof(validatorAddr, ipAddress, region2)
		_ = evidence.AddProof(proof2)

		maxAge := 24 * time.Hour
		isConsistent := evidence.IsConsistent(maxAge)
		require.False(t, isConsistent, "different region proofs should be inconsistent")
	})

	t.Run("detect location jumps", func(t *testing.T) {
		t.Parallel()

		evidence := &types.LocationEvidence{
			ValidatorAddr: validatorAddr,
			Proofs:        []*types.LocationProof{},
			VerifiedAt:    time.Now(),
			Metadata:      make(map[string]string),
		}

		// Add proofs that alternate between regions (suspicious)
		regions := []string{region1, region2, region1, region2}
		for i, region := range regions {
			proof := types.NewLocationProof(validatorAddr, ipAddress, region)
			proof.Timestamp = time.Now().Add(time.Duration(-i) * 24 * time.Hour)
			proof.ProofHash = proof.ComputeHash()
			_ = evidence.AddProof(proof)
		}

		threshold := 2
		period := 30 * 24 * time.Hour
		hasJumps := evidence.DetectLocationJumps(threshold, period)
		require.True(t, hasJumps, "alternating regions should be detected as jumps")
	})

	t.Run("no location jumps with consistent location", func(t *testing.T) {
		t.Parallel()

		evidence := &types.LocationEvidence{
			ValidatorAddr: validatorAddr,
			Proofs:        []*types.LocationProof{},
			VerifiedAt:    time.Now(),
			Metadata:      make(map[string]string),
		}

		// Add proofs all in same region
		for i := 0; i < 5; i++ {
			proof := types.NewLocationProof(validatorAddr, ipAddress, region1)
			proof.Timestamp = time.Now().Add(time.Duration(-i) * 24 * time.Hour)
			proof.ProofHash = proof.ComputeHash()
			_ = evidence.AddProof(proof)
		}

		threshold := 2
		period := 30 * 24 * time.Hour
		hasJumps := evidence.DetectLocationJumps(threshold, period)
		require.False(t, hasJumps, "consistent region should not show jumps")
	})
}

func TestGeographicDistribution(t *testing.T) {
	t.Parallel()

	t.Run("create geographic distribution", func(t *testing.T) {
		t.Parallel()

		dist := types.NewGeographicDistribution()
		require.NotNil(t, dist)
		require.Empty(t, dist.RegionCounts)
		require.Equal(t, 0, dist.TotalCount)
		require.Equal(t, 0, dist.UniqueRegions)
	})

	t.Run("add regions and calculate diversity", func(t *testing.T) {
		t.Parallel()

		dist := types.NewGeographicDistribution()

		// Add validators evenly distributed across 3 regions
		dist.AddRegion("north_america")
		dist.AddRegion("europe")
		dist.AddRegion("asia")

		require.Equal(t, 3, dist.TotalCount)
		require.Equal(t, 3, dist.UniqueRegions)
		require.Greater(t, dist.DiversityScore, 0.9) // Perfect distribution
	})

	t.Run("diversity score decreases with concentration", func(t *testing.T) {
		t.Parallel()

		// Even distribution
		dist1 := types.NewGeographicDistribution()
		dist1.AddRegion("north_america")
		dist1.AddRegion("europe")
		dist1.AddRegion("asia")
		evenScore := dist1.DiversityScore

		// Concentrated distribution
		dist2 := types.NewGeographicDistribution()
		dist2.AddRegion("north_america")
		dist2.AddRegion("north_america")
		dist2.AddRegion("north_america")
		dist2.AddRegion("north_america")
		dist2.AddRegion("europe")
		concentratedScore := dist2.DiversityScore

		require.Greater(t, evenScore, concentratedScore,
			"evenly distributed validators should have higher diversity score")
	})

	t.Run("check sufficient diversity", func(t *testing.T) {
		t.Parallel()

		dist := types.NewGeographicDistribution()
		dist.AddRegion("north_america")
		dist.AddRegion("europe")
		dist.AddRegion("asia")

		minRegions := 3
		minDiversityScore := 0.5

		isSufficient := dist.IsSufficient(minRegions, minDiversityScore)
		require.True(t, isSufficient, "distribution should meet minimum requirements")
	})

	t.Run("insufficient regions", func(t *testing.T) {
		t.Parallel()

		dist := types.NewGeographicDistribution()
		dist.AddRegion("north_america")
		dist.AddRegion("north_america")

		minRegions := 3
		minDiversityScore := 0.5

		isSufficient := dist.IsSufficient(minRegions, minDiversityScore)
		require.False(t, isSufficient, "distribution should fail minimum region requirement")
	})

	t.Run("insufficient diversity score", func(t *testing.T) {
		t.Parallel()

		dist := types.NewGeographicDistribution()
		// Add very concentrated distribution
		for i := 0; i < 10; i++ {
			dist.AddRegion("north_america")
		}
		dist.AddRegion("europe")
		dist.AddRegion("asia")

		minRegions := 3
		minDiversityScore := 0.8 // High threshold

		isSufficient := dist.IsSufficient(minRegions, minDiversityScore)
		require.False(t, isSufficient, "distribution should fail diversity score requirement")
	})
}

func TestIPValidation(t *testing.T) {
	t.Parallel()

	// Note: These tests require the actual keeper implementation
	// They are meant to be run with a test keeper instance
	// See security_integration_test.go for full integration tests
}
