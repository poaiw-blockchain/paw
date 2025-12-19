package keeper

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/compute/types"
)

// Circuit breaker state keys
var (
	CircuitBreakerEnabledKey = []byte("circuit_breaker_enabled")
	CircuitBreakerReasonKey  = []byte("circuit_breaker_reason")
	CircuitBreakerActorKey   = []byte("circuit_breaker_actor")
	JobCancellationPrefix    = []byte("job_cancellation/")
	ReputationOverridePrefix = []byte("reputation_override/")
)

// IsCircuitBreakerOpen checks if the Compute circuit breaker is open (paused)
func (k Keeper) IsCircuitBreakerOpen(ctx context.Context) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	bz := store.Get(CircuitBreakerEnabledKey)
	if bz == nil {
		return false
	}
	return bz[0] == 1
}

// OpenCircuitBreaker opens the circuit breaker (pauses all Compute operations)
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

// CloseCircuitBreaker closes the circuit breaker (resumes Compute operations)
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
			return fmt.Errorf("Compute operations paused by %s: %s", actor, reason)
		}
	}
	return nil
}

// Provider-specific circuit breakers

// IsProviderCircuitBreakerOpen checks if a specific provider's circuit breaker is open
func (k Keeper) IsProviderCircuitBreakerOpen(ctx context.Context, providerAddr string) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)
	key := append(CircuitBreakerEnabledKey, []byte(providerAddr)...)
	bz := store.Get(key)
	if bz == nil {
		return false
	}
	return bz[0] == 1
}

