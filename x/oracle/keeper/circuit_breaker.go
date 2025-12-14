package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/oracle/types"
)

// Circuit breaker state keys
var (
	CircuitBreakerEnabledKey = []byte("circuit_breaker_enabled")
	CircuitBreakerReasonKey  = []byte("circuit_breaker_reason")
	CircuitBreakerActorKey   = []byte("circuit_breaker_actor")
	PriceOverridePrefix      = []byte("price_override/")
	SlashingDisabledKey      = []byte("slashing_disabled")
)

// IsCircuitBreakerOpen checks if the Oracle circuit breaker is open (paused)
func (k Keeper) IsCircuitBreakerOpen(ctx context.Context) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	bz := store.Get(CircuitBreakerEnabledKey)
	if bz == nil {
		return false
	}
	return bz[0] == 1
}

// OpenCircuitBreaker opens the circuit breaker (pauses all Oracle operations)
func (k Keeper) OpenCircuitBreaker(ctx context.Context, actor, reason string) error {
	if k.IsCircuitBreakerOpen(ctx) {
		return types.ErrCircuitBreakerAlreadyOpen
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	store.Set(CircuitBreakerEnabledKey, []byte{1})
	store.Set(CircuitBreakerReasonKey, []byte(reason))
	store.Set(CircuitBreakerActorKey, []byte(actor))

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCircuitBreakerOpen,
			sdk.NewAttribute(types.AttributeKeyActor, actor),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	return nil
}

// CloseCircuitBreaker closes the circuit breaker (resumes Oracle operations)
func (k Keeper) CloseCircuitBreaker(ctx context.Context, actor, reason string) error {
	if !k.IsCircuitBreakerOpen(ctx) {
		return types.ErrCircuitBreakerAlreadyClosed
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	store.Delete(CircuitBreakerEnabledKey)
	store.Delete(CircuitBreakerReasonKey)
	store.Delete(CircuitBreakerActorKey)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCircuitBreakerClose,
			sdk.NewAttribute(types.AttributeKeyActor, actor),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	return nil
}

// GetCircuitBreakerState retrieves the current circuit breaker state
func (k Keeper) GetCircuitBreakerState(ctx context.Context) (bool, string, string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	enabled := store.Get(CircuitBreakerEnabledKey)
	if enabled == nil || enabled[0] == 0 {
		return false, "", ""
	}

	reason := string(store.Get(CircuitBreakerReasonKey))
	actor := string(store.Get(CircuitBreakerActorKey))

	return true, reason, actor
}

// CheckCircuitBreaker checks if operations are allowed and returns an error if blocked
func (k Keeper) CheckCircuitBreaker(ctx context.Context) error {
	if k.IsCircuitBreakerOpen(ctx) {
		enabled, reason, actor := k.GetCircuitBreakerState(ctx)
		if enabled {
			return fmt.Errorf("Oracle operations paused by %s: %s", actor, reason)
		}
	}
	return nil
}

// Price override functionality

// SetPriceOverride sets an emergency price override for a trading pair
func (k Keeper) SetPriceOverride(ctx context.Context, pair string, price *big.Int, durationSecs int64, actor, reason string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	expiresAt := sdkCtx.BlockTime().Unix() + durationSecs

	// Store as a simple encoded structure
	// In production, this would use a proto message
	override := &types.PriceData{
		Pair:      pair,
		Price:     price.String(),
		Timestamp: expiresAt,
		Source:    actor,
	}

	bz := k.cdc.MustMarshal(override)
	key := append(PriceOverridePrefix, []byte(pair)...)
	store.Set(key, bz)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypePriceOverride,
			sdk.NewAttribute(types.AttributeKeyPair, pair),
			sdk.NewAttribute(types.AttributeKeyPrice, price.String()),
			sdk.NewAttribute(types.AttributeKeyActor, actor),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	return nil
}

// GetPriceOverride retrieves the price override for a pair if it exists and is not expired
func (k Keeper) GetPriceOverride(ctx context.Context, pair string) (*big.Int, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	key := append(PriceOverridePrefix, []byte(pair)...)
	bz := store.Get(key)
	if bz == nil {
		return nil, false
	}

	var override types.PriceData
	k.cdc.MustUnmarshal(bz, &override)

	// Check if expired
	if sdkCtx.BlockTime().Unix() > override.Timestamp {
		store.Delete(key)
		return nil, false
	}

	price := new(big.Int)
	price.SetString(override.Price, 10)
	return price, true
}

// ClearPriceOverride removes a price override
func (k Keeper) ClearPriceOverride(ctx context.Context, pair string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	key := append(PriceOverridePrefix, []byte(pair)...)
	store.Delete(key)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypePriceOverrideClear,
			sdk.NewAttribute(types.AttributeKeyPair, pair),
		),
	)
}

