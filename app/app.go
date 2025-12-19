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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	sdkante "github.com/cosmos/cosmos-sdk/x/auth/ante"
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

	// IBC modules
	capability "github.com/cosmos/ibc-go/modules/capability"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ibctransfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

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

	// Health endpoints
	"github.com/paw-chain/paw/app/health"

	// AnteHandler
	pawante "github.com/paw-chain/paw/app/ante"

	// Telemetry
	pawtelemetry "github.com/paw-chain/paw/app/telemetry"
)

const (
	AccountAddressPrefix = "paw"
	Name                 = "paw"
)

var (
	// DefaultNodeHome is the default home directory for the application daemon.
	DefaultNodeHome string

	ModuleBasics module.BasicManager

	moduleBasicsOnce sync.Once
	moduleBasics     module.BasicManager
)

func init() {
	// Ensure the SDK bech32 prefixes are configured before any package seals
	// the global config. This prevents panics during CLI initialization when
	// other modules attempt to use the default (cosmos) prefixes.
	SetConfig()

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".paw")
	ModuleBasics = GetBasicManager()
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
	interfaceRegistry codectypes.InterfaceRegistry
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

	// IBC keepers
	IBCKeeper            *ibckeeper.Keeper
	TransferKeeper       *ibctransferkeeper.Keeper
	CapabilityKeeper     *capabilitykeeper.Keeper
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper

	// PAW custom keepers
	DEXKeeper     *dexkeeper.Keeper
	ComputeKeeper *computekeeper.Keeper
	OracleKeeper  *oraclekeeper.Keeper

	// module basics derived from the module manager
	BasicModuleManager module.BasicManager

	// the module manager
	mm *module.Manager

	// simulation manager
	sm *module.SimulationManager

	// module configurator
	configurator module.Configurator

	// telemetry for OpenTelemetry tracing and metrics
	telemetryProvider *pawtelemetry.Provider
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

	encodingConfig := makeBaseEncodingConfig()
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
		// IBC modules
		capabilitytypes.StoreKey,
		ibcexported.StoreKey,
		ibctransfertypes.StoreKey,
		// PAW custom modules
		dextypes.StoreKey,
		computetypes.StoreKey,
		oracletypes.StoreKey,
	)
	tkeys := storetypes.NewTransientStoreKeys(paramstypes.TStoreKey)
	memKeys := storetypes.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)

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

	// Initialize IBC Capability Keeper and Scoped Keepers
	capKeeper, scopedIBCKeeper, scopedTransferKeeper, scopedComputeKeeper, scopedDEXKeeper, scopedOracleKeeper :=
		app.setupIBCCapabilities(appCodec, keys, memKeys)
	app.CapabilityKeeper = capKeeper
	_ = scopedComputeKeeper
	_ = scopedDEXKeeper
	_ = scopedOracleKeeper

	// Seal the capability keeper to prevent further scoped keepers from being created
	app.CapabilityKeeper.Seal()

	// Initialize IBC Keeper
	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		keys[ibcexported.StoreKey],
		app.GetSubspace(ibcexported.ModuleName),
		app.StakingKeeper,
		app.UpgradeKeeper,
		scopedIBCKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Initialize Transfer Keeper
	transferKeeper := ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		app.GetSubspace(ibctransfertypes.ModuleName),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		scopedTransferKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.TransferKeeper = &transferKeeper

	// Store scoped keepers for later use
	app.ScopedIBCKeeper = scopedIBCKeeper
	app.ScopedTransferKeeper = scopedTransferKeeper

	// Initialize PAW custom keepers with IBC
	app.DEXKeeper = dexkeeper.NewKeeper(
		appCodec,
		keys[dextypes.StoreKey],
		app.BankKeeper,
		app.IBCKeeper,
		app.IBCKeeper.PortKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		scopedDEXKeeper,
	)

	app.ComputeKeeper = computekeeper.NewKeeper(
		appCodec,
		keys[computetypes.StoreKey],
		app.BankKeeper,
		app.AccountKeeper,
		app.StakingKeeper,
		app.SlashingKeeper,
		app.IBCKeeper,
		app.IBCKeeper.PortKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		scopedComputeKeeper,
	)

	app.OracleKeeper = oraclekeeper.NewKeeper(
		appCodec,
		keys[oracletypes.StoreKey],
		app.BankKeeper,
		app.StakingKeeper,
		app.SlashingKeeper,
		app.IBCKeeper,
		app.IBCKeeper.PortKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		scopedOracleKeeper,
	)

	// Create IBC Router
	ibcRouter := porttypes.NewRouter()

	// Wire IBC Transfer Module
	transferModule := ibctransfer.NewIBCModule(*app.TransferKeeper)
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferModule)

	// Wire Compute IBC Module
	computeIBCModule := computemodule.NewIBCModule(*app.ComputeKeeper, appCodec)
	ibcRouter.AddRoute(computetypes.PortID, computeIBCModule)

	// Wire DEX IBC Module
	dexIBCModule := dexmodule.NewIBCModule(*app.DEXKeeper, appCodec)
	ibcRouter.AddRoute(dextypes.PortID, dexIBCModule)

	// Wire Oracle IBC Module
	oracleIBCModule := oraclemodule.NewIBCModule(*app.OracleKeeper, appCodec)
	ibcRouter.AddRoute(oracletypes.PortID, oracleIBCModule)

	// Set IBC Router on IBC Keeper
	app.IBCKeeper.SetRouter(ibcRouter)

	/****  Module Options ****/

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
		genutil.NewAppModule(app.AccountKeeper, app.StakingKeeper, app, txConfig),
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
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

		// IBC modules
		ibc.NewAppModule(app.IBCKeeper),
		ibctransfer.NewAppModule(*app.TransferKeeper),

		// PAW custom modules
		dexmodule.NewAppModule(appCodec, app.DEXKeeper, app.AccountKeeper, app.BankKeeper),
		computemodule.NewAppModule(appCodec, app.ComputeKeeper, app.AccountKeeper, app.BankKeeper),
		oraclemodule.NewAppModule(appCodec, app.OracleKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper),
	)

	// Build a BasicModuleManager from the fully wired module manager so AppModuleBasic
	// instances carry the correct codec and address codec wiring for CLI commands.
	basicOverrides := map[string]module.AppModuleBasic{
		genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		govtypes.ModuleName: gov.NewAppModuleBasic(
			[]govclient.ProposalHandler{
				paramsclient.ProposalHandler,
			},
		),
	}

	basicManager := module.BasicManager{}
	for name, mod := range app.mm.Modules {
		if appMod, ok := mod.(module.AppModule); ok {
			basicManager[name] = appMod
		}
	}
	for name, override := range basicOverrides {
		basicManager[name] = override
	}

	app.BasicModuleManager = basicManager
	app.BasicModuleManager.RegisterInterfaces(interfaceRegistry)
	RegisterLegacyAminoCodecWithManager(legacyAmino, app.BasicModuleManager)
	setModuleBasics(app.BasicModuleManager)

	// Register module services (msg and query servers)
	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	if err := app.mm.RegisterServices(app.configurator); err != nil {
		panic(fmt.Errorf("failed to register module services: %w", err))
	}

	// Set init genesis order
	// NOTE: The genutil module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must be initialized first to allow IBC modules to bind ports
	app.mm.SetOrderInitGenesis(
		capabilitytypes.ModuleName,
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
		// IBC modules (after capability)
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
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
		// IBC modules
		capabilitytypes.ModuleName,
		ibcexported.ModuleName,
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
		// IBC modules
		capabilitytypes.ModuleName,
		ibcexported.ModuleName,
		// PAW custom modules
		dextypes.ModuleName,
		computetypes.ModuleName,
		oracletypes.ModuleName,
	)

	app.sm = module.NewSimulationManager(
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper, nil, app.GetSubspace(minttypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName), app.interfaceRegistry),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		params.NewAppModule(app.ParamsKeeper),
		evidence.NewAppModule(app.EvidenceKeeper),

		// PAW custom modules
		dexmodule.NewAppModule(appCodec, app.DEXKeeper, app.AccountKeeper, app.BankKeeper),
		computemodule.NewAppModule(appCodec, app.ComputeKeeper, app.AccountKeeper, app.BankKeeper),
		oraclemodule.NewAppModule(appCodec, app.OracleKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper),
	)

	app.sm.RegisterStoreDecoders()

	// Initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// Initialize and seal the baseapp configuration
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)

	// Setup AnteHandler
	anteHandler, err := pawante.NewAnteHandler(
		&pawante.HandlerOptions{
			AccountKeeper:   app.AccountKeeper,
			BankKeeper:      app.BankKeeper,
			FeegrantKeeper:  app.FeeGrantKeeper,
			SignModeHandler: txConfig.SignModeHandler(),
			SigGasConsumer:  sdkante.DefaultSigVerificationGasConsumer,
			IBCKeeper:       app.IBCKeeper,
			ComputeKeeper:   app.ComputeKeeper,
			DEXKeeper:       app.DEXKeeper,
			OracleKeeper:    app.OracleKeeper,
		},
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create AnteHandler: %s", err))
	}
	app.SetAnteHandler(anteHandler)

	// Setup upgrade handlers
	app.setupUpgradeHandlers()

	// Setup upgrade store loaders for handling store upgrades
	app.setupUpgradeStoreLoaders()

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			panic(err)
		}

		// Set the query multi-store to enable state queries
		// This is required for gRPC/REST queries to work properly
		app.SetQueryMultiStore(app.CommitMultiStore())
	}

	// Initialize OpenTelemetry tracing and metrics
	telemetryEnabled := cast.ToBool(appOpts.Get("telemetry.enabled"))
	if telemetryEnabled {
		jaegerEndpoint := cast.ToString(appOpts.Get("telemetry.jaeger-endpoint"))
		if jaegerEndpoint == "" {
			jaegerEndpoint = "http://localhost:4318"
		}

		sampleRate := cast.ToFloat64(appOpts.Get("telemetry.sample-rate"))
		if sampleRate == 0 {
			sampleRate = 0.1 // Default 10% sampling
		}

		chainID := cast.ToString(appOpts.Get("chain-id"))
		if chainID == "" {
			chainID = "paw-testnet-1"
		}

		environment := cast.ToString(appOpts.Get("telemetry.environment"))
		if environment == "" {
			environment = "testnet"
		}

		telemetryProvider, err := pawtelemetry.NewProvider(pawtelemetry.Config{
			Enabled:           true,
			JaegerEndpoint:    jaegerEndpoint,
			SampleRate:        sampleRate,
			Environment:       environment,
			ChainID:           chainID,
			PrometheusEnabled: cast.ToBool(appOpts.Get("telemetry.prometheus-enabled")),
			MetricsPort:       cast.ToString(appOpts.Get("telemetry.metrics-port")),
		})
		if err != nil {
			logger.Error("Failed to initialize OpenTelemetry", "error", err)
		} else {
			app.telemetryProvider = telemetryProvider
			logger.Info("OpenTelemetry tracing initialized",
				"jaeger_endpoint", jaegerEndpoint,
				"sample_rate", sampleRate,
				"environment", environment,
				"chain_id", chainID,
			)

			// Perform telemetry health check
			if err := telemetryProvider.HealthCheck(); err != nil {
				logger.Error("Telemetry health check failed", "error", err)
			} else {
				logger.Info("Telemetry health check passed",
					"prometheus_enabled", cast.ToBool(appOpts.Get("telemetry.prometheus-enabled")),
				)
			}
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
func (app *PAWApp) InterfaceRegistry() codectypes.InterfaceRegistry {
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
	subspace, found := app.ParamsKeeper.GetSubspace(moduleName)
	if !found {
		panic(fmt.Sprintf("subspace not found for module: %s", moduleName))
	}
	return subspace
}

// SimulationManager implements the SimulationApp interface
func (app *PAWApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// ModuleManager exposes the module manager for simulations and testing.
func (app *PAWApp) ModuleManager() *module.Manager {
	return app.mm
}

// Configurator exposes the module configurator for testing and migrations.
func (app *PAWApp) Configurator() module.Configurator {
	return app.configurator
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
		if err := server.RegisterSwaggerAPI(clientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
			app.Logger().Error("failed to register swagger API routes", "err", err)
		}
	}

	// Register health check endpoints so operators have a consistent probe surface
	healthCfg := health.DefaultConfig()
	if nodeURI := normalizeRPCURL(clientCtx.NodeURI); nodeURI != "" {
		healthCfg.RPCURL = nodeURI
	}

	checker, err := health.NewChecker(app.Logger(), healthCfg, clientCtx)
	if err != nil {
		app.Logger().Error("failed to initialize health checker", "err", err)
		return
	}

	checker.RegisterRoutes(apiSvr.Router)
	app.Logger().Info("registered API health endpoints",
		"rpc_url", healthCfg.RPCURL,
		"cache_duration", healthCfg.CacheDuration,
	)
}

// normalizeRPCURL converts a Cosmos SDK node URI (tcp or http) into a usable RPC URL.
func normalizeRPCURL(nodeURI string) string {
	nodeURI = strings.TrimSpace(nodeURI)
	if nodeURI == "" {
		return ""
	}

	switch {
	case strings.HasPrefix(nodeURI, "tcp://"):
		return "http://" + strings.TrimPrefix(nodeURI, "tcp://")
	case strings.HasPrefix(nodeURI, "http://"), strings.HasPrefix(nodeURI, "https://"):
		return nodeURI
	default:
		return nodeURI
	}
}

// LoadHeight loads a particular height
func (app *PAWApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// InitChainer application updates at chain initialization.
//
//nolint:gocritic // sdk.Context passed by value per Cosmos SDK application interface.
func (app *PAWApp) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		return nil, err
	}
	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// BeginBlocker application updates every begin block.
//
//nolint:gocritic // sdk.Context passed by value per Cosmos SDK application interface.
func (app *PAWApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	// Trace block processing with OpenTelemetry
	if app.telemetryProvider != nil {
		tracedCtx, span := pawtelemetry.StartBlockSpan(ctx.Context(), ctx.BlockHeight(), sdk.ConsAddress(ctx.BlockHeader().ProposerAddress).String())
		defer span.End()
		ctx = ctx.WithContext(tracedCtx)
	}

	return app.mm.BeginBlock(ctx)
}

// EndBlocker application updates every end block.
//
//nolint:gocritic // sdk.Context passed by value per Cosmos SDK application interface.
func (app *PAWApp) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	// Trace block processing with OpenTelemetry
	if app.telemetryProvider != nil {
		tracedCtx, span := pawtelemetry.StartModuleSpan(ctx.Context(), "block", "end")
		defer span.End()
		ctx = ctx.WithContext(tracedCtx)
	}

	return app.mm.EndBlock(ctx)
}

// RegisterTendermintService registers the Tendermint service on the provided server.
//
//nolint:gocritic // client.Context passed by value per Cosmos SDK service interface.
func (app *PAWApp) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterServiceServer(app.GRPCQueryRouter(), cmtservice.NewQueryServer(clientCtx, app.interfaceRegistry, app.Query))
}

