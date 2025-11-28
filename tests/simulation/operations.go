package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	// Bank operations
	banksim "github.com/cosmos/cosmos-sdk/x/bank/simulation"

	// Staking operations
	stakingsim "github.com/cosmos/cosmos-sdk/x/staking/simulation"

	// Distribution operations
	distrsim "github.com/cosmos/cosmos-sdk/x/distribution/simulation"

	// Slashing operations
	slashingsim "github.com/cosmos/cosmos-sdk/x/slashing/simulation"

	// Governance operations
	govsim "github.com/cosmos/cosmos-sdk/x/gov/simulation"

	// PAW app and modules
	"github.com/paw-chain/paw/app"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
	computetypes "github.com/paw-chain/paw/x/compute/types"
)

// WeightedOperations returns all the operations from the module with their respective weights
func SimulationOperations(app *app.PAWApp, cdc codec.JSONCodec, config simtypes.Config) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	// Bank module operations (high weight - most common)
	operations = append(operations, banksim.WeightedOperations(
		app.AppCodec(),
		app.AccountKeeper,
		app.BankKeeper,
	)...)

	// Staking module operations
	operations = append(operations, stakingsim.WeightedOperations(
		app.AppCodec(),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
	)...)

	// Distribution module operations
	operations = append(operations, distrsim.WeightedOperations(
		app.AppCodec(),
		app.AccountKeeper,
		app.BankKeeper,
		app.DistrKeeper,
		app.StakingKeeper,
	)...)

	// Slashing module operations
	operations = append(operations, slashingsim.WeightedOperations(
		app.AppCodec(),
		app.AccountKeeper,
		app.BankKeeper,
		app.SlashingKeeper,
		app.StakingKeeper,
	)...)

	// Governance module operations
	operations = append(operations, govsim.WeightedOperations(
		app.AppCodec(),
		app.AccountKeeper,
		app.BankKeeper,
		app.GovKeeper,
		nil, // No legacy proposal content
	)...)

	// DEX module operations
	operations = append(operations, getDEXOperations(app, cdc, config)...)

	// Oracle module operations
	operations = append(operations, getOracleOperations(app, cdc, config)...)

	// Compute module operations
	operations = append(operations, getComputeOperations(app, cdc, config)...)

	return operations
}

// getDEXOperations returns weighted DEX module operations
func getDEXOperations(app *app.PAWApp, cdc codec.JSONCodec, config simtypes.Config) []simtypes.WeightedOperation {
	return []simtypes.WeightedOperation{
		simulation.NewWeightedOperation(
			30,
			SimulateMsgCreatePool(app.AccountKeeper, app.BankKeeper, app.DEXKeeper),
		),
		simulation.NewWeightedOperation(
			50,
			SimulateMsgSwap(app.AccountKeeper, app.BankKeeper, app.DEXKeeper),
		),
		simulation.NewWeightedOperation(
			40,
			SimulateMsgAddLiquidity(app.AccountKeeper, app.BankKeeper, app.DEXKeeper),
		),
		simulation.NewWeightedOperation(
			20,
			SimulateMsgRemoveLiquidity(app.AccountKeeper, app.BankKeeper, app.DEXKeeper),
		),
	}
}

// getOracleOperations returns weighted Oracle module operations
func getOracleOperations(app *app.PAWApp, cdc codec.JSONCodec, config simtypes.Config) []simtypes.WeightedOperation {
	return []simtypes.WeightedOperation{
		simulation.NewWeightedOperation(
			60,
			SimulateMsgSubmitPrice(app.AccountKeeper, app.StakingKeeper, app.OracleKeeper),
		),
		simulation.NewWeightedOperation(
			10,
			SimulateMsgDelegateFeeder(app.AccountKeeper, app.StakingKeeper, app.OracleKeeper),
		),
	}
}

// getComputeOperations returns weighted Compute module operations
func getComputeOperations(app *app.PAWApp, cdc codec.JSONCodec, config simtypes.Config) []simtypes.WeightedOperation {
	return []simtypes.WeightedOperation{
		simulation.NewWeightedOperation(
			20,
			SimulateMsgSubmitRequest(app.AccountKeeper, app.BankKeeper, app.ComputeKeeper),
		),
		simulation.NewWeightedOperation(
			15,
			SimulateMsgSubmitResult(app.AccountKeeper, app.ComputeKeeper),
		),
		simulation.NewWeightedOperation(
			10,
			SimulateMsgRegisterProvider(app.AccountKeeper, app.BankKeeper, app.ComputeKeeper),
		),
	}
}

