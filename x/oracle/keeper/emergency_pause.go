package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// EmergencyPauseOracle pauses all oracle operations
// Can be called by emergency admin or governance authority
func (k Keeper) EmergencyPauseOracle(ctx context.Context, pausedBy, reason string) error {
	// Check if already paused
	if k.IsOraclePaused(ctx) {
		return types.ErrOraclePaused
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Create pause state
	pauseState := types.EmergencyPauseState{
		Paused:         true,
		PausedBy:       pausedBy,
		PauseReason:    reason,
		PausedAtHeight: sdkCtx.BlockHeight(),
	}

	// Marshal and store
	bz, err := k.cdc.Marshal(&pauseState)
	if err != nil {
		return fmt.Errorf("PauseOracle: failed to marshal pause state: %w", err)
	}

	store.Set(types.EmergencyPauseStateKey, bz)

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeEmergencyPause,
			sdk.NewAttribute(types.AttributeKeyPausedBy, pausedBy),
			sdk.NewAttribute(types.AttributeKeyPauseReason, reason),
			sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", sdkCtx.BlockHeight())),
		),
	)

	return nil
}

// ResumeOracle resumes normal oracle operations
// Can only be called by governance authority
func (k Keeper) ResumeOracle(ctx context.Context, resumedBy, reason string) error {
	// Check if paused
	if !k.IsOraclePaused(ctx) {
		return types.ErrOracleNotPaused
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Remove pause state
	store.Delete(types.EmergencyPauseStateKey)

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeEmergencyResume,
			sdk.NewAttribute(types.AttributeKeyActor, resumedBy),
			sdk.NewAttribute(types.AttributeKeyResumeReason, reason),
			sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", sdkCtx.BlockHeight())),
		),
	)

	return nil
}

// IsOraclePaused checks if the oracle module is currently paused
func (k Keeper) IsOraclePaused(ctx context.Context) bool {
	store := k.getStore(ctx)
	bz := store.Get(types.EmergencyPauseStateKey)
	if bz == nil {
		return false
	}

	var pauseState types.EmergencyPauseState
	if err := k.cdc.Unmarshal(bz, &pauseState); err != nil {
		return false
	}

	return pauseState.Paused
}

// GetEmergencyPauseState retrieves the current emergency pause state
func (k Keeper) GetEmergencyPauseState(ctx context.Context) (*types.EmergencyPauseState, error) {
	store := k.getStore(ctx)
	bz := store.Get(types.EmergencyPauseStateKey)
	if bz == nil {
		return &types.EmergencyPauseState{Paused: false}, nil
	}

	var pauseState types.EmergencyPauseState
	if err := k.cdc.Unmarshal(bz, &pauseState); err != nil {
		return nil, fmt.Errorf("GetEmergencyPauseState: failed to unmarshal pause state: %w", err)
	}

	return &pauseState, nil
}

// CheckEmergencyPause checks if oracle is paused and returns an error if so
func (k Keeper) CheckEmergencyPause(ctx context.Context) error {
	if !k.IsOraclePaused(ctx) {
		return nil
	}

	pauseState, err := k.GetEmergencyPauseState(ctx)
	if err != nil {
		return types.ErrOraclePaused
	}

	return types.ErrOraclePaused.Wrapf(
		"paused by %s at height %d: %s",
		pauseState.PausedBy,
		pauseState.PausedAtHeight,
		pauseState.PauseReason,
	)
}
