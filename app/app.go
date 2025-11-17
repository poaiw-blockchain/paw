// Package app provides the PAW blockchain application implementation.
//
// This package defines the PAW application structure, initializes all Cosmos SDK
// modules, and configures the application for production use. It integrates the
// core modules (bank, staking, governance) with custom PAW modules (DEX, Oracle,
// Compute) to create a complete blockchain application.
//
// Key features:
//   - Cosmos SDK v0.50+ integration
//   - Tendermint BFT consensus
//   - Custom module initialization (DEX, Oracle, Compute)
//   - State management via IAVL trees
//   - gRPC and REST API exposure
//   - Transaction routing and processing
//
// The App struct is the central type that coordinates all blockchain operations,
// from transaction processing to consensus participation and state management.
package app

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/evidence"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/spf13/cast"

	// CosmWasm - commented out until IBC is initialized
	// "github.com/CosmWasm/wasmd/x/wasm"
	// wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types" // Keep for store key only

	// PAW custom modules
	dexmodule "github.com/paw-chain/paw/x/dex"
	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	dextypes "github.com/paw-chain/paw/x/dex/types"

	computemodule "github.com/paw-chain/paw/x/compute"
	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
	computetypes "github.com/paw-chain/paw/x/compute/types"

	oraclemodule "github.com/paw-chain/paw/x/oracle"
	oraclekeeper "github.com/paw-chain/paw/x/oracle/keeper"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

const (
	AccountAddressPrefix = "paw"
	Name                 = "paw"
)

var (
	// DefaultNodeHome is the default home directory for the application daemon.
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(
			[]govclient.ProposalHandler{
				paramsclient.ProposalHandler,
			},
		),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		vesting.AppModuleBasic{},
		consensus.AppModuleBasic{},
		// wasm.AppModuleBasic{}, // TODO: Uncomment when WasmKeeper is initialized

		// PAW custom modules
		dexmodule.AppModuleBasic{},
		computemodule.AppModuleBasic{},
		oraclemodule.AppModuleBasic{},
	)
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".paw")
}

var (
	_ runtime.AppI            = (*PAWApp)(nil)
	_ servertypes.Application = (*PAWApp)(nil)
)

// PAWApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type PAWApp struct {
	*baseapp.BaseApp

	cdc               *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry
	txConfig          client.TxConfig

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	SlashingKeeper        slashingkeeper.Keeper
	MintKeeper            mintkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             *govkeeper.Keeper
	CrisisKeeper          *crisiskeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper

	// WASM keeper - commented out until IBC is initialized
	// WasmKeeper wasmkeeper.Keeper

	// PAW custom keepers
	DEXKeeper     dexkeeper.Keeper
	ComputeKeeper *computekeeper.Keeper
	OracleKeeper  *oraclekeeper.Keeper

	// the module manager
	mm *module.Manager

	// simulation manager
	sm *module.SimulationManager

	// module configurator
	configurator module.Configurator
}

