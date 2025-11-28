package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/paw-chain/paw/x/dex/types"
)

// DefaultParams returns default parameters for the dex module
func DefaultParams() types.Params {
	return types.Params{
		SwapFee:            math.LegacyNewDecWithPrec(3, 3),  // 0.3%
		LpFee:              math.LegacyNewDecWithPrec(25, 4), // 0.25% (of 0.3%)
		ProtocolFee:        math.LegacyNewDecWithPrec(5, 4),  // 0.05% (of 0.3%)
		MinLiquidity:       math.NewInt(1000),                // Minimum initial liquidity
		MaxSlippagePercent: math.LegacyNewDecWithPrec(5, 2),  // 5%
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
		return types.Params{}, err
	}
	return params, nil
}

// SetParams sets the parameters in the store
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(ParamsKey, bz)
	return nil
}
