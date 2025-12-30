package keeper

import (
	"context"
	"fmt"

	"github.com/paw-chain/paw/x/oracle/types"
)

// GetParams retrieves the oracle module parameters
func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	store := k.getStore(ctx)
	bz := store.Get(ParamsKey)
	if bz == nil {
		return types.Params{}, nil
	}

	var params types.Params
	if err := k.cdc.Unmarshal(bz, &params); err != nil {
		return types.Params{}, err
	}
	return params, nil
}

// SetParams sets the oracle module parameters
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return fmt.Errorf("SetParams: failed to marshal params: %w", err)
	}
	store.Set(ParamsKey, bz)
	return nil
}
