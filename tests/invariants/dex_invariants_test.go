package invariants

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/app"
	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	dextypes "github.com/paw-chain/paw/x/dex/types"
)

// DEXInvariantTestSuite tests DEX module invariants for liquidity pools
// These are critical to ensure constant product formula and LP share conservation
type DEXInvariantTestSuite struct {
	suite.Suite
	app       *app.PAWApp
	ctx       sdk.Context
	msgServer dextypes.MsgServer
}

// SetupTest initializes the test environment before each test
func (suite *DEXInvariantTestSuite) SetupTest() {
	suite.app = app.Setup(suite.T(), false)
	suite.ctx = suite.app.BaseApp.NewContext(false)
	suite.msgServer = dexkeeper.NewMsgServerImpl(*suite.app.DEXKeeper)
}

// TestConstantProductInvariant verifies the constant product formula k = reserveA * reserveB
// This is the most critical DEX invariant - K should never decrease (only increase with fees)
func (suite *DEXInvariantTestSuite) TestConstantProductInvariant() {
	creator := sdk.AccAddress([]byte("pool_creator_______"))

	// Fund creator account
	initialFunds := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(10000000)),
		sdk.NewCoin("uatom", math.NewInt(10000000)),
	)
	suite.fundAccount(creator, initialFunds)

	// Create pool with initial liquidity
	createMsg := &dextypes.MsgCreatePool{
		Creator: creator.String(),
		TokenA:  "upaw",
		TokenB:  "uatom",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(2000000),
	}

	_, err := suite.msgServer.CreatePool(suite.ctx, createMsg)
	suite.NoError(err)

	// Get the created pool
	pools := suite.app.DEXKeeper.GetAllPools(suite.ctx)
	suite.Require().Len(pools, 1)
	pool := pools[0]

	// Calculate initial K
	initialK := pool.ReserveA.Mul(pool.ReserveB)
	suite.Require().True(initialK.GT(math.ZeroInt()), "Initial K must be positive")

	// Perform swap - this should increase K due to fees
	swapper := sdk.AccAddress([]byte("swapper_____________"))
	swapFunds := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(100000)))
	suite.fundAccount(swapper, swapFunds)

	swapMsg := &dextypes.MsgSwap{
		Trader:     swapper.String(),
		PoolId:     pool.Id,
		TokenIn:    "upaw",
		AmountIn:   math.NewInt(10000),
		MinAmountOut: math.NewInt(1), // Accept any output for test
	}

	_, err = suite.msgServer.Swap(suite.ctx, swapMsg)
	suite.NoError(err)

	// Get updated pool
	poolAfterSwap, found := suite.app.DEXKeeper.GetPool(suite.ctx, pool.Id)
	suite.Require().True(found)

	// Calculate K after swap
	kAfterSwap := poolAfterSwap.ReserveA.Mul(poolAfterSwap.ReserveB)

	// Critical invariant: K should never decrease
	// After swap with fees, K should be >= initial K
	suite.Require().True(
		kAfterSwap.GTE(initialK),
		"Constant product invariant violated: K decreased from %s to %s",
		initialK.String(),
		kAfterSwap.String(),
	)
}

