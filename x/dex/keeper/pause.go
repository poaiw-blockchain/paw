package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// IsPaused checks if the module is currently paused
func (k Keeper) IsPaused(ctx sdk.Context) bool {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PausedKey)
	if bz == nil {
		return false
	}
	return bz[0] == 1
}

// SetPaused sets the paused state of the module
func (k Keeper) SetPaused(ctx sdk.Context, paused bool) {
	store := ctx.KVStore(k.storeKey)
	if paused {
		store.Set(types.PausedKey, []byte{1})

		// Emit pause event
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"module_paused",
				sdk.NewAttribute("module", types.ModuleName),
				sdk.NewAttribute("paused_at", fmt.Sprintf("%d", ctx.BlockHeight())),
			),
		)
	} else {
		store.Set(types.PausedKey, []byte{0})

		// Emit unpause event
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"module_unpaused",
				sdk.NewAttribute("module", types.ModuleName),
				sdk.NewAttribute("unpaused_at", fmt.Sprintf("%d", ctx.BlockHeight())),
			),
		)
	}
}

// PauseModule pauses the module (governance only)
func (k Keeper) PauseModule(ctx sdk.Context) error {
	if k.IsPaused(ctx) {
		return types.ErrModulePaused.Wrap("module is already paused")
	}

	k.SetPaused(ctx, true)
	k.Logger(ctx).Info("DEX module paused", "height", ctx.BlockHeight())

	return nil
}

// UnpauseModule unpauses the module (governance only)
func (k Keeper) UnpauseModule(ctx sdk.Context) error {
	if !k.IsPaused(ctx) {
		return types.ErrInvalidParams.Wrap("module is not paused")
	}

	k.SetPaused(ctx, false)
	k.Logger(ctx).Info("DEX module unpaused", "height", ctx.BlockHeight())

	return nil
}

// RequireNotPaused returns an error if the module is paused
func (k Keeper) RequireNotPaused(ctx sdk.Context) error {
	if k.IsPaused(ctx) {
		return types.ErrModulePaused.Wrap("module operations are currently paused")
	}
	return nil
}