// RegisterTxService registers the tx service on the provided server.
//
//nolint:gocritic // client.Context passed by value per Cosmos SDK service interface.
func (app *PAWApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterNodeService registers the node gRPC service on the provided server.
//
//nolint:gocritic // client.Context passed by value per Cosmos SDK service interface.
func (app *PAWApp) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	node.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
}

// Close gracefully shuts down the application, including telemetry providers.
func (app *PAWApp) Close() error {
	if app.telemetryProvider != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.telemetryProvider.Shutdown(shutdownCtx); err != nil {
			app.Logger().Error("Failed to shutdown telemetry provider", "error", err)
			return err
		}
	}
	return nil
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

// prepForZeroHeightGenesis prepares app state for a zero-height genesis export.
//
//nolint:gocritic // sdk.Context passed by value per Cosmos SDK convention.
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
		if _, err := app.DistrKeeper.WithdrawValidatorCommission(ctx, valBz); err != nil {
			ctx.Logger().Error("failed to withdraw commission", "validator", val.GetOperator(), "error", err)
		}
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

		delAddr, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress) //nolint:staticcheck // DelegatorAddress retained for SDK compatibility during reward withdrawal.
		if err != nil {
			panic(err)
		}

		if _, err := app.DistrKeeper.WithdrawDelegationRewards(ctx, delAddr, valAddr); err != nil {
			ctx.Logger().Error(
				"failed to withdraw delegation rewards",
				"delegator", delAddr.String(),
				"validator", valAddr.String(),
				"error", err,
			)
		}
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
		delAddr, err := sdk.AccAddressFromBech32(del.DelegatorAddress) //nolint:staticcheck // DelegatorAddress retained for SDK compatibility during reward reset.
		if err != nil {
			panic(err)
		}
		err = app.DistrKeeper.SetDelegatorStartingInfo(ctx, valAddr, delAddr, distrtypes.NewDelegatorStartingInfo(2, math.LegacyOneDec(), 0))
		if err != nil {
			panic(err)
		}
	}
}

