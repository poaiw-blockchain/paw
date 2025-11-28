package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/compute/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// RegisterProvider handles the registration of a new compute provider
func (ms msgServer) RegisterProvider(goCtx context.Context, msg *types.MsgRegisterProvider) (*types.MsgRegisterProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Parse provider address
	providerAddr, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, fmt.Errorf("invalid provider address: %w", err)
	}

	// Register provider
	if err := ms.Keeper.RegisterProvider(
		ctx,
		providerAddr,
		msg.Moniker,
		msg.Endpoint,
		msg.AvailableSpecs,
		msg.Pricing,
		msg.Stake,
	); err != nil {
		return nil, err
	}

	return &types.MsgRegisterProviderResponse{}, nil
}

// UpdateProvider handles updates to an existing provider's information
func (ms msgServer) UpdateProvider(goCtx context.Context, msg *types.MsgUpdateProvider) (*types.MsgUpdateProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Parse provider address
	providerAddr, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, fmt.Errorf("invalid provider address: %w", err)
	}

	// Update provider
	if err := ms.Keeper.UpdateProvider(
		ctx,
		providerAddr,
		msg.Moniker,
		msg.Endpoint,
		msg.AvailableSpecs,
		msg.Pricing,
	); err != nil {
		return nil, err
	}

	return &types.MsgUpdateProviderResponse{}, nil
}

// DeactivateProvider handles provider deactivation
func (ms msgServer) DeactivateProvider(goCtx context.Context, msg *types.MsgDeactivateProvider) (*types.MsgDeactivateProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Parse provider address
	providerAddr, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, fmt.Errorf("invalid provider address: %w", err)
	}

	// Deactivate provider
	if err := ms.Keeper.DeactivateProvider(ctx, providerAddr); err != nil {
		return nil, err
	}

	return &types.MsgDeactivateProviderResponse{}, nil
}

// SubmitRequest handles the submission of a new compute request
func (ms msgServer) SubmitRequest(goCtx context.Context, msg *types.MsgSubmitRequest) (*types.MsgSubmitRequestResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Parse requester address
	requesterAddr, err := sdk.AccAddressFromBech32(msg.Requester)
	if err != nil {
		return nil, fmt.Errorf("invalid requester address: %w", err)
	}

	// Submit request
	requestID, err := ms.Keeper.SubmitRequest(
		ctx,
		requesterAddr,
		msg.Specs,
		msg.ContainerImage,
		msg.Command,
		msg.EnvVars,
		msg.MaxPayment,
		msg.PreferredProvider,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgSubmitRequestResponse{
		RequestId: requestID,
	}, nil
}

// CancelRequest handles the cancellation of a pending request
func (ms msgServer) CancelRequest(goCtx context.Context, msg *types.MsgCancelRequest) (*types.MsgCancelRequestResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Parse requester address
	requesterAddr, err := sdk.AccAddressFromBech32(msg.Requester)
	if err != nil {
		return nil, fmt.Errorf("invalid requester address: %w", err)
	}

	// Cancel request
	if err := ms.Keeper.CancelRequest(ctx, requesterAddr, msg.RequestId); err != nil {
		return nil, err
	}

	return &types.MsgCancelRequestResponse{}, nil
}

// SubmitResult handles the submission of a compute result by a provider
func (ms msgServer) SubmitResult(goCtx context.Context, msg *types.MsgSubmitResult) (*types.MsgSubmitResultResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Parse provider address
	providerAddr, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, fmt.Errorf("invalid provider address: %w", err)
	}

	// Submit result
	if err := ms.Keeper.SubmitResult(
		ctx,
		providerAddr,
		msg.RequestId,
		msg.OutputHash,
		msg.OutputUrl,
		msg.ExitCode,
		msg.LogsUrl,
		msg.VerificationProof,
	); err != nil {
		return nil, err
	}

	return &types.MsgSubmitResultResponse{}, nil
}

// UpdateParams handles updates to module parameters
func (ms msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Check authority
	if ms.Keeper.authority != msg.Authority {
		return nil, fmt.Errorf("invalid authority: expected %s, got %s", ms.Keeper.authority, msg.Authority)
	}

	// Update params
	if err := ms.Keeper.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}
