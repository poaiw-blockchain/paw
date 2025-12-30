package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// FeeCollector manages DEX fee collection and distribution
type FeeCollector struct {
	keeper *Keeper
}

// NewFeeCollector creates a new fee collector
func NewFeeCollector(k *Keeper) *FeeCollector {
	return &FeeCollector{keeper: k}
}

// CollectSwapFees calculates and collects fees from a swap operation
func (k Keeper) CollectSwapFees(ctx context.Context, poolID uint64, tokenIn string, amountIn math.Int) (lpFee, protocolFee math.Int, err error) {
	// Get parameters
	params, err := k.GetParams(ctx)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), fmt.Errorf("CollectSwapFees: get params: %w", err)
	}
	// Calculate total swap fee
	totalFeeAmount := math.LegacyNewDecFromInt(amountIn).Mul(params.SwapFee).TruncateInt()

	// Calculate LP fee (goes to liquidity providers)
	lpFee = math.LegacyNewDecFromInt(totalFeeAmount).Mul(params.LpFee).TruncateInt()

	// Calculate protocol fee (goes to protocol treasury)
	protocolFee = math.LegacyNewDecFromInt(totalFeeAmount).Mul(params.ProtocolFee).TruncateInt()

	// Ensure fees don't exceed total fee
	totalCalculated := lpFee.Add(protocolFee)

	if totalCalculated.GT(totalFeeAmount) {
		// Adjust to prevent overflow
		protocolFee = totalFeeAmount.Sub(lpFee)
	}

	// Store fees for the pool
	if err := k.accumulateFees(ctx, poolID, tokenIn, lpFee, protocolFee); err != nil {
		return math.ZeroInt(), math.ZeroInt(), fmt.Errorf("CollectSwapFees: accumulate fees: %w", err)
	}

	// Transfer collected fees to the fee collector module account
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress()
	feeCollectorAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	if !totalFeeAmount.IsZero() {
		totalFees := sdk.NewCoins(sdk.NewCoin(tokenIn, totalFeeAmount))
		if err := k.bankKeeper.SendCoins(sdkCtx, moduleAddr, feeCollectorAddr, totalFees); err != nil {
			return math.ZeroInt(), math.ZeroInt(), fmt.Errorf("failed to send swap fees to collector: %w", err)
		}
	}

	return lpFee, protocolFee, nil
}

// accumulateFees stores accumulated fees for a pool
func (k Keeper) accumulateFees(ctx context.Context, poolID uint64, token string, lpFee, protocolFee math.Int) error {
	store := k.getStore(ctx)

	// Store LP fees (accumulated in the pool, claimed by LPs proportionally)
	lpFeeKey := types.GetPoolLPFeeKey(poolID, token)
	currentLPFee := math.ZeroInt()

	if bz := store.Get(lpFeeKey); bz != nil {
		if err := currentLPFee.Unmarshal(bz); err != nil {
			return types.ErrInvalidState.Wrap("failed to unmarshal LP fee")
		}
	}

	newLPFee := currentLPFee.Add(lpFee)

	bz, err := newLPFee.Marshal()
	if err != nil {
		return types.ErrInvalidState.Wrap("failed to marshal LP fee")
	}
	store.Set(lpFeeKey, bz)

	// Store protocol fees (accumulated globally, claimed by governance)
	protocolFeeKey := types.GetProtocolFeeKey(token)
	currentProtocolFee := math.ZeroInt()

	if bz := store.Get(protocolFeeKey); bz != nil {
		if err := currentProtocolFee.Unmarshal(bz); err != nil {
			return types.ErrInvalidState.Wrap("failed to unmarshal protocol fee")
		}
	}

	newProtocolFee := currentProtocolFee.Add(protocolFee)

	bz, err = newProtocolFee.Marshal()
	if err != nil {
		return types.ErrInvalidState.Wrap("failed to marshal protocol fee")
	}
	store.Set(protocolFeeKey, bz)

	// Emit fee collection event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dex_fees_collected",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("token", token),
			sdk.NewAttribute("lp_fee", lpFee.String()),
			sdk.NewAttribute("protocol_fee", protocolFee.String()),
		),
	)

	return nil
}