// setupIBCCapabilities initializes IBC capability keeper and creates scoped keepers
// for IBC modules (transfer, compute, dex, oracle). This must be called before
// initializing the IBC keeper and sealing capabilities.
//
//nolint:gocritic // multiple return values required to wire all scoped keepers.
func (app *PAWApp) setupIBCCapabilities(
	appCodec codec.BinaryCodec,
	keys map[string]*storetypes.KVStoreKey,
	memKeys map[string]*storetypes.MemoryStoreKey,
) (
	*capabilitykeeper.Keeper,
	capabilitykeeper.ScopedKeeper, // IBC
	capabilitykeeper.ScopedKeeper, // Transfer
	capabilitykeeper.ScopedKeeper, // Compute
	capabilitykeeper.ScopedKeeper, // DEX
	capabilitykeeper.ScopedKeeper, // Oracle
) {
	// Initialize capability keeper and seal the capability keeper so all persistent capabilities
	// are loaded in-memory and prevent any further modules from creating scoped sub-keepers.
	capabilityKeeper := capabilitykeeper.NewKeeper(
		appCodec,
		keys[capabilitytypes.StoreKey],
		memKeys[capabilitytypes.MemStoreKey],
	)

	// Create scoped keepers for IBC modules
	// Each IBC module needs its own scoped capability keeper to manage port bindings
	scopedIBCKeeper := capabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	scopedTransferKeeper := capabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	scopedComputeKeeper := capabilityKeeper.ScopeToModule(computetypes.ModuleName)
	scopedDEXKeeper := capabilityKeeper.ScopeToModule(dextypes.ModuleName)
	scopedOracleKeeper := capabilityKeeper.ScopeToModule(oracletypes.ModuleName)

	return capabilityKeeper,
		scopedIBCKeeper,
		scopedTransferKeeper,
		scopedComputeKeeper,
		scopedDEXKeeper,
		scopedOracleKeeper
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
	// IBC modules
	paramsKeeper.Subspace(ibcexported.ModuleName)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	// PAW custom modules
	paramsKeeper.Subspace(dextypes.ModuleName)
	paramsKeeper.Subspace(computetypes.ModuleName)
	paramsKeeper.Subspace(oracletypes.ModuleName)

	return paramsKeeper
}

