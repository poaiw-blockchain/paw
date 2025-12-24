package keeper

import (
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MigrateStoreKeys migrates all store keys from the old non-namespaced format
// to the new namespaced format (0x03 prefix for Oracle module).
// This should be called during a chain upgrade.
//
// Old format: []byte{0xNN, ...}
// New format: []byte{0x03, 0xNN, ...}
func (k Keeper) MigrateStoreKeys(ctx sdk.Context) error {
	store := ctx.KVStore(k.storeKey)

	// Define old key prefixes (without namespace)
	oldKeyPrefixes := []struct {
		old []byte
		new []byte
	}{
		{[]byte{0x01}, ParamsKey},
		{[]byte{0x02}, PriceKeyPrefix},
		{[]byte{0x03}, ValidatorPriceKeyPrefix},
		{[]byte{0x04}, ValidatorOracleKeyPrefix},
		{[]byte{0x05}, PriceSnapshotKeyPrefix},
		{[]byte{0x06}, FeederDelegationKeyPrefix},
		{[]byte{0x07}, SubmissionByHeightPrefix},
		{[]byte{0x08}, ValidatorAccuracyKeyPrefix},
		{[]byte{0x09}, AccuracyBonusPoolKey},
		{[]byte{0x0A}, GeographicInfoKeyPrefix},
		{[]byte{0x0C}, OutlierHistoryKeyPrefix},
	}

	// Migrate each prefix
	for _, p := range oldKeyPrefixes {
		if err := k.migratePrefix(store, p.old, p.new); err != nil {
			return fmt.Errorf("failed to migrate prefix %x: %w", p.old, err)
		}
	}

	return nil
}

// migratePrefix migrates all keys with a given prefix to a new prefix
func (k Keeper) migratePrefix(store storetypes.KVStore, oldPrefix, newPrefix []byte) error {
	oldStore := prefix.NewStore(store, oldPrefix)

	// Collect all keys to migrate (can't modify while iterating)
	type migration struct {
		oldKey []byte
		value  []byte
	}
	var migrations []migration

	iter := oldStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		migrations = append(migrations, migration{
			oldKey: iter.Key(),
			value:  iter.Value(),
		})
	}

	// Perform migrations
	for _, m := range migrations {
		// Construct new key: newPrefix + oldKey (without old prefix)
		newKey := append(newPrefix, m.oldKey...)

		// Write to new key
		store.Set(newKey, m.value)

		// Delete old key
		oldFullKey := append(oldPrefix, m.oldKey...)
		store.Delete(oldFullKey)
	}

	return nil
}

// GetOldKey converts a new namespaced key back to the old format
// Useful for backwards compatibility reads during migration period
func GetOldKey(namespacedKey []byte) []byte {
	if len(namespacedKey) < 2 {
		return namespacedKey
	}
	// Strip the module namespace byte (first byte)
	if namespacedKey[0] == ModuleNamespace {
		return namespacedKey[1:]
	}
	return namespacedKey
}

// GetNewKey adds the module namespace to an old key
// Useful for forwards compatibility during migration period
func GetNewKey(oldKey []byte) []byte {
	if len(oldKey) == 0 {
		return oldKey
	}
	// Add module namespace if not already present
	if oldKey[0] != ModuleNamespace {
		return append([]byte{ModuleNamespace}, oldKey...)
	}
	return oldKey
}
