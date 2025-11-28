package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/dex/types"
)

// Task 174 & 175: State Corruption Recovery and Backup/Restore

// BackupData represents a complete backup of DEX state
type BackupData struct {
	Version          string
	Timestamp        time.Time
	BlockHeight      int64
	Pools            []types.Pool
	Params           types.Params
	LiquidityShares  map[string]map[uint64]string // provider -> poolID -> shares
	ProtocolFees     map[string]string            // token -> amount
	PoolLPFees       map[uint64]map[string]string // poolID -> token -> amount
	CircuitBreakers  map[uint64]BackupCircuitBreakerState
	NextPoolID       uint64
	Checksum         string
}

// BackupCircuitBreakerState represents circuit breaker state for backup
type BackupCircuitBreakerState struct {
	Active      bool
	TriggeredAt int64
	Reason      string
}

// ExportState exports the complete DEX state for backup
func (k Keeper) ExportState(ctx context.Context) (*BackupData, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	backup := &BackupData{
		Version:         "1.0.0",
		Timestamp:       sdkCtx.BlockTime(),
		BlockHeight:     sdkCtx.BlockHeight(),
		LiquidityShares: make(map[string]map[uint64]string),
		ProtocolFees:    make(map[string]string),
		PoolLPFees:      make(map[uint64]map[string]string),
		CircuitBreakers: make(map[uint64]BackupCircuitBreakerState),
	}

	// Export params
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get params: %w", err)
	}
	backup.Params = params

	// Export next pool ID
	nextIDKey := PoolCountKey
	if bz := store.Get(nextIDKey); bz != nil {
		backup.NextPoolID = sdk.BigEndianToUint64(bz)
	}

	// Export all pools
	poolIterator := storetypes.KVStorePrefixIterator(store, PoolKeyPrefix)
	defer poolIterator.Close()

	for ; poolIterator.Valid(); poolIterator.Next() {
		var pool types.Pool
		if err := k.cdc.Unmarshal(poolIterator.Value(), &pool); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pool: %w", err)
		}
		backup.Pools = append(backup.Pools, pool)
	}

	// Export liquidity shares
	liquidityIterator := storetypes.KVStorePrefixIterator(store, LiquidityShareKeyPrefix)
	defer liquidityIterator.Close()

	for ; liquidityIterator.Valid(); liquidityIterator.Next() {
		// Parse key to extract pool ID and provider
		key := liquidityIterator.Key()
		if len(key) < 9 { // prefix(1) + poolID(8)
			continue
		}

		poolID := sdk.BigEndianToUint64(key[1:9])
		provider := sdk.AccAddress(key[9:]).String()

		shares := string(liquidityIterator.Value())

		if backup.LiquidityShares[provider] == nil {
			backup.LiquidityShares[provider] = make(map[uint64]string)
		}
		backup.LiquidityShares[provider][poolID] = shares
	}

	// Export protocol fees
	protocolFeeIterator := storetypes.KVStorePrefixIterator(store, ProtocolFeeKeyPrefix)
	defer protocolFeeIterator.Close()

	for ; protocolFeeIterator.Valid(); protocolFeeIterator.Next() {
		token := string(protocolFeeIterator.Key()[len(ProtocolFeeKeyPrefix):])
		amount := string(protocolFeeIterator.Value())
		backup.ProtocolFees[token] = amount
	}

	// Export LP fees
	lpFeeIterator := storetypes.KVStorePrefixIterator(store, PoolLPFeeKeyPrefix)
	defer lpFeeIterator.Close()

	for ; lpFeeIterator.Valid(); lpFeeIterator.Next() {
		key := lpFeeIterator.Key()
		if len(key) < 9 {
			continue
		}

		poolID := sdk.BigEndianToUint64(key[1:9])
		token := string(key[9:])
		amount := string(lpFeeIterator.Value())

		if backup.PoolLPFees[poolID] == nil {
			backup.PoolLPFees[poolID] = make(map[string]string)
		}
		backup.PoolLPFees[poolID][token] = amount
	}

	// Calculate checksum
	backup.Checksum = calculateBackupChecksum(backup)

	return backup, nil
}

// ImportState imports a backup to restore DEX state
func (k Keeper) ImportState(ctx context.Context, backup *BackupData) error {
	// Verify checksum
	expectedChecksum := calculateBackupChecksum(backup)
	if backup.Checksum != expectedChecksum {
		return fmt.Errorf("backup checksum mismatch: expected %s, got %s", expectedChecksum, backup.Checksum)
	}

	store := k.getStore(ctx)

	// Clear existing state (if restoring)
	k.clearState(ctx)

	// Import params
	if err := k.SetParams(ctx, backup.Params); err != nil {
		return fmt.Errorf("failed to set params: %w", err)
	}

	// Import next pool ID
	store.Set(PoolCountKey, sdk.Uint64ToBigEndian(backup.NextPoolID))

	// Import pools
	for _, pool := range backup.Pools {
		if err := k.SetPool(ctx, &pool); err != nil {
			return fmt.Errorf("failed to set pool %d: %w", pool.Id, err)
		}
	}

	// Import liquidity shares
	for provider, pools := range backup.LiquidityShares {
		providerAddr, err := sdk.AccAddressFromBech32(provider)
		if err != nil {
			return fmt.Errorf("invalid provider address %s: %w", provider, err)
		}

		for poolID, shares := range pools {
			key := LiquidityShareKey(poolID, providerAddr)
			store.Set(key, []byte(shares))
		}
	}

	// Import protocol fees
	for token, amount := range backup.ProtocolFees {
		key := ProtocolFeeKey(token)
		store.Set(key, []byte(amount))
	}

	// Import LP fees
	for poolID, tokens := range backup.PoolLPFees {
		for token, amount := range tokens {
			key := PoolLPFeeKey(poolID, token)
			store.Set(key, []byte(amount))
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dex_state_imported",
			sdk.NewAttribute("pools_count", fmt.Sprintf("%d", len(backup.Pools))),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", backup.BlockHeight)),
			sdk.NewAttribute("timestamp", backup.Timestamp.String()),
		),
	)

	return nil
}

