package simapp

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/paw-chain/paw/app"
)

// AppStateFn returns the initial application state using a genesis or the simulation parameters
func AppStateFn(
	cdc codec.JSONCodec,
	simManager *module.SimulationManager,
	genesisState map[string]json.RawMessage,
) simtypes.AppStateFn {
	return func(
		r *rand.Rand,
		accs []simtypes.Account,
		config simtypes.Config,
	) (json.RawMessage, []simtypes.Account, string, time.Time) {
		// Randomize initial parameters
		var (
			numAccs            = 100
			numInitiallyBonded = 50
			initialStake       = math.NewInt(100000000000)
		)

		if len(accs) == 0 {
			accs = simtypes.RandomAccounts(r, numAccs)
		}

		// Generate random genesis time
		startTime := simtypes.RandTimestamp(r)

		// Generate randomized genesis state
		appParams := make(simtypes.AppParams)
		appState := make(map[string]json.RawMessage)

		if genesisState == nil {
			genesisState = app.NewDefaultGenesisState(config.ChainID)
		}

		// Auth genesis
		authGenesis := RandomizedAuthGenesisState(r, accs)
		appState[authtypes.ModuleName] = cdc.MustMarshalJSON(&authGenesis)

		// Bank genesis
		bankGenesis := RandomizedBankGenesisState(r, accs)
		appState[banktypes.ModuleName] = cdc.MustMarshalJSON(&bankGenesis)

		// Staking genesis
		stakingGenesis := RandomizedStakingGenesisState(r, accs, initialStake, numAccs, numInitiallyBonded)
		appState[stakingtypes.ModuleName] = cdc.MustMarshalJSON(&stakingGenesis)

		// DEX genesis (would need actual DEX types)
		// dexGenesis := RandomizedDEXGenesisState(r, accs)
		// appState["dex"] = cdc.MustMarshalJSON(&dexGenesis)

		// Use simulation manager to randomize all other genesis states
		simState := &module.SimulationState{
			AppParams: appParams,
			Cdc:       cdc,
			Rand:      r,
			Accounts:  accs,
			GenState:  appState,
		}
		simManager.GenerateGenesisStates(simState)

		appStateJSON, err := json.MarshalIndent(appState, "", "  ")
		if err != nil {
			panic(err)
		}

		return appStateJSON, accs, config.ChainID, startTime
	}
}

// RandomizedAuthGenesisState generates a random auth genesis state
func RandomizedAuthGenesisState(r *rand.Rand, accs []simtypes.Account) authtypes.GenesisState {
	accountNumber := uint64(0)

	genesisAccounts := make([]authtypes.GenesisAccount, len(accs))
	for i, acc := range accs {
		bacc := authtypes.NewBaseAccountWithAddress(acc.Address)
		bacc.AccountNumber = accountNumber
		accountNumber++

		genesisAccounts[i] = bacc
	}

	authGenesis := authtypes.NewGenesisState(
		authtypes.DefaultParams(),
		genesisAccounts,
	)

	return *authGenesis
}

// RandomizedBankGenesisState generates a random bank genesis state
func RandomizedBankGenesisState(r *rand.Rand, accs []simtypes.Account) banktypes.GenesisState {
	// Create initial balances
	balances := make([]banktypes.Balance, len(accs))
	totalSupply := sdk.NewCoins()

	for i, acc := range accs {
		// Random balance between 1M and 10M upaw
		balance := simtypes.RandIntBetween(r, 1000000, 10000000)
		coins := sdk.NewCoins(math.NewInt64Coin("upaw", int64(balance)))

		balances[i] = banktypes.Balance{
			Address: acc.Address.String(),
			Coins:   coins,
		}

		totalSupply = totalSupply.Add(coins...)
	}

	// Add additional denominations
	for i := range accs {
		// Random USDC balance
		usdcBalance := simtypes.RandIntBetween(r, 100000, 1000000)
		usdcCoins := sdk.NewCoins(math.NewInt64Coin("uusdc", int64(usdcBalance)))

		balances[i].Coins = balances[i].Coins.Add(usdcCoins...)
		totalSupply = totalSupply.Add(usdcCoins...)
	}

	bankGenesis := banktypes.NewGenesisState(
		banktypes.DefaultParams(),
		balances,
		totalSupply,
		[]banktypes.Metadata{
			{
				Description: "The native token of PAW Chain",
				DenomUnits: []*banktypes.DenomUnit{
					{Denom: "upaw", Exponent: uint32(0), Aliases: []string{"micropaw"}},
					{Denom: "paw", Exponent: uint32(6), Aliases: []string{}},
				},
				Base:    "upaw",
				Display: "paw",
				Name:    "PAW",
				Symbol:  "PAW",
			},
			{
				Description: "USD Coin",
				DenomUnits: []*banktypes.DenomUnit{
					{Denom: "uusdc", Exponent: uint32(0), Aliases: []string{"microusdc"}},
					{Denom: "usdc", Exponent: uint32(6), Aliases: []string{}},
				},
				Base:    "uusdc",
				Display: "usdc",
				Name:    "USDC",
				Symbol:  "USDC",
			},
		},
		[]banktypes.SendEnabled{},
	)

	return *bankGenesis
}

