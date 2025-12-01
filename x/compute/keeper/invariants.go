package keeper

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/compute/types"
)

// RegisterInvariants registers all compute module invariants
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "escrow-balance",
		EscrowBalanceInvariant(k))
	ir.RegisterRoute(types.ModuleName, "provider-stake",
		ProviderStakeInvariant(k))
	ir.RegisterRoute(types.ModuleName, "request-status",
		RequestStatusInvariant(k))
	ir.RegisterRoute(types.ModuleName, "nonce-uniqueness",
		NonceUniquenessInvariant(k))
	ir.RegisterRoute(types.ModuleName, "dispute-index",
		DisputeIndexInvariant(k))
	ir.RegisterRoute(types.ModuleName, "appeal-index",
		AppealIndexInvariant(k))
}

// AllInvariants runs all invariants of the compute module
func AllInvariants(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		res, stop := EscrowBalanceInvariant(k)(ctx)
		if stop {
			return res, stop
		}
		res, stop = ProviderStakeInvariant(k)(ctx)
		if stop {
			return res, stop
		}
		res, stop = RequestStatusInvariant(k)(ctx)
		if stop {
			return res, stop
		}
		res, stop = NonceUniquenessInvariant(k)(ctx)
		if stop {
			return res, stop
		}
		res, stop = DisputeIndexInvariant(k)(ctx)
		if stop {
			return res, stop
		}
		return AppealIndexInvariant(k)(ctx)
	}
}

// EscrowBalanceInvariant checks that the sum of all escrow amounts equals the module account balance
func EscrowBalanceInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			totalEscrow sdk.Coins
			broken      bool
			msg         string
		)

		// Get module account balance
		moduleAddr := k.accountKeeper.GetModuleAddress(types.ModuleName)
		moduleBalance := k.bankKeeper.GetAllBalances(ctx, moduleAddr)

		// Sum all escrow amounts from active requests
		err := k.IterateRequests(ctx, func(request types.Request) (bool, error) {
			if request.Status == types.REQUEST_STATUS_PENDING ||
				request.Status == types.REQUEST_STATUS_PROCESSING {
				totalEscrow = totalEscrow.Add(sdk.NewCoin("upaw", request.EscrowedAmount))
			}
			return false, nil
		})

		if err != nil {
			broken = true
			msg = fmt.Sprintf("error iterating requests: %v", err)
			return sdk.FormatInvariant(
				types.ModuleName, "escrow-balance",
				msg,
			), broken
		}

		// Add all provider stakes to total escrow
		err = k.IterateProviders(ctx, func(provider types.Provider) (bool, error) {
			totalEscrow = totalEscrow.Add(sdk.NewCoin("upaw", provider.Stake))
			return false, nil
		})

		if err != nil {
			broken = true
			msg = fmt.Sprintf("error iterating providers: %v", err)
			return sdk.FormatInvariant(
				types.ModuleName, "escrow-balance",
				msg,
			), broken
		}

		// Check if totals match
		if !totalEscrow.Equal(moduleBalance) {
			broken = true
			msg = fmt.Sprintf(
				"total escrow does not match module balance\n"+
					"\ttotal escrow: %s\n"+
					"\tmodule balance: %s\n"+
					"\tdifference: %s",
				totalEscrow, moduleBalance, totalEscrow.Sub(moduleBalance...),
			)
		}

		return sdk.FormatInvariant(
			types.ModuleName, "escrow-balance",
			msg,
		), broken
	}
}

// ProviderStakeInvariant checks that all provider stakes meet minimum requirements
func ProviderStakeInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			broken bool
			msg    string
			count  int
		)

		params, err := k.GetParams(ctx)
		if err != nil {
			broken = true
			msg = fmt.Sprintf("error getting params: %v", err)
			return sdk.FormatInvariant(
				types.ModuleName, "provider-stake",
				msg,
			), broken
		}

		minStake := params.MinProviderStake

		err = k.IterateProviders(ctx, func(provider types.Provider) (bool, error) {
			// Only active providers need to maintain minimum stake
			if !provider.Active {
				return false, nil
			}

			if provider.Stake.LT(minStake) {
				broken = true
				count++
				msg += fmt.Sprintf(
					"active provider %s has stake %s below minimum %s\n",
					provider.Address, provider.Stake, minStake,
				)
			}
			return false, nil
		})

		if err != nil {
			broken = true
			msg = fmt.Sprintf("error iterating providers: %v", err)
		}

		if count > 0 {
			msg = fmt.Sprintf("%d active providers with insufficient stake\n%s", count, msg)
		}

		return sdk.FormatInvariant(
			types.ModuleName, "provider-stake",
			msg,
		), broken
	}
}

// RequestStatusInvariant checks that request states are consistent
func RequestStatusInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			broken bool
			msg    string
			issues []string
		)

		err := k.IterateRequests(ctx, func(request types.Request) (bool, error) {
			// Check that completed or failed requests have results
			if request.Status == types.REQUEST_STATUS_COMPLETED ||
				request.Status == types.REQUEST_STATUS_FAILED {
				result, err := k.GetResult(ctx, request.Id)
				if err != nil || result == nil {
					issues = append(issues, fmt.Sprintf(
						"request %d has status %s but no result",
						request.Id, request.Status.String(),
					))
					broken = true
				}
			}

			// Check that processing requests have assigned providers
			if request.Status == types.REQUEST_STATUS_PROCESSING {
				if request.Provider == "" {
					issues = append(issues, fmt.Sprintf(
						"request %d is processing but has no assigned provider",
						request.Id,
					))
					broken = true
				}
			}

			// Check that escrow amounts are positive for active requests
			if request.Status == types.REQUEST_STATUS_PENDING ||
				request.Status == types.REQUEST_STATUS_PROCESSING {
				if request.EscrowedAmount.IsZero() {
					issues = append(issues, fmt.Sprintf(
						"request %d is active but has zero escrow",
						request.Id,
					))
					broken = true
				}
			}

			return false, nil
		})

		if err != nil {
			broken = true
			msg = fmt.Sprintf("error iterating requests: %v", err)
		}

		if len(issues) > 0 {
			msg = fmt.Sprintf("%d inconsistent request states:\n", len(issues))
			for _, issue := range issues {
				msg += fmt.Sprintf("  - %s\n", issue)
			}
		}

		return sdk.FormatInvariant(
			types.ModuleName, "request-status",
			msg,
		), broken
	}
}

