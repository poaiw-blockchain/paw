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
	"github.com/paw-chain/paw/x/compute/types"
)

// StateBackupData represents a complete backup of compute module state
type StateBackupData struct {
	Version       string
	Timestamp     time.Time
	BlockHeight   int64
	Params        types.Params
	Governance    types.GovernanceParams
	Providers     []types.Provider
	Requests      []types.Request
	Results       []types.Result
	Disputes      []types.Dispute
	SlashRecords  []types.SlashRecord
	Appeals       []types.Appeal
	Escrows       map[string]string // request_id -> amount
	Nonces        map[string]uint64 // provider -> nonce
	NextRequestID uint64
	NextDisputeID uint64
	NextSlashID   uint64
	NextAppealID  uint64
	Checksum      string
}

// ExportState exports the complete compute module state for backup
func (k Keeper) ExportState(ctx context.Context) (*StateBackupData, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	backup := &StateBackupData{
		Version:     "1.0.0",
		Timestamp:   sdkCtx.BlockTime(),
		BlockHeight: sdkCtx.BlockHeight(),
		Escrows:     make(map[string]string),
		Nonces:      make(map[string]uint64),
	}

	// Export params
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get params: %w", err)
	}
	backup.Params = params

	// Export governance params
	govParams, err := k.GetGovernanceParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get governance params: %w", err)
	}
	backup.Governance = govParams

	// Export providers
	providerIterator := storetypes.KVStorePrefixIterator(store, types.ProviderKeyPrefix)
	defer providerIterator.Close()

	for ; providerIterator.Valid(); providerIterator.Next() {
		var provider types.Provider
		if err := k.cdc.Unmarshal(providerIterator.Value(), &provider); err != nil {
			return nil, fmt.Errorf("failed to unmarshal provider: %w", err)
		}
		backup.Providers = append(backup.Providers, provider)
	}

	// Export requests
	requestIterator := storetypes.KVStorePrefixIterator(store, types.RequestKeyPrefix)
	defer requestIterator.Close()

	for ; requestIterator.Valid(); requestIterator.Next() {
		var request types.Request
		if err := k.cdc.Unmarshal(requestIterator.Value(), &request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request: %w", err)
		}
		backup.Requests = append(backup.Requests, request)
	}

	// Export results
	resultIterator := storetypes.KVStorePrefixIterator(store, types.ResultKeyPrefix)
	defer resultIterator.Close()

	for ; resultIterator.Valid(); resultIterator.Next() {
		var result types.Result
		if err := k.cdc.Unmarshal(resultIterator.Value(), &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result: %w", err)
		}
		backup.Results = append(backup.Results, result)
	}

	// Export disputes
	disputeIterator := storetypes.KVStorePrefixIterator(store, DisputeKeyPrefix)
	defer disputeIterator.Close()
	for ; disputeIterator.Valid(); disputeIterator.Next() {
		var dispute types.Dispute
		if err := k.cdc.Unmarshal(disputeIterator.Value(), &dispute); err != nil {
			return nil, fmt.Errorf("failed to unmarshal dispute: %w", err)
		}
		backup.Disputes = append(backup.Disputes, dispute)
	}

	// Export slash records
	slashIterator := storetypes.KVStorePrefixIterator(store, SlashRecordKeyPrefix)
	defer slashIterator.Close()
	for ; slashIterator.Valid(); slashIterator.Next() {
		var record types.SlashRecord
		if err := k.cdc.Unmarshal(slashIterator.Value(), &record); err != nil {
			return nil, fmt.Errorf("failed to unmarshal slash record: %w", err)
		}
		backup.SlashRecords = append(backup.SlashRecords, record)
	}

	// Export appeals
	appealIterator := storetypes.KVStorePrefixIterator(store, AppealKeyPrefix)
	defer appealIterator.Close()
	for ; appealIterator.Valid(); appealIterator.Next() {
		var appeal types.Appeal
		if err := k.cdc.Unmarshal(appealIterator.Value(), &appeal); err != nil {
			return nil, fmt.Errorf("failed to unmarshal appeal: %w", err)
		}
		backup.Appeals = append(backup.Appeals, appeal)
	}

	// Export escrows
	escrowIterator := storetypes.KVStorePrefixIterator(store, types.EscrowKeyPrefix)
	defer escrowIterator.Close()

	for ; escrowIterator.Valid(); escrowIterator.Next() {
		requestID := string(escrowIterator.Key()[len(types.EscrowKeyPrefix):])
		amount := string(escrowIterator.Value())
		backup.Escrows[requestID] = amount
	}

	// Export nonces
	nonceIterator := storetypes.KVStorePrefixIterator(store, types.NonceKeyPrefix)
	defer nonceIterator.Close()

	for ; nonceIterator.Valid(); nonceIterator.Next() {
		provider := string(nonceIterator.Key()[len(types.NonceKeyPrefix):])
		nonce := sdk.BigEndianToUint64(nonceIterator.Value())
		backup.Nonces[provider] = nonce
	}

	if nextRequestID, err := k.getNextRequestIDForExport(ctx); err == nil {
		backup.NextRequestID = nextRequestID
	}
	if nextDisputeID, err := k.getNextDisputeIDForExport(ctx); err == nil {
		backup.NextDisputeID = nextDisputeID
	}
	if nextSlashID, err := k.getNextSlashIDForExport(ctx); err == nil {
		backup.NextSlashID = nextSlashID
	}
	if nextAppealID, err := k.getNextAppealIDForExport(ctx); err == nil {
		backup.NextAppealID = nextAppealID
	}

	// Calculate checksum
	backup.Checksum = calculateComputeChecksum(backup)

	return backup, nil
}

