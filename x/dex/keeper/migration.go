package keeper

import (
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MigrateStoreKeys migrates all store keys from the old non-namespaced format
// to the new namespaced format (0x02 prefix for DEX module).
// This should be called during a chain upgrade.
//
// Old format: []byte{0xNN, ...}
// New format: []byte{0x02, 0xNN, ...}
func (k Keeper) MigrateStoreKeys(ctx sdk.Context) error {
	store := ctx.KVStore(k.storeKey)

	// Define old key prefixes (without namespace)
	oldKeyPrefixes := []struct {
		old []byte
		new []byte
	}{
		{[]byte{0x01}, PoolKeyPrefix},
		{[]byte{0x02}, PoolCountKey},
		{[]byte{0x03}, PoolByTokensKeyPrefix},
		{[]byte{0x04}, LiquidityKeyPrefix},
		{[]byte{0x05}, ParamsKey},
		{[]byte{0x06}, CircuitBreakerKeyPrefix},
		{[]byte{0x07}, LastLiquidityActionKeyPrefix},
		{[]byte{0x08}, ReentrancyLockKeyPrefix},
		{[]byte{0x09}, PoolLPFeeKeyPrefix},
		{[]byte{0x0A}, ProtocolFeeKeyPrefix},
		{[]byte{0x0B}, LiquidityShareKeyPrefix},
		{[]byte{0x0C}, RateLimitKeyPrefix},
		{[]byte{0x0D}, RateLimitByHeightPrefix},
		{[]byte{0x0E}, PoolTWAPKeyPrefix},
		{[]byte{0x15}, ActivePoolsKeyPrefix},
		// Limit order keys (old single-byte to new namespaced)
		{[]byte{0x0E}, LimitOrderKeyPrefix},      // Note: 0x0E also used for TWAP, limit orders were added later
		{[]byte{0x0F}, LimitOrderCountKey},
		{[]byte{0x10}, LimitOrderByOwnerPrefix},
		{[]byte{0x11}, LimitOrderByPoolPrefix},
		{[]byte{0x12}, LimitOrderByPricePrefix},
		{[]byte{0x13}, LimitOrderOpenPrefix},
		// Commit-reveal keys (old single-byte to new namespaced)
		{[]byte{0x16}, SwapCommitmentKeyPrefix},
		{[]byte{0x17}, SwapCommitmentByExpiryPrefix},
		{[]byte{0x18}, SwapCommitmentByTraderPrefix},
		{[]byte{0x19}, SwapCommitKeyPrefix}, // Governance commit-reveal (unified with SwapCommitmentKeyPrefix)
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
