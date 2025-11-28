package simulation

import (
	"math/rand"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// Simulation operation weights constants
const (
	OpWeightMsgSubmitPrice = "op_weight_msg_submit_price"
	OpWeightMsgDelegateFeeder = "op_weight_msg_delegate_feeder"

	DefaultWeightMsgSubmitPrice   = 80
	DefaultWeightMsgDelegateFeeder = 10
)

// WeightedOperations returns all the oracle module operations with their respective weights.
func WeightedOperations(
	appParams simtypes.AppParams,
	cdc simtypes.Codec,
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	sk types.StakingKeeper,
) simulation.WeightedOperations {
	var (
		weightMsgSubmitPrice   int
		weightMsgDelegateFeeder int
	)

	appParams.GetOrGenerate(OpWeightMsgSubmitPrice, &weightMsgSubmitPrice, nil,
		func(_ *rand.Rand) {
			weightMsgSubmitPrice = DefaultWeightMsgSubmitPrice
		},
	)

	appParams.GetOrGenerate(OpWeightMsgDelegateFeeder, &weightMsgDelegateFeeder, nil,
		func(_ *rand.Rand) {
			weightMsgDelegateFeeder = DefaultWeightMsgDelegateFeeder
		},
	)

	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			weightMsgSubmitPrice,
			SimulateMsgSubmitPrice(k, ak, bk, sk),
		),
		simulation.NewWeightedOperation(
			weightMsgDelegateFeeder,
			SimulateMsgDelegateFeeder(k, ak, bk, sk),
		),
	}
}

// SimulateMsgSubmitPrice generates a MsgSubmitPrice with random values
func SimulateMsgSubmitPrice(k keeper.Keeper, ak types.AccountKeeper, bk types.BankKeeper, sk types.StakingKeeper) simtypes.Operation {
	return func(
		r *rand.Rand, app interface{}, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Get a random validator
		validators, err := sk.GetAllValidators(ctx)
		if err != nil || len(validators) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgSubmitPrice, "no validators"), nil, nil
		}

		validator := validators[r.Intn(len(validators))]
		valAddr := validator.GetOperator()

		// Find corresponding account
		var simAccount simtypes.Account
		found := false
		for _, acc := range accs {
			if sdk.ValAddress(acc.Address).Equals(valAddr) {
				simAccount = acc
				found = true
				break
			}
		}
		if !found {
			// Use first account as feeder
			simAccount = accs[0]
		}

		// Random asset and price
		assets := []string{"BTC", "ETH", "ATOM", "OSMO"}
		asset := assets[r.Intn(len(assets))]

		// Generate realistic price (between 0.01 and 100,000)
		priceInt := int64(simtypes.RandIntBetween(r, 1, 10000000))
		price := math.LegacyNewDec(priceInt).QuoInt64(100)

		msg := &types.MsgSubmitPrice{
			Validator: valAddr.String(),
			Feeder:    simAccount.Address.String(),
			Asset:     asset,
			Price:     price,
		}

		txCtx := simulation.OperationInput{
			R:             r,
			App:           app.(*module.SimulationManager),
			TxGen:         nil,
			Cdc:           nil,
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

// SimulateMsgDelegateFeeder generates a MsgDelegateFeeder with random values
func SimulateMsgDelegateFeeder(k keeper.Keeper, ak types.AccountKeeper, bk types.BankKeeper, sk types.StakingKeeper) simtypes.Operation {
	return func(
		r *rand.Rand, app interface{}, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Get a random validator
		validators, err := sk.GetAllValidators(ctx)
		if err != nil || len(validators) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgDelegateFeeder, "no validators"), nil, nil
		}

		validator := validators[r.Intn(len(validators))]
		valAddr := validator.GetOperator()

		// Find corresponding validator account
		var simAccount simtypes.Account
		found := false
		for _, acc := range accs {
			if sdk.ValAddress(acc.Address).Equals(valAddr) {
				simAccount = acc
				found = true
				break
			}
		}
		if !found {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgDelegateFeeder, "validator account not found"), nil, nil
		}

		// Random feeder account
		feederAccount, _ := simtypes.RandomAcc(r, accs)

		msg := &types.MsgDelegateFeeder{
			Validator: valAddr.String(),
			Feeder:    feederAccount.Address.String(),
		}

		txCtx := simulation.OperationInput{
			R:             r,
			App:           app.(*module.SimulationManager),
			TxGen:         nil,
			Cdc:           nil,
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