// TestPoolReservesMatchModuleBalance verifies pool reserves equal module account balance
// If this fails, tokens are lost or improperly accounted for
func (suite *DEXInvariantTestSuite) TestPoolReservesMatchModuleBalance() {
	creator := sdk.AccAddress([]byte("creator_____________"))

	// Fund creator
	initialFunds := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(5000000)),
		sdk.NewCoin("uatom", math.NewInt(5000000)),
	)
	suite.fundAccount(creator, initialFunds)

	// Create pool
	createMsg := &dextypes.MsgCreatePool{
		Creator: creator.String(),
		TokenA:  "upaw",
		TokenB:  "uatom",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(1000000),
	}

	_, err := suite.msgServer.CreatePool(suite.ctx, createMsg)
	suite.NoError(err)

	// Get all pools and sum reserves by token
	pools := suite.app.DEXKeeper.GetAllPools(suite.ctx)
	reserveTotals := make(map[string]math.Int)

	for _, pool := range pools {
		// Add TokenA reserves
		if current, exists := reserveTotals[pool.TokenA]; exists {
			reserveTotals[pool.TokenA] = current.Add(pool.ReserveA)
		} else {
			reserveTotals[pool.TokenA] = pool.ReserveA
		}

		// Add TokenB reserves
		if current, exists := reserveTotals[pool.TokenB]; exists {
			reserveTotals[pool.TokenB] = current.Add(pool.ReserveB)
		} else {
			reserveTotals[pool.TokenB] = pool.ReserveB
		}
	}

	// Get module account balance
	moduleAddr := suite.app.AccountKeeper.GetModuleAddress(dextypes.ModuleName)
	moduleBalances := suite.app.BankKeeper.GetAllBalances(suite.ctx, moduleAddr)

	// Verify each denom's reserves match module balance
	for _, balance := range moduleBalances {
		reserveTotal, exists := reserveTotals[balance.Denom]
		if exists {
			suite.Require().True(
				balance.Amount.Equal(reserveTotal),
				"Pool reserves don't match module balance for %s: module=%s, reserves=%s",
				balance.Denom,
				balance.Amount.String(),
				reserveTotal.String(),
			)
		}
	}
}

// TestLPShareConservation verifies sum of all LP shares equals pool's total shares
// This ensures no shares are created or destroyed improperly
func (suite *DEXInvariantTestSuite) TestLPShareConservation() {
	creator := sdk.AccAddress([]byte("creator_____________"))
	lp1 := sdk.AccAddress([]byte("lp_provider_1_______"))
	lp2 := sdk.AccAddress([]byte("lp_provider_2_______"))

	// Fund accounts
	fundAmount := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(10000000)),
		sdk.NewCoin("uatom", math.NewInt(10000000)),
	)
	suite.fundAccount(creator, fundAmount)
	suite.fundAccount(lp1, fundAmount)
	suite.fundAccount(lp2, fundAmount)

	// Create pool
	createMsg := &dextypes.MsgCreatePool{
		Creator: creator.String(),
		TokenA:  "upaw",
		TokenB:  "uatom",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(1000000),
	}

	_, err := suite.msgServer.CreatePool(suite.ctx, createMsg)
	suite.NoError(err)

	pools := suite.app.DEXKeeper.GetAllPools(suite.ctx)
	suite.Require().Len(pools, 1)
	poolId := pools[0].Id

	// Add liquidity from multiple providers
	addLiqMsg1 := &dextypes.MsgAddLiquidity{
		Provider: lp1.String(),
		PoolId:   poolId,
		AmountA:  math.NewInt(500000),
		AmountB:  math.NewInt(500000),
	}
	_, err = suite.msgServer.AddLiquidity(suite.ctx, addLiqMsg1)
	suite.NoError(err)

	addLiqMsg2 := &dextypes.MsgAddLiquidity{
		Provider: lp2.String(),
		PoolId:   poolId,
		AmountA:  math.NewInt(300000),
		AmountB:  math.NewInt(300000),
	}
	_, err = suite.msgServer.AddLiquidity(suite.ctx, addLiqMsg2)
	suite.NoError(err)

	// Get updated pool
	pool, found := suite.app.DEXKeeper.GetPool(suite.ctx, poolId)
	suite.Require().True(found)

	// Get shares for each provider
	creatorShares := suite.app.DEXKeeper.GetLPShares(suite.ctx, poolId, creator)
	lp1Shares := suite.app.DEXKeeper.GetLPShares(suite.ctx, poolId, lp1)
	lp2Shares := suite.app.DEXKeeper.GetLPShares(suite.ctx, poolId, lp2)

	// Sum all shares
	totalShares := creatorShares.Add(lp1Shares).Add(lp2Shares)

	// Verify sum equals pool's total shares
	suite.Require().True(
		totalShares.Equal(pool.TotalShares),
		"LP share conservation violated: sum=%s, pool_total=%s",
		totalShares.String(),
		pool.TotalShares.String(),
	)
}