// NewPAWApp returns a reference to an initialized PAW application.
func NewPAWApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *PAWApp {
	// Set SDK config before creating any addresses
	SetConfig()

	encodingConfig := MakeEncodingConfig()
	appCodec := encodingConfig.Codec
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry
	txConfig := encodingConfig.TxConfig

	bApp := baseapp.NewBaseApp(Name, logger, db, txConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)
	bApp.SetTxEncoder(txConfig.TxEncoder())

	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		minttypes.StoreKey, distrtypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, paramstypes.StoreKey, upgradetypes.StoreKey,
		feegrant.StoreKey, evidencetypes.StoreKey, consensusparamtypes.StoreKey,
		crisistypes.StoreKey,
		wasmtypes.StoreKey,
		// PAW custom modules
		dextypes.StoreKey,
		computetypes.StoreKey,
		oracletypes.StoreKey,
	)
	tkeys := storetypes.NewTransientStoreKeys(paramstypes.TStoreKey)
	memKeys := storetypes.NewMemoryStoreKeys()

	app := &PAWApp{
		BaseApp:           bApp,
		cdc:               legacyAmino,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		txConfig:          txConfig,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	// Initialize keepers
	app.ParamsKeeper = initParamsKeeper(appCodec, legacyAmino, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])

	// set the BaseApp's parameter store
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]), authtypes.NewModuleAddress(govtypes.ModuleName).String(), runtime.EventService{})
	bApp.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

	// add keepers
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec, runtime.NewKVStoreService(keys[authtypes.StoreKey]), authtypes.ProtoBaseAccount, maccPerms, authcodec.NewBech32Codec(AccountAddressPrefix), AccountAddressPrefix, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec, runtime.NewKVStoreService(keys[banktypes.StoreKey]), app.AccountKeeper, BlockedModuleAccountAddrs(), authtypes.NewModuleAddress(govtypes.ModuleName).String(), logger,
	)

	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(keys[stakingtypes.StoreKey]), app.AccountKeeper, app.BankKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(), authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()), authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	app.MintKeeper = mintkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(keys[minttypes.StoreKey]), app.StakingKeeper,
		app.AccountKeeper, app.BankKeeper, authtypes.FeeCollectorName, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(keys[distrtypes.StoreKey]), app.AccountKeeper, app.BankKeeper,
		app.StakingKeeper, authtypes.FeeCollectorName, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec, legacyAmino, runtime.NewKVStoreService(keys[slashingtypes.StoreKey]), app.StakingKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	invCheckPeriod := cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod))
	app.CrisisKeeper = crisiskeeper.NewKeeper(appCodec, runtime.NewKVStoreService(keys[crisistypes.StoreKey]), invCheckPeriod,
		app.BankKeeper, authtypes.FeeCollectorName, authtypes.NewModuleAddress(govtypes.ModuleName).String(), app.AccountKeeper.AddressCodec())

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(keys[feegrant.StoreKey]), app.AccountKeeper)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks(), app.SlashingKeeper.Hooks()),
	)

	app.UpgradeKeeper = upgradekeeper.NewKeeper(map[int64]bool{}, runtime.NewKVStoreService(keys[upgradetypes.StoreKey]), appCodec, DefaultNodeHome, app.BaseApp, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	// register the proposal types
	govRouter := govv1beta1.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.ParamsKeeper))

	govConfig := govtypes.DefaultConfig()

	app.GovKeeper = govkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(keys[govtypes.StoreKey]), app.AccountKeeper, app.BankKeeper,
		app.StakingKeeper, app.DistrKeeper, app.MsgServiceRouter(), govConfig, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Set legacy router for backwards compatibility with gov v1beta1
	app.GovKeeper.SetLegacyRouter(govRouter)

	app.EvidenceKeeper = *evidencekeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(keys[evidencetypes.StoreKey]), app.StakingKeeper, app.SlashingKeeper, app.AccountKeeper.AddressCodec(), runtime.ProvideCometInfoService(),
	)

	// TODO: Initialize WASM keeper (requires IBC setup first)
	//
	// SECURITY REQUIREMENTS for CosmWasm initialization:
	// 1. Set upload access to GOVERNANCE ONLY (not everyone!)
	//    - Use AllowNobody for production (governance proposals only)
	//    - Never use AllowEverybody in production
	// 2. Configure secure defaults:
	//    - SmartQueryGasLimit: 3_000_000 (prevent DoS)
	//    - MemoryCacheSize: 100 (limit cache to 100MB)
	//    - ContractDebugMode: false (disable debug in production)
	// 3. Supported features: "iterator,staking,stargate"
	// 4. wasmDir: filepath.Join(DefaultNodeHome, "wasm")
	//
	// Example initialization (requires IBC):
	//   wasmDir := filepath.Join(DefaultNodeHome, "wasm")
	//   wasmConfig := wasmtypes.WasmConfig{
	//       SmartQueryGasLimit: 3_000_000,
	//       MemoryCacheSize: 100,
	//       ContractDebugMode: false,
	//   }
	//   app.WasmKeeper = wasmkeeper.NewKeeper(
	//       appCodec,
	//       runtime.NewKVStoreService(keys[wasmtypes.StoreKey]),
	//       app.AccountKeeper,
	//       app.BankKeeper,
	//       app.StakingKeeper,
	//       app.DistrKeeper,
	//       app.IBCKeeper.ChannelKeeper, // Requires IBC
	//       app.IBCKeeper.ChannelKeeper, // Requires IBC
	//       app.IBCKeeper.PortKeeper,    // Requires IBC
	//       app.ScopedWasmKeeper,        // Requires IBC
	//       app.TransferKeeper,          // Requires IBC
	//       app.MsgServiceRouter(),
	//       app.GRPCQueryRouter(),
	//       wasmDir,
	//       wasmConfig,
	//       "iterator,staking,stargate",
	//       authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	//   )
	//
	// NOTE: CosmWasm module is registered in ModuleBasics and Module Manager
	//       but keeper initialization requires IBC to be set up first.
	//       This is a deliberate ordering dependency.

	// Initialize PAW custom keepers
	app.DEXKeeper = dexkeeper.NewKeeper(
		appCodec,
		keys[dextypes.StoreKey],
		app.BankKeeper,
	)

	app.ComputeKeeper = computekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[computetypes.StoreKey]),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.OracleKeeper = oraclekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[oracletypes.StoreKey]),
		app.BankKeeper,
		app.StakingKeeper,
		app.SlashingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	/****  Module Options ****/

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
		genutil.NewAppModule(app.AccountKeeper, app.StakingKeeper, app, txConfig),
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		crisis.NewAppModule(app.CrisisKeeper, false, app.GetSubspace(crisistypes.ModuleName)),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper, nil, app.GetSubspace(minttypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName), app.interfaceRegistry),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(app.UpgradeKeeper, app.AccountKeeper.AddressCodec()),
		evidence.NewAppModule(app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),
		// wasm.NewAppModule(...), // TODO: Uncomment when WasmKeeper is initialized

		// PAW custom modules
		dexmodule.NewAppModule(appCodec, app.DEXKeeper),
		computemodule.NewAppModule(appCodec, app.ComputeKeeper),
		oraclemodule.NewAppModule(appCodec, app.OracleKeeper),
	)

	// Set init genesis order
	// NOTE: The genutil module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: wasm module should be initialized after all modules it depends on
	app.mm.SetOrderInitGenesis(
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		crisistypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		consensusparamtypes.ModuleName,
		// wasmtypes.ModuleName, // TODO: Add when WasmKeeper is initialized
		// PAW custom modules
		dextypes.ModuleName,
		computetypes.ModuleName,
		oracletypes.ModuleName,
	)

	// Set begin blockers order
	app.mm.SetOrderBeginBlockers(
		minttypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		genutiltypes.ModuleName,
		// PAW custom modules
		dextypes.ModuleName,
		computetypes.ModuleName,
		oracletypes.ModuleName,
	)

	// Set end blockers order
	app.mm.SetOrderEndBlockers(
		crisistypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		feegrant.ModuleName,
		// PAW custom modules
		dextypes.ModuleName,
		computetypes.ModuleName,
		oracletypes.ModuleName,
	)

	// Initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// Initialize and seal the baseapp configuration
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)

	// Register upgrade handlers if needed
	// app.UpgradeKeeper.SetUpgradeHandler(...)

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			panic(err)
		}
	}

	return app
}

