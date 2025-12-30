package keeper

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

// TestVersionConstants verifies version constants are defined.
func TestVersionConstants(t *testing.T) {
	require.Equal(t, "v1.0.0", OracleKeeperVersion)
	require.Equal(t, "v1.0.0", DexKeeperVersion)
	require.Equal(t, "v1.0.0", ComputeKeeperVersion)
}

// TestPriceInfoStruct tests PriceInfo data structure.
func TestPriceInfoStruct(t *testing.T) {
	info := PriceInfo{
		Asset:       "BTC/USD",
		Price:       sdkmath.LegacyNewDec(50000),
		BlockHeight: 12345,
	}

	require.Equal(t, "BTC/USD", info.Asset)
	require.True(t, info.Price.Equal(sdkmath.LegacyNewDec(50000)))
	require.Equal(t, int64(12345), info.BlockHeight)
}

// TestPoolInfoStruct tests PoolInfo data structure.
func TestPoolInfoStruct(t *testing.T) {
	info := PoolInfo{
		PoolID:      1,
		TokenA:      "upaw",
		TokenB:      "uatom",
		ReserveA:    sdkmath.NewInt(1000000),
		ReserveB:    sdkmath.NewInt(500000),
		TotalShares: sdkmath.NewInt(750000),
	}

	require.Equal(t, uint64(1), info.PoolID)
	require.Equal(t, "upaw", info.TokenA)
	require.Equal(t, "uatom", info.TokenB)
	require.True(t, info.ReserveA.Equal(sdkmath.NewInt(1000000)))
	require.True(t, info.ReserveB.Equal(sdkmath.NewInt(500000)))
	require.True(t, info.TotalShares.Equal(sdkmath.NewInt(750000)))
}

// TestProviderInfoStruct tests ProviderInfo data structure.
func TestProviderInfoStruct(t *testing.T) {
	info := ProviderInfo{
		Address:    nil, // AccAddress requires proper initialization
		Stake:      sdkmath.NewInt(1000000),
		Reputation: 95,
		IsActive:   true,
	}

	require.True(t, info.Stake.Equal(sdkmath.NewInt(1000000)))
	require.Equal(t, uint64(95), info.Reputation)
	require.True(t, info.IsActive)
}

// TestRequestInfoStruct tests RequestInfo data structure.
func TestRequestInfoStruct(t *testing.T) {
	info := RequestInfo{
		RequestID:  123,
		Requester:  nil,
		Provider:   nil,
		Status:     "PENDING",
		MaxPayment: sdkmath.NewInt(5000),
	}

	require.Equal(t, uint64(123), info.RequestID)
	require.Equal(t, "PENDING", info.Status)
	require.True(t, info.MaxPayment.Equal(sdkmath.NewInt(5000)))
}

// TestInterfaceNilSafety verifies interfaces can be nil-checked.
func TestInterfaceNilSafety(t *testing.T) {
	var oracleKeeper OracleKeeperV1
	require.Nil(t, oracleKeeper)

	var dexKeeper DexKeeperV1
	require.Nil(t, dexKeeper)

	var computeKeeper ComputeKeeperV1
	require.Nil(t, computeKeeper)
}

// TestInterfaceCompatibility verifies extended interfaces embed base interfaces.
func TestInterfaceCompatibility(t *testing.T) {
	// This is a compile-time check - if it compiles, interfaces are compatible
	// OracleKeeperV1Extended embeds OracleKeeperV1
	var _ OracleKeeperV1 = (OracleKeeperV1Extended)(nil)

	// DexKeeperV1Extended embeds DexKeeperV1
	var _ DexKeeperV1 = (DexKeeperV1Extended)(nil)

	// ComputeKeeperV1Extended embeds ComputeKeeperV1
	var _ ComputeKeeperV1 = (ComputeKeeperV1Extended)(nil)
}
