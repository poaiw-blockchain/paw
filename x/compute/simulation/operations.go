package simulation

import (
	"math/rand"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// Simulation operation weights constants
const (
	OpWeightMsgRegisterProvider   = "op_weight_msg_register_provider"
	OpWeightMsgUpdateProvider      = "op_weight_msg_update_provider"
	OpWeightMsgDeactivateProvider  = "op_weight_msg_deactivate_provider"
	OpWeightMsgSubmitRequest       = "op_weight_msg_submit_request"
	OpWeightMsgSubmitResult        = "op_weight_msg_submit_result"
	OpWeightMsgCancelRequest       = "op_weight_msg_cancel_request"

	DefaultWeightMsgRegisterProvider  = 20
	DefaultWeightMsgUpdateProvider    = 10
	DefaultWeightMsgDeactivateProvider = 5
	DefaultWeightMsgSubmitRequest      = 50
	DefaultWeightMsgSubmitResult       = 40
	DefaultWeightMsgCancelRequest      = 10
)

// WeightedOperations returns all the compute module operations with their respective weights.
func WeightedOperations(
	appParams simtypes.AppParams,
	cdc codec.JSONCodec,
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simulation.WeightedOperations {
	var (
		weightMsgRegisterProvider  int
		weightMsgUpdateProvider    int
		weightMsgDeactivateProvider int
		weightMsgSubmitRequest      int
		weightMsgSubmitResult       int
		weightMsgCancelRequest      int
	)

	appParams.GetOrGenerate(OpWeightMsgRegisterProvider, &weightMsgRegisterProvider, nil,
		func(_ *rand.Rand) {
			weightMsgRegisterProvider = DefaultWeightMsgRegisterProvider
		},
	)

	appParams.GetOrGenerate(OpWeightMsgUpdateProvider, &weightMsgUpdateProvider, nil,
		func(_ *rand.Rand) {
			weightMsgUpdateProvider = DefaultWeightMsgUpdateProvider
		},
	)

	appParams.GetOrGenerate(OpWeightMsgDeactivateProvider, &weightMsgDeactivateProvider, nil,
		func(_ *rand.Rand) {
			weightMsgDeactivateProvider = DefaultWeightMsgDeactivateProvider
		},
	)

	appParams.GetOrGenerate(OpWeightMsgSubmitRequest, &weightMsgSubmitRequest, nil,
		func(_ *rand.Rand) {
			weightMsgSubmitRequest = DefaultWeightMsgSubmitRequest
		},
	)

	appParams.GetOrGenerate(OpWeightMsgSubmitResult, &weightMsgSubmitResult, nil,
		func(_ *rand.Rand) {
			weightMsgSubmitResult = DefaultWeightMsgSubmitResult
		},
	)

	appParams.GetOrGenerate(OpWeightMsgCancelRequest, &weightMsgCancelRequest, nil,
		func(_ *rand.Rand) {
			weightMsgCancelRequest = DefaultWeightMsgCancelRequest
		},
	)

	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			weightMsgRegisterProvider,
			SimulateMsgRegisterProvider(k, ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgUpdateProvider,
			SimulateMsgUpdateProvider(k, ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgDeactivateProvider,
			SimulateMsgDeactivateProvider(k, ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgSubmitRequest,
			SimulateMsgSubmitRequest(k, ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgSubmitResult,
			SimulateMsgSubmitResult(k, ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgCancelRequest,
			SimulateMsgCancelRequest(k, ak, bk),
		),
	}
}

// SimulateMsgRegisterProvider generates a MsgRegisterProvider with random values
func SimulateMsgRegisterProvider(k keeper.Keeper, ak types.AccountKeeper, bk types.BankKeeper) simtypes.Operation {
	return func(
		r *rand.Rand, app interface{}, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		// Get minimum stake requirement
		params, err := k.GetParams(ctx)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgRegisterProvider, "unable to get params"), nil, nil
		}

		// Check if account has enough balance
		spendable := bk.SpendableCoins(ctx, simAccount.Address)
		if !spendable.IsAllGTE(params.MinProviderStake) {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgRegisterProvider, "insufficient balance"), nil, nil
		}

		msg := &types.MsgRegisterProvider{
			Provider: simAccount.Address.String(),
			Endpoint: "https://github.com/compute",
			Pricing: &types.Pricing{
				CpuPricePerHour:    math.NewInt(int64(simtypes.RandIntBetween(r, 100, 1000))),
				MemoryPricePerHour: math.NewInt(int64(simtypes.RandIntBetween(r, 50, 500))),
				GpuPricePerHour:    math.NewInt(int64(simtypes.RandIntBetween(r, 500, 5000))),
			},
			Stake: params.MinProviderStake,
		}

		txCtx := simulation.OperationInput{
			R:               r,
			App:             app.(*module.SimulationManager),
			TxGen:           nil,
			Cdc:             nil,
			Msg:             msg,
			Context:         ctx,
			SimAccount:      simAccount,
			AccountKeeper:   ak,
			Bankkeeper:      bk,
			ModuleName:      types.ModuleName,
			CoinsSpentInMsg: msg.Stake,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

// SimulateMsgUpdateProvider generates a MsgUpdateProvider with random values
func SimulateMsgUpdateProvider(k keeper.Keeper, ak types.AccountKeeper, bk types.BankKeeper) simtypes.Operation {
	return func(
		r *rand.Rand, app interface{}, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Find a random registered provider
		var providers []types.Provider
		err := k.IterateProviders(ctx, func(provider types.Provider) (bool, error) {
			providers = append(providers, provider)
			return false, nil
		})
		if err != nil || len(providers) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgUpdateProvider, "no providers found"), nil, nil
		}

		provider := providers[r.Intn(len(providers))]
		providerAddr, _ := sdk.AccAddressFromBech32(provider.Address)

		// Find the corresponding account
		var simAccount simtypes.Account
		found := false
		for _, acc := range accs {
			if acc.Address.Equals(providerAddr) {
				simAccount = acc
				found = true
				break
			}
		}
		if !found {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgUpdateProvider, "provider account not found"), nil, nil
		}

		msg := &types.MsgUpdateProvider{
			Provider: provider.Address,
			Endpoint: "https://updated.github.com/compute",
			Pricing: &types.Pricing{
				CpuPricePerHour:    math.NewInt(int64(simtypes.RandIntBetween(r, 100, 1000))),
				MemoryPricePerHour: math.NewInt(int64(simtypes.RandIntBetween(r, 50, 500))),
				GpuPricePerHour:    math.NewInt(int64(simtypes.RandIntBetween(r, 500, 5000))),
			},
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

// SimulateMsgDeactivateProvider generates a MsgDeactivateProvider with random values
func SimulateMsgDeactivateProvider(k keeper.Keeper, ak types.AccountKeeper, bk types.BankKeeper) simtypes.Operation {
	return func(
		r *rand.Rand, app interface{}, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Find a random active provider
		var activeProviders []types.Provider
		err := k.IterateActiveProviders(ctx, func(provider types.Provider) (bool, error) {
			activeProviders = append(activeProviders, provider)
			return false, nil
		})
		if err != nil || len(activeProviders) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgDeactivateProvider, "no active providers"), nil, nil
		}

		provider := activeProviders[r.Intn(len(activeProviders))]
		providerAddr, _ := sdk.AccAddressFromBech32(provider.Address)

		var simAccount simtypes.Account
		found := false
		for _, acc := range accs {
			if acc.Address.Equals(providerAddr) {
				simAccount = acc
				found = true
				break
			}
		}
		if !found {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgDeactivateProvider, "provider account not found"), nil, nil
		}

		msg := &types.MsgDeactivateProvider{
			Provider: provider.Address,
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

// SimulateMsgSubmitRequest generates a MsgSubmitRequest with random values
func SimulateMsgSubmitRequest(k keeper.Keeper, ak types.AccountKeeper, bk types.BankKeeper) simtypes.Operation {
	return func(
		r *rand.Rand, app interface{}, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		// Random compute specs
		specs := &types.ComputeSpecs{
			Cpu:        uint64(simtypes.RandIntBetween(r, 1, 16)),
			Memory:     uint64(simtypes.RandIntBetween(r, 1024, 16384)),
			Gpu:        uint64(simtypes.RandIntBetween(r, 0, 2)),
			Storage:    uint64(simtypes.RandIntBetween(r, 10, 100)),
			DurationMs: uint64(simtypes.RandIntBetween(r, 1000, 3600000)),
		}

		// Estimate payment needed
		payment := sdk.NewCoins(sdk.NewInt64Coin("upaw", int64(simtypes.RandIntBetween(r, 1000, 100000))))

		spendable := bk.SpendableCoins(ctx, simAccount.Address)
		if !spendable.IsAllGTE(payment) {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgSubmitRequest, "insufficient balance"), nil, nil
		}

		msg := &types.MsgSubmitRequest{
			Requester:     simAccount.Address.String(),
			ContainerImage: "alpine:latest",
			Specs:         specs,
			Payment:       payment,
			OutputUrl:     "https://storage.github.com/output",
		}

		txCtx := simulation.OperationInput{
			R:               r,
			App:             app.(*module.SimulationManager),
			TxGen:           nil,
			Cdc:             nil,
			Msg:             msg,
			Context:         ctx,
			SimAccount:      simAccount,
			AccountKeeper:   ak,
			Bankkeeper:      bk,
			ModuleName:      types.ModuleName,
			CoinsSpentInMsg: payment,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

// SimulateMsgSubmitResult generates a MsgSubmitResult with random values
func SimulateMsgSubmitResult(k keeper.Keeper, ak types.AccountKeeper, bk types.BankKeeper) simtypes.Operation {
	return func(
		r *rand.Rand, app interface{}, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Find a random processing request
		var processingRequests []types.Request
		err := k.IterateRequestsByStatus(ctx, types.RequestStatus_PROCESSING, func(request types.Request) (bool, error) {
			processingRequests = append(processingRequests, request)
			return false, nil
		})
		if err != nil || len(processingRequests) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgSubmitResult, "no processing requests"), nil, nil
		}

		request := processingRequests[r.Intn(len(processingRequests))]
		providerAddr, _ := sdk.AccAddressFromBech32(request.Provider)

		var simAccount simtypes.Account
		found := false
		for _, acc := range accs {
			if acc.Address.Equals(providerAddr) {
				simAccount = acc
				found = true
				break
			}
		}
		if !found {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgSubmitResult, "provider account not found"), nil, nil
		}

		msg := &types.MsgSubmitResult{
			Provider:  request.Provider,
			RequestId: request.Id,
			ResultUrl: "https://storage.github.com/result",
			Success:   true,
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

// SimulateMsgCancelRequest generates a MsgCancelRequest with random values
func SimulateMsgCancelRequest(k keeper.Keeper, ak types.AccountKeeper, bk types.BankKeeper) simtypes.Operation {
	return func(
		r *rand.Rand, app interface{}, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		// Find pending requests by this requester
		var pendingRequests []types.Request
		err := k.IterateRequestsByRequester(ctx, simAccount.Address, func(request types.Request) (bool, error) {
			if request.Status == types.RequestStatus_PENDING {
				pendingRequests = append(pendingRequests, request)
			}
			return false, nil
		})
		if err != nil || len(pendingRequests) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgCancelRequest, "no pending requests"), nil, nil
		}

		request := pendingRequests[r.Intn(len(pendingRequests))]

		msg := &types.MsgCancelRequest{
			Requester: simAccount.Address.String(),
			RequestId: request.Id,
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