// RandomizedStakingGenesisState generates a random staking genesis state
func RandomizedStakingGenesisState(
	r *rand.Rand,
	accs []simtypes.Account,
	initialStake math.Int,
	numAccs, numInitiallyBonded int,
) stakingtypes.GenesisState {
	// Create validators from first N accounts
	validators := make([]stakingtypes.Validator, numInitiallyBonded)
	delegations := make([]stakingtypes.Delegation, numInitiallyBonded)

	for i := 0; i < numInitiallyBonded && i < len(accs); i++ {
		pubKeyAny, err := codectypes.NewAnyWithValue(accs[i].ConsKey.PubKey())
		if err != nil {
			panic(err)
		}

		val := stakingtypes.Validator{
			OperatorAddress:   sdk.ValAddress(accs[i].Address).String(),
			ConsensusPubkey:   pubKeyAny,
			Jailed:            false,
			Status:            stakingtypes.Bonded,
			Tokens:            initialStake,
			DelegatorShares:   math.LegacyNewDecFromInt(initialStake),
			Description:       stakingtypes.Description{Moniker: fmt.Sprintf("validator-%d", i)},
			UnbondingHeight:   int64(0),
			UnbondingTime:     time.Unix(0, 0).UTC(),
			Commission:        stakingtypes.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()),
			MinSelfDelegation: math.OneInt(),
		}

		validators[i] = val

		delegations[i] = stakingtypes.Delegation{
			DelegatorAddress: accs[i].Address.String(),
			ValidatorAddress: sdk.ValAddress(accs[i].Address).String(),
			Shares:           math.LegacyNewDecFromInt(initialStake),
		}
	}

	stakingGenesis := stakingtypes.NewGenesisState(
		stakingtypes.DefaultParams(),
		validators,
		delegations,
	)

	return *stakingGenesis
}

// RandomizedDEXGenesisState generates a random DEX genesis state
// This is a placeholder - implement based on actual DEX types
/*
func RandomizedDEXGenesisState(r *rand.Rand, accs []simtypes.Account) dextypes.GenesisState {
	// Create random pools
	numPools := simtypes.RandIntBetween(r, 5, 20)
	pools := make([]dextypes.Pool, numPools)

	for i := 0; i < numPools; i++ {
		// Random reserves
		reserveA := simtypes.RandIntBetween(r, 1000000, 10000000)
		reserveB := simtypes.RandIntBetween(r, 1000000, 10000000)

		pools[i] = dextypes.Pool{
			Id:       uint64(i + 1),
			DenomA:   "upaw",
			DenomB:   "uusdc",
			ReserveA: math.NewInt(int64(reserveA)),
			ReserveB: math.NewInt(int64(reserveB)),
			TotalShares: math.NewInt(int64(reserveA * reserveB)).ApproxSqrt(),
		}
	}

	return dextypes.GenesisState{
		Params: dextypes.DefaultParams(),
		Pools:  pools,
	}
}
*/

// RandomizeParamChanges randomizes all parameters for simulation
func RandomizeParamChanges(r *rand.Rand) []simtypes.LegacyParamChange {
	return ParamChanges(r)
}
