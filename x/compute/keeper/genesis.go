package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/compute/types"
)

// InitGenesis initializes the compute module's state from a genesis state
func (k Keeper) InitGenesis(ctx context.Context, data types.GenesisState) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if err := k.BindPort(sdkCtx); err != nil {
		return fmt.Errorf("failed to bind IBC port: %w", err)
	}

	// Set params
	if err := k.SetParams(ctx, data.Params); err != nil {
		return fmt.Errorf("failed to set params: %w", err)
	}

	// Set governance params (use defaults if not provided)
	govParams := data.GovernanceParams
	if govParams.DisputeDeposit.IsNil() || !govParams.DisputeDeposit.IsPositive() {
		govParams = types.DefaultGovernanceParams()
	}
	if err := k.SetGovernanceParams(ctx, govParams); err != nil {
		return fmt.Errorf("failed to set governance params: %w", err)
	}

	var maxRequestID uint64

	// Initialize providers
	for _, provider := range data.Providers {
		if err := k.SetProvider(ctx, provider); err != nil {
			return fmt.Errorf("failed to initialize provider %s: %w", provider.Address, err)
		}

		// Set active provider index if active
		if provider.Active {
			addr, err := sdk.AccAddressFromBech32(provider.Address)
			if err != nil {
				return fmt.Errorf("invalid provider address %s: %w", provider.Address, err)
			}
			if err := k.setActiveProviderIndex(ctx, addr, true); err != nil {
				return fmt.Errorf("failed to set active provider index: %w", err)
			}
		}
	}

	// Initialize requests
	for _, request := range data.Requests {
		if err := k.SetRequest(ctx, request); err != nil {
			return fmt.Errorf("failed to initialize request %d: %w", request.Id, err)
		}

		if request.Id > maxRequestID {
			maxRequestID = request.Id
		}

		// Set request indexes
		if err := k.setRequestIndexes(ctx, request); err != nil {
			return fmt.Errorf("failed to set request indexes for request %d: %w", request.Id, err)
		}
	}

	// Initialize results
	for _, result := range data.Results {
		if err := k.SetResult(ctx, &result); err != nil {
			return fmt.Errorf("failed to initialize result for request %d: %w", result.RequestId, err)
		}
	}

	// Set next request ID
	nextRequestID := data.NextRequestId
	if nextRequestID == 0 || nextRequestID <= maxRequestID {
		nextRequestID = maxRequestID + 1
	}
	if err := k.setNextRequestID(ctx, nextRequestID); err != nil {
		return fmt.Errorf("failed to set next request ID: %w", err)
	}

	var (
		maxDisputeID uint64
		maxSlashID   uint64
		maxAppealID  uint64
	)

	// Initialize disputes
	for _, dispute := range data.Disputes {
		if dispute.Id > maxDisputeID {
			maxDisputeID = dispute.Id
		}
		if err := k.setDispute(ctx, dispute); err != nil {
			return fmt.Errorf("failed to initialize dispute %d: %w", dispute.Id, err)
		}
	}

	nextDisputeID := data.NextDisputeId
	if nextDisputeID == 0 || nextDisputeID <= maxDisputeID {
		nextDisputeID = maxDisputeID + 1
	}
	if err := k.setNextDisputeID(ctx, nextDisputeID); err != nil {
		return fmt.Errorf("failed to set next dispute ID: %w", err)
	}

	// Initialize slash records
	for _, record := range data.SlashRecords {
		if record.Id > maxSlashID {
			maxSlashID = record.Id
		}
		if err := k.setSlashRecord(ctx, record); err != nil {
			return fmt.Errorf("failed to initialize slash record %d: %w", record.Id, err)
		}
	}

	nextSlashID := data.NextSlashId
	if nextSlashID == 0 || nextSlashID <= maxSlashID {
		nextSlashID = maxSlashID + 1
	}
	if err := k.setNextSlashID(ctx, nextSlashID); err != nil {
		return fmt.Errorf("failed to set next slash ID: %w", err)
	}

	// Initialize appeals
	for _, appeal := range data.Appeals {
		if appeal.Id > maxAppealID {
			maxAppealID = appeal.Id
		}
		if err := k.setAppeal(ctx, appeal); err != nil {
			return fmt.Errorf("failed to initialize appeal %d: %w", appeal.Id, err)
		}
	}

	nextAppealID := data.NextAppealId
	if nextAppealID == 0 || nextAppealID <= maxAppealID {
		nextAppealID = maxAppealID + 1
	}
	if err := k.setNextAppealID(ctx, nextAppealID); err != nil {
		return fmt.Errorf("failed to set next appeal ID: %w", err)
	}

	var maxEscrowNonce uint64

	// Initialize escrow states
	for _, escrowState := range data.EscrowStates {
		if escrowState.Nonce > maxEscrowNonce {
			maxEscrowNonce = escrowState.Nonce
		}

		if err := k.SetEscrowState(ctx, escrowState); err != nil {
			return fmt.Errorf("failed to initialize escrow state for request %d: %w", escrowState.RequestId, err)
		}

		// Restore timeout indexes for LOCKED and CHALLENGED escrows
		// This is critical for automatic expiry processing in EndBlocker
		if escrowState.Status == types.ESCROW_STATUS_LOCKED || escrowState.Status == types.ESCROW_STATUS_CHALLENGED {
			if err := k.setEscrowTimeoutIndex(ctx, escrowState.RequestId, escrowState.ExpiresAt); err != nil {
				return fmt.Errorf("failed to restore timeout index for escrow %d: %w", escrowState.RequestId, err)
			}
		}
	}

	// Set next escrow nonce
	nextEscrowNonce := data.NextEscrowNonce
	if nextEscrowNonce == 0 || nextEscrowNonce <= maxEscrowNonce {
		nextEscrowNonce = maxEscrowNonce + 1
	}
	if err := k.setNextEscrowNonce(ctx, nextEscrowNonce); err != nil {
		return fmt.Errorf("failed to set next escrow nonce: %w", err)
	}

	return nil
}

