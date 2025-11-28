package simulation

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/paw-chain/paw/app"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
	computetypes "github.com/paw-chain/paw/x/compute/types"
)

// AppStateFn returns the initial application state using a genesis or the simulation parameters.
// It panics if the user provides files for both of them.
// If a file is not given for the genesis or the sim params, it creates a randomized one.
func AppStateFn(cdc codec.JSONCodec, bm module.BasicManager) simtypes.AppStateFn {
	return func(r *rand.Rand, accs []simtypes.Account, config simtypes.Config,
	) (appState json.RawMessage, simAccs []simtypes.Account, chainID string, genesisTimestamp int64) {

		if config.ExportParamsPath != "" {
			panic("Params export is not supported")
		}

		if config.GenesisFile != "" {
			// Use provided genesis file
			appState, simAccs, chainID, genesisTimestamp = loadGenesisState(cdc, config.GenesisFile)
		} else {
			// Generate random genesis state
			appState, simAccs, chainID, genesisTimestamp = generateGenesisState(r, cdc, accs, config, bm)
		}

		return appState, simAccs, chainID, genesisTimestamp
	}
}

// generateGenesisState generates a random GenesisState for simulation
func generateGenesisState(
	r *rand.Rand,
	cdc codec.JSONCodec,
	accs []simtypes.Account,
	config simtypes.Config,
	bm module.BasicManager,
) (json.RawMessage, []simtypes.Account, string, int64) {

	genesisState := bm.DefaultGenesis(cdc)

	// Customize Bank genesis
	bankGenesis := banktypes.GetGenesisStateFromAppState(cdc, genesisState)
	bankGenesis.Params = banktypes.DefaultParams()

	// Fund simulation accounts
	balances := make([]banktypes.Balance, len(accs))
	for i, acc := range accs {
		balances[i] = banktypes.Balance{
			Address: acc.Address.String(),
			Coins: sdk.NewCoins(
				sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100000000000)), // 100k tokens
				sdk.NewCoin("uatom", math.NewInt(100000000)),
				sdk.NewCoin("uosmo", math.NewInt(100000000)),
			),
		}
	}
	bankGenesis.Balances = balances

	// Update total supply
	bankGenesis.Supply = sdk.NewCoins(
		sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100000000000*int64(len(accs)))),
		sdk.NewCoin("uatom", math.NewInt(100000000*int64(len(accs)))),
		sdk.NewCoin("uosmo", math.NewInt(100000000*int64(len(accs)))),
	)

	genesisState[banktypes.ModuleName] = cdc.MustMarshalJSON(bankGenesis)

	// Customize Staking genesis
	stakingGenesis := stakingtypes.GetGenesisStateFromAppState(cdc, genesisState)
	stakingGenesis.Params.BondDenom = sdk.DefaultBondDenom
	stakingGenesis.Params.UnbondingTime = simtypes.RandTimeDuration(r, 60*60*24*7, 60*60*24*21) // 1-3 weeks
	stakingGenesis.Params.MaxValidators = uint32(simtypes.RandIntBetween(r, 50, 100))
	stakingGenesis.Params.MaxEntries = uint32(simtypes.RandIntBetween(r, 5, 10))
	genesisState[stakingtypes.ModuleName] = cdc.MustMarshalJSON(stakingGenesis)

	// Customize DEX genesis
	dexGenesis := &dextypes.GenesisState{
		Params: randomDEXParams(r),
		Pools:  []dextypes.Pool{},
		NextPoolId: 1,
	}
	genesisState[dextypes.ModuleName] = cdc.MustMarshalJSON(dexGenesis)

	// Customize Oracle genesis
	oracleGenesis := &oracletypes.GenesisState{
		Params: randomOracleParams(r),
		Prices: []oracletypes.Price{},
	}
	genesisState[oracletypes.ModuleName] = cdc.MustMarshalJSON(oracleGenesis)

	// Customize Compute genesis
	computeGenesis := &computetypes.GenesisState{
		Params: randomComputeParams(r),
		Requests: []computetypes.Request{},
		Providers: []computetypes.Provider{},
	}
	genesisState[computetypes.ModuleName] = cdc.MustMarshalJSON(computeGenesis)

	appState, err := json.Marshal(genesisState)
	if err != nil {
		panic(err)
	}

	return appState, accs, config.ChainID, config.GenesisTime
}

