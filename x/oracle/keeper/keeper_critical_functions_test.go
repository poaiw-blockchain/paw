package keeper_test

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// AggregateCrossChainPrices: use cached remote prices; ensure weighting & empty error paths.
func TestAggregateCrossChainPrices(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	sdkCtx := ctx

	// Seed two active sources with cached prices
	require.NoError(t, k.RegisterCrossChainOracleSource(sdkCtx, "osmosis-1", "band", "conn-0", "channel-0"))
	require.NoError(t, k.RegisterCrossChainOracleSource(sdkCtx, "cosmoshub-4", "band", "conn-1", "channel-1"))

	// Boost reputation to non-zero to avoid zero-weight panic
	store := sdkCtx.KVStore(k.GetStoreKey())
	setSource := func(chain string) {
		src := keeper.CrossChainOracleSource{
			ChainID:           chain,
			OracleType:        "band",
			ConnectionID:      "conn",
			ChannelID:         "chan",
			Reputation:        math.LegacyNewDec(1),
			LastHeartbeat:     sdkCtx.BlockTime(),
			TotalQueries:      1,
			SuccessfulQueries: 1,
			Active:            true,
		}
		bz, err := json.Marshal(src)
		require.NoError(t, err)
		store.Set([]byte("oracle_source_"+chain), bz)
	}
	setSource("osmosis-1")
	setSource("cosmoshub-4")

	seed := func(chain string, price int64, conf int64) {
		k.StoreCachedPrice(sdkCtx, "ATOM", chain, &keeper.CrossChainPriceData{ // helper exposed via keeper alias in tests
			Source:      chain,
			Symbol:      "ATOM",
			Price:       math.LegacyNewDec(price),
			Timestamp:   sdkCtx.BlockTime(),
			Confidence:  math.LegacyNewDec(conf),
			OracleCount: 3,
		})
	}
	seed("osmosis-1", 100, 1)
	seed("cosmoshub-4", 104, 1)

	agg, err := k.AggregateCrossChainPrices(ctx, "ATOM")
	require.NoError(t, err)
	require.True(t, agg.WeightedPrice.GT(math.LegacyNewDec(100)))
	require.True(t, agg.WeightedPrice.LT(math.LegacyNewDec(104)))

	// Remove all cached prices -> expect error
	k.ClearCachedPrice(sdkCtx, "ATOM", "osmosis-1")
	k.ClearCachedPrice(sdkCtx, "ATOM", "cosmoshub-4")
	_, err = k.AggregateCrossChainPrices(ctx, "ATOM")
	require.Error(t, err)
}

// DetectCollusionPatterns security-critical: identical prices and near-simultaneous submissions.
func TestDetectCollusionPatterns(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	flagged, _ := k.DetectCollusionPatterns(ctx, "ATOM", []types.ValidatorPrice{
		{ValidatorAddr: "val1", Asset: "ATOM", Price: math.LegacyNewDec(100)},
		{ValidatorAddr: "val2", Asset: "ATOM", Price: math.LegacyNewDec(100)},
		{ValidatorAddr: "val3", Asset: "ATOM", Price: math.LegacyNewDec(100)},
	})
	require.True(t, flagged)

	flagged, _ = k.DetectCollusionPatterns(ctx, "ATOM", []types.ValidatorPrice{
		{ValidatorAddr: "val1", Asset: "ATOM", Price: math.LegacyNewDec(100)},
		{ValidatorAddr: "val2", Asset: "ATOM", Price: math.LegacyNewDec(101)},
		{ValidatorAddr: "val3", Asset: "ATOM", Price: math.LegacyNewDec(99)},
	})
	require.False(t, flagged)
}

// ImplementFlashLoanDetection: detect spike and ignore normal volatility.
func TestImplementFlashLoanDetection(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	flagged, err := k.ImplementFlashLoanDetection(ctx, "ATOM", math.LegacyNewDec(500))
	require.NoError(t, err)
	require.True(t, flagged)

	flagged, err = k.ImplementFlashLoanDetection(ctx, "ATOM", math.LegacyNewDec(102))
	require.NoError(t, err)
	require.False(t, flagged)
}

// TrackGeographicDiversity: concentrated vs diversified.
func TestTrackGeographicDiversity(t *testing.T) {
	k, sk, ctx := keepertest.OracleKeeper(t)

	set := func(valAddr string, region string) {
		addr := sdk.ValAddress([]byte(valAddr))
		require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, sk, addr))
		require.NoError(t, k.SetValidatorOracle(ctx, types.ValidatorOracle{ValidatorAddr: addr.String(), GeographicRegion: region, IsActive: true}))
	}

	set("val1________________", "NA")
	set("val2________________", "EU")
	set("val3________________", "APAC")
	set("val4________________", "SA")

	div, err := k.TrackGeographicDiversity(ctx)
	require.NoError(t, err)
	require.True(t, div.DiversityScore.GT(math.LegacyNewDecWithPrec(7, 1)))

	// Concentrated scenario
	k, sk, ctx = keepertest.OracleKeeper(t)
	set("a___________________", "NA")
	set("b___________________", "NA")
	set("c___________________", "NA")

	div, err = k.TrackGeographicDiversity(ctx)
	require.NoError(t, err)
	require.True(t, div.DiversityScore.LT(math.LegacyNewDecWithPrec(5, 1)))
}

// CheckCircuitBreakerWithRecovery: trip on deviation and recover after cooldown.
func TestCheckCircuitBreakerWithRecovery(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// seed price record
	require.NoError(t, k.SetPrice(ctx, types.Price{Asset: "ATOM", Price: math.LegacyNewDec(100), BlockHeight: ctx.BlockHeight()}))

	// open breaker manually
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.GetStoreKey())
	store.Set(keeper.CircuitBreakerEnabledKey, []byte{1})
	store.Set(keeper.CircuitBreakerReasonKey, []byte("test"))
	store.Set(keeper.CircuitBreakerActorKey, []byte("tester"))

	tripped, err := k.CheckCircuitBreakerWithRecovery(ctx)
	require.NoError(t, err)
	require.True(t, tripped)

	// advance height beyond cooldown and ensure state clears
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 200)
	enabled, _, _ := k.GetCircuitBreakerState(ctx)
	require.False(t, enabled)
}
