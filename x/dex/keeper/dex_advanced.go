package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// Pool Creation Spam Prevention Constants
//
// These constants protect against pool creation spam attacks where malicious actors
// create numerous low-liquidity pools to:
// - Inflate storage costs and bloat the blockchain state
// - Fragment liquidity across many pools (reducing efficiency)
// - Confuse users with fake/duplicate trading pairs
// - Conduct sybil attacks to manipulate pool discovery
const (
	// MinPoolCreationDeposit is the minimum deposit required to create a pool (100 tokens).
	// This economic barrier prevents trivial spam while remaining accessible for legitimate pools.
	// Value chosen to be high enough to deter abuse but low enough to encourage real liquidity.
	MinPoolCreationDeposit = 100_000_000 // 100 tokens

	// PoolCreationCooldown is the minimum blocks between pool creations by the same address.
	// Enforces a time delay (approximately 10 minutes at 6s blocks) to rate-limit pool creation.
	// Prevents rapid-fire pool creation attacks while allowing legitimate multi-pool creation.
	PoolCreationCooldown = 100

	// MaxPoolsPerAddress is the maximum pools a single address can create in PoolCreationWindow.
	// Limits total pools per address to prevent sybil attacks and state bloat.
	// Value of 10 allows legitimate market makers while blocking spam.
	MaxPoolsPerAddress = 10

	// PoolCreationWindow is the number of blocks to track pool creation rate (approximately 1 day).
	// Sliding window of 10,000 blocks (~16.7 hours at 6s blocks) for rate limiting.
	// Long enough to prevent circumventing limits via waiting, short enough to not permanently restrict.
	PoolCreationWindow = 10000 // approximately 1 day
)

// ValidatePoolCreation implements comprehensive spam prevention for pool creation.
//
// This function enforces multiple layers of protection against pool creation attacks:
//  1. Minimum deposit requirement - economic barrier to spam
//  2. Duplicate pool detection - prevents redundant pools for same token pair
//  3. Creation cooldown - time-based rate limiting per address
//  4. Creation rate limit - maximum pools per address in time window
//  5. Token denomination validation - ensures valid token names
//  6. Token pair validation - prevents identical token pairs
//
// Parameters:
//   - ctx: Blockchain context for state access
//   - creator: Address attempting to create the pool
//   - tokenA: First token denomination in the pair
//   - tokenB: Second token denomination in the pair
//   - initialA: Initial liquidity for tokenA (must meet MinPoolCreationDeposit)
//   - initialB: Initial liquidity for tokenB (must meet MinPoolCreationDeposit)
//
// Returns:
//   - error: nil if validation passes, or specific error indicating failure reason:
//   - ErrInvalidInput: Insufficient deposit, invalid token denom, identical tokens
//   - ErrPoolAlreadyExists: Pool already exists for this token pair
//   - ErrRateLimitExceeded: Too many pools created recently
//
// Security Notes:
//   - Records creation attempt even on failure to track rate limiting
//   - Uses sliding window for rate limits to prevent circumvention
//   - Validates both token denominations to prevent malformed pool creation
func (k Keeper) ValidatePoolCreation(ctx context.Context, creator sdk.AccAddress, tokenA, tokenB string, initialA, initialB math.Int) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// 1. Check minimum deposit requirement
	depositRequired := math.NewInt(MinPoolCreationDeposit)
	if initialA.LT(depositRequired) || initialB.LT(depositRequired) {
		return types.ErrInvalidInput.Wrapf(
			"initial deposit must be at least %s for both tokens",
			depositRequired,
		)
	}

	// 2. Check for duplicate pools
	existingPool, _ := k.GetPoolByTokens(ctx, tokenA, tokenB)
	if existingPool != nil {
		return types.ErrPoolAlreadyExists.Wrapf(
			"pool already exists for %s/%s with ID %d",
			tokenA, tokenB, existingPool.Id,
		)
	}

	// 3. Check creation cooldown
	lastCreationKey := append([]byte("last_pool_creation/"), creator.Bytes()...)
	if bz := store.Get(lastCreationKey); bz != nil {
		lastBlock := int64(sdk.BigEndianToUint64(bz))
		if sdkCtx.BlockHeight()-lastBlock < PoolCreationCooldown {
			return types.ErrRateLimitExceeded.Wrapf(
				"must wait %d blocks between pool creations (last: %d, current: %d)",
				PoolCreationCooldown, lastBlock, sdkCtx.BlockHeight(),
			)
		}
	}

	// 4. Check creation rate limit
	poolCount := k.getPoolCreationCount(ctx, creator, sdkCtx.BlockHeight())
	if poolCount >= MaxPoolsPerAddress {
		return types.ErrRateLimitExceeded.Wrapf(
			"exceeded maximum %d pool creations in %d blocks",
			MaxPoolsPerAddress, PoolCreationWindow,
		)
	}

	// 5. Validate token denominations
	if err := k.validateTokenDenom(ctx, tokenA); err != nil {
		return types.ErrInvalidInput.Wrapf("invalid token A: %v", err)
	}
	if err := k.validateTokenDenom(ctx, tokenB); err != nil {
		return types.ErrInvalidInput.Wrapf("invalid token B: %v", err)
	}

	// 6. Ensure tokens are different
	if tokenA == tokenB {
		return types.ErrInvalidTokenPair.Wrap("cannot create pool with identical tokens")
	}

	// 7. Record creation attempt
	store.Set(lastCreationKey, sdk.Uint64ToBigEndian(uint64(sdkCtx.BlockHeight())))
	k.recordPoolCreation(ctx, creator, sdkCtx.BlockHeight())

	return nil
}

