package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	v2 "github.com/paw-chain/paw/x/oracle/migrations/v2"
)

// Migrator is a struct for handling in-place store migrations for the oracle module.
// It encapsulates the keeper and provides methods for migrating between different
// consensus versions of the module.
type Migrator struct {
	keeper *Keeper
}

// NewMigrator returns a new Migrator instance for the oracle module.
// The migrator is used by the module manager to execute store migrations
// during chain upgrades.
func NewMigrator(keeper *Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 migrates the oracle module state from consensus version 1 to 2.
// This migration performs:
// - Price feed state validation
// - Validator vote validation
// - Cryptoeconomic features initialization
//
// This migration is idempotent and can be safely run multiple times.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	ctx.Logger().Info("Executing oracle module migration from v1 to v2")
	return v2.Migrate(ctx, m.keeper.storeKey, m.keeper.cdc)
}
