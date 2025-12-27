package v2

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

var (
	// Key prefixes - MUST match current namespaced prefixes in keeper/keys.go
	// All DEX module keys use 0x02 namespace prefix
	PoolKeyPrefix             = []byte{0x02, 0x01}
	PoolCounterKey            = []byte{0x02, 0x02}
	ParamsKey                 = []byte{0x02, 0x05}
	LiquidityKeyPrefix        = []byte{0x02, 0x04}
	PoolByTokensKeyPrefix     = []byte{0x02, 0x03}
	CircuitBreakerKeyPrefix   = []byte{0x02, 0x06}
	LastLiquidityActionPrefix = []byte{0x02, 0x07}
)

type circuitBreakerState struct {
	Enabled           bool           `json:"enabled"`
	PausedUntil       time.Time      `json:"paused_until"`
	LastPrice         math.LegacyDec `json:"last_price"`
	TriggeredBy       string         `json:"triggered_by"`
	TriggerReason     string         `json:"trigger_reason"`
	NotificationsSent int            `json:"notifications_sent"`
	LastNotification  time.Time      `json:"last_notification"`
	PersistenceKey    string         `json:"persistence_key"`
}

// Migrate implements store migrations from v1 to v2 for the DEX module.
// This migration performs the following operations:
// 1. Validates existing pool state and fixes inconsistencies
// 2. Rebuilds pool indexes
// 3. Validates liquidity provider positions
// 4. Initializes circuit breaker states for existing pools
// 5. Updates params with new fields
// 6. Validates pool counter consistency
func Migrate(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Starting DEX module v1 to v2 migration")

	store := ctx.KVStore(storeKey)

	// Step 1: Validate and rebuild pool indexes
	if err := rebuildPoolIndexes(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to rebuild pool indexes: %w", err)
	}

	// Step 2: Validate pool states and fix reserve inconsistencies
	if err := validatePoolStates(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to validate pool states: %w", err)
	}

	// Step 3: Validate liquidity positions
	if err := validateLiquidityPositions(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to validate liquidity positions: %w", err)
	}

	// Step 4: Initialize circuit breaker states
	if err := initializeCircuitBreakers(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to initialize circuit breakers: %w", err)
	}

	// Step 5: Update params with new fields
	if err := migrateParams(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to migrate params: %w", err)
	}

	// Step 6: Validate pool counter consistency
	if err := validatePoolCounter(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to validate pool counter: %w", err)
	}

	ctx.Logger().Info("DEX module v1 to v2 migration completed successfully")
	return nil
}

// rebuildPoolIndexes rebuilds all pool secondary indexes
func rebuildPoolIndexes(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Rebuilding pool indexes")

	// Clear existing token pair indexes
	clearPrefix(store, PoolByTokensKeyPrefix)

	// Iterate through all pools and rebuild indexes
	iterator := storetypes.KVStorePrefixIterator(store, PoolKeyPrefix)
	defer iterator.Close()

	count := 0
	for ; iterator.Valid(); iterator.Next() {
		var pool types.Pool
		if err := cdc.Unmarshal(iterator.Value(), &pool); err != nil {
			return fmt.Errorf("failed to unmarshal pool: %w", err)
		}

		// Rebuild token pair index
		tokenPairKey := getPoolByTokensKey(pool.TokenA, pool.TokenB)
		poolIDBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(poolIDBytes, pool.Id)
		store.Set(tokenPairKey, poolIDBytes)

		count++
	}

	ctx.Logger().Info("Pool indexes rebuilt", "count", count)
	return nil
}

// validatePoolStates validates and fixes pool state inconsistencies
func validatePoolStates(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Validating pool states")

	iterator := storetypes.KVStorePrefixIterator(store, PoolKeyPrefix)
	defer iterator.Close()

	fixed := 0
	for ; iterator.Valid(); iterator.Next() {
		var pool types.Pool
		if err := cdc.Unmarshal(iterator.Value(), &pool); err != nil {
			return fmt.Errorf("failed to unmarshal pool: %w", err)
		}

		needsUpdate := false

		// Ensure reserves are non-negative
		if pool.ReserveA.IsNegative() {
			ctx.Logger().Warn("fixing negative reserve A", "pool_id", pool.Id, "old", pool.ReserveA)
			pool.ReserveA = math.ZeroInt()
			needsUpdate = true
		}

		if pool.ReserveB.IsNegative() {
			ctx.Logger().Warn("fixing negative reserve B", "pool_id", pool.Id, "old", pool.ReserveB)
			pool.ReserveB = math.ZeroInt()
			needsUpdate = true
		}

		// Ensure total shares are non-negative
		if pool.TotalShares.IsNegative() {
			ctx.Logger().Warn("fixing negative total shares", "pool_id", pool.Id, "old", pool.TotalShares)
			pool.TotalShares = math.ZeroInt()
			needsUpdate = true
		}

		// Validate token ordering (should be lexicographic)
		if pool.TokenA > pool.TokenB {
			ctx.Logger().Warn("fixing token ordering", "pool_id", pool.Id,
				"old_a", pool.TokenA, "old_b", pool.TokenB)
			pool.TokenA, pool.TokenB = pool.TokenB, pool.TokenA
			pool.ReserveA, pool.ReserveB = pool.ReserveB, pool.ReserveA
			needsUpdate = true
		}

		if needsUpdate {
			bz, err := cdc.Marshal(&pool)
			if err != nil {
				return fmt.Errorf("failed to marshal pool: %w", err)
			}
			store.Set(iterator.Key(), bz)
			fixed++
		}
	}

	ctx.Logger().Info("Pool states validated", "fixed", fixed)
	return nil
}

