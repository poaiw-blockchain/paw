package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	"github.com/paw-chain/paw/x/dex/types"
)

// DefaultParams returns default parameters for the dex module
func DefaultParams() types.Params {
	return types.Params{
		SwapFee:                             math.LegacyNewDecWithPrec(3, 3),  // 0.3%
		LpFee:                               math.LegacyNewDecWithPrec(25, 4), // 0.25% (of 0.3%)
		ProtocolFee:                         math.LegacyNewDecWithPrec(5, 4),  // 0.05% (of 0.3%)
		MinLiquidity:                        math.NewInt(1000),                // Minimum initial liquidity
		MaxSlippagePercent:                  math.LegacyNewDecWithPrec(5, 2),  // 5%
		MaxPoolDrainPercent:                 math.LegacyNewDecWithPrec(30, 2), // 30% per swap
		FlashLoanProtectionBlocks:           100, // SEC-18: ~10 min at 6s blocks
		AuthorizedChannels:                  []types.AuthorizedChannel{},
		PoolCreationGas:                     1000,
		SwapValidationGas:                   1500,
		LiquidityGas:                        1200,
		UpgradePreserveCircuitBreakerState:  true,                             // Default to preserving pause state across upgrades
		RecommendedMaxSlippage:              math.LegacyNewDecWithPrec(3, 2),  // 3% recommended max
		EnableCommitReveal:                  false,                            // Disabled for testnet
		CommitRevealDelay:                   10,                               // 10 blocks
		CommitTimeoutBlocks:                 100,                              // 100 blocks
	}
}

// GetParams returns the current parameters from the store
func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	store := k.getStore(ctx)
	bz := store.Get(ParamsKey)
	if bz == nil {
		return DefaultParams(), nil
	}

	var params types.Params
	if err := k.cdc.Unmarshal(bz, &params); err != nil {
		return types.Params{}, fmt.Errorf("GetParams: unmarshal: %w", err)
	}
	return params, nil
}

// SetParams sets the parameters in the store
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return fmt.Errorf("SetParams: marshal: %w", err)
	}
	store.Set(ParamsKey, bz)
	return nil
}
