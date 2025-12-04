package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	"github.com/paw-chain/paw/x/dex/types"
)

// GetPoolTWAP retrieves the TWAP record for a pool if it exists.
func (k Keeper) GetPoolTWAP(ctx context.Context, poolID uint64) (*types.PoolTWAP, bool, error) {
	store := k.getStore(ctx)
	bz := store.Get(PoolTWAPKey(poolID))
	if bz == nil {
		return nil, false, nil
	}

	var record types.PoolTWAP
	if err := k.cdc.Unmarshal(bz, &record); err != nil {
		return nil, false, err
	}
	return &record, true, nil
}

// SetPoolTWAP stores the TWAP metadata for a pool.
func (k Keeper) SetPoolTWAP(ctx context.Context, record types.PoolTWAP) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&record)
	if err != nil {
		return err
	}
	store.Set(PoolTWAPKey(record.PoolId), bz)
	return nil
}

// GetAllPoolTWAPs returns all TWAP records in state.
func (k Keeper) GetAllPoolTWAPs(ctx context.Context) ([]types.PoolTWAP, error) {
	store := k.getStore(ctx)
	iter := storetypes.KVStorePrefixIterator(store, PoolTWAPKeyPrefix)
	defer iter.Close()

	var records []types.PoolTWAP
	for ; iter.Valid(); iter.Next() {
		var record types.PoolTWAP
		if err := k.cdc.Unmarshal(iter.Value(), &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}