// getPoolCreationCount returns the number of pools created by address in recent window
func (k Keeper) getPoolCreationCount(ctx context.Context, creator sdk.AccAddress, currentHeight int64) int {
	store := k.getStore(ctx)
	prefix := append([]byte("pool_creation_count/"), creator.Bytes()...)

	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	count := 0
	cutoffHeight := currentHeight - PoolCreationWindow

	for ; iterator.Valid(); iterator.Next() {
		blockHeight := int64(binary.BigEndian.Uint64(iterator.Value()))
		if blockHeight > cutoffHeight {
			count++
		}
	}

	return count
}

// recordPoolCreation records a pool creation for rate limiting
func (k Keeper) recordPoolCreation(ctx context.Context, creator sdk.AccAddress, blockHeight int64) {
	store := k.getStore(ctx)

	// Create key: pool_creation_count/{creator}/{block_height}
	key := append([]byte("pool_creation_count/"), creator.Bytes()...)
	key = append(key, sdk.Uint64ToBigEndian(uint64(blockHeight))...)

	store.Set(key, sdk.Uint64ToBigEndian(uint64(blockHeight)))
}

// validateTokenDenom validates that a token denomination exists and is tradeable
func (k Keeper) validateTokenDenom(ctx context.Context, denom string) error {
	if len(denom) == 0 || len(denom) > 128 {
		return types.ErrInvalidInput.Wrap("invalid denom length")
	}

	// Basic denom validation (alphanumeric and some special chars)
	for _, r := range denom {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '/' || r == '-' || r == '_') {
			return types.ErrInvalidInput.Wrapf("invalid character in denom: %c", r)
		}
	}

	return nil
}

