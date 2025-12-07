package dex

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/paw-chain/paw/x/dex/client/cli"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/simulation"
	dextypes "github.com/paw-chain/paw/x/dex/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// AppModuleBasic defines the basic application module for the dex module.
type AppModuleBasic struct {
	cdc codec.Codec
}

// Name returns the dex module's name.
func (AppModuleBasic) Name() string {
	return dextypes.ModuleName
}

// RegisterLegacyAminoCodec registers the dex module's types on the LegacyAmino codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	dextypes.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers the dex module's interface types
func (AppModuleBasic) RegisterInterfaces(registry types.InterfaceRegistry) {
	dextypes.RegisterInterfaces(registry)
}

// DefaultGenesis returns the default genesis state for the dex module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(dextypes.DefaultGenesis())
}

// ValidateGenesis validates the provided genesis state.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var genState dextypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return err
	}
	return genState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the dex module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {}

// GetTxCmd returns the root tx command for the dex module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the root query command for the dex module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// AppModule implements an application module for the dex module.
type AppModule struct {
	AppModuleBasic
	keeper        *keeper.Keeper
	accountKeeper dextypes.AccountKeeper
	bankKeeper    dextypes.BankKeeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper *keeper.Keeper, ak dextypes.AccountKeeper, bk dextypes.BankKeeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{cdc: cdc},
		keeper:         keeper,
		accountKeeper:  ak,
		bankKeeper:     bk,
	}
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// IsOnePerModuleType implements the appmodule.AppModule interface.
func (am AppModule) IsOnePerModuleType() {}

// RegisterInvariants registers the DEX module's invariants.
func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
	keeper.RegisterInvariants(ir, *am.keeper)
}

// RegisterServices registers the module's services
func (am AppModule) RegisterServices(cfg module.Configurator) {
	// Register Msg service
	msgServer := keeper.NewMsgServerImpl(*am.keeper)
	dextypes.RegisterMsgServer(cfg.MsgServer(), msgServer)

	// Register Query service
	queryServer := keeper.NewQueryServerImpl(*am.keeper)
	dextypes.RegisterQueryServer(cfg.QueryServer(), queryServer)

	// Register module migrations
	m := keeper.NewMigrator(*am.keeper)
	if err := cfg.RegisterMigration(dextypes.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 1 to 2: %v", dextypes.ModuleName, err))
	}
}

// ConsensusVersion implements AppModule/ConsensusVersion.
// It returns the current consensus version of the module.
func (AppModule) ConsensusVersion() uint64 { return 2 }

// GenerateGenesisState performs the dex module's genesis initialization.
func (am AppModule) GenerateGenesisState(simState *module.SimulationState) {
	// This function is empty now but will be populated with simulation logic later.
}

// RegisterStoreDecoder registers a decoder for dex module's types
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {
	// This function is empty now but will be populated with simulation logic later.
}

// WeightedOperations returns simulation operations for the dex module
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return simulation.WeightedOperations(
		simState.AppParams, simState.Cdc, simState.TxConfig, *am.keeper,
		am.accountKeeper, am.bankKeeper,
	)
}

// BeginBlock executes all ABCI BeginBlock logic for the dex module
// This is called at the beginning of every block to process scheduled tasks
func (am AppModule) BeginBlock(ctx context.Context) error {
	return am.keeper.BeginBlocker(ctx)
}

// EndBlock executes all ABCI EndBlock logic for the dex module
// This is called at the end of every block to process time-based operations
func (am AppModule) EndBlock(ctx context.Context) error {
	return am.keeper.EndBlocker(ctx)
}

// InitGenesis initializes module state from genesis data.
func (am AppModule) InitGenesis(ctx context.Context, cdc codec.JSONCodec, data json.RawMessage) error {
	var genState dextypes.GenesisState
	if err := cdc.UnmarshalJSON(data, &genState); err != nil {
		return err
	}
	return am.keeper.InitGenesis(ctx, genState)
}

// ExportGenesis exports module state to genesis.
func (am AppModule) ExportGenesis(ctx context.Context, cdc codec.JSONCodec) (json.RawMessage, error) {
	genState, err := am.keeper.ExportGenesis(ctx)
	if err != nil {
		return nil, err
	}
	return cdc.MarshalJSON(genState)
}
