package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/dex/types"
)

const (
	// MinimumLiquidity is the minimum liquidity locked permanently for the first LP
	// This prevents division by zero and manipulation attacks
	MinimumLiquidity = 1000

	// MinSharesForRedemption is the minimum shares required to redeem liquidity
	MinSharesForRedemption = 100
)

var (
	// Maximum liquidity concentration allowed for a single provider (40%)
	MaxLiquidityConcentration = math.LegacyMustNewDecFromStr("0.4")

	// Minimum liquidity required to activate protection
	MinLiquidityForProtection = math.NewInt(1000000)

	maxSingleProviderShare = math.LegacyNewDecWithPrec(95, 2) // 95%
)

// ValidateLPMinting performs comprehensive security checks for LP token minting
func (k Keeper) ValidateLPMinting(ctx context.Context, pool *types.Pool, provider sdk.AccAddress, sharesToMint math.Int) error {
	// 1. Check minimum share amount
	if sharesToMint.LT(math.NewInt(MinSharesForRedemption)) {
		return types.ErrInvalidInput.Wrapf(
			"shares to mint %s is below minimum %d",
			sharesToMint, MinSharesForRedemption,
		)
	}

	// 2. Calculate new total shares
	newTotalShares := pool.TotalShares.Add(sharesToMint)

	// 3. Check for share supply manipulation
	// Ensure shares don't exceed reasonable bounds relative to reserves
	if err := k.validateShareSupply(pool, newTotalShares); err != nil {
		return err
	}

	// 4. Check concentration risk - prevent single provider from dominating pool
	if err := k.validateProviderConcentration(ctx, pool.Id, provider, sharesToMint, newTotalShares); err != nil {
		return err
	}

	// 5. Validate price ratio hasn't been manipulated
	if err := k.validatePriceRatio(pool); err != nil {
		return err
	}

	// 6. Check for suspicious liquidity patterns
	if err := k.detectSuspiciousLiquidity(ctx, pool, provider, sharesToMint); err != nil {
		return err
	}

	return nil
}

// validateShareSupply ensures share supply is proportional to reserves
func (k Keeper) validateShareSupply(pool *types.Pool, totalShares math.Int) error {
	// Calculate geometric mean of reserves: sqrt(reserveA * reserveB)
	k_value := pool.ReserveA.Mul(pool.ReserveB)

	// For very large numbers, use decimal approximation
	kDec := math.LegacyNewDecFromInt(k_value)
	sqrtK, err := kDec.ApproxSqrt()
	if err != nil {
		return types.ErrInvalidInput.Wrapf("failed to compute sqrt(k): %v", err)
	}

	// Total shares should be approximately equal to sqrt(k)
	// Allow up to 10x variance for rounding and initial liquidity
	sharesDec := math.LegacyNewDecFromInt(totalShares)

	maxReasonableShares := sqrtK.Mul(math.LegacyNewDec(10))
	if sharesDec.GT(maxReasonableShares) {
		return types.ErrInvalidInput.Wrapf(
			"share supply %s exceeds reasonable maximum %s for reserves",
			totalShares, maxReasonableShares.TruncateInt(),
		)
	}

	// Minimum check: shares should be at least sqrt(k) / 10
	minReasonableShares := sqrtK.Quo(math.LegacyNewDec(10))
	if sharesDec.LT(minReasonableShares) && !totalShares.LT(math.NewInt(MinimumLiquidity)) {
		return types.ErrInvalidInput.Wrapf(
			"share supply %s below reasonable minimum %s for reserves",
			totalShares, minReasonableShares.TruncateInt(),
		)
	}

	return nil
}

// validateProviderConcentration prevents single provider from dominating pool
func (k Keeper) validateProviderConcentration(ctx context.Context, poolID uint64, provider sdk.AccAddress, newShares, totalShares math.Int) error {
	// Get provider's current shares
	currentShares, err := k.GetLiquidityShares(ctx, poolID, provider)
	if err != nil {
		return err
	}

	// Calculate provider's new total shares
	providerTotalShares := currentShares.Add(newShares)

	// Calculate provider's percentage
	providerPercentage := math.LegacyNewDecFromInt(providerTotalShares).
		Quo(math.LegacyNewDecFromInt(totalShares))

	// Check concentration limit (95%)
	if providerPercentage.GT(maxSingleProviderShare) {
		return types.ErrInvalidInput.Wrapf(
			"provider would own %s%% of pool, exceeds maximum %s%%",
			providerPercentage.Mul(math.LegacyNewDec(100)).TruncateInt(),
			maxSingleProviderShare.Mul(math.LegacyNewDec(100)).TruncateInt(),
		)
	}

	return nil
}