// TestNoNegativeReserves ensures pool reserves are always non-negative
func (suite *DEXInvariantTestSuite) TestNoNegativeReserves() {
	creator := sdk.AccAddress([]byte("creator_____________"))

	// Fund creator
	initialFunds := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(5000000)),
		sdk.NewCoin("uatom", math.NewInt(5000000)),
	)
	suite.fundAccount(creator, initialFunds)

	// Create pool
	createMsg := &dextypes.MsgCreatePool{
		Creator: creator.String(),
		TokenA:  "upaw",
		TokenB:  "uatom",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(1000000),
	}

	_, err := suite.msgServer.CreatePool(suite.ctx, createMsg)
	suite.NoError(err)

	// Check all pools have non-negative reserves
	pools := suite.app.DEXKeeper.GetAllPools(suite.ctx)
	for _, pool := range pools {
		suite.Require().False(
			pool.ReserveA.IsNegative(),
			"Pool %d has negative ReserveA: %s",
			pool.Id,
			pool.ReserveA.String(),
		)
		suite.Require().False(
			pool.ReserveB.IsNegative(),
			"Pool %d has negative ReserveB: %s",
			pool.Id,
			pool.ReserveB.String(),
		)
		suite.Require().False(
			pool.TotalShares.IsNegative(),
			"Pool %d has negative TotalShares: %s",
			pool.Id,
			pool.TotalShares.String(),
		)
	}
}

// TestPoolReserveRatiosWithinBounds verifies reserve ratios stay reasonable
// Extreme ratios could indicate bugs or manipulation
func (suite *DEXInvariantTestSuite) TestPoolReserveRatiosWithinBounds() {
	creator := sdk.AccAddress([]byte("creator_____________"))

	// Fund creator
	initialFunds := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(10000000)),
		sdk.NewCoin("uatom", math.NewInt(10000000)),
	)
	suite.fundAccount(creator, initialFunds)

	// Create pool with reasonable ratio (1:2)
	createMsg := &dextypes.MsgCreatePool{
		Creator: creator.String(),
		TokenA:  "upaw",
		TokenB:  "uatom",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(2000000),
	}

	_, err := suite.msgServer.CreatePool(suite.ctx, createMsg)
	suite.NoError(err)

	// Check all pools
	pools := suite.app.DEXKeeper.GetAllPools(suite.ctx)
	for _, pool := range pools {
		// Reserves must be positive
		suite.Require().True(
			pool.ReserveA.GT(math.ZeroInt()),
			"Pool %d has zero or negative ReserveA",
			pool.Id,
		)
		suite.Require().True(
			pool.ReserveB.GT(math.ZeroInt()),
			"Pool %d has zero or negative ReserveB",
			pool.Id,
		)

		// Calculate ratio (larger / smaller should be < 1,000,000 for sanity)
		// This prevents extreme price ratios that could indicate issues
		var ratio math.LegacyDec
		if pool.ReserveA.GT(pool.ReserveB) {
			ratio = math.LegacyNewDecFromInt(pool.ReserveA).Quo(math.LegacyNewDecFromInt(pool.ReserveB))
		} else {
			ratio = math.LegacyNewDecFromInt(pool.ReserveB).Quo(math.LegacyNewDecFromInt(pool.ReserveA))
		}

		maxRatio := math.LegacyNewDec(1000000) // 1:1,000,000 max ratio
		suite.Require().True(
			ratio.LT(maxRatio),
			"Pool %d has extreme reserve ratio: %s",
			pool.Id,
			ratio.String(),
		)
	}
}