// DEX Simulation Operations

// SimulateMsgCreatePool simulates MsgCreatePool
func SimulateMsgCreatePool(ak AccountKeeper, bk BankKeeper, k DEXKeeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string) (
		simtypes.OperationMsg, []simtypes.FutureOperation, error) {

		creator, _ := simtypes.RandomAcc(r, accs)
		msg := &dextypes.MsgCreatePool{
			Creator: creator.Address.String(),
			TokenA:  "upaw",
			TokenB:  randomDenom(r),
			AmountA: simtypes.RandomAmount(r, sdk.NewInt(1000000)),
			AmountB: simtypes.RandomAmount(r, sdk.NewInt(1000000)),
		}

		return simtypes.NoOpMsg(dextypes.ModuleName, msg.Type(), "create pool"), nil, nil
	}
}

// SimulateMsgSwap simulates MsgSwap
func SimulateMsgSwap(ak AccountKeeper, bk BankKeeper, k DEXKeeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string) (
		simtypes.OperationMsg, []simtypes.FutureOperation, error) {

		trader, _ := simtypes.RandomAcc(r, accs)

		// Get random pool
		pools := k.GetAllPools(ctx)
		if len(pools) == 0 {
			return simtypes.NoOpMsg(dextypes.ModuleName, "swap", "no pools"), nil, nil
		}

		pool := pools[r.Intn(len(pools))]

		msg := &dextypes.MsgSwap{
			Trader:       trader.Address.String(),
			PoolId:       pool.Id,
			TokenIn:      pool.TokenA,
			AmountIn:     simtypes.RandomAmount(r, sdk.NewInt(10000)),
			MinAmountOut: sdk.NewInt(1),
		}

		return simtypes.NoOpMsg(dextypes.ModuleName, msg.Type(), "swap"), nil, nil
	}
}

// SimulateMsgAddLiquidity simulates MsgAddLiquidity
func SimulateMsgAddLiquidity(ak AccountKeeper, bk BankKeeper, k DEXKeeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string) (
		simtypes.OperationMsg, []simtypes.FutureOperation, error) {

		provider, _ := simtypes.RandomAcc(r, accs)

		pools := k.GetAllPools(ctx)
		if len(pools) == 0 {
			return simtypes.NoOpMsg(dextypes.ModuleName, "add_liquidity", "no pools"), nil, nil
		}

		pool := pools[r.Intn(len(pools))]

		msg := &dextypes.MsgAddLiquidity{
			Provider: provider.Address.String(),
			PoolId:   pool.Id,
			AmountA:  simtypes.RandomAmount(r, sdk.NewInt(100000)),
			AmountB:  simtypes.RandomAmount(r, sdk.NewInt(100000)),
		}

		return simtypes.NoOpMsg(dextypes.ModuleName, msg.Type(), "add liquidity"), nil, nil
	}
}

// SimulateMsgRemoveLiquidity simulates MsgRemoveLiquidity
func SimulateMsgRemoveLiquidity(ak AccountKeeper, bk BankKeeper, k DEXKeeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string) (
		simtypes.OperationMsg, []simtypes.FutureOperation, error) {

		provider, _ := simtypes.RandomAcc(r, accs)

		pools := k.GetAllPools(ctx)
		if len(pools) == 0 {
			return simtypes.NoOpMsg(dextypes.ModuleName, "remove_liquidity", "no pools"), nil, nil
		}

		pool := pools[r.Intn(len(pools))]

		msg := &dextypes.MsgRemoveLiquidity{
			Provider: provider.Address.String(),
			PoolId:   pool.Id,
			Shares:   simtypes.RandomAmount(r, sdk.NewInt(1000)),
		}

		return simtypes.NoOpMsg(dextypes.ModuleName, msg.Type(), "remove liquidity"), nil, nil
	}
}

// Oracle Simulation Operations

// SimulateMsgSubmitPrice simulates MsgSubmitPrice
func SimulateMsgSubmitPrice(ak AccountKeeper, sk StakingKeeper, k OracleKeeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string) (
		simtypes.OperationMsg, []simtypes.FutureOperation, error) {

		validator, _ := simtypes.RandomAcc(r, accs)

		msg := &oracletypes.MsgSubmitPrice{
			Validator: validator.Address.String(),
			Asset:     randomAsset(r),
			Price:     randomPrice(r),
		}

		return simtypes.NoOpMsg(oracletypes.ModuleName, msg.Type(), "submit price"), nil, nil
	}
}

