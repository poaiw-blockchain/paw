package simapp

import (
	"fmt"
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
)

// Simulation parameter constants
const (
	// Staking parameters
	StakePerAccount           = "stake_per_account"
	InitiallyBondedValidators = "initially_bonded_validators"

	// Bank parameters
	InitialAccountBalance = "initial_account_balance"

	// DEX parameters
	InitialPoolCount     = "initial_pool_count"
	InitialLiquidity     = "initial_liquidity"
	SwapProbability      = "swap_probability"
	AddLiquidityProb     = "add_liquidity_probability"
	RemoveLiquidityProb  = "remove_liquidity_probability"

	// Compute parameters
	ComputeOperationProb = "compute_operation_probability"

	// Oracle parameters
	OracleFeedProb       = "oracle_feed_probability"
)

// SimulationParams defines the parameters for the simulation
type SimulationParams struct {
	// Account parameters
	StakePerAccount       sdk.Int
	InitialAccountBalance sdk.Int

	// Staking parameters
	InitiallyBondedValidators int

	// DEX parameters
	InitialPoolCount    int
	InitialLiquidity    sdk.Int
	SwapProbability     sdk.Dec
	AddLiquidityProb    sdk.Dec
	RemoveLiquidityProb sdk.Dec

	// Compute parameters
	ComputeOperationProb sdk.Dec

	// Oracle parameters
	OracleFeedProb sdk.Dec
}

// DefaultSimulationParams returns default simulation parameters
func DefaultSimulationParams() SimulationParams {
	return SimulationParams{
		StakePerAccount:           sdk.NewInt(100000000000),      // 100k tokens
		InitialAccountBalance:     sdk.NewInt(1000000000000),     // 1M tokens
		InitiallyBondedValidators: 50,
		InitialPoolCount:          10,
		InitialLiquidity:          sdk.NewInt(10000000000),       // 10k tokens per pool
		SwapProbability:           sdk.NewDecWithPrec(30, 2),     // 30%
		AddLiquidityProb:          sdk.NewDecWithPrec(10, 2),     // 10%
		RemoveLiquidityProb:       sdk.NewDecWithPrec(10, 2),     // 10%
		ComputeOperationProb:      sdk.NewDecWithPrec(5, 2),      // 5%
		OracleFeedProb:            sdk.NewDecWithPrec(15, 2),     // 15%
	}
}

// RandomizedParams creates randomized simulation parameters
func RandomizedParams(r *rand.Rand) SimulationParams {
	return SimulationParams{
		StakePerAccount:           simulation.RandomAmount(r, sdk.NewInt(1000000000000)),
		InitialAccountBalance:     simulation.RandomAmount(r, sdk.NewInt(10000000000000)),
		InitiallyBondedValidators: simulation.RandIntBetween(r, 10, 100),
		InitialPoolCount:          simulation.RandIntBetween(r, 5, 20),
		InitialLiquidity:          simulation.RandomAmount(r, sdk.NewInt(100000000000)),
		SwapProbability:           simulation.RandomDecAmount(r, sdk.NewDecWithPrec(50, 2)),
		AddLiquidityProb:          simulation.RandomDecAmount(r, sdk.NewDecWithPrec(30, 2)),
		RemoveLiquidityProb:       simulation.RandomDecAmount(r, sdk.NewDecWithPrec(30, 2)),
		ComputeOperationProb:      simulation.RandomDecAmount(r, sdk.NewDecWithPrec(20, 2)),
		OracleFeedProb:            simulation.RandomDecAmount(r, sdk.NewDecWithPrec(40, 2)),
	}
}

// ParamChanges defines the parameters that can be modified by parameter change proposals
func ParamChanges(r *rand.Rand) []simulation.LegacyParamChange {
	return []simulation.LegacyParamChange{
		simulation.NewSimLegacyParamChange(
			"bank",
			"SendEnabled",
			func(r *rand.Rand) string {
				return fmt.Sprintf("%v", simulation.RandomBool(r))
			},
		),
		simulation.NewSimLegacyParamChange(
			"staking",
			"MaxValidators",
			func(r *rand.Rand) string {
				return fmt.Sprintf("%d", simulation.RandIntBetween(r, 50, 200))
			},
		),
		simulation.NewSimLegacyParamChange(
			"staking",
			"UnbondingTime",
			func(r *rand.Rand) string {
				return fmt.Sprintf("%d", simulation.RandIntBetween(r, 60, 60*60*24*21)) // 1 min to 21 days
			},
		),
		simulation.NewSimLegacyParamChange(
			"dex",
			"SwapFee",
			func(r *rand.Rand) string {
				// Random fee between 0.1% and 1%
				fee := simulation.RandIntBetween(r, 10, 100)
				return fmt.Sprintf("\"%d\"", fee)
			},
		),
		simulation.NewSimLegacyParamChange(
			"dex",
			"MinLiquidity",
			func(r *rand.Rand) string {
				return fmt.Sprintf("\"%d\"", simulation.RandIntBetween(r, 100, 10000))
			},
		),
	}
}

// RandomAccounts creates random accounts for simulation
func RandomAccounts(r *rand.Rand, n int) []simulation.Account {
	accs := make([]simulation.Account, n)
	for i := 0; i < n; i++ {
		accs[i] = simulation.RandomAccount(r, r.Int())
	}
	return accs
}

// WeightedOperations returns the default weighted operations for simulation
func WeightedOperations() simulation.WeightedOperations {
	// This would be populated with actual operations
	// Example structure:
	/*
	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			100, // weight
			SimulateMsgSwap(am accountKeeper, bk bankKeeper, dk dexKeeper),
		),
		simulation.NewWeightedOperation(
			50,
			SimulateMsgAddLiquidity(am accountKeeper, bk bankKeeper, dk dexKeeper),
		),
		simulation.NewWeightedOperation(
			50,
			SimulateMsgRemoveLiquidity(am accountKeeper, bk bankKeeper, dk dexKeeper),
		),
		// Add more operations
	}
	*/
	return nil
}
