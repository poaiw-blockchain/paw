package simulation_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"

	"github.com/paw-chain/paw/app"
)

// TestFullAppSimulation runs a full simulation of the PAW chain
func TestFullAppSimulation(t *testing.T) {
	config := simcli.NewConfigFromFlags()
	config.ChainID = "paw-sim"

	db, dir, logger, skip, err := simtestutil.SetupSimulation(
		config,
		"paw-simulation",
		"Simulation",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	if skip {
		t.Skip("skipping application simulation")
	}
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.RemoveAll(dir))
	}()

	encCfg := app.MakeEncodingConfig()

	pawApp := app.NewPAWApp(
		logger,
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID(config.ChainID),
	)

	require.Equal(t, "paw", pawApp.Name())

	// Run simulation
	_, simParams, simErr := simulation.SimulateFromSeed(
		t,
		os.Stdout,
		pawApp.BaseApp,
		simtestutil.AppStateFn(
			pawApp.AppCodec(),
			pawApp.SimulationManager(),
			app.NewDefaultGenesisState(config.ChainID),
		),
		simtypes.RandomAccounts,
		simtestutil.SimulationOperations(pawApp, pawApp.AppCodec(), config),
		pawApp.ModuleAccountAddrs(),
		config,
		pawApp.AppCodec(),
	)

	// Export and import state
	err = simtestutil.CheckExportSimulation(pawApp, config, simParams)
	require.NoError(t, err)
	require.NoError(t, simErr)

	if config.Commit {
		simtestutil.PrintStats(db)
	}
}

// TestAppStateDeterminism tests that the app produces the same state given the same seed
func TestAppStateDeterminism(t *testing.T) {
	if !simcli.FlagEnabledValue {
		t.Skip("skipping application simulation")
	}

	config := simcli.NewConfigFromFlags()
	config.InitialBlockHeight = 1
	config.ExportParamsPath = ""
	config.OnOperation = false
	config.AllInvariants = false
	config.ChainID = "paw-sim"

	numSeeds := 3
	numTimesToRunPerSeed := 5

	appHashList := make([]json.RawMessage, numTimesToRunPerSeed)
	appOptions := make(simtestutil.AppOptionsMap, 0)

	for i := 0; i < numSeeds; i++ {
		config.Seed = rand.Int63()

		for j := 0; j < numTimesToRunPerSeed; j++ {
			var logger log.Logger
			if simcli.FlagVerboseValue {
				logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
			} else {
				logger = log.NewNopLogger()
			}

			db := dbm.NewMemDB()
			encCfg := app.MakeEncodingConfig()

			pawApp := app.NewPAWApp(
				logger,
				db,
				nil,
				true,
				map[int64]bool{},
				app.DefaultNodeHome,
				simcli.FlagPeriodValue,
				encCfg,
				app.GetEnabledProposals(),
				baseapp.SetChainID(config.ChainID),
			)

			fmt.Printf(
				"running non-determinism simulation; seed %d: %d/%d, attempt: %d/%d\n",
				config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
			)

			_, _, err := simulation.SimulateFromSeed(
				t,
				os.Stdout,
				pawApp.BaseApp,
				simtestutil.AppStateFn(
					pawApp.AppCodec(),
					pawApp.SimulationManager(),
					app.NewDefaultGenesisState(config.ChainID),
				),
				simtypes.RandomAccounts,
				simtestutil.SimulationOperations(pawApp, pawApp.AppCodec(), config),
				pawApp.ModuleAccountAddrs(),
				config,
				pawApp.AppCodec(),
			)
			require.NoError(t, err)

			if config.Commit {
				simtestutil.PrintStats(db)
			}

			appHash := pawApp.LastCommitID().Hash
			appHashList[j] = appHash

			if j != 0 {
				require.Equal(
					t, string(appHashList[0]), string(appHashList[j]),
					"non-determinism in seed %d: %d/%d, attempt: %d/%d\n",
					config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
				)
			}
		}
	}
}