// ImportState imports a backup to restore compute module state
func (k Keeper) ImportState(ctx context.Context, backup *StateBackupData) error {
	// Verify checksum
	expectedChecksum := calculateComputeChecksum(backup)
	if backup.Checksum != expectedChecksum {
		return fmt.Errorf("backup checksum mismatch: expected %s, got %s", expectedChecksum, backup.Checksum)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	// Clear existing state
	k.clearComputeState(ctx)

	// Import params
	if err := k.SetParams(ctx, backup.Params); err != nil {
		return fmt.Errorf("failed to set params: %w", err)
	}

	// Import governance params with defaults if missing
	govParams := backup.Governance
	if govParams.DisputeDeposit.IsNil() || !govParams.DisputeDeposit.IsPositive() {
		govParams = types.DefaultGovernanceParams()
	}
	if err := k.SetGovernanceParams(ctx, govParams); err != nil {
		return fmt.Errorf("failed to set governance params: %w", err)
	}

	var (
		maxRequestID uint64
		maxDisputeID uint64
		maxSlashID   uint64
		maxAppealID  uint64
	)

	// Import providers
	for _, provider := range backup.Providers {
		if err := k.SetProvider(ctx, provider); err != nil {
			return fmt.Errorf("failed to set provider %s: %w", provider.Address, err)
		}

		if provider.Active {
			addr, err := sdk.AccAddressFromBech32(provider.Address)
			if err == nil {
				if err := k.setActiveProviderIndex(ctx, addr, true); err != nil {
					return fmt.Errorf("failed to index active provider %s: %w", provider.Address, err)
				}
			}
		}
	}

	// Import requests
	for _, request := range backup.Requests {
		if err := k.SetRequest(ctx, request); err != nil {
			return fmt.Errorf("failed to set request %d: %w", request.Id, err)
		}
		if request.Id > maxRequestID {
			maxRequestID = request.Id
		}
		if err := k.setRequestIndexes(ctx, request); err != nil {
			return fmt.Errorf("failed to set request indexes for %d: %w", request.Id, err)
		}
	}

	// Import results
	for _, result := range backup.Results {
		if err := k.SetResult(ctx, &result); err != nil {
			return fmt.Errorf("failed to set result for request %d: %w", result.RequestId, err)
		}
	}

	// Import disputes
	for _, dispute := range backup.Disputes {
		if dispute.Id > maxDisputeID {
			maxDisputeID = dispute.Id
		}
		if err := k.setDispute(ctx, dispute); err != nil {
			return fmt.Errorf("failed to set dispute %d: %w", dispute.Id, err)
		}
	}

	// Import slash records
	for _, record := range backup.SlashRecords {
		if record.Id > maxSlashID {
			maxSlashID = record.Id
		}
		if err := k.setSlashRecord(ctx, record); err != nil {
			return fmt.Errorf("failed to set slash record %d: %w", record.Id, err)
		}
	}

	// Import appeals
	for _, appeal := range backup.Appeals {
		if appeal.Id > maxAppealID {
			maxAppealID = appeal.Id
		}
		if err := k.setAppeal(ctx, appeal); err != nil {
			return fmt.Errorf("failed to set appeal %d: %w", appeal.Id, err)
		}
	}

	// Import escrows
	for requestID, amount := range backup.Escrows {
		key := append(types.EscrowKeyPrefix, []byte(requestID)...)
		store.Set(key, []byte(amount))
	}

	// Import nonces
	for provider, nonce := range backup.Nonces {
		key := append(types.NonceKeyPrefix, []byte(provider)...)
		store.Set(key, sdk.Uint64ToBigEndian(nonce))
	}

	nextRequestID := backup.NextRequestID
	if nextRequestID == 0 || nextRequestID <= maxRequestID {
		nextRequestID = maxRequestID + 1
	}
	if err := k.setNextRequestID(ctx, nextRequestID); err != nil {
		return fmt.Errorf("failed to set next request ID: %w", err)
	}

	nextDisputeID := backup.NextDisputeID
	if nextDisputeID == 0 || nextDisputeID <= maxDisputeID {
		nextDisputeID = maxDisputeID + 1
	}
	if err := k.setNextDisputeID(ctx, nextDisputeID); err != nil {
		return fmt.Errorf("failed to set next dispute ID: %w", err)
	}

	nextSlashID := backup.NextSlashID
	if nextSlashID == 0 || nextSlashID <= maxSlashID {
		nextSlashID = maxSlashID + 1
	}
	if err := k.setNextSlashID(ctx, nextSlashID); err != nil {
		return fmt.Errorf("failed to set next slash ID: %w", err)
	}

	nextAppealID := backup.NextAppealID
	if nextAppealID == 0 || nextAppealID <= maxAppealID {
		nextAppealID = maxAppealID + 1
	}
	if err := k.setNextAppealID(ctx, nextAppealID); err != nil {
		return fmt.Errorf("failed to set next appeal ID: %w", err)
	}

	sdkCtx = sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_state_imported",
			sdk.NewAttribute("providers_count", fmt.Sprintf("%d", len(backup.Providers))),
			sdk.NewAttribute("requests_count", fmt.Sprintf("%d", len(backup.Requests))),
			sdk.NewAttribute("disputes_count", fmt.Sprintf("%d", len(backup.Disputes))),
			sdk.NewAttribute("appeals_count", fmt.Sprintf("%d", len(backup.Appeals))),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", backup.BlockHeight)),
		),
	)

	return nil
}