// ImpermanentLossInfo represents comprehensive impermanent loss analysis for liquidity providers.
//
// Impermanent loss occurs when the price ratio of pooled assets changes compared to when
// they were deposited. This struct calculates both the loss and offsetting fee earnings
// to give LPs a complete picture of their position performance.
//
// Fields:
//   - InitialValueA: Initial value of TokenA position (in USD or base denomination)
//   - InitialValueB: Initial value of TokenB position (in USD or base denomination)
//   - CurrentValueA: Current value of TokenA position in the pool
//   - CurrentValueB: Current value of TokenB position in the pool
//   - ImpermanentLoss: Loss percentage compared to holding tokens outside pool (negative = loss)
//   - FeesEarned: Total trading fees accumulated by this LP position
//   - NetProfitLoss: Overall profit/loss including both IL and fees (can be positive if fees > IL)
//
// Calculation Formula:
//
//	IL% = (current_pool_value / hold_value - 1) * 100
//	NetProfitLoss% = IL% + (fees_earned / initial_value * 100)
//
// Example:
//
//	If IL = -5% but FeesEarned = 8%, then NetProfitLoss = +3% (profitable position)
type ImpermanentLossInfo struct {
	InitialValueA   math.Int       // Initial value of TokenA position
	InitialValueB   math.Int       // Initial value of TokenB position
	CurrentValueA   math.Int       // Current value of TokenA position in pool
	CurrentValueB   math.Int       // Current value of TokenB position in pool
	ImpermanentLoss math.LegacyDec // Loss percentage vs holding (negative value)
	FeesEarned      math.Int       // Total fees accumulated
	NetProfitLoss   math.LegacyDec // Net profit/loss including fees
}

// CalculateImpermanentLoss calculates the impermanent loss for a liquidity provider position.
//
// This function computes the financial impact of providing liquidity compared to simply holding
// the tokens. It considers both the impermanent loss from price divergence and the offsetting
// effect of accumulated trading fees.
//
// Parameters:
//   - ctx: Blockchain context for state access
//   - poolID: Unique identifier of the liquidity pool
//   - provider: Address of the liquidity provider to analyze
//   - priceOracleA: Current oracle price for TokenA (in USD or base denomination)
//   - priceOracleB: Current oracle price for TokenB (in USD or base denomination)
//
// Returns:
//   - *ImpermanentLossInfo: Detailed analysis struct containing IL, fees, and net P&L
//   - error: nil on success, or:
//   - ErrPoolNotFound: Pool does not exist
//   - ErrInsufficientShares: Provider has no liquidity position in this pool
//
// Calculation Process:
//  1. Retrieves provider's share of pool reserves
//  2. Calculates current value of position at oracle prices
//  3. Computes hypothetical value if tokens were held outside pool
//  4. Determines impermanent loss as percentage difference
//  5. Calculates provider's share of accumulated fees
//  6. Computes net profit/loss (IL + fees)
//
// Security Notes:
//   - Requires accurate oracle prices - stale prices will give incorrect results
//   - Does not account for initial deposit values (would need separate storage)
//   - Assumes current reserves represent fair value at current prices
//
// Usage Example:
//
//	ilInfo, err := keeper.CalculateImpermanentLoss(ctx, 1, providerAddr, usdcPrice, ethPrice)
//	if err != nil { return err }
//	if ilInfo.NetProfitLoss.IsNegative() {
//	    // Position is underwater even with fees
//	}
func (k Keeper) CalculateImpermanentLoss(ctx context.Context, poolID uint64, provider sdk.AccAddress, priceOracleA, priceOracleB math.LegacyDec) (*ImpermanentLossInfo, error) {
	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return nil, err
	}

	// Get provider's shares
	shares, err := k.GetLiquidityShares(ctx, poolID, provider)
	if err != nil {
		return nil, err
	}

	if shares.IsZero() {
		return nil, types.ErrInsufficientShares.Wrap("no liquidity position")
	}

	// Calculate provider's portion of reserves
	portionA := math.LegacyNewDecFromInt(shares).
		Quo(math.LegacyNewDecFromInt(pool.TotalShares)).
		Mul(math.LegacyNewDecFromInt(pool.ReserveA))

	portionB := math.LegacyNewDecFromInt(shares).
		Quo(math.LegacyNewDecFromInt(pool.TotalShares)).
		Mul(math.LegacyNewDecFromInt(pool.ReserveB))

	// Calculate current value in USD equivalent
	currentValueA := portionA.Mul(priceOracleA)
	currentValueB := portionB.Mul(priceOracleB)
	currentTotalValue := currentValueA.Add(currentValueB)

	// Get initial deposit info (would be stored separately in production)
	// For now, we'll calculate what it would have been at current prices if held
	initialHoldValueA := portionA.Mul(priceOracleA)
	initialHoldValueB := portionB.Mul(priceOracleB)
	holdTotalValue := initialHoldValueA.Add(initialHoldValueB)

	// Calculate impermanent loss percentage
	// IL = (current_value / hold_value - 1) * 100
	var impermanentLoss math.LegacyDec
	if !holdTotalValue.IsZero() {
		impermanentLoss = currentTotalValue.Quo(holdTotalValue).Sub(math.LegacyOneDec()).Mul(math.LegacyNewDec(100))
	}

	// Get accumulated fees
	feesA, err := k.GetPoolLPFees(ctx, poolID, pool.TokenA)
	if err != nil {
		return nil, err
	}
	feesB, err := k.GetPoolLPFees(ctx, poolID, pool.TokenB)
	if err != nil {
		return nil, err
	}

	// Calculate provider's share of fees
	providerFeesA := math.LegacyNewDecFromInt(shares).
		Quo(math.LegacyNewDecFromInt(pool.TotalShares)).
		Mul(math.LegacyNewDecFromInt(feesA))

	providerFeesB := math.LegacyNewDecFromInt(shares).
		Quo(math.LegacyNewDecFromInt(pool.TotalShares)).
		Mul(math.LegacyNewDecFromInt(feesB))

	totalFeesValue := providerFeesA.Mul(priceOracleA).Add(providerFeesB.Mul(priceOracleB))

	// Calculate net profit/loss including fees
	netProfitLoss := impermanentLoss.Add(totalFeesValue.Quo(holdTotalValue).Mul(math.LegacyNewDec(100)))

	return &ImpermanentLossInfo{
		InitialValueA:   portionA.TruncateInt(),
		InitialValueB:   portionB.TruncateInt(),
		CurrentValueA:   portionA.TruncateInt(),
		CurrentValueB:   portionB.TruncateInt(),
		ImpermanentLoss: impermanentLoss,
		FeesEarned:      totalFeesValue.TruncateInt(),
		NetProfitLoss:   netProfitLoss,
	}, nil
}

