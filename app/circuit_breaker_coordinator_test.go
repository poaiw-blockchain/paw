package app

import (
	"context"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// TestCircuitBreakerStatus tests the CircuitBreakerStatus struct.
func TestCircuitBreakerStatus(t *testing.T) {
	// Test default values
	status := CircuitBreakerStatus{}
	require.False(t, status.OracleOpen)
	require.False(t, status.DEXOpen)
	require.False(t, status.ComputeOpen)
	require.False(t, status.AnyOpen)

	// Test with one breaker open
	status.OracleOpen = true
	status.AnyOpen = status.OracleOpen || status.DEXOpen || status.ComputeOpen
	require.True(t, status.AnyOpen)

	// Test with reason
	status.OracleReason = "price manipulation detected"
	require.Equal(t, "price manipulation detected", status.OracleReason)
}

// TestCircuitBreakerCoordinatorWithNilKeepers tests coordinator with nil keepers.
func TestCircuitBreakerCoordinatorWithNilKeepers(t *testing.T) {
	// Create coordinator with nil keepers
	coordinator := NewCircuitBreakerCoordinator(nil, nil, nil)
	require.NotNil(t, coordinator)

	// Test availability checks - should return true when keepers are nil
	ctx := context.Background()
	require.True(t, coordinator.IsOracleAvailable(ctx))
	require.True(t, coordinator.IsDEXAvailable(ctx))
	require.True(t, coordinator.IsComputeAvailable(ctx))

	// GetGlobalStatus should not panic with nil keepers
	status := coordinator.GetGlobalStatus(ctx)
	require.False(t, status.AnyOpen)
}

// TestOracleHooksAdapterInterface ensures the adapter implements the interface.
func TestOracleHooksAdapterInterface(t *testing.T) {
	coordinator := NewCircuitBreakerCoordinator(nil, nil, nil)
	adapter := NewOracleHooksAdapter(coordinator)
	require.NotNil(t, adapter)

	ctx := context.Background()

	// Test all hook methods return nil
	err := adapter.AfterPriceAggregated(ctx, "BTC/USD", sdkmath.LegacyOneDec(), 100)
	require.NoError(t, err)

	err = adapter.AfterPriceSubmitted(ctx, "validator1", "BTC/USD", sdkmath.LegacyOneDec())
	require.NoError(t, err)

	// OnCircuitBreakerTriggered should not panic with nil keepers
	err = adapter.OnCircuitBreakerTriggered(ctx, "test reason")
	require.NoError(t, err)
}

// TestDexHooksAdapterInterface ensures the adapter implements the interface.
func TestDexHooksAdapterInterface(t *testing.T) {
	coordinator := NewCircuitBreakerCoordinator(nil, nil, nil)
	adapter := NewDexHooksAdapter(coordinator)
	require.NotNil(t, adapter)

	ctx := context.Background()

	// Test all hook methods return nil
	err := adapter.AfterSwap(ctx, 1, "sender", "tokenA", "tokenB", sdkmath.OneInt(), sdkmath.OneInt())
	require.NoError(t, err)

	err = adapter.AfterPoolCreated(ctx, 1, "tokenA", "tokenB", "creator")
	require.NoError(t, err)

	err = adapter.AfterLiquidityChanged(ctx, 1, "provider", sdkmath.OneInt(), sdkmath.OneInt(), true)
	require.NoError(t, err)

	// OnCircuitBreakerTriggered should not panic with nil keepers
	err = adapter.OnCircuitBreakerTriggered(ctx, "test reason")
	require.NoError(t, err)
}

// TestComputeHooksAdapterInterface ensures the adapter implements the interface.
func TestComputeHooksAdapterInterface(t *testing.T) {
	coordinator := NewCircuitBreakerCoordinator(nil, nil, nil)
	adapter := NewComputeHooksAdapter(coordinator)
	require.NotNil(t, adapter)

	ctx := context.Background()
	addr := sdk.AccAddress([]byte("test-address"))

	// Test all hook methods return nil
	err := adapter.AfterJobCompleted(ctx, 1, addr, []byte("result"))
	require.NoError(t, err)

	err = adapter.AfterJobFailed(ctx, 1, "test failure")
	require.NoError(t, err)

	err = adapter.AfterProviderRegistered(ctx, addr, sdkmath.OneInt())
	require.NoError(t, err)

	err = adapter.AfterProviderSlashed(ctx, addr, sdkmath.OneInt(), "test slash")
	require.NoError(t, err)

	// OnCircuitBreakerTriggered should not panic with nil coordinator
	adapter2 := &ComputeHooksAdapter{coordinator: nil}
	err = adapter2.OnCircuitBreakerTriggered(ctx, "test reason")
	require.NoError(t, err)
}

// TestEventTypes tests that event and attribute constants are defined correctly.
func TestEventTypes(t *testing.T) {
	require.Equal(t, "circuit_breaker_propagated", EventTypeCircuitBreakerPropagated)
	require.Equal(t, "source_module", AttributeKeySourceModule)
	require.Equal(t, "target_module", AttributeKeyTargetModule)
	require.Equal(t, "propagation_reason", AttributeKeyPropagationReason)
}

// TestModuleNames tests that module name constants are defined correctly.
func TestModuleNames(t *testing.T) {
	require.Equal(t, "oracle", ModuleOracle)
	require.Equal(t, "dex", ModuleDEX)
	require.Equal(t, "compute", ModuleCompute)
}
