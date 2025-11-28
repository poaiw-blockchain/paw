package compute

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/paw-chain/paw/x/compute/client/cli"
	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/simulation"
	computetypes "github.com/paw-chain/paw/x/compute/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// AppModuleBasic defines the basic application module for the compute module.
type AppModuleBasic struct {
	cdc codec.Codec
}

// Name returns the compute module's name.
func (AppModuleBasic) Name() string {
	return computetypes.ModuleName
}

// RegisterLegacyAminoCodec registers the compute module's types on the LegacyAmino codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	computetypes.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers the compute module's interface types
func (AppModuleBasic) RegisterInterfaces(registry types.InterfaceRegistry) {
	computetypes.RegisterInterfaces(registry)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the compute module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {}

// GetTxCmd returns the root tx command for the compute module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the root query command for the compute module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// AppModule implements an application module for the compute module.
type AppModule struct {
	AppModuleBasic
	keeper *keeper.Keeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper *keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{cdc: cdc},
		keeper:         keeper,
	}
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// IsOnePerModuleType implements the appmodule.AppModule interface.
func (am AppModule) IsOnePerModuleType() {}

// RegisterInvariants registers the compute module's invariants.
func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
	keeper.RegisterInvariants(ir, *am.keeper)
}

// RegisterServices registers the module's services
func (am AppModule) RegisterServices(cfg module.Configurator) {
	computetypes.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(*am.keeper))

	// Create rate limiter: 100 requests per second with burst of 200
	rateLimiter := keeper.NewRateLimiter(100, 200)

	// Wrap query server with rate limiting
	baseQueryServer := keeper.NewQueryServerImpl(*am.keeper)
	rateLimitedQueryServer := keeper.NewRateLimitedQueryServer(baseQueryServer, rateLimiter)
	computetypes.RegisterQueryServer(cfg.QueryServer(), rateLimitedQueryServer)

	// Register module migrations
	m := keeper.NewMigrator(*am.keeper)
	if err := cfg.RegisterMigration(computetypes.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(err)
	}
}

// ConsensusVersion implements AppModule/ConsensusVersion.
// It returns the current consensus version of the module.
func (AppModule) ConsensusVersion() uint64 { return 2 }

// GenerateGenesisState performs the compute module's genesis initialization.
func (am AppModule) GenerateGenesisState(simState *module.SimulationState) {
	// This function is empty now but will be populated with simulation logic later.
}

// RegisterStoreDecoder registers a decoder for compute module's types
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {
	// This function is empty now but will be populated with simulation logic later.
}

// WeightedOperations returns simulation operations for the compute module
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return simulation.WeightedOperations(
		simState.AppParams, simState.Cdc, *am.keeper,
		simState.AccountKeeper, simState.BankKeeper,
	)
}

// BeginBlock executes all ABCI BeginBlock logic for the compute module
// This is called at the beginning of every block to process scheduled tasks
func (am AppModule) BeginBlock(ctx context.Context) error {
	return am.keeper.BeginBlocker(ctx)
}

// EndBlock executes all ABCI EndBlock logic for the compute module
// This is called at the end of every block to process time-based operations
func (am AppModule) EndBlock(ctx context.Context) error {
	return am.keeper.EndBlocker(ctx)
}