// validateLiquidityPositions validates liquidity provider positions
func validateLiquidityPositions(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Validating liquidity positions")

	// Build a map of total shares per pool
	poolTotalShares := make(map[uint64]math.Int)

	iterator := storetypes.KVStorePrefixIterator(store, LiquidityKeyPrefix)
	defer iterator.Close()

	fixed := 0
	for ; iterator.Valid(); iterator.Next() {
		// Decode pool ID and address from key
		key := iterator.Key()
		if len(key) < 9 { // prefix + 8-byte pool ID
			continue
		}

		poolID := binary.BigEndian.Uint64(key[1:9])

		// Get liquidity amount
		var shares math.Int
		if err := shares.Unmarshal(iterator.Value()); err != nil {
			ctx.Logger().Error("failed to unmarshal liquidity shares", "pool_id", poolID)
			continue
		}

		// Fix negative shares
		if shares.IsNegative() {
			ctx.Logger().Warn("fixing negative liquidity shares", "pool_id", poolID)
			shares = math.ZeroInt()
			bz, err := shares.Marshal()
			if err != nil {
				return err
			}
			store.Set(key, bz)
			fixed++
		}

		// Accumulate total shares
		if existing, ok := poolTotalShares[poolID]; ok {
			poolTotalShares[poolID] = existing.Add(shares)
		} else {
			poolTotalShares[poolID] = shares
		}
	}

	// Validate that sum of LP shares matches pool total shares
	poolIterator := storetypes.KVStorePrefixIterator(store, PoolKeyPrefix)
	defer poolIterator.Close()

	for ; poolIterator.Valid(); poolIterator.Next() {
		var pool types.Pool
		if err := cdc.Unmarshal(poolIterator.Value(), &pool); err != nil {
			continue
		}

		lpTotal, exists := poolTotalShares[pool.Id]
		if !exists {
			lpTotal = math.ZeroInt()
		}

		// Allow small discrepancies due to rounding
		diff := pool.TotalShares.Sub(lpTotal).Abs()
		maxDiff := math.NewInt(100)

		if diff.GT(maxDiff) {
			ctx.Logger().Warn("liquidity position mismatch",
				"pool_id", pool.Id,
				"pool_total_shares", pool.TotalShares,
				"lp_total_shares", lpTotal,
				"difference", diff)
		}
	}

	ctx.Logger().Info("Liquidity positions validated", "fixed", fixed)
	return nil
}

// initializeCircuitBreakers initializes circuit breaker states for all pools
func initializeCircuitBreakers(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Initializing circuit breakers")

	iterator := storetypes.KVStorePrefixIterator(store, PoolKeyPrefix)
	defer iterator.Close()

	count := 0
	for ; iterator.Valid(); iterator.Next() {
		var pool types.Pool
		if err := cdc.Unmarshal(iterator.Value(), &pool); err != nil {
			return fmt.Errorf("failed to unmarshal pool: %w", err)
		}

		// Check if circuit breaker already exists
		cbKey := getCircuitBreakerKey(pool.Id)
		if store.Has(cbKey) {
			continue
		}

		state := circuitBreakerState{
			Enabled:        false,
			PausedUntil:    time.Time{},
			LastPrice:      math.LegacyZeroDec(),
			TriggeredBy:    "migration_init",
			TriggerReason:  "",
			PersistenceKey: fmt.Sprintf("pool_%d", pool.Id),
		}
		if !pool.ReserveA.IsZero() && !pool.ReserveB.IsZero() {
			state.LastPrice = math.LegacyNewDecFromInt(pool.ReserveB).Quo(math.LegacyNewDecFromInt(pool.ReserveA))
		}
		cbData, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("failed to marshal circuit breaker state for pool %d: %w", pool.Id, err)
		}
		store.Set(cbKey, cbData)

		count++
	}

	ctx.Logger().Info("Circuit breakers initialized", "count", count)
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

	if params.SwapFee.IsNil() || params.SwapFee.IsZero() {
		params.SwapFee = math.LegacyMustNewDecFromStr("0.003") // 0.3%
		updated = true
	}

	if params.MinLiquidity.IsZero() {
		params.MinLiquidity = math.NewInt(1000)
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

// validatePoolCounter validates and fixes pool counter consistency
func validatePoolCounter(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Validating pool counter")

	// Find the highest pool ID
	iterator := storetypes.KVStorePrefixIterator(store, PoolKeyPrefix)
	defer iterator.Close()

	maxID := uint64(0)
	for ; iterator.Valid(); iterator.Next() {
		var pool types.Pool
		if err := cdc.Unmarshal(iterator.Value(), &pool); err != nil {
			return fmt.Errorf("failed to unmarshal pool: %w", err)
		}

		if pool.Id > maxID {
			maxID = pool.Id
		}
	}

	// Get current counter
	counterBz := store.Get(PoolCounterKey)
	currentCounter := uint64(0)
	if counterBz != nil {
		currentCounter = binary.BigEndian.Uint64(counterBz)
	}

	// If counter is less than or equal to maxID, update it
	if currentCounter <= maxID {
		newCounter := maxID + 1
		counterBz := make([]byte, 8)
		binary.BigEndian.PutUint64(counterBz, newCounter)
		store.Set(PoolCounterKey, counterBz)
		ctx.Logger().Info("Updated pool counter", "old", currentCounter, "new", newCounter, "max_id", maxID)
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

func getPoolByTokensKey(tokenA, tokenB string) []byte {
	// Ensure consistent ordering
	if tokenA > tokenB {
		tokenA, tokenB = tokenB, tokenA
	}
	key := append(PoolByTokensKeyPrefix, []byte(tokenA)...)
	key = append(key, []byte("/")...)
	key = append(key, []byte(tokenB)...)
	return key
}

func getCircuitBreakerKey(poolID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, poolID)
	return append(CircuitBreakerKeyPrefix, bz...)
}
