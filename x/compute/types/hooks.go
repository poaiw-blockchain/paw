package types

import (
	"context"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ComputeHooks defines the interface for compute module callbacks.
// ARCH-2: Enables cross-module notifications for compute events.
type ComputeHooks interface {
	// AfterJobCompleted is called when a compute job finishes successfully.
	// Enables post-job processing (e.g., result validation, callback execution).
	AfterJobCompleted(ctx context.Context, requestID uint64, provider sdk.AccAddress, result []byte) error

	// AfterJobFailed is called when a compute job fails or times out.
	// Enables failure handling and retry logic in dependent modules.
	AfterJobFailed(ctx context.Context, requestID uint64, reason string) error

	// AfterProviderRegistered is called when a new compute provider registers.
	// Enables provider indexing and capacity tracking.
	AfterProviderRegistered(ctx context.Context, provider sdk.AccAddress, stake sdkmath.Int) error

	// AfterProviderSlashed is called when a provider is slashed for misbehavior.
	// Enables reputation tracking and dependent module updates.
	AfterProviderSlashed(ctx context.Context, provider sdk.AccAddress, slashAmount sdkmath.Int, reason string) error

	// OnCircuitBreakerTriggered is called when the compute circuit breaker activates.
	// Dependent modules should pause compute-related operations.
	OnCircuitBreakerTriggered(ctx context.Context, reason string) error
}

// MultiComputeHooks combines multiple compute hooks into a single hook that calls all of them.
type MultiComputeHooks []ComputeHooks

// NewMultiComputeHooks creates a new MultiComputeHooks from a list of hooks.
func NewMultiComputeHooks(hooks ...ComputeHooks) MultiComputeHooks {
	return hooks
}

// AfterJobCompleted calls AfterJobCompleted on all registered hooks.
func (h MultiComputeHooks) AfterJobCompleted(ctx context.Context, requestID uint64, provider sdk.AccAddress, result []byte) error {
	for _, hook := range h {
		if hook == nil {
			continue
		}
		if err := hook.AfterJobCompleted(ctx, requestID, provider, result); err != nil {
			return err
		}
	}
	return nil
}

// AfterJobFailed calls AfterJobFailed on all registered hooks.
func (h MultiComputeHooks) AfterJobFailed(ctx context.Context, requestID uint64, reason string) error {
	for _, hook := range h {
		if hook == nil {
			continue
		}
		if err := hook.AfterJobFailed(ctx, requestID, reason); err != nil {
			return err
		}
	}
	return nil
}

// AfterProviderRegistered calls AfterProviderRegistered on all registered hooks.
func (h MultiComputeHooks) AfterProviderRegistered(ctx context.Context, provider sdk.AccAddress, stake sdkmath.Int) error {
	for _, hook := range h {
		if hook == nil {
			continue
		}
		if err := hook.AfterProviderRegistered(ctx, provider, stake); err != nil {
			return err
		}
	}
	return nil
}

// AfterProviderSlashed calls AfterProviderSlashed on all registered hooks.
func (h MultiComputeHooks) AfterProviderSlashed(ctx context.Context, provider sdk.AccAddress, slashAmount sdkmath.Int, reason string) error {
	for _, hook := range h {
		if hook == nil {
			continue
		}
		if err := hook.AfterProviderSlashed(ctx, provider, slashAmount, reason); err != nil {
			return err
		}
	}
	return nil
}

// OnCircuitBreakerTriggered calls OnCircuitBreakerTriggered on all registered hooks.
func (h MultiComputeHooks) OnCircuitBreakerTriggered(ctx context.Context, reason string) error {
	for _, hook := range h {
		if hook == nil {
			continue
		}
		if err := hook.OnCircuitBreakerTriggered(ctx, reason); err != nil {
			return err
		}
	}
	return nil
}