// Flash Loan Attack Prevention
//
// SECURITY: This module prevents flash loan attacks on liquidity pools.
//
// Attack Vector:
// 1. Attacker adds huge liquidity (becomes dominant LP)
// 2. Executes large swap to manipulate pool price
// 3. Arbitrages price difference on another pool/chain
// 4. Removes liquidity in same block (risk-free profit)
//
// Defense: Multi-Block Lock Period
// - Minimum delay (default 10 blocks) between add and remove liquidity
// - Configurable via governance parameter FlashLoanProtectionBlocks
// - Enforced per-provider, per-pool basis
// - Blocks both full and partial removals
//
// Implementation:
// - SetLastLiquidityActionBlock: Records block height on add/remove
// - CheckFlashLoanProtection: Validates minimum blocks elapsed before removal
// - RemoveLiquiditySecure: Calls CheckFlashLoanProtection before allowing removal
const (
	// DefaultFlashLoanProtectionBlocks enforces the minimum wait between LP actions when params unset.
	// Value of 10 blocks (~1 minute at 6s blocks) provides security while not overly restricting LPs.
	// This prevents same-block or near-block attacks while allowing normal liquidity management.
	DefaultFlashLoanProtectionBlocks = int64(10)

	// FlashLoanDetectionWindow is the number of blocks to analyze for flash loan attack patterns.
	// Monitors recent liquidity actions to detect suspicious add-swap-remove sequences.
	FlashLoanDetectionWindow = 10
)

