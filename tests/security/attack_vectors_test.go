package security_test

import (
	"fmt"
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// AttackVectorsTestSuite focuses on regression tests for the highest-risk security flaws.
type AttackVectorsTestSuite struct {
	suite.Suite
	app *app.PAWApp
	ctx sdk.Context
}

func (suite *AttackVectorsTestSuite) SetupTest() {
	suite.app, suite.ctx = keepertest.SetupTestApp(suite.T())
}

func TestAttackVectorsTestSuite(t *testing.T) {
	suite.Run(t, new(AttackVectorsTestSuite))
}

func (suite *AttackVectorsTestSuite) fundAccount(coins sdk.Coins) sdk.AccAddress {
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, coins))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, addr, coins))

	return addr
}

func (suite *AttackVectorsTestSuite) createBalancedPool(provider sdk.AccAddress) uint64 {
	initial := sdkmath.NewInt(5_000_000)
	pool, err := suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		provider,
		"upaw",
		"uusdc",
		initial,
		initial,
	)
	suite.Require().NoError(err)
	return pool.Id
}

// TestReentrancyProtection ensures the DEX reentrancy guard surfaces ErrReentrancy when nested swaps are attempted.
func (suite *AttackVectorsTestSuite) TestReentrancyProtection() {
	lp := suite.fundAccount(sdk.NewCoins(
		sdk.NewCoin("upaw", sdkmath.NewInt(10_000_000)),
		sdk.NewCoin("uusdc", sdkmath.NewInt(10_000_000)),
	))
	poolID := suite.createBalancedPool(lp)

	trader := suite.fundAccount(sdk.NewCoins(
		sdk.NewCoin("upaw", sdkmath.NewInt(1_000_000)),
	))

	guard := dexkeeper.NewReentrancyGuard()
	suite.ctx = suite.ctx.WithValue("reentrancy_guard", guard)
	lockKey := fmt.Sprintf("%d:swap", poolID)

	suite.Require().NoError(guard.Lock(lockKey))
	_, err := suite.app.DEXKeeper.ExecuteSwapSecure(
		suite.ctx,
		trader,
		poolID,
		"upaw",
		"uusdc",
		sdkmath.NewInt(1_000),
		sdkmath.NewInt(1),
	)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, dextypes.ErrReentrancy)

	guard.Unlock(lockKey)

	// After the guard is released the swap should succeed again.
	_, err = suite.app.DEXKeeper.ExecuteSwapSecure(
		suite.ctx,
		trader,
		poolID,
		"upaw",
		"uusdc",
		sdkmath.NewInt(500),
		sdkmath.NewInt(1),
	)
	suite.Require().NoError(err)
}

// TestIntegerOverflow validates that SafeMath helpers and oracle TWAP overflow guards reject unsafe values.
func (suite *AttackVectorsTestSuite) TestIntegerOverflow() {
	limit := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	nearLimit := new(big.Int).Sub(limit, big.NewInt(1))
	huge := sdkmath.NewIntFromBigInt(nearLimit)

	_, err := dexkeeper.SafeAdd(huge, sdkmath.OneInt())
	suite.Require().Error(err, "SafeAdd must reject overflow beyond 2^256")

	_, err = computekeeper.SafeMul(huge, sdkmath.NewInt(2))
	suite.Require().Error(err, "SafeMul must reject overflow")

	params, err := suite.app.OracleKeeper.GetParams(suite.ctx)
	suite.Require().NoError(err)
	params.TwapLookbackWindow = 128
	suite.Require().NoError(suite.app.OracleKeeper.SetParams(suite.ctx, params))

	asset := "PAW/USD"
	baseTime := suite.ctx.BlockTime().Unix()
	largeDelta := int64(2_000_000_000_000_000_000) // >1e18 to trip overflow guard

	snapshots := []oracletypes.PriceSnapshot{
		{
			Asset:       asset,
			Price:       sdkmath.LegacyNewDec(1),
			BlockHeight: suite.ctx.BlockHeight(),
			BlockTime:   baseTime,
		},
		{
			Asset:       asset,
			Price:       sdkmath.LegacyNewDec(1),
			BlockHeight: suite.ctx.BlockHeight() + 1,
			BlockTime:   baseTime + largeDelta,
		},
	}

	for _, snap := range snapshots {
		suite.Require().NoError(suite.app.OracleKeeper.SetPriceSnapshot(suite.ctx, snap))
	}

	_, err = suite.app.OracleKeeper.CalculateVolumeWeightedTWAP(suite.ctx, asset)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "time delta too large")
}

// TestMEVProtection proves swaps with unrealistic slippage expectations are rejected.
func (suite *AttackVectorsTestSuite) TestMEVProtection() {
	lp := suite.fundAccount(sdk.NewCoins(
		sdk.NewCoin("upaw", sdkmath.NewInt(10_000_000)),
		sdk.NewCoin("uusdc", sdkmath.NewInt(10_000_000)),
	))
	poolID := suite.createBalancedPool(lp)

	trader := suite.fundAccount(sdk.NewCoins(
		sdk.NewCoin("upaw", sdkmath.NewInt(2_000_000)),
	))

	_, err := suite.app.DEXKeeper.ExecuteSwapSecure(
		suite.ctx,
		trader,
		poolID,
		"upaw",
		"uusdc",
		sdkmath.NewInt(10_000),
		sdkmath.NewInt(5_000_000),
	)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, dextypes.ErrSlippageTooHigh)
}

// TestDuplicateSubmission ensures the nonce store prevents replaying identical IBC packets.
func (suite *AttackVectorsTestSuite) TestDuplicateSubmission() {
	channelID := "channel-0"
	sender := "cosmos1attackernonce000000000000000000000000"
	timestamp := suite.ctx.BlockTime().Unix()

	err := suite.app.DEXKeeper.ValidateIncomingPacketNonce(suite.ctx, channelID, sender, 1, timestamp)
	suite.Require().NoError(err)

	err = suite.app.DEXKeeper.ValidateIncomingPacketNonce(suite.ctx, channelID, sender, 1, timestamp)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, dextypes.ErrInvalidNonce)

	// New nonce should succeed
	err = suite.app.DEXKeeper.ValidateIncomingPacketNonce(suite.ctx, channelID, sender, 2, timestamp+1)
	suite.Require().NoError(err)
}
