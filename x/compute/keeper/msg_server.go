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

// CreateDispute handles dispute creation with deposit locking and evidence capture.
func (ms msgServer) CreateDispute(goCtx context.Context, msg *types.MsgCreateDispute) (*types.MsgCreateDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	requester, err := sdk.AccAddressFromBech32(msg.Requester)
	if err != nil {
		return nil, fmt.Errorf("invalid requester: %w", err)
	}

	disputeID, err := ms.Keeper.CreateDispute(ctx, requester, msg.RequestId, msg.Reason, msg.Evidence, msg.DepositAmount)
	if err != nil {
		return nil, err
	}

	return &types.MsgCreateDisputeResponse{DisputeId: disputeID}, nil
}

// VoteOnDispute records validator votes with justification and power.
func (ms msgServer) VoteOnDispute(goCtx context.Context, msg *types.MsgVoteOnDispute) (*types.MsgVoteOnDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	validator, err := sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return nil, fmt.Errorf("invalid validator: %w", err)
	}

	if err := ms.Keeper.VoteOnDispute(ctx, validator, msg.DisputeId, msg.Vote, msg.Justification); err != nil {
		return nil, err
	}

	return &types.MsgVoteOnDisputeResponse{}, nil
}

// ResolveDispute settles a dispute, applies slashing/refunds, and finalizes escrow.
func (ms msgServer) ResolveDispute(goCtx context.Context, msg *types.MsgResolveDispute) (*types.MsgResolveDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	authority, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return nil, fmt.Errorf("invalid authority: %w", err)
	}

	resolution, err := ms.Keeper.ResolveDispute(ctx, authority, msg.DisputeId)
	if err != nil {
		return nil, err
	}

	// Post-resolution settlement: handle escrow and slashing in keeper
	if err := ms.Keeper.SettleDisputeOutcome(ctx, msg.DisputeId, resolution); err != nil {
		return nil, err
	}

	return &types.MsgResolveDisputeResponse{Resolution: resolution}, nil
}

// SubmitEvidence attaches additional evidence to an active dispute.
func (ms msgServer) SubmitEvidence(goCtx context.Context, msg *types.MsgSubmitEvidence) (*types.MsgSubmitEvidenceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	submitter, err := sdk.AccAddressFromBech32(msg.Submitter)
	if err != nil {
		return nil, fmt.Errorf("invalid submitter: %w", err)
	}

	if err := ms.Keeper.SubmitEvidence(ctx, submitter, msg.DisputeId, msg.EvidenceType, msg.Data, msg.Description); err != nil {
		return nil, err
	}

	return &types.MsgSubmitEvidenceResponse{}, nil
}

// AppealSlashing lets providers challenge a slash with a weighted vote appeal.
func (ms msgServer) AppealSlashing(goCtx context.Context, msg *types.MsgAppealSlashing) (*types.MsgAppealSlashingResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	provider, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, fmt.Errorf("invalid provider: %w", err)
	}

	appealID, err := ms.Keeper.CreateAppeal(ctx, provider, msg.SlashId, msg.Justification, msg.DepositAmount)
	if err != nil {
		return nil, err
	}

	return &types.MsgAppealSlashingResponse{AppealId: appealID}, nil
}

// VoteOnAppeal records validator votes on an appeal.
func (ms msgServer) VoteOnAppeal(goCtx context.Context, msg *types.MsgVoteOnAppeal) (*types.MsgVoteOnAppealResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	validator, err := sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return nil, fmt.Errorf("invalid validator: %w", err)
	}

	if err := ms.Keeper.VoteOnAppeal(ctx, validator, msg.AppealId, msg.Approve, msg.Justification); err != nil {
		return nil, err
	}

	return &types.MsgVoteOnAppealResponse{}, nil
}

// ResolveAppeal finalizes an appeal and applies state changes to the slash record.
func (ms msgServer) ResolveAppeal(goCtx context.Context, msg *types.MsgResolveAppeal) (*types.MsgResolveAppealResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	authority, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return nil, fmt.Errorf("invalid authority: %w", err)
	}

	approved, err := ms.Keeper.ResolveAppeal(ctx, authority, msg.AppealId)
	if err != nil {
		return nil, err
	}

	if err := ms.Keeper.ApplyAppealOutcome(ctx, msg.AppealId, approved); err != nil {
		return nil, err
	}

	return &types.MsgResolveAppealResponse{Approved: approved}, nil
}

// UpdateGovernanceParams updates dispute/appeal governance settings.
func (ms msgServer) UpdateGovernanceParams(goCtx context.Context, msg *types.MsgUpdateGovernanceParams) (*types.MsgUpdateGovernanceParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	authority, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return nil, fmt.Errorf("invalid authority: %w", err)
	}
	if authority.String() != ms.Keeper.authority {
		return nil, fmt.Errorf("unauthorized governance params update: expected %s", ms.Keeper.authority)
	}

	if err := ms.Keeper.SetGovernanceParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateGovernanceParamsResponse{}, nil
}
