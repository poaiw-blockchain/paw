package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/dex/keeper"
)

func TestComputeSwapCommitmentHash(t *testing.T) {
	trader := sdk.AccAddress([]byte("test_trader_address_"))
	poolID := uint64(1)
	tokenIn := "tokenA"
	tokenOut := "tokenB"
	amountIn := math.NewInt(1000000)
	minAmountOut := math.NewInt(900000)
	salt := []byte("random_salt_12345678")

	// Compute hash
	hash1 := keeper.ComputeSwapCommitmentHash(poolID, tokenIn, tokenOut, amountIn, minAmountOut, salt, trader)
	require.NotNil(t, hash1)
	require.Len(t, hash1, 32) // SHA256 produces 32 bytes

	// Same parameters should produce same hash
	hash2 := keeper.ComputeSwapCommitmentHash(poolID, tokenIn, tokenOut, amountIn, minAmountOut, salt, trader)
	require.Equal(t, hash1, hash2)

	// Different salt should produce different hash
	differentSalt := []byte("different_salt_val__")
	hash3 := keeper.ComputeSwapCommitmentHash(poolID, tokenIn, tokenOut, amountIn, minAmountOut, differentSalt, trader)
	require.NotEqual(t, hash1, hash3)

	// Different amount should produce different hash
	differentAmount := math.NewInt(2000000)
	hash4 := keeper.ComputeSwapCommitmentHash(poolID, tokenIn, tokenOut, differentAmount, minAmountOut, salt, trader)
	require.NotEqual(t, hash1, hash4)

	// Different trader should produce different hash
	differentTrader := sdk.AccAddress([]byte("different_trader____"))
	hash5 := keeper.ComputeSwapCommitmentHash(poolID, tokenIn, tokenOut, amountIn, minAmountOut, salt, differentTrader)
	require.NotEqual(t, hash1, hash5)
}

func TestSwapCommitmentKey(t *testing.T) {
	hash := []byte("test_commitment_hash_1234567890_")
	key := keeper.SwapCommitmentKey(hash)

	require.NotNil(t, key)
	require.True(t, len(key) > len(hash))
	// Key should start with prefix
	require.Equal(t, keeper.SwapCommitmentKeyPrefix, key[:len(keeper.SwapCommitmentKeyPrefix)])
}

func TestSwapCommitmentByExpiryKey(t *testing.T) {
	expiryBlock := int64(100)
	hash := []byte("test_hash_12345678901234567890__")

	key := keeper.SwapCommitmentByExpiryKey(expiryBlock, hash)
	require.NotNil(t, key)
	// Key should contain prefix + 8 bytes for block height + hash
	expectedMinLen := len(keeper.SwapCommitmentByExpiryPrefix) + 8 + len(hash)
	require.Equal(t, expectedMinLen, len(key))
}

func TestSwapCommitmentByTraderKey(t *testing.T) {
	trader := sdk.AccAddress([]byte("test_trader_address_"))
	hash := []byte("test_hash_12345678901234567890__")

	key := keeper.SwapCommitmentByTraderKey(trader, hash)
	require.NotNil(t, key)
	// Key should contain prefix + trader address + hash
	expectedMinLen := len(keeper.SwapCommitmentByTraderPrefix) + len(trader.Bytes()) + len(hash)
	require.Equal(t, expectedMinLen, len(key))
}

func TestCommitRevealConstants(t *testing.T) {
	// Verify constants are reasonable
	require.Equal(t, "0.05", keeper.LargeSwapThresholdPercent)
	require.Equal(t, int64(2), keeper.RevealDelayBlocks)
	require.Equal(t, int64(50), keeper.CommitExpiryBlocks)
	require.Equal(t, int64(1000000), keeper.CommitDepositAmount)

	// Reveal delay should be less than expiry
	require.Less(t, keeper.RevealDelayBlocks, keeper.CommitExpiryBlocks)
}