// validatePriceRatio checks for extreme price manipulation
func (k Keeper) validatePriceRatio(pool *types.Pool) error {
	if pool.ReserveA.IsZero() || pool.ReserveB.IsZero() {
		return types.ErrInvalidPoolState.Wrap("pool has zero reserves")
	}

	// Calculate price ratio: reserveA / reserveB
	priceRatio := math.LegacyNewDecFromInt(pool.ReserveA).
		Quo(math.LegacyNewDecFromInt(pool.ReserveB))

	// Check for extreme ratios (more than 1,000,000:1 or less than 1:1,000,000)
	maxRatio := math.LegacyNewDec(1_000_000)
	minRatio := math.LegacyNewDecWithPrec(1, 6) // 0.000001

	if priceRatio.GT(maxRatio) {
		return types.ErrInvalidPoolState.Wrapf(
			"price ratio %s exceeds maximum %s",
			priceRatio, maxRatio,
		)
	}

	if priceRatio.LT(minRatio) {
		return types.ErrInvalidPoolState.Wrapf(
			"price ratio %s below minimum %s",
			priceRatio, minRatio,
		)
	}

	return nil
}

// detectSuspiciousLiquidity checks for suspicious liquidity addition patterns
func (k Keeper) detectSuspiciousLiquidity(ctx context.Context, pool *types.Pool, provider sdk.AccAddress, sharesToMint math.Int) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 1. Check for rapid consecutive liquidity additions (potential flash loan attack)
	lastBlock, found, err := k.GetLastLiquidityActionBlock(ctx, pool.Id, provider)
	if err != nil {
		return err
	}

	currentHeight := sdkCtx.BlockHeight()
	if found && currentHeight-lastBlock < 5 {
		// Require at least 5 blocks between liquidity actions
		return types.ErrFlashLoanDetected.Wrapf(
			"liquidity actions too frequent: last at block %d, current %d",
			lastBlock, currentHeight,
		)
	}

	// 2. Check for disproportionately large single addition
	if !pool.TotalShares.IsZero() {
		additionPercentage := math.LegacyNewDecFromInt(sharesToMint).
			Quo(math.LegacyNewDecFromInt(pool.TotalShares))

		// Flag if single addition is more than 50% of existing pool
		if additionPercentage.GT(math.LegacyNewDecWithPrec(50, 2)) {
			// Emit warning event but allow (could be legitimate large LP)
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeDexLargeLiquidityAddition,
					sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", pool.Id)),
					sdk.NewAttribute(types.AttributeKeyProvider, provider.String()),
					sdk.NewAttribute(types.AttributeKeyPercentage, additionPercentage.Mul(math.LegacyNewDec(100)).String()),
				),
			)
		}
	}

	// 3. Check for initial liquidity lock
	if pool.TotalShares.IsZero() || pool.TotalShares.LT(math.NewInt(MinimumLiquidity)) {
		// First liquidity provider must lock minimum liquidity permanently
		if sharesToMint.LT(math.NewInt(MinimumLiquidity)) {
			return types.ErrInvalidInput.Wrapf(
				"initial liquidity must be at least %d shares",
				MinimumLiquidity,
			)
		}
	}

	return nil
}

