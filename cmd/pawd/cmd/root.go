package cmd

import (
	"errors"
	"io"
	"os"
	"sync"

	"cosmossdk.io/log"
	cmtcfg "github.com/cometbft/cometbft/config"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"

	// "github.com/cosmos/cosmos-sdk/client/snapshot"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/paw-chain/paw/app"
)

// NewRootCmd creates a new root command for pawd. It is called once in the
// main function.
func NewRootCmd(addHomeFlag bool) *cobra.Command {
	// Ensure SDK bech32 prefixes are configured prior to CLI usage.
	initSDKConfig()

	encodingConfig := app.MakeEncodingConfig()

	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithHomeDir(app.DefaultNodeHome).
		WithViper("")

	rootCmd := &cobra.Command{
		Use:   "pawd",
		Short: "PAW Blockchain Daemon",
		Long: `PAW is a manageable layer-1 blockchain designed for AI workload verification
with a built-in DEX, secure API compute aggregation, and multi-device wallet support.`,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx = initClientCtx.WithCmdContext(cmd.Context())
			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			// Ensure TxConfig is always populated; ReadPersistentCommandFlags/ReadFromClientConfig
			// can overwrite the client context and nil out the TxConfig, which causes CLI tx
			// commands to panic when preparing the factory.
			initClientCtx = initClientCtx.WithTxConfig(encodingConfig.TxConfig)
			if initClientCtx.TxConfig == nil {
				reboundCfg := app.MakeEncodingConfig()
				initClientCtx = initClientCtx.WithTxConfig(reboundCfg.TxConfig)
			}

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := initAppConfig()
			customCometConfig := initCometBFTConfig()

			return server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customCometConfig)
		},
	}

	initRootCmd(rootCmd, encodingConfig, addHomeFlag)

	return rootCmd
}

func initRootCmd(rootCmd *cobra.Command, encodingConfig app.EncodingConfig, addHomeFlag bool) {
	if addHomeFlag && rootCmd.PersistentFlags().Lookup(flags.FlagHome) == nil {
		rootCmd.PersistentFlags().String(flags.FlagHome, app.DefaultNodeHome, "directory for config and data")
	}
	if rootCmd.PersistentFlags().Lookup(flags.FlagChainID) == nil {
		rootCmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")
	}

	rootCmd.AddCommand(
		InitCmd(app.ModuleBasics, app.DefaultNodeHome),
		genutilcli.ValidateGenesisCmd(app.ModuleBasics),
		AddGenesisAccountCmd(app.DefaultNodeHome),
		GenTxCmd(app.ModuleBasics, encodingConfig.TxConfig, banktypes.GenesisBalancesIterator{}, app.DefaultNodeHome),
		CollectGenTxsCmd(app.ModuleBasics, app.DefaultNodeHome, banktypes.GenesisBalancesIterator{}, genutiltypes.DefaultMessageValidator),
		debug.Cmd(),
		// Configuration is managed via `pawd config` in v0.50; re-enable here once upstream exposes a replacement helper.
		pruning.Cmd(newApp, app.DefaultNodeHome),
		// snapshot.Cmd(newApp),
	)

	server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, appExport, addModuleInitFlags)

	// add keybase, auxiliary RPC, query, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		queryCommand(),
		txCommand(),
		newKeysCmd(false), // PAW custom keys command with BIP39 support (home flag provided by root)
	)
}

func addModuleInitFlags(startCmd *cobra.Command) {
	crisis.AddModuleInitFlags(startCmd)
}

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.ValidatorCommand(),
		server.QueryBlockCmd(),
		authcmd.QueryTxsByEventsCmd(),
		server.QueryBlocksCmd(),
		authcmd.QueryTxCmd(),
		server.QueryBlockResultsCmd(),
	)

	app.ModuleBasics.AddQueryCommands(cmd)

	return cmd
}

func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		authcmd.GetSimulateCmd(),
	)

	app.ModuleBasics.AddTxCommands(cmd)

	return cmd
}

func newApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	baseappOptions := server.DefaultBaseappOptions(appOpts)
	return app.NewPAWApp(
		logger,
		db,
		traceStore,
		true,
		appOpts,
		baseappOptions...,
	)
}

func appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	var pawApp *app.PAWApp
	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home not set")
	}

	viperAppOpts, ok := appOpts.(*viper.Viper)
	if !ok {
		return servertypes.ExportedApp{}, errors.New("appOpts is not viper.Viper")
	}

	// overwrite the FlagInvCheckPeriod
	viperAppOpts.Set(server.FlagInvCheckPeriod, 1)
	appOpts = viperAppOpts

	if height != -1 {
		pawApp = app.NewPAWApp(logger, db, traceStore, false, appOpts)

		if err := pawApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	} else {
		pawApp = app.NewPAWApp(logger, db, traceStore, true, appOpts)
	}

	return pawApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

// initSDKConfig initializes the SDK config with PAW chain prefix
var sdkConfigOnce sync.Once

func initSDKConfig() {
	sdkConfigOnce.Do(func() {
		app.SetConfig()
	})
}

// initAppConfig helps to override default appConfig template and configs.
func initAppConfig() (configTemplate string, cfg interface{}) {
	type CustomAppConfig struct {
		serverconfig.Config
	}

	// Optionally allow the chain developer to overwrite the SDK's default
	// server config.
	srvCfg := serverconfig.DefaultConfig()
	// The SDK's default minimum gas price is set to "" (empty value) inside
	// app.toml. If left empty by validators, the node will halt on startup.
	// However, the chain developer can set a default app.toml value for their
	// validators here.
	//
	// In summary:
	// - if you leave srvCfg.MinGasPrices = "", all validators MUST tweak their
	//   own app.toml config,
	// - if you set srvCfg.MinGasPrices non-empty, validators CAN tweak their
	//   own app.toml to override, or use this default value.
	//
	// In PAW, we set the min gas prices to 0.001upaw
	srvCfg.MinGasPrices = app.DefaultMinGasPrice.String()

	// Enable API and gRPC by default for full nodes and query endpoints.
	// Validators running dedicated sentry nodes may disable API in their config.
	srvCfg.API.Enable = true
	srvCfg.API.Swagger = false
	srvCfg.GRPC.Enable = true

	customAppConfig := CustomAppConfig{
		Config: *srvCfg,
	}

	customAppTemplate := serverconfig.DefaultConfigTemplate

	return customAppTemplate, customAppConfig
}

// initCometBFTConfig helps to override default CometBFT Config values.
func initCometBFTConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()

	// Set custom timeout values for 4-second block time
	cfg.Consensus.TimeoutPropose = 3000 // 3 seconds
	cfg.Consensus.TimeoutProposeDelta = 500
	cfg.Consensus.TimeoutPrevote = 1000 // 1 second
	cfg.Consensus.TimeoutPrevoteDelta = 500
	cfg.Consensus.TimeoutPrecommit = 1000 // 1 second
	cfg.Consensus.TimeoutPrecommitDelta = 500
	cfg.Consensus.TimeoutCommit = 0

	return cfg
}