// ValidateState performs comprehensive state validation
func (k Keeper) ValidateState(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	// Validate providers
	providerIterator := storetypes.KVStorePrefixIterator(store, types.ProviderKeyPrefix)
	defer providerIterator.Close()

	for ; providerIterator.Valid(); providerIterator.Next() {
		var provider types.Provider
		if err := k.cdc.Unmarshal(providerIterator.Value(), &provider); err != nil {
			return fmt.Errorf("corrupt provider data at key %x: %w", providerIterator.Key(), err)
		}

		// Validate provider fields
		if provider.Address == "" {
			return fmt.Errorf("provider has empty address")
		}

		if provider.Stake.IsNil() || provider.Stake.IsNegative() {
			return fmt.Errorf("provider %s has invalid stake: %s", provider.Address, provider.Stake)
		}
	}

	// Validate requests
	requestIterator := storetypes.KVStorePrefixIterator(store, types.RequestKeyPrefix)
	defer requestIterator.Close()

	for ; requestIterator.Valid(); requestIterator.Next() {
		var request types.Request
		if err := k.cdc.Unmarshal(requestIterator.Value(), &request); err != nil {
			return fmt.Errorf("corrupt request data at key %x: %w", requestIterator.Key(), err)
		}

		// Validate request fields
		if request.Requester == "" {
			return fmt.Errorf("request %d has empty requester", request.Id)
		}

		if request.Status == types.REQUEST_STATUS_UNSPECIFIED {
			return fmt.Errorf("request %d has empty status", request.Id)
		}
	}

	// Validate escrows match requests
	escrowIterator := storetypes.KVStorePrefixIterator(store, types.EscrowKeyPrefix)
	defer escrowIterator.Close()

	for ; escrowIterator.Valid(); escrowIterator.Next() {
		requestIDStr := string(escrowIterator.Key()[len(types.EscrowKeyPrefix):])
		// Parse request ID
		var requestID uint64
		if _, err := fmt.Sscanf(requestIDStr, "%d", &requestID); err != nil {
			return fmt.Errorf("invalid request ID in escrow key: %s", requestIDStr)
		}

		// Verify corresponding request exists
		request, err := k.GetRequest(ctx, requestID)
		if err != nil {
			return fmt.Errorf("escrow exists for non-existent request %d", requestID)
		}

		// Verify escrow amount is valid
		var amount math.Int
		if err := amount.Unmarshal(escrowIterator.Value()); err != nil {
			return fmt.Errorf("corrupt escrow amount for request %d", requestID)
		}

		if amount.IsNil() || amount.IsNegative() {
			return fmt.Errorf("invalid escrow amount for request %d: %s", requestID, amount)
		}

		// Verify escrow matches request state
		if request.Status == types.REQUEST_STATUS_COMPLETED || request.Status == types.REQUEST_STATUS_CANCELLED {
			return fmt.Errorf("escrow should not exist for %s request %d", request.Status, requestID)
		}
	}

	return nil
}