// SaveBackup saves backup data to a file
func (backup *BackupData) SaveToFile(filepath string) error {
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// LoadBackup loads backup data from a file
func LoadBackupFromFile(filepath string) (*BackupData, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	var backup BackupData
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup: %w", err)
	}

	// Verify checksum
	expectedChecksum := calculateBackupChecksum(&backup)
	if backup.Checksum != expectedChecksum {
		return nil, fmt.Errorf("backup file corrupted: checksum mismatch")
	}

	return &backup, nil
}

// calculateBackupChecksum calculates a checksum for backup data
func calculateBackupChecksum(backup *BackupData) string {
	// Create a copy without checksum for hashing
	temp := *backup
	temp.Checksum = ""

	data, _ := json.Marshal(temp)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// clearState clears all DEX state (use with caution!)
func (k Keeper) clearState(ctx context.Context) {
	store := k.getStore(ctx)

	// Clear pools
	clearPrefix(store, PoolKeyPrefix)

	// Clear liquidity shares
	clearPrefix(store, LiquidityShareKeyPrefix)

	// Clear protocol fees
	clearPrefix(store, ProtocolFeeKeyPrefix)

	// Clear LP fees
	clearPrefix(store, PoolLPFeeKeyPrefix)

	// Clear circuit breakers
	clearPrefix(store, CircuitBreakerKeyPrefix)
}

// clearPrefix removes all keys with a given prefix
func clearPrefix(store storetypes.KVStore, prefix []byte) {
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	keys := [][]byte{}
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, iterator.Key())
	}

	for _, key := range keys {
		store.Delete(key)
	}
}

// ValidateState performs comprehensive state validation
func (k Keeper) ValidateState(ctx context.Context) error {
	store := k.getStore(ctx)

	// Validate all pools
	poolIterator := storetypes.KVStorePrefixIterator(store, PoolKeyPrefix)
	defer poolIterator.Close()

	for ; poolIterator.Valid(); poolIterator.Next() {
		var pool types.Pool
		if err := k.cdc.Unmarshal(poolIterator.Value(), &pool); err != nil {
			return fmt.Errorf("corrupt pool data at key %x: %w", poolIterator.Key(), err)
		}

		// Validate pool invariants
		if err := k.ValidatePoolState(&pool); err != nil {
			return fmt.Errorf("invalid pool state for pool %d: %w", pool.Id, err)
		}

		// Verify constant product invariant
		k_value := pool.ReserveA.Mul(pool.ReserveB)
		if k_value.IsZero() {
			return fmt.Errorf("pool %d has zero k-value", pool.Id)
		}
	}

	// Validate liquidity shares sum up correctly
	sharesByPool := make(map[uint64]math.Int)
	liquidityIterator := storetypes.KVStorePrefixIterator(store, LiquidityShareKeyPrefix)
	defer liquidityIterator.Close()

	for ; liquidityIterator.Valid(); liquidityIterator.Next() {
		key := liquidityIterator.Key()
		if len(key) < 9 {
			return fmt.Errorf("invalid liquidity share key: %x", key)
		}

		poolID := sdk.BigEndianToUint64(key[1:9])

		var shares math.Int
		if err := shares.Unmarshal(liquidityIterator.Value()); err != nil {
			return fmt.Errorf("corrupt liquidity share data: %w", err)
		}

		if !sharesByPool[poolID].IsZero() {
			sharesByPool[poolID] = sharesByPool[poolID].Add(shares)
		} else {
			sharesByPool[poolID] = shares
		}
	}

	// Verify total shares match
	for poolID, totalShares := range sharesByPool {
		pool, err := k.GetPool(ctx, poolID)
		if err != nil {
			return fmt.Errorf("pool %d referenced in liquidity but not found", poolID)
		}

		if !pool.TotalShares.Equal(totalShares) {
			return fmt.Errorf("pool %d total shares mismatch: pool has %s, sum of shares is %s",
				poolID, pool.TotalShares, totalShares)
		}
	}

	return nil
}

// RecoverFromCorruption attempts to recover from state corruption
func (k Keeper) RecoverFromCorruption(ctx context.Context, backupPath string) error {
	// Load backup
	backup, err := LoadBackupFromFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to load backup: %w", err)
	}

	// Validate current state
	if err := k.ValidateState(ctx); err == nil {
		return fmt.Errorf("state is valid, recovery not needed")
	}

	// Import backup state
	if err := k.ImportState(ctx, backup); err != nil {
		return fmt.Errorf("failed to import backup: %w", err)
	}

	// Validate recovered state
	if err := k.ValidateState(ctx); err != nil {
		return fmt.Errorf("recovered state is still invalid: %w", err)
	}

	return nil
}

// CreateCheckpoint creates a state checkpoint
func (k Keeper) CreateCheckpoint(ctx context.Context, name string) error {
	backup, err := k.ExportState(ctx)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	filename := fmt.Sprintf("dex_checkpoint_%s_%d.json", name, sdkCtx.BlockHeight())

	return backup.SaveToFile(filename)
}

// ListCheckpoints lists available checkpoints
func ListCheckpoints(directory string) ([]string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	checkpoints := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > 15 && entry.Name()[:15] == "dex_checkpoint_" {
			checkpoints = append(checkpoints, entry.Name())
		}
	}

	return checkpoints, nil
}
