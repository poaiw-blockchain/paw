package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	"github.com/paw-chain/paw/x/compute/types"
)

// GetParams retrieves the module parameters from the store
func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	store := k.getStore(ctx)
	bz := store.Get(ParamsKey)

	if bz == nil {
		return k.DefaultParams(), nil
	}

	var params types.Params
	if err := k.cdc.Unmarshal(bz, &params); err != nil {
		return types.Params{}, fmt.Errorf("GetParams: unmarshal: %w", err)
	}

	return params, nil
}

// SetParams stores the module parameters.
// Note: This does NOT automatically invalidate the authorized channels cache.
// Callers that modify AuthorizedChannels must call InvalidateChannelCache() separately,
// or use SetAuthorizedChannels/SetAuthorizedChannelsWithValidation for automatic invalidation.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return fmt.Errorf("SetParams: marshal: %w", err)
	}

	store.Set(ParamsKey, bz)
	return nil
}

// DefaultParams returns the default module parameters
func (k Keeper) DefaultParams() types.Params {
	return types.Params{
		MinProviderStake:           math.NewInt(1000000), // 1 token with 6 decimals
		VerificationTimeoutSeconds: 3600,                 // 1 hour
		MaxRequestTimeoutSeconds:   86400,                // 24 hours
		ReputationSlashPercentage:  10,                   // 10% slash on failure
		StakeSlashPercentage:       5,                    // 5% stake slash on malicious behavior
		MinReputationScore:         50,                   // minimum 50/100 reputation
		EscrowReleaseDelaySeconds:  300,                  // 5 minutes delay
		AuthorizedChannels:         []types.AuthorizedChannel{},
	}
}