// RecoverFromCorruption attempts to recover from state corruption
func (k Keeper) RecoverFromCorruption(ctx context.Context, backupPath string) error {
	// Load backup
	backup, err := LoadComputeBackupFromFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to load backup: %w", err)
	}

	// Validate current state
	if err := k.ValidateState(ctx); err == nil {
		return fmt.Errorf("state is valid, recovery not needed")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Error("State corruption detected, initiating recovery", "backup", backupPath)

	// Import backup state
	if err := k.ImportState(ctx, backup); err != nil {
		return fmt.Errorf("failed to import backup: %w", err)
	}

	// Validate recovered state
	if err := k.ValidateState(ctx); err != nil {
		return fmt.Errorf("recovered state is still invalid: %w", err)
	}

	sdkCtx.Logger().Info("State recovery completed successfully")

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_state_recovered",
			sdk.NewAttribute("backup_height", fmt.Sprintf("%d", backup.BlockHeight)),
			sdk.NewAttribute("backup_timestamp", backup.Timestamp.String()),
		),
	)

	return nil
}

// CreateCheckpoint creates a state checkpoint
func (k Keeper) CreateCheckpoint(ctx context.Context, name string) error {
	backup, err := k.ExportState(ctx)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	filename := fmt.Sprintf("compute_checkpoint_%s_%d.json", name, sdkCtx.BlockHeight())

	return backup.SaveToFile(filename)
}

// SaveToFile saves backup data to a file
func (backup *StateBackupData) SaveToFile(filepath string) error {
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// LoadComputeBackupFromFile loads backup data from a file
func LoadComputeBackupFromFile(filepath string) (*StateBackupData, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	var backup StateBackupData
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup: %w", err)
	}

	// Verify checksum
	expectedChecksum := calculateComputeChecksum(&backup)
	if backup.Checksum != expectedChecksum {
		return nil, fmt.Errorf("backup file corrupted: checksum mismatch")
	}

	return &backup, nil
}

// calculateComputeChecksum calculates a checksum for backup data
func calculateComputeChecksum(backup *StateBackupData) string {
	// Create a copy without checksum for hashing
	temp := *backup
	temp.Checksum = ""

	data, _ := json.Marshal(temp)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// clearComputeState clears all compute module state (use with caution!)
func (k Keeper) clearComputeState(ctx context.Context) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	// Clear providers
	clearStorePrefix(store, types.ProviderKeyPrefix)
	clearStorePrefix(store, ActiveProvidersPrefix)

	// Clear requests
	clearStorePrefix(store, types.RequestKeyPrefix)
	clearStorePrefix(store, RequestsByRequesterPrefix)
	clearStorePrefix(store, RequestsByProviderPrefix)
	clearStorePrefix(store, RequestsByStatusPrefix)

	// Clear results
	clearStorePrefix(store, types.ResultKeyPrefix)

	// Clear escrows
	clearStorePrefix(store, types.EscrowKeyPrefix)

	// Clear nonces
	clearStorePrefix(store, types.NonceKeyPrefix)

	// Clear dispute/appeal state
	clearStorePrefix(store, DisputeKeyPrefix)
	clearStorePrefix(store, EvidenceKeyPrefix)
	clearStorePrefix(store, DisputesByRequestPrefix)
	clearStorePrefix(store, DisputesByStatusPrefix)
	clearStorePrefix(store, SlashRecordKeyPrefix)
	clearStorePrefix(store, SlashRecordsByProviderPrefix)
	clearStorePrefix(store, AppealKeyPrefix)
	clearStorePrefix(store, AppealsByStatusPrefix)

	// Clear singleton keys
	store.Delete(ParamsKey)
	store.Delete(GovernanceParamsKey)
	store.Delete(NextRequestIDKey)
	store.Delete(NextDisputeIDKey)
	store.Delete(NextSlashIDKey)
	store.Delete(NextAppealIDKey)
}

// clearStorePrefix removes all keys with a given prefix
func clearStorePrefix(store storetypes.KVStore, prefix []byte) {
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

// Placeholder methods (should be implemented based on actual keeper)

/*
func (k Keeper) SetProvider(ctx sdk.Context, provider types.Provider) {
    // ...
}

func (k Keeper) SetRequest(ctx sdk.Context, request types.Request) {
    // ...
}

func (k Keeper) GetRequest(ctx sdk.Context, requestID uint64) (types.Request, bool) {
    // ...
}
*/
/*
func (k Keeper) SetProvider(ctx context.Context, provider *types.ComputeProvider) error {
	// Implementation needed
	return nil
}

func (k Keeper) SetRequest(ctx context.Context, request *types.ComputeRequest) error {
	// Implementation needed
	return nil
}
*/
/*
func (k Keeper) GetRequest(ctx context.Context, requestID string) (*types.ComputeRequest, error) {
	// Implementation needed
	return nil, nil
}
*/

/*
func (k Keeper) GetRequest(ctx context.Context, requestID string) (*types.ComputeRequest, error) {
	// Implementation needed
	return &types.ComputeRequest{}, nil
}
*/