// TestSwapInvariantPreservation verifies swaps preserve invariants
func (suite *DEXInvariantTestSuite) TestSwapInvariantPreservation() {
	creator := sdk.AccAddress([]byte("creator_____________"))
	trader := sdk.AccAddress([]byte("trader______________"))

	// Fund accounts
	creatorFunds := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(5000000)),
		sdk.NewCoin("uatom", math.NewInt(5000000)),
	)
	traderFunds := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(1000000)),
	)
	suite.fundAccount(creator, creatorFunds)
	suite.fundAccount(trader, traderFunds)

	// Create pool
	createMsg := &dextypes.MsgCreatePool{
		Creator: creator.String(),
		TokenA:  "upaw",
		TokenB:  "uatom",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(1000000),
	}

	_, err := suite.msgServer.CreatePool(suite.ctx, createMsg)
	suite.NoError(err)

	pools := suite.app.DEXKeeper.GetAllPools(suite.ctx)
	poolId := pools[0].Id

	// Record state before swap
	poolBefore, _ := suite.app.DEXKeeper.GetPool(suite.ctx, poolId)
	kBefore := poolBefore.ReserveA.Mul(poolBefore.ReserveB)
	moduleAddr := suite.app.AccountKeeper.GetModuleAddress(dextypes.ModuleName)
	moduleBalanceBefore := suite.app.BankKeeper.GetAllBalances(suite.ctx, moduleAddr)

	// Perform swap
	swapMsg := &dextypes.MsgSwap{
		Trader:       trader.String(),
		PoolId:       poolId,
		TokenIn:      "upaw",
		AmountIn:     math.NewInt(10000),
		MinAmountOut: math.NewInt(1),
	}

	_, err = suite.msgServer.Swap(suite.ctx, swapMsg)
	suite.NoError(err)

	// Check state after swap
	poolAfter, _ := suite.app.DEXKeeper.GetPool(suite.ctx, poolId)
	kAfter := poolAfter.ReserveA.Mul(poolAfter.ReserveB)
	moduleBalanceAfter := suite.app.BankKeeper.GetAllBalances(suite.ctx, moduleAddr)

	// K should not decrease (fees make it increase)
	suite.Require().True(
		kAfter.GTE(kBefore),
		"K decreased after swap: before=%s, after=%s",
		kBefore.String(),
		kAfter.String(),
	)

	// Module balance should not decrease (it should increase with fees)
	for _, coinBefore := range moduleBalanceBefore {
		coinAfter := moduleBalanceAfter.AmountOf(coinBefore.Denom)
		suite.Require().True(
			coinAfter.GTE(coinBefore.Amount),
			"Module balance decreased for %s: before=%s, after=%s",
			coinBefore.Denom,
			coinBefore.Amount.String(),
			coinAfter.String(),
		)
	}

	// Total shares should not change on swap
	suite.Require().True(
		poolAfter.TotalShares.Equal(poolBefore.TotalShares),
		"Total shares changed after swap: before=%s, after=%s",
		poolBefore.TotalShares.String(),
		poolAfter.TotalShares.String(),
	)
}

// TestMinLiquidityInvariant ensures pools maintain minimum liquidity
func (suite *DEXInvariantTestSuite) TestMinLiquidityInvariant() {
	creator := sdk.AccAddress([]byte("creator_____________"))

	// Fund creator
	initialFunds := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(5000000)),
		sdk.NewCoin("uatom", math.NewInt(5000000)),
	)
	suite.fundAccount(creator, initialFunds)

	// Get minimum liquidity requirement
	params := suite.app.DEXKeeper.GetParams(suite.ctx)

	// Create pool with sufficient liquidity
	createMsg := &dextypes.MsgCreatePool{
		Creator: creator.String(),
		TokenA:  "upaw",
		TokenB:  "uatom",
		AmountA: params.MinLiquidity.Add(math.NewInt(100000)),
		AmountB: params.MinLiquidity.Add(math.NewInt(100000)),
	}

	_, err := suite.msgServer.CreatePool(suite.ctx, createMsg)
	suite.NoError(err)

	// Check all pools meet minimum liquidity
	pools := suite.app.DEXKeeper.GetAllPools(suite.ctx)
	for _, pool := range pools {
		suite.Require().True(
			pool.ReserveA.GTE(params.MinLiquidity),
			"Pool %d ReserveA below minimum: %s < %s",
			pool.Id,
			pool.ReserveA.String(),
			params.MinLiquidity.String(),
		)
		suite.Require().True(
			pool.ReserveB.GTE(params.MinLiquidity),
			"Pool %d ReserveB below minimum: %s < %s",
			pool.Id,
			pool.ReserveB.String(),
			params.MinLiquidity.String(),
		)
	}
}

// Helper function to fund an account
func (suite *DEXInvariantTestSuite) fundAccount(addr sdk.AccAddress, coins sdk.Coins) {
	err := suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, coins)
	suite.NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, addr, coins)
	suite.NoError(err)
}

func TestDEXInvariantTestSuite(t *testing.T) {
	suite.Run(t, new(DEXInvariantTestSuite))
}