// ValidateLPBurning performs security checks for LP token burning
func (k Keeper) ValidateLPBurning(ctx context.Context, pool *types.Pool, provider sdk.AccAddress, sharesToBurn math.Int) error {
	// 1. Check minimum shares
	if sharesToBurn.LT(math.NewInt(MinSharesForRedemption)) {
		return types.ErrInvalidInput.Wrapf(
			"shares to burn %s is below minimum %d",
			sharesToBurn, MinSharesForRedemption,
		)
	}

	// 2. Get provider's shares
	providerShares, err := k.GetLiquidityShares(ctx, pool.Id, provider)
	if err != nil {
		return err
	}

	// 3. Validate sufficient balance
	if sharesToBurn.GT(providerShares) {
		return types.ErrInsufficientShares.Wrapf(
			"insufficient shares: have %s, trying to burn %s",
			providerShares, sharesToBurn,
		)
	}

	// 4. Ensure pool maintains minimum liquidity
	newTotalShares := pool.TotalShares.Sub(sharesToBurn)

	if newTotalShares.LT(math.NewInt(MinimumLiquidity)) && !newTotalShares.IsZero() {
		return types.ErrInvalidInput.Wrapf(
			"withdrawal would leave pool with insufficient liquidity: %s < %d",
			newTotalShares, MinimumLiquidity,
		)
	}

	// 5. Check that withdrawal doesn't drain pool excessively
	burnPercentage := math.LegacyNewDecFromInt(sharesToBurn).
		Quo(math.LegacyNewDecFromInt(pool.TotalShares))

	// Allow up to 90% withdrawal in a single transaction
	if burnPercentage.GT(math.LegacyNewDecWithPrec(90, 2)) {
		return types.ErrInvalidInput.Wrapf(
			"single withdrawal of %s%% exceeds maximum 90%%",
			burnPercentage.Mul(math.LegacyNewDec(100)),
		)
	}

	return nil
}

// LockInitialLiquidity locks the first minimum liquidity permanently
func (k Keeper) LockInitialLiquidity(ctx context.Context, poolID uint64) error {
	// Create a burn address for locked liquidity
	burnAddr := sdk.AccAddress([]byte("locked_liquidity___"))

	// Set minimum locked shares
	lockedShares := math.NewInt(MinimumLiquidity)

	if err := k.SetLiquidityShares(ctx, poolID, burnAddr, lockedShares); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDexLiquidityLocked,
			sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute(types.AttributeKeyLockedShares, lockedShares.String()),
		),
	)

	return nil
}

// GetLastLiquidityActionBlock returns the last block when provider added/removed liquidity
func (k Keeper) GetLastLiquidityActionBlock(ctx context.Context, poolID uint64, provider sdk.AccAddress) (int64, bool, error) {
	store := k.getStore(ctx)
	key := LastLiquidityActionKey(poolID, provider)

	bz := store.Get(key)
	if bz == nil {
		return 0, false, nil
	}

	if len(bz) != 8 {
		return 0, false, types.ErrInvalidState.Wrap("invalid block height data")
	}

	height := int64(sdk.BigEndianToUint64(bz))
	return height, true, nil
}

// SetLastLiquidityActionBlock records the block when provider added/removed liquidity
func (k Keeper) SetLastLiquidityActionBlock(ctx context.Context, poolID uint64, provider sdk.AccAddress) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)
	key := LastLiquidityActionKey(poolID, provider)

	heightBz := sdk.Uint64ToBigEndian(uint64(sdkCtx.BlockHeight()))
	store.Set(key, heightBz)

	return nil
}

// ValidateInitialLiquidity validates the first liquidity addition to a pool
func (k Keeper) ValidateInitialLiquidity(amountA, amountB math.Int) error {
	params, _ := k.GetParams(context.Background())

	// Check minimum liquidity amounts
	if amountA.LT(params.MinLiquidity) {
		return types.ErrInvalidInput.Wrapf(
			"amount A %s below minimum %s",
			amountA, params.MinLiquidity,
		)
	}

	if amountB.LT(params.MinLiquidity) {
		return types.ErrInvalidInput.Wrapf(
			"amount B %s below minimum %s",
			amountB, params.MinLiquidity,
		)
	}

	// Calculate initial shares (sqrt of product)
	product := amountA.Mul(amountB)
	productDec := math.LegacyNewDecFromInt(product)
	sqrtProd, err := productDec.ApproxSqrt()
	if err != nil {
		return types.ErrInvalidInput.Wrapf("failed to compute initial liquidity sqrt: %v", err)
	}
	initialShares := sqrtProd.TruncateInt()

	// Ensure initial shares meet minimum
	if initialShares.LT(math.NewInt(MinimumLiquidity)) {
		return types.ErrInvalidInput.Wrapf(
			"initial liquidity results in %s shares, below minimum %d",
			initialShares, MinimumLiquidity,
		)
	}

	return nil
}
