package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// Circuit breaker state keys
var (
	CircuitBreakerEnabledKey = []byte("circuit_breaker_enabled")
	CircuitBreakerReasonKey  = []byte("circuit_breaker_reason")
	CircuitBreakerActorKey   = []byte("circuit_breaker_actor")
)

// IsCircuitBreakerOpen checks if the DEX circuit breaker is open (paused)
func (k Keeper) IsCircuitBreakerOpen(ctx context.Context) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	bz := store.Get(CircuitBreakerEnabledKey)
	if bz == nil {
		return false
	}
	return bz[0] == 1
}

// OpenCircuitBreaker opens the circuit breaker (pauses all DEX operations)
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

// CloseCircuitBreaker closes the circuit breaker (resumes DEX operations)
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
			return fmt.Errorf("DEX operations paused by %s: %s", actor, reason)
		}
	}
	return nil
}

// Pool-specific circuit breakers

// IsPoolCircuitBreakerOpen checks if a specific pool's circuit breaker is open
func (k Keeper) IsPoolCircuitBreakerOpen(ctx context.Context, poolID uint64) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	key := append(CircuitBreakerEnabledKey, sdk.Uint64ToBigEndian(poolID)...)
	bz := store.Get(key)
	if bz == nil {
		return false
	}
	return bz[0] == 1
}

// OpenPoolCircuitBreaker opens the circuit breaker for a specific pool
func (k Keeper) OpenPoolCircuitBreaker(ctx context.Context, poolID uint64, actor, reason string) error {
	if k.IsPoolCircuitBreakerOpen(ctx, poolID) {
		return types.ErrCircuitBreakerAlreadyOpen
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	poolKey := append(CircuitBreakerEnabledKey, sdk.Uint64ToBigEndian(poolID)...)
	reasonKey := append(CircuitBreakerReasonKey, sdk.Uint64ToBigEndian(poolID)...)
	actorKey := append(CircuitBreakerActorKey, sdk.Uint64ToBigEndian(poolID)...)

	store.Set(poolKey, []byte{1})
	store.Set(reasonKey, []byte(reason))
	store.Set(actorKey, []byte(actor))

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCircuitBreakerOpen,
			sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute(types.AttributeKeyActor, actor),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	return nil
}

// ClosePoolCircuitBreaker closes the circuit breaker for a specific pool
func (k Keeper) ClosePoolCircuitBreaker(ctx context.Context, poolID uint64, actor, reason string) error {
	if !k.IsPoolCircuitBreakerOpen(ctx, poolID) {
		return types.ErrCircuitBreakerAlreadyClosed
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	poolKey := append(CircuitBreakerEnabledKey, sdk.Uint64ToBigEndian(poolID)...)
	reasonKey := append(CircuitBreakerReasonKey, sdk.Uint64ToBigEndian(poolID)...)
	actorKey := append(CircuitBreakerActorKey, sdk.Uint64ToBigEndian(poolID)...)

	store.Delete(poolKey)
	store.Delete(reasonKey)
	store.Delete(actorKey)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCircuitBreakerClose,
			sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute(types.AttributeKeyActor, actor),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	return nil
}

// CheckPoolCircuitBreaker checks if pool operations are allowed
func (k Keeper) CheckPoolCircuitBreaker(ctx context.Context, poolID uint64) error {
	// Check global circuit breaker first
	if err := k.CheckCircuitBreaker(ctx); err != nil {
		return fmt.Errorf("CheckPoolCircuitBreaker: global breaker: %w", err)
	}

	// Check pool-specific circuit breaker
	if k.IsPoolCircuitBreakerOpen(ctx, poolID) {
		return fmt.Errorf("pool %d operations paused", poolID)
	}

	return nil
}