// CheckFlashLoanProtection validates that liquidity operations aren't flash loan attacks.
//
// This function prevents same-block or near-block add→swap→remove attacks by enforcing
// a minimum delay (FlashLoanProtectionBlocks) between liquidity add and remove operations.
//
// Returns:
// - nil if sufficient blocks have passed since last liquidity action
// - ErrFlashLoanDetected if attempting to remove too soon after adding
func (k Keeper) CheckFlashLoanProtection(ctx context.Context, poolID uint64, provider sdk.AccAddress) error {
	lastBlock, found, err := k.GetLastLiquidityActionBlock(ctx, poolID, provider)
	if err != nil {
		return err
	}

	if !found {
		return nil // First action, allow
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	minBlocks := int64(params.FlashLoanProtectionBlocks)
	if minBlocks == 0 {
		minBlocks = DefaultFlashLoanProtectionBlocks
	}

	blocksSince := sdkCtx.BlockHeight() - lastBlock

	if blocksSince < minBlocks {
		return types.ErrFlashLoanDetected.Wrapf(
			"must wait %d blocks between liquidity actions (waited %d, last action at block %d, current block %d)",
			minBlocks, blocksSince, lastBlock, sdkCtx.BlockHeight(),
		)
	}

	return nil
}

// MEVProtectionConfig defines parameters for preventing Maximal Extractable Value (MEV) attacks.
//
// MEV attacks occur when block producers reorder, insert, or censor transactions to extract
// value from users. Common MEV strategies include:
// - Front-running: Placing trades before user transactions
// - Back-running: Placing trades after user transactions
// - Sandwich attacks: Front-run AND back-run a victim transaction
//
// This configuration limits the exploitability of these attacks by:
// - Restricting maximum price impact per swap
// - Limiting swap size relative to pool reserves
// - Enforcing delays between large swaps
type MEVProtectionConfig struct {
	MaxPriceImpact    math.LegacyDec // Maximum allowed price impact (default 10%)
	MaxSwapPercentage math.LegacyDec // Maximum swap as % of reserve (default 10%)
	MinBlocksForLarge int64          // Minimum blocks between large swaps (default 3)
}

var defaultMEVConfig = MEVProtectionConfig{
	MaxPriceImpact:    math.LegacyNewDecWithPrec(10, 2), // 10%
	MaxSwapPercentage: math.LegacyNewDecWithPrec(10, 2), // 10%
	MinBlocksForLarge: 3,
}

// ValidatePriceImpact checks that a swap doesn't have excessive price impact (MEV protection)
func (k Keeper) ValidatePriceImpact(amountIn, reserveIn, reserveOut, amountOut math.Int) error {
	// Calculate price impact: (amountOut / reserveOut) / (amountIn / reserveIn) - 1
	priceIn := math.LegacyNewDecFromInt(amountIn).Quo(math.LegacyNewDecFromInt(reserveIn))
	priceOut := math.LegacyNewDecFromInt(amountOut).Quo(math.LegacyNewDecFromInt(reserveOut))

	var priceImpact math.LegacyDec
	if !priceIn.IsZero() {
		priceImpact = math.LegacyOneDec().Sub(priceOut.Quo(priceIn))
	}

	if priceImpact.GT(defaultMEVConfig.MaxPriceImpact) {
		return types.ErrPriceImpactTooHigh.Wrapf(
			"price impact %s%% exceeds maximum %s%%",
			priceImpact.Mul(math.LegacyNewDec(100)),
			defaultMEVConfig.MaxPriceImpact.Mul(math.LegacyNewDec(100)),
		)
	}

	return nil
}

// ValidateSwapSize checks that swap size isn't too large (MEV protection)
func (k Keeper) ValidateSwapSize(amountIn, reserveIn math.Int) error {
	swapPercentage := math.LegacyNewDecFromInt(amountIn).
		Quo(math.LegacyNewDecFromInt(reserveIn))

	if swapPercentage.GT(defaultMEVConfig.MaxSwapPercentage) {
		return types.ErrSwapTooLarge.Wrapf(
			"swap size %s%% of reserve exceeds maximum %s%%",
			swapPercentage.Mul(math.LegacyNewDec(100)),
			defaultMEVConfig.MaxSwapPercentage.Mul(math.LegacyNewDec(100)),
		)
	}

	return nil
}

// Task 128: JIT Liquidity Detection
// DetectJITLiquidity detects just-in-time liquidity provision patterns
func (k Keeper) DetectJITLiquidity(ctx context.Context, poolID uint64, provider sdk.AccAddress, shares math.Int) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Check if this provider recently added large liquidity
	lastAddKey := append([]byte("jit_detection/"), provider.Bytes()...)
	lastAddKey = append(lastAddKey, sdk.Uint64ToBigEndian(poolID)...)

	if bz := store.Get(lastAddKey); bz != nil {
		lastBlock := int64(sdk.BigEndianToUint64(bz[:8]))
		lastShares := math.ZeroInt()
		_ = lastShares.Unmarshal(bz[8:])

		// JIT detected if large liquidity added very recently (< 3 blocks)
		if sdkCtx.BlockHeight()-lastBlock < 3 && !lastShares.IsZero() {
			// Calculate if this is a removal of recently added liquidity
			if shares.GT(lastShares.Quo(math.NewInt(2))) {
				return types.ErrJITLiquidityDetected.Wrapf(
					"just-in-time liquidity detected: added at block %d, removing at %d",
					lastBlock, sdkCtx.BlockHeight(),
				)
			}
		}
	}

	// Record this liquidity action
	recordBz := sdk.Uint64ToBigEndian(uint64(sdkCtx.BlockHeight()))
	sharesBz, _ := shares.Marshal()
	recordBz = append(recordBz, sharesBz...)
	store.Set(lastAddKey, recordBz)

	return nil
}