// setupUpgradeHandlers configures upgrade handlers for all planned chain upgrades.
// Each upgrade handler defines the migration logic for transitioning from one version to another.
func (app *PAWApp) setupUpgradeHandlers() {
	// Register v1.1.0 upgrade handler
	app.setupV1_1_0Upgrade()

	// Register v1.2.0 upgrade handler
	app.setupV1_2_0Upgrade()

	// Register v1.3.0 upgrade handler
	app.setupV1_3_0Upgrade()
}

// setupV1_1_0Upgrade configures the v1.1.0 upgrade handler.
// This upgrade includes:
// - Compute module: Add missing escrow timeout indexes, initialize circuit breakers, migrate nonce storage
// - DEX module: Fix reentrancy guards, add circuit breakers, migrate flash loan protection, add pool metadata
// - Oracle module: Migrate price format, initialize miss counters, add vote periods
func (app *PAWApp) setupV1_1_0Upgrade() {
	const upgradeName = "v1.1.0"

	app.UpgradeKeeper.SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info("Running v1.1.0 upgrade handler",
				"upgrade_name", plan.Name,
				"upgrade_height", plan.Height,
			)

			toVM, err := app.mm.RunMigrations(ctx, app.configurator, fromVM)
			if err != nil {
				return nil, err
			}

			app.Logger().Info("v1.1.0 upgrade completed successfully")
			return toVM, nil
		},
	)
}

