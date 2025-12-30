package types

import (
	"context"

	sdkmath "cosmossdk.io/math"
)

// DexHooks defines the interface for DEX module callbacks.
// ARCH-2: Enables cross-module notifications for DEX events.
type DexHooks interface {
	// AfterSwap is called after a successful swap operation.
	// Enables post-swap processing (e.g., analytics, fee distribution).
	AfterSwap(ctx context.Context, poolID uint64, sender string, tokenIn, tokenOut string, amountIn, amountOut sdkmath.Int) error

	// AfterPoolCreated is called after a new liquidity pool is created.
	// Enables pool tracking and indexing by external modules.
	AfterPoolCreated(ctx context.Context, poolID uint64, tokenA, tokenB string, creator string) error

	// AfterLiquidityChanged is called when liquidity is added or removed.
	// Enables TVL tracking and liquidity monitoring.
	AfterLiquidityChanged(ctx context.Context, poolID uint64, provider string, deltaA, deltaB sdkmath.Int, isAdd bool) error

	// OnCircuitBreakerTriggered is called when the DEX circuit breaker activates.
	// Dependent modules should pause DEX-related operations.
	OnCircuitBreakerTriggered(ctx context.Context, reason string) error
}

// MultiDexHooks combines multiple DEX hooks into a single hook that calls all of them.
type MultiDexHooks []DexHooks

// NewMultiDexHooks creates a new MultiDexHooks from a list of hooks.
func NewMultiDexHooks(hooks ...DexHooks) MultiDexHooks {
	return hooks
}

// AfterSwap calls AfterSwap on all registered hooks.
func (h MultiDexHooks) AfterSwap(ctx context.Context, poolID uint64, sender string, tokenIn, tokenOut string, amountIn, amountOut sdkmath.Int) error {
	for _, hook := range h {
		if hook == nil {
			continue
		}
		if err := hook.AfterSwap(ctx, poolID, sender, tokenIn, tokenOut, amountIn, amountOut); err != nil {
			return err
		}
	}
	return nil
}

// AfterPoolCreated calls AfterPoolCreated on all registered hooks.
func (h MultiDexHooks) AfterPoolCreated(ctx context.Context, poolID uint64, tokenA, tokenB string, creator string) error {
	for _, hook := range h {
		if hook == nil {
			continue
		}
		if err := hook.AfterPoolCreated(ctx, poolID, tokenA, tokenB, creator); err != nil {
			return err
		}
	}
	return nil
}

// AfterLiquidityChanged calls AfterLiquidityChanged on all registered hooks.
func (h MultiDexHooks) AfterLiquidityChanged(ctx context.Context, poolID uint64, provider string, deltaA, deltaB sdkmath.Int, isAdd bool) error {
	for _, hook := range h {
		if hook == nil {
			continue
		}
		if err := hook.AfterLiquidityChanged(ctx, poolID, provider, deltaA, deltaB, isAdd); err != nil {
			return err
		}
	}
	return nil
}

// OnCircuitBreakerTriggered calls OnCircuitBreakerTriggered on all registered hooks.
func (h MultiDexHooks) OnCircuitBreakerTriggered(ctx context.Context, reason string) error {
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