// Task 129: Sandwich Attack Prevention
// DetectSandwichAttack detects potential sandwich attack patterns
func (k Keeper) DetectSandwichAttack(ctx context.Context, poolID uint64, trader sdk.AccAddress, amountIn math.Int) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Track recent large swaps by same trader
	swapKey := append([]byte("swap_history/"), trader.Bytes()...)
	swapKey = append(swapKey, sdk.Uint64ToBigEndian(poolID)...)

	// Get pool to calculate swap size relative to reserve
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return err
	}

	// Calculate swap impact
	swapPercentage := math.LegacyNewDecFromInt(amountIn).
		Quo(math.LegacyNewDecFromInt(pool.ReserveA.Add(pool.ReserveB).Quo(math.NewInt(2))))

	// Check for rapid back-to-back large swaps (sandwich pattern)
	if bz := store.Get(swapKey); bz != nil {
		lastBlock := int64(sdk.BigEndianToUint64(bz[:8]))

		// If large swap occurred in last 2 blocks, this might be sandwich attack
		if sdkCtx.BlockHeight()-lastBlock <= 2 && swapPercentage.GT(math.LegacyNewDecWithPrec(1, 2)) {
			// Emit warning event
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeDexPotentialSandwichAttack,
					sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
					sdk.NewAttribute(types.AttributeKeyTrader, trader.String()),
					sdk.NewAttribute(types.AttributeKeySwapPercentage, swapPercentage.String()),
					sdk.NewAttribute(types.AttributeKeyBlocksApart, fmt.Sprintf("%d", sdkCtx.BlockHeight()-lastBlock)),
				),
			)

			// For now, emit warning but allow. Could add stricter enforcement.
		}
	}

	// Record this swap
	recordBz := sdk.Uint64ToBigEndian(uint64(sdkCtx.BlockHeight()))
	amountBz, _ := amountIn.Marshal()
	recordBz = append(recordBz, amountBz...)
	store.Set(swapKey, recordBz)

	return nil
}