// Name returns the name of the App
func (app *PAWApp) Name() string { return app.BaseApp.Name() }

// LegacyAmino returns PAWApp's amino codec.
func (app *PAWApp) LegacyAmino() *codec.LegacyAmino {
	return app.cdc
}

// AppCodec returns PAW's app codec.
func (app *PAWApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns PAW's InterfaceRegistry
func (app *PAWApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns PAWApp's TxConfig
func (app *PAWApp) TxConfig() client.TxConfig {
	return app.txConfig
}

// GetKey returns the KVStoreKey for the provided store key.
func (app *PAWApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
func (app *PAWApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// SimulationManager implements the SimulationApp interface
func (app *PAWApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *PAWApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// Register gRPC Gateway routes for all modules
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register legacy REST routes (if needed)
	if apiConfig.Swagger {
		server.RegisterSwaggerAPI(clientCtx, apiSvr.Router, apiConfig.Swagger)
	}
}

// LoadHeight loads a particular height
func (app *PAWApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// InitChainer application updates at chain initialization
func (app *PAWApp) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		return nil, err
	}
	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// BeginBlocker application updates every begin block
func (app *PAWApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return app.mm.BeginBlock(ctx)
}

// EndBlocker application updates every end block
func (app *PAWApp) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.mm.EndBlock(ctx)
}

// RegisterTendermintService registers the Tendermint service on the provided server
func (app *PAWApp) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterServiceServer(app.GRPCQueryRouter(), cmtservice.NewQueryServer(clientCtx, app.interfaceRegistry, app.Query))
}

// RegisterTxService registers the tx service on the provided server
func (app *PAWApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterNodeService registers the node gRPC service on the provided server
func (app *PAWApp) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	node.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
}

// ExportAppStateAndValidators exports the state of the application for a genesis
// file.
func (app *PAWApp) ExportAppStateAndValidators(
	forZeroHeight bool, jailAllowedAddrs []string, modulesToExport []string,
) (servertypes.ExportedApp, error) {
	// Create context for export
	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})

	// If exporting for zero height, we need to do some cleanup
	if forZeroHeight {
		app.prepForZeroHeightGenesis(ctx, jailAllowedAddrs)
	}

	// Export genesis state from all modules
	genState, err := app.mm.ExportGenesisForModules(ctx, app.appCodec, modulesToExport)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	// Marshal genesis state
	appState, err := json.MarshalIndent(genState, "", "  ")
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	// Get validators
	validators, err := staking.WriteValidators(ctx, app.StakingKeeper)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	return servertypes.ExportedApp{
		AppState:        appState,
		Validators:      validators,
		Height:          app.LastBlockHeight(),
		ConsensusParams: app.BaseApp.GetConsensusParams(ctx),
	}, nil
}