// GetPoolLPFees returns accumulated LP fees for a pool and token
func (k Keeper) GetPoolLPFees(ctx context.Context, poolID uint64, token string) (math.Int, error) {
	store := k.getStore(ctx)
	key := types.GetPoolLPFeeKey(poolID, token)

	bz := store.Get(key)
	if bz == nil {
		return math.ZeroInt(), nil
	}

	fee := math.ZeroInt()
	if err := fee.Unmarshal(bz); err != nil {
		return math.ZeroInt(), types.ErrInvalidState.Wrap("failed to unmarshal LP fee")
	}

	return fee, nil
}

// GetProtocolFees returns accumulated protocol fees for a token
func (k Keeper) GetProtocolFees(ctx context.Context, token string) (math.Int, error) {
	store := k.getStore(ctx)
	key := types.GetProtocolFeeKey(token)

	bz := store.Get(key)
	if bz == nil {
		return math.ZeroInt(), nil
	}

	fee := math.ZeroInt()
	if err := fee.Unmarshal(bz); err != nil {
		return math.ZeroInt(), types.ErrInvalidState.Wrap("failed to unmarshal protocol fee")
	}

	return fee, nil
}

// ClaimLPFees allows liquidity providers to claim their share of fees
// FIXED DATA-5: Uses CacheContext pattern for atomic state changes.
// Previously, fee state was updated BEFORE SendCoins - if transfer failed,
// state was corrupted with reduced fees but user didn't receive tokens.
// Now: all state changes happen in cache, transfer occurs, cache is only
// written if transfer succeeds. This follows checks-effects-interactions.
func (k Keeper) ClaimLPFees(ctx context.Context, provider sdk.AccAddress, poolID uint64) error {
	// CHECKS: Validate inputs and calculate fees without modifying state
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return fmt.Errorf("ClaimLPFees: get pool: %w", err)
	}

	shares, err := k.GetLiquidityShares(ctx, poolID, provider)
	if err != nil {
		return fmt.Errorf("ClaimLPFees: get liquidity shares: %w", err)
	}

	if shares.IsZero() {
		return types.ErrInsufficientLiquidity.Wrap("no liquidity shares to claim fees")
	}

	// Calculate provider's share of fees for each token (read-only)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress()

	// Structure to track fee updates for atomic application
	type feeUpdate struct {
		token         string
		providerShare math.Int
		newLPFees     math.Int
	}
	var feeUpdates []feeUpdate
	coinsToSend := sdk.NewCoins()

	for _, token := range []string{pool.TokenA, pool.TokenB} {
		totalLPFees, err := k.GetPoolLPFees(ctx, poolID, token)
		if err != nil {
			return fmt.Errorf("ClaimLPFees: get pool LP fees for %s: %w", token, err)
		}

		if totalLPFees.IsZero() {
			continue
		}

		// Calculate provider's share: (provider_shares / total_shares) * total_lp_fees
		providerShare := math.LegacyNewDecFromInt(shares).
			Quo(math.LegacyNewDecFromInt(pool.TotalShares)).
			Mul(math.LegacyNewDecFromInt(totalLPFees)).
			TruncateInt()

		if providerShare.IsZero() {
			continue
		}

		// Track the update for later atomic application
		feeUpdates = append(feeUpdates, feeUpdate{
			token:         token,
			providerShare: providerShare,
			newLPFees:     totalLPFees.Sub(providerShare),
		})
		coinsToSend = coinsToSend.Add(sdk.NewCoin(token, providerShare))
	}

	// Nothing to claim
	if coinsToSend.IsZero() {
		return nil
	}

	// INTERACTIONS: Perform external call (SendCoins) FIRST
	// Use CacheContext so if transfer fails, no state changes persist
	cacheCtx, writeFn := sdkCtx.CacheContext()

	// Apply state updates in cache context
	for _, update := range feeUpdates {
		store := k.getStore(cacheCtx)
		key := types.GetPoolLPFeeKey(poolID, update.token)
		bz, err := update.newLPFees.Marshal()
		if err != nil {
			return types.ErrInvalidState.Wrap("failed to marshal LP fee")
		}
		store.Set(key, bz)
	}

	// Transfer in cache context - if this fails, cache is discarded
	if err := k.bankKeeper.SendCoins(cacheCtx, moduleAddr, provider, coinsToSend); err != nil {
		// Cache is automatically discarded - no state corruption
		return types.ErrInsufficientLiquidity.Wrapf("failed to send fees: %v", err)
	}

	// EFFECTS: Only commit state changes AFTER successful transfer
	writeFn()

	// Emit event (after successful commit)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dex_lp_fees_claimed",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("provider", provider.String()),
			sdk.NewAttribute("shares", shares.String()),
			sdk.NewAttribute("fees_claimed", coinsToSend.String()),
		),
	)

	return nil
}

