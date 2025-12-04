package ante

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	oraclekeeper "github.com/paw-chain/paw/x/oracle/keeper"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// OracleDecorator validates oracle module-specific transaction requirements
type OracleDecorator struct {
	keeper oraclekeeper.Keeper
}

// NewOracleDecorator creates a new OracleDecorator
func NewOracleDecorator(keeper oraclekeeper.Keeper) OracleDecorator {
	return OracleDecorator{
		keeper: keeper,
	}
}

// AnteHandle implements the AnteDecorator interface
func (od OracleDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// Skip validation during simulation
	if simulate {
		return next(ctx, tx, simulate)
	}

	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *oracletypes.MsgSubmitPrice:
			if err := od.validateSubmitPrice(ctx, msg); err != nil {
				return ctx, err
			}
		case *oracletypes.MsgDelegateFeedConsent:
			if err := od.validateDelegateFeedConsent(ctx, msg); err != nil {
				return ctx, err
			}
		}
	}

	return next(ctx, tx, simulate)
}

// validateSubmitPrice performs additional validation for price submissions
func (od OracleDecorator) validateSubmitPrice(ctx sdk.Context, msg *oracletypes.MsgSubmitPrice) error {
	validator, err := sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid validator address: %s", err)
	}

	feeder, err := sdk.AccAddressFromBech32(msg.Feeder)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid feeder address: %s", err)
	}

	if msg.Asset == "" {
		return sdkerrors.ErrInvalidRequest.Wrap("asset cannot be empty")
	}

	// Validate price is positive
	if msg.Price.IsNil() || msg.Price.LTE(math.LegacyZeroDec()) {
		return sdkerrors.ErrInvalidRequest.Wrap("price must be positive")
	}

	goCtx := sdk.WrapSDKContext(ctx)

	if err := od.keeper.ValidateFeeder(goCtx, validator, feeder); err != nil {
		return sdkerrors.ErrUnauthorized.Wrap(err.Error())
	}

	return nil
}

// validateDelegateFeedConsent performs additional validation for feeder delegation
func (od OracleDecorator) validateDelegateFeedConsent(ctx sdk.Context, msg *oracletypes.MsgDelegateFeedConsent) error {
	validator, err := sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid validator address: %s", err)
	}

	delegate, err := sdk.AccAddressFromBech32(msg.Delegate)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid delegate address: %s", err)
	}

	goCtx := sdk.WrapSDKContext(ctx)
	isActive, err := od.keeper.IsActiveValidator(goCtx, validator)
	if err != nil {
		return fmt.Errorf("failed to verify validator activity: %w", err)
	}

	if !isActive {
		return sdkerrors.ErrUnauthorized.Wrap("validator is not bonded")
	}

	if !od.keeper.IsAuthorizedFeeder(ctx, delegate, validator) {
		return sdkerrors.ErrUnauthorized.Wrap("delegate not authorized for validator")
	}

	return nil
}
