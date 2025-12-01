package simulation

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"

	"github.com/paw-chain/paw/app"
)

func newSimConfig(chainID string) simtypes.Config {
	return simtypes.Config{
		ChainID:            chainID,
		Seed:               simcli.DefaultSeedValue,
		InitialBlockHeight: 1,
		NumBlocks:          200,
		BlockSize:          200,
		DBBackend:          "goleveldb",
	}
}

func init() {
	simcli.GetSimulatorFlags()
}

// TestFullAppSimulation runs the full application simulation for 500 blocks
// This is the main simulation test that exercises all modules with random operations
func TestFullAppSimulation(t *testing.T) {
	config := newSimConfig("paw-sim-1")
	config.NumBlocks = 500
	config.BlockSize = 200 // Operations per block
	config.Seed = 42       // Fixed seed for deterministic testing
	config.OnOperation = false
	config.AllInvariants = true
	config.Commit = true

	db, dir, logger, skip, err := simtestutil.SetupSimulation(
		config,
		"leveldb-app-sim",
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

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = dir
	appOptions[server.FlagInvCheckPeriod] = 1

	// Create simulation app
	simApp := app.NewPAWApp(
		logger,
		db,
		nil,
		true,
		appOptions,
		baseapp.SetChainID(config.ChainID),
	)
	require.Equal(t, "paw", simApp.Name())

	// Run simulation
	_, simParams, simErr := simulation.SimulateFromSeed(
		t,
		os.Stdout,
		simApp.BaseApp,
		AppStateFn(simApp.AppCodec(), simApp.BasicModuleManager),
		simtypes.RandomAccounts,
		SimulationOperations(simApp, simApp.AppCodec(), config),
		BlockedAddresses(),
		config,
		simApp.AppCodec(),
	)

	// Export state after simulation
	err = simtestutil.CheckExportSimulation(simApp, config, simParams)
	require.NoError(t, err)
	require.NoError(t, simErr)

	if config.Commit {
		simtestutil.PrintStats(db)
	}
}

// TestAppStateDeterminism verifies that the application state is deterministic
// Same seed should produce the same final state
func TestAppStateDeterminism(t *testing.T) {
	if !simcli.FlagEnabledValue {
		t.Skip("skipping application simulation")
	}

	config := newSimConfig("paw-sim-1")
	config.NumBlocks = 100
	config.BlockSize = 200
	config.Seed = 42 // Fixed seed
	config.OnOperation = false
	config.AllInvariants = false // Disable for speed
	config.Commit = true

	numSeeds := 3
	numTimesToRunPerSeed := 5

	appHashList := make([]json.RawMessage, numTimesToRunPerSeed)
	appOptions := make(simtestutil.AppOptionsMap, 0)

	for i := 0; i < numSeeds; i++ {
		config.Seed = int64(i)

		for j := 0; j < numTimesToRunPerSeed; j++ {
			db, dir, logger, _, err := simtestutil.SetupSimulation(
				config,
				"leveldb-app-sim",
				"Simulation",
				simcli.FlagVerboseValue,
				simcli.FlagEnabledValue,
			)
			require.NoError(t, err)

			appOptions[flags.FlagHome] = dir
			appOptions[server.FlagInvCheckPeriod] = 1

			simApp := app.NewPAWApp(
				logger,
				db,
				nil,
				true,
				appOptions,
				baseapp.SetChainID(config.ChainID),
			)

			fmt.Printf(
				"running non-determinism simulation; seed %d: %d/%d, attempt: %d/%d\n",
				config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
			)

			_, _, err = simulation.SimulateFromSeed(
				t,
				os.Stdout,
				simApp.BaseApp,
				AppStateFn(simApp.AppCodec(), simApp.BasicModuleManager),
				simtypes.RandomAccounts,
				SimulationOperations(simApp, simApp.AppCodec(), config),
				BlockedAddresses(),
				config,
				simApp.AppCodec(),
			)
			require.NoError(t, err)

			if config.Commit {
				simtestutil.PrintStats(db)
			}

			appHash := simApp.LastCommitID().Hash
			appHashList[j] = appHash

			// Cleanup
			require.NoError(t, db.Close())
			require.NoError(t, os.RemoveAll(dir))
		}

		// Check all app hashes are the same for this seed
		for k := 1; k < numTimesToRunPerSeed; k++ {
			require.Equal(
				t,
				appHashList[0],
				appHashList[k],
				"non-determinism in seed %d: %d/%d, attempt: %d/%d\n",
				config.Seed, i+1, numSeeds, k+1, numTimesToRunPerSeed,
			)
		}
	}
}

// TestAppImportExport verifies that the application can export and import state
func TestAppImportExport(t *testing.T) {
	config := newSimConfig("paw-sim-1")
	config.NumBlocks = 100
	config.BlockSize = 25
	config.Seed = 42
	config.OnOperation = false
	config.AllInvariants = true
	config.Commit = true

	db, dir, logger, skip, err := simtestutil.SetupSimulation(
		config,
		"leveldb-app-sim",
		"Simulation",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	if skip {
		t.Skip("skipping application import/export simulation")
	}
	require.NoError(t, err)

	defer func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.RemoveAll(dir))
	}()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = dir
	appOptions[server.FlagInvCheckPeriod] = 1

	simApp := app.NewPAWApp(
		logger,
		db,
		nil,
		true,
		appOptions,
		baseapp.SetChainID(config.ChainID),
	)
	require.Equal(t, "paw", simApp.Name())

	// Run simulation
	_, _, simErr := simulation.SimulateFromSeed(
		t,
		os.Stdout,
		simApp.BaseApp,
		AppStateFn(simApp.AppCodec(), simApp.BasicModuleManager),
		simtypes.RandomAccounts,
		SimulationOperations(simApp, simApp.AppCodec(), config),
		BlockedAddresses(),
		config,
		simApp.AppCodec(),
	)
	require.NoError(t, simErr)

	if config.Commit {
		simtestutil.PrintStats(db)
	}

	fmt.Printf("exporting genesis...\n")

	exported, err := simApp.ExportAppStateAndValidators(false, []string{}, []string{})
	require.NoError(t, err)

	fmt.Printf("importing genesis...\n")

	newDB, newDir, _, _, err := simtestutil.SetupSimulation(
		config,
		"leveldb-app-sim-2",
		"Simulation-2",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		require.NoError(t, newDB.Close())
		require.NoError(t, os.RemoveAll(newDir))
	}()

	newAppOptions := make(simtestutil.AppOptionsMap, 0)
	newAppOptions[flags.FlagHome] = newDir
	newAppOptions[server.FlagInvCheckPeriod] = 1

	newApp := app.NewPAWApp(
		log.NewNopLogger(),
		newDB,
		nil,
		true,
		newAppOptions,
		baseapp.SetChainID(config.ChainID),
	)
	require.Equal(t, "paw", newApp.Name())

	var genesisState app.GenesisState
	err = json.Unmarshal(exported.AppState, &genesisState)
	require.NoError(t, err)

	ctxA := simApp.NewContextLegacy(true, cmtproto.Header{Height: simApp.LastBlockHeight()})
	ctxB := newApp.NewContextLegacy(true, cmtproto.Header{Height: newApp.LastBlockHeight()})

	_, err = newApp.ModuleManager().InitGenesis(ctxB, simApp.AppCodec(), genesisState)
	require.NoError(t, err)

	err = newApp.StoreConsensusParams(ctxB, exported.ConsensusParams)
	require.NoError(t, err)

	fmt.Printf("comparing stores...\n")

	type StoreKeysPrefixes struct {
		A        storetypes.StoreKey
		B        storetypes.StoreKey
		Prefixes [][]byte
	}

	storeKeysPrefixes := []StoreKeysPrefixes{
		{simApp.GetKey("bank"), newApp.GetKey("bank"), [][]byte{}},
		{simApp.GetKey("staking"), newApp.GetKey("staking"), [][]byte{}},
		{simApp.GetKey("dex"), newApp.GetKey("dex"), [][]byte{}},
		{simApp.GetKey("oracle"), newApp.GetKey("oracle"), [][]byte{}},
		{simApp.GetKey("compute"), newApp.GetKey("compute"), [][]byte{}},
	}

	for _, skp := range storeKeysPrefixes {
		storeA := ctxA.KVStore(skp.A)
		storeB := ctxB.KVStore(skp.B)

		failedKVAs, failedKVBs := simtestutil.DiffKVStores(storeA, storeB, skp.Prefixes)
		require.Equal(t, len(failedKVAs), len(failedKVBs), "unequal sets of key-values to compare")

		fmt.Printf("compared %d different key/value pairs between %s and %s\n", len(failedKVAs), skp.A, skp.B)
		require.Equal(t, 0, len(failedKVAs), simtestutil.GetSimulationLog(skp.A.Name(), simApp.SimulationManager().StoreDecoders, failedKVAs, failedKVBs))
	}
}

