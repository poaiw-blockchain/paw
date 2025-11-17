// Package keeper implements the core business logic for the PAW Oracle module.
//
// The Oracle keeper manages decentralized price feeds submitted by validators,
// aggregates prices using median calculation for outlier resistance, and
// enforces quality control through slashing mechanisms for inaccurate or
// inactive validators.
//
// Key features include:
//   - Validator-based price submission with cryptographic verification
//   - Median price aggregation resistant to manipulation
//   - Automatic slashing for inaccurate or missing price submissions
//   - Rate limiting to prevent spam and ensure data freshness
//   - Emergency pause/resume functionality for crisis management
package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// Keeper maintains the state of the Oracle module.
//
// The Keeper is responsible for:
//   - Managing price feed submissions from validators
//   - Aggregating prices using outlier-resistant median calculation
//   - Enforcing quality control through validator slashing
//   - Maintaining historical price data for analytics
//   - Rate limiting price submissions to ensure freshness
//   - Emergency pause/resume of oracle operations
type Keeper struct {
	cdc            codec.BinaryCodec    // Binary codec for state serialization
	storeService   store.KVStoreService // KVStore service for oracle state
	bankKeeper     types.BankKeeper     // Bank keeper for reward distribution
	stakingKeeper  types.StakingKeeper  // Staking keeper for validator queries
	slashingKeeper types.SlashingKeeper // Slashing keeper for penalizing bad actors
	authority      string               // Module authority (usually governance module account)
}

// NewKeeper creates a new Oracle Keeper instance.
//
// Parameters:
//   - cdc: Binary codec for state serialization
//   - storeService: KVStore service for accessing oracle state
//   - bankKeeper: Bank keeper for managing reward distributions
//   - stakingKeeper: Staking keeper for validator set queries
//   - slashingKeeper: Slashing keeper for penalizing inaccurate submissions
//   - authority: Bech32 address with governance authority (typically gov module)
//
// Returns a configured Keeper instance ready for use.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	bankKeeper types.BankKeeper,
	stakingKeeper types.StakingKeeper,
	slashingKeeper types.SlashingKeeper,
	authority string,
) *Keeper {
	return &Keeper{
		cdc:            cdc,
		storeService:   storeService,
		bankKeeper:     bankKeeper,
		stakingKeeper:  stakingKeeper,
		slashingKeeper: slashingKeeper,
		authority:      authority,
	}
}

// Logger returns a module-specific logger with contextual information.
//
// The logger includes the module name prefix for easy identification
// in log output when debugging or monitoring oracle operations.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	// Set params
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(fmt.Sprintf("failed to set params: %s", err))
	}

	// Set price feeds
	for _, priceFeed := range genState.PriceFeeds {
		if err := k.SetPriceFeed(ctx, priceFeed); err != nil {
			k.Logger(ctx).Error("failed to set price feed during genesis", "asset", priceFeed.Asset, "error", err)
		}
	}

	k.Logger(ctx).Info("Oracle module genesis initialized", "price_feeds", len(genState.PriceFeeds))
}

// ExportGenesis returns the module's exported genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	priceFeeds := k.GetAllPriceFeeds(ctx)

	return &types.GenesisState{
		Params:     params,
		PriceFeeds: priceFeeds,
	}
}

// GetParams gets all parameters from the store
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.KeyPrefix("params"))
	if err != nil || bz == nil {
		return types.DefaultParams()
	}

	var params types.Params
	if err := k.cdc.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}

	return params
}

// SetParams sets the module parameters
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	store := k.storeService.OpenKVStore(ctx)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	return store.Set(types.KeyPrefix("params"), bz)
}

// GetAuthority returns the module's authority (governance account)
func (k Keeper) GetAuthority() string {
	return k.authority
}