// setupV1_2_0Upgrade configures the v1.2.0 upgrade handler.
// This upgrade is a placeholder for future upgrades.
func (app *PAWApp) setupV1_2_0Upgrade() {
	const upgradeName = "v1.2.0"

	app.UpgradeKeeper.SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info("Running v1.2.0 upgrade handler",
				"upgrade_name", plan.Name,
				"upgrade_height", plan.Height,
			)

			// Run migrations for all modules
			toVM, err := app.mm.RunMigrations(ctx, app.configurator, fromVM)
			if err != nil {
				return nil, err
			}

			app.Logger().Info("v1.2.0 upgrade completed successfully")
			return toVM, nil
		},
	)
}

// setupV1_3_0Upgrade configures the v1.3.0 upgrade handler.
// This upgrade is a placeholder for future upgrades.
func (app *PAWApp) setupV1_3_0Upgrade() {
	const upgradeName = "v1.3.0"

	app.UpgradeKeeper.SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info("Running v1.3.0 upgrade handler",
				"upgrade_name", plan.Name,
				"upgrade_height", plan.Height,
			)

			// Run migrations for all modules
			toVM, err := app.mm.RunMigrations(ctx, app.configurator, fromVM)
			if err != nil {
				return nil, err
			}

			app.Logger().Info("v1.3.0 upgrade completed successfully")
			return toVM, nil
		},
	)
}