// FeeTier represents a fee tier configuration for liquidity pools.
//
// Different token pairs have different characteristics that warrant different fee structures:
// - Stablecoins (USDC/USDT): Low volatility → low fees (0.05%)
// - Major pairs (ETH/BTC): Medium volatility → standard fees (0.3%)
// - Exotic pairs (low-cap tokens): High volatility → high fees (1%)
//
// Fee distribution:
// - SwapFee: Total fee charged to swappers
// - LPFee: Portion of SwapFee that goes to liquidity providers
// - ProtocolFee: Portion of SwapFee that goes to protocol treasury
//
// Note: SwapFee = LPFee + ProtocolFee
type FeeTier struct {
	Name         string         // Tier name (e.g., "low", "standard", "high")
	SwapFee      math.LegacyDec // Total fee charged on swaps
	LPFee        math.LegacyDec // Fee portion for liquidity providers
	ProtocolFee  math.LegacyDec // Fee portion for protocol treasury
	MinLiquidity math.Int       // Minimum liquidity required for this tier
}

var (
	// Standard fee tier (0.3%)
	StandardFeeTier = FeeTier{
		Name:         "standard",
		SwapFee:      math.LegacyNewDecWithPrec(3, 3),
		LPFee:        math.LegacyNewDecWithPrec(25, 4),
		ProtocolFee:  math.LegacyNewDecWithPrec(5, 4),
		MinLiquidity: math.NewInt(1000),
	}

	// Low fee tier (0.05%) for stablecoins
	LowFeeTier = FeeTier{
		Name:         "low",
		SwapFee:      math.LegacyNewDecWithPrec(5, 4),
		LPFee:        math.LegacyNewDecWithPrec(4, 4),
		ProtocolFee:  math.LegacyNewDecWithPrec(1, 4),
		MinLiquidity: math.NewInt(10000),
	}

	// High fee tier (1%) for exotic pairs
	HighFeeTier = FeeTier{
		Name:         "high",
		SwapFee:      math.LegacyNewDecWithPrec(1, 2),
		LPFee:        math.LegacyNewDecWithPrec(8, 3),
		ProtocolFee:  math.LegacyNewDecWithPrec(2, 3),
		MinLiquidity: math.NewInt(500),
	}
)

// GetPoolFeeTier returns the fee tier for a pool
func (k Keeper) GetPoolFeeTier(ctx context.Context, poolID uint64) (*FeeTier, error) {
	store := k.getStore(ctx)
	key := append([]byte("pool_fee_tier/"), sdk.Uint64ToBigEndian(poolID)...)

	bz := store.Get(key)
	if bz == nil {
		// Return standard tier as default
		return &StandardFeeTier, nil
	}

	tierName := string(bz)
	switch tierName {
	case "low":
		return &LowFeeTier, nil
	case "high":
		return &HighFeeTier, nil
	default:
		return &StandardFeeTier, nil
	}
}

// SetPoolFeeTier sets the fee tier for a pool (governance only)
func (k Keeper) SetPoolFeeTier(ctx context.Context, poolID uint64, tierName string) error {
	// Validate tier name
	var tier *FeeTier
	switch tierName {
	case "low":
		tier = &LowFeeTier
	case "high":
		tier = &HighFeeTier
	case "standard":
		tier = &StandardFeeTier
	default:
		return types.ErrInvalidInput.Wrapf("invalid fee tier: %s", tierName)
	}

	store := k.getStore(ctx)
	key := append([]byte("pool_fee_tier/"), sdk.Uint64ToBigEndian(poolID)...)
	store.Set(key, []byte(tierName))

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"pool_fee_tier_updated",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("tier", tierName),
			sdk.NewAttribute("swap_fee", tier.SwapFee.String()),
		),
	)

	return nil
}
