package keeper

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TASK 94: Implement audit trail for governance actions and state changes

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	Timestamp   time.Time
	BlockHeight int64
	TxHash      string
	Action      string
	Actor       string
	Target      string
	OldValue    string
	NewValue    string
	Success     bool
	ErrorMsg    string
	Metadata    map[string]string
}

// AuditCategory defines different types of auditable actions
type AuditCategory string

const (
	AuditCategoryGovernance  AuditCategory = "governance"
	AuditCategoryProvider    AuditCategory = "provider"
	AuditCategoryJob         AuditCategory = "job"
	AuditCategoryEscrow      AuditCategory = "escrow"
	AuditCategorySecurity    AuditCategory = "security"
	AuditCategoryIBC         AuditCategory = "ibc"
	AuditCategoryParams      AuditCategory = "params"
)

// LogAuditEntry records an audit trail entry
func (k Keeper) LogAuditEntry(ctx sdk.Context, entry AuditEntry) error {
	// Add blockchain context
	entry.BlockHeight = ctx.BlockHeight()
	entry.Timestamp = ctx.BlockTime()

	// Try to get tx hash from context
	if txBytes := ctx.TxBytes(); len(txBytes) > 0 {
		// Use SHA256 of tx bytes as hash
		hash := sha256.Sum256(txBytes)
		entry.TxHash = fmt.Sprintf("%X", hash)
	}

	// Store audit entry
	store := ctx.KVStore(k.storeKey)
	key := k.getAuditKey(entry)

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	store.Set(key, data)

	// Emit audit event for external monitoring
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"audit_trail",
			sdk.NewAttribute("action", entry.Action),
			sdk.NewAttribute("actor", entry.Actor),
			sdk.NewAttribute("target", entry.Target),
			sdk.NewAttribute("success", fmt.Sprintf("%t", entry.Success)),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", entry.BlockHeight)),
		),
	)

	// Log for node operators
	if entry.Success {
		ctx.Logger().Info("audit: "+entry.Action,
			"actor", entry.Actor,
			"target", entry.Target,
		)
	} else {
		ctx.Logger().Warn("audit: "+entry.Action+" (failed)",
			"actor", entry.Actor,
			"target", entry.Target,
			"error", entry.ErrorMsg,
		)
	}

	return nil
}

// getAuditKey generates storage key for audit entry
func (k Keeper) getAuditKey(entry AuditEntry) []byte {
	// Key format: audit_[blockheight]_[timestamp]_[action]
	key := fmt.Sprintf("audit_%d_%d_%s",
		entry.BlockHeight,
		entry.Timestamp.Unix(),
		entry.Action,
	)
	return []byte(key)
}

// AuditGovernanceAction records governance-related actions
func (k Keeper) AuditGovernanceAction(
	ctx sdk.Context,
	action string,
	actor string,
	target string,
	success bool,
	metadata map[string]string,
) error {
	entry := AuditEntry{
		Action:   fmt.Sprintf("governance.%s", action),
		Actor:    actor,
		Target:   target,
		Success:  success,
		Metadata: metadata,
	}

	return k.LogAuditEntry(ctx, entry)
}

// AuditProviderAction records provider-related actions
func (k Keeper) AuditProviderAction(
	ctx sdk.Context,
	action string,
	providerAddr string,
	jobID string,
	success bool,
	errorMsg string,
) error {
	entry := AuditEntry{
		Action:   fmt.Sprintf("provider.%s", action),
		Actor:    providerAddr,
		Target:   jobID,
		Success:  success,
		ErrorMsg: errorMsg,
	}

	return k.LogAuditEntry(ctx, entry)
}

// AuditSecurityEvent records security-related events
func (k Keeper) AuditSecurityEvent(
	ctx sdk.Context,
	eventType string,
	source string,
	details string,
	severity string,
) error {
	entry := AuditEntry{
		Action:  fmt.Sprintf("security.%s", eventType),
		Actor:   source,
		Target:  details,
		Success: false, // Security events are typically warnings/alerts
		Metadata: map[string]string{
			"severity": severity,
		},
	}

	return k.LogAuditEntry(ctx, entry)
}

// AuditParamChange records parameter changes
func (k Keeper) AuditParamChange(
	ctx sdk.Context,
	authority string,
	paramName string,
	oldValue string,
	newValue string,
) error {
	entry := AuditEntry{
		Action:   "params.update",
		Actor:    authority,
		Target:   paramName,
		OldValue: oldValue,
		NewValue: newValue,
		Success:  true,
	}

	return k.LogAuditEntry(ctx, entry)
}

// AuditStateChange records state changes with old and new values
func (k Keeper) AuditStateChange(
	ctx sdk.Context,
	stateType string,
	identifier string,
	oldState interface{},
	newState interface{},
	actor string,
) error {
	oldJSON, _ := json.Marshal(oldState)
	newJSON, _ := json.Marshal(newState)

	entry := AuditEntry{
		Action:   fmt.Sprintf("state.%s", stateType),
		Actor:    actor,
		Target:   identifier,
		OldValue: string(oldJSON),
		NewValue: string(newJSON),
		Success:  true,
	}

	return k.LogAuditEntry(ctx, entry)
}

// QueryAuditTrail retrieves audit entries for a given time range
func (k Keeper) QueryAuditTrail(
	ctx sdk.Context,
	startHeight int64,
	endHeight int64,
	maxEntries int,
) ([]AuditEntry, error) {
	store := ctx.KVStore(k.storeKey)
	prefix := []byte("audit_")

	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	entries := make([]AuditEntry, 0, maxEntries)
	count := 0

	for ; iterator.Valid() && count < maxEntries; iterator.Next() {
		var entry AuditEntry
		if err := json.Unmarshal(iterator.Value(), &entry); err != nil {
			continue
		}

		// Filter by block height range
		if entry.BlockHeight >= startHeight && entry.BlockHeight <= endHeight {
			entries = append(entries, entry)
			count++
		}
	}

	return entries, nil
}

// CleanupOldAuditEntries removes audit entries older than retention period
func (k Keeper) CleanupOldAuditEntries(ctx sdk.Context, retentionBlocks int64) error {
	store := ctx.KVStore(k.storeKey)
	cutoffHeight := ctx.BlockHeight() - retentionBlocks

	if cutoffHeight <= 0 {
		return nil
	}

	prefix := []byte("audit_")
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	deletedCount := 0
	for ; iterator.Valid(); iterator.Next() {
		var entry AuditEntry
		if err := json.Unmarshal(iterator.Value(), &entry); err != nil {
			continue
		}

		if entry.BlockHeight < cutoffHeight {
			store.Delete(iterator.Key())
			deletedCount++
		}
	}

	if deletedCount > 0 {
		ctx.Logger().Info("cleaned up old audit entries",
			"count", deletedCount,
			"cutoff_height", cutoffHeight,
		)
	}

	return nil
}
