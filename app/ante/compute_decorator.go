package ante

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
	computetypes "github.com/paw-chain/paw/x/compute/types"
)

// ComputeDecorator validates compute module-specific transaction requirements
type ComputeDecorator struct {
	keeper computekeeper.Keeper
}

// NewComputeDecorator creates a new ComputeDecorator
func NewComputeDecorator(keeper computekeeper.Keeper) ComputeDecorator {
	return ComputeDecorator{
		keeper: keeper,
	}
}

// AnteHandle implements the AnteDecorator interface
func (cd ComputeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// Skip validation during simulation
	if simulate {
		return next(ctx, tx, simulate)
	}

	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *computetypes.MsgSubmitRequest:
			if err := cd.validateSubmitRequest(ctx, msg); err != nil {
				return ctx, err
			}
		case *computetypes.MsgRegisterProvider:
			if err := cd.validateRegisterProvider(ctx, msg); err != nil {
				return ctx, err
			}
		case *computetypes.MsgSubmitResult:
			if err := cd.validateSubmitResult(ctx, msg); err != nil {
				return ctx, err
			}
		}
	}

	return next(ctx, tx, simulate)
}

// validateSubmitRequest performs additional validation for compute requests
func (cd ComputeDecorator) validateSubmitRequest(ctx sdk.Context, msg *computetypes.MsgSubmitRequest) error {
	// Check if requester has sufficient balance for max payment
	requester, err := sdk.AccAddressFromBech32(msg.Requester)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid requester address: %s", err)
	}

	// Consume gas for validation
	ctx.GasMeter().ConsumeGas(1000, "compute request validation")

	if msg.MaxPayment.IsNegative() {
		return sdkerrors.ErrInvalidRequest.Wrap("max payment must be non-negative")
	}

	if err := cd.keeper.ValidateRequesterBalance(ctx, requester, msg.MaxPayment); err != nil {
		return err
	}

	return nil
}

// validateRegisterProvider performs additional validation for provider registration
func (cd ComputeDecorator) validateRegisterProvider(ctx sdk.Context, msg *computetypes.MsgRegisterProvider) error {
	provider, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid provider address: %s", err)
	}

	// Consume gas for validation
	ctx.GasMeter().ConsumeGas(1500, "provider registration validation")

	// Check if provider is already registered
	existingProvider, err := cd.keeper.GetProvider(sdk.WrapSDKContext(ctx), provider)
	if err == nil && existingProvider != nil && existingProvider.Active {
		return sdkerrors.ErrInvalidRequest.Wrap("provider already registered and active")
	}

	// Get module params
	params, err := cd.keeper.GetParams(sdk.WrapSDKContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	// Validate minimum stake
	if msg.Stake.LT(params.MinProviderStake) {
		return sdkerrors.ErrInvalidRequest.Wrapf("stake %s is less than minimum %s",
			msg.Stake.String(), params.MinProviderStake.String())
	}

	return nil
}

// validateSubmitResult performs additional validation for result submission
func (cd ComputeDecorator) validateSubmitResult(ctx sdk.Context, msg *computetypes.MsgSubmitResult) error {
	provider, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid provider address: %s", err)
	}

	// Consume gas for validation
	ctx.GasMeter().ConsumeGas(2000, "result submission validation")

	// Verify provider is registered and active
	existingProvider, err := cd.keeper.GetProvider(sdk.WrapSDKContext(ctx), provider)
	if err != nil {
		return sdkerrors.ErrNotFound.Wrap("provider not found")
	}

	if !existingProvider.Active {
		return sdkerrors.ErrInvalidRequest.Wrap("provider is not active")
	}

	// Verify request exists and is assigned to this provider
	request, err := cd.keeper.GetRequest(sdk.WrapSDKContext(ctx), msg.RequestId)
	if err != nil {
		return sdkerrors.ErrNotFound.Wrapf("request %d not found", msg.RequestId)
	}

	if request.Provider != msg.Provider {
		return sdkerrors.ErrUnauthorized.Wrapf("request %d is not assigned to provider %s", msg.RequestId, msg.Provider)
	}

	if request.Status != computetypes.REQUEST_STATUS_ASSIGNED {
		return sdkerrors.ErrInvalidRequest.Wrapf("request %d is not in ASSIGNED status", msg.RequestId)
	}

	return nil
}
