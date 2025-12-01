package simulation

import (
	"math/rand"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// Simulation operation weights constants
const (
	OpWeightMsgCreatePool      = "op_weight_msg_create_pool"
	OpWeightMsgAddLiquidity    = "op_weight_msg_add_liquidity"
	OpWeightMsgRemoveLiquidity = "op_weight_msg_remove_liquidity"
	OpWeightMsgSwap            = "op_weight_msg_swap"

	DefaultWeightMsgCreatePool      = 15
	DefaultWeightMsgAddLiquidity    = 30
	DefaultWeightMsgRemoveLiquidity = 20
	DefaultWeightMsgSwap            = 50
)

// WeightedOperations returns all the DEX module operations with their respective weights.
func WeightedOperations(
	appParams simtypes.AppParams,
	cdc codec.JSONCodec,
	txGen client.TxConfig,
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simulation.WeightedOperations {
	protoCdc, _ := cdc.(*codec.ProtoCodec)

	var (
		weightMsgCreatePool      int
		weightMsgAddLiquidity    int
		weightMsgRemoveLiquidity int
		weightMsgSwap            int
	)

	appParams.GetOrGenerate(OpWeightMsgCreatePool, &weightMsgCreatePool, nil,
		func(_ *rand.Rand) {
			weightMsgCreatePool = DefaultWeightMsgCreatePool
		},
	)

	appParams.GetOrGenerate(OpWeightMsgAddLiquidity, &weightMsgAddLiquidity, nil,
		func(_ *rand.Rand) {
			weightMsgAddLiquidity = DefaultWeightMsgAddLiquidity
		},
	)

	appParams.GetOrGenerate(OpWeightMsgRemoveLiquidity, &weightMsgRemoveLiquidity, nil,
		func(_ *rand.Rand) {
			weightMsgRemoveLiquidity = DefaultWeightMsgRemoveLiquidity
		},
	)

	appParams.GetOrGenerate(OpWeightMsgSwap, &weightMsgSwap, nil,
		func(_ *rand.Rand) {
			weightMsgSwap = DefaultWeightMsgSwap
		},
	)

	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			weightMsgCreatePool,
			SimulateMsgCreatePool(txGen, protoCdc, k, ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgAddLiquidity,
			SimulateMsgAddLiquidity(txGen, protoCdc, k, ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgRemoveLiquidity,
			SimulateMsgRemoveLiquidity(txGen, protoCdc, k, ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgSwap,
			SimulateMsgSwap(txGen, protoCdc, k, ak, bk),
		),
	}
}

// SimulateMsgCreatePool generates a MsgCreatePool with random values
func SimulateMsgCreatePool(
	txGen client.TxConfig,
	cdc *codec.ProtoCodec,
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		// Generate random token denoms
		denoms := []string{"upaw", "uatom", "uosmo", "ujuno"}
		tokenA := denoms[r.Intn(len(denoms))]
		tokenB := denoms[r.Intn(len(denoms))]

		if tokenA == tokenB {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgCreatePool, "same token"), nil, nil
		}

		// Random amounts
		amountA := math.NewInt(int64(simtypes.RandIntBetween(r, 1000, 1000000)))
		amountB := math.NewInt(int64(simtypes.RandIntBetween(r, 1000, 1000000)))

		coins := sdk.NewCoins(
			sdk.NewCoin(tokenA, amountA),
			sdk.NewCoin(tokenB, amountB),
		)

		spendable := bk.SpendableCoins(ctx, simAccount.Address)
		if !spendable.IsAllGTE(coins) {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgCreatePool, "insufficient balance"), nil, nil
		}

		msg := &types.MsgCreatePool{
			Creator: simAccount.Address.String(),
			TokenA:  tokenA,
			TokenB:  tokenB,
			AmountA: amountA,
			AmountB: amountB,
		}

		txCtx := simulation.OperationInput{
			R:               r,
			App:             app,
			TxGen:           txGen,
			Cdc:             cdc,
			Msg:             msg,
			Context:         ctx,
			SimAccount:      simAccount,
			AccountKeeper:   ak,
			Bankkeeper:      bk,
			ModuleName:      types.ModuleName,
			CoinsSpentInMsg: coins,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

// SimulateMsgAddLiquidity generates a MsgAddLiquidity with random values
func SimulateMsgAddLiquidity(
	txGen client.TxConfig,
	cdc *codec.ProtoCodec,
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		// Get random pool
		pools, err := k.GetAllPools(ctx)
		if err != nil || len(pools) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgAddLiquidity, "no pools"), nil, nil
		}

		pool := pools[r.Intn(len(pools))]

		// Random amounts
		amountA := math.NewInt(int64(simtypes.RandIntBetween(r, 100, 10000)))
		amountB := math.NewInt(int64(simtypes.RandIntBetween(r, 100, 10000)))

		coins := sdk.NewCoins(
			sdk.NewCoin(pool.TokenA, amountA),
			sdk.NewCoin(pool.TokenB, amountB),
		)

		spendable := bk.SpendableCoins(ctx, simAccount.Address)
		if !spendable.IsAllGTE(coins) {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgAddLiquidity, "insufficient balance"), nil, nil
		}

		msg := &types.MsgAddLiquidity{
			Provider: simAccount.Address.String(),
			PoolId:   pool.Id,
			AmountA:  amountA,
			AmountB:  amountB,
		}

		txCtx := simulation.OperationInput{
			R:               r,
			App:             app,
			TxGen:           txGen,
			Cdc:             cdc,
			Msg:             msg,
			Context:         ctx,
			SimAccount:      simAccount,
			AccountKeeper:   ak,
			Bankkeeper:      bk,
			ModuleName:      types.ModuleName,
			CoinsSpentInMsg: coins,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

// SimulateMsgRemoveLiquidity generates a MsgRemoveLiquidity with random values
func SimulateMsgRemoveLiquidity(
	txGen client.TxConfig,
	cdc *codec.ProtoCodec,
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		// Get random pool
		pools, err := k.GetAllPools(ctx)
		if err != nil || len(pools) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgRemoveLiquidity, "no pools"), nil, nil
		}

		pool := pools[r.Intn(len(pools))]

		// Check if user has liquidity
		shares, err := k.GetLiquidity(ctx, pool.Id, simAccount.Address)
		if err != nil || shares.IsZero() {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgRemoveLiquidity, "no liquidity"), nil, nil
		}

		// Remove random portion
		sharesToRemove := math.NewInt(int64(simtypes.RandIntBetween(r, 1, int(shares.Int64()))))

		msg := &types.MsgRemoveLiquidity{
			Provider: simAccount.Address.String(),
			PoolId:   pool.Id,
			Shares:   sharesToRemove,
		}

		txCtx := simulation.OperationInput{
			R:             r,
			App:           app,
			TxGen:         txGen,
			Cdc:           cdc,
			Msg:           msg,
			Context:       ctx,
			SimAccount:    simAccount,
			AccountKeeper: ak,
			Bankkeeper:    bk,
			ModuleName:    types.ModuleName,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

// SimulateMsgSwap generates a MsgSwap with random values
func SimulateMsgSwap(
	txGen client.TxConfig,
	cdc *codec.ProtoCodec,
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		// Get random pool
		pools, err := k.GetAllPools(ctx)
		if err != nil || len(pools) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgSwap, "no pools"), nil, nil
		}

		pool := pools[r.Intn(len(pools))]

		// Random swap direction
		var tokenIn, tokenOut string
		if r.Intn(2) == 0 {
			tokenIn = pool.TokenA
			tokenOut = pool.TokenB
		} else {
			tokenIn = pool.TokenB
			tokenOut = pool.TokenA
		}

		// Random amount
		amountIn := math.NewInt(int64(simtypes.RandIntBetween(r, 10, 1000)))
		coin := sdk.NewCoin(tokenIn, amountIn)

		spendable := bk.SpendableCoins(ctx, simAccount.Address)
		if spendable.AmountOf(tokenIn).LT(amountIn) {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgSwap, "insufficient balance"), nil, nil
		}

		msg := &types.MsgSwap{
			Trader:       simAccount.Address.String(),
			PoolId:       pool.Id,
			TokenIn:      tokenIn,
			TokenOut:     tokenOut,
			AmountIn:     amountIn,
			MinAmountOut: math.NewInt(1),
		}

		txCtx := simulation.OperationInput{
			R:               r,
			App:             app,
			TxGen:           txGen,
			Cdc:             cdc,
			Msg:             msg,
			Context:         ctx,
			SimAccount:      simAccount,
			AccountKeeper:   ak,
			Bankkeeper:      bk,
			ModuleName:      types.ModuleName,
			CoinsSpentInMsg: sdk.NewCoins(coin),
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}
