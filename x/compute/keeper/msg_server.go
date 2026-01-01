package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
	sharedkeeper "github.com/paw-chain/paw/x/shared/keeper"
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
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	// Parse provider address
	providerAddr, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid provider address: %v", err)
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
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	// Parse provider address
	providerAddr, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid provider address: %v", err)
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
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	// Parse provider address
	providerAddr, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid provider address: %v", err)
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
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	// Parse requester address
	requesterAddr, err := sdk.AccAddressFromBech32(msg.Requester)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid requester address: %v", err)
	}

	// Check rate limits before processing request
	if err := ms.Keeper.CheckRequestRateLimit(ctx, requesterAddr); err != nil {
		return nil, err
	}

	// SEC-12: Validate requester has sufficient balance BEFORE accepting request
	// This prevents requests from being accepted when the requester cannot pay
	if err := ms.Keeper.ValidateRequesterBalance(ctx, requesterAddr, msg.MaxPayment); err != nil {
		return nil, err
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

	// Record request for rate limiting (after successful submission)
	ms.Keeper.RecordComputeRequest(ctx, requesterAddr)

	return &types.MsgSubmitRequestResponse{
		RequestId: requestID,
	}, nil
}

// CancelRequest handles the cancellation of a pending request
func (ms msgServer) CancelRequest(goCtx context.Context, msg *types.MsgCancelRequest) (*types.MsgCancelRequestResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	// Parse requester address
	requesterAddr, err := sdk.AccAddressFromBech32(msg.Requester)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid requester address: %v", err)
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
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	// Parse provider address
	providerAddr, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid provider address: %v", err)
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
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	// Check authority
	if err := sharedkeeper.ValidateAuthority(ms.Keeper.authority, msg.Authority); err != nil {
		return nil, err
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
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	requester, err := sdk.AccAddressFromBech32(msg.Requester)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid requester: %v", err)
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
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	validator, err := sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid validator: %v", err)
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
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	authority, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid authority: %v", err)
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
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	submitter, err := sdk.AccAddressFromBech32(msg.Submitter)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid submitter: %v", err)
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
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	provider, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid provider: %v", err)
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
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	validator, err := sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid validator: %v", err)
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
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}
	authority, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid authority: %v", err)
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
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	// Validate authority
	if err := sharedkeeper.ValidateAuthority(ms.Keeper.authority, msg.Authority); err != nil {
		return nil, err
	}

	if err := ms.Keeper.SetGovernanceParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateGovernanceParamsResponse{}, nil
}

// RegisterSigningKey handles the registration of a provider's signing key.
// SEC-2 FIX: Providers MUST explicitly register their signing key before submitting results.
// This prevents trust-on-first-use attacks where an attacker could submit a result with
// their own key before the legitimate provider registers.
func (ms msgServer) RegisterSigningKey(goCtx context.Context, msg *types.MsgRegisterSigningKey) (*types.MsgRegisterSigningKeyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, types.ErrValidationFailed.Wrap(err.Error())
	}

	// Parse provider address
	providerAddr, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid provider address: %v", err)
	}

	// Register the signing key
	if err := ms.Keeper.RegisterSigningKey(ctx, providerAddr, msg.PublicKey, msg.OldKeySignature); err != nil {
		return nil, err
	}

	return &types.MsgRegisterSigningKeyResponse{}, nil
}