// OpenProviderCircuitBreaker opens the circuit breaker for a specific provider
func (k Keeper) OpenProviderCircuitBreaker(ctx context.Context, providerAddr, actor, reason string) error {
	if k.IsProviderCircuitBreakerOpen(ctx, providerAddr) {
		return types.ErrCircuitBreakerAlreadyOpen
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	providerKey := append(CircuitBreakerEnabledKey, []byte(providerAddr)...)
	reasonKey := append(CircuitBreakerReasonKey, []byte(providerAddr)...)
	actorKey := append(CircuitBreakerActorKey, []byte(providerAddr)...)

	store.Set(providerKey, []byte{1})
	store.Set(reasonKey, []byte(reason))
	store.Set(actorKey, []byte(actor))

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCircuitBreakerOpen,
			sdk.NewAttribute(types.AttributeKeyProvider, providerAddr),
			sdk.NewAttribute(types.AttributeKeyActor, actor),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	return nil
}

// CloseProviderCircuitBreaker closes the circuit breaker for a specific provider
func (k Keeper) CloseProviderCircuitBreaker(ctx context.Context, providerAddr, actor, reason string) error {
	if !k.IsProviderCircuitBreakerOpen(ctx, providerAddr) {
		return types.ErrCircuitBreakerAlreadyClosed
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	providerKey := append(CircuitBreakerEnabledKey, []byte(providerAddr)...)
	reasonKey := append(CircuitBreakerReasonKey, []byte(providerAddr)...)
	actorKey := append(CircuitBreakerActorKey, []byte(providerAddr)...)

	store.Delete(providerKey)
	store.Delete(reasonKey)
	store.Delete(actorKey)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCircuitBreakerClose,
			sdk.NewAttribute(types.AttributeKeyProvider, providerAddr),
			sdk.NewAttribute(types.AttributeKeyActor, actor),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	return nil
}

// CheckProviderCircuitBreaker checks if provider operations are allowed
func (k Keeper) CheckProviderCircuitBreaker(ctx context.Context, providerAddr string) error {
	// Check global circuit breaker first
	if err := k.CheckCircuitBreaker(ctx); err != nil {
		return err
	}

	// Check provider-specific circuit breaker
	if k.IsProviderCircuitBreakerOpen(ctx, providerAddr) {
		return fmt.Errorf("provider %s operations paused", providerAddr)
	}

	return nil
}

// Job cancellation

// CancelJob marks a job for emergency cancellation
func (k Keeper) CancelJob(ctx context.Context, jobID, actor, reason string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	// Store cancellation metadata
	metadata := map[string]string{
		"job_id":    jobID,
		"actor":     actor,
		"reason":    reason,
		"timestamp": fmt.Sprintf("%d", sdkCtx.BlockTime().Unix()),
	}

	// Encode as JSON for simplicity
	// In production, this would use a proto message
	bz, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	key := append(JobCancellationPrefix, []byte(jobID)...)
	store.Set(key, bz)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeJobCancellation,
			sdk.NewAttribute(types.AttributeKeyJobID, jobID),
			sdk.NewAttribute(types.AttributeKeyActor, actor),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	return nil
}

// IsJobCancelled checks if a job has been marked for cancellation
func (k Keeper) IsJobCancelled(ctx context.Context, jobID string) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	key := append(JobCancellationPrefix, []byte(jobID)...)
	return store.Has(key)
}

// GetJobCancellation retrieves the cancellation details for a job
func (k Keeper) GetJobCancellation(ctx context.Context, jobID string) (map[string]string, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	key := append(JobCancellationPrefix, []byte(jobID)...)
	bz := store.Get(key)
	if bz == nil {
		return nil, false
	}

	var metadata map[string]string
	if err := json.Unmarshal(bz, &metadata); err != nil {
		return nil, false
	}

	return metadata, true
}

// ClearJobCancellation removes a job cancellation record
func (k Keeper) ClearJobCancellation(ctx context.Context, jobID string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	key := append(JobCancellationPrefix, []byte(jobID)...)
	store.Delete(key)
}

// Reputation override

// SetReputationOverride sets a temporary reputation override for a provider
func (k Keeper) SetReputationOverride(ctx context.Context, providerAddr string, score int64, actor, reason string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	metadata := map[string]string{
		"provider":  providerAddr,
		"score":     fmt.Sprintf("%d", score),
		"actor":     actor,
		"reason":    reason,
		"timestamp": fmt.Sprintf("%d", sdkCtx.BlockTime().Unix()),
	}

	bz, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	key := append(ReputationOverridePrefix, []byte(providerAddr)...)
	store.Set(key, bz)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeReputationOverride,
			sdk.NewAttribute(types.AttributeKeyProvider, providerAddr),
			sdk.NewAttribute(types.AttributeKeyScore, fmt.Sprintf("%d", score)),
			sdk.NewAttribute(types.AttributeKeyActor, actor),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	return nil
}

// GetReputationOverride retrieves a reputation override if it exists
func (k Keeper) GetReputationOverride(ctx context.Context, providerAddr string) (int64, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	key := append(ReputationOverridePrefix, []byte(providerAddr)...)
	bz := store.Get(key)
	if bz == nil {
		return 0, false
	}

	var metadata map[string]string
	if err := json.Unmarshal(bz, &metadata); err != nil {
		return 0, false
	}

	var score int64
	if _, err := fmt.Sscanf(metadata["score"], "%d", &score); err != nil {
		return 0, false
	}

	return score, true
}

// ClearReputationOverride removes a reputation override
func (k Keeper) ClearReputationOverride(ctx context.Context, providerAddr string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	key := append(ReputationOverridePrefix, []byte(providerAddr)...)
	store.Delete(key)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeReputationOverrideClear,
			sdk.NewAttribute(types.AttributeKeyProvider, providerAddr),
		),
	)
}

// GetReputationWithOverride retrieves a provider's reputation, checking for overrides first
func (k Keeper) GetReputationWithOverride(ctx context.Context, providerAddrStr string) (int64, bool) {
	// Check for override first
	if overrideScore, hasOverride := k.GetReputationOverride(ctx, providerAddrStr); hasOverride {
		return overrideScore, true
	}

	// Fall back to normal reputation retrieval
	providerAddr, err := sdk.AccAddressFromBech32(providerAddrStr)
	if err != nil {
		return 0, false
	}

	rep, err := k.GetProviderReputation(ctx, providerAddr)
	if err != nil {
		return 0, false
	}
	return int64(rep.OverallScore), true
}