// TestAppSimulationAfterImport tests that app state can be imported and simulated
func TestAppSimulationAfterImport(t *testing.T) {
	config := simcli.NewConfigFromFlags()
	config.ChainID = "paw-sim"

	db, dir, logger, skip, err := simtestutil.SetupSimulation(
		config,
		"paw-simulation-after-import",
		"SimulationAfterImport",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	if skip {
		t.Skip("skipping application simulation after import")
	}
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.RemoveAll(dir))
	}()

	encCfg := app.MakeEncodingConfig()

	pawApp := app.NewPAWApp(
		logger,
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID(config.ChainID),
	)

	require.Equal(t, "paw", pawApp.Name())

	// Run simulation
	stopEarly, simParams, simErr := simulation.SimulateFromSeed(
		t,
		os.Stdout,
		pawApp.BaseApp,
		simtestutil.AppStateFn(
			pawApp.AppCodec(),
			pawApp.SimulationManager(),
			app.NewDefaultGenesisState(config.ChainID),
		),
		simtypes.RandomAccounts,
		simtestutil.SimulationOperations(pawApp, pawApp.AppCodec(), config),
		pawApp.ModuleAccountAddrs(),
		config,
		pawApp.AppCodec(),
	)

	require.NoError(t, simErr)

	if config.Commit {
		simtestutil.PrintStats(db)
	}

	if stopEarly {
		t.Log("simulation stopped early")
	}

	err = simtestutil.CheckExportSimulation(pawApp, config, simParams)
	require.NoError(t, err)

	// Export the app state
	exported, err := pawApp.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)

	// Create new app and import state
	db2 := dbm.NewMemDB()
	pawApp2 := app.NewPAWApp(
		logger,
		db2,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		simcli.FlagPeriodValue,
		encCfg,
		app.GetEnabledProposals(),
		baseapp.SetChainID(config.ChainID),
	)

	var genesisState app.GenesisState
	err = json.Unmarshal(exported.AppState, &genesisState)
	require.NoError(t, err)

	ctxB := pawApp2.NewContext(true, tmproto.Header{Height: pawApp.LastBlockHeight()})
	_, err = pawApp2.ModuleManager.InitGenesis(ctxB, pawApp2.AppCodec(), genesisState)
	require.NoError(t, err)

	// Run simulation on imported state
	_, _, err = simulation.SimulateFromSeed(
		t,
		os.Stdout,
		pawApp2.BaseApp,
		simtestutil.AppStateFn(
			pawApp2.AppCodec(),
			pawApp2.SimulationManager(),
			genesisState,
		),
		simtypes.RandomAccounts,
		simtestutil.SimulationOperations(pawApp2, pawApp2.AppCodec(), config),
		pawApp2.ModuleAccountAddrs(),
		config,
		pawApp2.AppCodec(),
	)
	require.NoError(t, err)
}

// TestSimulationWithInvariants runs simulation and checks all invariants
func TestSimulationWithInvariants(t *testing.T) {
	config := simcli.NewConfigFromFlags()
	config.ChainID = "paw-sim"
	config.AllInvariants = true // Enable all invariants

	db, dir, logger, skip, err := simtestutil.SetupSimulation(
		config,
		"paw-simulation-invariants",
		"SimulationInvariants",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	if skip {
		t.Skip("skipping application simulation with invariants")
	}
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.RemoveAll(dir))
	}()

	encCfg := app.MakeEncodingConfig()

	pawApp := app.NewPAWApp(
		logger,
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID(config.ChainID),
	)

	// Run simulation with invariant checks
	_, simParams, simErr := simulation.SimulateFromSeed(
		t,
		os.Stdout,
		pawApp.BaseApp,
		simtestutil.AppStateFn(
			pawApp.AppCodec(),
			pawApp.SimulationManager(),
			app.NewDefaultGenesisState(config.ChainID),
		),
		simtypes.RandomAccounts,
		simtestutil.SimulationOperations(pawApp, pawApp.AppCodec(), config),
		pawApp.ModuleAccountAddrs(),
		config,
		pawApp.AppCodec(),
	)

	require.NoError(t, simErr)

	if config.Commit {
		simtestutil.PrintStats(db)
	}

	err = simtestutil.CheckExportSimulation(pawApp, config, simParams)
	require.NoError(t, err)
}

// BenchmarkSimulation runs a benchmark of the simulation
func BenchmarkSimulation(b *testing.B) {
	config := simcli.NewConfigFromFlags()
	config.ChainID = "paw-sim"

	FlagSeedValue := rand.Int63()
	config.Seed = FlagSeedValue

	for n := 0; n < b.N; n++ {
		db := dbm.NewMemDB()
		encCfg := app.MakeEncodingConfig()

		pawApp := app.NewPAWApp(
			log.NewNopLogger(),
			db,
			nil,
			true,
			map[int64]bool{},
			app.DefaultNodeHome,
			simcli.FlagPeriodValue,
			encCfg,
			app.GetEnabledProposals(),
			baseapp.SetChainID(config.ChainID),
		)

		_, _, err := simulation.SimulateFromSeed(
			b,
			os.Stdout,
			pawApp.BaseApp,
			simtestutil.AppStateFn(
				pawApp.AppCodec(),
				pawApp.SimulationManager(),
				app.NewDefaultGenesisState(config.ChainID),
			),
			simtypes.RandomAccounts,
			simtestutil.SimulationOperations(pawApp, pawApp.AppCodec(), config),
			pawApp.ModuleAccountAddrs(),
			config,
			pawApp.AppCodec(),
		)
		require.NoError(b, err)

		db.Close()
	}
}
