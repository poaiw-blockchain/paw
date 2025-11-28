package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/compute/types"
)

// InitGenesis initializes the compute module's state from a genesis state
func (k Keeper) InitGenesis(ctx context.Context, data types.GenesisState) error {
	// Set params
	if err := k.SetParams(ctx, data.Params); err != nil {
		return fmt.Errorf("failed to set params: %w", err)
	}

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
	if err := k.setNextRequestID(ctx, data.NextRequestId); err != nil {
		return fmt.Errorf("failed to set next request ID: %w", err)
	}

	// Initialize escrow states if provided
	/*
	for _, escrow := range data.EscrowStates {
		if err := k.SetEscrowState(ctx, escrow); err != nil {
			return fmt.Errorf("failed to initialize escrow state for request %d: %w", escrow.RequestID, err)
		}

		// Recreate timeout index if escrow is not yet released/refunded
		if escrow.Status == types.EscrowStatus_ESCROW_STATUS_LOCKED ||
		   escrow.Status == types.EscrowStatus_ESCROW_STATUS_CHALLENGED {
			if err := k.setEscrowTimeoutIndex(ctx, escrow.RequestID, escrow.ExpiresAt); err != nil {
				return fmt.Errorf("failed to set escrow timeout index for request %d: %w", escrow.RequestID, err)
			}
		}
	}
	*/

	/*
	// Set next escrow nonce if provided
	if data.NextEscrowNonce > 0 {
		store := k.getStore(ctx)
		nonceBz := make([]byte, 8)
		for i := 0; i < 8; i++ {
			nonceBz[i] = byte(data.NextEscrowNonce >> (8 * (7 - i)))
		}
		store.Set(NextEscrowNonceKey, nonceBz)
	}
	*/

	return nil
}

// ExportGenesis exports the compute module's state to a genesis state
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	// Get params
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get params: %w", err)
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

	// Export all escrow states
	var escrowStates []types.EscrowState
	for _, request := range requests {
		escrow, err := k.GetEscrowState(ctx, request.Id)
		if err == nil && escrow != nil {
			escrowStates = append(escrowStates, *escrow)
		}
	}

	// Get next escrow nonce
	store := k.getStore(ctx)
	var nextEscrowNonce uint64 = 1
	nonceBz := store.Get(NextEscrowNonceKey)
	if nonceBz != nil && len(nonceBz) == 8 {
		for i := 0; i < 8; i++ {
			nextEscrowNonce |= uint64(nonceBz[i]) << (8 * (7 - i))
		}
	}

	return &types.GenesisState{
		Params:           params,
		Providers:        providers,
		Requests:         requests,
		Results:          results,
		NextRequestId:    nextRequestID,
		// EscrowStates:     escrowStates,
		// NextEscrowNonce:  nextEscrowNonce,
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