// TestAppSimulationAfterImport verifies simulation works after importing state
func TestAppSimulationAfterImport(t *testing.T) {
	config := newSimConfig("paw-sim-1")
	config.NumBlocks = 100
	config.BlockSize = 25
	config.Seed = 42
	config.OnOperation = false
	config.AllInvariants = true
	config.Commit = true

	db, dir, logger, skip, err := simtestutil.SetupSimulation(
		config,
		"leveldb-app-sim",
		"Simulation",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	if skip {
		t.Skip("skipping simulation after import")
	}
	require.NoError(t, err)

	defer func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.RemoveAll(dir))
	}()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = dir
	appOptions[server.FlagInvCheckPeriod] = 1

	simApp := app.NewPAWApp(
		logger,
		db,
		nil,
		true,
		appOptions,
		baseapp.SetChainID(config.ChainID),
	)
	require.Equal(t, "paw", simApp.Name())

	// Run initial simulation
	stopEarly, _, simErr := simulation.SimulateFromSeed(
		t,
		os.Stdout,
		simApp.BaseApp,
		AppStateFn(simApp.AppCodec(), simApp.BasicModuleManager),
		simtypes.RandomAccounts,
		SimulationOperations(simApp, simApp.AppCodec(), config),
		BlockedAddresses(),
		config,
		simApp.AppCodec(),
	)
	require.NoError(t, simErr)

	if config.Commit {
		simtestutil.PrintStats(db)
	}

	if stopEarly {
		fmt.Println("can't export or import a zero-validator genesis, exiting test...")
		return
	}

	fmt.Printf("exporting genesis...\n")

	exported, err := simApp.ExportAppStateAndValidators(true, []string{}, []string{})
	require.NoError(t, err)
	require.NotEmpty(t, exported.AppState)

	fmt.Printf("importing genesis...\n")

	newDB, newDir, _, _, err := simtestutil.SetupSimulation(
		config,
		"leveldb-app-sim-2",
		"Simulation-2",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, newDB.Close())
		require.NoError(t, os.RemoveAll(newDir))
	}()

	newAppOptions := make(simtestutil.AppOptionsMap, 0)
	newAppOptions[flags.FlagHome] = newDir
	newAppOptions[server.FlagInvCheckPeriod] = 1

	newApp := app.NewPAWApp(
		log.NewNopLogger(),
		newDB,
		nil,
		true,
		newAppOptions,
		baseapp.SetChainID(config.ChainID),
	)
	require.Equal(t, "paw", newApp.Name())

	_, _, err = simulation.SimulateFromSeed(
		t,
		os.Stdout,
		newApp.BaseApp,
		AppStateFn(newApp.AppCodec(), newApp.BasicModuleManager),
		simtypes.RandomAccounts,
		SimulationOperations(newApp, newApp.AppCodec(), config),
		BlockedAddresses(),
		config,
		newApp.AppCodec(),
	)
	require.NoError(t, err)
}