// setupUpgradeStoreLoaders configures store loaders to handle store upgrades during chain upgrades.
// This function checks if an upgrade is scheduled and configures the appropriate store changes
// (adding new module stores, removing old ones) for each upgrade.
func (app *PAWApp) setupUpgradeStoreLoaders() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	// If no upgrade is scheduled or the upgrade height is skipped, return early
	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	// Configure store upgrades based on the upgrade name
	var storeUpgrades *storetypes.StoreUpgrades

	switch upgradeInfo.Name {
	case "v1.1.0":
		// v1.1.0 doesn't add or remove any stores
		storeUpgrades = &storetypes.StoreUpgrades{
			Added:   []string{},
			Deleted: []string{},
		}
	case "v1.2.0":
		// v1.2.0 placeholder for future store changes
		storeUpgrades = &storetypes.StoreUpgrades{
			Added:   []string{},
			Deleted: []string{},
		}
	case "v1.3.0":
		// v1.3.0 placeholder for future store changes
		storeUpgrades = &storetypes.StoreUpgrades{
			Added:   []string{},
			Deleted: []string{},
		}
	}

	// If store upgrades are defined, configure the store loader
	if storeUpgrades != nil {
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, storeUpgrades))
	}
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

// GetBasicManager returns the module basic manager, constructing it once using a
// lightweight in-memory app to ensure module basics are fully wired with their
// codecs. The manager is cached for reuse across CLI, tests, and app
// initialization flows.
func GetBasicManager() module.BasicManager {
	moduleBasicsOnce.Do(func() {
		tempApp := NewPAWApp(log.NewNopLogger(), dbm.NewMemDB(), io.Discard, false, emptyAppOptions{})
		moduleBasics = tempApp.BasicModuleManager
	})

	ModuleBasics = moduleBasics
	return moduleBasics
}

// setModuleBasics caches the basic manager if it hasn't been initialized yet.
func setModuleBasics(bm module.BasicManager) {
	if bm == nil {
		return
	}
	if moduleBasics == nil {
		moduleBasics = bm
	}
}

type emptyAppOptions struct{}

// Get implements AppOptions.
func (emptyAppOptions) Get(_ string) interface{} { return nil }

// module account permissions
var maccPerms = map[string][]string{
	authtypes.FeeCollectorName:     nil,
	distrtypes.ModuleName:          nil,
	minttypes.ModuleName:           {authtypes.Minter},
	stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
	stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
	govtypes.ModuleName:            {authtypes.Burner},
	// IBC modules
	ibctransfertypes.ModuleName: {authtypes.Minter, authtypes.Burner},
	// PAW custom modules
	dextypes.ModuleName:     {authtypes.Minter, authtypes.Burner},
	computetypes.ModuleName: {authtypes.Minter, authtypes.Burner},
	oracletypes.ModuleName:  nil,
}
