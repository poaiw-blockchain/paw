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

	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// Simulation operation weights constants
const (
	OpWeightMsgRegisterProvider   = "op_weight_msg_register_provider"   // #nosec G101 - identifier string
	OpWeightMsgUpdateProvider     = "op_weight_msg_update_provider"     // #nosec G101 - identifier string
	OpWeightMsgDeactivateProvider = "op_weight_msg_deactivate_provider" // #nosec G101 - identifier string
	OpWeightMsgSubmitRequest      = "op_weight_msg_submit_request"      // #nosec G101 - identifier string
	OpWeightMsgSubmitResult       = "op_weight_msg_submit_result"       // #nosec G101 - identifier string
	OpWeightMsgCancelRequest      = "op_weight_msg_cancel_request"      // #nosec G101 - identifier string

	DefaultWeightMsgRegisterProvider   = 20
	DefaultWeightMsgUpdateProvider     = 10
	DefaultWeightMsgDeactivateProvider = 5
	DefaultWeightMsgSubmitRequest      = 50
	DefaultWeightMsgSubmitResult       = 40
	DefaultWeightMsgCancelRequest      = 10
)

// WeightedOperations returns all the compute module operations with their respective weights.
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
		weightMsgRegisterProvider   int
		weightMsgUpdateProvider     int
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
			SimulateMsgRegisterProvider(txGen, protoCdc, k, ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgUpdateProvider,
			SimulateMsgUpdateProvider(txGen, protoCdc, k, ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgDeactivateProvider,
			SimulateMsgDeactivateProvider(txGen, protoCdc, k, ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgSubmitRequest,
			SimulateMsgSubmitRequest(txGen, protoCdc, k, ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgSubmitResult,
			SimulateMsgSubmitResult(txGen, protoCdc, k, ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgCancelRequest,
			SimulateMsgCancelRequest(txGen, protoCdc, k, ak, bk),
		),
	}
}

// SimulateMsgRegisterProvider generates a MsgRegisterProvider with random values
func SimulateMsgRegisterProvider(
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

		// Get minimum stake requirement
		params, err := k.GetParams(ctx)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgRegisterProvider, "unable to get params"), nil, nil
		}

		// Check if account has enough balance
		spendable := bk.SpendableCoins(ctx, simAccount.Address)
		minStakeCoins := sdk.NewCoins(sdk.NewCoin("upaw", params.MinProviderStake))
		if !spendable.IsAllGTE(minStakeCoins) {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgRegisterProvider, "insufficient balance"), nil, nil
		}

		msg := &types.MsgRegisterProvider{
			Provider: simAccount.Address.String(),
			Moniker:  "sim-provider",
			Endpoint: "https://github.com/compute",
			AvailableSpecs: types.ComputeSpec{
				CpuCores:       types.SaturateIntToUint64(simtypes.RandIntBetween(r, 2, 16)),
				MemoryMb:       types.SaturateIntToUint64(simtypes.RandIntBetween(r, 2048, 16384)),
				GpuCount:       types.SaturateIntToUint32(simtypes.RandIntBetween(r, 0, 2)),
				GpuType:        "generic",
				StorageGb:      types.SaturateIntToUint64(simtypes.RandIntBetween(r, 10, 200)),
				TimeoutSeconds: types.SaturateIntToUint64(simtypes.RandIntBetween(r, 300, 7200)),
			},
			Pricing: types.Pricing{
				CpuPricePerMcoreHour:  math.LegacyNewDecFromInt(math.NewInt(int64(simtypes.RandIntBetween(r, 1, 10)))),
				MemoryPricePerMbHour:  math.LegacyNewDecFromInt(math.NewInt(int64(simtypes.RandIntBetween(r, 1, 10)))),
				GpuPricePerHour:       math.LegacyNewDecFromInt(math.NewInt(int64(simtypes.RandIntBetween(r, 10, 50)))),
				StoragePricePerGbHour: math.LegacyNewDecFromInt(math.NewInt(int64(simtypes.RandIntBetween(r, 1, 5)))),
			},
			Stake: params.MinProviderStake,
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
			CoinsSpentInMsg: minStakeCoins,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

// SimulateMsgUpdateProvider generates a MsgUpdateProvider with random values
func SimulateMsgUpdateProvider(
	txGen client.TxConfig,
	cdc *codec.ProtoCodec,
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
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
			Moniker:  "sim-provider-updated",
			Endpoint: "https://updated.github.com/compute",
			Pricing: &types.Pricing{
				CpuPricePerMcoreHour:  math.LegacyNewDecFromInt(math.NewInt(int64(simtypes.RandIntBetween(r, 1, 10)))),
				MemoryPricePerMbHour:  math.LegacyNewDecFromInt(math.NewInt(int64(simtypes.RandIntBetween(r, 1, 10)))),
				GpuPricePerHour:       math.LegacyNewDecFromInt(math.NewInt(int64(simtypes.RandIntBetween(r, 10, 50)))),
				StoragePricePerGbHour: math.LegacyNewDecFromInt(math.NewInt(int64(simtypes.RandIntBetween(r, 1, 5)))),
			},
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

// SimulateMsgDeactivateProvider generates a MsgDeactivateProvider with random values
func SimulateMsgDeactivateProvider(
	txGen client.TxConfig,
	cdc *codec.ProtoCodec,
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
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

// SimulateMsgSubmitRequest generates a MsgSubmitRequest with random values
func SimulateMsgSubmitRequest(
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

		// Random compute specs
		specs := types.ComputeSpec{
			CpuCores:       types.SaturateIntToUint64(simtypes.RandIntBetween(r, 1, 16)),
			MemoryMb:       types.SaturateIntToUint64(simtypes.RandIntBetween(r, 1024, 16384)),
			GpuCount:       types.SaturateIntToUint32(simtypes.RandIntBetween(r, 0, 2)),
			GpuType:        "generic",
			StorageGb:      types.SaturateIntToUint64(simtypes.RandIntBetween(r, 10, 100)),
			TimeoutSeconds: types.SaturateIntToUint64(simtypes.RandIntBetween(r, 60, 3600)),
		}

		// Estimate payment needed
		paymentAmount := math.NewInt(int64(simtypes.RandIntBetween(r, 1000, 100000)))
		paymentCoins := sdk.NewCoins(sdk.NewCoin("upaw", paymentAmount))

		spendable := bk.SpendableCoins(ctx, simAccount.Address)
		if !spendable.IsAllGTE(paymentCoins) {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgSubmitRequest, "insufficient balance"), nil, nil
		}

		msg := &types.MsgSubmitRequest{
			Requester:      simAccount.Address.String(),
			ContainerImage: "alpine:latest",
			Specs:          specs,
			Command:        []string{"echo", "hello"},
			EnvVars:        map[string]string{"ENV": "prod"},
			MaxPayment:     paymentAmount,
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
			CoinsSpentInMsg: paymentCoins,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

// SimulateMsgSubmitResult generates a MsgSubmitResult with random values
func SimulateMsgSubmitResult(
	txGen client.TxConfig,
	cdc *codec.ProtoCodec,
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Find a random processing request
		var processingRequests []types.Request
		err := k.IterateRequestsByStatus(ctx, types.REQUEST_STATUS_PROCESSING, func(request types.Request) (bool, error) {
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
			Provider:   request.Provider,
			RequestId:  request.Id,
			OutputHash: "hash123",
			OutputUrl:  "https://storage.github.com/result",
			ExitCode:   0,
			LogsUrl:    "https://storage.github.com/logs",
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

// SimulateMsgCancelRequest generates a MsgCancelRequest with random values
func SimulateMsgCancelRequest(
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

		// Find pending requests by this requester
		var pendingRequests []types.Request
		err := k.IterateRequestsByRequester(ctx, simAccount.Address, func(request types.Request) (bool, error) {
			if request.Status == types.REQUEST_STATUS_PENDING {
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