// loadGenesisState loads genesis state from file
func loadGenesisState(cdc codec.JSONCodec, genesisFile string) (
	json.RawMessage, []simtypes.Account, string, int64,
) {
	panic("Genesis file loading not implemented for simulation")
}

// RandomizedParams creates randomized parameter changes for param change proposals
func RandomizedParams(r *rand.Rand) []simtypes.LegacyParamChange {
	return []simtypes.LegacyParamChange{
		// DEX params
		{
			Subspace: dextypes.ModuleName,
			Key:      "SwapFee",
			SimValue: func(r *rand.Rand) string {
				return fmt.Sprintf("\"%s\"", randomSwapFee(r))
			},
		},
		{
			Subspace: dextypes.ModuleName,
			Key:      "MinLiquidity",
			SimValue: func(r *rand.Rand) string {
				return fmt.Sprintf("\"%s\"", randomMinLiquidity(r))
			},
		},
		// Oracle params
		{
			Subspace: oracletypes.ModuleName,
			Key:      "VotePeriod",
			SimValue: func(r *rand.Rand) string {
				return fmt.Sprintf("%d", randomVotePeriod(r))
			},
		},
		{
			Subspace: oracletypes.ModuleName,
			Key:      "SlashFraction",
			SimValue: func(r *rand.Rand) string {
				return fmt.Sprintf("\"%s\"", randomSlashFraction(r))
			},
		},
		// Compute params
		{
			Subspace: computetypes.ModuleName,
			Key:      "MinProviderStake",
			SimValue: func(r *rand.Rand) string {
				return fmt.Sprintf("\"%s\"", randomMinProviderStake(r))
			},
		},
	}
}

// Random parameter generators for DEX
func randomDEXParams(r *rand.Rand) dextypes.Params {
	return dextypes.Params{
		SwapFee:            randomSwapFee(r),
		LpFee:              randomLPFee(r),
		ProtocolFee:        randomProtocolFee(r),
		MinLiquidity:       randomMinLiquidity(r),
		MaxSlippagePercent: randomMaxSlippage(r),
	}
}

func randomSwapFee(r *rand.Rand) math.LegacyDec {
	// 0.1% to 1%
	return math.LegacyNewDecWithPrec(int64(simtypes.RandIntBetween(r, 1, 10)), 3)
}

func randomLPFee(r *rand.Rand) math.LegacyDec {
	// 0.01% to 0.5%
	return math.LegacyNewDecWithPrec(int64(simtypes.RandIntBetween(r, 1, 50)), 4)
}

func randomProtocolFee(r *rand.Rand) math.LegacyDec {
	// 0.01% to 0.1%
	return math.LegacyNewDecWithPrec(int64(simtypes.RandIntBetween(r, 1, 10)), 4)
}

func randomMinLiquidity(r *rand.Rand) math.Int {
	// 100 to 10000
	return math.NewInt(int64(simtypes.RandIntBetween(r, 100, 10000)))
}

func randomMaxSlippage(r *rand.Rand) math.LegacyDec {
	// 1% to 10%
	return math.LegacyNewDecWithPrec(int64(simtypes.RandIntBetween(r, 1, 10)), 2)
}