// WithdrawProtocolFees allows governance to withdraw protocol fees
// Uses CacheContext pattern for atomic state changes (same fix as ClaimLPFees).
func (k Keeper) WithdrawProtocolFees(ctx context.Context, recipient sdk.AccAddress, token string, amount math.Int) error {
	// CHECKS: Validate inputs without modifying state
	totalFees, err := k.GetProtocolFees(ctx, token)
	if err != nil {
		return fmt.Errorf("WithdrawProtocolFees: get protocol fees for %s: %w", token, err)
	}

	if amount.GT(totalFees) {
		return types.ErrInsufficientLiquidity.Wrapf(
			"withdrawal amount %s exceeds available fees %s",
			amount, totalFees,
		)
	}

	// Calculate new fee balance
	newFees := totalFees.Sub(amount)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress()
	coin := sdk.NewCoin(token, amount)

	// Use CacheContext for atomic execution
	cacheCtx, writeFn := sdkCtx.CacheContext()

	// Apply state update in cache
	store := k.getStore(cacheCtx)
	key := types.GetProtocolFeeKey(token)
	bz, err := newFees.Marshal()
	if err != nil {
		return types.ErrInvalidState.Wrap("failed to marshal protocol fee")
	}
	store.Set(key, bz)

	// INTERACTIONS: Transfer in cache context
	if err := k.bankKeeper.SendCoins(cacheCtx, moduleAddr, recipient, sdk.NewCoins(coin)); err != nil {
		// Cache is automatically discarded - no state corruption
		return types.ErrInsufficientLiquidity.Wrapf("failed to send fees: %v", err)
	}

	// EFFECTS: Commit state changes only after successful transfer
	writeFn()

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dex_protocol_fees_withdrawn",
			sdk.NewAttribute("token", token),
			sdk.NewAttribute("amount", amount.String()),
			sdk.NewAttribute("recipient", recipient.String()),
		),
	)

	return nil
}

// GetLiquidityShares returns the liquidity shares for a provider in a pool
func (k Keeper) GetLiquidityShares(ctx context.Context, poolID uint64, provider sdk.AccAddress) (math.Int, error) {
	store := k.getStore(ctx)
	key := types.GetLiquidityShareKey(poolID, provider)

	bz := store.Get(key)
	if bz == nil {
		return math.ZeroInt(), nil
	}

	shares := math.ZeroInt()
	if err := shares.Unmarshal(bz); err != nil {
		return math.ZeroInt(), types.ErrInvalidState.Wrap("failed to unmarshal shares")
	}

	return shares, nil
}

// SetLiquidityShares sets the liquidity shares for a provider in a pool
func (k Keeper) SetLiquidityShares(ctx context.Context, poolID uint64, provider sdk.AccAddress, shares math.Int) error {
	store := k.getStore(ctx)
	key := types.GetLiquidityShareKey(poolID, provider)

	if shares.IsZero() {
		store.Delete(key)
		return nil
	}

	bz, err := shares.Marshal()
	if err != nil {
		return types.ErrInvalidState.Wrap("failed to marshal shares")
	}
	store.Set(key, bz)

	return nil
}

// DistributeFees is called during EndBlock to distribute accumulated fees
func (k Keeper) DistributeFees(ctx context.Context) error {
	// This could be extended to automatically distribute fees
	// For now, fees accumulate and are claimed on-demand
	return nil
}

// GetTotalProtocolFeesValue returns the total value of all protocol fees
func (k Keeper) GetTotalProtocolFeesValue(ctx context.Context) (sdk.Coins, error) {
	store := k.getStore(ctx)

	// Iterate over all protocol fee entries
	coins := sdk.NewCoins()
	prefix := types.ProtocolFeeKeyPrefix

	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// Extract token from key
		token := string(iterator.Key()[len(prefix):])

		// Unmarshal fee amount
		fee := math.ZeroInt()
		if err := fee.Unmarshal(iterator.Value()); err != nil {
			continue // Skip invalid entries
		}

		if !fee.IsZero() {
			coins = coins.Add(sdk.NewCoin(token, fee))
		}
	}

	return coins, nil
}
