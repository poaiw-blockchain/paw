// Package keeper implements the core business logic for the PAW Compute module.
//
// The Compute keeper manages secure API key aggregation, compute task routing,
// and provider registration with TEE (Trusted Execution Environment) integration.
//
// Key features include:
//   - Provider registration with stake requirements
//   - Compute request submission and routing
//   - Task status tracking and result verification
//   - TEE attestation and verification
//   - Automatic fee distribution and reward settlement
package keeper

import (
	"encoding/binary"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

var (
	// ParamsKey is the KVStore key for module parameters
	ParamsKey = []byte{0x01}

	// NextTaskIDKey is the KVStore key for the next task ID counter
	NextTaskIDKey = []byte{0x02}

	// TaskPrefix is the KVStore key prefix for compute tasks
	TaskPrefix = []byte{0x03}
)

// Keeper maintains the state of the Compute module.
//
// The Keeper is responsible for:
//   - Managing compute provider registration and staking
//   - Routing compute requests to available providers
//   - Tracking task execution status and results
//   - Distributing rewards and penalties
//   - Enforcing TEE attestation requirements
type Keeper struct {
	cdc          codec.BinaryCodec    // Binary codec for state serialization
	storeService store.KVStoreService // KVStore service for compute state
	bankKeeper   types.BankKeeper     // Bank keeper for fee distribution
	authority    string               // Module authority (usually governance module account)
}

// NewKeeper creates a new Compute Keeper instance.
//
// Parameters:
//   - cdc: Binary codec for state serialization
//   - storeService: KVStore service for accessing compute state
//   - bankKeeper: Bank keeper for managing fee distributions
//   - authority: Bech32 address with governance authority (typically gov module)
//
// Returns a configured Keeper instance ready for use.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	bankKeeper types.BankKeeper,
	authority string,
) *Keeper {
	return &Keeper{
		cdc:          cdc,
		storeService: storeService,
		bankKeeper:   bankKeeper,
		authority:    authority,
	}
}

// Logger returns a module-specific logger with contextual information.
//
// The logger includes the module name prefix for easy identification
// in log output when debugging or monitoring compute operations.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetParams returns the module parameters
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(ParamsKey)
	if err != nil {
		panic(err)
	}
	if bz == nil {
		return types.DefaultParams()
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the module parameters
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz := k.cdc.MustMarshal(&params)
	if err := store.Set(ParamsKey, bz); err != nil {
		panic(err)
	}
}

// GetNextTaskID returns the next task ID to be used
func (k Keeper) GetNextTaskID(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(NextTaskIDKey)
	if err != nil {
		panic(err)
	}
	if bz == nil {
		return 1
	}
	return binary.BigEndian.Uint64(bz)
}

// SetNextTaskID sets the next task ID
func (k Keeper) SetNextTaskID(ctx sdk.Context, id uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	if err := store.Set(NextTaskIDKey, bz); err != nil {
		panic(err)
	}
}

// GetTask returns a task by ID
func (k Keeper) GetTask(ctx sdk.Context, id uint64) (*types.ComputeTask, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := append(TaskPrefix, sdk.Uint64ToBigEndian(id)...)
	bz, err := store.Get(key)
	if err != nil {
		panic(err)
	}
	if bz == nil {
		return nil, false
	}

	var task types.ComputeTask
	k.cdc.MustUnmarshal(bz, &task)
	return &task, true
}

// SetTask sets a task in the store
func (k Keeper) SetTask(ctx sdk.Context, task types.ComputeTask) {
	store := k.storeService.OpenKVStore(ctx)
	key := append(TaskPrefix, sdk.Uint64ToBigEndian(task.Id)...)
	bz := k.cdc.MustMarshal(&task)
	if err := store.Set(key, bz); err != nil {
		panic(err)
	}
}

// GetAllTasks returns all tasks
func (k Keeper) GetAllTasks(ctx sdk.Context) []types.ComputeTask {
	store := k.storeService.OpenKVStore(ctx)

	// Create end key by incrementing the last byte of the prefix
	endKey := make([]byte, len(TaskPrefix))
	copy(endKey, TaskPrefix)
	endKey[len(endKey)-1]++

	iterator, err := store.Iterator(TaskPrefix, endKey)
	if err != nil {
		panic(err)
	}
	defer iterator.Close()

	var tasks []types.ComputeTask
	for ; iterator.Valid(); iterator.Next() {
		var task types.ComputeTask
		k.cdc.MustUnmarshal(iterator.Value(), &task)
		tasks = append(tasks, task)
	}

	return tasks
}

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	// Set params
	k.SetParams(ctx, genState.Params)

	// Set next task ID
	k.SetNextTaskID(ctx, genState.NextTaskId)

	// Set all tasks
	for _, task := range genState.Tasks {
		k.SetTask(ctx, task)
	}
}

// ExportGenesis returns the module's exported genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:     k.GetParams(ctx),
		Tasks:      k.GetAllTasks(ctx),
		NextTaskId: k.GetNextTaskID(ctx),
	}
}
