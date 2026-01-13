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
	keeper *oraclekeeper.Keeper
}

// NewOracleDecorator creates a new OracleDecorator
func NewOracleDecorator(keeper *oraclekeeper.Keeper) *OracleDecorator {
	return &OracleDecorator{
		keeper: keeper,
	}
}

// AnteHandle implements the AnteDecorator interface.
//
//nolint:gocritic // sdk.Context is passed by value per Cosmos SDK AnteHandler contract.
func (od *OracleDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// Skip validation during simulation
	if simulate {
		return next(ctx, tx, simulate)
	}

	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		// Check if module is enabled for any Oracle message
		switch msg.(type) {
		case *oracletypes.MsgSubmitPrice, *oracletypes.MsgDelegateFeedConsent:
			params, err := od.keeper.GetParams(ctx)
			if err != nil {
				return ctx, fmt.Errorf("failed to get Oracle params: %w", err)
			}
			if !params.Enabled {
				return ctx, oracletypes.ErrModuleDisabled.Wrap("Oracle module is disabled by governance - enable via governance proposal")
			}
		}

		// Additional message-specific validation
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

// validateSubmitPrice performs additional validation for price submissions.
//
//nolint:gocritic // sdk.Context intentionally passed by value to match keeper expectations.
func (od *OracleDecorator) validateSubmitPrice(ctx sdk.Context, msg *oracletypes.MsgSubmitPrice) error {
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

	if err := od.keeper.ValidateFeeder(ctx, validator, feeder); err != nil {
		return sdkerrors.ErrUnauthorized.Wrap(err.Error())
	}

	return nil
}

// validateDelegateFeedConsent performs additional validation for feeder delegation.
//
//nolint:gocritic // sdk.Context intentionally passed by value to match keeper expectations.
func (od *OracleDecorator) validateDelegateFeedConsent(ctx sdk.Context, msg *oracletypes.MsgDelegateFeedConsent) error {
	validator, err := sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid validator address: %s", err)
	}

	delegate, err := sdk.AccAddressFromBech32(msg.Delegate)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid delegate address: %s", err)
	}

	isActive, err := od.keeper.IsActiveValidator(ctx, validator)
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