// SimulateMsgDelegateFeeder simulates MsgDelegateFeeder
func SimulateMsgDelegateFeeder(ak AccountKeeper, sk StakingKeeper, k OracleKeeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string) (
		simtypes.OperationMsg, []simtypes.FutureOperation, error) {

		validator, _ := simtypes.RandomAcc(r, accs)
		feeder, _ := simtypes.RandomAcc(r, accs)

		msg := &oracletypes.MsgDelegateFeederConsent{
			Validator: validator.Address.String(),
			Feeder:    feeder.Address.String(),
		}

		return simtypes.NoOpMsg(oracletypes.ModuleName, msg.Type(), "delegate feeder"), nil, nil
	}
}

// Compute Simulation Operations

// SimulateMsgSubmitRequest simulates MsgSubmitRequest
func SimulateMsgSubmitRequest(ak AccountKeeper, bk BankKeeper, k ComputeKeeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string) (
		simtypes.OperationMsg, []simtypes.FutureOperation, error) {

		requester, _ := simtypes.RandomAcc(r, accs)

		msg := &computetypes.MsgSubmitRequest{
			Requester:      requester.Address.String(),
			ContainerImage: "ubuntu:latest",
			Cpu:            uint64(r.Intn(8) + 1),
			Memory:         uint64(r.Intn(16) + 1) * 1024,
			Disk:           uint64(r.Intn(100) + 1) * 1024,
			Timeout:        uint64(r.Intn(3600) + 60),
			MaxPayment:     simtypes.RandomAmount(r, sdk.NewInt(1000000)),
		}

		return simtypes.NoOpMsg(computetypes.ModuleName, msg.Type(), "submit request"), nil, nil
	}
}

// SimulateMsgSubmitResult simulates MsgSubmitResult
func SimulateMsgSubmitResult(ak AccountKeeper, k ComputeKeeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string) (
		simtypes.OperationMsg, []simtypes.FutureOperation, error) {

		provider, _ := simtypes.RandomAcc(r, accs)

		requests := k.GetAllRequests(ctx)
		if len(requests) == 0 {
			return simtypes.NoOpMsg(computetypes.ModuleName, "submit_result", "no requests"), nil, nil
		}

		request := requests[r.Intn(len(requests))]

		msg := &computetypes.MsgSubmitResult{
			Provider:   provider.Address.String(),
			RequestId:  request.Id,
			ResultHash: randomHash(r),
		}

		return simtypes.NoOpMsg(computetypes.ModuleName, msg.Type(), "submit result"), nil, nil
	}
}

// SimulateMsgRegisterProvider simulates MsgRegisterProvider
func SimulateMsgRegisterProvider(ak AccountKeeper, bk BankKeeper, k ComputeKeeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string) (
		simtypes.OperationMsg, []simtypes.FutureOperation, error) {

		provider, _ := simtypes.RandomAcc(r, accs)

		msg := &computetypes.MsgRegisterProvider{
			Provider: provider.Address.String(),
			Cpu:      uint64(r.Intn(64) + 1),
			Memory:   uint64(r.Intn(128) + 1) * 1024,
			Disk:     uint64(r.Intn(1000) + 1) * 1024,
			Stake:    simtypes.RandomAmount(r, sdk.NewInt(10000000)),
		}

		return simtypes.NoOpMsg(computetypes.ModuleName, msg.Type(), "register provider"), nil, nil
	}
}

// Helper functions

func randomDenom(r *rand.Rand) string {
	denoms := []string{"uatom", "uosmo", "uusdc", "uusdt", "ueth", "ubtc"}
	return denoms[r.Intn(len(denoms))]
}

func randomAsset(r *rand.Rand) string {
	assets := []string{"BTC/USD", "ETH/USD", "ATOM/USD", "OSMO/USD", "PAW/USD"}
	return assets[r.Intn(len(assets))]
}

func randomPrice(r *rand.Rand) sdk.Dec {
	// Random price between 0.01 and 100000
	base := r.Intn(100000) + 1
	return sdk.NewDec(int64(base)).Quo(sdk.NewDec(100))
}

func randomHash(r *rand.Rand) string {
	const letters = "abcdef0123456789"
	b := make([]byte, 64)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}