// GetPriceWithOverride retrieves a price, checking for overrides first
func (k Keeper) GetPriceWithOverride(ctx context.Context, pair string) (*big.Int, bool) {
	// Check for override first
	if overridePrice, hasOverride := k.GetPriceOverride(ctx, pair); hasOverride {
		return overridePrice, true
	}

	// Fall back to normal price retrieval
	return k.GetPrice(ctx, pair)
}

// Slashing control

// DisableSlashing temporarily disables Oracle slashing
func (k Keeper) DisableSlashing(ctx context.Context, actor, reason string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	store.Set(SlashingDisabledKey, []byte{1})

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSlashingDisabled,
			sdk.NewAttribute(types.AttributeKeyActor, actor),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	return nil
}

// EnableSlashing re-enables Oracle slashing
func (k Keeper) EnableSlashing(ctx context.Context, actor, reason string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	store.Delete(SlashingDisabledKey)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSlashingEnabled,
			sdk.NewAttribute(types.AttributeKeyActor, actor),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	return nil
}

// IsSlashingDisabled checks if slashing is currently disabled
func (k Keeper) IsSlashingDisabled(ctx context.Context) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	bz := store.Get(SlashingDisabledKey)
	if bz == nil {
		return false
	}
	return bz[0] == 1
}

// Feed-specific circuit breakers

// IsFeedCircuitBreakerOpen checks if a specific feed's circuit breaker is open
func (k Keeper) IsFeedCircuitBreakerOpen(ctx context.Context, feedType string) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	key := append(CircuitBreakerEnabledKey, []byte(feedType)...)
	bz := store.Get(key)
	if bz == nil {
		return false
	}
	return bz[0] == 1
}

// OpenFeedCircuitBreaker opens the circuit breaker for a specific feed type
func (k Keeper) OpenFeedCircuitBreaker(ctx context.Context, feedType, actor, reason string) error {
	if k.IsFeedCircuitBreakerOpen(ctx, feedType) {
		return types.ErrCircuitBreakerAlreadyOpen
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	feedKey := append(CircuitBreakerEnabledKey, []byte(feedType)...)
	reasonKey := append(CircuitBreakerReasonKey, []byte(feedType)...)
	actorKey := append(CircuitBreakerActorKey, []byte(feedType)...)

	store.Set(feedKey, []byte{1})
	store.Set(reasonKey, []byte(reason))
	store.Set(actorKey, []byte(actor))

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCircuitBreakerOpen,
			sdk.NewAttribute(types.AttributeKeyFeedType, feedType),
			sdk.NewAttribute(types.AttributeKeyActor, actor),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	return nil
}

// CloseFeedCircuitBreaker closes the circuit breaker for a specific feed type
func (k Keeper) CloseFeedCircuitBreaker(ctx context.Context, feedType, actor, reason string) error {
	if !k.IsFeedCircuitBreakerOpen(ctx, feedType) {
		return types.ErrCircuitBreakerAlreadyClosed
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	feedKey := append(CircuitBreakerEnabledKey, []byte(feedType)...)
	reasonKey := append(CircuitBreakerReasonKey, []byte(feedType)...)
	actorKey := append(CircuitBreakerActorKey, []byte(feedType)...)

	store.Delete(feedKey)
	store.Delete(reasonKey)
	store.Delete(actorKey)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCircuitBreakerClose,
			sdk.NewAttribute(types.AttributeKeyFeedType, feedType),
			sdk.NewAttribute(types.AttributeKeyActor, actor),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	return nil
}

// CheckFeedCircuitBreaker checks if feed operations are allowed
func (k Keeper) CheckFeedCircuitBreaker(ctx context.Context, feedType string) error {
	// Check global circuit breaker first
	if err := k.CheckCircuitBreaker(ctx); err != nil {
		return err
	}

	// Check feed-specific circuit breaker
	if k.IsFeedCircuitBreakerOpen(ctx, feedType) {
		return fmt.Errorf("feed type %s operations paused", feedType)
	}

	return nil
}