// TestMultiSeedFullSimulation runs full simulation with different random seeds
func TestMultiSeedFullSimulation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-seed simulation in short mode")
	}

	seeds := []int64{1, 2, 3, 5, 8, 13, 21}

	for _, seed := range seeds {
		seed := seed // capture range variable
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			t.Parallel()

			config := newSimConfig(fmt.Sprintf("paw-sim-%d", seed))
			config.NumBlocks = 200
			config.BlockSize = 100
			config.Seed = seed
			config.OnOperation = false
			config.AllInvariants = true
			config.Commit = true

			db, dir, logger, _, err := simtestutil.SetupSimulation(
				config,
				"leveldb-app-sim",
				"Simulation",
				simcli.FlagVerboseValue,
				simcli.FlagEnabledValue,
			)
			require.NoError(t, err)

			defer func() {
				require.NoError(t, db.Close())
				require.NoError(t, os.RemoveAll(dir))
			}()

			appOptions := make(simtestutil.AppOptionsMap, 0)
			appOptions[flags.FlagHome] = dir
			appOptions[server.FlagInvCheckPeriod] = 1

			simApp := app.NewPAWApp(
				logger,
				db,
				nil,
				true,
				appOptions,
				baseapp.SetChainID(config.ChainID),
			)

			_, simParams, simErr := simulation.SimulateFromSeed(
				t,
				os.Stdout,
				simApp.BaseApp,
				AppStateFn(simApp.AppCodec(), simApp.BasicModuleManager),
				simtypes.RandomAccounts,
				SimulationOperations(simApp, simApp.AppCodec(), config),
				BlockedAddresses(),
				config,
				simApp.AppCodec(),
			)

			require.NoError(t, simErr)
			require.NotNil(t, simParams)
		})
	}
}

// BenchmarkFullAppSimulation benchmarks the full application simulation
func BenchmarkFullAppSimulation(b *testing.B) {
	config := newSimConfig("paw-sim-1")
	config.NumBlocks = 500
	config.BlockSize = 200
	config.Seed = 42
	config.OnOperation = false
	config.AllInvariants = false // Disable for benchmark speed
	config.Commit = true

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		db, dir, logger, _, err := simtestutil.SetupSimulation(
			config,
			"leveldb-app-sim",
			"Simulation",
			simcli.FlagVerboseValue,
			simcli.FlagEnabledValue,
		)
		require.NoError(b, err)

		appOptions := make(simtestutil.AppOptionsMap, 0)
		appOptions[flags.FlagHome] = dir
		appOptions[server.FlagInvCheckPeriod] = 0 // Disable for speed

		simApp := app.NewPAWApp(
			logger,
			db,
			nil,
			true,
			appOptions,
			baseapp.SetChainID(config.ChainID),
		)

		_, _, err = simulation.SimulateFromSeed(
			b,
			os.Stdout,
			simApp.BaseApp,
			AppStateFn(simApp.AppCodec(), simApp.BasicModuleManager),
			simtypes.RandomAccounts,
			SimulationOperations(simApp, simApp.AppCodec(), config),
			BlockedAddresses(),
			config,
			simApp.AppCodec(),
		)
		require.NoError(b, err)

		require.NoError(b, db.Close())
		require.NoError(b, os.RemoveAll(dir))
	}
}