// NonceUniquenessInvariant checks that nonces are unique per provider
func NonceUniquenessInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			broken bool
			msg    string
		)

		// Track nonces per provider to check for duplicates
		providerNonces := make(map[string]map[uint64]bool)

		store := k.getStore(ctx)
		iter := storetypes.KVStorePrefixIterator(store, NonceKeyPrefix)
		defer iter.Close()

		duplicates := 0
		for ; iter.Valid(); iter.Next() {
			// Extract provider and nonce from key
			// Format: NonceKeyPrefix(1) + provider(20) + nonce(8)
			key := iter.Key()
			if len(key) < 29 { // 1 + 20 + 8
				continue
			}

			provider := sdk.AccAddress(key[1:21]).String()
			nonce := sdk.BigEndianToUint64(key[21:29])

			if providerNonces[provider] == nil {
				providerNonces[provider] = make(map[uint64]bool)
			}

			if providerNonces[provider][nonce] {
				broken = true
				duplicates++
				msg += fmt.Sprintf(
					"duplicate nonce %d for provider %s\n",
					nonce, provider,
				)
			}

			providerNonces[provider][nonce] = true
		}

		if duplicates > 0 {
			msg = fmt.Sprintf("%d duplicate nonces found:\n%s", duplicates, msg)
		}

		return sdk.FormatInvariant(
			types.ModuleName, "nonce-uniqueness",
			msg,
		), broken
	}
}

// DisputeIndexInvariant ensures disputes have valid indexes and counters.
func DisputeIndexInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		store := k.getStore(ctx)
		iter := storetypes.KVStorePrefixIterator(store, DisputeKeyPrefix)
		defer iter.Close()

		var (
			broken    bool
			msg       string
			maxID     uint64
			failCount int
		)

		for ; iter.Valid(); iter.Next() {
			var dispute types.Dispute
			if err := k.cdc.Unmarshal(iter.Value(), &dispute); err != nil {
				broken = true
				failCount++
				msg += fmt.Sprintf("failed to unmarshal dispute: %v\n", err)
				continue
			}

			if dispute.Id > maxID {
				maxID = dispute.Id
			}

			// verify request index exists
			if !store.Has(DisputeByRequestKey(dispute.RequestId, dispute.Id)) {
				broken = true
				failCount++
				msg += fmt.Sprintf("missing dispute-by-request index for dispute %d\n", dispute.Id)
			}

			// verify status index exists
			if !store.Has(DisputeByStatusKey(uint32(dispute.Status), dispute.Id)) {
				broken = true
				failCount++
				msg += fmt.Sprintf("missing dispute-by-status index for dispute %d\n", dispute.Id)
			}
		}

		// ensure next ID counter is ahead of max ID
		nextID, err := k.getNextDisputeIDForExport(ctx)
		if err == nil && nextID <= maxID {
			broken = true
			failCount++
			msg += fmt.Sprintf("next dispute id %d not greater than max id %d\n", nextID, maxID)
		}

		if failCount > 0 && msg == "" {
			msg = "dispute index invariant failed"
		}

		return sdk.FormatInvariant(types.ModuleName, "dispute-index", msg), broken
	}
}

// AppealIndexInvariant ensures appeals have valid indexes and references.
func AppealIndexInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		store := k.getStore(ctx)
		iter := storetypes.KVStorePrefixIterator(store, AppealKeyPrefix)
		defer iter.Close()

		var (
			broken    bool
			msg       string
			maxID     uint64
			failCount int
		)

		for ; iter.Valid(); iter.Next() {
			var appeal types.Appeal
			if err := k.cdc.Unmarshal(iter.Value(), &appeal); err != nil {
				broken = true
				failCount++
				msg += fmt.Sprintf("failed to unmarshal appeal: %v\n", err)
				continue
			}

			if appeal.Id > maxID {
				maxID = appeal.Id
			}

			// status index must exist
			if !store.Has(AppealByStatusKey(uint32(appeal.Status), appeal.Id)) {
				broken = true
				failCount++
				msg += fmt.Sprintf("missing appeal-by-status index for appeal %d\n", appeal.Id)
			}

			// referenced slash record must exist
			if _, err := k.getSlashRecord(ctx, appeal.SlashId); err != nil {
				broken = true
				failCount++
				msg += fmt.Sprintf("appeal %d references missing slash record %d\n", appeal.Id, appeal.SlashId)
			}
		}

		nextID, err := k.getNextAppealIDForExport(ctx)
		if err == nil && nextID <= maxID {
			broken = true
			failCount++
			msg += fmt.Sprintf("next appeal id %d not greater than max id %d\n", nextID, maxID)
		}

		if failCount > 0 && msg == "" {
			msg = "appeal index invariant failed"
		}

		return sdk.FormatInvariant(types.ModuleName, "appeal-index", msg), broken
	}
}