// ExportGenesis exports the compute module's state to a genesis state
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	// Get params
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get params: %w", err)
	}

	govParams, err := k.GetGovernanceParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get governance params: %w", err)
	}

	// Collect all providers
	var providers []types.Provider
	if err := k.IterateProviders(ctx, func(provider types.Provider) (bool, error) {
		providers = append(providers, provider)
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to iterate providers: %w", err)
	}

	// Collect all requests
	var requests []types.Request
	if err := k.IterateRequests(ctx, func(request types.Request) (bool, error) {
		requests = append(requests, request)
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to iterate requests: %w", err)
	}

	// Collect all results
	var results []types.Result
	for _, request := range requests {
		result, err := k.GetResult(ctx, request.Id)
		if err == nil {
			results = append(results, *result)
		}
	}

	// Get next request ID
	nextRequestID, err := k.getNextRequestIDForExport(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get next request ID: %w", err)
	}

	store := k.getStore(ctx)

	// Collect disputes
	var disputes []types.Dispute
	disputeIter := storetypes.KVStorePrefixIterator(store, DisputeKeyPrefix)
	defer disputeIter.Close()
	for ; disputeIter.Valid(); disputeIter.Next() {
		var dispute types.Dispute
		if err := k.cdc.Unmarshal(disputeIter.Value(), &dispute); err != nil {
			return nil, fmt.Errorf("failed to unmarshal dispute: %w", err)
		}
		disputes = append(disputes, dispute)
	}

	// Collect slash records
	var slashRecords []types.SlashRecord
	slashIter := storetypes.KVStorePrefixIterator(store, SlashRecordKeyPrefix)
	defer slashIter.Close()
	for ; slashIter.Valid(); slashIter.Next() {
		var record types.SlashRecord
		if err := k.cdc.Unmarshal(slashIter.Value(), &record); err != nil {
			return nil, fmt.Errorf("failed to unmarshal slash record: %w", err)
		}
		slashRecords = append(slashRecords, record)
	}

	// Collect appeals
	var appeals []types.Appeal
	appealIter := storetypes.KVStorePrefixIterator(store, AppealKeyPrefix)
	defer appealIter.Close()
	for ; appealIter.Valid(); appealIter.Next() {
		var appeal types.Appeal
		if err := k.cdc.Unmarshal(appealIter.Value(), &appeal); err != nil {
			return nil, fmt.Errorf("failed to unmarshal appeal: %w", err)
		}
		appeals = append(appeals, appeal)
	}

	nextDisputeID, err := k.getNextDisputeIDForExport(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get next dispute ID: %w", err)
	}

	nextSlashID, err := k.getNextSlashIDForExport(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get next slash ID: %w", err)
	}

	nextAppealID, err := k.getNextAppealIDForExport(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get next appeal ID: %w", err)
	}

	// Collect all escrow states
	var escrowStates []types.EscrowState
	escrowIter := storetypes.KVStorePrefixIterator(store, EscrowStateKeyPrefix)
	defer escrowIter.Close()
	for ; escrowIter.Valid(); escrowIter.Next() {
		var escrowState types.EscrowState
		if err := k.cdc.Unmarshal(escrowIter.Value(), &escrowState); err != nil {
			return nil, fmt.Errorf("failed to unmarshal escrow state: %w", err)
		}
		escrowStates = append(escrowStates, escrowState)
	}

	// Get next escrow nonce
	nextEscrowNonce, err := k.getNextEscrowNonceForExport(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get next escrow nonce: %w", err)
	}

	return &types.GenesisState{
		Params:           params,
		GovernanceParams: govParams,
		Providers:        providers,
		Requests:         requests,
		Results:          results,
		Disputes:         disputes,
		SlashRecords:     slashRecords,
		Appeals:          appeals,
		EscrowStates:     escrowStates,
		NextRequestId:    nextRequestID,
		NextDisputeId:    nextDisputeID,
		NextSlashId:      nextSlashID,
		NextAppealId:     nextAppealID,
		NextEscrowNonce:  nextEscrowNonce,
	}, nil
}

// setNextRequestID sets the next request ID (used in genesis initialization)
func (k Keeper) setNextRequestID(ctx context.Context, nextID uint64) error {
	store := k.getStore(ctx)
	bz := make([]byte, 8)
	bz[0] = byte(nextID >> 56)
	bz[1] = byte(nextID >> 48)
	bz[2] = byte(nextID >> 40)
	bz[3] = byte(nextID >> 32)
	bz[4] = byte(nextID >> 24)
	bz[5] = byte(nextID >> 16)
	bz[6] = byte(nextID >> 8)
	bz[7] = byte(nextID)
	store.Set(NextRequestIDKey, bz)
	return nil
}

// getNextRequestIDForExport gets the current next request ID without incrementing
func (k Keeper) getNextRequestIDForExport(ctx context.Context) (uint64, error) {
	store := k.getStore(ctx)
	bz := store.Get(NextRequestIDKey)

	if bz == nil {
		return 1, nil
	}

	nextID := uint64(bz[0])<<56 | uint64(bz[1])<<48 | uint64(bz[2])<<40 | uint64(bz[3])<<32 |
		uint64(bz[4])<<24 | uint64(bz[5])<<16 | uint64(bz[6])<<8 | uint64(bz[7])

	return nextID, nil
}

func (k Keeper) setNextDisputeID(ctx context.Context, nextID uint64) error {
	store := k.getStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, nextID)
	store.Set(NextDisputeIDKey, bz)
	return nil
}