// prepForZeroHeightGenesis prepares app state for a zero-height genesis export
func (app *PAWApp) prepForZeroHeightGenesis(ctx sdk.Context, jailAllowedAddrs []string) {
	// Create map of allowed addresses for unjailing
	allowedAddrsMap := make(map[string]bool)
	for _, addr := range jailAllowedAddrs {
		allowedAddrsMap[addr] = true
	}

	// Withdraw all validator commission
	err := app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			panic(err)
		}
		_, _ = app.DistrKeeper.WithdrawValidatorCommission(ctx, valBz)
		return false
	})
	if err != nil {
		panic(err)
	}

	// Withdraw all delegator rewards
	dels, err := app.StakingKeeper.GetAllDelegations(ctx)
	if err != nil {
		panic(err)
	}
	for _, delegation := range dels {
		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			panic(err)
		}

		delAddr, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
		if err != nil {
			panic(err)
		}

		_, _ = app.DistrKeeper.WithdrawDelegationRewards(ctx, delAddr, valAddr)
	}

	// Clear validator slash events
	app.DistrKeeper.DeleteAllValidatorSlashEvents(ctx)

	// Clear validator historical rewards
	app.DistrKeeper.DeleteAllValidatorHistoricalRewards(ctx)

	// Set current validator historical rewards to zero period
	err = app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			panic(err)
		}
		err = app.DistrKeeper.SetValidatorHistoricalRewards(ctx, valBz, 0, distrtypes.NewValidatorHistoricalRewards(sdk.DecCoins{}, 1))
		if err != nil {
			panic(err)
		}
		return false
	})
	if err != nil {
		panic(err)
	}

	// Set accumulated commissions to zero
	err = app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			panic(err)
		}
		err = app.DistrKeeper.SetValidatorAccumulatedCommission(ctx, valBz, distrtypes.ValidatorAccumulatedCommission{Commission: sdk.DecCoins{}})
		if err != nil {
			panic(err)
		}
		return false
	})
	if err != nil {
		panic(err)
	}

	// Set outstanding rewards to zero
	err = app.DistrKeeper.SetValidatorOutstandingRewards(ctx, sdk.ValAddress{}, distrtypes.ValidatorOutstandingRewards{Rewards: sdk.DecCoins{}})
	if err != nil {
		panic(err)
	}

	// Reset context height to zero
	ctx = ctx.WithBlockHeight(0)

	// Reinitialize all validators
	err = app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		// Jail validators that are not in the allowed list
		valAddr := val.GetOperator()
		if !allowedAddrsMap[valAddr] {
			consAddr, err := val.GetConsAddr()
			if err != nil {
				panic(err)
			}
			err = app.SlashingKeeper.Jail(ctx, consAddr)
			if err != nil {
				panic(err)
			}
			// Reset signing info
			signingInfo, err := app.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
			if err == nil {
				signingInfo.StartHeight = 0
				signingInfo.JailedUntil = ctx.BlockTime()
				err = app.SlashingKeeper.SetValidatorSigningInfo(ctx, consAddr, signingInfo)
				if err != nil {
					panic(err)
				}
			}
		}

		// Initialize validator distribution record
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(valAddr)
		if err != nil {
			panic(err)
		}
		err = app.DistrKeeper.SetValidatorHistoricalRewards(ctx, valBz, 0, distrtypes.NewValidatorHistoricalRewards(sdk.DecCoins{}, 1))
		if err != nil {
			panic(err)
		}
		err = app.DistrKeeper.SetValidatorCurrentRewards(ctx, valBz, distrtypes.NewValidatorCurrentRewards(sdk.DecCoins{}, 1))
		if err != nil {
			panic(err)
		}

		return false
	})
	if err != nil {
		panic(err)
	}

	// Reinitialize all delegations
	for _, del := range dels {
		valAddr, err := sdk.ValAddressFromBech32(del.ValidatorAddress)
		if err != nil {
			panic(err)
		}
		delAddr, err := sdk.AccAddressFromBech32(del.DelegatorAddress)
		if err != nil {
			panic(err)
		}
		err = app.DistrKeeper.SetDelegatorStartingInfo(ctx, valAddr, delAddr, distrtypes.NewDelegatorStartingInfo(2, math.LegacyOneDec(), 0))
		if err != nil {
			panic(err)
		}
	}
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName)
	paramsKeeper.Subspace(crisistypes.ModuleName)
	// PAW custom modules
	paramsKeeper.Subspace(dextypes.ModuleName)
	paramsKeeper.Subspace(computetypes.ModuleName)
	paramsKeeper.Subspace(oracletypes.ModuleName)

	return paramsKeeper
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	return maccPerms
}

// BlockedModuleAccountAddrs returns all the app's blocked module account
// addresses.
func BlockedModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range GetMaccPerms() {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// module account permissions
var maccPerms = map[string][]string{
	authtypes.FeeCollectorName:     nil,
	distrtypes.ModuleName:          nil,
	minttypes.ModuleName:           {authtypes.Minter},
	stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
	stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
	govtypes.ModuleName:            {authtypes.Burner},
	wasmtypes.ModuleName:           {authtypes.Burner},
	// PAW custom modules
	dextypes.ModuleName:     {authtypes.Minter, authtypes.Burner},
	computetypes.ModuleName: nil,
	oracletypes.ModuleName:  nil,
}