// Random parameter generators for Oracle
func randomOracleParams(r *rand.Rand) oracletypes.Params {
	return oracletypes.Params{
		VotePeriod:         randomVotePeriod(r),
		VoteThreshold:      randomVoteThreshold(r),
		SlashFraction:      randomSlashFraction(r),
		SlashWindow:        randomSlashWindow(r),
		MinValidPerWindow:  randomMinValidPerWindow(r),
		TwapLookbackWindow: randomTwapLookback(r),
	}
}

func randomVotePeriod(r *rand.Rand) uint64 {
	// 5 to 50 blocks
	return uint64(simtypes.RandIntBetween(r, 5, 50))
}

func randomVoteThreshold(r *rand.Rand) math.LegacyDec {
	// 50% to 67%
	return math.LegacyNewDecWithPrec(int64(simtypes.RandIntBetween(r, 50, 67)), 2)
}

func randomSlashFraction(r *rand.Rand) math.LegacyDec {
	// 0.01% to 1%
	return math.LegacyNewDecWithPrec(int64(simtypes.RandIntBetween(r, 1, 100)), 4)
}

func randomSlashWindow(r *rand.Rand) uint64 {
	// 100 to 1000 blocks
	return uint64(simtypes.RandIntBetween(r, 100, 1000))
}

func randomMinValidPerWindow(r *rand.Rand) uint64 {
	// 50 to 90 percent of slash window
	window := randomSlashWindow(r)
	return window * uint64(simtypes.RandIntBetween(r, 50, 90)) / 100
}

func randomTwapLookback(r *rand.Rand) uint64 {
	// 10 to 100 blocks
	return uint64(simtypes.RandIntBetween(r, 10, 100))
}

// Random parameter generators for Compute
func randomComputeParams(r *rand.Rand) computetypes.Params {
	return computetypes.Params{
		MinProviderStake:    randomMinProviderStake(r),
		RequestTimeout:      randomRequestTimeout(r),
		ResultTimeout:       randomResultTimeout(r),
		MinGasPrice:         randomMinGasPrice(r),
		MaxRequestsPerBlock: randomMaxRequestsPerBlock(r),
	}
}

func randomMinProviderStake(r *rand.Rand) math.Int {
	// 1000 to 100000
	return math.NewInt(int64(simtypes.RandIntBetween(r, 1000, 100000)))
}

func randomRequestTimeout(r *rand.Rand) uint64 {
	// 100 to 1000 blocks
	return uint64(simtypes.RandIntBetween(r, 100, 1000))
}

func randomResultTimeout(r *rand.Rand) uint64 {
	// 50 to 500 blocks
	return uint64(simtypes.RandIntBetween(r, 50, 500))
}

func randomMinGasPrice(r *rand.Rand) math.LegacyDec {
	// 0.001 to 0.1
	return math.LegacyNewDecWithPrec(int64(simtypes.RandIntBetween(r, 1, 100)), 3)
}

func randomMaxRequestsPerBlock(r *rand.Rand) uint64 {
	// 10 to 100
	return uint64(simtypes.RandIntBetween(r, 10, 100))
}

// BlockedAddresses returns all the app's blocked account addresses.
func BlockedAddresses() map[string]bool {
	modAccAddrs := make(map[string]bool)

	// Add module account addresses that should not receive funds in simulation
	// This prevents simulation from trying to send funds to module accounts
	blockedAddrs := []string{
		"paw17xpfvakm2amg962yls6f84z3kell8c5lserqta", // fee_collector
		"paw1m3h30wlvsf8llruxtpukdvsy0km2kum8g38c8q", // distribution
		"paw1jv65s3grqf6v6jl3dp4t6c9t9rk99cd88lyufl", // bonded_tokens_pool
		"paw1fl48vsnmsdzcv85q5d2q4z5ajdha8yu34mf0eh", // not_bonded_tokens_pool
		"paw10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn", // gov
		"paw1tygms3xhhs3yv487phx3dw4a95jn7t7lpm470r", // mint
	}

	for _, addr := range blockedAddrs {
		modAccAddrs[addr] = true
	}

	return modAccAddrs
}
