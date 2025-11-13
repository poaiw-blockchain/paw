package simapp

import (
	"math/rand"

	"cosmossdk.io/math"
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
	InitialPoolCount    = "initial_pool_count"
	InitialLiquidity    = "initial_liquidity"
	SwapProbability     = "swap_probability"
	AddLiquidityProb    = "add_liquidity_probability"
	RemoveLiquidityProb = "remove_liquidity_probability"

	// Compute parameters
	ComputeOperationProb = "compute_operation_probability"

	// Oracle parameters
	OracleFeedProb = "oracle_feed_probability"
)

// SimulationParams defines the parameters for the simulation
type SimulationParams struct {
	// Account parameters
	StakePerAccount       math.Int
	InitialAccountBalance math.Int

	// Staking parameters
	InitiallyBondedValidators int

	// DEX parameters
	InitialPoolCount    int
	InitialLiquidity    math.Int
	SwapProbability     math.LegacyDec
	AddLiquidityProb    math.LegacyDec
	RemoveLiquidityProb math.LegacyDec

	// Compute parameters
	ComputeOperationProb math.LegacyDec

	// Oracle parameters
	OracleFeedProb math.LegacyDec
}

// DefaultSimulationParams returns default simulation parameters
func DefaultSimulationParams() SimulationParams {
	return SimulationParams{
		StakePerAccount:           math.NewInt(100000000000),  // 100k tokens
		InitialAccountBalance:     math.NewInt(1000000000000), // 1M tokens
		InitiallyBondedValidators: 50,
		InitialPoolCount:          10,
		InitialLiquidity:          math.NewInt(10000000000),         // 10k tokens per pool
		SwapProbability:           math.LegacyNewDecWithPrec(30, 2), // 30%
		AddLiquidityProb:          math.LegacyNewDecWithPrec(10, 2), // 10%
		RemoveLiquidityProb:       math.LegacyNewDecWithPrec(10, 2), // 10%
		ComputeOperationProb:      math.LegacyNewDecWithPrec(5, 2),  // 5%
		OracleFeedProb:            math.LegacyNewDecWithPrec(15, 2), // 15%
	}
}

// RandomizedParams creates randomized simulation parameters
func RandomizedParams(r *rand.Rand) SimulationParams {
	return SimulationParams{
		StakePerAccount:           simulation.RandomAmount(r, math.NewInt(1000000000000)),
		InitialAccountBalance:     simulation.RandomAmount(r, math.NewInt(10000000000000)),
		InitiallyBondedValidators: simulation.RandIntBetween(r, 10, 100),
		InitialPoolCount:          simulation.RandIntBetween(r, 5, 20),
		InitialLiquidity:          simulation.RandomAmount(r, math.NewInt(100000000000)),
		SwapProbability:           simulation.RandomDecAmount(r, math.LegacyNewDecWithPrec(50, 2)),
		AddLiquidityProb:          simulation.RandomDecAmount(r, math.LegacyNewDecWithPrec(30, 2)),
		RemoveLiquidityProb:       simulation.RandomDecAmount(r, math.LegacyNewDecWithPrec(30, 2)),
		ComputeOperationProb:      simulation.RandomDecAmount(r, math.LegacyNewDecWithPrec(20, 2)),
		OracleFeedProb:            simulation.RandomDecAmount(r, math.LegacyNewDecWithPrec(40, 2)),
	}
}

// ParamChanges defines the parameters that can be modified by parameter change proposals
// Note: Legacy param changes are deprecated in SDK v0.50, returning empty slice
func ParamChanges(r *rand.Rand) []simulation.LegacyParamChange {
	// TODO: Update to use new param change mechanism when needed
	return []simulation.LegacyParamChange{}
}

// RandomAccounts creates random accounts for simulation
func RandomAccounts(r *rand.Rand, n int) []simulation.Account {
	// Use the SDK's RandomAccounts function instead
	return simulation.RandomAccounts(r, n)
}

// WeightedOperations returns the default weighted operations for simulation
func WeightedOperations() []simulation.WeightedOperation {
	// This would be populated with actual operations
	// Example structure:
	/*
		return []simulation.WeightedOperation{
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
