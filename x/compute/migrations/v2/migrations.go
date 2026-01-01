package v2

import (
	"encoding/binary"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// Key prefixes for v1->v2 migration.
// NOTE: These are intentionally duplicated from keeper/keys.go rather than imported.
// Migrations must use the key prefixes as they existed at the version being migrated FROM.
// If keeper/keys.go changes in future versions, old migrations must still work with
// the original key layout. Importing from keeper would create import cycles and could
// break migrations if keys are ever modified.
// See keeper/keys.go for the canonical current key definitions.
var (
	RequestKeyPrefix         = []byte{0x01, 0x03}
	ProviderKeyPrefix        = []byte{0x01, 0x02}
	ParamsKey                = []byte{0x01, 0x01}
	NextRequestIDKey         = []byte{0x01, 0x05}
	RequestsByProviderPrefix = []byte{0x01, 0x07}
	RequestsByStatusPrefix   = []byte{0x01, 0x08}
	ActiveProvidersPrefix    = []byte{0x01, 0x09}
)

// Escrow key prefixes for migration.
// EscrowStateKeyPrefix and EscrowTimeoutPrefix are legacy keys from v1.
// EscrowTimeoutReversePrefix matches keeper/keys.go (0x01, 0x1D).
var (
	EscrowStateKeyPrefix       = []byte{0x20}
	EscrowTimeoutPrefix        = []byte{0x21}
	EscrowTimeoutReversePrefix = []byte{0x01, 0x1D}
)

// Migrate implements store migrations from v1 to v2 for the compute module.
// This migration performs the following operations:
// 1. Validates existing state consistency
// 2. Rebuilds all secondary indexes
// 3. Migrates request status enum values (if changed)
// 4. Updates params with new fields
// 5. Validates and fixes provider reputation scores
// 6. DATA-7: Rebuilds escrow timeout indexes
func Migrate(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Starting compute module v1 to v2 migration")

	store := ctx.KVStore(storeKey)

	// Step 1: Validate and rebuild request indexes
	if err := rebuildRequestIndexes(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to rebuild request indexes: %w", err)
	}

	// Step 2: Validate and rebuild provider indexes
	if err := rebuildProviderIndexes(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to rebuild provider indexes: %w", err)
	}

	// Step 3: Validate provider reputation scores
	if err := validateProviderReputations(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to validate provider reputations: %w", err)
	}

	// Step 4: Update params with new fields (if any)
	if err := migrateParams(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to migrate params: %w", err)
	}

	// Step 5: Validate request counter consistency
	if err := validateRequestCounter(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to validate request counter: %w", err)
	}

	// Step 6: DATA-7: Rebuild escrow timeout indexes to ensure consistency
	if err := rebuildEscrowTimeoutIndexes(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to rebuild escrow timeout indexes: %w", err)
	}

	ctx.Logger().Info("Compute module v1 to v2 migration completed successfully")
	return nil
}

// rebuildRequestIndexes rebuilds all request secondary indexes
func rebuildRequestIndexes(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Rebuilding request indexes")

	// Clear existing indexes
	clearPrefix(store, RequestsByStatusPrefix)
	clearPrefix(store, RequestsByProviderPrefix)

	// Iterate through all requests and rebuild indexes
	iterator := storetypes.KVStorePrefixIterator(store, RequestKeyPrefix)
	defer iterator.Close()

	count := 0
	for ; iterator.Valid(); iterator.Next() {
		var request types.Request
		if err := cdc.Unmarshal(iterator.Value(), &request); err != nil {
			return fmt.Errorf("failed to unmarshal request: %w", err)
		}

		// Rebuild status index - uses 4 bytes for status (uint32) to match keeper
		statusBz := make([]byte, 4)
		binary.BigEndian.PutUint32(statusBz, uint32(request.Status))
		statusKey := append(append(RequestsByStatusPrefix, statusBz...), getRequestIDBytes(request.Id)...)
		store.Set(statusKey, []byte{})

		// Rebuild provider index
		providerAddr, err := sdk.AccAddressFromBech32(request.Provider)
		if err != nil {
			ctx.Logger().Error("invalid provider address in request", "request_id", request.Id, "provider", request.Provider)
			continue
		}
		providerKey := append(append(RequestsByProviderPrefix, providerAddr.Bytes()...), getRequestIDBytes(request.Id)...)
		store.Set(providerKey, []byte{})

		count++
	}

	ctx.Logger().Info("Request indexes rebuilt", "count", count)
	return nil
}

// rebuildProviderIndexes rebuilds all provider secondary indexes
func rebuildProviderIndexes(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Rebuilding provider indexes")

	// Clear existing active providers index
	clearPrefix(store, ActiveProvidersPrefix)

	// Iterate through all providers and rebuild active index
	iterator := storetypes.KVStorePrefixIterator(store, ProviderKeyPrefix)
	defer iterator.Close()

	activeCount := 0
	for ; iterator.Valid(); iterator.Next() {
		var provider types.Provider
		if err := cdc.Unmarshal(iterator.Value(), &provider); err != nil {
			return fmt.Errorf("failed to unmarshal provider: %w", err)
		}

		if provider.Active {
			providerAddr, err := sdk.AccAddressFromBech32(provider.Address)
			if err != nil {
				ctx.Logger().Error("invalid provider address", "address", provider.Address)
				continue
			}
			activeKey := append(ActiveProvidersPrefix, providerAddr.Bytes()...)
			store.Set(activeKey, providerAddr.Bytes())
			activeCount++
		}
	}

	ctx.Logger().Info("Provider indexes rebuilt", "active_count", activeCount)
	return nil
}

// validateProviderReputations validates and fixes provider reputation scores
func validateProviderReputations(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Validating provider reputations")

	iterator := storetypes.KVStorePrefixIterator(store, ProviderKeyPrefix)
	defer iterator.Close()

	fixed := 0
	for ; iterator.Valid(); iterator.Next() {
		var provider types.Provider
		if err := cdc.Unmarshal(iterator.Value(), &provider); err != nil {
			return fmt.Errorf("failed to unmarshal provider: %w", err)
		}

		needsUpdate := false

		// Ensure reputation is between 0 and 100
		if provider.Reputation == 0 {
			ctx.Logger().Warn("fixing zero reputation", "provider", provider.Address, "old", provider.Reputation)
			provider.Reputation = 1
			needsUpdate = true
		} else if provider.Reputation > 100 {
			ctx.Logger().Warn("fixing reputation above 100", "provider", provider.Address, "old", provider.Reputation)
			provider.Reputation = 100
			needsUpdate = true
		}

		// Recalculate reputation based on success/failure ratio if it seems off
		total := provider.TotalRequestsCompleted + provider.TotalRequestsFailed
		if total > 0 {
			expectedReputation := uint64(provider.TotalRequestsCompleted * 100 / total)
			diff := uint64(0)
			if uint64(provider.Reputation) > expectedReputation {
				diff = uint64(provider.Reputation) - expectedReputation
			} else {
				diff = expectedReputation - uint64(provider.Reputation)
			}

			// If reputation is off by more than 10 points, recalculate
			if diff > 10 {
				ctx.Logger().Warn("recalculating reputation", "provider", provider.Address,
					"old", provider.Reputation, "new", expectedReputation,
					"completed", provider.TotalRequestsCompleted, "failed", provider.TotalRequestsFailed)
				provider.Reputation = types.SaturateUint64ToUint32(expectedReputation)
				needsUpdate = true
			}
		}

		if needsUpdate {
			bz, err := cdc.Marshal(&provider)
			if err != nil {
				return fmt.Errorf("failed to marshal provider: %w", err)
			}
			store.Set(iterator.Key(), bz)
			fixed++
		}
	}

	ctx.Logger().Info("Provider reputations validated", "fixed", fixed)
	return nil
}

// migrateParams updates params with new fields
func migrateParams(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Migrating params")

	bz := store.Get(ParamsKey)
	if bz == nil {
		// No params yet, use defaults
		params := types.DefaultParams()
		newBz, err := cdc.Marshal(&params)
		if err != nil {
			return fmt.Errorf("failed to marshal default params: %w", err)
		}
		store.Set(ParamsKey, newBz)
		ctx.Logger().Info("Initialized default params")
		return nil
	}

	var params types.Params
	if err := cdc.Unmarshal(bz, &params); err != nil {
		return fmt.Errorf("failed to unmarshal params: %w", err)
	}

	// Add new fields with default values if they're zero
	updated := false

	if params.MinProviderStake.IsZero() {
		params.MinProviderStake = math.NewInt(1000000) // 1 PAW
		updated = true
	}

	if params.MaxRequestTimeoutSeconds == 0 {
		params.MaxRequestTimeoutSeconds = 3600 // 1 hour
		updated = true
	}

	if updated {
		newBz, err := cdc.Marshal(&params)
		if err != nil {
			return fmt.Errorf("failed to marshal updated params: %w", err)
		}
		store.Set(ParamsKey, newBz)
		ctx.Logger().Info("Updated params with new fields")
	}

	return nil
}

// validateRequestCounter validates and fixes request counter consistency
func validateRequestCounter(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Validating request counter")

	// Find the highest request ID
	iterator := storetypes.KVStorePrefixIterator(store, RequestKeyPrefix)
	defer iterator.Close()

	maxID := uint64(0)
	for ; iterator.Valid(); iterator.Next() {
		var request types.Request
		if err := cdc.Unmarshal(iterator.Value(), &request); err != nil {
			return fmt.Errorf("failed to unmarshal request: %w", err)
		}

		if request.Id > maxID {
			maxID = request.Id
		}
	}

	// Get current counter
	counterBz := store.Get(NextRequestIDKey)
	currentCounter := uint64(0)
	if counterBz != nil {
		currentCounter = binary.BigEndian.Uint64(counterBz)
	}

	// If counter is less than maxID, update it
	if currentCounter <= maxID {
		newCounter := maxID + 1
		counterBz := make([]byte, 8)
		binary.BigEndian.PutUint64(counterBz, newCounter)
		store.Set(NextRequestIDKey, counterBz)
		ctx.Logger().Info("Updated request counter", "old", currentCounter, "new", newCounter, "max_id", maxID)
	}

	return nil
}

// Helper functions

func clearPrefix(store storetypes.KVStore, prefix []byte) {
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var keys [][]byte
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, iterator.Key())
	}

	for _, key := range keys {
		store.Delete(key)
	}
}

func getRequestIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// rebuildEscrowTimeoutIndexes rebuilds escrow timeout indexes from escrow state.
// DATA-7: This ensures timeout indexes are consistent after migrations that may have
// created escrow states without corresponding timeout indexes.
func rebuildEscrowTimeoutIndexes(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Rebuilding escrow timeout indexes")

	// Clear existing timeout indexes and reverse indexes to rebuild from scratch
	clearPrefix(store, EscrowTimeoutPrefix)
	clearPrefix(store, EscrowTimeoutReversePrefix)

	// Iterate through all escrow states and rebuild timeout indexes for LOCKED and CHALLENGED escrows
	iterator := storetypes.KVStorePrefixIterator(store, EscrowStateKeyPrefix)
	defer iterator.Close()

	count := 0
	for ; iterator.Valid(); iterator.Next() {
		var escrow types.EscrowState
		if err := cdc.Unmarshal(iterator.Value(), &escrow); err != nil {
			return fmt.Errorf("failed to unmarshal escrow state: %w", err)
		}

		// Only LOCKED and CHALLENGED escrows need timeout indexes
		// RELEASED and REFUNDED escrows should NOT have timeout indexes
		if escrow.Status == types.ESCROW_STATUS_LOCKED || escrow.Status == types.ESCROW_STATUS_CHALLENGED {
			// Create timeout index key: prefix + timestamp + requestID
			timeBz := make([]byte, 8)
			binary.BigEndian.PutUint64(timeBz, uint64(escrow.ExpiresAt.Unix()))
			idBz := make([]byte, 8)
			binary.BigEndian.PutUint64(idBz, escrow.RequestId)

			timeoutKey := append(append(EscrowTimeoutPrefix, timeBz...), idBz...)
			store.Set(timeoutKey, []byte{})

			// Create reverse index: prefix + requestID -> timestamp
			reverseKey := append(EscrowTimeoutReversePrefix, idBz...)
			store.Set(reverseKey, timeBz)

			count++
		}
	}

	ctx.Logger().Info("Escrow timeout indexes rebuilt", "count", count)
	return nil
}
