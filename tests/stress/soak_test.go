//go:build stress
// +build stress

package stress_test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// TestTwentyFourHourSoak provides a manual 24h+ soak harness guarded by STRESS_SOAK=1.
func TestTwentyFourHourSoak(t *testing.T) {
	if os.Getenv("STRESS_SOAK") == "" {
		t.Skip("set STRESS_SOAK=1 to run 24h soak; optional STRESS_SOAK_DURATION/OPS/CONCURRENCY to tune")
	}

	config := MarathonWorkloadConfig()
	config.ReportingInterval = 15 * time.Minute

	if dur := os.Getenv("STRESS_SOAK_DURATION"); dur != "" {
		parsed, err := time.ParseDuration(dur)
		require.NoError(t, err, "invalid STRESS_SOAK_DURATION")
		config.Duration = parsed
	}
	if ops := os.Getenv("STRESS_SOAK_OPS"); ops != "" {
		val, err := strconv.Atoi(ops)
		require.NoError(t, err, "invalid STRESS_SOAK_OPS")
		config.OperationsPerSec = val
	}
	if conc := os.Getenv("STRESS_SOAK_CONCURRENCY"); conc != "" {
		val, err := strconv.Atoi(conc)
		require.NoError(t, err, "invalid STRESS_SOAK_CONCURRENCY")
		config.Concurrency = val
	}

	sta := NewLightweightStressTestApp(t)
	defer sta.Cleanup()

	pools := CreateTestPools(t, sta, 25)
	providers := CreateTestComputeProviders(t, sta, 10)
	feeds := CreateTestOracleFeeds(t, sta, 15)
	trader := fundSoakTrader(t, sta, pools)

	executor := NewWorkloadExecutor(t, config)

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration+5*time.Minute)
	defer cancel()

	operation := func(ctx context.Context) error {
		switch time.Now().UnixNano() % 4 { // rotate workloads
		case 0:
			return performSoakSwap(sta, trader, pools)
		case 1:
			return performSoakComputeMutation(sta, providers)
		case 2:
			return performSoakOracleUpdate(sta, feeds)
		default:
			return performSoakQuery(sta, pools)
		}
	}

	executor.Execute(ctx, operation)
}

func fundSoakTrader(t *testing.T, sta *StressTestApp, poolIDs []uint64) sdk.AccAddress {
	trader := dextypes.TestAddr()
	denoms := map[string]struct{}{}
	for _, id := range poolIDs {
		pool, err := sta.App.DEXKeeper.GetPool(sta.Ctx, id)
		require.NoError(t, err)
		denoms[pool.TokenA] = struct{}{}
		denoms[pool.TokenB] = struct{}{}
	}

	var coins sdk.Coins
	for denom := range denoms {
		coins = coins.Add(sdk.NewCoin(denom, math.NewInt(100_000_000)))
	}

	require.NoError(t, sta.App.BankKeeper.MintCoins(sta.Ctx, dextypes.ModuleName, coins))
	require.NoError(t, sta.App.BankKeeper.SendCoinsFromModuleToAccount(sta.Ctx, dextypes.ModuleName, trader, coins))
	return trader
}

func performSoakSwap(sta *StressTestApp, trader sdk.AccAddress, poolIDs []uint64) error {
	if len(poolIDs) == 0 {
		return nil
	}

	poolID := poolIDs[time.Now().UnixNano()%int64(len(poolIDs))]
	pool, err := sta.App.DEXKeeper.GetPool(sta.Ctx, poolID)
	if err != nil {
		return err
	}

	amountIn := math.NewInt(25_000)
	returnErr := func(tokenIn, tokenOut string) error {
		_, swapErr := sta.App.DEXKeeper.ExecuteSwapSecure(
			sta.Ctx,
			trader,
			poolID,
			tokenIn,
			tokenOut,
			amountIn,
			math.NewInt(1),
		)
		return swapErr
	}

	if err := returnErr(pool.TokenA, pool.TokenB); err != nil {
		return err
	}
	return nil
}

func performSoakComputeMutation(sta *StressTestApp, providers []sdk.AccAddress) error {
	if len(providers) == 0 {
		return nil
	}

	provider := providers[time.Now().UnixNano()%int64(len(providers))]
	req := computetypes.Request{
		Id:             uint64(time.Now().UnixNano()),
		Requester:      dextypes.TestAddr().String(),
		Provider:       provider.String(),
		Status:         computetypes.REQUEST_STATUS_COMPLETED,
		MaxPayment:     math.NewInt(1_000),
		EscrowedAmount: math.NewInt(1_000),
		CompletedAt:    ptrTime(time.Now()),
	}

	return sta.App.ComputeKeeper.SetRequest(sta.Ctx, req)
}

func performSoakOracleUpdate(sta *StressTestApp, feeds []string) error {
	if len(feeds) == 0 {
		return nil
	}

	feed := feeds[time.Now().UnixNano()%int64(len(feeds))]
	price := oracletypes.Price{
		Asset:         feed,
		Price:         math.LegacyNewDec(1_000 + time.Now().UnixNano()%500),
		BlockHeight:   sta.Ctx.BlockHeight(),
		BlockTime:     sta.Ctx.BlockTime().Unix(),
		NumValidators: 3,
	}

	return sta.App.OracleKeeper.SetPrice(sta.Ctx, price)
}

func performSoakQuery(sta *StressTestApp, pools []uint64) error {
	if len(pools) == 0 {
		return nil
	}

	_, err := sta.App.DEXKeeper.GetPool(sta.Ctx, pools[time.Now().UnixNano()%int64(len(pools))])
	return err
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
