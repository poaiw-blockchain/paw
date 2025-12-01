package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/math"
	storeprefix "cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/paw-chain/paw/x/compute/types"
)

func (k Keeper) getNextSlashID(ctx context.Context) (uint64, error) {
	store := k.getStore(ctx)
	bz := store.Get(NextSlashIDKey)
	var nextID uint64 = 1
	if bz != nil {
		nextID = binary.BigEndian.Uint64(bz)
	}
	next := make([]byte, 8)
	binary.BigEndian.PutUint64(next, nextID+1)
	store.Set(NextSlashIDKey, next)
	return nextID, nil
}

func (k Keeper) setSlashRecord(ctx context.Context, record types.SlashRecord) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&record)
	if err != nil {
		return err
	}
	store.Set(SlashRecordKey(record.Id), bz)

	// index by provider
	provider, err := sdk.AccAddressFromBech32(record.Provider)
	if err != nil {
		return err
	}
	store.Set(SlashRecordByProviderKey(provider, record.Id), []byte{})
	return nil
}

func (k Keeper) getSlashRecord(ctx context.Context, id uint64) (*types.SlashRecord, error) {
	store := k.getStore(ctx)
	bz := store.Get(SlashRecordKey(id))
	if bz == nil {
		return nil, fmt.Errorf("slash record %d not found", id)
	}
	var record types.SlashRecord
	if err := k.cdc.Unmarshal(bz, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (k Keeper) recordSlash(ctx context.Context, provider sdk.AccAddress, requestID, disputeID uint64, amount math.Int, reason string) (uint64, error) {
	slashID, err := k.getNextSlashID(ctx)
	if err != nil {
		return 0, err
	}
	record := types.SlashRecord{
		Id:        slashID,
		Provider:  provider.String(),
		RequestId: requestID,
		DisputeId: disputeID,
		Amount:    amount,
		Reason:    reason,
		SlashedAt: sdk.UnwrapSDKContext(ctx).BlockTime(),
		Appealed:  false,
		AppealId:  0,
	}
	if err := k.setSlashRecord(ctx, record); err != nil {
		return 0, err
	}
	return slashID, nil
}

// listSlashRecords paginates slash records globally or by provider if address provided.
func (k Keeper) listSlashRecords(ctx context.Context, provider sdk.AccAddress, pageReq *query.PageRequest) ([]types.SlashRecord, *query.PageResponse, error) {
	store := k.getStore(ctx)
	var (
		records []types.SlashRecord
		pageRes *query.PageResponse
		err     error
	)

	if provider.Empty() {
		view := storeprefix.NewStore(store, SlashRecordKeyPrefix)
		pageRes, err = query.Paginate(view, pageReq, func(key []byte, value []byte) error {
			var rec types.SlashRecord
			if err := k.cdc.Unmarshal(value, &rec); err != nil {
				return err
			}
			records = append(records, rec)
			return nil
		})
	} else {
		prefix := SlashRecordByProviderKey(provider, 0)[:len(SlashRecordsByProviderPrefix)+len(provider.Bytes())]
		view := storeprefix.NewStore(store, prefix)
		pageRes, err = query.Paginate(view, pageReq, func(key []byte, value []byte) error {
			if len(key) < 8 {
				return nil
			}
			slashID := binary.BigEndian.Uint64(key[len(key)-8:])
			rec, err := k.getSlashRecord(ctx, slashID)
			if err != nil {
				return err
			}
			records = append(records, *rec)
			return nil
		})
	}

	if err != nil {
		return nil, nil, err
	}

	return records, pageRes, nil
}
