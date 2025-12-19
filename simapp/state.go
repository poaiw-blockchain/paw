package simapp

import (
	"encoding/json"
	"fmt"
	stdmath "math"
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
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

func toUint64(value int) uint64 {
	if value < 0 {
		return 0
	}
	return uint64(value)
}

func toUint32(value int) uint32 {
	if value < 0 {
		return 0
	}
	if value > int(stdmath.MaxUint32) {
		return stdmath.MaxUint32
	}
	return uint32(value)
}

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
		appState := make(map[string]json.RawMessage, len(genesisState))

		// Start from provided genesis (or defaults) so we don't drop module defaults.
		for k, v := range genesisState {
			appState[k] = v
		}

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

		// DEX genesis
		dexGenesis := RandomizedDEXGenesisState(r, accs, startTime)
		appState[dextypes.ModuleName] = cdc.MustMarshalJSON(&dexGenesis)

		// Oracle genesis
		oracleGenesis := RandomizedOracleGenesisState(r, accs, startTime, numInitiallyBonded)
		appState[oracletypes.ModuleName] = cdc.MustMarshalJSON(&oracleGenesis)

		// Compute genesis
		computeGenesis := RandomizedComputeGenesisState(r, accs, startTime)
		appState[computetypes.ModuleName] = cdc.MustMarshalJSON(&computeGenesis)

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
		coins := sdk.NewCoins(sdk.NewInt64Coin("upaw", int64(balance)))

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
		usdcCoins := sdk.NewCoins(sdk.NewInt64Coin("uusdc", int64(usdcBalance)))

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

// RandomizedDEXGenesisState generates a realistic DEX genesis state with seeded pools and TWAPs.
func RandomizedDEXGenesisState(r *rand.Rand, accs []simtypes.Account, genesisTime time.Time) dextypes.GenesisState {
	poolCount := simtypes.RandIntBetween(r, 3, 6)
	pools := make([]dextypes.Pool, poolCount)
	twaps := make([]dextypes.PoolTWAP, poolCount)

	creator := accs[0].Address.String()
	for i := 0; i < poolCount; i++ {
		reserveA := simtypes.RandIntBetween(r, 1_000_000, 10_000_000)
		reserveB := simtypes.RandIntBetween(r, 2_000_000, 15_000_000)

		geomMean := uint64(stdmath.Sqrt(float64(reserveA * reserveB)))
		if geomMean == 0 {
			geomMean = 1
		}

		poolID := toUint64(i) + 1
		pools[i] = dextypes.Pool{
			Id:          poolID,
			TokenA:      "upaw",
			TokenB:      "uusdc",
			ReserveA:    math.NewInt(int64(reserveA)),
			ReserveB:    math.NewInt(int64(reserveB)),
			TotalShares: math.NewIntFromUint64(geomMean),
			Creator:     creator,
		}

		price := math.LegacyNewDec(int64(reserveB)).QuoInt64(int64(reserveA))
		twaps[i] = dextypes.PoolTWAP{
			PoolId:          poolID,
			LastPrice:       price,
			CumulativePrice: price,
			TotalSeconds:    1,
			LastTimestamp:   genesisTime.Unix(),
			TwapPrice:       price,
		}
	}

	return dextypes.GenesisState{
		Params:          dextypes.DefaultParams(),
		Pools:           pools,
		NextPoolId:      toUint64(poolCount) + 1,
		PoolTwapRecords: twaps,
	}
}

// RandomizedOracleGenesisState seeds validator oracles and a baseline price.
func RandomizedOracleGenesisState(r *rand.Rand, accs []simtypes.Account, genesisTime time.Time, bondedValidators int) oracletypes.GenesisState {
	maxValidators := bondedValidators
	if maxValidators > len(accs) {
		maxValidators = len(accs)
	}
	if maxValidators == 0 {
		return *oracletypes.DefaultGenesis()
	}

	validatorOracles := make([]oracletypes.ValidatorOracle, maxValidators)
	validatorPrices := make([]oracletypes.ValidatorPrice, maxValidators)

	asset := "PAW/USD"
	basePrice := math.LegacyNewDec(int64(simtypes.RandIntBetween(r, 50_00, 150_00))).QuoInt64(100)

	for i := 0; i < maxValidators; i++ {
		valAddr := sdk.ValAddress(accs[i].Address).String()
		validatorOracles[i] = oracletypes.ValidatorOracle{
			ValidatorAddr:    valAddr,
			MissCounter:      0,
			TotalSubmissions: 0,
			IsActive:         true,
		}

		// introduce slight jitter per validator
		jitter := math.LegacyNewDec(int64(simtypes.RandIntBetween(r, -50, 50))).QuoInt64(10_000)
		price := basePrice.Add(jitter)
		if price.LTE(math.LegacyZeroDec()) {
			price = basePrice
		}
		validatorPrices[i] = oracletypes.ValidatorPrice{
			ValidatorAddr: valAddr,
			Asset:         asset,
			Price:         price,
			BlockHeight:   1,
			VotingPower:   1,
		}
	}

	price := oracletypes.Price{
		Asset:         asset,
		Price:         basePrice,
		BlockHeight:   1,
		BlockTime:     genesisTime.Unix(),
		NumValidators: toUint32(maxValidators),
	}

	snapshot := oracletypes.PriceSnapshot{
		Asset:       asset,
		Price:       basePrice,
		BlockHeight: 1,
		BlockTime:   genesisTime.Unix(),
	}

	return oracletypes.GenesisState{
		Params:           oracletypes.DefaultParams(),
		Prices:           []oracletypes.Price{price},
		ValidatorPrices:  validatorPrices,
		ValidatorOracles: validatorOracles,
		PriceSnapshots:   []oracletypes.PriceSnapshot{snapshot},
	}
}

// RandomizedComputeGenesisState seeds a small provider set for simulations.
func RandomizedComputeGenesisState(r *rand.Rand, accs []simtypes.Account, genesisTime time.Time) computetypes.GenesisState {
	providerCount := simtypes.RandIntBetween(r, 2, minInt(5, len(accs)))
	providers := make([]computetypes.Provider, providerCount)

	for i := 0; i < providerCount; i++ {
		addr := accs[i].Address.String()
		cpu := toUint64(simtypes.RandIntBetween(r, 500, 2000))   // 0.5–2 cores (millicores)
		mem := toUint64(simtypes.RandIntBetween(r, 512, 4096))   // MB
		storage := toUint64(simtypes.RandIntBetween(r, 10, 200)) // GB
		timeout := toUint64(simtypes.RandIntBetween(r, 300, 1800))

		providers[i] = computetypes.Provider{
			Address:  addr,
			Moniker:  fmt.Sprintf("provider-%d", i+1),
			Endpoint: fmt.Sprintf("https://provider-%d.paw.sim", i+1),
			AvailableSpecs: computetypes.ComputeSpec{
				CpuCores:       cpu,
				MemoryMb:       mem,
				GpuCount:       0,
				GpuType:        "",
				StorageGb:      storage,
				TimeoutSeconds: timeout,
			},
			Pricing: computetypes.Pricing{
				CpuPricePerMcoreHour:  math.LegacyMustNewDecFromStr("0.0005"),
				MemoryPricePerMbHour:  math.LegacyMustNewDecFromStr("0.0001"),
				GpuPricePerHour:       math.LegacyMustNewDecFromStr("0.00"),
				StoragePricePerGbHour: math.LegacyMustNewDecFromStr("0.00001"),
			},
			Stake:                  math.NewInt(1_000_000_000),
			Reputation:             toUint32(simtypes.RandIntBetween(r, 70, 100)),
			TotalRequestsCompleted: 0,
			TotalRequestsFailed:    0,
			Active:                 true,
			RegisteredAt:           genesisTime,
			LastActiveAt:           genesisTime,
		}
	}

	return computetypes.GenesisState{
		Params:           computetypes.DefaultParams(),
		GovernanceParams: computetypes.DefaultGovernanceParams(),
		Providers:        providers,
		Requests:         []computetypes.Request{},
		Results:          []computetypes.Result{},
		Disputes:         []computetypes.Dispute{},
		SlashRecords:     []computetypes.SlashRecord{},
		Appeals:          []computetypes.Appeal{},
		NextRequestId:    1,
		NextDisputeId:    1,
		NextSlashId:      1,
		NextAppealId:     1,
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RandomizeParamChanges randomizes all parameters for simulation
func RandomizeParamChanges(r *rand.Rand) []simtypes.LegacyParamChange {
	return ParamChanges(r)
}
