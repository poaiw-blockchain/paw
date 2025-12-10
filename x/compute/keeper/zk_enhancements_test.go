package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestVerifyZKProof(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("empty proof returns error", func(t *testing.T) {
		err := k.VerifyZKProof(sdkCtx, []byte{}, []byte("result"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("empty result data returns error", func(t *testing.T) {
		err := k.VerifyZKProof(sdkCtx, []byte("proof"), []byte{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("result data too short returns error", func(t *testing.T) {
		err := k.VerifyZKProof(sdkCtx, []byte("proof"), make([]byte, 30))
		require.Error(t, err)
		require.Contains(t, err.Error(), "too short")
	})
}

func TestVerifyZKProofWithParams(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	provider := sdk.AccAddress([]byte("test_provider_addr_"))

	t.Run("nil proof returns error", func(t *testing.T) {
		valid, err := k.VerifyZKProofWithParams(sdkCtx, nil, 1, make([]byte, 32), provider)
		require.Error(t, err)
		require.False(t, valid)
		require.Contains(t, err.Error(), "nil")
	})

	t.Run("empty proof data returns error", func(t *testing.T) {
		zkProof := &types.ZKProof{
			Proof:       []byte{},
			ProofSystem: "groth16",
		}
		valid, err := k.VerifyZKProofWithParams(sdkCtx, zkProof, 1, make([]byte, 32), provider)
		require.Error(t, err)
		require.False(t, valid)
		require.Contains(t, err.Error(), "empty")
	})

	t.Run("invalid result hash size returns error", func(t *testing.T) {
		zkProof := &types.ZKProof{
			Proof:       []byte("proof"),
			ProofSystem: "groth16",
		}
		valid, err := k.VerifyZKProofWithParams(sdkCtx, zkProof, 1, make([]byte, 16), provider)
		require.Error(t, err)
		require.False(t, valid)
		require.Contains(t, err.Error(), "32 bytes")
	})

	t.Run("empty provider address returns error", func(t *testing.T) {
		zkProof := &types.ZKProof{
			Proof:       []byte("proof"),
			ProofSystem: "groth16",
		}
		valid, err := k.VerifyZKProofWithParams(sdkCtx, zkProof, 1, make([]byte, 32), sdk.AccAddress{})
		require.Error(t, err)
		require.False(t, valid)
		require.Contains(t, err.Error(), "empty")
	})
}

func TestValidateZKProofVersion(t *testing.T) {
	k, _ := setupKeeperForTest(t)

	t.Run("nil proof returns nil (version check disabled)", func(t *testing.T) {
		err := k.ValidateZKProofVersion(nil)
		require.NoError(t, err)
	})

	t.Run("empty proof returns nil", func(t *testing.T) {
		zkProof := &types.ZKProof{}
		err := k.ValidateZKProofVersion(zkProof)
		require.NoError(t, err)
	})

	t.Run("valid proof returns nil", func(t *testing.T) {
		zkProof := &types.ZKProof{
			Proof:       []byte("proof"),
			ProofSystem: "groth16",
		}
		err := k.ValidateZKProofVersion(zkProof)
		require.NoError(t, err)
	})
}

func TestVerifyTrustedSetup(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	t.Run("unknown circuit returns error", func(t *testing.T) {
		err := k.VerifyTrustedSetup(ctx, "unknown-circuit")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
}

func TestGetTrustedSetup(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	t.Run("unknown circuit returns error", func(t *testing.T) {
		_, err := k.GetTrustedSetup(ctx, "unknown")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
}

func TestValidateCircuitConstraints(t *testing.T) {
	k, _ := setupKeeperForTest(t)

	t.Run("known circuit with matching constraints", func(t *testing.T) {
		err := k.ValidateCircuitConstraints("compute_verification_v1", 50000)
		require.NoError(t, err)
	})

	t.Run("known circuit with mismatched constraints", func(t *testing.T) {
		err := k.ValidateCircuitConstraints("compute_verification_v1", 99999)
		require.Error(t, err)
		require.Contains(t, err.Error(), "mismatch")
	})

	t.Run("unknown circuit returns error", func(t *testing.T) {
		err := k.ValidateCircuitConstraints("unknown_circuit", 50000)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown circuit")
	})
}

func TestGetCircuitConstraintCount(t *testing.T) {
	k, _ := setupKeeperForTest(t)

	t.Run("compute_verification_v1", func(t *testing.T) {
		count, err := k.GetCircuitConstraintCount("compute_verification_v1")
		require.NoError(t, err)
		require.Equal(t, uint64(50000), count)
	})

	t.Run("compute_verification_v2", func(t *testing.T) {
		count, err := k.GetCircuitConstraintCount("compute_verification_v2")
		require.NoError(t, err)
		require.Equal(t, uint64(75000), count)
	})

	t.Run("result_aggregation", func(t *testing.T) {
		count, err := k.GetCircuitConstraintCount("result_aggregation")
		require.NoError(t, err)
		require.Equal(t, uint64(100000), count)
	})

	t.Run("unknown circuit", func(t *testing.T) {
		_, err := k.GetCircuitConstraintCount("unknown")
		require.Error(t, err)
	})
}

func TestValidatePublicInputs(t *testing.T) {
	k, _ := setupKeeperForTest(t)

	t.Run("matching input count", func(t *testing.T) {
		zkProof := &types.ZKProof{
			PublicInputs: make([]byte, 64),
		}
		err := k.ValidatePublicInputs(zkProof, 64)
		require.NoError(t, err)
	})

	t.Run("mismatched input count", func(t *testing.T) {
		zkProof := &types.ZKProof{
			PublicInputs: make([]byte, 32),
		}
		err := k.ValidatePublicInputs(zkProof, 64)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid public input count")
	})

	t.Run("zero expected with empty inputs", func(t *testing.T) {
		zkProof := &types.ZKProof{
			PublicInputs: []byte{},
		}
		err := k.ValidatePublicInputs(zkProof, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "required")
	})
}

func TestGetCachedProofVerification(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	t.Run("cache miss returns error", func(t *testing.T) {
		_, err := k.GetCachedProofVerification(ctx, "nonexistent_hash")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not in cache")
	})
}

func TestCacheProofVerification(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	t.Run("cache proof verification entry", func(t *testing.T) {
		proofHash := "test_hash_123"
		err := k.CacheProofVerification(ctx, proofHash, true, 500000, 1)
		require.NoError(t, err)

		// Verify it's cached
		cached, err := k.GetCachedProofVerification(ctx, proofHash)
		require.NoError(t, err)
		require.NotNil(t, cached)
		require.True(t, cached.Verified)
	})
}

func TestVerifyProofWithCache(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	t.Run("nil proof returns error", func(t *testing.T) {
		valid, err := k.VerifyProofWithCache(ctx, nil, 1)
		require.Error(t, err)
		require.False(t, valid)
	})

	t.Run("returns cached result if available", func(t *testing.T) {
		zkProof := &types.ZKProof{
			Proof:        []byte("cached_proof_test_data"),
			PublicInputs: make([]byte, 60),
			ProofSystem:  "groth16",
		}

		proofHash := k.calculateProofHash(zkProof)
		err := k.CacheProofVerification(ctx, proofHash, true, 500000, 1)
		require.NoError(t, err)

		valid, err := k.VerifyProofWithCache(ctx, zkProof, 1)
		require.NoError(t, err)
		require.True(t, valid)
	})
}

func TestScheduleKeyRotation(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	t.Run("schedule rotation with valid interval", func(t *testing.T) {
		interval := 24 * time.Hour
		err := k.ScheduleKeyRotation(ctx, "compute_verification_v1", interval)
		require.NoError(t, err)
	})

	t.Run("schedule rotation with short interval", func(t *testing.T) {
		interval := time.Minute
		err := k.ScheduleKeyRotation(ctx, "compute_verification_v1", interval)
		require.NoError(t, err)
	})
}

func TestCheckAndPerformKeyRotation(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	t.Run("no scheduled rotation", func(t *testing.T) {
		err := k.CheckAndPerformKeyRotation(ctx)
		require.NoError(t, err)
	})
}

func TestInitializeMPCCeremony(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	t.Run("initialize ceremony with valid params", func(t *testing.T) {
		minContributions := 3
		circuitID := "test_circuit"

		ceremony, err := k.InitializeMPCCeremony(ctx, circuitID, minContributions)
		require.NoError(t, err)
		require.NotNil(t, ceremony)
		require.NotEmpty(t, ceremony.ID)
		require.Equal(t, circuitID, ceremony.CircuitID)
		require.Equal(t, minContributions, ceremony.MinContributions)
	})

	t.Run("initialize ceremony with different circuit", func(t *testing.T) {
		ceremony, err := k.InitializeMPCCeremony(ctx, "another_circuit", 2)
		require.NoError(t, err)
		require.NotNil(t, ceremony)
	})
}

func TestCleanupOldProofCache(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	t.Run("cleanup with no entries", func(t *testing.T) {
		err := k.CleanupOldProofCache(ctx, time.Hour)
		require.NoError(t, err)
	})

	t.Run("cleanup with zero retention period", func(t *testing.T) {
		err := k.CleanupOldProofCache(ctx, 0)
		require.NoError(t, err)
	})
}

func TestCalculateProofHash(t *testing.T) {
	k, _ := setupKeeperForTest(t)

	t.Run("different proofs produce different hashes", func(t *testing.T) {
		proof1 := &types.ZKProof{Proof: []byte("proof1")}
		proof2 := &types.ZKProof{Proof: []byte("proof2")}
		hash1 := k.calculateProofHash(proof1)
		hash2 := k.calculateProofHash(proof2)
		require.NotEqual(t, hash1, hash2)
	})

	t.Run("same proof produces same hash", func(t *testing.T) {
		proof := &types.ZKProof{Proof: []byte("same_proof")}
		hash1 := k.calculateProofHash(proof)
		hash2 := k.calculateProofHash(proof)
		require.Equal(t, hash1, hash2)
	})

	t.Run("empty proof data", func(t *testing.T) {
		// Empty proof (not nil) - internal function assumes non-nil proof
		// Nil proof protection is handled by VerifyProofWithCache
		proof := &types.ZKProof{Proof: []byte{}}
		hash := k.calculateProofHash(proof)
		require.NotEmpty(t, hash)
	})
}

func TestZKProofVersionConstants(t *testing.T) {
	require.Equal(t, "1.0", ZKProofVersion1)
	require.Equal(t, "2.0", ZKProofVersion2)
	require.Equal(t, ZKProofVersion2, CurrentZKVersion)
}

func TestProofCacheEntry_Struct(t *testing.T) {
	entry := ProofCacheEntry{
		ProofHash:       "test_hash",
		Verified:        true,
		VerifiedAt:      time.Now(),
		VerificationGas: 500000,
		RequestID:       123,
	}
	require.NotEmpty(t, entry.ProofHash)
	require.True(t, entry.Verified)
	require.Equal(t, uint64(500000), entry.VerificationGas)
	require.Equal(t, uint64(123), entry.RequestID)
}

func TestTrustedSetup_Struct(t *testing.T) {
	setup := TrustedSetup{
		CircuitID:         "test_circuit",
		SetupHash:         "hash123",
		Contributors:      []string{"addr1", "addr2"},
		ContributionCount: 2,
		Finalized:         true,
	}
	require.NotEmpty(t, setup.CircuitID)
	require.NotEmpty(t, setup.SetupHash)
	require.Len(t, setup.Contributors, 2)
	require.True(t, setup.Finalized)
}