// SubmitBatchRequests handles submission of multiple compute requests in a single transaction
// AGENT-1: Enables batch compute request submission for agents with reduced gas overhead
func (ms msgServer) SubmitBatchRequests(goCtx context.Context, msg *types.MsgSubmitBatchRequests) (*types.MsgSubmitBatchRequestsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate message
	if msg == nil {
		return nil, types.ErrInvalidRequest.Wrap("nil message")
	}

	// Validate batch size (max 20 requests per batch)
	const maxBatchSize = 20
	if len(msg.Requests) == 0 {
		return nil, types.ErrInvalidRequest.Wrap("empty request batch")
	}
	if len(msg.Requests) > maxBatchSize {
		return nil, types.ErrInvalidRequest.Wrapf("batch size %d exceeds maximum %d", len(msg.Requests), maxBatchSize)
	}

	// Validate requester address
	if _, err := sdk.AccAddressFromBech32(msg.Requester); err != nil {
		return nil, types.ErrInvalidRequest.Wrapf("invalid requester address: %v", err)
	}

	// SEC-2.4: Pre-check gas limit for batch requests to prevent DoS
	// Each request consumes approximately 10,000 gas (validation + provider search + escrow + storage)
	// We check upfront if the batch would exceed safe limits
	const gasPerRequest = uint64(10000)
	const maxBatchGas = uint64(150000) // Safe limit for batch operations
	estimatedGas := gasPerRequest * uint64(len(msg.Requests))
	remainingGas := ctx.GasMeter().Limit() - ctx.GasMeter().GasConsumed()
	if estimatedGas > maxBatchGas {
		return nil, types.ErrInvalidRequest.Wrapf("batch estimated gas %d exceeds max batch gas %d", estimatedGas, maxBatchGas)
	}
	if estimatedGas > remainingGas {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"batch_request_gas_exceeded",
				sdk.NewAttribute("estimated_gas", fmt.Sprintf("%d", estimatedGas)),
				sdk.NewAttribute("remaining_gas", fmt.Sprintf("%d", remainingGas)),
				sdk.NewAttribute("batch_size", fmt.Sprintf("%d", len(msg.Requests))),
			),
		)
		return nil, types.ErrInvalidRequest.Wrapf("batch estimated gas %d exceeds remaining gas %d", estimatedGas, remainingGas)
	}

	results := make([]types.BatchRequestResult, 0, len(msg.Requests))
	var successCount uint64
	totalDeposit := math.ZeroInt()

	// Process each request
	for _, reqItem := range msg.Requests {
		result := types.BatchRequestResult{
			Success: false,
		}

		// Build a single request message
		singleReq := &types.MsgSubmitRequest{
			Requester:         msg.Requester,
			Specs:             reqItem.Specs,
			ContainerImage:    reqItem.ContainerImage,
			Command:           reqItem.Command,
			EnvVars:           reqItem.EnvVars,
			MaxPayment:        reqItem.MaxPayment,
			PreferredProvider: reqItem.PreferredProvider,
		}

		// Submit the request
		resp, reqErr := ms.SubmitRequest(goCtx, singleReq)
		if reqErr != nil {
			result.Error = reqErr.Error()
			results = append(results, result)
			// Continue with next request - batch is not atomic for requests
			continue
		}

		result.RequestId = resp.RequestId
		result.Success = true
		successCount++
		// SEC-3.3: Use SafeAdd to prevent overflow when accumulating batch deposits
		// While math.Int uses big integers and is generally overflow-safe, explicit SafeAdd
		// provides defense-in-depth and clearer error messages if overflow occurs
		newTotalDeposit, err := totalDeposit.SafeAdd(reqItem.MaxPayment)
		if err != nil {
			// Overflow detected - fail the batch request to prevent unexpected behavior
			return nil, types.ErrInvalidRequest.Wrapf("batch deposit overflow: totalDeposit=%s + payment=%s: %v",
				totalDeposit.String(), reqItem.MaxPayment.String(), err)
		}
		totalDeposit = newTotalDeposit
		results = append(results, result)
	}

	// Emit batch request event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"batch_compute_requests",
			sdk.NewAttribute("requester", msg.Requester),
			sdk.NewAttribute("total_requests", fmt.Sprintf("%d", len(msg.Requests))),
			sdk.NewAttribute("successful_requests", fmt.Sprintf("%d", successCount)),
			sdk.NewAttribute("total_deposit", totalDeposit.String()),
		),
	)

	return &types.MsgSubmitBatchRequestsResponse{
		Results:            results,
		TotalRequests:      uint64(len(msg.Requests)),
		SuccessfulRequests: successCount,
		TotalDeposit:       totalDeposit,
	}, nil
}