func (k Keeper) setNextSlashID(ctx context.Context, nextID uint64) error {
	store := k.getStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, nextID)
	store.Set(NextSlashIDKey, bz)
	return nil
}

func (k Keeper) setNextAppealID(ctx context.Context, nextID uint64) error {
	store := k.getStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, nextID)
	store.Set(NextAppealIDKey, bz)
	return nil
}

func (k Keeper) getNextDisputeIDForExport(ctx context.Context) (uint64, error) {
	store := k.getStore(ctx)
	bz := store.Get(NextDisputeIDKey)
	if bz == nil {
		return 1, nil
	}
	return binary.BigEndian.Uint64(bz), nil
}

func (k Keeper) getNextSlashIDForExport(ctx context.Context) (uint64, error) {
	store := k.getStore(ctx)
	bz := store.Get(NextSlashIDKey)
	if bz == nil {
		return 1, nil
	}
	return binary.BigEndian.Uint64(bz), nil
}

func (k Keeper) getNextAppealIDForExport(ctx context.Context) (uint64, error) {
	store := k.getStore(ctx)
	bz := store.Get(NextAppealIDKey)
	if bz == nil {
		return 1, nil
	}
	return binary.BigEndian.Uint64(bz), nil
}

func (k Keeper) getNextEscrowNonceForExport(ctx context.Context) (uint64, error) {
	store := k.getStore(ctx)
	bz := store.Get(NextEscrowNonceKey)
	if bz == nil {
		return 1, nil
	}
	return binary.BigEndian.Uint64(bz), nil
}

func (k Keeper) setNextEscrowNonce(ctx context.Context, nextNonce uint64) error {
	store := k.getStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, nextNonce)
	store.Set(NextEscrowNonceKey, bz)
	return nil
}
